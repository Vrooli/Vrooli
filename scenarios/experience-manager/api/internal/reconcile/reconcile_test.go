package reconcile

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"experience-manager/internal/spec"
	testdb "experience-manager/internal/testutil/db"

	apidb "github.com/vrooli/api-core/database"
)

func TestBASCapturerLiveCaptureContract(t *testing.T) {
	baseURL := strings.TrimSpace(os.Getenv("EXPERIENCE_BAS_INTEGRATION_URL"))
	if baseURL == "" {
		t.Skip("set EXPERIENCE_BAS_INTEGRATION_URL to exercise the live BAS CaptureService contract")
	}
	capturer := BASCapturer{Resolve: func(context.Context) (string, error) { return baseURL, nil }}
	snapshot, err := capturer.CaptureAccessibility(context.Background(), CaptureTarget{
		Scenario:       "landing-page-business-suite",
		Route:          "/admin/login",
		PageID:         "admin-login",
		StateID:        "default",
		ViewportID:     "desktop",
		ViewportWidth:  1280,
		ViewportHeight: 720,
	})
	if err != nil {
		t.Fatalf("live BAS CaptureAccessibility: %v", err)
	}
	if snapshot.Contract != snapshotContract || len(snapshot.Flatten()) == 0 {
		t.Fatalf("live BAS snapshot = contract %q, nodes %d; want %q and nodes", snapshot.Contract, len(snapshot.Flatten()), snapshotContract)
	}
}

func TestParseCaptureMillisecondsAcceptsConnectAndLegacyJSON(t *testing.T) {
	for _, tc := range []struct {
		raw  json.RawMessage
		want int64
	}{
		{raw: json.RawMessage(`210`), want: 210},
		{raw: json.RawMessage(`"210"`), want: 210},
		{raw: json.RawMessage(`null`), want: 0},
		{raw: json.RawMessage(`"invalid"`), want: 0},
	} {
		if got := parseCaptureMilliseconds(tc.raw); got != tc.want {
			t.Errorf("parseCaptureMilliseconds(%s) = %d, want %d", tc.raw, got, tc.want)
		}
	}
}

type fakeCapturer struct {
	snapshot         Snapshot
	snapshotsByState map[string]Snapshot
	err              error
	targets          *[]CaptureTarget
	mu               *sync.Mutex
}

func (f fakeCapturer) CaptureAccessibility(_ context.Context, target CaptureTarget) (Snapshot, error) {
	if f.targets != nil {
		if f.mu != nil {
			f.mu.Lock()
			defer f.mu.Unlock()
		}
		*f.targets = append(*f.targets, target)
	}
	snapshot := f.snapshot
	if byState, ok := f.snapshotsByState[target.StateID]; ok {
		snapshot = byState
	}
	if snapshot.Contract == snapshotContract && snapshot.Root.Bounds == nil && target.ViewportWidth > 0 && target.ViewportHeight > 0 {
		snapshot.Root.Bounds = &Bounds{Width: float64(target.ViewportWidth), Height: float64(target.ViewportHeight)}
	}
	return snapshot, f.err
}

type concurrentCapturer struct {
	snapshot Snapshot
	mu       sync.Mutex
	inFlight int
	max      int
}

func (c *concurrentCapturer) CaptureAccessibility(_ context.Context, target CaptureTarget) (Snapshot, error) {
	c.mu.Lock()
	c.inFlight++
	if c.inFlight > c.max {
		c.max = c.inFlight
	}
	c.mu.Unlock()
	time.Sleep(30 * time.Millisecond)
	c.mu.Lock()
	c.inFlight--
	c.mu.Unlock()
	snapshot := c.snapshot
	snapshot.Root.Bounds = &Bounds{Width: float64(target.ViewportWidth), Height: float64(target.ViewportHeight)}
	return snapshot, nil
}

func TestDraftCalibrationEmitsExpectedMatrixFailuresOnly(t *testing.T) { // [REQ:EXPERIEN-P0-003]
	report, err := spec.ParseScenario(filepath.Join(repoRoot(t), "scenarios", "business-health"))
	if err != nil {
		t.Fatalf("ParseScenario: %v", err)
	}
	findings := Check{Capturer: fakeCapturer{snapshot: passingSnapshot()}, CaptureProfiles: testProfiles()}.Run(context.Background(), report)
	if got := len(findings); got != 8 {
		t.Fatalf("findings = %d, want 8: %+v", got, findings)
	}
	want := map[string]int{
		spec.CodeClaimFailed:   4,
		spec.CodeClaimUnproven: 4,
	}
	for _, finding := range findings {
		want[finding.Code]--
		if finding.Severity == spec.SeverityError {
			t.Fatalf("draft calibration must be advisory, got error: %+v", finding)
		}
	}
	for code, remaining := range want {
		if remaining != 0 {
			t.Fatalf("code %s remaining count = %d", code, remaining)
		}
	}
}

func TestActivePageReconcilesAgainstAccessibilitySnapshot(t *testing.T) { // [REQ:EXPERIEN-P0-003]
	report := activeReport("primary", spec.Binding{TestID: "primary-action"})
	page := report.Spec.Pages["home"]
	page.Elements = append(page.Elements, spec.Element{ID: "summary", Role: "status"})
	page.Claims = append(page.Claims,
		spec.Claim{ID: "summary-first", Type: "reading-order", Tier: "machine", Elements: []string{"summary", "primary"}, States: []string{"default"}},
	)
	page.Bindings.Elements["summary"] = spec.Binding{TestID: "summary"}
	report.Spec.Pages["home"] = page

	findings := Check{Capturer: fakeCapturer{snapshot: passingSnapshot()}}.Run(context.Background(), report)
	if len(findings) != 0 {
		t.Fatalf("expected active page to reconcile green, got %+v", findings)
	}
}

