package matchers

import (
	"github.com/splitio/go-client/splitio/service/dtos"
	"reflect"
	"testing"
)

func TestPartOfSetMatcher(t *testing.T) {
	attrName := "setdata"
	dto := &dtos.MatcherDTO{
		MatcherType: "PART_OF_SET",
		Whitelist: &dtos.WhitelistMatcherDataDTO{
			Whitelist: []string{"one", "two", "three", "four"},
		},
		KeySelector: &dtos.KeySelectorDTO{
			Attribute: &attrName,
		},
	}

	matcher, err := BuildMatcher(dto, nil)
	if err != nil {
		t.Error("There should be no errors when building the matcher")
		t.Error(err)
	}

	matcherType := reflect.TypeOf(matcher).String()
	if matcherType != "*matchers.PartOfSetMatcher" {
		t.Errorf("Incorrect matcher constructed. Should be *matchers.PartOfSetMatcher and was %s", matcherType)
	}

	if !matcher.Match("asd", map[string]interface{}{"setdata": []string{"one", "two", "three", "four"}}, nil) {
		t.Error("Matcher an equal set")
	}

	if !matcher.Match("asd", map[string]interface{}{"setdata": []string{"one", "two", "three", "four", "five"}}, nil) {
		t.Error("Matcher should match a superset")
	}

	if matcher.Match("asd", map[string]interface{}{"setdata": []string{}}, nil) {
		t.Error("Matcher should not match an empty set")
	}

	if matcher.Match("asd", map[string]interface{}{"setdata": []string{"one", "two", "three"}}, nil) {
		t.Error("Matcher should not match a subset")
	}
}
