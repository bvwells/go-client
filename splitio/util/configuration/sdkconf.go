// Package configuration ...
// Contains configuration structures used to setup the SDK
package configuration

import (
	"errors"
	"fmt"
	"github.com/splitio/go-client/splitio/util/impressionlistener"
	"github.com/splitio/go-toolkit/datastructures/set"
	"github.com/splitio/go-toolkit/logging"
	"github.com/splitio/go-toolkit/nethelpers"
	"os/user"
	"path"
	"strings"
)

// SplitSdkConfig struct ...
// struct used to setup a Split.io SDK client.
//
// Parameters:
// - Apikey: (Required) API-KEY used to authenticate user requests
// - OperationMode (Required) Must be one of ["inmemory-standalone", "redis-consumer", "redis-standalone"]
// - InstanceName (Optional) Name to be used when submitting metrics & impressions to split servers
// - Logger: (Optional) Custom logger complying with logging.LoggerInterface
// - LoggerConfig: (Optional) Options to setup the sdk's own logger
// - TaskPeriods: (Optional) How often should each task run
// - Redis: (Required for "redis-consumer" & "redis-standalone" operation modes. Sets up Redis config
// - Advanced: (Optional) Sets up various advanced options for the sdk
type SplitSdkConfig struct {
	OperationMode   string
	InstanceName    string
	IPAddress       string
	BlockUntilReady int
	SplitFile       string
	LabelsEnabled   bool
	Logger          logging.LoggerInterface
	LoggerConfig    *logging.LoggerOptions
	TaskPeriods     *TaskPeriods
	Advanced        *AdvancedConfig
	Redis           *RedisConfig
}

// TaskPeriods struct is used to configure the period for each synchronization task
type TaskPeriods struct {
	SplitSync      int
	SegmentSync    int
	ImpressionSync int
	GaugeSync      int
	CounterSync    int
	LatencySync    int
}

// RedisConfig struct is used to cofigure the redis parameters
type RedisConfig struct {
	Host     string
	Port     int
	Database int
	Password string
	Prefix   string
}

// AdvancedConfig exposes more configurable parameters that can be used to further tailor the sdk to the user's needs
type AdvancedConfig struct {
	ImpressionListener impressionlistener.ListenerInterface
	HTTPTimeout        int
	SdkURL             string
	EventsURL          string
	SegmentQueueSize   int
	SegmentWorkers     int
}

// Default returns a config struct with all the default values
func Default() *SplitSdkConfig {

	ipAddress, err := nethelpers.ExternalIP()
	if err != nil {
		ipAddress = "unknown"
	}

	var splitFile string
	usr, err := user.Current()
	if err != nil {
		splitFile = "splits"
	} else {
		splitFile = path.Join(usr.HomeDir, ".splits")
	}

	return &SplitSdkConfig{
		OperationMode:   "inmemory-standalone",
		LabelsEnabled:   true,
		BlockUntilReady: defaultBlockUntilReady,
		IPAddress:       ipAddress,
		InstanceName:    fmt.Sprintf("ip-%s", strings.Replace(ipAddress, ".", "-", -1)),
		Logger:          nil,
		LoggerConfig:    &logging.LoggerOptions{},
		SplitFile:       splitFile,
		Redis: &RedisConfig{
			Database: 0,
			Host:     "localhost",
			Password: "",
			Port:     6379,
			Prefix:   "",
		},
		TaskPeriods: &TaskPeriods{
			CounterSync:    defaultTaskPeriod,
			GaugeSync:      defaultTaskPeriod,
			LatencySync:    defaultTaskPeriod,
			ImpressionSync: defaultTaskPeriod,
			SegmentSync:    defaultTaskPeriod,
			SplitSync:      defaultTaskPeriod,
		},
		Advanced: &AdvancedConfig{
			EventsURL:          "",
			SdkURL:             "",
			HTTPTimeout:        0,
			ImpressionListener: nil,
			SegmentQueueSize:   500,
			SegmentWorkers:     10,
		},
	}
}

// Validate checks that the parameters passed by the user are correct and returns an error if something is wrong
func Validate(apikey string, cfg *SplitSdkConfig) error {
	// Fail if no apikey is provided
	if apikey == "" && cfg.OperationMode != "localhost" {
		return errors.New("Config parameter \"Apikey\" is mandatory for operation modes other than localhost")
	}

	// To keep the interface consistent with other sdks we accept "localhost" as an apikey,
	// which sets the operation mode to localhost
	if apikey == "localhost" {
		cfg.OperationMode = "localhost"
	}

	// Fail if an invalid operation-mode is provided
	operationModes := set.NewSet(
		"localhost",
		"inmemory-standalone",
		"redis-consumer",
		"redis-standalone",
	)

	if !operationModes.Has(cfg.OperationMode) {
		return fmt.Errorf("OperationMode parameter must be one of: %v", operationModes.List())
	}

	return nil
}
