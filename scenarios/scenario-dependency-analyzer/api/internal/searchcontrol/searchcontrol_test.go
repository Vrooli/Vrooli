package searchcontrol

import (
	"context"
	"log"
	"testing"

	"connectrpc.com/connect"

	aisearchpkg "github.com/vrooli/ai-go/search"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

type fakeReindexer struct {
	applied   []string // provider IDs ApplyTuning was called for
	reindexed bool
}

func (f *fakeReindexer) Reindex(context.Context, string, bool) (string, int, int, error) {
	f.reindexed = true
	return "job-1", 3, 0, nil
}

func (f *fakeReindexer) ReindexStatus(string) (string, int, int, string, bool) {
	return "completed", 3, 3, "", true
}
func (f *fakeReindexer) ReindexCancel(string) bool { return false }
func (f *fakeReindexer) ApplyTuning(_ context.Context, providerID string, _ aisearchpkg.TuningConfig) error {
	f.applied = append(f.applied, providerID)
	return nil
}

type fakeConfigWriter struct {
	gotProvider string
}

func (w *fakeConfigWriter) WriteTuning(providerID string, tuning aisearchpkg.TuningConfig, _ bool) (aisearchpkg.TuningConfig, bool, bool, error) {
	w.gotProvider = providerID
	return tuning, false, true, nil
}

// newHandler wires a handler whose token store maps both the scenarios and
// resources leaves to the same test token (so authorizeAny + authorizeProvider
// both resolve), plus a default "" for absent providers.
func newHandler(token string, ri Reindexer, cw ConfigWriter) *connectHandler {
	store := NewTokenStore()
	if token != "" {
		store.Set("scenario-dependency-analyzer.scenarios", token)
		store.Set("scenario-dependency-analyzer.resources", token)
		store.Set("scenario-dependency-analyzer.dependencies", token)
	}
	return &connectHandler{deps: Deps{
		Logger:       log.Default(),
		Reindexer:    ri,
		ConfigWriter: cw,
		Gate:         &Gate{Tokens: store},
	}}
}

// TestControlTokenGate verifies the gate denies an empty or mismatched token and
// admits the exact match — the whole security model of the control plane.
func TestControlTokenGate(t *testing.T) {
	h := newHandler("secret", &fakeReindexer{}, &fakeConfigWriter{})

	// empty presented token → denied even with a real cached token
	_, err := h.Reindex(context.Background(), connect.NewRequest(&controlv1.ReindexRequest{}))
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("empty token: code = %v, want permission_denied", connect.CodeOf(err))
	}

	// mismatch → denied
	_, err = h.Reindex(context.Background(), connect.NewRequest(&controlv1.ReindexRequest{ControlToken: "wrong"}))
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("mismatch: code = %v, want permission_denied", connect.CodeOf(err))
	}

	// exact match → admitted
	resp, err := h.Reindex(context.Background(), connect.NewRequest(&controlv1.ReindexRequest{ControlToken: "secret"}))
	if err != nil {
		t.Fatalf("exact token: unexpected error %v", err)
	}
	if resp.Msg.GetJobId() != "job-1" {
		t.Fatalf("job id = %q", resp.Msg.GetJobId())
	}
}

// TestEmptyCachedTokenAlwaysDenies confirms an un-minted (empty cached) token
// denies even when the caller presents an empty token (no "" == "" admit).
func TestEmptyCachedTokenAlwaysDenies(t *testing.T) {
	h := newHandler("", &fakeReindexer{}, &fakeConfigWriter{})
	_, err := h.Reindex(context.Background(), connect.NewRequest(&controlv1.ReindexRequest{ControlToken: ""}))
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("code = %v, want permission_denied", connect.CodeOf(err))
	}
}

// TestWriteConfigRoutesByProvider verifies WriteConfig requires a provider_id,
// persists for that provider, and applies the live tuning to the same provider.
func TestWriteConfigRoutesByProvider(t *testing.T) {
	ri := &fakeReindexer{}
	cw := &fakeConfigWriter{}
	h := newHandler("secret", ri, cw)

	// missing provider_id → invalid argument
	_, err := h.WriteConfig(context.Background(), connect.NewRequest(&controlv1.WriteConfigRequest{ControlToken: "secret"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("missing provider: code = %v, want invalid_argument", connect.CodeOf(err))
	}

	const provider = "scenario-dependency-analyzer.scenarios"
	resp, err := h.WriteConfig(context.Background(), connect.NewRequest(&controlv1.WriteConfigRequest{
		ControlToken: "secret",
		ProviderId:   provider,
		Tuning:       &registryv1.Tuning{Engine: "dense"},
	}))
	if err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}
	if !resp.Msg.GetWritten() {
		t.Fatal("written = false, want true")
	}
	if cw.gotProvider != provider {
		t.Fatalf("config writer got provider %q, want %q", cw.gotProvider, provider)
	}
	if len(ri.applied) != 1 || ri.applied[0] != provider {
		t.Fatalf("ApplyTuning routed to %v, want [%s]", ri.applied, provider)
	}
}
