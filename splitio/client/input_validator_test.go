package client

import (
	"math"
	"math/rand"
	"strings"
	"sync"
	"testing"

	"github.com/splitio/go-client/splitio/conf"
	spConf "github.com/splitio/go-split-commons/conf"
	"github.com/splitio/go-split-commons/dtos"
	"github.com/splitio/go-split-commons/service"
	authMocks "github.com/splitio/go-split-commons/service/mocks"
	"github.com/splitio/go-split-commons/storage/mocks"
	"github.com/splitio/go-split-commons/storage/mutexmap"
	"github.com/splitio/go-split-commons/storage/mutexqueue"
	"github.com/splitio/go-split-commons/synchronizer"
	"github.com/splitio/go-toolkit/logging"
)

type MockWriter struct {
	mutex  sync.RWMutex
	strMsg string
}

func (m *MockWriter) Write(p []byte) (n int, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.strMsg = string(p[:])
	return 0, nil
}

func (m *MockWriter) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.strMsg = ""
}

func (m *MockWriter) Get() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.strMsg
}

var mW MockWriter

func expectedLogMessage(expectedMessage string, t *testing.T) {
	if !strings.Contains(mW.strMsg, expectedMessage) {
		t.Error("Message error is different from the expected: " + mW.Get())
	}
	mW.Reset()
}

func getMockedLogger() logging.LoggerInterface {
	return logging.NewLogger(&logging.LoggerOptions{
		LogLevel:      5,
		ErrorWriter:   &mW,
		WarningWriter: &mW,
		InfoWriter:    nil,
		DebugWriter:   nil,
		VerboseWriter: nil,
	})
}

func getClient() SplitClient {
	logger := getMockedLogger()
	cfg := conf.Default()
	factory := &SplitFactory{cfg: cfg}

	client := SplitClient{
		evaluator:   &mockEvaluator{},
		impressions: mutexqueue.NewMQImpressionsStorage(cfg.Advanced.ImpressionsQueueSize, make(chan string, 1), logger),
		metrics:     mutexmap.NewMMMetricsStorage(),
		logger:      logger,
		validator: inputValidation{
			logger: logger,
			splitStorage: mocks.MockSplitStorage{
				TrafficTypeExistsCall: func(trafficType string) bool {
					switch trafficType {
					case "trafictype":
						return true
					default:
						return false
					}
				},
			},
		},
		events: mocks.MockEventStorage{
			PushCall: func(event dtos.EventDTO, size int) error { return nil },
		},
		factory: factory,
	}
	factory.status.Store(sdkStatusReady)
	return client
}

func TestFactoryWithNilApiKey(t *testing.T) {
	cfg := conf.Default()
	cfg.Logger = getMockedLogger()
	_, err := NewSplitFactory("", cfg)

	if err == nil {
		t.Error("Should be error")
	}

	expected := "Factory instantiation: you passed an empty apikey, apikey must be a non-empty string"
	if !strings.Contains(mW.Get(), expected) {
		t.Error("Error is distinct from the expected one", mW.Get())
	}
	mW.Reset()
}

func getLongKey() string {
	m := ""
	for n := 0; n <= 256; n++ {
		m += "m"
	}
	return m
}

