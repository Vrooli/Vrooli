package core

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestModelSourceReachesEveryLaunch(t *testing.T) {
	for _, durable := range []bool{false, true} {
		for _, continuation := range []bool{false, true} {
			t.Run(fmt.Sprintf("durable=%v/continue=%v", durable, continuation), func(t *testing.T) {
				codec := newFakeCodec()
				codec.rt = domain.RunnerTypeCodex
				launcher := &fakeLauncher{stdout: "{\"type\":\"message\",\"content\":\"done\"}\n"}
				r := newRunnerForTest(t, codec, launcher)
				req := newExecuteRequest(uuid.New(), "test", &recordingSink{})
				req.Environment = map[string]string{
					"VROOLI_MODEL_SOURCE":       `{"id":"credits","kind":"vrooli_credits","model":"model-a","base_url":"http://127.0.0.1:1234/api/v1/ai","secret_env":"VROOLI_CODING_SESSION_KEY"}`,
					"VROOLI_CODING_SESSION_KEY": "child-key",
				}
				if durable {
					stdout, err := os.CreateTemp(t.TempDir(), "transcript-*.ndjson")
					if err != nil {
						t.Fatal(err)
					}
					defer stdout.Close()
					req.Transcript = &runner.TranscriptConfig{TranscriptPath: stdout.Name(), StdoutFile: stdout}
				}
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				var err error
				if continuation {
					_, err = r.Continue(ctx, runner.ContinueRequest{RunID: req.RunID, SessionID: "session", Prompt: req.Prompt, WorkingDir: req.WorkingDir, EventSink: req.EventSink, ResolvedConfig: req.ResolvedConfig, Transcript: req.Transcript, Environment: req.Environment})
				} else {
					_, err = r.Execute(ctx, req)
				}
				if err != nil {
					t.Fatal(err)
				}
				launch := launcher.lastRequestSnapshot()
				if !strings.Contains(strings.Join(launch.Args, " "), `model_provider="vrooli-source"`) || !strings.Contains(strings.Join(launch.Env, " "), "VROOLI_CODING_SESSION_KEY=child-key") {
					t.Fatalf("model source missing from launch: args=%q env=%q", launch.Args, launch.Env)
				}
			})
		}
	}
}

// SYNTHETIC provider envelopes: actual limit captures remain operator-owned.
func TestCodingLimitIsTakenOnlyFromCurrentStructuredStream(t *testing.T) {
	for _, durable := range []bool{false, true} {
		for _, tc := range []struct {
			name, output string
			want         bool
		}{
			{"explicit", `{"type":"error","error":{"code":"usage_limit_reached"}}` + "\n", true},
			{"assistant-text", `{"type":"message","content":"usage_limit_reached"}` + "\n", false},
			{"transient", `{"type":"error","error":{"code":"rate_limit_error"}}` + "\n", false},
		} {
			t.Run(fmt.Sprintf("durable=%v/%s", durable, tc.name), func(t *testing.T) {
				codec := newFakeCodec()
				codec.rt = domain.RunnerTypeCodex
				r := newRunnerForTest(t, codec, &fakeLauncher{stdout: tc.output})
				req := newExecuteRequest(uuid.New(), "test", &recordingSink{})
				if durable {
					stdout, err := os.CreateTemp(t.TempDir(), "transcript-*.ndjson")
					if err != nil {
						t.Fatal(err)
					}
					defer stdout.Close()
					// An older limit in the same file must not trigger this invocation.
					if _, err := stdout.WriteString(`{"type":"error","error":{"code":"usage_limit_reached"}}` + "\n"); err != nil {
						t.Fatal(err)
					}
					req.Transcript = &runner.TranscriptConfig{TranscriptPath: stdout.Name(), StdoutFile: stdout}
				}
				result, err := r.Execute(context.Background(), req)
				if err != nil {
					t.Fatal(err)
				}
				if got := result.CodingLimit != nil; got != tc.want {
					t.Fatalf("limit present = %v, want %v", got, tc.want)
				}
			})
		}
	}
}
