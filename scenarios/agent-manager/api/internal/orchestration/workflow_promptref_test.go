package orchestration

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"agent-manager/internal/promptmanager"
	"agent-manager/internal/testutil"
)

// fakePromptSource is a prompt-manager stand-in implementing both the read and
// source-snapshot seams so promptRef resolution can be exercised without a live
// prompt-manager.
type fakePromptSource struct {
	content   string
	hash      string
	rev       int
	err       error
	calls     int
	variables map[string]string
	withScope bool
}

func (f *fakePromptSource) ReadSkill(_ context.Context, _ string, _ map[string]string, _ bool) (string, error) {
	return f.content, f.err
}

func (f *fakePromptSource) ReadSkillSource(_ context.Context, skillID string, variables map[string]string, withScope bool) (promptmanager.SkillSourceSnapshot, error) {
	f.calls++
	f.variables = clonePromptVariables(variables)
	f.withScope = withScope
	if f.err != nil {
		return promptmanager.SkillSourceSnapshot{}, f.err
	}
	return promptmanager.SkillSourceSnapshot{SkillID: skillID, Revision: f.rev, VariantID: "control", Content: f.content, ContentHash: f.hash}, nil
}

func newPromptRefOrchestrator(t *testing.T, client promptmanager.Client) *Orchestrator {
	t.Helper()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	opts := []Option{
		WithEvents(eventStore),
		WithWorkflowRepository(repos.Workflows),
		WithWorkflowExecutionRepository(repos.WorkflowExecutions),
	}
	if client != nil {
		opts = append(opts, WithPromptClient(client))
	}
	return New(repos.Profiles, repos.Tasks, repos.Runs, opts...)
}

// promptRefWorkflow is a single-node declaration (sugar) whose run node resolves
// its prompt from prompt-manager instead of inlining it.
const promptRefWorkflow = `{
  "schemaVersion": "agent-workflow/v1",
  "owner": "fixture-scn",
  "key": "fixture-scn/refround",
  "version": "1.0.0",
  "description": "promptRef fixture",
  "inputSchema": {"type": "object", "additionalProperties": true},
  "outputSchema": {"type": "object", "properties": {"result": {"type": "object", "additionalProperties": true}}, "required": ["result"], "additionalProperties": false},
  "nodes": [
    {"id": "work", "kind": "run", "run": {"profileKey": "fixture-scn/default", "promptRef": {"skillId": "fixture-skill"}, "maxTurns": 5, "timeoutSeconds": 300}}
  ],
  "budgets": {"wallTimeSeconds": 1200, "maxTurns": 12, "maxTokens": 30000, "maxCostUsd": 5, "maxNodeAttempts": 2, "maxChildren": 1, "maxConcurrency": 1, "maxRecursion": 1, "maxRetries": 1, "maxWaitSeconds": 60}
}`

func TestPromptRefResolvesAndPinsProvenance(t *testing.T) {
	prompt := &fakePromptSource{content: "Resolved prompt body.", hash: "sha256:abc", rev: 3}
	o := newPromptRefOrchestrator(t, prompt)
	ctx := context.Background()
	scenarioRoot, servicePath := writeScenarioFixture(t, map[string]string{
		".vrooli/service.json":               declarationManifest(".vrooli/agent-manager/default.json", ".vrooli/agent-manager/ref.json"),
		".vrooli/agent-manager/default.json": fixtureProfile,
		".vrooli/agent-manager/ref.json":     promptRefWorkflow,
	})
	res, err := o.reconcileScenarioDeclarationsAt(ctx, "fixture-scn", scenarioRoot, servicePath, false, false)
	if err != nil {
		t.Fatalf("reconcile error = %v", err)
	}
	if res.WorkflowsCreated != 1 || res.WorkflowsFailed != 0 {
		t.Fatalf("workflows created=%d failed=%d results=%+v", res.WorkflowsCreated, res.WorkflowsFailed, res.WorkflowResults)
	}
	if prompt.calls != 1 {
		t.Fatalf("expected one skill resolution, got %d", prompt.calls)
	}
	revision, err := o.workflows.GetActive(ctx, "fixture-scn", "fixture-scn/refround")
	if err != nil || revision == nil {
		t.Fatalf("active revision: %v %v", revision, err)
	}
	run := revision.Definition.Nodes[0].Run
	if run.PromptRef != nil {
		t.Fatalf("promptRef was not cleared after resolution: %+v", run.PromptRef)
	}
	if run.PromptTemplate != "Resolved prompt body." {
		t.Fatalf("resolved content not embedded: %q", run.PromptTemplate)
	}
	if run.PromptProvenance == nil || run.PromptProvenance.SkillID != "fixture-skill" || run.PromptProvenance.ContentHash != "sha256:abc" || run.PromptProvenance.Revision != 3 {
		t.Fatalf("provenance not pinned: %+v", run.PromptProvenance)
	}
	if revision.Digest == "" {
		t.Fatal("resolved revision earned no digest")
	}
}

