package contextcapture

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
)

func TestRenderSelectedRegionPreservesOriginalAndClipsSourceAnnotations(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(Schema)))
	input, now := captureFixture(t)
	source := image.NewRGBA(image.Rect(0, 0, 8, 6))
	for y := 0; y < 6; y++ {
		for x := 0; x < 8; x++ {
			source.SetRGBA(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 100, A: 255})
		}
	}
	var encoded bytes.Buffer
	require.NoError(t, png.Encode(&encoded, source))
	input.PNG = encoded.Bytes()
	input.Source.Bounds = Bounds{X: -100, Y: -50, Width: 8, Height: 6}
	input.Region = Region{X: 2, Y: 1, Width: 4, Height: 3}
	// The first line crosses the crop. The second is wholly outside it.
	input.Strokes = [][]Point{{{X: 0, Y: 3}, {X: 8, Y: 3}}, {{X: 0, Y: 5}, {X: 8, Y: 5}}}
	svc := NewService(NewSQLiteRepository(db), blobstore.NewMemoryBlobStore(), func() time.Time { return now })
	doc, err := svc.Import(ctx, "alice", input, time.Hour)
	require.NoError(t, err)
	_, before, err := svc.Read(ctx, "alice", doc.ID)
	require.NoError(t, err)
	rendered, pixels, digest, err := svc.Render(ctx, "alice", doc.ID)
	require.NoError(t, err)
	require.Equal(t, doc, rendered)
	require.Len(t, digest, 64)
	result, err := png.Decode(bytes.NewReader(pixels))
	require.NoError(t, err)
	require.Equal(t, image.Rect(0, 0, 4, 3), result.Bounds())
	for x := 0; x < 4; x++ {
		require.Equal(t, color.RGBA{R: uint8(x + 2), G: 1, B: 100, A: 255}, color.RGBAModel.Convert(result.At(x, 0)))
		require.Equal(t, color.RGBA{R: 220, G: 38, B: 38, A: 255}, color.RGBAModel.Convert(result.At(x, 1)))
	}
	_, after, err := svc.Read(ctx, "alice", doc.ID)
	require.NoError(t, err)
	require.Equal(t, before, after)
	_, again, againDigest, err := svc.Render(ctx, "alice", doc.ID)
	require.NoError(t, err)
	require.Equal(t, pixels, again)
	require.Equal(t, digest, againDigest)
	_, pixels, _, err = svc.Render(ctx, "bob", doc.ID)
	require.ErrorIs(t, err, ErrUnavailable)
	require.Nil(t, pixels)
	now = doc.ExpiresAt
	_, pixels, _, err = svc.Render(ctx, "alice", doc.ID)
	require.ErrorIs(t, err, ErrUnavailable)
	require.Nil(t, pixels)
}

func TestRenderRejectsInvalidRegionAndHonorsCancellation(t *testing.T) {
	input, now := captureFixture(t)
	doc, pixels, err := Prepare("alice", input, now, time.Hour)
	require.NoError(t, err)
	doc.Region.Width = doc.Source.Bounds.Width + 1
	_, err = renderImage(context.Background(), doc, pixels)
	require.ErrorIs(t, err, ErrInvalid)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = renderImage(ctx, doc, pixels)
	require.ErrorIs(t, err, context.Canceled)
}
