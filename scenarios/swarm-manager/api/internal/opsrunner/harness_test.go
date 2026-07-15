package opsrunner

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/opscatalog"
)

// --- in-memory domain locator (test double for FSLocator) ---

type memLocator struct{ root string }

func (m memLocator) AgentOpsDir(kind agentops.TargetKind, id string) (string, error) {
	if err := validateDomainToken(id); err != nil {
		return "", err
	}
	safe := strings.ReplaceAll(id, "/", "__")
	return filepath.Join(m.root, string(kind), safe, agentOpsSubdir), nil
}

func (m memLocator) Scan() ([]string, error) {
	var out []string
	err := filepath.WalkDir(m.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() && d.Name() == agentOpsSubdir {
			out = append(out, path)
			return fs.SkipDir
		}
		return nil
	})
	return out, err
}

// --- deterministic mode preparer (no engine, no agent) ---

type fakePreparer struct {
	// compiledMode overrides the compiled-mode bytes for a mode+revision, used to
	// simulate a source edit that changes the compiled bundle.
	compiledMode func(mode, revision string) json.RawMessage
	incompatible bool
	missingRev   bool
}

func (f fakePreparer) Prepare(_ context.Context, req PrepareRequest) (Prepared, error) {
	cm := json.RawMessage(fmt.Sprintf(`{"mode":%q,"revision":%q}`, req.Mode, req.ModeRevision))
	if f.compiledMode != nil {
		cm = f.compiledMode(req.Mode, req.ModeRevision)
	}
	inputs := json.RawMessage(`{}`)
	if len(req.CallerInputs) > 0 {
		b, err := json.Marshal(req.CallerInputs)
		if err != nil {
			return Prepared{}, err
		}
		inputs = b
	}
	return Prepared{
		Mode: req.Mode, ModeRevision: req.ModeRevision,
		CompiledMode:          cm,
		PromptCatalog:         json.RawMessage(`{"prompts":["p-alpha","p-beta"]}`),
		PromptCatalogRevision: "pc-1",
		EffectiveInputs:       inputs,
	}, nil
}

func (f fakePreparer) RevisionExists(string, string) bool { return !f.missingRev }
func (f fakePreparer) ModeCompatible(string, agentops.OperationID, agentops.TargetKind) bool {
	return !f.incompatible
}

// --- deterministic execution driver (no agent spawn) ---

type fakeDriver struct {
	outcome     string
	disposition Disposition
	result      json.RawMessage
	runID       string
}

func (f fakeDriver) Drive(_ context.Context, _ Prepared, _ RunHandle) (ExecutionOutcome, error) {
	return ExecutionOutcome{Outcome: f.outcome, Disposition: f.disposition, Result: f.result, RunID: f.runID}, nil
}

// --- in-memory run-owner index ---

type memRunOwners struct {
	mu   sync.Mutex
	refs []EvidenceRef
}

func (m *memRunOwners) IndexRunOwner(_ context.Context, ref EvidenceRef) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refs = append(m.refs, ref)
	return nil
}

// --- catalog builder ---