func TestWorkflowPromptStalenessFlipsAndReconcileClears(t *testing.T) {
	prompt := &fakePromptSource{content: "First body.", hash: "sha256:one", rev: 1}
	o := newPromptRefOrchestrator(t, prompt)
	ctx := context.Background()
	declaration := strings.Replace(promptRefWorkflow, `"skillId": "fixture-skill"`, `"skillId": "fixture-skill", "variables": {"project": "alpha"}, "withScope": true`, 1)
	scenarioRoot, servicePath := writeScenarioFixture(t, map[string]string{
		".vrooli/service.json":               declarationManifest(".vrooli/agent-manager/default.json", ".vrooli/agent-manager/ref.json"),
		".vrooli/agent-manager/default.json": fixtureProfile,
		".vrooli/agent-manager/ref.json":     declaration,
	})
	if _, err := o.reconcileScenarioDeclarationsAt(ctx, "fixture-scn", scenarioRoot, servicePath, false, false); err != nil {
		t.Fatalf("initial reconcile: %v", err)
	}
	current, err := o.GetWorkflowRevision(ctx, "fixture-scn", "fixture-scn/refround", "")
	if err != nil || current.PromptStale {
		t.Fatalf("initial prompt status: revision=%+v err=%v", current, err)
	}
	if prompt.variables["project"] != "alpha" || !prompt.withScope {
		t.Fatalf("staleness read did not retain promptRef contract: variables=%v withScope=%v", prompt.variables, prompt.withScope)
	}
	prompt.content, prompt.hash, prompt.rev = "Changed body.", "sha256:two", 2
	stale, err := o.GetWorkflowRevision(ctx, "fixture-scn", "fixture-scn/refround", "")
	if err != nil || !stale.PromptStale {
		t.Fatalf("changed prompt should be stale: revision=%+v err=%v", stale, err)
	}
	if _, err := o.reconcileScenarioDeclarationsAt(ctx, "fixture-scn", scenarioRoot, servicePath, false, false); err != nil {
		t.Fatalf("reconcile changed prompt: %v", err)
	}
	resolved, err := o.GetWorkflowRevision(ctx, "fixture-scn", "fixture-scn/refround", "")
	if err != nil || resolved.PromptStale {
		t.Fatalf("reconciled prompt should be current: revision=%+v err=%v", resolved, err)
	}
}

func TestPromptRefChangedSkillProducesNewRevision(t *testing.T) {
	prompt := &fakePromptSource{content: "First body.", hash: "sha256:one", rev: 1}
	o := newPromptRefOrchestrator(t, prompt)
	ctx := context.Background()
	scenarioRoot, servicePath := writeScenarioFixture(t, map[string]string{
		".vrooli/service.json":               declarationManifest(".vrooli/agent-manager/default.json", ".vrooli/agent-manager/ref.json"),
		".vrooli/agent-manager/default.json": fixtureProfile,
		".vrooli/agent-manager/ref.json":     promptRefWorkflow,
	})
	first, err := o.reconcileScenarioDeclarationsAt(ctx, "fixture-scn", scenarioRoot, servicePath, false, false)
	if err != nil || first.WorkflowsCreated != 1 {
		t.Fatalf("first reconcile: created=%d err=%v", first.WorkflowsCreated, err)
	}
	firstDigest := first.WorkflowResults[0].Digest

	// The source file is byte-identical; only the skill content changed. The new
	// resolved content must yield a new digest and a fresh activated revision.
	prompt.content = "Second body."
	prompt.hash = "sha256:two"
	prompt.rev = 2
	second, err := o.reconcileScenarioDeclarationsAt(ctx, "fixture-scn", scenarioRoot, servicePath, false, false)
	if err != nil {
		t.Fatalf("second reconcile: %v", err)
	}
	if second.WorkflowResults[0].Digest == firstDigest {
		t.Fatalf("changed skill did not produce a new digest: %s", firstDigest)
	}
	if second.WorkflowResults[0].Status != WorkflowReconcileActivated {
		t.Fatalf("expected activated on skill change, got %s", second.WorkflowResults[0].Status)
	}
}

