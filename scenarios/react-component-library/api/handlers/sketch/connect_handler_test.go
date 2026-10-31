package sketch

import (
	"bytes"
	"context"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/storage"
	"net/http/httptest"
	"os"
	"path/filepath"
	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/catalogsearch"
	"testing"
	"time"

	"connectrpc.com/connect"
	previewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/preview"
	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"
	sketchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch/sketch_v1connect"
	"react-component-library/internal/preview"
	internal "react-component-library/internal/sketch"
)

func TestWireEditsRequireExactRevisionAndPreserveRestoreData(t *testing.T) {
	// [REQ:EPD-001]
	root := t.TempDir()
	path := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"claims":[{"id":"preserve"}],"sketch":{"viewport":"desktop","x-extension":true}}`), 0600); err != nil {
		t.Fatal(err)
	}
	_, handler := sketchconnect.NewSketchServiceHandler(NewConnectHandler(Deps{Store: internal.NewStore(root)}))
	server := httptest.NewServer(handler)
	defer server.Close()
	client := sketchconnect.NewSketchServiceClient(server.Client(), server.URL)
	ctx := context.Background()
	target := &sketchv1.SketchTarget{Scenario: "demo", Page: "home"}
	initial, err := client.GetSketch(ctx, connect.NewRequest(&sketchv1.GetSketchRequest{Target: target}))
	if err != nil {
		t.Fatal(err)
	}
	if initial.Msg.GetContentHash() == "" {
		t.Fatal("read omitted content hash")
	}
	_, err = client.Placeholder(ctx, connect.NewRequest(&sketchv1.PlaceholderRequest{Target: target, Region: "main", Placeholder: "custom", Intent: "Collect a domain-specific response"}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("missing precondition: %v", err)
	}
	placed, err := client.Placeholder(ctx, connect.NewRequest(&sketchv1.PlaceholderRequest{Target: target, Region: "main", Placeholder: "custom", Intent: "Collect a domain-specific response", ExpectedContentHash: initial.Msg.GetContentHash()}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.AddNote(ctx, connect.NewRequest(&sketchv1.AddNoteRequest{Target: target, Scope: "page", Text: "stale edit", ExpectedContentHash: initial.Msg.GetContentHash()}))
	if connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("stale edit: %v", err)
	}
	wireError := err.(*connect.Error)
	if len(wireError.Details()) != 1 {
		t.Fatalf("missing conflict detail: %v", err)
	}
	detail, err := wireError.Details()[0].Value()
	if err != nil {
		t.Fatal(err)
	}
	conflict, ok := detail.(*sketchv1.RevisionConflict)
	if !ok || conflict.GetCurrentContentHash() != placed.Msg.GetContentHash() {
		t.Fatalf("conflict detail: %v", detail)
	}
	unplaced, err := client.Unplace(ctx, connect.NewRequest(&sketchv1.UnplaceRequest{Target: target, Region: "main", Reason: "Relocate the primary region", ExpectedContentHash: placed.Msg.GetContentHash()}))
	if err != nil {
		t.Fatal(err)
	}
	items := unplaced.Msg.GetSketch().GetUnplaced()
	if len(items) != 1 || items[0].GetRegion() != "main" || items[0].GetPlacement().GetFills().GetPlaceholder() != "custom" {
		t.Fatalf("wire discarded restorable placement: %v", items)
	}
	saved, err := client.PutSketch(ctx, connect.NewRequest(&sketchv1.PutSketchRequest{Target: target, Sketch: unplaced.Msg.GetSketch(), ExpectedContentHash: unplaced.Msg.GetContentHash()}))
	if err != nil {
		t.Fatal(err)
	}
	if saved.Msg.GetChanged() || saved.Msg.GetContentHash() != unplaced.Msg.GetContentHash() {
		t.Fatal("wire round trip changed revision")
	}
	history, err := client.GetHistory(ctx, connect.NewRequest(&sketchv1.GetHistoryRequest{Target: target}))
	if err != nil {
		t.Fatal(err)
	}
	if len(history.Msg.GetRevisions()) != 3 || history.Msg.GetCurrentContentHash() != saved.Msg.GetContentHash() {
		t.Fatalf("wire history mismatch: %v", history.Msg)
	}
	recovered, err := client.RecoverSketch(ctx, connect.NewRequest(&sketchv1.RecoverSketchRequest{Target: target, ExpectedContentHash: saved.Msg.GetContentHash()}))
	if err != nil || recovered.Msg.GetChanged() {
		t.Fatalf("wire recovery: %v %v", recovered, err)
	}
	restored, err := internal.NewStore(root).Load("demo", "home")
	if err != nil {
		t.Fatal(err)
	}
	if len(restored.Unplaced) != 1 || restored.Unplaced[0].Placement == nil {
		t.Fatal("PutSketch lost restoration data")
	}
	if _, err := internal.Restore(restored, "custom", "main"); err != nil {
		t.Fatal(err)
	}
}

func TestSketchMutationsUseLeaseAndNeverFallBackToLive(t *testing.T) {
	live := t.TempDir()
	leased := t.TempDir()
	seed := func(root, viewport string) string {
		t.Helper()
		path := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(`{"sketch":{"viewport":"`+viewport+`"}}`), 0600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	livePath := seed(live, "live")
	seed(filepath.Join(leased, "workspace"), "leased")
	roots := filerouting.New(storage.Paths{ConfigDir: filepath.Join(t.TempDir(), "config")})
	clock := schedule.NewFake(time.Unix(100, 0))
	roots.SetClock(clock)
	handler := NewConnectHandler(Deps{Store: internal.NewStore(live), StoreFor: routedRepository(live, roots), RecordWrite: roots.RecordWrite})
	target := &sketchv1.SketchTarget{Scenario: "demo", Page: "home"}
	ctx := database.WithTestMode(context.Background())
	_, err := handler.GetSketch(ctx, connect.NewRequest(&sketchv1.GetSketchRequest{Target: target}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("test read without lease: %v", err)
	}
	if err := roots.InstallTestRoots(storage.Paths{ConfigDir: leased}, "design-test", time.Minute); err != nil {
		t.Fatal(err)
	}
	current, err := handler.GetSketch(ctx, connect.NewRequest(&sketchv1.GetSketchRequest{Target: target}))
	if err != nil {
		t.Fatal(err)
	}
	if current.Msg.GetSketch().GetViewport() != "leased" {
		t.Fatal("test request read live source")
	}
	written, err := handler.PutSketch(ctx, connect.NewRequest(&sketchv1.PutSketchRequest{Target: target, ExpectedContentHash: current.Msg.GetContentHash(), Sketch: &sketchv1.Sketch{Viewport: "edited"}}))
	if err != nil {
		t.Fatal(err)
	}
	if !written.Msg.GetChanged() {
		t.Fatal("leased mutation did not write")
	}
	raw, err := os.ReadFile(livePath)
	if err != nil || string(raw) != `{"sketch":{"viewport":"live"}}` {
		t.Fatalf("live tree changed: %s %v", raw, err)
	}
	stats := roots.LeaseStats()
	if stats.TestRootWrites != 1 || stats.PrimaryWritesDuringTestMode != 0 {
		t.Fatalf("write routing: %+v", stats)
	}
	clock.Advance(2 * time.Minute)
	_, err = handler.PutSketch(ctx, connect.NewRequest(&sketchv1.PutSketchRequest{Target: target, ExpectedContentHash: written.Msg.GetContentHash(), Sketch: &sketchv1.Sketch{Viewport: "expired"}}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("expired lease mutation: %v", err)
	}
	raw, err = os.ReadFile(livePath)
	if err != nil || string(raw) != `{"sketch":{"viewport":"live"}}` {
		t.Fatal("expired lease leaked to live tree")
	}
}

func TestImportDryRunConflictAndIdempotentPublication(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{"page":{"purpose":"Review important incoming work and respond."},"claims":[{"id":"preserve"}],"sketch":{"x-extension":true}}`)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	h := NewConnectHandler(Deps{Store: internal.NewStore(root), ImportSearch: func(context.Context) (*catalogsearch.Index, error) { return catalogsearch.New(), nil }})
	target := &sketchv1.SketchTarget{Scenario: "demo", Page: "home"}
	preview, err := h.ImportPage(context.Background(), connect.NewRequest(&sketchv1.ImportPageRequest{Target: target}))
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, after) || preview.Msg.Written || len(preview.Msg.Sketch.Regions) != 1 {
		t.Fatal("dry-run mutated page or failed to propose L0 region")
	}
	_, err = h.ImportPage(context.Background(), connect.NewRequest(&sketchv1.ImportPageRequest{Target: target, Write: true, ExpectedContentHash: "stale"}))
	if connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("stale import: %v", err)
	}
	saved, err := h.ImportPage(context.Background(), connect.NewRequest(&sketchv1.ImportPageRequest{Target: target, Write: true, ExpectedContentHash: preview.Msg.ContentHash}))
	if err != nil {
		t.Fatal(err)
	}
	repeat, err := h.ImportPage(context.Background(), connect.NewRequest(&sketchv1.ImportPageRequest{Target: target, Write: true, ExpectedContentHash: saved.Msg.ContentHash}))
	if err != nil {
		t.Fatal(err)
	}
	if repeat.Msg.ContentHash != saved.Msg.ContentHash {
		t.Fatal("repeat import created a new revision")
	}
	after, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(after, []byte(`"claims":[{"id":"preserve"}]`)) || !bytes.Contains(after, []byte(`"x-extension": true`)) {
		t.Fatalf("import discarded authored contract: %s", after)
	}
}

