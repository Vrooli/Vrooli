package platform

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestProcessArgumentBoundariesAndEmptyValuesArePreserved(t *testing.T) {
	encoded := []byte("/opt/service\x00--literal=one two\x00\x00tabs\there\x00")
	got, err := parseNULProcessArguments(encoded)
	if err != nil || !reflect.DeepEqual(got, []string{"/opt/service", "--literal=one two", "", "tabs\there"}) {
		t.Fatalf("exact launch arguments were lost: %v", err)
	}
}

func TestProcessArgumentsRefuseMissingTruncatedAndUnboundedEvidenceWithoutLeakingValues(t *testing.T) {
	for name, data := range map[string][]byte{
		"empty":            nil,
		"empty executable": {0},
		"unterminated":     []byte("/opt/service\x00--token=private-fixture-value"),
		"too large":        append([]byte("/opt/service\x00"), bytes.Repeat([]byte{'x'}, maxProcessArgumentBytes)...),
		"too many":         append([]byte("/opt/service\x00"), bytes.Repeat([]byte{'x', 0}, maxProcessArguments)...),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := parseNULProcessArguments(data)
			if err == nil || strings.Contains(err.Error(), "private-fixture-value") {
				t.Fatalf("unproven argument block accepted or leaked a value: %v", err)
			}
		})
	}
}