func TestTreatmentValidatorOnKeys(t *testing.T) {
	client := getClient()
	// Nil
	expectedTreatment(client.Treatment(nil, "feature", nil), "control", t)
	expectedLogMessage("Treatment: you passed a nil key, key must be a non-empty string", t)

	// Boolean
	expectedTreatment(client.Treatment(true, "feature", nil), "control", t)
	expectedLogMessage("Treatment: you passed an invalid key, key must be a non-empty string", t)

	// Trimmed
	expectedTreatment(client.Treatment("     ", "feature", nil), "control", t)
	expectedLogMessage("Treatment: you passed an empty key, key must be a non-empty string", t)

	// Long
	expectedTreatment(client.Treatment(getLongKey(), "feature", nil), "control", t)
	expectedLogMessage("Treatment: key too long - must be 250 characters or less", t)

	// String
	expectedTreatment(client.Treatment("key", "feature", nil), "TreatmentA", t)
	expectedLogMessage("", t)

	// Int
	expectedTreatment(client.Treatment(123, "feature", nil), "TreatmentA", t)
	expectedLogMessage("Treatment: key %!s(int=123) is not of type string, converting", t)

	// Int32
	expectedTreatment(client.Treatment(int32(123), "feature", nil), "TreatmentA", t)
	expectedLogMessage("Treatment: key %!s(int32=123) is not of type string, converting", t)

	// Int 64
	expectedTreatment(client.Treatment(int64(123), "feature", nil), "TreatmentA", t)
	expectedLogMessage("Treatment: key %!s(int64=123) is not of type string, converting", t)

	// Float
	expectedTreatment(client.Treatment(1.3, "feature", nil), "TreatmentA", t)
	expectedLogMessage("Treatment: key %!s(float64=1.3) is not of type string, converting", t)

	// NaN
	expectedTreatment(client.Treatment(math.NaN, "feature", nil), "control", t)
	expectedLogMessage("Treatment: you passed an invalid key, key must be a non-empty string", t)

	// Inf
	expectedTreatment(client.Treatment(math.Inf, "feature", nil), "control", t)
	expectedLogMessage("Treatment: you passed an invalid key, key must be a non-empty string", t)
}

func getKey(matchingKey string, bucketingKey string) *Key {
	return &Key{
		MatchingKey:  matchingKey,
		BucketingKey: bucketingKey,
	}
}

func TestTreatmentValidatorWithKeyObject(t *testing.T) {
	client := getClient()
	// Empty
	expectedTreatment(client.Treatment(getKey("", "bucketing"), "feature", nil), "control", t)
	expectedLogMessage("Treatment: you passed an empty matchingKey, matchingKey must be a non-empty string", t)

	// Long
	expectedTreatment(client.Treatment(getKey(getLongKey(), "bucketing"), "feature", nil), "control", t)
	expectedLogMessage("Treatment: matchingKey too long - must be 250 characters or less", t)

	// Empty Bucketing
	expectedTreatment(client.Treatment(getKey("matching", ""), "feature", nil), "control", t)
	expectedLogMessage("Treatment: you passed an empty bucketingKey, bucketingKey must be a non-empty string", t)

	// Long Bucketing
	expectedTreatment(client.Treatment(getKey("matching", getLongKey()), "feature", nil), "control", t)
	expectedLogMessage("Treatment: bucketingKey too long - must be 250 characters or less", t)

	// Ok
	expectedTreatment(client.Treatment(getKey("matching", "bucketing"), "feature", nil), "TreatmentA", t)
	expectedLogMessage("", t)
}

func TestTreatmentValidatorOnFeatureName(t *testing.T) {
	client := getClient()
	// Empty
	expectedTreatment(client.Treatment("key", "", nil), "control", t)
	expectedLogMessage("Treatment: you passed an empty featureName, featureName must be a non-empty string", t)

	// Trimmed
	expectedTreatment(client.Treatment("key", "  feature   ", nil), "TreatmentA", t)
	expectedLogMessage("Treatment: split name '  feature   ' has extra whitespace, trimming", t)

	// Non Existent
	expectedTreatment(client.Treatment("key", "feature_non_existent", nil), "control", t)
	expectedLogMessage("Treatment: you passed feature_non_existent that does not exist in this environment, please double check what Splits exist in the web console", t)

	// Non Existent
	expectedTreatmentAndConfig(client.TreatmentWithConfig("key", "feature_non_existent", nil), "control", "", t)
	expectedLogMessage("TreatmentWithConfig: you passed feature_non_existent that does not exist in this environment, please double check what Splits exist in the web console", t)
}

func expectedTreatments(key interface{}, features []string, length int, t *testing.T) map[string]string {
	client := getClient()
	result := client.Treatments(key, features, nil)
	if len(result) != length {
		t.Error("Wrong len of elements")
	}
	return result
}

