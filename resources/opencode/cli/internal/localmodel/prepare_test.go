package localmodel

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

type call struct {
	name string
	args []string
}

type fakeRunner struct {
	calls       []call
	failCall    string
	failRelease bool
	cancelOn    string
	cancel      context.CancelFunc
	reservation string
	doctor      string
	chat        string
	tools       string
	progress    string
}

func (f *fakeRunner) Run(ctx context.Context, name string, args []string, progress io.Writer) ([]byte, error) {
	current := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, call{name: name, args: append([]string(nil), args...)})
	if f.cancelOn != "" && strings.HasPrefix(current, f.cancelOn) && f.cancel != nil {
		f.cancel()
		return nil, ctx.Err()
	}
	if f.failRelease && name == "resource-ollama" && len(args) > 1 && args[0] == "capacity" && args[1] == "release-model-reservation" {
		return nil, errors.New("fixture release failure")
	}
	if progress != nil {
		_, _ = io.WriteString(progress, "pull progress\n")
		f.progress += "pull progress\n"
	}
	if current == f.failCall {
		return nil, errors.New("fixture failure")
	}
	switch {
	case name == "resource-ollama" && len(args) >= 3 && reflect.DeepEqual(args[:3], []string{"policy", "resolve", "--role"}):
		return []byte(`{"model":"huihui_ai/granite4.1-abliterated:8b"}`), nil
	case name == "resource-ollama" && len(args) > 1 && args[0] == "capacity" && args[1] == "reserve-model":
		if f.reservation != "" {
			return []byte(f.reservation), nil
		}
		return []byte(`{"id":"reservation-1","role":"chat.uncensored","model":"huihui_ai/granite4.1-abliterated:8b","requested_bytes":11918534246}`), nil
	case name == "resource-ollama" && len(args) > 1 && args[0] == "capacity" && args[1] == "confirm-model-reservation":
		return []byte(`{"id":"reservation-1","claim_id":"claim-1","role":"chat.uncensored","model":"huihui_ai/granite4.1-abliterated:8b","requested_bytes":11918534246,"capacity_bytes":12884901888,"admission_bytes":11918534246}`), nil
	case name == "resource-ollama" && len(args) > 1 && args[0] == "capacity" && args[1] == "release-model-reservation":
		return []byte(`{"released":true}`), nil
	case name == "resource-ollama" && len(args) > 1 && args[0] == "models" && args[1] == "doctor":
		if f.doctor != "" {
			return []byte(f.doctor), nil
		}
		return []byte(`{"pass":true,"models":[{"role":"chat.uncensored","model":"huihui_ai/granite4.1-abliterated:8b","pass":true,"installed":true,"live_capabilities":["completion","tools"]}]}`), nil
	case name == "resource-ollama" && len(args) > 1 && args[0] == "models" && args[1] == "probe-tools":
		if f.tools != "" {
			return []byte(f.tools), nil
		}
		return []byte(`{"model":"huihui_ai/granite4.1-abliterated:8b","supports_tools":true}`), nil
	case name == "resource-ollama" && len(args) > 1 && args[0] == "gateway" && args[1] == "chat":
		if f.chat != "" {
			return []byte(f.chat), nil
		}
		return []byte(`{"response":"Hello!","eval_count":2}`), nil
	default:
		return []byte("{}"), nil
	}
}

func TestPrepareCancellationAttemptsDetachedReservationCleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	runner := &fakeRunner{cancelOn: "vrooli resource start ollama", cancel: cancel}
	_, err := Prepare(ctx, "chat.uncensored", Hooks{
		Runner: runner, CommitSelection: func(context.Context, string) error { t.Fatal("cancelled preparation selected a model"); return nil },
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Prepare error = %v, want context cancellation", err)
	}
	for _, c := range runner.calls {
		if len(c.args) > 1 && c.args[0] == "capacity" && c.args[1] == "release-model-reservation" {
			return
		}
	}
	t.Fatalf("reservation cleanup was not attempted after cancellation: %+v", runner.calls)
}

func TestSelectJSONReceiptIncludesReservationCleanupFailure(t *testing.T) {
	runner := &fakeRunner{
		failCall:    "vrooli resource start ollama",
		failRelease: true,
	}
	var stdout, stderr strings.Builder
	h := &Handlers{Hooks: Hooks{Runner: runner}, Stdout: &stdout, Stderr: &stderr}
	err := h.Select([]string{"--role", "chat.uncensored", "--timeout-seconds", "60", "--json"})
	if err == nil || !strings.Contains(err.Error(), "fixture release failure") {
		t.Fatalf("Select error = %v, want joined cleanup failure", err)
	}
	var receipt Receipt
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &receipt); err != nil {
		t.Fatalf("decode JSON receipt %q: %v", stdout.String(), err)
	}
	if receipt.Status != "failed" || !strings.Contains(receipt.Error, "fixture release failure") {
		t.Fatalf("receipt = %+v, want cleanup failure", receipt)
	}
}

func TestPrepareReportsValidationErrorsWithoutExternalEffects(t *testing.T) {
	runner := &fakeRunner{}
	receipt, err := Prepare(context.Background(), "", Hooks{Runner: runner})
	if err == nil || receipt.Status != "failed" || receipt.Stage != "validate" || receipt.Error == "" || len(runner.calls) != 0 {
		t.Fatalf("receipt=%+v err=%v calls=%+v", receipt, err, runner.calls)
	}
}

func TestReservationLeaseOutlivesTheConfiguredOperationDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Hour)
	defer cancel()
	lease := time.Duration(reservationTTLSeconds(ctx)) * time.Second
	remaining, _ := ctx.Deadline()
	if lease <= time.Until(remaining) {
		t.Fatalf("reservation lease %s does not outlive operation deadline %s", lease, time.Until(remaining))
	}
}