func TestTemplateWireRequiresConfirmationAndRegionNotesUseDeclaredIdentity(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{"regions":[{"id":"semantic-region","purpose":"Keep its purpose"}],"sketch":{"regions":[{"id":"old"}],"placements":[{"region":"old","fills":{"asset":"controls.button"},"state":"declared"}]}}`)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	store := internal.NewStore(root)
	current, err := store.Read("demo", "home")
	if err != nil {
		t.Fatal(err)
	}
	accepts := ""
	h := NewConnectHandler(Deps{Store: store, Catalog: func(context.Context) ([]catalogcoverage.Asset, error) {
		return []catalogcoverage.Asset{{ID: "templates.new", Kind: "page-template", Regions: []string{"main"}, RegionAccepts: map[string]string{"main": accepts}}, {ID: "controls.button", Domain: "controls"}}, nil
	}})
	target := &sketchv1.SketchTarget{Scenario: "demo", Page: "home"}
	note, err := h.AddNote(context.Background(), connect.NewRequest(&sketchv1.AddNoteRequest{Target: target, Scope: "semantic-region", Text: "Preserve this interaction", ExpectedContentHash: current.ContentHash}))
	if err != nil {
		t.Fatalf("authored region note refused: %v", err)
	}
	_, err = h.AddNote(context.Background(), connect.NewRequest(&sketchv1.AddNoteRequest{Target: target, Scope: "absent-region", Text: "Unknown scope", ExpectedContentHash: note.Msg.ContentHash}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("unknown scope accepted: %v", err)
	}
	request := &sketchv1.SetTemplateRequest{Target: target, Asset: "templates.new", ExpectedContentHash: note.Msg.ContentHash}
	accepts = "forms"
	request.Remap = []*sketchv1.RegionRemap{{From: "old", To: "main"}}
	_, err = h.SetTemplate(context.Background(), connect.NewRequest(request))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("incompatible filler accepted: %v", err)
	}
	constrained, _ := store.Read("demo", "home")
	if constrained.ContentHash != note.Msg.ContentHash {
		t.Fatal("incompatible swap changed authored page")
	}
	accepts = ""
	request.Remap = nil
	preview, err := h.SetTemplate(context.Background(), connect.NewRequest(request))
	if err != nil {
		t.Fatal(err)
	}
	after, err := store.Read("demo", "home")
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Msg.RequiresConfirmation || preview.Msg.Changed || after.ContentHash != note.Msg.ContentHash || len(preview.Msg.NewlyUnplaced) != 1 {
		t.Fatalf("unconfirmed swap published: %v", preview.Msg)
	}
	request.ConfirmUnplaced = true
	request.Preview = true
	_, err = h.SetTemplate(context.Background(), connect.NewRequest(request))
	if err != nil {
		t.Fatal(err)
	}
	after, _ = store.Read("demo", "home")
	if after.ContentHash != note.Msg.ContentHash {
		t.Fatal("explicit preview published")
	}
	request.Preview = false
	saved, err := h.SetTemplate(context.Background(), connect.NewRequest(request))
	if err != nil {
		t.Fatal(err)
	}
	if !saved.Msg.Changed || saved.Msg.RequiresConfirmation || len(saved.Msg.Sketch.Unplaced) != 1 {
		t.Fatalf("confirmed swap: %v", saved.Msg)
	}
	_, err = h.SetTemplate(context.Background(), connect.NewRequest(request))
	if connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("stale template write: %v", err)
	}
}

func TestRenderSettingsSurviveWireConversion(t *testing.T) {
	doc := internal.Document{Render: &internal.RenderSettings{TemplateExport: "CollectionPage", Regions: []internal.RenderRegion{{ID: "inspector", Export: "Button", Slot: []string{"data", "inspector"}}}, Bindings: map[string]any{"inspector": map[string]any{"onClick": map[string]any{"$handler": "inspect"}}}, Fixtures: []internal.RenderFixture{{Target: "inspector", Asset: "fixtures.user-directory", Version: "1.0.0", State: "typical", Field: "records", Prop: []string{"items"}}}}}
	doc.Regions = []internal.Region{{ID: "inspector", Locked: true}}
	got := fromProto(toProto(doc))
	if len(got.Regions) != 1 || !got.Regions[0].Locked {
		t.Fatal("region lock lost on wire")
	}
	if got.Render == nil || got.Render.TemplateExport != "CollectionPage" || len(got.Render.Fixtures) != 1 || got.Render.Regions[0].Slot[1] != "inspector" {
		t.Fatalf("render contract lost: %+v", got.Render)
	}
	handler := got.Render.Bindings["inspector"].(map[string]any)["onClick"].(map[string]any)["$handler"]
	if handler != "inspect" {
		t.Fatal("inert callback marker changed", handler)
	}
}

type renderProbe struct {
	calls int
	run   func()
}

func (p *renderProbe) RenderPrepared(_ context.Context, prepared preview.PreparedComposition, _ *previewv1.RenderCompositionRequest) (*connect.Response[previewv1.RenderCompositionResponse], error) {
	p.calls++
	if p.run != nil {
		p.run()
	}
	return connect.NewResponse(&previewv1.RenderCompositionResponse{Html: "<main>fixture</main>", RenderHash: prepared.Composition.Revision}), nil
}
func TestRenderSketchRejectsStaleInputsAndConcurrentEdits(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"sketch":{"template":{"asset":"templates.page","version":"1.0.0"},"render":{"templateExport":"Page","bindings":{"$template":{}}}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	store := internal.NewStore(root)
	snapshot, err := store.Read("demo", "home")
	if err != nil {
		t.Fatal(err)
	}
	probe := &renderProbe{}
	h := NewConnectHandler(Deps{Store: store, Renderer: probe})
	request := &sketchv1.RenderSketchRequest{Target: &sketchv1.SketchTarget{Scenario: "demo", Page: "home"}, ExpectedContentHash: "stale", MissingLabel: "Missing", FailedLabel: "Failed"}
	_, err = h.RenderSketch(context.Background(), connect.NewRequest(request))
	if connect.CodeOf(err) != connect.CodeAborted || probe.calls != 0 {
		t.Fatal("stale request reached renderer", err)
	}
	request.ExpectedContentHash = snapshot.ContentHash
	result, err := h.RenderSketch(context.Background(), connect.NewRequest(request))
	if err != nil || result.Msg.RenderHash != snapshot.ContentHash {
		t.Fatal("exact snapshot did not render", err)
	}
	probe.run = func() {
		doc := snapshot.Document
		doc.Viewport = "phone"
		if _, err := store.Save("demo", "home", snapshot.ContentHash, doc); err != nil {
			t.Fatal(err)
		}
	}
	_, err = h.RenderSketch(context.Background(), connect.NewRequest(request))
	if connect.CodeOf(err) != connect.CodeAborted {
		t.Fatal("concurrent edit returned stale render evidence", err)
	}
}

func TestArchivedCandidateRenderIgnoresLaterPageEdits(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"sketch":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	store := internal.NewStore(root)
	base, err := store.Read("demo", "home")
	if err != nil {
		t.Fatal(err)
	}
	doc := internal.Document{Template: &internal.AssetRef{Asset: "templates.page", Version: "1.0.0"}, Render: &internal.RenderSettings{TemplateExport: "Page", Bindings: map[string]any{"$template": map[string]any{}}}}
	candidate, err := store.SaveCandidate("demo", "home", "candidate-one", base.ContentHash, doc)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Save("demo", "home", base.ContentHash, internal.Document{Viewport: "phone"}); err != nil {
		t.Fatal(err)
	}
	h := NewConnectHandler(Deps{Store: store, Renderer: &renderProbe{}})
	result, err := h.RenderCandidate(context.Background(), connect.NewRequest(&sketchv1.RenderCandidateRequest{Candidate: &sketchv1.CandidateReference{Scenario: "demo", DesignId: candidate.DesignID, Hash: candidate.Hash}, MissingLabel: "Missing", FailedLabel: "Failed"}))
	if err != nil || result.Msg.RenderHash != candidate.Hash {
		t.Fatal("archived render lost immutable identity", err)
	}
}

func TestDesignConstraintsWirePreservesPresenceAndExplicitFalse(t *testing.T) {
	disabled := false
	for _, constraints := range []*internal.DesignConstraints{nil, {}, {DesignSource: "DESIGN.md", PreserveRoutes: &disabled, PreserveBusinessBehavior: &disabled}} {
		original := internal.Document{Intent: &internal.DesignIntent{Intent: "Browse", Constraints: constraints}}
		wire := toProto(original)
		restored := intentFromProto(wire.Intent)
		if constraints == nil {
			if restored.Constraints != nil {
				t.Fatal("legacy absent constraints changed")
			}
			continue
		}
		c := restored.Constraints
		if c == nil || c.DesignSource != constraints.DesignSource {
			t.Fatal("source lost")
		}
		if constraints.PreserveRoutes == nil {
			if c.PreserveRoutes != nil || c.PreserveBusinessBehavior != nil {
				t.Fatal("absent flags acquired values")
			}
		} else if c.PreserveRoutes == nil || *c.PreserveRoutes || c.PreserveBusinessBehavior == nil || *c.PreserveBusinessBehavior {
			t.Fatal("explicit false lost")
		}
	}
}

func TestProposalReadsRoutedSourceAndRejectsStaleSourceHash(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "scenarios", "demo")
	if err := os.MkdirAll(filepath.Join(dir, "experience", "pages"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "experience", "pages", "home.json"), []byte(`{"sketch":{"viewport":"desktop"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "DESIGN.md"), []byte("# Routed design\n"), 0600); err != nil {
		t.Fatal(err)
	}
	routed := internal.NewStore(root)
	h := NewConnectHandler(Deps{
		Store:        internal.NewStore(t.TempDir()),
		StoreFor:     func(context.Context) (internal.Repository, error) { return routed, nil },
		Catalog:      func(context.Context) ([]catalogcoverage.Asset, error) { return nil, nil },
		ImportSearch: func(context.Context) (*catalogsearch.Index, error) { return catalogsearch.New(), nil },
	})
	snapshot, err := routed.Read("demo", "home")
	if err != nil {
		t.Fatal(err)
	}
	request := &sketchv1.ProposeSketchRequest{
		Target: &sketchv1.SketchTarget{Scenario: "demo", Page: "home"}, ExpectedContentHash: snapshot.ContentHash, CandidateLimit: 1,
		Intent: &sketchv1.DesignIntent{Intent: "Browse", Users: []string{"Operator"}, PrimaryTasks: []string{"Find records"}, Target: "react-vite", Kit: "vrooli-default", Viewports: []string{"desktop"},
			Constraints: &sketchv1.DesignConstraints{DesignSource: "path:DESIGN.md"}},
	}
	first, err := h.ProposeSketch(context.Background(), connect.NewRequest(request))
	if err != nil {
		t.Fatal(err)
	}
	source := first.Msg.DesignSource
	if source == nil || source.Content != "# Routed design\n" || source.Path != "DESIGN.md" || len(source.ContentHash) != 64 {
		t.Fatalf("wrong source: %+v", source)
	}
	request.Intent.Constraints.DesignSourceHash = source.ContentHash
	if err := os.WriteFile(filepath.Join(dir, "DESIGN.md"), []byte("# New design\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := h.ProposeSketch(context.Background(), connect.NewRequest(request)); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("stale source accepted: %v", err)
	}
	request.Intent.Constraints.DesignSource = ""
	if _, err := h.ProposeSketch(context.Background(), connect.NewRequest(request)); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("source hash without source accepted: %v", err)
	}
	request.Intent.Constraints.DesignSourceHash = ""
	absent, err := h.ProposeSketch(context.Background(), connect.NewRequest(request))
	if err != nil || absent.Msg.DesignSource != nil {
		t.Fatal("optional source failed", err)
	}
	after, err := routed.Read("demo", "home")
	if err != nil || after.ContentHash != snapshot.ContentHash {
		t.Fatal("proposal mutated page", err)
	}
}