func TestTreatmentsValidator(t *testing.T) {
	client := getClient()
	// Empty features
	expectedTreatments("key", []string{""}, 0, t)
	expectedLogMessage("Treatments: features must be a non-empty array", t)

	// Inf
	result := expectedTreatments(math.Inf, []string{"feature"}, 1, t)
	expectedTreatment(result["feature"], "control", t)
	expectedLogMessage("Treatments: you passed an invalid key, key must be a non-empty string", t)

	// Float
	result = expectedTreatments(1.3, []string{"feature"}, 1, t)
	expectedTreatment(result["feature"], "TreatmentA", t)
	expectedLogMessage("Treatments: key %!s(float64=1.3) is not of type string, converting", t)

	// Trimmed
	result = expectedTreatments("key", []string{" some_feature  "}, 1, t)
	expectedTreatment(result["some_feature"], "control", t)
	expectedLogMessage("Treatments: split name ' some_feature  ' has extra whitespace, trimming", t)

	// Non Existent
	result = expectedTreatments("key", []string{"feature_non_existent"}, 1, t)
	expectedTreatment(result["feature_non_existent"], "control", t)
	expectedLogMessage("Treatments: you passed feature_non_existent that does not exist in this environment, please double check what Splits exist in the web console", t)

	// Non Existent Config
	resultWithConfig := client.TreatmentsWithConfig("key", []string{"feature_non_existent"}, nil)
	expectedTreatmentAndConfig(resultWithConfig["feature_non_existent"], "control", "", t)
	expectedLogMessage("TreatmentsWithConfig: you passed feature_non_existent that does not exist in this environment, please double check what Splits exist in the web console", t)
}

func TestValidatorOnDestroy(t *testing.T) {
	logger := getMockedLogger()
	sync, _ := synchronizer.NewSynchronizerManager(
		synchronizer.NewLocal(3, &service.SplitAPI{}, mocks.MockSplitStorage{}, logger),
		logger,
		spConf.AdvancedConfig{},
		authMocks.MockAuthClient{},
		mocks.MockSplitStorage{},
		make(chan int, 1),
	)
	factory := &SplitFactory{
		cfg:         conf.Default(),
		syncManager: sync,
	}
	factory.status.Store(sdkStatusReady)
	var client2 = SplitClient{
		evaluator:   &mockEvaluator{},
		impressions: mutexqueue.NewMQImpressionsStorage(5000, make(chan string, 1), logger),
		metrics:     mutexmap.NewMMMetricsStorage(),
		logger:      logger,
		validator:   inputValidation{logger: logger},
		factory:     factory,
	}

	var manager = SplitManager{
		logger:    logger,
		validator: inputValidation{logger: logger},
		factory:   factory,
	}

	client2.Destroy()

	expectedTreatment(client2.Treatment("key", "  feature   ", nil), "control", t)
	expectedLogMessage("Client has already been destroyed - no calls possible", t)

	result := client2.Treatments("key", []string{"some_feature"}, nil)
	expectedTreatment(result["some_feature"], "control", t)
	expectedLogMessage("Client has already been destroyed - no calls possible", t)

	expectedTrack(client2.Track("key", "trafficType", "eventType", 0, nil), "Client has already been destroyed - no calls possible", t)

	manager.Split("feature")
	expectedLogMessage("Client has already been destroyed - no calls possible", t)
}

func expectedTrack(err error, expected string, t *testing.T) {
	if err != nil && err.Error() != expected {
		t.Error("Wrong error", err.Error())
	}
	expectedLogMessage(expected, t)
}

