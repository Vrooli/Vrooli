package labels

import "testing"

type testEnum string

func (value testEnum) String() string { return string(value) }

func TestEnumFormatsKnownValuesAndUsesFallbackForUnknownValues(t *testing.T) {
	tests := []struct {
		name     string
		value    testEnum
		prefix   string
		fallback string
		format   func(string) string
		want     string
	}{
		{name: "upper status", value: "STATUS_PASS", prefix: "STATUS_", fallback: "UNKNOWN", format: func(raw string) string { return raw }, want: "PASS"},
		{name: "lower words", value: "CAPABILITY_STT", prefix: "CAPABILITY_", fallback: "unknown", format: LowerWords, want: "stt"},
		{name: "unspecified", value: "STATUS_UNSPECIFIED", prefix: "STATUS_", fallback: "UNKNOWN", format: func(raw string) string { return raw }, want: "UNKNOWN"},
		{name: "wrong prefix", value: "STATE_READY", prefix: "STATUS_", fallback: "UNKNOWN", format: func(raw string) string { return raw }, want: "UNKNOWN"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Enum(test.value, test.prefix, test.fallback, test.format); got != test.want {
				t.Fatalf("Enum() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestLowerWords(t *testing.T) {
	if got := LowerWords("PROCESS_STATE_READY"); got != "process-state-ready" {
		t.Fatalf("LowerWords() = %q", got)
	}
}
