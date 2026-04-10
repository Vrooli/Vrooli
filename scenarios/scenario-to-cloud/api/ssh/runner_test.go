package ssh

import (
	"strings"
	"testing"
)

func TestBoundedBuffer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		limit      int
		writes     []string
		wantOutput string
		wantSuffix string // if non-empty, check output ends with this
	}{
		{
			name:       "write within limit",
			limit:      100,
			writes:     []string{"hello world"},
			wantOutput: "hello world",
		},
		{
			name:       "write at exact boundary",
			limit:      11,
			writes:     []string{"hello world"},
			wantOutput: "hello world",
		},
		{
			name:       "write exceeding limit",
			limit:      5,
			writes:     []string{"hello world"},
			wantSuffix: "[output truncated]",
		},
		{
			name:       "multiple writes accumulating past limit",
			limit:      10,
			writes:     []string{"hello", " ", "world", "!"},
			wantSuffix: "[output truncated]",
		},
		{
			name:       "zero-limit buffer",
			limit:      0,
			writes:     []string{"anything"},
			wantOutput: "[output truncated]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := newBoundedBuffer(tt.limit)
			for _, w := range tt.writes {
				_, err := buf.Write([]byte(w))
				if err != nil {
					t.Fatalf("Write returned error: %v", err)
				}
			}
			got := buf.String()
			if tt.wantOutput != "" && got != tt.wantOutput {
				t.Errorf("String() = %q, want %q", got, tt.wantOutput)
			}
			if tt.wantSuffix != "" && !strings.HasSuffix(got, tt.wantSuffix) {
				t.Errorf("String() = %q, want suffix %q", got, tt.wantSuffix)
			}
		})
	}
}

func TestExitCode(t *testing.T) {
	t.Parallel()

	if got := exitCode(nil); got != 0 {
		t.Errorf("exitCode(nil) = %d, want 0", got)
	}
}
