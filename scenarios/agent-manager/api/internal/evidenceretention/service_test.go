package evidenceretention

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var now = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

func write(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func elf(size int) []byte {
	b := make([]byte, size)
	copy(b, elfMagic)
	return b
}

// fixture builds one effort's evidence with every class next to real
// acceptance evidence, plus a work-record tree citing some of it.
func fixture(t *testing.T, effort string) (root, records string) {
	t.Helper()
	base := t.TempDir()
	root = filepath.Join(base, "evidence")
	records = filepath.Join(base, "repo")
	e := filepath.Join(root, effort)
	write(t, filepath.Join(e, "E7/release/tool_blender-renderer_linux_amd64/blender"), elf(2<<20))
	write(t, filepath.Join(e, "E3/source/blender-5.2.2-linux-x64.tar.xz"), []byte("archive"))
	write(t, filepath.Join(e, "E1/stage/node_modules/x/index.js"), []byte("x"))
	write(t, filepath.Join(e, "E3/demo/files/data/renderjobs/a/worker"), elf(2<<20))
	write(t, filepath.Join(e, "E3/demo/final.mp4"), []byte("video"))
	write(t, filepath.Join(e, "E3/demo/frames/0001.png"), []byte("f1"))
	write(t, filepath.Join(e, "E3/demo/frames/0002.png"), []byte("f2"))
	// Frames with no encoded output are evidence, not intermediates.
	write(t, filepath.Join(e, "E4/only/frames/0001.png"), []byte("f1"))
	write(t, filepath.Join(e, "E4/only/frames/0002.png"), []byte("f2"))
	// Cited frames stay even with a video beside them.
	write(t, filepath.Join(e, "E5/cited/final.mp4"), []byte("video"))
	write(t, filepath.Join(e, "E5/cited/source-frames/0001.png"), []byte("f1"))
	write(t, filepath.Join(e, "E5/cited/source-frames/0002.png"), []byte("f2"))
	// Packaged deliverables and small binaries are never build_binary.
	write(t, filepath.Join(e, "build/App-1.0.0.AppImage"), elf(2<<20))
	write(t, filepath.Join(e, "E1/screenshot.png"), []byte("shot"))
	write(t, filepath.Join(e, "E1/report.json"), []byte(`{"ok":true}`))
	write(t, filepath.Join(records, ".vrooli/work/delivery/x/LOG.md"), []byte(
		"Accepted E1: ~/.vrooli/evidence/"+effort+"/E1/report.json and whole checkpoint /home/u/.vrooli/evidence/"+effort+"/E7.\n"+
			"Frame proof: `/home/u/.vrooli/evidence/"+effort+"/E5/cited/source-frames/0002.png`.\n"))
	return root, records
}

func newService(t *testing.T, root, records string, efforts []Effort) *Service {
	t.Helper()
	svc, err := New(Options{
		Root: root, ReferenceRoots: []string{records}, ReceiptDir: filepath.Join(filepath.Dir(root), "receipts"),
		Efforts: func(context.Context) ([]Effort, error) { return efforts, nil },
		Now:     func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func candidatesByPath(r EffortReport) map[string]Candidate {
	out := map[string]Candidate{}
	for _, c := range r.Candidates {
		out[c.Path] = c
	}
	return out
}

func TestScanClassifiesRegenerableIntermediatesAndProtectsCitations(t *testing.T) {
	root, records := fixture(t, "demo-delivery-2026-10-20")
	report, err := newService(t, root, records, nil).Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Efforts) != 1 {
		t.Fatalf("efforts = %d", len(report.Efforts))
	}
	got := candidatesByPath(report.Efforts[0])
	p := "demo-delivery-2026-10-20/"
	want := map[string]Class{
		p + "E7/release/tool_blender-renderer_linux_amd64": ClassPlatformRelease,
		p + "E3/source/blender-5.2.2-linux-x64.tar.xz":     ClassReleaseArchive,
		p + "E1/stage/node_modules":                        ClassDependencyTree,
		p + "E3/demo/files/data/renderjobs/a/worker":       ClassBuildBinary,
		p + "E3/demo/frames":                               ClassRenderFrames,
		p + "E5/cited/source-frames":                       ClassRenderFrames,
	}
	for path, class := range want {
		if got[path].Class != class {
			t.Errorf("%s: class %q, want %q", path, got[path].Class, class)
		}
	}
	if len(got) != len(want) {
		t.Errorf("unexpected candidates: %v", got)
	}
	// A citation inside a candidate protects it; a citation of an ancestor
	// directory (the whole E7 checkpoint) does not.
	if !got[p+"E5/cited/source-frames"].Protected {
		t.Error("cited frames must be protected")
	}
	if got[p+"E7/release/tool_blender-renderer_linux_amd64"].Protected {
		t.Error("an ancestor citation must not protect a staged release")
	}
	if report.Efforts[0].Lifecycle.State != StateUnenrolled || report.Efforts[0].Lifecycle.AutoEligible {
		t.Errorf("lifecycle = %+v", report.Efforts[0].Lifecycle)
	}
	b := report.Efforts[0].Bytes
	if b.Regenerable+b.ProtectedRegenerable+b.Referenced+b.Unknown != b.Total || b.Referenced == 0 || b.Unknown == 0 {
		t.Errorf("byte breakdown does not add up: %+v", b)
	}
}

func TestAutomaticPruneOnlyTouchesTerminalEffortsPastGrace(t *testing.T) {
	root, records := fixture(t, "demo-delivery-2026-10-20")
	cases := []struct {
		name    string
		efforts []Effort
		removes bool
	}{
		{"unenrolled", nil, false},
		{"active", []Effort{{Ref: "effort:a", Keys: []string{"Demo delivery"}}}, false},
		{"inside grace", []Effort{{Ref: "effort:a", Keys: []string{"demo"}, Terminal: true, TerminalAt: now.Add(-24 * time.Hour)}}, false},
		{"one live match", []Effort{
			{Ref: "effort:a", Keys: []string{"demo"}, Terminal: true, TerminalAt: now.Add(-30 * 24 * time.Hour)},
			{Ref: "effort:b", Keys: []string{".vrooli/work/delivery/demo-delivery"}},
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			receipt, err := newService(t, root, records, tc.efforts).Prune(context.Background(), PruneRequest{})
			if err != nil {
				t.Fatal(err)
			}
			if len(receipt.Removed) != 0 {
				t.Fatalf("removed %v", receipt.Removed)
			}
		})
	}
	terminal := []Effort{{Ref: "effort:a", Keys: []string{".vrooli/work/delivery/demo-delivery"}, Terminal: true, TerminalAt: now.Add(-8 * 24 * time.Hour)}}
	receipt, err := newService(t, root, records, terminal).Prune(context.Background(), PruneRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(receipt.Removed) != 5 || receipt.RemovedBytes == 0 || receipt.ReceiptPath == "" {
		t.Fatalf("receipt = %+v", receipt)
	}
	e := filepath.Join(root, "demo-delivery-2026-10-20")
	for _, gone := range []string{"E7/release/tool_blender-renderer_linux_amd64", "E3/demo/frames", "E1/stage/node_modules", "E3/demo/files/data/renderjobs/a/worker", "E3/source/blender-5.2.2-linux-x64.tar.xz"} {
		if _, err := os.Stat(filepath.Join(e, gone)); !os.IsNotExist(err) {
			t.Errorf("%s still present", gone)
		}
	}
	for _, kept := range []string{"E3/demo/final.mp4", "E4/only/frames/0001.png", "E5/cited/source-frames/0002.png", "build/App-1.0.0.AppImage", "E1/screenshot.png", "E1/report.json"} {
		if _, err := os.Stat(filepath.Join(e, kept)); err != nil {
			t.Errorf("evidence %s removed: %v", kept, err)
		}
	}
	var persisted Receipt
	data, err := os.ReadFile(receipt.ReceiptPath)
	if err != nil || json.Unmarshal(data, &persisted) != nil || len(persisted.Removed) != 5 {
		t.Fatalf("persisted receipt unreadable or incomplete: %v %s", err, data)
	}
}

func TestOperatorPruneNamesEffortAndHonoursClassesAndDryRun(t *testing.T) {
	root, records := fixture(t, "demo-2026-10-20")
	svc := newService(t, root, records, nil)
	dry, err := svc.Prune(context.Background(), PruneRequest{Efforts: []string{"demo-2026-10-20", "missing"}, Classes: []Class{ClassBuildBinary}, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if dry.Mode != "operator" || len(dry.Removed) != 1 || dry.Removed[0].Class != ClassBuildBinary || dry.ReceiptPath != "" {
		t.Fatalf("dry run = %+v", dry)
	}
	if _, err := os.Stat(filepath.Join(root, dry.Removed[0].Path)); err != nil {
		t.Fatal("dry run removed a file")
	}
	if len(dry.Skipped) != 1 || dry.Skipped[0].Effort != "missing" {
		t.Errorf("skipped = %+v", dry.Skipped)
	}
	if _, err := svc.Prune(context.Background(), PruneRequest{Efforts: []string{"../etc"}}); err == nil {
		t.Fatal("path-like effort name accepted")
	}
	if _, err := svc.Prune(context.Background(), PruneRequest{Classes: []Class{"screenshots"}}); err == nil {
		t.Fatal("unknown class accepted")
	}
	// A live enrollment refuses even an operator-named prune.
	live := newService(t, root, records, []Effort{{Ref: "effort:x", Keys: []string{"demo"}}})
	r, err := live.Prune(context.Background(), PruneRequest{Efforts: []string{"demo-2026-10-20"}})
	if err != nil || len(r.Removed) != 0 || len(r.Skipped) != 1 {
		t.Fatalf("live prune = %+v %v", r, err)
	}
}

func TestPruneHandlerRequiresOwnerAndReason(t *testing.T) {
	root, records := fixture(t, "demo-2026-10-20")
	svc := newService(t, root, records, nil)
	deny := NewHandler(svc, func(*http.Request) (string, int, error) { return "", http.StatusUnauthorized, os.ErrPermission }, nil)
	rec := httptest.NewRecorder()
	deny.Prune(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"efforts":["demo-2026-10-20"],"reason":"r"}`)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized prune status = %d", rec.Code)
	}
	allow := NewHandler(svc, func(*http.Request) (string, int, error) { return "owner", 0, nil }, nil)
	rec = httptest.NewRecorder()
	allow.Prune(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"efforts":["demo-2026-10-20"]}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("reasonless prune status = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	allow.Prune(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"efforts":["demo-2026-10-20"],"classes":["release_archive"],"reason":"effort finished"}`)))
	var receipt Receipt
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &receipt) != nil || receipt.Actor != "owner" || len(receipt.Removed) != 1 {
		t.Fatalf("authorized prune = %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	deny.Report(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("report status = %d", rec.Code)
	}
}

func TestNormalizeKeyMatchesDeliveryHomesAndDatedDirectories(t *testing.T) {
	for raw, want := range map[string]string{
		".vrooli/work/delivery/desktop-packaging-delivery": "desktop-packaging",
		"desktop-packaging":                     "desktop-packaging",
		"device-presentation-2026-10-18":        "device-presentation",
		"rcl-prompt-manager-ux-design-campaign": "rcl-prompt-manager-ux-design",
		"Native-feel desktop packaging":         "native-feel-desktop-packaging",
		"vega-launch-video-retry-2026-09-22":    "vega-launch-video-retry",
	} {
		if got := normalizeKey(raw); got != want {
			t.Errorf("normalizeKey(%q) = %q, want %q", raw, got, want)
		}
	}
}
