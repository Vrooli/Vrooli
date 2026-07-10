package runs

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"connectrpc.com/connect"

	sharedartifacts "test-genie/internal/shared/artifacts"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	visualpb "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth"
)

// writePageScreenshot renders a page screenshot artifact plus page metadata so
// ListRunVisuals reports the route.
func writePageScreenshot(t *testing.T, root, runID, page string, img image.Image) {
	t.Helper()
	pagesDir := sharedartifacts.RunUISmokePagesDir(filepath.Join(root, "demo"), runID)
	dir := filepath.Join(pagesDir, sanitizePage(page))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "page.json"), []byte(`{"page":"`+page+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "screenshot.png"), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func sanitizePage(page string) string {
	if page == "/" {
		return "_root_"
	}
	out := make([]rune, 0, len(page))
	for _, r := range page {
		if r == '/' {
			r = '_'
		}
		out = append(out, r)
	}
	return string(out)
}

func gradient(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := uint8(x * 255 / w)
			img.Set(x, y, color.RGBA{v, v, v, 255})
		}
	}
	return img
}

func solid(w, h int, c color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func TestCompareRunVisuals(t *testing.T) {
	svc, root := newTestService(t)
	svc.SetVisualHealthComparer(fakeVisualHealthComparer{})

	// Baseline run: two identical-content pages.
	writePageScreenshot(t, root, "base", "/", gradient(120, 90))
	writePageScreenshot(t, root, "base", "/dashboard", gradient(120, 90))
	// Current run: "/" unchanged, "/dashboard" repainted solid (changed),
	// "/new" added, "/dashboard"... and the baseline-only page handled below.
	writePageScreenshot(t, root, "cur", "/", gradient(120, 90))
	writePageScreenshot(t, root, "cur", "/dashboard", solid(120, 90, color.RGBA{200, 30, 30, 255}))
	writePageScreenshot(t, root, "cur", "/new", gradient(120, 90))

	resp, err := svc.CompareRunVisuals(context.Background(), connect.NewRequest(&runspb.CompareRunVisualsRequest{
		Scenario: "demo", BaseRunId: "base", CurrentRunId: "cur",
	}))
	if err != nil {
		t.Fatalf("CompareRunVisuals: %v", err)
	}
	byPage := map[string]*runspb.VisualDelta{}
	for _, d := range resp.Msg.GetDeltas() {
		byPage[d.GetPage()] = d
	}

	if got := byPage["/"]; got == nil || got.GetStatus() != "identical" {
		t.Errorf("/ status = %v, want identical", got)
	}
	if got := byPage["/dashboard"]; got == nil || got.GetStatus() != "changed" || got.GetChangedFraction() <= 0 {
		t.Errorf("/dashboard = %+v, want changed with magnitude", got)
	} else if got.GetBaseArtifactId() == "" || got.GetCurrentArtifactId() == "" || got.GetBaseArtifactId() == got.GetCurrentArtifactId() {
		t.Errorf("/dashboard artifact links = %+v, want distinct run-scoped ids", got)
	}
	if got := byPage["/new"]; got == nil || got.GetStatus() != "added" {
		t.Errorf("/new status = %v, want added", got)
	}
}

func TestCompareRunVisualsRemovedPage(t *testing.T) {
	svc, root := newTestService(t)
	svc.SetVisualHealthComparer(fakeVisualHealthComparer{})
	writePageScreenshot(t, root, "base", "/", gradient(80, 60))
	writePageScreenshot(t, root, "base", "/gone", gradient(80, 60))
	writePageScreenshot(t, root, "cur", "/", gradient(80, 60))

	resp, err := svc.CompareRunVisuals(context.Background(), connect.NewRequest(&runspb.CompareRunVisualsRequest{
		Scenario: "demo", BaseRunId: "base", CurrentRunId: "cur",
	}))
	if err != nil {
		t.Fatalf("CompareRunVisuals: %v", err)
	}
	for _, d := range resp.Msg.GetDeltas() {
		if d.GetPage() == "/gone" && d.GetStatus() != "removed" {
			t.Errorf("/gone status = %s, want removed", d.GetStatus())
		}
	}
}

func TestCompareRunVisualsRequiresRunIDs(t *testing.T) {
	svc, _ := newTestService(t)
	_, err := svc.CompareRunVisuals(context.Background(), connect.NewRequest(&runspb.CompareRunVisualsRequest{
		Scenario: "demo", BaseRunId: "", CurrentRunId: "cur",
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

type fakeVisualHealthComparer struct{}

func (fakeVisualHealthComparer) CompareArtifacts(_ context.Context, req *visualpb.CompareArtifactsRequest) (*visualpb.CompareArtifactsResponse, error) {
	base := map[string]*visualpb.CompareArtifact{}
	for _, art := range req.GetBase() {
		base[art.GetPage()] = art
	}
	cur := map[string]*visualpb.CompareArtifact{}
	for _, art := range req.GetCurrent() {
		cur[art.GetPage()] = art
	}
	seen := map[string]struct{}{}
	for page := range base {
		seen[page] = struct{}{}
	}
	for page := range cur {
		seen[page] = struct{}{}
	}
	pages := make([]string, 0, len(seen))
	for page := range seen {
		pages = append(pages, page)
	}
	sort.Strings(pages)
	resp := &visualpb.CompareArtifactsResponse{}
	for _, page := range pages {
		baseArt, inBase := base[page]
		curArt, inCur := cur[page]
		label := curArt.GetLabel()
		if !inCur {
			label = baseArt.GetLabel()
		}
		delta := &visualpb.VisualDelta{Page: page, Label: label}
		switch {
		case inBase && inCur && bytes.Equal(baseArt.GetScreenshotPng(), curArt.GetScreenshotPng()):
			delta.Status = "identical"
		case inBase && inCur:
			delta.Status = "changed"
			delta.ChangedFraction = 0.25
		case inCur:
			delta.Status = "added"
		default:
			delta.Status = "removed"
		}
		resp.Deltas = append(resp.Deltas, delta)
	}
	return resp, nil
}
