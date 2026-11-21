package designs

import (
	"bytes"
	"context"
	"encoding/binary"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func fixedNow() func() time.Time {
	return func() time.Time { return time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC) }
}

func pngHeader(width, height uint32) []byte {
	data := append([]byte{}, []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}...)
	data = append(data, 0, 0, 0, 13) // IHDR length
	data = append(data, "IHDR"...)
	var dimensions [8]byte
	binary.BigEndian.PutUint32(dimensions[0:4], width)
	binary.BigEndian.PutUint32(dimensions[4:8], height)
	return append(data, dimensions[:]...)
}

func gifHeader(width, height uint16) []byte {
	data := []byte("GIF89a")
	var dimensions [4]byte
	binary.LittleEndian.PutUint16(dimensions[0:2], width)
	binary.LittleEndian.PutUint16(dimensions[2:4], height)
	return append(data, dimensions[:]...)
}

func TestImageDimensionsReadsHeaders(t *testing.T) {
	cases := map[string]struct {
		mime   string
		data   []byte
		width  int
		height int
	}{
		"png": {mime: "image/png", data: pngHeader(640, 480), width: 640, height: 480},
		"gif": {mime: "image/gif", data: gifHeader(320, 200), width: 320, height: 200},
		"svg": {mime: "image/svg+xml", data: []byte(`<svg width="10" height="10"></svg>`)},
		"unknown": {
			mime: "image/webp",
			data: []byte("RIFF\x00\x00\x00\x00WEBPnope"),
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			width, height := imageDimensions(tc.mime, tc.data)
			require.Equal(t, tc.width, width)
			require.Equal(t, tc.height, height)
		})
	}
}

func TestCreateAssetStoresMetadataAndBytes(t *testing.T) {
	repo := newTestRepository(t)
	repo.now = fixedNow()
	design, err := repo.Create(context.Background(), CreateInput{Name: "Assets"})
	require.NoError(t, err)
	other, err := repo.Create(context.Background(), CreateInput{Name: "Other"})
	require.NoError(t, err)

	asset, err := repo.CreateAsset(context.Background(), AssetInput{
		DesignID: design.ID,
		Name:     "  hero.png ",
		Mime:     "IMAGE/PNG; charset=binary",
		Data:     pngHeader(1200, 800),
		Fidelity: FidelityCaptured,
	})
	require.NoError(t, err)
	require.Equal(t, design.ID, asset.DesignID)
	require.Equal(t, "hero.png", asset.Name)
	require.Equal(t, "image/png", asset.Mime)
	require.Equal(t, int64(len(pngHeader(1200, 800))), asset.Bytes)
	require.Equal(t, 1200, asset.Width)
	require.Equal(t, 800, asset.Height)
	require.Equal(t, FidelityCaptured, asset.Fidelity)
	require.Equal(t, AssetURL(design.ID, asset.ID), asset.URL)
	require.Equal(t, "2026-11-19T12:00:00.000Z", asset.AddedAt)

	listed, err := repo.ListAssets(context.Background(), design.ID)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, asset.ID, listed[0].ID)

	all, err := repo.ListAllAssets(context.Background())
	require.NoError(t, err)
	require.Len(t, all, 1)

	got, data, err := repo.GetAsset(context.Background(), design.ID, asset.ID)
	require.NoError(t, err)
	require.Equal(t, asset.ID, got.ID)
	require.Equal(t, pngHeader(1200, 800), data)

	_, _, err = repo.GetAsset(context.Background(), other.ID, asset.ID)
	require.ErrorIs(t, err, ErrNotFound, "an asset is only visible through its design")
}

