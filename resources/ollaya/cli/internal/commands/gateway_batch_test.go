package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	sharedpolicy "github.com/vrooli/ai-provider-go/policy"
	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/upstream"
)

// batchDaemon answers each state with its own text as the choice, refuses the
// state "too long" as truncated, and counts what the batch cost the daemon.
type batchDaemon struct {
	server                *httptest.Server
	tags, decides         atomic.Int32
	mu                    sync.Mutex
	decidesBeforeReleased int32
}

func newBatchDaemon(t *testing.T, installed map[string]string) *batchDaemon {
	t.Helper()
	d := &batchDaemon{decidesBeforeReleased: -1}
	d.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			d.tags.Add(1)
			var models []map[string]string
			for name, digest := range installed {
				models = append(models, map[string]string{"name": name, "digest": digest})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"models": models})
		case "/api/decide":
			d.decides.Add(1)
			var body struct {
				State     string          `json:"state"`
				Questions json.RawMessage `json:"questions"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.State == "too long" {
				_, _ = w.Write([]byte(`{"model":"winnow:e4b","answers":{},"state_truncated":true}`))
				return
			}
			fmt.Fprintf(w, `{"model":"winnow:e4b","answers":{"q":{"type":"choice","choice":%q}},"usage":{"input_tokens":3}}`, body.State)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(d.server.Close)
	return d
}

// A batch proves rights and the manifest once, loads the model with its first
// state alone inside the load slot, then answers the rest; one state's refusal
// is that state's error and every other state is answered in order.
func TestDecideBatchAdmitsOnceAndAnswersEveryStateInOrder(t *testing.T) {
	contextValue := fixtureRightsContextValue()
	modelPath, digest, _ := fixtureRightsModel(t, "winnow:e4b", contextValue)
	daemon := newBatchDaemon(t, map[string]string{"winnow:e4b": digest})
	states := []string{"s0", "s1", "too long", "s3", "s4", "s5", "s6"}
	encoded, _ := json.Marshal(map[string]any{"questions": map[string]any{"q": map[string]any{"type": "choice"}}, "states": states})
	h, stdout, _ := handlers(t, map[string]string{"OLLAYA_URL": daemon.server.URL, "OLLAYA_MODEL_POLICY_PATH": modelPath}, string(encoded))
	contextReads := 0
	h.RightsContext = func(context.Context) (sharedpolicy.RightsContext, error) {
		contextReads++
		return contextValue, nil
	}
	slotTaken := 0
	h.LoadSlot = func(context.Context) (func(), error) {
		slotTaken++
		return func() {
			daemon.mu.Lock()
			daemon.decidesBeforeReleased = daemon.decides.Load()
			daemon.mu.Unlock()
		}, nil
	}
	args := append(localDecisionArgs(t, modelPath, "decide.local.accurate", "winnow:e4b", digest, contextValue), "--concurrency", "3")
	if err := h.DecideBatch(args); err != nil {
		t.Fatal(err)
	}
	var response BatchResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode batch output: %v (%s)", err, stdout.String())
	}
	if len(response.Results) != len(states) {
		t.Fatalf("results = %d, want one per state", len(response.Results))
	}
	for index, item := range response.Results {
		if states[index] == "too long" {
			if item.Error == nil || item.Error.Code != upstream.CodeContextOverflow || item.Result != nil {
				t.Fatalf("item %d = %+v, want its own context_overflow", index, item)
			}
			continue
		}
		if item.Error != nil || item.Result == nil || !strings.Contains(string(item.Result.Answers), `"`+states[index]+`"`) {
			t.Fatalf("item %d = %+v, want the answer for %q in request order", index, item, states[index])
		}
	}
	if got := daemon.tags.Load(); got != 2 || contextReads != 2 || slotTaken != 1 {
		t.Fatalf("tags=%d contextReads=%d slot=%d; want one double-checked rights and manifest proof and one load slot for the whole batch", got, contextReads, slotTaken)
	}
	if got := daemon.decides.Load(); got != int32(len(states)) {
		t.Fatalf("decides = %d, want one per state", got)
	}
	if daemon.decidesBeforeReleased != 1 {
		t.Fatalf("load slot released after %d decisions, want after the first (the load) and before the rest", daemon.decidesBeforeReleased)
	}
}

// A refusal that concerns the whole batch fails the command once with the
// typed marker and never reaches the daemon's decision endpoint.
func TestDecideBatchRefusesTheBatchBeforeAnyDecision(t *testing.T) {
	contextValue := fixtureRightsContextValue()
	modelPath, digest, _ := fixtureRightsModel(t, "winnow:e4b", contextValue)
	daemon := newBatchDaemon(t, map[string]string{"winnow:e4b": digest})
	h, _, stderr := handlers(t, map[string]string{"OLLAYA_URL": daemon.server.URL, "OLLAYA_MODEL_POLICY_PATH": modelPath}, `{"questions":{"q":{"type":"noul"}},"states":["a","b"]}`)
	h.RightsContext = func(context.Context) (sharedpolicy.RightsContext, error) { return contextValue, nil }
	h.CapacityAdmit = func(context.Context, string, int64) (bool, string, error) { return false, "only 1 GiB free", nil }
	h.LoadSlot = func(context.Context) (func(), error) { return func() {}, nil }
	err := h.DecideBatch(localDecisionArgs(t, modelPath, "decide.local.accurate", "winnow:e4b", digest, contextValue))
	if err == nil || !strings.Contains(stderr.String(), upstream.ProviderErrorMarker) || !strings.Contains(stderr.String(), upstream.CodeInsufficientVRAM) {
		t.Fatalf("err=%v stderr=%q; want one typed insufficient_vram marker for the batch", err, stderr.String())
	}
	if got := daemon.decides.Load(); got != 0 {
		t.Fatalf("decides = %d, want none after a batch refusal", got)
	}
}

func TestDecideBatchRejectsMalformedBatches(t *testing.T) {
	for name, body := range map[string]string{
		"no states":    `{"questions":{"q":{"type":"noul"}},"states":[]}`,
		"no questions": `{"states":["a"]}`,
		"too many":     `{"questions":{"q":{"type":"noul"}},"states":[` + strings.TrimSuffix(strings.Repeat(`"s",`, maxBatchStates+1), ",") + `]}`,
	} {
		h, _, _ := handlers(t, nil, body)
		if err := h.DecideBatch([]string{"--role", "decide.local", "--input-stdin", "--json"}); err == nil {
			t.Fatalf("%s: DecideBatch accepted a malformed batch", name)
		}
	}
	h, _, _ := handlers(t, nil, `{"questions":{},"states":["a"]}`)
	if err := h.DecideBatch([]string{"--role", "decide.local", "--input-stdin", "--concurrency", "0"}); err == nil {
		t.Fatal("DecideBatch accepted --concurrency 0")
	}
}
