package export

import "testing"

func TestSanitizeFilename(t *testing.T) {
	for _, tc := range []struct{ name, input, fallback, want string }{
		{name: "trims and removes separators and nulls", input: " /private\x00/shot ", fallback: "default", want: "-private-shot"},
		{name: "uses caller fallback for blank name", input: " \t ", fallback: "browser-automation-replay", want: "browser-automation-replay"},
		{name: "keeps ordinary filename", input: "capture.webm", fallback: "default", want: "capture.webm"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := SanitizeFilename(tc.input, tc.fallback); got != tc.want { t.Fatalf("SanitizeFilename(%q, %q) = %q, want %q", tc.input, tc.fallback, got, tc.want) }
		})
	}
}
