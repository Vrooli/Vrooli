package searchcontrol_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"cli-health/handlers/searchcontrol"

	aisearchpkg "github.com/vrooli/aisearch-go"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

// --- fakes -----------------------------------------------------------------

type fakeReindexer struct {
	jobID     string
	upserts   int
	deletes   int
	reindexN  int
	lastScope string
	lastDry   bool

	statusOK    bool
	statusState string

	cancelled bool

	applyN      int
	lastApplied aisearchpkg.TuningConfig
	applyErr    error
}

func (f *fakeReindexer) Reindex(_ context.Context, scope string, dryRun bool) (string, int, int, error) {
	f.reindexN++
	f.lastScope = scope
	f.lastDry = dryRun
	return f.jobID, f.upserts, f.deletes, nil
}

func (f *fakeReindexer) ReindexStatus(jobID string) (string, int, int, string, bool) {
	if !f.statusOK {
		return "", 0, 0, "", false
	}
	return f.statusState, 1, 2, "", true
}

func (f *fakeReindexer) ReindexCancel(string) bool { return f.cancelled }

func (f *fakeReindexer) ApplyTuning(_ context.Context, tuning aisearchpkg.TuningConfig) (string, int, int, error) {
	f.applyN++
	f.lastApplied = tuning
	if f.applyErr != nil {
		return "", 0, 0, f.applyErr
	}
	return f.jobID, f.upserts, f.deletes, nil
}

type fakeConfigWriter struct {
	effective    aisearchpkg.TuningConfig
	idxChanged   bool
	written      bool
	err          error
	calls        int
	lastProvider string
	lastDry      bool
}

func (f *fakeConfigWriter) WriteTuning(providerID string, _ aisearchpkg.TuningConfig, dryRun bool) (aisearchpkg.TuningConfig, bool, bool, error) {
	f.calls++
	f.lastProvider = providerID
	f.lastDry = dryRun
	return f.effective, f.idxChanged, f.written, f.err
}

const testToken = "s3cr3t-control-token"

func enabledGate() *searchcontrol.Gate {
	return &searchcontrol.Gate{Token: func() string { return testToken }}
}

func newHandler(g *searchcontrol.Gate, r searchcontrol.Reindexer, w searchcontrol.ConfigWriter) interface {
	Reindex(context.Context, *connect.Request[controlv1.ReindexRequest]) (*connect.Response[controlv1.ReindexResponse], error)
	ReindexStatus(context.Context, *connect.Request[controlv1.ReindexStatusRequest]) (*connect.Response[controlv1.ReindexStatusResponse], error)
	ReindexCancel(context.Context, *connect.Request[controlv1.ReindexCancelRequest]) (*connect.Response[controlv1.ReindexCancelResponse], error)
	WriteConfig(context.Context, *connect.Request[controlv1.WriteConfigRequest]) (*connect.Response[controlv1.WriteConfigResponse], error)
} {
	return searchcontrol.NewConnectHandler(searchcontrol.Deps{
		Reindexer:    r,
		ConfigWriter: w,
		Gate:         g,
	})
}

// --- gate ------------------------------------------------------------------

func TestReindexDeniedWhenUnregistered(t *testing.T) {
	// No minted token yet (cached "") → the mutating plane is closed even when the
	// caller presents a token. Token presence is the only gate (no env flag).
	h := newHandler(&searchcontrol.Gate{Token: func() string { return "" }}, &fakeReindexer{}, &fakeConfigWriter{})
	_, err := h.Reindex(context.Background(), connect.NewRequest(&controlv1.ReindexRequest{ControlToken: testToken}))
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("unregistered provider: want PermissionDenied, got %v (%v)", connect.CodeOf(err), err)
	}
}

func TestReindexDeniedOnTokenMismatch(t *testing.T) {
	fr := &fakeReindexer{jobID: "job-1"}
	h := newHandler(enabledGate(), fr, &fakeConfigWriter{})
	_, err := h.Reindex(context.Background(), connect.NewRequest(&controlv1.ReindexRequest{ControlToken: "wrong"}))
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("bad token: want PermissionDenied, got %v", connect.CodeOf(err))
	}
	if fr.reindexN != 0 {
		t.Fatalf("reindex must not run on a denied request")
	}
}

func TestReindexDeniedOnEmptyToken(t *testing.T) {
	h := newHandler(enabledGate(), &fakeReindexer{}, &fakeConfigWriter{})
	_, err := h.Reindex(context.Background(), connect.NewRequest(&controlv1.ReindexRequest{ControlToken: ""}))
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("empty token: want PermissionDenied, got %v", connect.CodeOf(err))
	}
}

