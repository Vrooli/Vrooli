package modelstray

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"image-tools/internal/fetch"
)

type world struct {
	t       *testing.T
	home    string
	root    string
	entries []Entry
	adopted []string
	now     time.Time
}

func newWorld(t *testing.T) *world {
	t.Helper()
	base := t.TempDir()
	w := &world{t: t, home: filepath.Join(base, "home"), root: filepath.Join(base, "managed"), now: time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)}
	for _, d := range []string{w.home, filepath.Join(w.root, "models"), filepath.Join(w.root, "adapters"), filepath.Join(w.root, "staging")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return w
}

func (w *world) write(rel string, size int, age time.Duration) string {
	w.t.Helper()
	p := rel
	if !filepath.IsAbs(p) {
		p = filepath.Join(w.home, rel)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		w.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(strings.Repeat("x", size)), 0o644); err != nil {
		w.t.Fatal(err)
	}
	when := w.now.Add(-age)
	if err := os.Chtimes(p, when, when); err != nil {
		w.t.Fatal(err)
	}
	return p
}

func (w *world) scanner() *Scanner {
	s, err := New(Config{Root: w.root, Home: w.home, Now: func() time.Time { return w.now },
		Entries: func(context.Context) ([]Entry, error) { return w.entries, nil },
		Adopt: func(_ context.Context, kind, id string) error {
			w.adopted = append(w.adopted, kind+":"+id)
			return nil
		}})
	if err != nil {
		w.t.Fatal(err)
	}
	return s
}

func (w *world) find(report Report, suffix string) Finding {
	w.t.Helper()
	for _, f := range report.Findings {
		if strings.HasSuffix(f.Path, suffix) {
			return f
		}
	}
	w.t.Fatalf("no finding ending in %s: %+v", suffix, report.Findings)
	return Finding{}
}

func sd15() Entry {
	return Entry{Kind: "model", ID: "sd-1.5", Source: fetch.Source{
		Assets:        []fetch.Asset{{URL: "https://huggingface.co/stable-diffusion-v1-5/stable-diffusion-v1-5/resolve/abc/v1-5-pruned-emaonly.safetensors", Filename: "v1-5-pruned-emaonly.safetensors"}},
		HubCompanions: []fetch.RepoSpec{{RepoID: "stable-diffusion-v1-5/stable-diffusion-v1-5", Revision: strings.Repeat("a", 40), AllowPatterns: []string{"*.json"}}},
	}}
}

func TestScanFindsCatalogBytesInDefaultCachesAndLeavesOtherSoftwareAlone(t *testing.T) {
	w := newWorld(t)
	w.entries = []Entry{sd15(), {Kind: "model", ID: "u2net-full", Source: fetch.Source{Assets: []fetch.Asset{{URL: "https://github.com/danielgatis/rembg/releases/download/v0.0.0/u2net.onnx", Filename: "u2net.onnx"}}}}}
	hf := w.write(".cache/huggingface/hub/models--stable-diffusion-v1-5--stable-diffusion-v1-5/snapshots/abc/model_index.json", 2048, time.Hour)
	other := w.write(".cache/huggingface/hub/models--ACE-Step--acestep-v15-sft/snapshots/s/config.json", 4096, time.Hour)
	u2 := w.write(".u2net/u2net.onnx", 5000, time.Hour)
	ollama := w.write(".ollama/models/blobs/sha256-1", 9000, time.Hour)

	report, err := w.scanner().Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	f := w.find(report, "models--stable-diffusion-v1-5--stable-diffusion-v1-5")
	if f.Action != ActionAdopt || f.Owner != OwnerCatalog || f.Bytes != 2048 || f.CatalogIDs[0] != "model:sd-1.5" {
		t.Fatalf("catalog hub repo: %+v", f)
	}
	if want := filepath.Join(w.root, "cache", "huggingface", "hub", "models--stable-diffusion-v1-5--stable-diffusion-v1-5"); f.Destination != want {
		t.Fatalf("destination %q, want %q", f.Destination, want)
	}
	if g := w.find(report, "models--ACE-Step--acestep-v15-sft"); g.Action != ActionReportOnly || g.Owner != OwnerUnknown {
		t.Fatalf("another scenario's hub model must be report-only: %+v", g)
	}
	if g := w.find(report, "u2net.onnx"); g.Action != ActionAdopt || g.Destination != filepath.Join(w.root, "cache", "u2net", "u2net.onnx") {
		t.Fatalf("u2net: %+v", g)
	}
	if g := w.find(report, "/blobs"); g.Action != ActionReportOnly || g.Owner != OwnerOther {
		t.Fatalf("ollama must be report-only: %+v", g)
	}
	if report.Totals.AdoptableBytes != 2048+5000 || report.Totals.ReportOnlyBytes != 4096+9000 {
		t.Fatalf("totals %+v", report.Totals)
	}
	for _, p := range []string{hf, other, u2, ollama} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("a scan must not touch %s: %v", p, err)
		}
	}
	var sawDefault bool
	for _, sp := range report.Scanned {
		if strings.HasSuffix(sp.Path, ".insightface") && !sp.Exists {
			sawDefault = true
		}
	}
	if !sawDefault {
		t.Fatalf("the report must list the locations it looked at, including empty ones: %+v", report.Scanned)
	}
}