func TestActivePagePersistsPerClaimEvidence(t *testing.T) { // [REQ:EXPERIEN-P0-003]
	report := activeReport("primary", spec.Binding{TestID: "primary-action"})
	db := testdb.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("EnsureSchemas: %v", err)
	}
	repo := NewSQLiteRepository(db)
	now := time.Date(2026, 7, 5, 16, 30, 0, 0, time.UTC)

	findings := Check{
		Capturer:        fakeCapturer{snapshot: passingSnapshot()},
		Repository:      repo,
		Now:             func() time.Time { return now },
		CaptureProfiles: testProfiles(),
	}.Run(context.Background(), report)
	if len(findings) != 0 {
		t.Fatalf("expected active page to reconcile green, got %+v", findings)
	}
	rows, err := repo.ListEvidence(context.Background(), EvidenceFilter{Scenario: "demo", PageID: "home", ClaimID: "primary-present"})
	if err != nil {
		t.Fatalf("ListEvidence: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("evidence rows = %d, want 1: %+v", len(rows), rows)
	}
	if rows[0].Verdict != "passed" || rows[0].CaptureRef != "scenario=demo,path=/" || rows[0].AXNodeJSON == "{}" {
		t.Fatalf("unexpected evidence row: %+v", rows[0])
	}
	if rows[0].ViewportID != "desktop" || rows[0].ViewportWidth != 1280 || rows[0].ViewportHeight != 720 {
		t.Fatalf("unexpected evidence viewport: %+v", rows[0])
	}
}

func TestActivePagePersistsViewportMatrixEvidence(t *testing.T) { // [REQ:EXPERIEN-P0-003]
	report := activeReport("primary", spec.Binding{TestID: "primary-action"})
	db := testdb.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("EnsureSchemas: %v", err)
	}
	repo := NewSQLiteRepository(db)
	now := time.Date(2026, 7, 5, 16, 30, 0, 0, time.UTC)

	findings := Check{
		Capturer:   fakeCapturer{snapshot: passingSnapshot()},
		Repository: repo,
		Now:        func() time.Time { return now },
	}.Run(context.Background(), report)
	if len(findings) != 0 {
		t.Fatalf("expected active page to reconcile green, got %+v", findings)
	}
	rows, err := repo.ListEvidence(context.Background(), EvidenceFilter{Scenario: "demo", PageID: "home", ClaimID: "primary-present"})
	if err != nil {
		t.Fatalf("ListEvidence: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("evidence rows = %d, want 2: %+v", len(rows), rows)
	}
	got := map[string]bool{}
	for _, row := range rows {
		got[row.ViewportID] = true
		if row.Verdict != "passed" || row.ViewportWidth == 0 || row.ViewportHeight == 0 {
			t.Fatalf("unexpected matrix row: %+v", row)
		}
	}
	if !got["desktop"] || !got["mobile"] {
		t.Fatalf("viewport rows = %+v, want desktop and mobile", got)
	}
}

func TestActivePagePersistsPerTargetCaptureTimings(t *testing.T) {
	report := activeReport("primary", spec.Binding{TestID: "primary-action"})
	db := testdb.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("EnsureSchemas: %v", err)
	}
	repo := NewSQLiteRepository(db)
	snapshot := passingSnapshot()
	snapshot.Timing = CaptureTiming{TotalMilliseconds: 210, NavigationMilliseconds: 60, ReadinessWaitMilliseconds: 150, Strategy: "declared-surface", Outcome: "ready"}
	findings := Check{Capturer: fakeCapturer{snapshot: snapshot}, Repository: repo}.Run(context.Background(), report)
	if len(findings) != 0 {
		t.Fatalf("expected green reconciliation, got %+v", findings)
	}
	timings, err := repo.(CaptureTimingRepository).ListCaptureTimings(context.Background(), CaptureTimingFilter{Scenario: "demo", PageID: "home"})
	if err != nil {
		t.Fatalf("ListCaptureTimings: %v", err)
	}
	if len(timings) != 2 || timings[0].TotalMilliseconds != 210 || timings[0].Strategy != "declared-surface" || timings[0].Outcome != "ready" {
		t.Fatalf("unexpected capture timing history: %+v", timings)
	}
}

func TestActivePageBoundsCaptureConcurrencyToConfiguredWorkers(t *testing.T) {
	report := activeReport("primary", spec.Binding{TestID: "primary-action"})
	capturer := &concurrentCapturer{snapshot: passingSnapshot()}
	findings := Check{Capturer: capturer, CaptureConcurrency: 2}.Run(context.Background(), report)
	if len(findings) != 0 {
		t.Fatalf("expected green reconciliation, got %+v", findings)
	}
	capturer.mu.Lock()
	max := capturer.max
	capturer.mu.Unlock()
	if max != 2 {
		t.Fatalf("maximum simultaneous captures = %d, want 2", max)
	}
}

func TestActiveComponentBuildsHarnessTargetsAndPersistsEvidence(t *testing.T) { // [REQ:EXPERIEN-P0-003]
	report := activeComponentReport("action", spec.Binding{Selector: "button"})
	db := testdb.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("EnsureSchemas: %v", err)
	}
	repo := NewSQLiteRepository(db)
	var targets []CaptureTarget
	now := time.Date(2026, 7, 10, 14, 15, 0, 0, time.UTC)

	findings := Check{
		Capturer: fakeCapturer{
			snapshot: componentButtonSnapshot(24, 80, 140, 48),
			targets:  &targets,
		},
		Repository:      repo,
		Now:             func() time.Time { return now },
		CaptureProfiles: testProfiles(),
	}.Run(context.Background(), report)
	if len(findings) != 0 {
		t.Fatalf("expected active component to reconcile green, got %+v", findings)
	}
	if len(targets) != 1 {
		t.Fatalf("targets = %+v, want one component target", targets)
	}
	target := targets[0]
	if target.DocumentKind != "component" || target.Scenario != "react-component-library" || target.PageID != "button" || target.ComponentID != "button" {
		t.Fatalf("target identity = %+v, want RCL component button", target)
	}
	if target.StateID != "primary" || target.ExampleName != "primary" {
		t.Fatalf("target state/example = %+v, want primary example", target)
	}
	if !strings.HasPrefix(target.Route, "/preview/react-component-library:Button/harness.html?") ||
		!strings.Contains(target.Route, "example=primary") ||
		!strings.Contains(target.Route, "version=1.2.0") {
		t.Fatalf("target route = %q, want Button harness with version/example", target.Route)
	}
	rows, err := repo.ListEvidence(context.Background(), EvidenceFilter{Scenario: "react-component-library", PageID: "button", ClaimID: "action-present"})
	if err != nil {
		t.Fatalf("ListEvidence: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("evidence rows = %d, want 1: %+v", len(rows), rows)
	}
	if rows[0].Verdict != "passed" || rows[0].Route != target.Route || rows[0].CaptureRef != "scenario=react-component-library,path="+target.Route {
		t.Fatalf("unexpected component evidence row: %+v", rows[0])
	}
}