// --- reindex passthrough ---------------------------------------------------

func TestReindexAuthorizedPassesThrough(t *testing.T) {
	fr := &fakeReindexer{jobID: "job-7", upserts: 3, deletes: 1}
	h := newHandler(enabledGate(), fr, &fakeConfigWriter{})
	resp, err := h.Reindex(context.Background(), connect.NewRequest(&controlv1.ReindexRequest{
		ControlToken: testToken, Scope: "web-console", DryRun: true,
	}))
	if err != nil {
		t.Fatalf("Reindex: %v", err)
	}
	if resp.Msg.GetJobId() != "job-7" || resp.Msg.GetPlannedUpserts() != 3 || resp.Msg.GetPlannedDeletes() != 1 {
		t.Fatalf("unexpected response: %+v", resp.Msg)
	}
	if !resp.Msg.GetDryRun() {
		t.Fatalf("dry_run must echo back")
	}
	if fr.lastScope != "web-console" || !fr.lastDry {
		t.Fatalf("scope/dry-run not threaded: scope=%q dry=%v", fr.lastScope, fr.lastDry)
	}
}

func TestReindexStatusNotFound(t *testing.T) {
	h := newHandler(enabledGate(), &fakeReindexer{statusOK: false}, &fakeConfigWriter{})
	_, err := h.ReindexStatus(context.Background(), connect.NewRequest(&controlv1.ReindexStatusRequest{
		ControlToken: testToken, JobId: "missing",
	}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("want NotFound, got %v", connect.CodeOf(err))
	}
}

func TestReindexCancelAuthorized(t *testing.T) {
	h := newHandler(enabledGate(), &fakeReindexer{cancelled: true}, &fakeConfigWriter{})
	resp, err := h.ReindexCancel(context.Background(), connect.NewRequest(&controlv1.ReindexCancelRequest{
		ControlToken: testToken, JobId: "job-1",
	}))
	if err != nil {
		t.Fatalf("ReindexCancel: %v", err)
	}
	if !resp.Msg.GetCancelled() {
		t.Fatalf("expected cancelled=true")
	}
}

// --- write config ----------------------------------------------------------

func TestWriteConfigRequiresProviderID(t *testing.T) {
	fw := &fakeConfigWriter{}
	h := newHandler(enabledGate(), &fakeReindexer{}, fw)
	_, err := h.WriteConfig(context.Background(), connect.NewRequest(&controlv1.WriteConfigRequest{
		ControlToken: testToken, Tuning: &registryv1.Tuning{Engine: "dense"},
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("missing provider_id: want InvalidArgument, got %v", connect.CodeOf(err))
	}
	if fw.calls != 0 {
		t.Fatalf("writer must not be called without a provider_id")
	}
}

func TestWriteConfigRejectsInvalidTuning(t *testing.T) {
	fw := &fakeConfigWriter{}
	h := newHandler(enabledGate(), &fakeReindexer{}, fw)
	_, err := h.WriteConfig(context.Background(), connect.NewRequest(&controlv1.WriteConfigRequest{
		ControlToken: testToken, ProviderId: "cli-health.commands",
		Tuning: &registryv1.Tuning{Engine: "trie"}, // not a known engine
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("invalid tuning: want InvalidArgument, got %v", connect.CodeOf(err))
	}
	if fw.calls != 0 {
		t.Fatalf("invalid tuning must be rejected before the writer runs")
	}
}

func TestWriteConfigQueryTimeChangeNoReindex(t *testing.T) {
	fr := &fakeReindexer{jobID: "job-x"}
	fw := &fakeConfigWriter{
		effective:  aisearchpkg.CommandCorpusTuning(),
		idxChanged: false,
		written:    true,
	}
	h := newHandler(enabledGate(), fr, fw)
	resp, err := h.WriteConfig(context.Background(), connect.NewRequest(&controlv1.WriteConfigRequest{
		ControlToken: testToken, ProviderId: "cli-health.commands",
		Tuning: &registryv1.Tuning{Engine: "dense", RerankEnabled: true, RerankBlend: true, RerankShortlist: 80},
	}))
	if err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}
	if !resp.Msg.GetWritten() {
		t.Fatalf("expected written=true")
	}
	if resp.Msg.GetReindexTriggered() {
		t.Fatalf("query-time-only change must not trigger a reindex")
	}
	if fr.reindexN != 0 || fr.applyN != 0 {
		t.Fatalf("neither reindex nor apply must run for a query-time-only change (reindexN=%d applyN=%d)", fr.reindexN, fr.applyN)
	}
	if resp.Msg.GetEffective().GetRerankShortlist() != int32(aisearchpkg.CommandCorpusTuning().RerankShortlist) {
		t.Fatalf("effective tuning not echoed")
	}
}

func TestWriteConfigIndexTimeChangeAppliesTuningInProcess(t *testing.T) {
	fr := &fakeReindexer{jobID: "job-reindex-9"}
	eff := aisearchpkg.CommandCorpusTuning()
	eff.EmbedTaskPrefix = false // an index-time flip
	fw := &fakeConfigWriter{effective: eff, idxChanged: true, written: true}
	h := newHandler(enabledGate(), fr, fw)
	resp, err := h.WriteConfig(context.Background(), connect.NewRequest(&controlv1.WriteConfigRequest{
		ControlToken: testToken, ProviderId: "cli-health.commands",
		Tuning: &registryv1.Tuning{Engine: "dense", EmbedTaskPrefix: false, RerankEnabled: true, RerankBlend: true},
	}))
	if err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}
	if !resp.Msg.GetReindexTriggered() {
		t.Fatalf("index-time change must trigger an in-process apply")
	}
	if resp.Msg.GetReindexJobId() != "job-reindex-9" {
		t.Fatalf("apply job id not surfaced: %q", resp.Msg.GetReindexJobId())
	}
	// The index-time change must go through ApplyTuning (live engine rebuild +
	// re-embed), NOT the boot-recipe Reindex, and must carry the EFFECTIVE tuning.
	if fr.applyN != 1 || fr.reindexN != 0 {
		t.Fatalf("expected one in-process ApplyTuning and no boot-recipe reindex, got applyN=%d reindexN=%d", fr.applyN, fr.reindexN)
	}
	if fr.lastApplied.EmbedTaskPrefix != false || fr.lastApplied.RerankBlend != eff.RerankBlend {
		t.Fatalf("ApplyTuning got %+v, want the effective tuning %+v", fr.lastApplied, eff)
	}
}

func TestWriteConfigDryRunNoReindex(t *testing.T) {
	fr := &fakeReindexer{jobID: "job-x"}
	eff := aisearchpkg.CommandCorpusTuning()
	eff.EmbedTaskPrefix = false
	// dry run: index-time differs but writer reports written=false.
	fw := &fakeConfigWriter{effective: eff, idxChanged: true, written: false}
	h := newHandler(enabledGate(), fr, fw)
	resp, err := h.WriteConfig(context.Background(), connect.NewRequest(&controlv1.WriteConfigRequest{
		ControlToken: testToken, ProviderId: "cli-health.commands", DryRun: true,
		Tuning: &registryv1.Tuning{Engine: "dense", EmbedTaskPrefix: false, RerankEnabled: true, RerankBlend: true},
	}))
	if err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}
	if resp.Msg.GetWritten() || resp.Msg.GetReindexTriggered() {
		t.Fatalf("dry run must neither write nor reindex")
	}
	if !fw.lastDry {
		t.Fatalf("dry_run not threaded to the writer")
	}
	if fr.reindexN != 0 || fr.applyN != 0 {
		t.Fatalf("dry run must neither reindex nor apply (reindexN=%d applyN=%d)", fr.reindexN, fr.applyN)
	}
}

func TestWriteConfigUnknownProviderIsNotFound(t *testing.T) {
	fw := &fakeConfigWriter{err: aisearchpkg.ErrProviderNotInFile{ProviderID: "nope", Path: "/x/search.json"}}
	h := newHandler(enabledGate(), &fakeReindexer{}, fw)
	_, err := h.WriteConfig(context.Background(), connect.NewRequest(&controlv1.WriteConfigRequest{
		ControlToken: testToken, ProviderId: "nope",
		Tuning: &registryv1.Tuning{Engine: "dense"},
	}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("unknown provider: want NotFound, got %v", connect.CodeOf(err))
	}
}

func TestWriteConfigDeniedOnTokenMismatch(t *testing.T) {
	fw := &fakeConfigWriter{}
	h := newHandler(enabledGate(), &fakeReindexer{}, fw)
	_, err := h.WriteConfig(context.Background(), connect.NewRequest(&controlv1.WriteConfigRequest{
		ControlToken: "wrong", ProviderId: "cli-health.commands", Tuning: &registryv1.Tuning{Engine: "dense"},
	}))
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("bad token: want PermissionDenied, got %v", connect.CodeOf(err))
	}
	if fw.calls != 0 {
		t.Fatalf("writer must not run when the control token is rejected")
	}
}
