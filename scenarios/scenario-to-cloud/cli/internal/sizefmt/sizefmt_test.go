package sizefmt

import "testing"

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{name: "negative", bytes: -1, want: "-1B"},
		{name: "zero", bytes: 0, want: "0B"},
		{name: "byte", bytes: 1, want: "1B"},
		{name: "byte boundary", bytes: 1023, want: "1023B"},
		{name: "one kibibyte", bytes: 1024, want: "1.0K"},
		{name: "fractional kibibyte", bytes: 1536, want: "1.5K"},
		{name: "one mebibyte", bytes: 1024 * 1024, want: "1.0M"},
		{name: "one gibibyte", bytes: 1024 * 1024 * 1024, want: "1.0G"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatBytes(tt.bytes); got != tt.want {
				t.Fatalf("FormatBytes(%d)=%q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}