func TestActiveComponentFailuresAreAdvisory(t *testing.T) {
	report := activeComponentReport("action", spec.Binding{Selector: "button"})
	component := report.Spec.Components["button"]
	component.Claims = []spec.Claim{{
		ID:        "action-visible",
		Type:      "visible-without-scroll",
		Statement: "The component action remains in the harness viewport.",
		Tier:      "machine",
		Elements:  []string{"action"},
		States:    []string{"primary"},
	}}
	report.Spec.Components["button"] = component

	findings := Check{
		Capturer:        fakeCapturer{snapshot: componentButtonSnapshot(24, 900, 140, 48)},
		CaptureProfiles: testProfiles(),
	}.Run(context.Background(), report)
	if len(findings) != 1 || findings[0].Code != spec.CodeClaimFailed {
		t.Fatalf("expected one failed component claim, got %+v", findings)
	}
	if findings[0].Severity != spec.SeverityWarning {
		t.Fatalf("component finding severity = %s, want advisory warning", findings[0].Severity)
	}
}

func TestViewportScopedClaimCapturesMatchingProfileOnly(t *testing.T) { // [REQ:EXPERIEN-P0-003]
	report := activeReport("primary", spec.Binding{TestID: "primary-action"})
	page := report.Spec.Pages["home"]
	page.Claims[0].Viewports = []string{"mobile"}
	page.FloorOptOuts = allFloorOptOuts()
	report.Spec.Pages["home"] = page
	db := testdb.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("EnsureSchemas: %v", err)
	}
	repo := NewSQLiteRepository(db)
	var targets []CaptureTarget

	findings := Check{
		Capturer:   fakeCapturer{snapshot: passingSnapshot(), targets: &targets},
		Repository: repo,
		Now:        func() time.Time { return time.Date(2026, 7, 5, 16, 30, 0, 0, time.UTC) },
	}.Run(context.Background(), report)
	if len(findings) != 0 {
		t.Fatalf("expected viewport-scoped claim to reconcile green, got %+v", findings)
	}
	if len(targets) != 1 || targets[0].ViewportID != "mobile" {
		t.Fatalf("capture targets = %+v, want one mobile target", targets)
	}
	rows, err := repo.ListEvidence(context.Background(), EvidenceFilter{Scenario: "demo", PageID: "home", ClaimID: "primary-present"})
	if err != nil {
		t.Fatalf("ListEvidence: %v", err)
	}
	if len(rows) != 1 || rows[0].ViewportID != "mobile" {
		t.Fatalf("evidence rows = %+v, want one mobile row", rows)
	}
}

func TestViewportScopedClaimUsesProfileAlias(t *testing.T) { // [REQ:EXPERIEN-P0-003]
	report := activeReport("primary", spec.Binding{TestID: "primary-action"})
	page := report.Spec.Pages["home"]
	page.Claims[0].Viewports = []string{"wide"}
	page.FloorOptOuts = allFloorOptOuts()
	report.Spec.Pages["home"] = page
	var targets []CaptureTarget

	findings := Check{
		Capturer: fakeCapturer{snapshot: passingSnapshot(), targets: &targets},
	}.Run(context.Background(), report)
	if len(findings) != 0 {
		t.Fatalf("expected aliased viewport-scoped claim to reconcile green, got %+v", findings)
	}
	if len(targets) != 1 || targets[0].ViewportID != "desktop" {
		t.Fatalf("capture targets = %+v, want one desktop target for wide alias", targets)
	}
}

func TestViewportScopedClaimOutsideMatrixIsUnverifiable(t *testing.T) {
	report := activeReport("primary", spec.Binding{TestID: "primary-action"})
	page := report.Spec.Pages["home"]
	page.Claims[0].Viewports = []string{"tablet"}
	report.Spec.Pages["home"] = page

	findings := Check{Capturer: fakeCapturer{snapshot: passingSnapshot()}}.Run(context.Background(), report)
	if len(findings) != 1 || findings[0].Code != spec.CodeClaimUnverifiable {
		t.Fatalf("expected one unverifiable viewport finding, got %+v", findings)
	}
	if !strings.Contains(findings[0].Message, "tablet") {
		t.Fatalf("message = %q, want missing viewport", findings[0].Message)
	}
}

func TestActivePageFailsUnresolvedBinding(t *testing.T) {
	report := activeReport("primary", spec.Binding{TestID: "primary-action"})
	page := report.Spec.Pages["home"]
	page.Elements = append(page.Elements, spec.Element{ID: "missing", Role: "button"})
	page.Claims = append(page.Claims, spec.Claim{
		ID:       "missing-present",
		Type:     "element-present",
		Tier:     "machine",
		Elements: []string{"missing"},
		States:   []string{"default"},
	})
	page.Bindings.Elements["missing"] = spec.Binding{TestID: "not-rendered"}
	report.Spec.Pages["home"] = page

	findings := Check{Capturer: fakeCapturer{snapshot: passingSnapshot()}, CaptureProfiles: testProfiles()}.Run(context.Background(), report)
	if !hasCode(findings, spec.CodeBindingUnresolved) || !hasCode(findings, spec.CodeClaimFailed) {
		t.Fatalf("expected binding and claim failures, got %+v", findings)
	}
}