// writeCatalog writes a synthetic catalog to dir: the review-round contract, a
// system-default binding to a synthetic mode, and per-domain policies that fire
// open-review on an accepted outcome. Returns the pinned mode revision.
func writeCatalog(t *testing.T, dir, modeRevision string) *opscatalog.Catalog {
	t.Helper()
	write := func(rel string, v any) {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		raw, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var review agentops.OperationContract
	for _, oc := range agentops.SeedOperationContracts() {
		if oc.ID == agentops.OpReviewRound {
			review = oc
		}
	}
	write(filepath.Join(opscatalog.DirOperationContracts, "review-round.json"), review)
	write(filepath.Join(opscatalog.DirBindings, "review-default.json"), agentops.OperationBinding{
		Kind: "agentops-operation-binding", Operation: agentops.OpReviewRound,
		Layer: agentops.LayerSystemDefault, Mode: "synthetic-loop", ModeRevision: modeRevision,
	})
	for _, dk := range []string{"backlog-item", "initiative"} {
		write(filepath.Join(opscatalog.DirPolicy, dk+".json"), agentops.TransitionPolicy{
			Kind: "agentops-transition-policy", ID: dk + "-review", Version: "1.0.0", DomainKind: dk,
			Transitions: []agentops.PolicyTransition{
				{FromState: "running", OnOutcome: "accepted", Action: agentops.ActionOpenReview, ToState: "awaiting-decision"},
				{FromState: "running", OnOutcome: "failed", Action: agentops.ActionFailItem, ToState: "terminal-failed"},
			},
		})
	}
	c, err := opscatalog.Load(dir)
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	return c
}

// writeCatalogBoundTo is writeCatalog with the system-default binding pinned to
// a caller-chosen mode id (used to bind operations to real shipped modes).
func writeCatalogBoundTo(t *testing.T, dir, mode, modeRevision string) *opscatalog.Catalog {
	t.Helper()
	write := func(rel string, v any) {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		raw, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var review agentops.OperationContract
	for _, oc := range agentops.SeedOperationContracts() {
		if oc.ID == agentops.OpReviewRound {
			review = oc
		}
	}
	write(filepath.Join(opscatalog.DirOperationContracts, "review-round.json"), review)
	write(filepath.Join(opscatalog.DirBindings, "review-default.json"), agentops.OperationBinding{
		Kind: "agentops-operation-binding", Operation: agentops.OpReviewRound,
		Layer: agentops.LayerSystemDefault, Mode: mode, ModeRevision: modeRevision,
	})
	for _, dk := range []string{"backlog-item", "initiative"} {
		write(filepath.Join(opscatalog.DirPolicy, dk+".json"), agentops.TransitionPolicy{
			Kind: "agentops-transition-policy", ID: dk + "-review", Version: "1.0.0", DomainKind: dk,
			Transitions: []agentops.PolicyTransition{
				{FromState: "running", OnOutcome: "accepted", Action: agentops.ActionOpenReview, ToState: "awaiting-decision"},
			},
		})
	}
	c, err := opscatalog.Load(dir)
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	return c
}

// newRunner assembles a runner over in-memory seams rooted at storeRoot.
func newRunner(t *testing.T, catalog *opscatalog.Catalog, storeRoot string, prep ModePreparer, driver ExecutionDriver, owners RunOwnerIndex) (*Runner, *WorkflowRepo, *ExecutionStore) {
	t.Helper()
	loc := memLocator{root: storeRoot}
	repo := NewWorkflowRepo(loc)
	execStore := NewExecutionStore(loc)
	resolver := NewBindingResolver(catalog, NewFSOverrideStore(loc), preparerChecker{prep})
	dispatcher := NewDispatcher(NewActionRegistry(), repo)
	r, err := New(Config{
		Catalog: catalog, Resolver: resolver, Preparer: prep, Driver: driver,
		Repo: repo, Executions: execStore, Dispatcher: dispatcher, RunOwners: owners,
	})
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	return r, repo, execStore
}

// preparerChecker adapts a ModePreparer to the agentops.ModeChecker seam.
type preparerChecker struct{ p ModePreparer }

func (c preparerChecker) RevisionExists(mode, rev string) bool { return c.p.RevisionExists(mode, rev) }
func (c preparerChecker) ModeCompatible(mode string, op agentops.OperationID, target agentops.TargetKind) bool {
	return c.p.ModeCompatible(mode, op, target)
}

// writeContractOnly authors only the review-round contract (no binding) in a
// fresh dir, so binding resolution fails closed with ErrNoBinding.
func writeContractOnly(t *testing.T, dir string) *opscatalog.Catalog {
	t.Helper()
	var review agentops.OperationContract
	for _, oc := range agentops.SeedOperationContracts() {
		if oc.ID == agentops.OpReviewRound {
			review = oc
		}
	}
	p := filepath.Join(dir, opscatalog.DirOperationContracts, "review-round.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(review, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := opscatalog.Load(dir)
	if err != nil {
		t.Fatalf("load contract-only catalog: %v", err)
	}
	return c
}
