package report

import (
	"context"
	"testing"

	"development-toolchain-validator/domain/expectation"
	"development-toolchain-validator/domain/skill"
)

// --- mocks ---

type mockConnectionLister struct {
	conns []*skill.Connection
	err   error
}

func (m *mockConnectionLister) List(_ context.Context, opts skill.ListOptions) ([]*skill.Connection, error) {
	if m.err != nil {
		return nil, m.err
	}
	var out []*skill.Connection
	for _, c := range m.conns {
		if opts.ReferenceID != "" && c.ReferenceID != opts.ReferenceID {
			continue
		}
		if opts.SkillID != "" && c.SkillID != opts.SkillID {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

type mockExpectationLister struct {
	structural map[string][]*expectation.StructuralExpectation // keyed by connection_id
	cli        map[string][]*expectation.CLIAssertion          // keyed by connection_id
}

func (m *mockExpectationLister) ListStructural(_ context.Context, opts expectation.ListOptions) ([]*expectation.StructuralExpectation, error) {
	return m.structural[opts.ConnectionID], nil
}

func (m *mockExpectationLister) ListCLI(_ context.Context, opts expectation.ListOptions) ([]*expectation.CLIAssertion, error) {
	return m.cli[opts.ConnectionID], nil
}

type mockResultReader struct {
	results map[string][]*CLIResultRow // keyed by reference_id
}

func (m *mockResultReader) CLIResultsByReference(_ context.Context, refID string) ([]*CLIResultRow, error) {
	return m.results[refID], nil
}

// --- tests ---

func TestConflicts_NoConnections(t *testing.T) {
	svc := NewService(
		&mockConnectionLister{},
		&mockExpectationLister{},
		&mockResultReader{},
	)

	rpt, err := svc.Conflicts(context.Background(), ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rpt.TotalCount != 0 {
		t.Errorf("expected 0 conflicts, got %d", rpt.TotalCount)
	}
}

func TestConflicts_StructuralConflict(t *testing.T) {
	conns := []*skill.Connection{
		{ID: "c1", ReferenceID: "ref1", SkillID: "skill-a"},
		{ID: "c2", ReferenceID: "ref1", SkillID: "skill-b"},
	}

	exps := &mockExpectationLister{
		structural: map[string][]*expectation.StructuralExpectation{
			"c1": {{ID: "e1", ConnectionID: "c1", Type: "file", Pattern: "src/main.go", Required: true}},
			"c2": {{ID: "e2", ConnectionID: "c2", Type: "file", Pattern: "src/main.go", Required: false}},
		},
		cli: map[string][]*expectation.CLIAssertion{},
	}

	svc := NewService(&mockConnectionLister{conns: conns}, exps, &mockResultReader{})

	rpt, err := svc.Conflicts(context.Background(), ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rpt.TotalCount != 1 {
		t.Fatalf("expected 1 conflict, got %d", rpt.TotalCount)
	}
	c := rpt.Conflicts[0]
	if c.Type != ConflictStructural {
		t.Errorf("expected structural conflict, got %s", c.Type)
	}
	if c.SkillA != "skill-a" || c.SkillB != "skill-b" {
		t.Errorf("unexpected skills: %s, %s", c.SkillA, c.SkillB)
	}
}

func TestConflicts_CLIConflict(t *testing.T) {
	conns := []*skill.Connection{
		{ID: "c1", ReferenceID: "ref1", SkillID: "skill-a"},
		{ID: "c2", ReferenceID: "ref1", SkillID: "skill-b"},
	}

	exps := &mockExpectationLister{
		structural: map[string][]*expectation.StructuralExpectation{},
		cli: map[string][]*expectation.CLIAssertion{
			"c1": {{ID: "a1", ConnectionID: "c1", Command: "scenario-auditor --json", JSONPath: "$.score", ExpectedValue: 100}},
			"c2": {{ID: "a2", ConnectionID: "c2", Command: "scenario-auditor --json", JSONPath: "$.score", ExpectedValue: 90}},
		},
	}

	svc := NewService(&mockConnectionLister{conns: conns}, exps, &mockResultReader{})

	rpt, err := svc.Conflicts(context.Background(), ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rpt.TotalCount != 1 {
		t.Fatalf("expected 1 conflict, got %d", rpt.TotalCount)
	}
	if rpt.Conflicts[0].Type != ConflictCLI {
		t.Errorf("expected CLI conflict, got %s", rpt.Conflicts[0].Type)
	}
}

func TestConflicts_NoConflictWhenSameValues(t *testing.T) {
	conns := []*skill.Connection{
		{ID: "c1", ReferenceID: "ref1", SkillID: "skill-a"},
		{ID: "c2", ReferenceID: "ref1", SkillID: "skill-b"},
	}

	exps := &mockExpectationLister{
		structural: map[string][]*expectation.StructuralExpectation{
			"c1": {{ID: "e1", ConnectionID: "c1", Type: "file", Pattern: "src/main.go", Required: true}},
			"c2": {{ID: "e2", ConnectionID: "c2", Type: "file", Pattern: "src/main.go", Required: true}},
		},
		cli: map[string][]*expectation.CLIAssertion{},
	}

	svc := NewService(&mockConnectionLister{conns: conns}, exps, &mockResultReader{})

	rpt, err := svc.Conflicts(context.Background(), ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rpt.TotalCount != 0 {
		t.Errorf("expected 0 conflicts, got %d", rpt.TotalCount)
	}
}

func TestConflicts_SingleConnectionPerRef(t *testing.T) {
	conns := []*skill.Connection{
		{ID: "c1", ReferenceID: "ref1", SkillID: "skill-a"},
		{ID: "c2", ReferenceID: "ref2", SkillID: "skill-b"},
	}

	svc := NewService(
		&mockConnectionLister{conns: conns},
		&mockExpectationLister{structural: map[string][]*expectation.StructuralExpectation{}, cli: map[string][]*expectation.CLIAssertion{}},
		&mockResultReader{},
	)

	rpt, err := svc.Conflicts(context.Background(), ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rpt.TotalCount != 0 {
		t.Errorf("expected 0 conflicts (single connection per ref), got %d", rpt.TotalCount)
	}
}

func TestDrift_DetectsDriftedConnections(t *testing.T) {
	conns := []*skill.Connection{
		{ID: "c1", ReferenceID: "ref1", SkillID: "skill-a", SkillContentHash: "hash-old", SkillVersion: "1.0"},
		{ID: "c2", ReferenceID: "ref1", SkillID: "skill-b", SkillContentHash: "hash-same", SkillVersion: "2.0"},
	}

	svc := NewService(&mockConnectionLister{conns: conns}, &mockExpectationLister{}, &mockResultReader{})

	hashes := map[string]string{
		"skill-a":         "hash-new",
		"skill-a_version": "1.1",
		"skill-b":         "hash-same",
		"skill-b_version": "2.0",
	}

	rpt, err := svc.Drift(context.Background(), ListOptions{}, hashes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rpt.TotalConnections != 2 {
		t.Errorf("expected 2 total connections, got %d", rpt.TotalConnections)
	}
	if rpt.DriftedCount != 1 {
		t.Fatalf("expected 1 drifted, got %d", rpt.DriftedCount)
	}
	d := rpt.DriftedConnections[0]
	if d.SkillID != "skill-a" {
		t.Errorf("expected skill-a drifted, got %s", d.SkillID)
	}
	if !d.ContentChanged {
		t.Error("expected content changed")
	}
	if !d.VersionChanged {
		t.Error("expected version changed")
	}
}

func TestDrift_NoDrift(t *testing.T) {
	conns := []*skill.Connection{
		{ID: "c1", ReferenceID: "ref1", SkillID: "skill-a", SkillContentHash: "hash1"},
	}

	svc := NewService(&mockConnectionLister{conns: conns}, &mockExpectationLister{}, &mockResultReader{})

	rpt, err := svc.Drift(context.Background(), ListOptions{}, map[string]string{"skill-a": "hash1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rpt.DriftedCount != 0 {
		t.Errorf("expected 0 drifted, got %d", rpt.DriftedCount)
	}
}

func TestMaturity_HighMaturity(t *testing.T) {
	conns := []*skill.Connection{
		{ID: "c1", ReferenceID: "ref1", SkillID: "skill-a"},
	}

	exps := &mockExpectationLister{
		structural: map[string][]*expectation.StructuralExpectation{
			"c1": {{ID: "e1"}, {ID: "e2"}, {ID: "e3"}, {ID: "e4"}, {ID: "e5"}},
		},
		cli: map[string][]*expectation.CLIAssertion{
			"c1": {{ID: "a1"}, {ID: "a2"}, {ID: "a3"}, {ID: "a4"}, {ID: "a5"}},
		},
	}

	svc := NewService(&mockConnectionLister{conns: conns}, exps, &mockResultReader{})

	rpt, err := svc.Maturity(context.Background(), ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rpt.Skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(rpt.Skills))
	}
	sm := rpt.Skills[0]
	if sm.Level != MaturityHigh {
		t.Errorf("expected high maturity, got %s", sm.Level)
	}
	if sm.Score != 1.0 {
		t.Errorf("expected score 1.0, got %f", sm.Score)
	}
	if rpt.Distribution[MaturityHigh] != 1 {
		t.Errorf("expected 1 high in distribution, got %d", rpt.Distribution[MaturityHigh])
	}
}

func TestMaturity_LowMaturity(t *testing.T) {
	conns := []*skill.Connection{
		{ID: "c1", ReferenceID: "ref1", SkillID: "skill-a"},
	}

	exps := &mockExpectationLister{
		structural: map[string][]*expectation.StructuralExpectation{},
		cli:        map[string][]*expectation.CLIAssertion{},
	}

	svc := NewService(&mockConnectionLister{conns: conns}, exps, &mockResultReader{})

	rpt, err := svc.Maturity(context.Background(), ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sm := rpt.Skills[0]
	if sm.Level != MaturityLow {
		t.Errorf("expected low maturity, got %s", sm.Level)
	}
	if sm.Score != 0.0 {
		t.Errorf("expected score 0.0, got %f", sm.Score)
	}
}

func TestMaturity_MediumMaturity(t *testing.T) {
	conns := []*skill.Connection{
		{ID: "c1", ReferenceID: "ref1", SkillID: "skill-a"},
	}

	exps := &mockExpectationLister{
		structural: map[string][]*expectation.StructuralExpectation{
			"c1": {{ID: "e1"}},
		},
		cli: map[string][]*expectation.CLIAssertion{},
	}

	svc := NewService(&mockConnectionLister{conns: conns}, exps, &mockResultReader{})

	rpt, err := svc.Maturity(context.Background(), ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sm := rpt.Skills[0]
	if sm.Level != MaturityMedium {
		t.Errorf("expected medium maturity, got %s", sm.Level)
	}
}

func TestToolBaselines_AllPass(t *testing.T) {
	conns := []*skill.Connection{
		{ID: "c1", ReferenceID: "ref1", SkillID: "skill-a"},
	}

	results := &mockResultReader{
		results: map[string][]*CLIResultRow{
			"ref1": {
				{AssertionID: "a1", ConnectionID: "c1", Command: "scenario-auditor --json", Status: "pass"},
				{AssertionID: "a2", ConnectionID: "c1", Command: "scenario-auditor --json", Status: "pass"},
			},
		},
	}

	svc := NewService(&mockConnectionLister{conns: conns}, &mockExpectationLister{}, results)

	rpt, err := svc.ToolBaselines(context.Background(), ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rpt.TotalTools != 1 {
		t.Fatalf("expected 1 tool, got %d", rpt.TotalTools)
	}
	if rpt.PassingTools != 1 {
		t.Errorf("expected 1 passing, got %d", rpt.PassingTools)
	}
	bl := rpt.Baselines[0]
	if bl.Status != BaselinePass {
		t.Errorf("expected pass, got %s", bl.Status)
	}
	if bl.ToolName != "scenario-auditor" {
		t.Errorf("expected tool name scenario-auditor, got %s", bl.ToolName)
	}
}

func TestToolBaselines_WithFailures(t *testing.T) {
	conns := []*skill.Connection{
		{ID: "c1", ReferenceID: "ref1", SkillID: "skill-a"},
	}

	results := &mockResultReader{
		results: map[string][]*CLIResultRow{
			"ref1": {
				{AssertionID: "a1", ConnectionID: "c1", Command: "test-genie --json", Status: "pass"},
				{AssertionID: "a2", ConnectionID: "c1", Command: "test-genie --json", Status: "fail"},
			},
		},
	}

	svc := NewService(&mockConnectionLister{conns: conns}, &mockExpectationLister{}, results)

	rpt, err := svc.ToolBaselines(context.Background(), ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rpt.FailingTools != 1 {
		t.Errorf("expected 1 failing, got %d", rpt.FailingTools)
	}
	if rpt.Baselines[0].Status != BaselineFail {
		t.Errorf("expected fail, got %s", rpt.Baselines[0].Status)
	}
}

func TestToolBaselines_NoResults(t *testing.T) {
	conns := []*skill.Connection{
		{ID: "c1", ReferenceID: "ref1", SkillID: "skill-a"},
	}

	svc := NewService(&mockConnectionLister{conns: conns}, &mockExpectationLister{}, &mockResultReader{})

	rpt, err := svc.ToolBaselines(context.Background(), ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rpt.TotalTools != 0 {
		t.Errorf("expected 0 tools, got %d", rpt.TotalTools)
	}
}

func TestExtractToolName(t *testing.T) {
	tests := []struct {
		command string
		want    string
	}{
		{"scenario-auditor --json", "scenario-auditor"},
		{"test-genie run --format json", "test-genie"},
		{"singlecommand", "singlecommand"},
		{"cmd\targ", "cmd"},
	}
	for _, tt := range tests {
		got := extractToolName(tt.command)
		if got != tt.want {
			t.Errorf("extractToolName(%q) = %q, want %q", tt.command, got, tt.want)
		}
	}
}

func TestComputeMaturityScore(t *testing.T) {
	tests := []struct {
		hasStructural bool
		hasCLI        bool
		total         int
		want          float64
	}{
		{false, false, 0, 0.0},
		{true, false, 1, 0.42},
		{false, true, 1, 0.42},
		{true, true, 10, 1.0},
		{true, true, 20, 1.0}, // cap at 10
		{true, true, 5, 0.9},
	}
	for _, tt := range tests {
		got := computeMaturityScore(tt.hasStructural, tt.hasCLI, tt.total)
		if abs(got-tt.want) > 0.01 {
			t.Errorf("computeMaturityScore(%v, %v, %d) = %f, want %f",
				tt.hasStructural, tt.hasCLI, tt.total, got, tt.want)
		}
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