func TestVisibleWithoutScrollChecksBoundGeometry(t *testing.T) {
	report := activeReport("primary", spec.Binding{TestID: "primary-action"})
	page := report.Spec.Pages["home"]
	page.Claims = []spec.Claim{{
		ID:        "primary-above-fold",
		Type:      "visible-without-scroll",
		Statement: "The primary action is visible without scrolling.",
		Tier:      "machine",
		Elements:  []string{"primary"},
		States:    []string{"default"},
		Viewports: []string{"desktop"},
	}}
	page.FloorOptOuts = allFloorOptOuts()
	report.Spec.Pages["home"] = page
	snapshot := passingSnapshot()
	snapshot.Root.Children[1].Bounds = &Bounds{X: 24, Y: 760, Width: 120, Height: 48}

	findings := Check{Capturer: fakeCapturer{snapshot: snapshot}, CaptureProfiles: testProfiles()}.Run(context.Background(), report)
	if !hasCode(findings, spec.CodeClaimFailed) {
		t.Fatalf("expected visible-without-scroll failure, got %+v", findings)
	}

	snapshot.Root.Children[1].Bounds = &Bounds{X: 24, Y: 80, Width: 120, Height: 48}
	findings = Check{Capturer: fakeCapturer{snapshot: snapshot}, CaptureProfiles: testProfiles()}.Run(context.Background(), report)
	if len(findings) != 0 {
		t.Fatalf("expected visible-without-scroll pass, got %+v", findings)
	}
}

func TestStructuredComponentClaimsEvaluateGeometryAndAppearance(t *testing.T) {
	page := spec.PageDocument{
		Page: spec.PageIdentity{ID: "control", Routes: []string{"/"}},
		Elements: []spec.Element{
			{ID: "icon", Role: "x-icon"},
			{ID: "label", Role: "x-label"},
			{ID: "control", Role: "button"},
			{ID: "peer", Role: "button"},
		},
		Bindings: spec.Bindings{Elements: map[string]spec.Binding{
			"icon": {TestID: "icon"}, "label": {TestID: "label"}, "control": {TestID: "control"}, "peer": {TestID: "peer"},
		}},
	}
	target := CaptureTarget{StateID: "hover", ViewportID: "desktop", ViewportWidth: 1280, ViewportHeight: 720}

	spacing := spec.Claim{ID: "spacing", Type: "spacing", Tier: "machine", Elements: []string{"icon", "label"}, Params: map[string]any{"minSeparation": 4, "axis": "inline"}}
	nodes := []*AXNode{
		{Role: "x-icon", DOM: DOMNode{TestID: "icon"}, Bounds: &Bounds{X: 10, Y: 10, Width: 16, Height: 16}},
		{Role: "x-label", DOM: DOMNode{TestID: "label"}, Bounds: &Bounds{X: 30, Y: 10, Width: 50, Height: 16}},
	}
	if result := claimEvaluator("spacing")(page, spacing, target, nodes); !result.Pass {
		t.Fatalf("spacing should pass: %+v", result)
	}
	nodes[1].Bounds.X = 25
	if result := claimEvaluator("spacing")(page, spacing, target, nodes); result.Pass || !strings.Contains(result.Failure, "below") {
		t.Fatalf("spacing should fail with a named gap: %+v", result)
	}

	contrast := spec.Claim{ID: "contrast", Type: "state-contrast", Tier: "machine", Elements: []string{"control"}, Params: map[string]any{"state": "hover", "background": "#020617", "minContrastRatio": 3}}
	contrastNode := &AXNode{Role: "button", DOM: DOMNode{TestID: "control"}, Appearance: &Appearance{States: map[string]AppearanceState{"hover": {Foreground: "#111827", Background: "#020617"}}}}
	if result := claimEvaluator("state-contrast")(page, contrast, target, []*AXNode{contrastNode}); result.Pass || !strings.Contains(result.Failure, "contrast") {
		t.Fatalf("low hover contrast should fail: %+v", result)
	}
	contrastNode.Appearance.States["hover"] = AppearanceState{Foreground: "#ffffff", Background: "#020617"}
	if result := claimEvaluator("state-contrast")(page, contrast, target, []*AXNode{contrastNode}); !result.Pass {
		t.Fatalf("readable hover contrast should pass: %+v", result)
	}

	parity := spec.Claim{ID: "parity", Type: "size-parity", Tier: "machine", Elements: []string{"control", "peer"}, Params: map[string]any{"tolerance": 1}}
	parityNodes := []*AXNode{
		{Role: "button", DOM: DOMNode{TestID: "control"}, Bounds: &Bounds{Width: 40, Height: 40}},
		{Role: "button", DOM: DOMNode{TestID: "peer"}, Bounds: &Bounds{Width: 40, Height: 44}},
	}
	if result := claimEvaluator("size-parity")(page, parity, target, parityNodes); result.Pass {
		t.Fatalf("size parity should fail outside tolerance: %+v", result)
	}
	parityNodes[1].Bounds.Height = 41
	if result := claimEvaluator("size-parity")(page, parity, target, parityNodes); !result.Pass {
		t.Fatalf("size parity should pass within tolerance: %+v", result)
	}
}