func TestPromptRefResolutionFailureWithholdsRevision(t *testing.T) {
	prompt := &fakePromptSource{err: context.DeadlineExceeded}
	o := newPromptRefOrchestrator(t, prompt)
	ctx := context.Background()
	scenarioRoot, servicePath := writeScenarioFixture(t, map[string]string{
		".vrooli/service.json":               declarationManifest(".vrooli/agent-manager/default.json", ".vrooli/agent-manager/ref.json"),
		".vrooli/agent-manager/default.json": fixtureProfile,
		".vrooli/agent-manager/ref.json":     promptRefWorkflow,
	})
	res, err := o.reconcileScenarioDeclarationsAt(ctx, "fixture-scn", scenarioRoot, servicePath, false, false)
	if err != nil {
		t.Fatalf("reconcile error = %v", err)
	}
	if res.WorkflowsFailed != 1 || res.WorkflowsCreated != 0 {
		t.Fatalf("expected a withheld revision: created=%d failed=%d results=%+v", res.WorkflowsCreated, res.WorkflowsFailed, res.WorkflowResults)
	}
	revision, err := o.workflows.GetActive(ctx, "fixture-scn", "fixture-scn/refround")
	if err != nil {
		t.Fatal(err)
	}
	if revision != nil {
		t.Fatalf("a partial revision was registered despite resolution failure: %+v", revision)
	}
}

// [REQ:REQ-P2-005] TestPromptRefMissingSkillWithholdsRevision covers the named adversarial case:
// a promptRef pointing at a skill prompt-manager cannot resolve (not found) fails
// the source and withholds the whole atomic batch, exactly like an unreachable
// prompt-manager — never a partial revision with an empty prompt.
func TestPromptRefMissingSkillWithholdsRevision(t *testing.T) {
	prompt := &fakePromptSource{err: fmt.Errorf("skill \"fixture-skill\" not found")}
	o := newPromptRefOrchestrator(t, prompt)
	ctx := context.Background()
	scenarioRoot, servicePath := writeScenarioFixture(t, map[string]string{
		".vrooli/service.json":               declarationManifest(".vrooli/agent-manager/default.json", ".vrooli/agent-manager/ref.json"),
		".vrooli/agent-manager/default.json": fixtureProfile,
		".vrooli/agent-manager/ref.json":     promptRefWorkflow,
	})
	res, err := o.reconcileScenarioDeclarationsAt(ctx, "fixture-scn", scenarioRoot, servicePath, false, false)
	if err != nil {
		t.Fatalf("reconcile error = %v", err)
	}
	if res.WorkflowsFailed != 1 || res.WorkflowsCreated != 0 {
		t.Fatalf("missing skill must withhold the revision: %+v", res.WorkflowResults)
	}
	if !strings.Contains(res.WorkflowResults[0].Message, "not found") {
		t.Fatalf("expected a not-found diagnostic, got %q", res.WorkflowResults[0].Message)
	}
	revision, err := o.workflows.GetActive(ctx, "fixture-scn", "fixture-scn/refround")
	if err != nil {
		t.Fatal(err)
	}
	if revision != nil {
		t.Fatalf("a partial revision registered despite a missing skill: %+v", revision)
	}
}

func TestPromptRefRequiresSourceClient(t *testing.T) {
	o := newPromptRefOrchestrator(t, nil)
	ctx := context.Background()
	scenarioRoot, servicePath := writeScenarioFixture(t, map[string]string{
		".vrooli/service.json":               declarationManifest(".vrooli/agent-manager/default.json", ".vrooli/agent-manager/ref.json"),
		".vrooli/agent-manager/default.json": fixtureProfile,
		".vrooli/agent-manager/ref.json":     promptRefWorkflow,
	})
	res, err := o.reconcileScenarioDeclarationsAt(ctx, "fixture-scn", scenarioRoot, servicePath, false, false)
	if err != nil {
		t.Fatalf("reconcile error = %v", err)
	}
	if res.WorkflowsFailed != 1 || res.WorkflowsCreated != 0 {
		t.Fatalf("expected failure without a source client: %+v", res.WorkflowResults)
	}
	if !strings.Contains(res.WorkflowResults[0].Message, "source client") {
		t.Fatalf("unexpected message: %q", res.WorkflowResults[0].Message)
	}
}
