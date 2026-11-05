package candidates

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"brand-manager/internal/assets"
	"brand-manager/internal/generation"
	"brand-manager/internal/imagetools"
	"brand-manager/internal/render"
)

var (
	pngBytes = []byte("\x89PNG\r\n\x1a\nconcept")
	markSVG  = []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><path d="M10 10L90 90" stroke="#22d3ee"/></svg>`)
)

// memStore is an in-memory Store.
type memStore struct {
	mu   sync.Mutex
	rows []Candidate
	seq  int
}

func (m *memStore) Create(_ context.Context, c Candidate) (Candidate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seq++
	if c.ID == "" {
		c.ID = fmt.Sprintf("cand-%d", m.seq)
	}
	c.CreatedAt = time.Unix(int64(m.seq), 0)
	m.rows = append(m.rows, c)
	return c, nil
}

func (m *memStore) Get(_ context.Context, id string) (Candidate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.rows {
		if c.ID == id {
			return c, nil
		}
	}
	return Candidate{}, ErrNotFound{ID: id}
}

func (m *memStore) List(_ context.Context, brandID string, status Status, limit, _ int) ([]Candidate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Candidate
	for i := len(m.rows) - 1; i >= 0; i-- {
		c := m.rows[i]
		if c.BrandID == brandID && (status == "" || c.Status == status) {
			out = append(out, c)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *memStore) UpdateStatus(_ context.Context, id string, status Status, note string) (Candidate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.rows {
		if m.rows[i].ID == id {
			m.rows[i].Status = status
			if note != "" {
				m.rows[i].Note = note
			}
			return m.rows[i], nil
		}
	}
	return Candidate{}, ErrNotFound{ID: id}
}

func (m *memStore) SupersedePicked(_ context.Context, brandID, exceptID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.rows {
		if m.rows[i].BrandID == brandID && m.rows[i].Status == StatusPicked && m.rows[i].ID != exceptID {
			m.rows[i].Status = StatusSuperseded
		}
	}
	return nil
}

func (m *memStore) FindByAsset(_ context.Context, brandID, assetID string) (Candidate, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.rows {
		if c.BrandID == brandID && c.AssetID == assetID {
			return c, true, nil
		}
	}
	return Candidate{}, false, nil
}

func (m *memStore) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.rows)
}

// memAssets is an in-memory AssetStore.
type memAssets struct {
	mu   sync.Mutex
	byID map[string]assets.Content
	seq  int
}

func newMemAssets() *memAssets { return &memAssets{byID: map[string]assets.Content{}} }

func (m *memAssets) Upload(_ context.Context, in assets.UploadInput) (assets.Asset, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seq++
	id := fmt.Sprintf("asset-%d", m.seq)
	m.byID[id] = assets.Content{Filename: in.Filename, MimeType: in.MimeType, Bytes: in.Content}
	return assets.Asset{ID: id, BrandID: in.BrandID, Filename: in.Filename, MimeType: in.MimeType}, nil
}

func (m *memAssets) Download(_ context.Context, id string) (assets.Content, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.byID[id]
	if !ok {
		return assets.Content{}, fmt.Errorf("asset %q not found", id)
	}
	return c, nil
}

func (m *memAssets) put(id, mime string, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.byID[id] = assets.Content{Filename: id, MimeType: mime, Bytes: data}
}

// fakeBackend records image-tools calls.
type fakeBackend struct {
	mu          sync.Mutex
	generates   []generation.ImageGenerateRequest
	edits       []generation.ImageEditRequest
	rasterized  [][]byte
	vectorized  []imagetools.VectorizeOptions
	failRoles   map[string]bool
	tier        string
	delay       time.Duration
	inflight    int32
	maxInflight int32
}

func (f *fakeBackend) track() func() {
	n := atomic.AddInt32(&f.inflight, 1)
	for {
		cur := atomic.LoadInt32(&f.maxInflight)
		if n <= cur || atomic.CompareAndSwapInt32(&f.maxInflight, cur, n) {
			break
		}
	}
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	return func() { atomic.AddInt32(&f.inflight, -1) }
}

func (f *fakeBackend) output(role string) (generation.ImageOutput, error) {
	if f.failRoles[role] {
		return generation.ImageOutput{}, errors.New("role unavailable: " + role)
	}
	tier := f.tier
	if tier == "" {
		tier = "byok-cloud"
	}
	return generation.ImageOutput{Data: pngBytes, MimeType: "image/png", ModelID: "bytedance-seed/seedream-4.5", Tier: tier}, nil
}

func (f *fakeBackend) Generate(_ context.Context, req generation.ImageGenerateRequest) (generation.ImageOutput, error) {
	defer f.track()()
	f.mu.Lock()
	f.generates = append(f.generates, req)
	f.mu.Unlock()
	return f.output(req.OpenRouterRole)
}

func (f *fakeBackend) Edit(_ context.Context, req generation.ImageEditRequest) (generation.ImageOutput, error) {
	defer f.track()()
	f.mu.Lock()
	f.edits = append(f.edits, req)
	f.mu.Unlock()
	return f.output(req.OpenRouterRole)
}

func (f *fakeBackend) RemoveBackground(context.Context, generation.ImageRemoveBackgroundRequest) (generation.ImageOutput, error) {
	return generation.ImageOutput{Data: pngBytes, MimeType: "image/png"}, nil
}

func (f *fakeBackend) Vectorize(_ context.Context, _ []byte, opts imagetools.VectorizeOptions) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.vectorized = append(f.vectorized, opts)
	return markSVG, nil
}

func (f *fakeBackend) RemoveObject(context.Context, []byte, []byte) (generation.ImageOutput, error) {
	return generation.ImageOutput{Data: pngBytes, MimeType: "image/png"}, nil
}

func (f *fakeBackend) Rasterize(_ context.Context, svg []byte, _, _ int, _ string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rasterized = append(f.rasterized, svg)
	return []byte("\x89PNG\r\n\x1a\nrasterized"), nil
}

// fakeBrands resolves by id or case-insensitive name.
type fakeBrands map[string]BrandInfo

func (b fakeBrands) Resolve(_ context.Context, ref string) (BrandInfo, error) {
	for _, info := range b {
		if info.ID == ref || strings.EqualFold(info.Name, ref) {
			return info, nil
		}
	}
	return BrandInfo{}, errors.New("brand not found")
}

type noopMarks struct{}

func (noopMarks) SetMarkAsset(context.Context, string, string) error { return nil }

var midnight = render.Style{BackgroundTop: "#15243c", BackgroundBottom: "#0b1728", AccentColor: "#22d3ee", CornerRatio: 0.21875, MarkScale: 0.86}

func newTestService(backend *fakeBackend, brands fakeBrands) (*Service, *memStore, *memAssets) {
	store := &memStore{}
	store2 := newMemAssets()
	return NewService(store, store2, backend, noopMarks{}, brands), store, store2
}

func testBrands() fakeBrands {
	return fakeBrands{
		"vega":   {ID: "b-vega", Name: "Vega", Description: "real-time system monitoring", Style: midnight},
		"aquila": {ID: "b-aquila", Name: "Aquila", MarkAssetID: "aquila-mark", Style: midnight},
	}
}

func TestExploreRendersConceptsWithTheIllustrationRoleAndBrandPolicy(t *testing.T) {
	backend := &fakeBackend{}
	svc, _, _ := newTestService(backend, testBrands())

	got, _, err := svc.Explore(context.Background(), ExploreInput{BrandID: "b-vega", Concepts: []string{"lyre", "gauge"}, Variations: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 || len(backend.generates) != 4 {
		t.Fatalf("candidates=%d generates=%d, want 4 each", len(got), len(backend.generates))
	}
	for _, req := range backend.generates {
		if req.OpenRouterRole != roleConcept {
			t.Errorf("role = %q, want the illustration role %q", req.OpenRouterRole, roleConcept)
		}
		if !req.AllowBYOK || req.QualityPolicy != "quality" || req.FallbackPolicy != "any" {
			t.Errorf("policy = byok:%t quality:%q fallback:%q, want the brand image defaults", req.AllowBYOK, req.QualityPolicy, req.FallbackPolicy)
		}
		if req.Width != conceptEdge || req.Height != conceptEdge {
			t.Errorf("size = %dx%d, want %d", req.Width, req.Height, conceptEdge)
		}
		if !strings.Contains(req.Prompt, "Vega") || strings.Contains(req.Prompt, "flat colours") {
			t.Errorf("prompt should name the brand and ask for a rendered icon, got %q", req.Prompt)
		}
	}
	for _, c := range got {
		if c.Model != "bytedance-seed/seedream-4.5" || c.Role != roleConcept || c.Status != StatusProposed {
			t.Errorf("candidate = %+v, want the upstream model, role and PROPOSED", c)
		}
	}
}

func TestExploreFallsBackToTheQualityRole(t *testing.T) {
	backend := &fakeBackend{failRoles: map[string]bool{roleConcept: true}}
	svc, _, _ := newTestService(backend, testBrands())

	got, _, err := svc.Explore(context.Background(), ExploreInput{BrandID: "b-vega", Concepts: []string{"lyre"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Role != roleConceptQuality {
		t.Fatalf("candidates = %+v, want one rendered with %q", got, roleConceptQuality)
	}
}

func TestExploreRendersInParallelWithinTheBound(t *testing.T) {
	backend := &fakeBackend{delay: 40 * time.Millisecond}
	svc, _, _ := newTestService(backend, testBrands())

	start := time.Now()
	got, _, err := svc.Explore(context.Background(), ExploreInput{BrandID: "b-vega", Concepts: []string{"a", "b", "c", "d"}, Variations: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 8 {
		t.Fatalf("candidates = %d, want 8", len(got))
	}
	peak := atomic.LoadInt32(&backend.maxInflight)
	if peak < 2 || peak > exploreConcurrency {
		t.Fatalf("peak concurrency = %d, want between 2 and %d", peak, exploreConcurrency)
	}
	if elapsed := time.Since(start); elapsed > 8*backend.delay {
		t.Fatalf("explore took %s, no faster than serial", elapsed)
	}
}

func TestExploreKeepsEveryResultWhenTheCallerGoesAway(t *testing.T) {
	backend := &fakeBackend{delay: 10 * time.Millisecond}
	svc, store, _ := newTestService(backend, testBrands())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // the client disconnected before any generation finished

	got, _, err := svc.Explore(ctx, ExploreInput{BrandID: "b-vega", Concepts: []string{"a", "b"}, Variations: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 || store.count() != 4 {
		t.Fatalf("returned=%d stored=%d, want every generation stored", len(got), store.count())
	}
}

func TestExploreMatchesTheRasterAncestorOfAReferenceBrandsPickedMark(t *testing.T) {
	backend := &fakeBackend{}
	svc, store, assetStore := newTestService(backend, testBrands())
	ctx := context.Background()

	approved := []byte("\x89PNG\r\n\x1a\napproved-aquila-concept")
	assetStore.put("aquila-raster", "image/png", approved)
	assetStore.put("aquila-mark", "image/svg+xml", markSVG)
	parent, _ := store.Create(ctx, Candidate{BrandID: "b-aquila", AssetID: "aquila-raster", MediaType: "image/png", Status: StatusProposed})
	_, _ = store.Create(ctx, Candidate{BrandID: "b-aquila", AssetID: "aquila-mark", MediaType: "image/svg+xml", ParentID: parent.ID, Status: StatusPicked})

	got, _, err := svc.Explore(ctx, ExploreInput{BrandID: "b-vega", Concepts: []string{"lyre"}, Variations: 2, StyleReferenceBrand: "aquila"})
	if err != nil {
		t.Fatal(err)
	}
	if len(backend.generates) != 0 || len(backend.edits) != 2 {
		t.Fatalf("generates=%d edits=%d, want reference-conditioned edits only", len(backend.generates), len(backend.edits))
	}
	for _, req := range backend.edits {
		if string(req.Source) != string(approved) {
			t.Fatalf("reference = %q, want the approved raster concept", req.Source)
		}
		if req.OpenRouterRole != roleReference || !strings.Contains(req.Instruction, "reference") {
			t.Fatalf("edit role=%q instruction=%q, want %q and a reference prompt", req.OpenRouterRole, req.Instruction, roleReference)
		}
	}
	if len(backend.rasterized) != 0 {
		t.Fatalf("rasterized %d times, want the stored raster reused", len(backend.rasterized))
	}
	if !strings.Contains(got[0].Note, "Aquila") {
		t.Fatalf("note = %q, want the reference recorded", got[0].Note)
	}
}

func TestExploreComposesTheMarkWhenTheReferenceHasNoRasterAncestor(t *testing.T) {
	backend := &fakeBackend{}
	svc, store, assetStore := newTestService(backend, testBrands())
	ctx := context.Background()
	assetStore.put("aquila-mark", "image/svg+xml", markSVG)
	_, _ = store.Create(ctx, Candidate{BrandID: "b-aquila", AssetID: "aquila-mark", MediaType: "image/svg+xml", Status: StatusPicked})

	if _, _, err := svc.Explore(ctx, ExploreInput{BrandID: "b-vega", Concepts: []string{"lyre"}, StyleReferenceBrand: "b-aquila"}); err != nil {
		t.Fatal(err)
	}
	if len(backend.rasterized) != 1 || !strings.Contains(string(backend.rasterized[0]), "#15243c") {
		t.Fatalf("rasterized = %q, want the mark composed in the reference brand's container", backend.rasterized)
	}
	if string(backend.edits[0].Source) != "\x89PNG\r\n\x1a\nrasterized" {
		t.Fatalf("edit source = %q, want the rasterized composition", backend.edits[0].Source)
	}
}

func TestExploreRejectsAnUnknownStyleReferenceBeforeGenerating(t *testing.T) {
	backend := &fakeBackend{}
	svc, store, _ := newTestService(backend, testBrands())

	_, _, err := svc.Explore(context.Background(), ExploreInput{BrandID: "b-vega", Concepts: []string{"lyre"}, StyleReferenceBrand: "orion"})
	if err == nil || !strings.Contains(err.Error(), "orion") {
		t.Fatalf("err = %v, want an error naming the reference", err)
	}
	if len(backend.generates)+len(backend.edits) != 0 || store.count() != 0 {
		t.Fatal("a bad reference must fail before any generation is paid for")
	}
}

func TestExplorePreferVectorUsesTheVectorRole(t *testing.T) {
	backend := &fakeBackend{}
	svc, _, _ := newTestService(backend, testBrands())
	if _, _, err := svc.Explore(context.Background(), ExploreInput{BrandID: "b-vega", Concepts: []string{"lyre"}, PreferVector: true}); err != nil {
		t.Fatal(err)
	}
	if backend.generates[0].OpenRouterRole != roleVector {
		t.Fatalf("role = %q, want %q", backend.generates[0].OpenRouterRole, roleVector)
	}
}

func TestExploreWarnsWhenAConceptRendersLocally(t *testing.T) {
	backend := &fakeBackend{tier: "local-gpu"}
	svc, _, _ := newTestService(backend, testBrands())
	_, warnings, err := svc.Explore(context.Background(), ExploreInput{BrandID: "b-vega", Concepts: []string{"lyre"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "local model") {
		t.Fatalf("warnings = %q, want one local-model warning", warnings)
	}
}

func TestImportIsIdempotentPerAsset(t *testing.T) {
	svc, store, assetStore := newTestService(&fakeBackend{}, testBrands())
	assetStore.put("a1", "image/png", pngBytes)
	first, err := svc.Import(context.Background(), "b-vega", "a1", "lyre", "", "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.Import(context.Background(), "b-vega", "a1", "lyre again", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID || store.count() != 1 {
		t.Fatalf("ids %s/%s, rows %d: re-importing an asset must return the existing candidate", first.ID, second.ID, store.count())
	}
}

func TestPickVectorizesWithTheBrandsAccent(t *testing.T) {
	backend := &fakeBackend{}
	brands := testBrands()
	vega := brands["vega"]
	vega.Style.AccentColor = "#9ec5ff"
	brands["vega"] = vega
	svc, store, assetStore := newTestService(backend, brands)
	assetStore.put("raster", "image/png", pngBytes)
	cand, _ := store.Create(context.Background(), Candidate{BrandID: "b-vega", AssetID: "raster", MediaType: "image/png", Status: StatusProposed})

	if _, _, _, err := svc.Pick(context.Background(), cand.ID); err != nil {
		t.Fatal(err)
	}
	keep := append([]string(nil), backend.vectorized[0].KeepColors...)
	sort.Strings(keep)
	if strings.Join(keep, ",") != "#9ec5ff,#ffffff" {
		t.Fatalf("keep colours = %v, want white and the brand accent", keep)
	}
}

// image-tools' vectorize defaults (4 colours, min-area 40, tolerance 0.8) traced
// Rigel's star field as 68 contours, 57 of them specks under 20px, and dropped
// the thin constellation lines. A picked mark must trace with the settings that
// keep it clean, or every product in a line gets a different quality of mark.
func TestPickVectorizesWithTraceSafeDefaults(t *testing.T) {
	backend := &fakeBackend{}
	svc, store, assetStore := newTestService(backend, testBrands())
	assetStore.put("raster", "image/png", pngBytes)
	cand, _ := store.Create(context.Background(), Candidate{BrandID: "b-vega", AssetID: "raster", MediaType: "image/png", Status: StatusProposed})

	if _, _, _, err := svc.Pick(context.Background(), cand.ID); err != nil {
		t.Fatal(err)
	}
	got := backend.vectorized[0]
	if got.Colors != 3 {
		t.Fatalf("colours = %d, want 3", got.Colors)
	}
	if got.MinAreaPx != 60 {
		t.Fatalf("min area = %v px, want 60", got.MinAreaPx)
	}
	if got.TolerancePx != 0.25 {
		t.Fatalf("tolerance = %v px, want 0.25", got.TolerancePx)
	}
	if !got.DropBackgroundLayers || !got.ClipToLargestRoundedRegion || !got.Smoothing {
		t.Fatalf("drop=%t clip=%t smoothing=%t, want all true",
			got.DropBackgroundLayers, got.ClipToLargestRoundedRegion, got.Smoothing)
	}
}

func TestRefineInstructionUsesTheCloudEditRole(t *testing.T) {
	backend := &fakeBackend{}
	svc, store, assetStore := newTestService(backend, testBrands())
	assetStore.put("svg", "image/svg+xml", markSVG)
	cand, _ := store.Create(context.Background(), Candidate{BrandID: "b-vega", AssetID: "svg", MediaType: "image/svg+xml", Status: StatusProposed})

	if _, err := svc.Refine(context.Background(), cand.ID, RefineAction{Instruction: "thicker lines"}); err != nil {
		t.Fatal(err)
	}
	req := backend.edits[0]
	if req.OpenRouterRole != roleReference || !req.AllowBYOK || req.QualityPolicy != "quality" {
		t.Fatalf("edit = role:%q byok:%t quality:%q, want the cloud edit role with brand defaults", req.OpenRouterRole, req.AllowBYOK, req.QualityPolicy)
	}
	if len(backend.rasterized) != 1 {
		t.Fatal("an SVG candidate must be rasterized before an instruction edit")
	}
}