func TestComponentSourceClaimsDetectRemovedInteractionContracts(t *testing.T) {
	good := `const densityClasses = { comfortable: "gap-2" }; const variants = "bg-app-primary hover:brightness-95"; const sizeClasses = "min-h-10 min-w-10";`
	badSpacing := strings.Replace(good, "gap-2", "gap-0", 1)
	if sourceClaimPasses(spec.Claim{Type: "spacing"}, badSpacing) {
		t.Fatal("spacing source claim passed after positive gap was removed")
	}
	if !sourceClaimPasses(spec.Claim{Type: "spacing"}, good) {
		t.Fatal("spacing source claim rejected a positive gap")
	}
	badHover := strings.Replace(good, "hover:brightness-95", "brightness-95", 1)
	if sourceClaimPasses(spec.Claim{Type: "state-contrast"}, badHover) {
		t.Fatal("state-contrast source claim passed after hover treatment was removed")
	}
	if !sourceClaimPasses(spec.Claim{Type: "state-contrast"}, good) {
		t.Fatal("state-contrast source claim rejected a declared hover treatment")
	}
	if !sourceClaimPasses(spec.Claim{Type: "size-parity"}, good) {
		t.Fatal("size-parity source claim rejected the shared size scale")
	}
}

func TestCatalogComponentSourceClaimsPassForReleasedControls(t *testing.T) {
	report, err := spec.ParseScenario(filepath.Join(repoRoot(t), "scenarios", "react-component-library"))
	if err != nil {
		t.Fatalf("ParseScenario: %v", err)
	}
	for _, id := range []string{"button", "control-base", "voice-input-button"} {
		component := report.Spec.Components[id]
		findings := componentSourceFindings(report, "experience/components/"+id+".json", component)
		for _, finding := range findings {
			if finding.Code == spec.CodeClaimFailed {
				t.Fatalf("released component %q has a failed source claim: %+v", id, finding)
			}
		}
	}
}

func TestAffordancePresentChecksExpectedControls(t *testing.T) {
	report := activeReport("debt-table", spec.Binding{TestID: "debt-table"})
	page := report.Spec.Pages["home"]
	page.Elements = []spec.Element{{ID: "debt-table", Role: "table"}}
	page.Claims = []spec.Claim{{
		ID:        "table-affordances",
		Type:      "affordance-present",
		Statement: "The debt table can be searched, sorted, and filtered.",
		Tier:      "machine",
		Elements:  []string{"debt-table"},
		States:    []string{"default"},
		Params: map[string]any{
			"targetRole":  "table",
			"affordances": []string{"search", "sort", "filter"},
		},
	}}
	page.Bindings.Elements = map[string]spec.Binding{"debt-table": {TestID: "debt-table"}}
	page.FloorOptOuts = allFloorOptOuts()
	report.Spec.Pages["home"] = page

	snapshot := Snapshot{
		Contract: snapshotContract,
		Root: AXNode{Role: "WebArea", Children: []AXNode{
			{Role: "table", Name: "Debt table", Bounds: &Bounds{X: 0, Y: 100, Width: 500, Height: 300}, DOM: DOMNode{TestID: "debt-table"}},
			{Role: "textbox", Name: "Search debt", Bounds: &Bounds{X: 0, Y: 40, Width: 180, Height: 40}, DOM: DOMNode{TestID: "debt-search"}},
			{Role: "button", Name: "Sort by severity", Bounds: &Bounds{X: 190, Y: 40, Width: 120, Height: 40}, DOM: DOMNode{TestID: "debt-sort"}},
			{Role: "combobox", Name: "Filter status", Bounds: &Bounds{X: 320, Y: 40, Width: 140, Height: 40}, DOM: DOMNode{TestID: "debt-filter"}},
		}},
	}
	findings := Check{Capturer: fakeCapturer{snapshot: snapshot}, CaptureProfiles: testProfiles()}.Run(context.Background(), report)
	if len(findings) != 0 {
		t.Fatalf("expected affordance-present claim to pass, got %+v", findings)
	}

	snapshot.Root.Children = snapshot.Root.Children[:1]
	findings = Check{Capturer: fakeCapturer{snapshot: snapshot}, CaptureProfiles: testProfiles()}.Run(context.Background(), report)
	if !hasCode(findings, spec.CodeAffordanceMissing) {
		t.Fatalf("expected missing affordance finding, got %+v", findings)
	}
}

func TestAccessibleNameClaimReconcilesDeclaredIntent(t *testing.T) {
	report := activeReport("primary", spec.Binding{TestID: "primary-action"})
	page := report.Spec.Pages["home"]
	page.Elements = []spec.Element{{ID: "primary", Role: "button", Name: "Create report"}}
	page.Claims = []spec.Claim{{ID: "primary-name", Type: "accessible-name", Statement: "The primary action has the declared accessible name.", Tier: "machine", Elements: []string{"primary"}, States: []string{"default"}}}
	page.FloorOptOuts = allFloorOptOuts()
	report.Spec.Pages["home"] = page
	snapshot := passingSnapshot()
	snapshot.Root.Children[1].Name = "Create report"
	findings := (Check{Capturer: fakeCapturer{snapshot: snapshot}, CaptureProfiles: testProfiles()}).Run(context.Background(), report)
	if len(findings) != 0 {
		t.Fatalf("expected accessible name claim to pass, got %+v", findings)
	}
	snapshot.Root.Children[1].Name = "Unnamed action"
	findings = (Check{Capturer: fakeCapturer{snapshot: snapshot}, CaptureProfiles: testProfiles()}).Run(context.Background(), report)
	if !hasCode(findings, spec.CodeClaimFailed) {
		t.Fatalf("expected accessible name claim failure, got %+v", findings)
	}
}

