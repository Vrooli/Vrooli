package orchestration

import (
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// TestWebConsoleSessionURL covers the run-detail deep-link computation: it is
// set only for interactive runs with a session id and a resolved UI base, and
// omitted otherwise so clients fall back to the session id.
func TestWebConsoleSessionURL(t *testing.T) {
	interactive := &domain.Run{
		ID:                  uuid.New(),
		ExecutionMode:       domain.ExecutionModeInteractive,
		WebConsoleSessionID: "sess-abc",
	}
	codec := &domain.Run{
		ID:                  uuid.New(),
		ExecutionMode:       domain.ExecutionModeCodecPipe,
		WebConsoleSessionID: "sess-abc",
	}

	tests := []struct {
		name string
		o    *Orchestrator
		run  *domain.Run
		want string
	}{
		{
			name: "interactive with base",
			o:    &Orchestrator{webConsoleUIBase: "http://localhost:21233"},
			run:  interactive,
			want: "http://localhost:21233/?session=sess-abc",
		},
		{
			name: "trailing slash trimmed",
			o:    &Orchestrator{webConsoleUIBase: "http://localhost:21233/"},
			run:  interactive,
			want: "http://localhost:21233/?session=sess-abc",
		},
		{
			name: "no base resolves to empty",
			o:    &Orchestrator{webConsoleUIBase: ""},
			run:  interactive,
			want: "",
		},
		{
			name: "codec-pipe run gets no link",
			o:    &Orchestrator{webConsoleUIBase: "http://localhost:21233"},
			run:  codec,
			want: "",
		},
		{
			name: "interactive without session id gets no link",
			o:    &Orchestrator{webConsoleUIBase: "http://localhost:21233"},
			run:  &domain.Run{ExecutionMode: domain.ExecutionModeInteractive},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.o.webConsoleSessionURL(tt.run); got != tt.want {
				t.Errorf("webConsoleSessionURL = %q, want %q", got, tt.want)
			}
		})
	}
}