func makeBigString(length int) string {
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	asRuneSlice := make([]rune, length)
	for index := range asRuneSlice {
		asRuneSlice[index] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(asRuneSlice)
}

/*
@TODO: Move to Set log writer
func TestTrackValidators(t *testing.T) {
	client := getClient()
	// Empty key
	expectedTrack(client.Track("", "trafficType", "eventType", nil, nil), "Track: you passed an empty key, key must be a non-empty string", t)

	// Long key
	expectedTrack(client.Track(getLongKey(), "trafficType", "eventType", nil, nil), "Track: key too long - must be 250 characters or less", t)

	// Empty event type
	expectedTrack(client.Track("key", "trafficType", "", nil, nil), "Track: you passed an empty event type, event type must be a non-empty string", t)

	// Not match regex
	expected := "Track: you passed //, event name must adhere to " +
		"the regular expression ^[a-zA-Z0-9][-_.:a-zA-Z0-9]{0,79}$. This means an event " +
		"name must be alphanumeric, cannot be more than 80 characters long, and can " +
		"only include a dash, underscore, period, or colon as separators of " +
		"alphanumeric characters"
	expectedTrack(client.Track("key", "trafficType", "//", nil, nil), expected, t)

	// Empty traffic type
	expectedTrack(client.Track("key", "", "eventType", nil, nil), "Track: you passed an empty traffic type, traffic type must be a non-empty string", t)

	// Not matching traffic type
	expected = "Track: traffic type traffic does not have any corresponding Splits in this environment, make sure you’re tracking your events to a valid traffic type defined in the Split console"
	expectedTrack(client.Track("key", "traffic", "eventType", nil, nil), expected, t)

	// Uppercase traffic type
	expectedTrack(client.Track("key", "traficTYPE", "eventType", nil, nil), "Track: traffic type should be all lowercase - converting string to lowercase", t)

	// Traffic Type No Ocurrences
	err := client.Track("key", "trafficTypeNoOcurrences", "eventType", nil, nil)
	expectedLogMessage("Track: traffic type traffictypenoocurrences does not have any corresponding Splits in this environment, make sure you’re tracking your events to a valid traffic type defined in the Split console", t)
	if err != nil {
		t.Error("Should not be error")
	}

	// Value
	expectedTrack(client.Track("key", "traffic", "eventType", true, nil), "Track: value must be a number", t)

	// Properties
	props := make(map[string]interface{})
	for i := 0; i < 301; i++ {
		props[fmt.Sprintf("prop-%d", i)] = "asd"
	}
	expectedTrack(client.Track("key", "traffic", "eventType", 1, props), "Track: Event has more than 300 properties. Some of them will be trimmed when processed", t)

	// Properties > 32kb
	props2 := make(map[string]interface{})
	for i := 0; i < 299; i++ {
		props2[fmt.Sprintf("%s%d", makeBigString(255), i)] = makeBigString(255)
	}
	expectedTrack(client.Track("key", "traffic", "eventType", nil, props2), "The maximum size allowed for the properties is 32kb. Event not queued", t)

	// Ok
	err = client.Track("key", "traffic", "eventType", 1, nil)

	if err != nil {
		t.Error("Should not return error")
	}
}
*/

func TestLocalhostTrafficType(t *testing.T) {
	sdkConf := conf.Default()
	sdkConf.SplitFile = "../../testdata/splits.yaml"
	factory, _ := NewSplitFactory("localhost", sdkConf)
	client := factory.Client()

	_ = client.BlockUntilReady(1)

	factory.status.Store(sdkStatusInitializing)

	if client.isReady() {
		t.Error("Localhost should not be ready")
	}

	err := client.Track("key", "traffic", "eventType", nil, nil)

	if err != nil {
		t.Error("It should not inform any err")
	}

	expectedLogMessage("", t)
}

func TestTrackNotReadyYetTrafficType(t *testing.T) {
	logger := getMockedLogger()
	var factoryNotReady = &SplitFactory{}
	var clientNotReady = SplitClient{
		evaluator:   &mockEvaluator{},
		impressions: mutexqueue.NewMQImpressionsStorage(5000, make(chan string, 1), logger),
		metrics:     mutexmap.NewMMMetricsStorage(),
		logger:      logger,
		validator: inputValidation{
			logger:       logger,
			splitStorage: mocks.MockSplitStorage{},
		},
		events: mocks.MockEventStorage{
			PushCall: func(event dtos.EventDTO, size int) error { return nil },
		},
		factory: factoryNotReady,
	}

	factoryNotReady.status.Store(sdkStatusInitializing)

	expected := "Track: the SDK is not ready, results may be incorrect. Make sure to wait for SDK readiness before using this method"
	expectedTrack(clientNotReady.Track("key", "traffic", "eventType", nil, nil), expected, t)
}

func TestManagerWithEmptySplit(t *testing.T) {
	splitStorage := mutexmap.NewMMSplitStorage()
	factory := SplitFactory{}
	manager := SplitManager{
		splitStorage: splitStorage,
		logger:       getMockedLogger(),
	}

	factory.status.Store(sdkStatusReady)
	manager.factory = &factory

	manager.Split("")
	expectedLogMessage("Split: you passed an empty split name, split name must be a non-empty string", t)

	manager.Split("non_existent")
	expectedLogMessage("Split: you passed non_existent that does not exist in this environment, please double check what Splits exist in the web console", t)
}