func TestBaselineFloorClaimsFailFromGeometry(t *testing.T) {
	report := activeReport("primary", spec.Binding{TestID: "primary-action"})
	snapshot := passingSnapshot()
	snapshot.Root.Bounds = &Bounds{Width: 430, Height: 700}
	snapshot.Root.Children = append(snapshot.Root.Children, AXNode{
		Role:   "navigation",
		Name:   "Primary navigation",
		Bounds: &Bounds{X: 0, Y: 812, Width: 390, Height: 64},
		DOM:    DOMNode{Tag: "nav"},
		Children: []AXNode{{
			Role:   "button",
			Name:   "日本\n語",
			Bounds: &Bounds{X: 24, Y: 820, Width: 40, Height: 64},
			DOM:    DOMNode{TestID: "layout-bottom-nav-link-language"},
		}},
	})

	findings := Check{
		Capturer:        fakeCapturer{snapshot: snapshot},
		CaptureProfiles: []CaptureProfile{{ID: "mobile", Width: 390, Height: 844}},
	}.Run(context.Background(), report)
	for _, code := range []string{
		spec.CodeFloorNoDocOverflow,
		spec.CodeFloorViewportFill,
		spec.CodeFloorChromePinned,
		spec.CodeFloorSafeArea,
		spec.CodeFloorSingleLine,
		spec.CodeFloorTapTargetSize,
	} {
		if !hasCode(findings, code) {
			t.Fatalf("missing %s in findings: %+v", code, findings)
		}
	}
}

func TestSafeAreaFloorRecognizesScenarioMobileNavControls(t *testing.T) {
	report := activeReport("primary", spec.Binding{TestID: "primary-action"})
	snapshot := passingSnapshot()
	snapshot.Root.Children = append(snapshot.Root.Children, AXNode{
		Role:   "navigation",
		Name:   "Mobile navigation",
		Bounds: &Bounds{X: 0, Y: 780, Width: 390, Height: 64},
		DOM:    DOMNode{Tag: "nav", TestID: "mobile-nav"},
		Children: []AXNode{{
			Role:   "button",
			Name:   "Changes",
			Bounds: &Bounds{X: 0, Y: 800, Width: 78, Height: 44},
			DOM:    DOMNode{TestID: "mobile-nav-changes"},
		}},
	})

	findings := Check{
		Capturer:        fakeCapturer{snapshot: snapshot},
		CaptureProfiles: []CaptureProfile{{ID: "mobile", Width: 390, Height: 844}},
	}.Run(context.Background(), report)
	if !hasCode(findings, spec.CodeFloorSafeArea) {
		t.Fatalf("expected safe-area floor to catch scenario mobile nav controls, got %+v", findings)
	}
}

func TestBaselineFloorOptOutSuppressesOnlyNamedFloor(t *testing.T) {
	report := activeReport("primary", spec.Binding{TestID: "primary-action"})
	page := report.Spec.Pages["home"]
	page.FloorOptOuts = []spec.FloorOptOut{{
		Floor:  "viewport-fill",
		Reason: "This fixture intentionally models a short viewport surface.",
	}}
	report.Spec.Pages["home"] = page
	snapshot := passingSnapshot()
	snapshot.Root.Bounds = &Bounds{Width: 430, Height: 600}

	findings := Check{
		Capturer:        fakeCapturer{snapshot: snapshot},
		CaptureProfiles: []CaptureProfile{{ID: "mobile", Width: 390, Height: 844}},
	}.Run(context.Background(), report)
	if hasCode(findings, spec.CodeFloorViewportFill) {
		t.Fatalf("viewport-fill opt-out was ignored: %+v", findings)
	}
	if !hasCode(findings, spec.CodeFloorNoDocOverflow) {
		t.Fatalf("opt-out suppressed unrelated floors: %+v", findings)
	}
}

func TestNonDefaultStateClaimReportsSingleCaptureLimitation(t *testing.T) {
	report := activeReport("primary", spec.Binding{TestID: "primary-action"})
	page := report.Spec.Pages["home"]
	page.Claims = []spec.Claim{{
		ID:       "stale-distinct",
		Type:     "state-distinct",
		Tier:     "machine",
		Elements: []string{"primary"},
		States:   []string{"default", "stale"},
	}}
	report.Spec.Pages["home"] = page

	findings := Check{Capturer: fakeCapturer{snapshot: passingSnapshot()}, CaptureProfiles: testProfiles()}.Run(context.Background(), report)
	if len(findings) != 1 {
		t.Fatalf("findings = %d, want 1: %+v", len(findings), findings)
	}
	if findings[0].Code != spec.CodeClaimUnverifiable || findings[0].Severity != spec.SeverityWarning {
		t.Fatalf("finding = %+v, want claim_unverifiable warning", findings[0])
	}
	if !strings.Contains(findings[0].Message, "was not captured for distinct-state comparison") {
		t.Fatalf("message = %q", findings[0].Message)
	}
}

func TestNonDefaultStateSetupCapturesStateRoute(t *testing.T) { // [REQ:EXPERIEN-P0-003]
	report := activeReport("primary", spec.Binding{TestID: "primary-action"})
	page := report.Spec.Pages["home"]
	page.States = []spec.State{
		{ID: "default"},
		{ID: "empty", Setup: spec.Setup{
			Route:    "/scenario/demo",
			Query:    map[string]string{"state": "empty"},
			Hash:     "details",
			SettleMs: 1200,
		}},
	}
	page.Claims = []spec.Claim{{
		ID:       "empty-guides",
		Type:     "state-covered",
		Tier:     "machine",
		Elements: []string{"primary"},
		States:   []string{"empty"},
	}}
	report.Spec.Pages["home"] = page
	var targets []CaptureTarget
	db := testdb.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("EnsureSchemas: %v", err)
	}
	repo := NewSQLiteRepository(db)

	findings := Check{
		Capturer:   fakeCapturer{snapshot: passingSnapshot(), targets: &targets},
		Repository: repo,
		Now: func() time.Time {
			return time.Date(2026, 7, 5, 16, 30, 0, 0, time.UTC)
		},
		CaptureProfiles: testProfiles(),
	}.Run(context.Background(), report)
	if len(findings) != 0 {
		t.Fatalf("expected non-default state setup to reconcile green, got %+v", findings)
	}
	var target CaptureTarget
	for _, candidate := range targets {
		if candidate.StateID == "empty" {
			target = candidate
			break
		}
	}
	if target.StateID != "empty" || target.Route != "/scenario/demo?state=empty#details" || target.SettleMs != 1200 {
		t.Fatalf("target = %+v, want empty state setup route", target)
	}
	rows, err := repo.ListEvidence(context.Background(), EvidenceFilter{Scenario: "demo", PageID: "home", ClaimID: "empty-guides"})
	if err != nil {
		t.Fatalf("ListEvidence: %v", err)
	}
	if len(rows) != 1 || rows[0].StateID != "empty" || rows[0].Verdict != "passed" {
		t.Fatalf("evidence rows = %+v, want passed empty-state evidence", rows)
	}
}

