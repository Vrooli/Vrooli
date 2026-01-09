package storage

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileStorage_StoreAndGet(t *testing.T) {
	root := t.TempDir()
	store, err := NewFileStorage(root, nil)
	require.NoError(t, err)

	execID := uuid.New()
	data := mustPNG(t)
	info, err := store.StoreScreenshot(context.Background(), execID, "click", data, "image/png")
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, int64(len(data)), info.SizeBytes)
	assert.Equal(t, 1, info.Width)
	assert.Equal(t, 1, info.Height)

	objectName := info.URL[len("/api/v1/screenshots/"):]
	reader, meta, err := store.GetScreenshot(context.Background(), objectName)
	require.NoError(t, err)
	defer reader.Close()

	readData, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, data, readData)
	assert.Equal(t, int64(len(data)), meta.Size)
	assert.Equal(t, "image/png", meta.ContentType)
}

func TestFileStorage_DeleteAndList(t *testing.T) {
	root := t.TempDir()
	store, err := NewFileStorage(root, nil)
	require.NoError(t, err)

	execID := uuid.New()
	data := mustPNG(t)

	info1, err := store.StoreScreenshot(context.Background(), execID, "step1", data, "image/png")
	require.NoError(t, err)
	_, err = store.StoreScreenshot(context.Background(), execID, "step2", data, "image/png")
	require.NoError(t, err)

	objectName := info1.URL[len("/api/v1/screenshots/"):]

	list, err := store.ListExecutionScreenshots(context.Background(), execID)
	require.NoError(t, err)
	assert.Len(t, list, 2)

	require.NoError(t, store.DeleteScreenshot(context.Background(), objectName))
	list, err = store.ListExecutionScreenshots(context.Background(), execID)
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestFileStorage_HealthCheck(t *testing.T) {
	root := t.TempDir()
	store, err := NewFileStorage(root, nil)
	require.NoError(t, err)

	assert.NoError(t, store.HealthCheck(context.Background()))
}

func TestFileStorage_PathTraversalGuard(t *testing.T) {
	root := t.TempDir()
	store, err := NewFileStorage(root, nil)
	require.NoError(t, err)

	_, _, err = store.GetScreenshot(context.Background(), "../evil")
	assert.Error(t, err)
}

func mustPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode png: %v", err)
	}
	return buf.Bytes()
}