func TestAdoptMovesIntoTheManagedCacheAndWritesAReceipt(t *testing.T) {
	w := newWorld(t)
	w.entries = []Entry{sd15()}
	repo := ".cache/huggingface/hub/models--stable-diffusion-v1-5--stable-diffusion-v1-5"
	w.write(repo+"/blobs/b1", 100, time.Hour)
	if err := os.MkdirAll(filepath.Join(w.home, repo, "snapshots", "abc"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../blobs/b1", filepath.Join(w.home, repo, "snapshots", "abc", "config.json")); err != nil {
		t.Fatal(err)
	}
	s := w.scanner()
	report, _ := s.Scan(context.Background())
	f := w.find(report, "models--stable-diffusion-v1-5--stable-diffusion-v1-5")
	receipt, err := s.Adopt(context.Background(), f.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(w.home, repo)); !os.IsNotExist(err) {
		t.Fatalf("source still present: %v", err)
	}
	moved := filepath.Join(w.root, "cache", "huggingface", "hub", "models--stable-diffusion-v1-5--stable-diffusion-v1-5")
	if target, err := os.Readlink(filepath.Join(moved, "snapshots", "abc", "config.json")); err != nil || target != "../../blobs/b1" {
		t.Fatalf("hub snapshot symlinks must survive the move: %q %v", target, err)
	}
	if receipt.Bytes != 100 || receipt.Destination != moved {
		t.Fatalf("receipt %+v", receipt)
	}
	log, err := os.ReadFile(ReceiptsPath(w.root))
	if err != nil || !strings.Contains(string(log), `"action":"adopt"`) || !strings.Contains(string(log), "model:sd-1.5") {
		t.Fatalf("adoption must be recorded durably: %s %v", log, err)
	}
	again, _ := s.Scan(context.Background())
	for _, f := range again.Findings {
		if f.Kind == KindDefaultCache {
			t.Fatalf("adopted bytes must stop being strays: %+v", f)
		}
	}
}

func TestDuplicateOfManagedCopyIsReclaimableAndOnlyThen(t *testing.T) {
	w := newWorld(t)
	w.entries = []Entry{sd15()}
	repo := "models--stable-diffusion-v1-5--stable-diffusion-v1-5/snapshots/abc/model_index.json"
	w.write(".cache/huggingface/hub/"+repo, 500, time.Hour)
	managed := w.write(filepath.Join(w.root, "cache", "huggingface", "hub", repo), 500, time.Hour)
	s := w.scanner()
	report, _ := s.Scan(context.Background())
	f := w.find(report, "models--stable-diffusion-v1-5--stable-diffusion-v1-5")
	if f.Action != ActionReclaim {
		t.Fatalf("a duplicate of a managed copy: %+v", f)
	}
	if _, err := s.Adopt(context.Background(), f.ID); err == nil {
		t.Fatal("adopt must refuse a reclaim-only finding rather than overwrite")
	}
	if _, err := s.Reclaim(context.Background(), f.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(managed); err != nil {
		t.Fatalf("the managed copy must survive: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(w.home, ".cache", "huggingface", "hub", "models--stable-diffusion-v1-5--stable-diffusion-v1-5")); !os.IsNotExist(err) {
		t.Fatalf("duplicate not reclaimed: %v", err)
	}
}

func TestReclaimRefusesReportOnlyBytes(t *testing.T) {
	w := newWorld(t)
	p := w.write(".ollama/models/blobs/sha256-1", 10, time.Hour)
	s := w.scanner()
	report, _ := s.Scan(context.Background())
	f := w.find(report, "/blobs")
	if _, err := s.Reclaim(context.Background(), f.ID); err == nil {
		t.Fatal("another application's bytes must never be reclaimed")
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatal(err)
	}
}

func TestManagedRootEntriesWithoutReceiptsAreReportedWithAgeAwareActions(t *testing.T) {
	w := newWorld(t)
	w.entries = []Entry{
		{Kind: "model", ID: "tracked", Installed: true, Path: filepath.Join(w.root, "models", "tracked"), Source: fetch.Source{}},
		{Kind: "model", ID: "known-no-receipt", Source: fetch.Source{}},
	}
	w.write(filepath.Join(w.root, "models", "tracked", "w.bin"), 100, 100*time.Hour)
	w.write(filepath.Join(w.root, "models", "known-no-receipt", "w.bin"), 200, 100*time.Hour)
	w.write(filepath.Join(w.root, "models", "old-orphan", "w.bin"), 300, 100*time.Hour)
	w.write(filepath.Join(w.root, "models", "fresh-orphan", "w.bin"), 400, time.Minute)
	w.write(filepath.Join(w.root, "cache", "huggingface", "hub", "x"), 50, time.Hour)
	w.write(filepath.Join(w.root, "mystery.bin"), 60, time.Hour)
	s := w.scanner()
	report, _ := s.Scan(context.Background())
	for _, f := range report.Findings {
		if strings.HasSuffix(f.Path, "/models/tracked") {
			t.Fatalf("a tracked install is not a stray: %+v", f)
		}
	}
	if f := w.find(report, "models/known-no-receipt"); f.Action != ActionAdopt || f.Owner != OwnerCatalog {
		t.Fatalf("known entry: %+v", f)
	}
	if f := w.find(report, "models/old-orphan"); f.Action != ActionReclaim || f.Bytes != 300 {
		t.Fatalf("old orphan: %+v", f)
	}
	if f := w.find(report, "models/fresh-orphan"); f.Action != ActionReportOnly {
		t.Fatalf("a young unknown entry may be an install in progress: %+v", f)
	}
	if f := w.find(report, "cache/huggingface"); f.Kind != KindManagedCache || f.Action != ActionNone {
		t.Fatalf("managed cache: %+v", f)
	}
	if f := w.find(report, "mystery.bin"); f.Action != ActionReportOnly {
		t.Fatalf("unknown top-level entry: %+v", f)
	}
	known := w.find(report, "models/known-no-receipt")
	if _, err := s.Adopt(context.Background(), known.ID); err != nil {
		t.Fatal(err)
	}
	if len(w.adopted) != 1 || w.adopted[0] != "model:known-no-receipt" {
		t.Fatalf("receipt adoption not delegated to the installer: %v", w.adopted)
	}
	if _, err := os.Stat(filepath.Join(w.root, "models", "known-no-receipt", "w.bin")); err != nil {
		t.Fatalf("an in-place receipt must not move bytes: %v", err)
	}
	old := w.find(report, "models/old-orphan")
	if _, err := s.Reclaim(context.Background(), old.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(w.root, "models", "old-orphan")); !os.IsNotExist(err) {
		t.Fatal("old orphan not reclaimed")
	}
}

func TestExternalReferencesAreReportedNotMoved(t *testing.T) {
	w := newWorld(t)
	outside := w.write(filepath.Join(filepath.Dir(w.root), "elsewhere", "my.safetensors"), 700, time.Hour)
	w.entries = []Entry{{Kind: "model", ID: "local", Installed: true, Path: outside}}
	report, _ := w.scanner().Scan(context.Background())
	f := w.find(report, "my.safetensors")
	if f.Kind != KindExternalReference || f.Action != ActionReportOnly || f.Bytes != 700 {
		t.Fatalf("%+v", f)
	}
}

func TestMoveTreeCopiesVerifiesAndRefusesToOverwrite(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	_ = os.MkdirAll(filepath.Join(src, "a"), 0o755)
	_ = os.WriteFile(filepath.Join(src, "a", "f"), []byte("hello"), 0o644)
	dst := filepath.Join(dir, "dst")
	if err := copyTree(src, dst); err != nil {
		t.Fatal(err)
	}
	if got, want := measure(dst), measure(src); got.bytes != want.bytes || got.files != want.files {
		t.Fatalf("copy %v want %v", got, want)
	}
	if err := moveTree(src, dst); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("overwrite must be refused: %v", err)
	}
}

// Catalog entries that ship a config.json or model.safetensors must not claim
// every other repository in the Hugging Face cache that has files with those
// names: a Hub cache directory is owned only by the entry that names its repo.
func TestGenericFileNamesNeverClaimAnotherRepositoryOrAdoptItsBytes(t *testing.T) {
	w := newWorld(t)
	w.entries = []Entry{{Kind: "model", ID: "segformer", Source: fetch.Source{Assets: []fetch.Asset{
		{URL: "https://huggingface.co/mattmdjaga/segformer_b2_clothes/resolve/abc/config.json", Filename: "config.json"},
		{URL: "https://huggingface.co/mattmdjaga/segformer_b2_clothes/resolve/abc/model.safetensors", Filename: "model.safetensors"},
	}}}}
	pii := w.write(".cache/huggingface/hub/models--yonigo--distilbert-base-cased-pii-en/snapshots/s/model.safetensors", 3000, time.Hour)
	w.write(".cache/huggingface/hub/models--yonigo--distilbert-base-cased-pii-en/snapshots/s/config.json", 100, time.Hour)
	mine := w.write(".cache/huggingface/hub/models--mattmdjaga--segformer_b2_clothes/snapshots/s/config.json", 100, time.Hour)

	report, err := w.scanner().Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if f := w.find(report, "models--yonigo--distilbert-base-cased-pii-en"); f.Action != ActionReportOnly || len(f.CatalogIDs) != 0 {
		t.Fatalf("another tool's repository was claimed by file name: %+v", f)
	}
	if f := w.find(report, "models--mattmdjaga--segformer_b2_clothes"); f.Action != ActionAdopt || f.CatalogIDs[0] != "model:segformer" {
		t.Fatalf("the repository the entry names must still be adoptable: %+v", f)
	}
	for _, p := range []string{pii, mine} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("scan touched %s: %v", p, err)
		}
	}
}