func TestStateDistinctComparesCapturedStateFingerprints(t *testing.T) {
	report := activeReport("primary", spec.Binding{TestID: "primary-action"})
	page := report.Spec.Pages["home"]
	page.States = []spec.State{
		{ID: "default"},
		{ID: "stale", Setup: spec.Setup{Query: map[string]string{"state": "stale"}}},
	}
	page.Claims = []spec.Claim{{
		ID:       "stale-distinct",
		Type:     "state-distinct",
		Tier:     "machine",
		Elements: []string{"primary"},
		States:   []string{"default", "stale"},
	}}
	report.Spec.Pages["home"] = page

	findings := Check{Capturer: fakeCapturer{snapshot: passingSnapshot()}, CaptureProfiles: testProfiles()}.Run(context.Background(), report)
	if !hasCode(findings, spec.CodeClaimFailed) {
		t.Fatalf("expected identical state fingerprints to fail distinct claim, got %+v", findings)
	}

	stale := passingSnapshot()
	stale.Root.Children[1].Name = "Stale primary action"
	findings = Check{Capturer: fakeCapturer{
		snapshot:         passingSnapshot(),
		snapshotsByState: map[string]Snapshot{"stale": stale},
	}, CaptureProfiles: testProfiles()}.Run(context.Background(), report)
	if len(findings) != 0 {
		t.Fatalf("expected distinct state fingerprints to pass, got %+v", findings)
	}
}

func TestNonDefaultStateWithoutSetupReportsLimitation(t *testing.T) {
	report := activeReport("primary", spec.Binding{TestID: "primary-action"})
	page := report.Spec.Pages["home"]
	page.Claims = []spec.Claim{{
		ID:       "empty-covered",
		Type:     "state-covered",
		Tier:     "machine",
		Elements: []string{"primary"},
		States:   []string{"empty"},
	}}
	report.Spec.Pages["home"] = page

	findings := Check{Capturer: fakeCapturer{snapshot: passingSnapshot()}, CaptureProfiles: testProfiles()}.Run(context.Background(), report)
	if len(findings) != 1 || findings[0].Code != spec.CodeClaimUnverifiable {
		t.Fatalf("expected one missing setup limitation, got %+v", findings)
	}
	if !strings.Contains(findings[0].Message, "without deterministic setup") {
		t.Fatalf("message = %q", findings[0].Message)
	}
}

func TestActivePageWithNoJoinedBindingsSkipsAsUnavailable(t *testing.T) {
	report := activeReport("missing", spec.Binding{TestID: "not-rendered"})
	findings := Check{Capturer: fakeCapturer{snapshot: passingSnapshot()}, CaptureProfiles: testProfiles()}.Run(context.Background(), report)
	if len(findings) != 1 || findings[0].Code != spec.CodeCaptureUnavailable || findings[0].Severity != spec.SeverityInfo {
		t.Fatalf("expected capture unavailable info finding, got %+v", findings)
	}
}

func TestCaptureUnavailableIsSkippedInfo(t *testing.T) {
	report := activeReport("primary", spec.Binding{TestID: "primary-action"})
	findings := Check{Capturer: fakeCapturer{err: ErrCaptureUnavailable}, CaptureProfiles: testProfiles()}.Run(context.Background(), report)
	if len(findings) != 1 || findings[0].Code != spec.CodeCaptureUnavailable || findings[0].Severity != spec.SeverityInfo {
		t.Fatalf("expected capture unavailable info finding, got %+v", findings)
	}
}

