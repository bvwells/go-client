package matchers

import (
	"github.com/splitio/go-client/splitio/engine/grammar/matchers/datatypes"
)

// GreaterThanOrEqualToMatcher will match if two numbers or two datetimes are equal
type GreaterThanOrEqualToMatcher struct {
	Matcher
	ComparisonDataType string
	ComparisonValue    int64
}

// Match will match if the comparisonValue is greater than or equal to the matchingValue
func (m *GreaterThanOrEqualToMatcher) Match(key string, attributes map[string]interface{}, bucketingKey *string) bool {
	matchingRaw, err := m.matchingKey(key, attributes)
	if err != nil {
		return false
	}

	matchingValue, ok := matchingRaw.(int64)
	if !ok {
		var asInt int
		asInt, ok = matchingRaw.(int)
		if ok {
			matchingValue = int64(asInt)
		}
	}
	if !ok {
		return false
	}

	var comparisonValue int64
	switch m.ComparisonDataType {
	case datatypes.Number:
		comparisonValue = m.ComparisonValue
	case datatypes.Datetime:
		matchingValue = datatypes.ZeroSecondsTS(matchingValue)
		comparisonValue = datatypes.ZeroSecondsTS(m.ComparisonValue)
	default:
		return false
	}
	return matchingValue >= comparisonValue
}

// NewGreaterThanOrEqualToMatcher returns a pointer to a new instance of GreaterThanOrEqualToMatcher
func NewGreaterThanOrEqualToMatcher(negate bool, cmpVal int64, cmpType string, attributeName *string) *GreaterThanOrEqualToMatcher {
	return &GreaterThanOrEqualToMatcher{
		Matcher: Matcher{
			negate:        negate,
			attributeName: attributeName,
		},
		ComparisonValue:    cmpVal,
		ComparisonDataType: cmpType,
	}
}