func TestPrepareCapacityAdmitsAcquiresChecksAndSelects(t *testing.T) {
	runner := &fakeRunner{}
	var selected string
	var progress strings.Builder
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	receipt, err := Prepare(ctx, "chat.uncensored", Hooks{
		Runner: runner, Progress: &progress,
		CommitSelection: func(ctx context.Context, model string) error {
			if _, ok := ctx.Deadline(); !ok {
				t.Error("config commit lost the operation deadline")
			}
			selected = model
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != "ready" || receipt.Model != "huihui_ai/granite4.1-abliterated:8b" || receipt.SelectedModel != "ollama/"+receipt.Model {
		t.Fatalf("receipt = %+v", receipt)
	}
	want := []string{
		"resource-ollama policy resolve --role chat.uncensored --json",
		"resource-ollama capacity reserve-model --role chat.uncensored --ttl-seconds ",
		"resource-ollama capacity confirm-model-reservation --id reservation-1 --wait-seconds 30 --json",
		"vrooli resource start ollama --no-reclaim --capacity-claim-id claim-1 --capacity-claim-min-bytes 12884901888",
	}
	if len(runner.calls) < len(want) {
		t.Fatalf("calls = %+v", runner.calls)
	}
	for i, command := range want {
		got := runner.calls[i].name + " " + strings.Join(runner.calls[i].args, " ")
		if !strings.HasPrefix(got, command) {
			t.Fatalf("call %d = %q, want %q", i, got, command)
		}
	}
	reservationSeconds, err := strconv.Atoi(runner.calls[1].args[5])
	if err != nil || reservationSeconds < int((time.Minute+reservationTTLGrace).Seconds()) {
		t.Fatalf("reservation lease seconds = %q err=%v, want operation deadline plus cleanup grace", runner.calls[1].args[5], err)
	}
	var pullConfig string
	for _, c := range runner.calls {
		if c.name == "resource-ollama" && len(c.args) > 0 && c.args[0] == "ensure" {
			for i := range c.args {
				if c.args[i] == "--config-base64" && i+1 < len(c.args) {
					pullConfig = c.args[i+1]
				}
			}
		}
	}
	rawConfig, err := base64.StdEncoding.DecodeString(pullConfig)
	if err != nil {
		t.Fatalf("decode ensure config: %v", err)
	}
	var cfg struct {
		ModelRoles []struct {
			Role string `json:"role"`
		} `json:"model_roles"`
	}
	if err := json.Unmarshal(rawConfig, &cfg); err != nil || len(cfg.ModelRoles) != 1 || cfg.ModelRoles[0].Role != "chat.uncensored" {
		t.Fatalf("ensure config = %s err=%v", rawConfig, err)
	}
	if selected != receipt.Model || !strings.Contains(progress.String(), "pull progress") {
		t.Fatalf("selected=%q progress=%q", selected, progress.String())
	}
}

func TestPrepareReleasesParseableButMismatchedReservationReceipt(t *testing.T) {
	runner := &fakeRunner{reservation: `{"id":"reservation-1","role":"different-role","model":"huihui_ai/granite4.1-abliterated:8b","requested_bytes":11918534246}`}
	_, err := Prepare(context.Background(), "chat.uncensored", Hooks{Runner: runner, CommitSelection: func(context.Context, string) error { return nil }})
	if err == nil || !strings.Contains(err.Error(), "incomplete or mismatched") {
		t.Fatalf("Prepare error = %v, want invalid receipt error", err)
	}
	for _, c := range runner.calls {
		if len(c.args) > 1 && c.args[0] == "capacity" && c.args[1] == "release-model-reservation" {
			return
		}
	}
	t.Fatalf("reservation cleanup was not attempted after an invalid receipt: %+v", runner.calls)
}

func TestPrepareCapacityDenialStopsBeforePull(t *testing.T) {
	runner := &fakeRunner{failCall: "resource-ollama capacity confirm-model-reservation --id reservation-1 --wait-seconds 30 --json"}
	receipt, err := Prepare(context.Background(), "chat.uncensored", Hooks{
		Runner: runner, CommitSelection: func(context.Context, string) error { return nil },
	})
	if err == nil || !strings.Contains(err.Error(), "Ollama model-sized capacity admission") || receipt.Status != "failed" || receipt.Stage != "capacity-admission" || receipt.Error == "" {
		t.Fatalf("receipt=%+v error=%v, want capacity denial receipt", receipt, err)
	}
	for _, call := range runner.calls {
		if len(call.args) > 0 && call.args[0] == "ensure" {
			t.Fatal("model pull began after capacity admission failed")
		}
	}
}

func TestPrepareRequiresLiveReadinessAndChatBeforeOpenCodeMutation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		doctor string
		tools  string
		chat   string
	}{
		{name: "doctor skipped live show", doctor: `{"pass":true,"models":[{"role":"chat.uncensored","model":"huihui_ai/granite4.1-abliterated:8b","pass":true,"installed":true}]}`},
		{name: "tool probe only narrated tools", tools: `{"model":"huihui_ai/granite4.1-abliterated:8b","supports_tools":false}`},
		{name: "empty chat smoke", chat: `{"response":" ","eval_count":0}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := &fakeRunner{doctor: tc.doctor, tools: tc.tools, chat: tc.chat}
			committed := false
			_, err := Prepare(context.Background(), "chat.uncensored", Hooks{
				Runner: runner,
				CommitSelection: func(context.Context, string) error {
					committed = true
					return nil
				},
			})
			if err == nil || committed {
				t.Fatalf("error=%v committed=%v, want refusal before OpenCode mutation", err, committed)
			}
		})
	}
}