func TestBASCapturerPreservesScenarioTargetForDeclaredReadiness(t *testing.T) {
	screenshotPath := filepath.Join(t.TempDir(), "capture.png")
	if err := os.WriteFile(screenshotPath, []byte{0x89, 0x50, 0x4e, 0x47}, 0o644); err != nil {
		t.Fatalf("WriteFile screenshot: %v", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/browser_automation_studio.v1.capture.CaptureService/Capture", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			URL                 string `json:"url"`
			InlineAccessibility bool   `json:"inlineAccessibility"`
			Dimensions          struct {
				Width  int `json:"width"`
				Height int `json:"height"`
			} `json:"dimensions"`
			InteractionFlowJSON string `json:"interaction_flow_json"`
			WaitFor             struct {
				TimeoutMs int `json:"timeout_ms"`
			} `json:"wait_for"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Decode request: %v", err)
		}
		if got := req.URL; got != "scenario=web-console,path=/" {
			t.Fatalf("captured url = %q, want scenario target", got)
		}
		if !req.InlineAccessibility {
			t.Fatal("inlineAccessibility = false, want true")
		}
		if req.Dimensions.Width != 390 || req.Dimensions.Height != 844 {
			t.Fatalf("dimensions = %dx%d, want 390x844", req.Dimensions.Width, req.Dimensions.Height)
		}
		if got := req.WaitFor.TimeoutMs; got != 0 {
			t.Fatalf("waitFor.timeoutMs = %d, want no explicit wait", got)
		}
		if req.InteractionFlowJSON != "" {
			t.Fatalf("interactionFlowJson = %q, want semantic readiness", req.InteractionFlowJSON)
		}
		snapshot, err := json.Marshal(passingSnapshot())
		if err != nil {
			t.Fatalf("Marshal snapshot: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			// Connect's JSON codec emits lowerCamelCase response fields. Keep this
			// regression contract aligned with Browser Automation Studio's live
			// CaptureService rather than only exercising the legacy snake_case form.
			"accessibilityJson": string(snapshot),
			"readiness": map[string]any{
				"durationMs":              210,
				"navigationDurationMs":    60,
				"readinessWaitDurationMs": 150,
				"selectedStrategy":        "declared-surface",
				"outcome":                 "ready",
			},
			"artifacts": []map[string]any{{
				"type":       "CAPTURE_TYPE_SCREENSHOT",
				"path":       screenshotPath,
				"size_bytes": 4,
				"metadata":   map[string]string{"filename": "capture.png"},
			}},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	capturer := BASCapturer{
		Resolve: func(context.Context) (string, error) {
			return server.URL, nil
		},
		ResolveTarget: func(_ context.Context, target CaptureTarget) (string, error) {
			return "http://localhost:21233" + target.Route, nil
		},
		HTTPClient: server.Client(),
	}
	snapshot, err := capturer.CaptureAccessibility(context.Background(), CaptureTarget{
		Scenario:       "web-console",
		Route:          "/",
		PageID:         "workspace",
		ViewportID:     "mobile",
		ViewportWidth:  390,
		ViewportHeight: 844,
		SettleMs:       defaultSettleMs,
	})
	if err != nil {
		t.Fatalf("CaptureAccessibility: %v", err)
	}
	if !strings.HasPrefix(snapshot.ScreenshotRef, "data:image/png;base64,") {
		t.Fatalf("ScreenshotRef = %q, want png data URL", snapshot.ScreenshotRef)
	}
	if snapshot.Timing.TotalMilliseconds != 210 || snapshot.Timing.ReadinessWaitMilliseconds != 150 || snapshot.Timing.Strategy != "declared-surface" {
		t.Fatalf("Timing = %+v, want BAS readiness diagnostics", snapshot.Timing)
	}
}

func testProfiles() []CaptureProfile {
	return []CaptureProfile{{ID: "desktop", Width: 1280, Height: 720}}
}

func activeReport(elementID string, binding spec.Binding) spec.Report {
	return spec.Report{
		Scenario: "demo",
		Spec: &spec.ScenarioSpec{
			Index: spec.IndexDocument{Pages: []spec.DocumentRef{{ID: "home", Status: "active"}}},
			Pages: map[string]spec.PageDocument{"home": {
				Page:     spec.PageIdentity{ID: "home", Routes: []string{"/"}},
				Elements: []spec.Element{{ID: elementID, Role: "button"}},
				Claims: []spec.Claim{{
					ID:       elementID + "-present",
					Type:     "element-present",
					Tier:     "machine",
					Elements: []string{elementID},
					States:   []string{"default"},
				}},
				Bindings: spec.Bindings{Elements: map[string]spec.Binding{elementID: binding}},
			}},
		},
	}
}

func activeComponentReport(elementID string, binding spec.Binding) spec.Report {
	return spec.Report{
		Scenario: "react-component-library",
		Spec: &spec.ScenarioSpec{
			Index: spec.IndexDocument{Components: []spec.DocumentRef{{ID: "button", Status: "active"}}},
			Components: map[string]spec.ComponentDocument{"button": {
				Component: spec.ComponentIdentity{
					ID:          "button",
					Title:       "Button",
					Purpose:     "Provide a reusable action primitive.",
					ExamplesRef: "../../library/components/Button/versions/1.2.0/examples.json",
				},
				States: []spec.ComponentState{{ID: "primary", Example: "primary"}},
				Elements: []spec.Element{{
					ID:   elementID,
					Role: "button",
				}},
				Claims: []spec.Claim{{
					ID:       elementID + "-present",
					Type:     "element-present",
					Tier:     "machine",
					Elements: []string{elementID},
					States:   []string{"primary"},
				}},
				Bindings: spec.Bindings{Elements: map[string]spec.Binding{elementID: binding}},
				FloorOptOuts: []spec.FloorOptOut{{
					Floor:  "no-document-horizontal-overflow",
					Reason: "This fixture isolates authored component claim behavior.",
				}, {
					Floor:  "tap-target-size",
					Reason: "This fixture isolates authored component claim behavior.",
				}},
			}},
		},
	}
}

func passingSnapshot() Snapshot {
	return Snapshot{
		Contract: snapshotContract,
		Root: AXNode{Role: "WebArea", Children: []AXNode{
			{Role: "status", Bounds: &Bounds{X: 0, Y: 24, Width: 320, Height: 24}, DOM: DOMNode{TestID: "summary"}},
			{Role: "button", States: []string{"focusable"}, Bounds: &Bounds{X: 24, Y: 80, Width: 120, Height: 48}, DOM: DOMNode{TestID: "primary-action"}},
		}},
	}
}

func componentButtonSnapshot(x, y, width, height float64) Snapshot {
	return Snapshot{
		Contract: snapshotContract,
		Root: AXNode{Role: "WebArea", Bounds: &Bounds{Width: 1280, Height: 720}, Children: []AXNode{{
			Role:   "button",
			Name:   "Save changes",
			States: []string{"focusable"},
			Bounds: &Bounds{X: x, Y: y, Width: width, Height: height},
			DOM:    DOMNode{Tag: "button"},
		}}},
	}
}

func hasCode(findings []spec.Finding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func allFloorOptOuts() []spec.FloorOptOut {
	return []spec.FloorOptOut{
		{Floor: "no-document-horizontal-overflow", Reason: "This fixture isolates authored viewport-scoped claim behavior."},
		{Floor: "viewport-fill", Reason: "This fixture isolates authored viewport-scoped claim behavior."},
		{Floor: "chrome-pinned", Reason: "This fixture isolates authored viewport-scoped claim behavior."},
		{Floor: "safe-area-tap-targets", Reason: "This fixture isolates authored viewport-scoped claim behavior."},
		{Floor: "single-line-chrome", Reason: "This fixture isolates authored viewport-scoped claim behavior."},
		{Floor: "tap-target-size", Reason: "This fixture isolates authored viewport-scoped claim behavior."},
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "VISION.md")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repo root not found")
		}
		dir = parent
	}
}
