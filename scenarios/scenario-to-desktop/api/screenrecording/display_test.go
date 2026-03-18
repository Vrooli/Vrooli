package screenrecording

import (
	"testing"
)

func TestParseDisplayNumber(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{":99", 99, false},
		{":0", 0, false},
		{":1", 1, false},
		{"abc", 0, true},
		{":", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseDisplayNumber(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseDisplayNumber(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("ParseDisplayNumber(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestNewDisplayManager(t *testing.T) {
	dm := NewDisplayManager()
	if dm == nil {
		t.Fatal("NewDisplayManager returned nil")
	}
}
