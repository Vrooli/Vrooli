package export

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/storage"
)

func TestWriteHTMLBundle_IncludesAssetsAndIndex(t *testing.T) {
	ctx := context.Background()
	store := storage.NewMemoryStorage()
	executionID := uuid.New()
	workflowID := uuid.New()
	screenshotBytes := []byte("fake-png-bytes")

	if _, err := store.StoreScreenshot(ctx, executionID, "shot-1", screenshotBytes, "image/png"); err != nil {
		t.Fatalf("failed to store screenshot: %v", err)
	}

	objectNames, err := store.ListExecutionScreenshots(ctx, executionID)
	if err != nil {
		t.Fatalf("failed to list screenshots: %v", err)
	}
	if len(objectNames) != 1 {
		t.Fatalf("expected one stored screenshot, got %d", len(objectNames))
	}

	spec := &ReplayMovieSpec{
		Execution: ExportExecutionMetadata{
			ExecutionID:  executionID,
			WorkflowID:   workflowID,
			WorkflowName: "Demo Flow",
			Status:       "completed",
		},
		Theme: ExportTheme{
			BackgroundGradient: []string{"#111111", "#222222"},
			AccentColor:        "#33c3f0",
		},
		Presentation: ExportPresentation{
			Viewport: ExportDimensions{Width: 1280, Height: 720},
		},
		Assets: []ExportAsset{
			{
				ID:     "shot-1",
				Type:   "screenshot",
				Source: "/api/v1/screenshots/" + objectNames[0],
			},
		},
		Frames: []ExportFrame{
			{
				Index:             0,
				StepIndex:         0,
				NodeID:            "step-1",
				StepType:          "click",
				Title:             "Click button",
				Status:            "success",
				DurationMs:        1200,
				Viewport:          ExportDimensions{Width: 1280, Height: 720},
				ScreenshotAssetID: "shot-1",
			},
		},
	}

	var buf bytes.Buffer
	if err := WriteHTMLBundle(ctx, &buf, spec, store, nil, ""); err != nil {
		t.Fatalf("failed to write HTML bundle: %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("failed to read zip bundle: %v", err)
	}

	files := make(map[string][]byte, len(reader.File))
	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("failed to open zip file %s: %v", file.Name, err)
		}
		content, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatalf("failed to read zip file %s: %v", file.Name, err)
		}
		files[file.Name] = content
	}

	indexHTML, ok := files["index.html"]
	if !ok {
		t.Fatalf("index.html missing from HTML bundle")
	}
	if !strings.Contains(string(indexHTML), "Demo Flow") {
		t.Fatalf("expected workflow name in index.html payload")
	}

	readme, ok := files["README.txt"]
	if !ok || len(readme) == 0 {
		t.Fatalf("README.txt missing from HTML bundle")
	}

	assetPath := "assets/shot-1.png"
	assetBytes, ok := files[assetPath]
	if !ok {
		t.Fatalf("expected asset %s to be included in bundle", assetPath)
	}
	if !bytes.Equal(assetBytes, screenshotBytes) {
		t.Fatalf("expected asset bytes to match stored screenshot")
	}
}
