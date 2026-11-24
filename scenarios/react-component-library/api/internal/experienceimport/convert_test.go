package experienceimport

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
	"react-component-library/internal/workspaces"
)

type atomicImportFake struct {
	receipt   workspaces.ConversionReceipt
	workspace *workspacev1.Workspace
	creates   int
}

func (f *atomicImportFake) GetConversionReceipt(_ context.Context, _, _ string) (workspaces.ConversionReceipt, error) {
	if f.receipt.ID == "" {
		return workspaces.ConversionReceipt{}, workspaces.ErrNotFound
	}
	return f.receipt, nil
}

func (f *atomicImportFake) CreateWorkspaceWithConversionReceipt(_ context.Context, workspace *workspacev1.Workspace, input workspaces.ConversionReceiptInput) (*workspacev1.Workspace, workspaces.ConversionReceipt, error) {
	f.creates++
	if f.receipt.ID != "" {
		if f.receipt.SourceHash == input.SourceHash && f.receipt.TargetHash == input.TargetHash && f.receipt.ReceiptJSON == input.ReceiptJSON {
			return f.workspace, f.receipt, nil
		}
		return nil, workspaces.ConversionReceipt{}, &workspaces.ConversionReceiptConflictError{Existing: f.receipt}
	}
	f.workspace = workspace
	f.receipt = workspaces.ConversionReceipt{ID: "ir_saved", Scenario: input.Scenario, Page: input.Page, WorkspaceID: input.WorkspaceID, SourceHash: input.SourceHash, TargetHash: input.TargetHash, ReceiptJSON: input.ReceiptJSON}
	return f.workspace, f.receipt, nil
}

func TestAdoptPageIsAtomicRepeatSafeAndRejectsChangedSource(t *testing.T) {
	sourcePath := filepath.Join(experienceImportTestdata(t), "switchboard/experience/pages/conversations.json")
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	store := &atomicImportFake{}
	first, firstReceipt, already, err := AdoptPage(context.Background(), store, "switchboard", "conversations", source)
	if err != nil {
		t.Fatal(err)
	}
	if already || first == nil || firstReceipt.ID != "ir_saved" {
		t.Fatalf("first adoption = workspace:%v receipt:%+v already:%v", first, firstReceipt, already)
	}
	second, secondReceipt, already, err := AdoptPage(context.Background(), store, "switchboard", "conversations", source)
	if err != nil {
		t.Fatal(err)
	}
	if !already || second != first || secondReceipt != firstReceipt || store.creates != 2 {
		t.Fatalf("retry was not repeat-safe: already=%v sameWorkspace=%v sameReceipt=%v calls=%d", already, second == first, secondReceipt == firstReceipt, store.creates)
	}
	changed := append(append([]byte(nil), source...), ' ')
	if _, _, _, err := AdoptPage(context.Background(), store, "switchboard", "conversations", changed); err == nil || !errors.Is(err, workspaces.ErrConversionReceiptConflict) {
		t.Fatalf("changed source did not surface an explicit conversion conflict: %v", err)
	}
	if store.creates != 3 {
		t.Fatalf("changed source write attempt count = %d", store.creates)
	}
}