func TestCreateAssetRejectsInvalidUploads(t *testing.T) {
	repo := newTestRepository(t)
	design, err := repo.Create(context.Background(), CreateInput{Name: "Assets"})
	require.NoError(t, err)
	ctx := context.Background()

	for name, input := range map[string]AssetInput{
		"empty name":      {DesignID: design.ID, Mime: "image/png", Data: pngHeader(1, 1)},
		"non-image mime":  {DesignID: design.ID, Name: "x", Mime: "application/pdf", Data: pngHeader(1, 1)},
		"empty data":      {DesignID: design.ID, Name: "x", Mime: "image/png"},
		"over size limit": {DesignID: design.ID, Name: "x", Mime: "image/png", Data: make([]byte, MaxAssetBytes+1)},
		"bad fidelity":    {DesignID: design.ID, Name: "x", Mime: "image/png", Data: pngHeader(1, 1), Fidelity: "rendered"},
		"missing design":  {DesignID: "design-nope", Name: "x", Mime: "image/png", Data: pngHeader(1, 1)},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := repo.CreateAsset(ctx, input)
			if name == "missing design" {
				require.ErrorIs(t, err, ErrNotFound)
				return
			}
			require.ErrorIs(t, err, ErrInvalid)
		})
	}
}

func TestDeleteAssetIsScopedToDesign(t *testing.T) {
	repo := newTestRepository(t)
	owner, err := repo.Create(context.Background(), CreateInput{Name: "Owner"})
	require.NoError(t, err)
	other, err := repo.Create(context.Background(), CreateInput{Name: "Other"})
	require.NoError(t, err)

	asset, err := repo.CreateAsset(context.Background(), AssetInput{
		DesignID: owner.ID,
		Name:     "hero.png",
		Mime:     "image/png",
		Data:     pngHeader(10, 10),
	})
	require.NoError(t, err)

	// Deleting through another design is a no-op, not a cross-design delete.
	require.NoError(t, repo.DeleteAsset(context.Background(), other.ID, asset.ID))
	_, _, err = repo.GetAsset(context.Background(), owner.ID, asset.ID)
	require.NoError(t, err, "asset of A must survive a delete through B")

	require.NoError(t, repo.DeleteAsset(context.Background(), owner.ID, asset.ID))
	_, _, err = repo.GetAsset(context.Background(), owner.ID, asset.ID)
	require.ErrorIs(t, err, ErrNotFound)

	require.NoError(t, repo.DeleteAsset(context.Background(), owner.ID, asset.ID), "delete is idempotent")
}

func TestDeleteDesignRemovesAssets(t *testing.T) {
	repo := newTestRepository(t)
	design, err := repo.Create(context.Background(), CreateInput{Name: "Disposable"})
	require.NoError(t, err)
	asset, err := repo.CreateAsset(context.Background(), AssetInput{
		DesignID: design.ID,
		Name:     "hero.png",
		Mime:     "image/png",
		Data:     pngHeader(10, 10),
	})
	require.NoError(t, err)

	require.NoError(t, repo.Delete(context.Background(), design.ID))

	all, err := repo.ListAllAssets(context.Background())
	require.NoError(t, err)
	require.Empty(t, all)

	var count int
	require.NoError(t, repo.db.QueryRow(`SELECT COUNT(*) FROM studio_design_assets WHERE id = ?`, asset.ID).Scan(&count))
	require.Zero(t, count, "assets are deleted in the design-delete transaction")
}

func TestAssetDataSurvivesRoundTrip(t *testing.T) {
	repo := newTestRepository(t)
	design, err := repo.Create(context.Background(), CreateInput{Name: "Assets"})
	require.NoError(t, err)
	payload := append(pngHeader(42, 24), bytes.Repeat([]byte{7}, 32)...)
	asset, err := repo.CreateAsset(context.Background(), AssetInput{
		DesignID: design.ID,
		Name:     "round-trip.png",
		Mime:     "image/png",
		Data:     payload,
	})
	require.NoError(t, err)
	_, data, err := repo.GetAsset(context.Background(), design.ID, asset.ID)
	require.NoError(t, err)
	require.Equal(t, payload, data)
}