func TestInspectAgentDetailPageContractReportsParityObligations(t *testing.T) {
	source, err := os.ReadFile(filepath.Join(experienceImportTestdata(t), "switchboard/experience/pages/agent-detail.json"))
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := InspectPageContract(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(evidence.StateIDs) != 10 || len(evidence.StateDescriptions) != 10 || len(evidence.ElementIDs) != 10 || len(evidence.ClaimIDs) != 8 || len(evidence.RegionIDs) != 2 || len(evidence.ElementBindingIDs) != 10 || len(evidence.RegionBindingIDs) != 2 {
		t.Fatalf("agent-detail obligation inventory = %+v", evidence)
	}
	if evidence.StateIDs[0] != "default" || evidence.StateIDs[9] != "partial" || evidence.StateDescriptions[0] == "" {
		t.Fatalf("state contract order/descriptions were lost: %+v", evidence)
	}
	if !reflect.DeepEqual(evidence.RegionBindingIDs, []string{"activity-region", "grant-region"}) {
		t.Fatalf("region binding IDs are not stable/sorted: %v", evidence.RegionBindingIDs)
	}
	wantLifecycles := []RegionLifecycleEvidence{
		{RegionID: "grant-region", Kind: "async", States: []string{"loading", "ready", "error"}},
		{RegionID: "activity-region", Kind: "async", States: []string{"loading", "ready", "empty", "error"}},
	}
	if !reflect.DeepEqual(evidence.RegionLifecycles, wantLifecycles) {
		t.Fatalf("async lifecycle contract was lost: %+v", evidence.RegionLifecycles)
	}
}

func TestAdoptPagePersistsWorkspaceAndRecoveryReceiptTogether(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "adoption.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	if err := workspaces.Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join(experienceImportTestdata(t), "switchboard/experience/pages/conversations.json"))
	if err != nil {
		t.Fatal(err)
	}
	repo := workspaces.NewSQLiteRepository(db)
	created, receipt, reused, err := AdoptPage(context.Background(), repo, "switchboard", "conversations", source)
	if err != nil {
		t.Fatal(err)
	}
	if created == nil || created.GetId() != receipt.WorkspaceID || receipt.ID == "" || reused {
		t.Fatalf("first persisted adoption: workspace=%v receipt=%+v reused=%v", created, receipt, reused)
	}
	restored, err := RestorePage(func() Receipt {
		var value Receipt
		if err := json.Unmarshal([]byte(receipt.ReceiptJSON), &value); err != nil {
			t.Fatal(err)
		}
		return value
	}())
	if err != nil || !reflect.DeepEqual(restored, source) {
		t.Fatalf("durable receipt cannot restore exact source bytes: %v", err)
	}
	retried, retryReceipt, reused, err := AdoptPage(context.Background(), repo, "switchboard", "conversations", source)
	if err != nil || !reused || retried.GetId() != created.GetId() || retryReceipt.ID != receipt.ID {
		t.Fatalf("retry did not resolve to durable workspace/receipt: workspace=%v receipt=%+v reused=%v err=%v", retried, retryReceipt, reused, err)
	}
}

func TestPageConversionIsDryRunReversibleAndIdempotent(t *testing.T) {
	testdata := experienceImportTestdata(t)
	for _, tc := range []struct {
		page                 string
		wantSlots            int
		wantDevice           int
		wantButtons          int
		wantLegacyPlacements int
		wantUnplaced         int
		wantBindings         int
		wantNativeButtons    []string
		wantLegacyRegions    []string
	}{
		{
			page: "dashboard", wantSlots: 5, wantDevice: 2, wantUnplaced: 9, wantBindings: 1,
			wantNativeButtons: []string{}, wantLegacyRegions: []string{},
		},
		{
			page: "conversations", wantSlots: 10, wantDevice: 2, wantButtons: 2, wantLegacyPlacements: 8, wantBindings: 12,
			wantNativeButtons: []string{"start-action", "start-confirm"},
			wantLegacyRegions: []string{"page-heading", "filters", "thread-list-region", "transcript-region", "composer-region", "start-dialog", "agent-choice", "gate-region"},
		},
	} {
		t.Run(tc.page, func(t *testing.T) {
			sourcePath := filepath.Join(testdata, "switchboard/experience/pages/"+tc.page+".json")
			source, err := os.ReadFile(sourcePath)
			if err != nil {
				t.Fatal(err)
			}
			before := append([]byte(nil), source...)
			first, err := ConvertPage("switchboard", tc.page, source)
			if err != nil {
				t.Fatal(err)
			}
			second, err := ConvertPage("switchboard", tc.page, source)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(source, before) {
				t.Fatal("dry-run modified the source bytes")
			}
			if !reflect.DeepEqual(first.Receipt, second.Receipt) || !reflect.DeepEqual(first.Workspace, second.Workspace) {
				t.Fatal("repeated conversion did not produce an identical target and receipt")
			}
			if first.Workspace.GetBindingScenario() != "switchboard" || first.Workspace.GetBindingPage() != tc.page {
				t.Fatalf("workspace binding = %q/%q", first.Workspace.GetBindingScenario(), first.Workspace.GetBindingPage())
			}
			if first.Workspace.GetProvenance() != workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_IMPORTED {
				t.Fatalf("provenance = %v", first.Workspace.GetProvenance())
			}
			if !first.Receipt.Parity || first.Receipt.SourceHash == "" || first.Receipt.TargetHash == "" || len(first.Receipt.SourcePageJSON) == 0 || len(first.Receipt.SourceSketchJSON) == 0 {
				t.Fatalf("incomplete parity/rollback receipt: %+v", first.Receipt)
			}
			if err := VerifyResult(first); err != nil {
				t.Fatalf("valid dry-run receipt did not verify: %v", err)
			}
			if first.Receipt.RenderParity || first.Receipt.RenderGap == "" || !strings.Contains(first.Receipt.ContractGap, "not implemented") {
				t.Fatalf("native-render mismatch must remain explicit: %+v", first.Receipt)
			}
			contract, err := InspectPageContract(source)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(first.Receipt.ContractEvidence, contract) || len(contract.StateIDs) == 0 || len(contract.ElementIDs) == 0 || len(contract.ClaimIDs) == 0 {
				t.Fatalf("source page obligations were not carried into receipt: %+v", first.Receipt.ContractEvidence)
			}
			evidence := first.Receipt.RenderEvidence
			if len(evidence.MappedRegionIDs) != tc.wantSlots || len(evidence.NativeButtonRegionIDs) != tc.wantButtons || len(evidence.LegacyPlacementRegionIDs) != tc.wantLegacyPlacements || evidence.UnplacedItemCount != tc.wantUnplaced || evidence.RenderBindingCount != tc.wantBindings || !reflect.DeepEqual(evidence.NativeButtonRegionIDs, tc.wantNativeButtons) || !reflect.DeepEqual(evidence.LegacyPlacementRegionIDs, tc.wantLegacyRegions) {
				t.Fatalf("page-specific render evidence = %+v", evidence)
			}
			if evidence.TemplateExport != "CollectionPage" || evidence.TemplateAsset != "templates.collection-page" || evidence.TemplateVersion != "1.7.1" || len(evidence.TemplatePorts) != tc.wantSlots {
				t.Fatalf("template shell evidence = %+v", evidence)
			}
			for _, port := range evidence.TemplatePorts {
				if port.RegionID == "" || len(port.SlotPath) == 0 || port.NativeShell {
					t.Fatalf("port metadata was mistaken for native shell rendering: %+v", port)
				}
			}
			if tc.page == "dashboard" {
				var got []string
				for _, port := range evidence.TemplatePorts {
					got = append(got, port.TemplatePort)
				}
				want := []string{"header", "filters", "collection", "bulk-actions", "inspector"}
				if !reflect.DeepEqual(got, want) || !strings.Contains(first.Receipt.RenderGap, "does not render the templates.collection-page@1.7.1 shell") || !strings.Contains(first.Receipt.RenderGap, "[header, filters, collection, bulk-actions, inspector]") {
					t.Fatalf("dashboard exact template gap: ports=%v gap=%q", got, first.Receipt.RenderGap)
				}
			}
			page := first.Workspace.GetPages()[0]
			if page.GetPageTypeId() != "templates.collection-page" {
				t.Fatalf("page type = %q", page.GetPageTypeId())
			}
			var spec struct {
				Type     string `json:"type"`
				Children []struct {
					ID    string         `json:"id"`
					Type  string         `json:"type"`
					Props map[string]any `json:"props"`
				} `json:"children"`
			}
			if err := json.Unmarshal([]byte(page.GetSpecJson()), &spec); err != nil {
				t.Fatal(err)
			}
			if spec.Type != "page" || len(spec.Children) != tc.wantSlots {
				t.Fatalf("native page tree does not expose every legacy render port: root=%q slots=%d want=%d", spec.Type, len(spec.Children), tc.wantSlots)
			}
			if spec.Children[0].Type != "slot" || spec.Children[0].Props["legacyTemplateRegion"] == "" || spec.Children[0].Props["legacySlot"] == "" {
				t.Fatalf("first slot mapping = %+v", spec.Children[0])
			}
			if len(page.GetDisplays()) != tc.wantDevice {
				t.Fatalf("displays = %d, want %d", len(page.GetDisplays()), tc.wantDevice)
			}
			for _, display := range page.GetDisplays() {
				if len(display.GetStates()) != 1 {
					t.Fatalf("display %q has %d states", display.GetId(), len(display.GetStates()))
				}
				restored, err := RestoreSketch(display.GetStates()[0].GetArgsJson())
				if err != nil {
					t.Fatal(err)
				}
				if !sameJSON(restored, first.Receipt.SourceSketchJSON) {
					t.Fatalf("state %q cannot restore exact composition payload", display.GetStates()[0].GetId())
				}
				var adapter stateAdapter
				if err := json.Unmarshal([]byte(display.GetStates()[0].GetArgsJson()), &adapter); err != nil || !reflect.DeepEqual(adapter.SourcePageContract, contract) {
					t.Fatalf("state %q does not carry editable source obligations: %+v (%v)", display.GetStates()[0].GetId(), adapter.SourcePageContract, err)
				}
			}
		})
	}
}

func TestVerifyResultRejectsDamagedRecoveryEvidence(t *testing.T) {
	sourcePath := filepath.Join(experienceImportTestdata(t), "switchboard/experience/pages/conversations.json")
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	valid, err := ConvertPage("switchboard", "conversations", source)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(*Result){
		"source page":      func(r *Result) { r.Receipt.SourcePageJSON = append(r.Receipt.SourcePageJSON, ' ') },
		"source sketch":    func(r *Result) { r.Receipt.SourceSketchJSON = []byte(`{}`) },
		"target workspace": func(r *Result) { r.Workspace.Name = "changed" },
		"binding":          func(r *Result) { r.Workspace.BindingPage = "other" },
		"state adapter":    func(r *Result) { r.Workspace.Pages[0].Displays[0].States[0].ArgsJson = `{}` },
		"render evidence":  func(r *Result) { r.Receipt.RenderEvidence.LegacyPlacementRegionIDs = nil },
		"parity claim":     func(r *Result) { r.Receipt.Parity = false },
		"render parity":    func(r *Result) { r.Receipt.RenderParity = true },
		"render gap":       func(r *Result) { r.Receipt.RenderGap = "none" },
		"contract gap":     func(r *Result) { r.Receipt.ContractGap = "none" },
		"receipt identity": func(r *Result) { r.Receipt.Page = "dashboard" },
		"source revision":  func(r *Result) { r.Receipt.SourceRevision = "stale" },
	}
	for name, damage := range cases {
		t.Run(name, func(t *testing.T) {
			candidate := valid
			workspaceJSON, err := json.Marshal(valid.Workspace)
			if err != nil {
				t.Fatal(err)
			}
			var workspace workspacev1.Workspace
			if err := json.Unmarshal(workspaceJSON, &workspace); err != nil {
				t.Fatal(err)
			}
			candidate.Workspace = &workspace
			candidate.Receipt.SourcePageJSON = append([]byte(nil), valid.Receipt.SourcePageJSON...)
			candidate.Receipt.SourceSketchJSON = append([]byte(nil), valid.Receipt.SourceSketchJSON...)
			damage(&candidate)
			if err := VerifyResult(candidate); err == nil {
				t.Fatal("damaged recovery evidence was accepted")
			}
		})
	}
}

func TestReceiptSurvivesDurableJSONRoundTripAndRestoresExactPage(t *testing.T) {
	sourcePath := filepath.Join(experienceImportTestdata(t), "switchboard/experience/pages/conversations.json")
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ConvertPage("switchboard", "conversations", source)
	if err != nil {
		t.Fatal(err)
	}
	// Persist and reload the target aggregate and receipt independently, as a
	// caller would before authorizing any later workspace write.
	targetJSON, err := json.Marshal(result.Workspace)
	if err != nil {
		t.Fatal(err)
	}
	var reloadedWorkspace workspacev1.Workspace
	if err := json.Unmarshal(targetJSON, &reloadedWorkspace); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(result.Receipt)
	if err != nil {
		t.Fatal(err)
	}
	var persisted Receipt
	if err := json.Unmarshal(encoded, &persisted); err != nil {
		t.Fatal(err)
	}
	if err := VerifyResult(Result{Workspace: &reloadedWorkspace, Receipt: persisted}); err != nil {
		t.Fatalf("reloaded workspace and receipt did not verify together: %v", err)
	}
	restored, err := RestorePage(persisted)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(restored, source) {
		t.Fatal("receipt did not restore exact source page bytes after serialization")
	}
	if err := VerifySource(persisted, source); err != nil {
		t.Fatalf("unchanged source did not verify: %v", err)
	}
	if err := VerifySource(persisted, append(append([]byte(nil), source...), ' ')); err == nil {
		t.Fatal("stale source bytes were accepted")
	}
	mutated := persisted
	mutated.SourceSketchJSON = []byte(`{}`)
	if _, err := RestorePage(mutated); err == nil {
		t.Fatal("receipt with mismatched sketch evidence restored a page")
	}
	restored[0] ^= 0xff
	if !reflect.DeepEqual(persisted.SourcePageJSON, source) {
		t.Fatal("mutating restored page changed receipt-owned bytes")
	}
}

func TestDashboardConversionRejectsIdentityAndUnsupportedShape(t *testing.T) {
	for _, input := range []struct {
		name, scenario, page, json string
	}{
		{"page identity", "switchboard", "other", `{"page":{"id":"dashboard"},"sketch":{}}`},
		{"missing sketch", "switchboard", "dashboard", `{"page":{"id":"dashboard"}}`},
		{"no render", "switchboard", "dashboard", `{"page":{"id":"dashboard"},"sketch":{"template":{"asset":"x"}}}`},
	} {
		t.Run(input.name, func(t *testing.T) {
			if _, err := ConvertPage(input.scenario, input.page, []byte(input.json)); err == nil {
				t.Fatal("expected conversion error")
			}
		})
	}
}

func TestConversationButtonPlacementsGetNativeVisualFallbacks(t *testing.T) {
	sourcePath := filepath.Join(experienceImportTestdata(t), "switchboard/experience/pages/conversations.json")
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ConvertPage("switchboard", "conversations", source)
	if err != nil {
		t.Fatal(err)
	}
	if result.Receipt.RenderParity || !strings.Contains(result.Receipt.RenderGap, "render bindings") {
		t.Fatalf("button-only mapping must not overclaim full page parity: %+v", result.Receipt)
	}
	if !reflect.DeepEqual(result.Receipt.RenderEvidence.NativeButtonRegionIDs, []string{"start-action", "start-confirm"}) || len(result.Receipt.RenderEvidence.LegacyPlacementRegionIDs) != 8 {
		t.Fatalf("native versus legacy render evidence = %+v", result.Receipt.RenderEvidence)
	}
	var tree templateNode
	if err := json.Unmarshal([]byte(result.Workspace.GetPages()[0].GetSpecJson()), &tree); err != nil {
		t.Fatal(err)
	}
	found := map[string]templateNode{}
	for _, slot := range tree.Children {
		if len(slot.Children) == 1 && slot.Children[0].Type == "button" {
			label, ok := slot.Children[0].Props["text"].(string)
			if !ok {
				t.Fatalf("button fallback label has type %T", slot.Children[0].Props["text"])
			}
			if slot.Children[0].Props["variant"] != "primary" || slot.Children[0].Props["shape"] != "pill" || slot.Children[0].Props["ariaLabel"] != label {
				t.Errorf("button fallback props = %+v", slot.Children[0].Props)
			}
			found[slot.ID] = slot.Children[0]
		}
	}
	if len(found) != 2 || found["start-action"].Props["text"] != "Start a conversation" || found["start-confirm"].Props["text"] != "Start conversation" {
		t.Fatalf("native button fallbacks = %+v", found)
	}
	for _, slot := range tree.Children {
		if slot.ID == "thread-list-region" && len(slot.Children) != 0 {
			t.Fatalf("non-button placement unexpectedly received a native fallback: %+v", slot)
		}
	}
}

func TestInventoryCompositionsMatchesCurrentRepository(t *testing.T) {
	scenarioRoot := experienceImportTestdata(t)
	items, err := InventoryCompositions(scenarioRoot)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"switchboard/agent-detail", "switchboard/conversations", "switchboard/dashboard"}
	got := make([]string, 0, len(items))
	for _, item := range items {
		got = append(got, item.Scenario+"/"+item.Page)
		if item.ContentHash == "" {
			t.Errorf("%s has no content hash", item.Scenario+"/"+item.Page)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("inventory identities = %v, want %v", got, want)
	}
	convertible := map[string]bool{}
	for _, item := range items {
		convertible[item.Page] = item.CanConvert
	}
	if !convertible["dashboard"] || !convertible["conversations"] {
		t.Fatalf("template/render pages should be convertible: %+v", items)
	}
	for _, item := range items {
		if item.Page == "agent-detail" && (item.CanConvert || !strings.Contains(item.ConverterError, "template/render") || !strings.Contains(item.ConverterError, "does not define native Workspace content or behavior")) {
			t.Errorf("placement-only page should remain unsupported explicitly: %+v", item)
		}
	}
}

func TestConvertPageRejectsPlacementOnlyComposition(t *testing.T) {
	sourcePath := filepath.Join(experienceImportTestdata(t), "switchboard/experience/pages/agent-detail.json")
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ConvertPage("switchboard", "agent-detail", source); err == nil || !strings.Contains(err.Error(), "template/render") || !strings.Contains(err.Error(), "does not define native Workspace content or behavior") {
		t.Fatalf("placement-only conversion error = %v, want explicit missing render contract and native behavior boundary", err)
	}
}

func TestInventoryCompositionsFiltersIntentOnlyAndReportsUnsupported(t *testing.T) {
	root := t.TempDir()
	writeInventoryPage(t, root, "alpha", "composed", `{"page":{"id":"composed"},"sketch":{"template":{"asset":"templates.page"},"render":{"templateExport":"Page","regions":[{"id":"main","slot":["main"]}]},"placements":[{"region":"main","fills":{"asset":"controls.button"}}],"regions":[{"id":"main"}]}}`)
	writeInventoryPage(t, root, "alpha", "intent", `{"page":{"id":"intent"},"sketch":{"viewport":"desktop","intent":{"summary":"Collect information"},"regions":[{"id":"main","purpose":"Collect a response"}]}}`)
	items, err := InventoryCompositions(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Page != "composed" {
		t.Fatalf("inventory = %+v, intent-only sketch should be excluded", items)
	}
	if items[0].CanConvert || !strings.Contains(items[0].ConverterError, "not renderable") {
		t.Fatalf("unsupported page conversion status = %+v", items[0])
	}
}

func TestInventoryCompositionsRejectsMalformedJSON(t *testing.T) {
	root := t.TempDir()
	writeInventoryPage(t, root, "alpha", "broken", `{"page":`)
	if _, err := InventoryCompositions(root); err == nil || !strings.Contains(err.Error(), "inspect") {
		t.Fatalf("malformed page error = %v, want contextual error", err)
	}
}

func TestInventoryCompositionsRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()
	if err := os.MkdirAll(filepath.Join(external, "experience", "pages"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(external, "experience"), filepath.Join(root, "alpha", "experience")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := InventoryCompositions(root); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlink escape error = %v, want rejection", err)
	}
}

func writeInventoryPage(t *testing.T, root, scenario, page, body string) {
	t.Helper()
	path := filepath.Join(root, scenario, "experience", "pages", page+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func experienceImportTestdata(t *testing.T) string {
	t.Helper()
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(workingDirectory, "testdata")
}
