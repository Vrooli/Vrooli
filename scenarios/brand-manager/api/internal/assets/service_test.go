package assets_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"brand-manager/internal/assets"
	mocks "brand-manager/internal/assets/mocks"

	"github.com/stretchr/testify/require"
)

func newService(repo *mocks.FakeRepository, blobs *mocks.FakeBlobStore, known ...string) assets.Service {
	resolver := mocks.FakeBrandResolver{Known: map[string]bool{}}
	for _, b := range known {
		resolver.Known[b] = true
	}
	return assets.NewService(repo, blobs, resolver, nil)
}

func pngBytes() []byte { return []byte("\x89PNG\r\n\x1a\nfake") }

func TestService_Upload_RejectsMissingBrandID(t *testing.T) {
	repo := &mocks.FakeRepository{}
	blobs := &mocks.FakeBlobStore{}
	svc := newService(repo, blobs)

	_, err := svc.Upload(context.Background(), assets.UploadInput{Filename: "logo.png", Content: pngBytes()})
	require.Error(t, err)
	var inv assets.ErrInvalidAsset
	require.True(t, errors.As(err, &inv), "expected ErrInvalidAsset, got %T", err)
	require.Equal(t, "brand_id", inv.Field)
	require.Equal(t, int64(0), repo.UpsertCalls.Load(), "validation must reject before the repository")
}

func TestService_Upload_RejectsUnknownBrand(t *testing.T) {
	repo := &mocks.FakeRepository{}
	blobs := &mocks.FakeBlobStore{}
	svc := newService(repo, blobs) // no known brands

	_, err := svc.Upload(context.Background(), assets.UploadInput{BrandID: "ghost", Filename: "logo.png", Content: pngBytes()})
	require.Error(t, err)
	var inv assets.ErrInvalidAsset
	require.True(t, errors.As(err, &inv))
	require.Equal(t, "brand_id", inv.Field)
	require.Equal(t, int64(0), repo.UpsertCalls.Load(), "an unknown brand must reject before writing the blob or row")
}

func TestService_Upload_RejectsPathInFilename(t *testing.T) {
	svc := newService(&mocks.FakeRepository{}, &mocks.FakeBlobStore{}, "b1")
	_, err := svc.Upload(context.Background(), assets.UploadInput{BrandID: "b1", Filename: "../escape.png", Content: pngBytes()})
	require.Error(t, err)
	var inv assets.ErrInvalidAsset
	require.True(t, errors.As(err, &inv))
	require.Equal(t, "filename", inv.Field)
}

func TestService_Upload_RejectsUnsupportedMime(t *testing.T) {
	svc := newService(&mocks.FakeRepository{}, &mocks.FakeBlobStore{}, "b1")
	_, err := svc.Upload(context.Background(), assets.UploadInput{BrandID: "b1", Filename: "notes.txt", Content: []byte("hi")})
	require.Error(t, err)
	var inv assets.ErrInvalidAsset
	require.True(t, errors.As(err, &inv))
	require.Equal(t, "mime_type", inv.Field, "an un-inferable / non-image type is rejected")
}

func TestService_Upload_RejectsEmptyContent(t *testing.T) {
	svc := newService(&mocks.FakeRepository{}, &mocks.FakeBlobStore{}, "b1")
	_, err := svc.Upload(context.Background(), assets.UploadInput{BrandID: "b1", Filename: "logo.png"})
	require.Error(t, err)
	var inv assets.ErrInvalidAsset
	require.True(t, errors.As(err, &inv))
	require.Equal(t, "content", inv.Field)
}

func TestService_Upload_InfersMimeFromExtension(t *testing.T) {
	repo := &mocks.FakeRepository{}
	blobs := &mocks.FakeBlobStore{}
	svc := newService(repo, blobs, "b1")

	got, err := svc.Upload(context.Background(), assets.UploadInput{BrandID: "b1", Filename: "logo.SVG", Content: []byte("<svg/>")})
	require.NoError(t, err)
	require.Equal(t, "image/svg+xml", got.MimeType, "mime is inferred from the extension, case-insensitively")
	require.Equal(t, int64(len("<svg/>")), got.Size)
	require.NotEmpty(t, got.ID)
}

func TestService_Upload_ReuploadReplacesAndKeepsID(t *testing.T) {
	repo := &mocks.FakeRepository{}
	blobs := &mocks.FakeBlobStore{}
	svc := newService(repo, blobs, "b1")
	ctx := context.Background()

	first, err := svc.Upload(ctx, assets.UploadInput{BrandID: "b1", Filename: "logo.png", Content: pngBytes()})
	require.NoError(t, err)

	second, err := svc.Upload(ctx, assets.UploadInput{BrandID: "b1", Filename: "logo.png", MimeType: "image/png", Content: []byte("\x89PNGbigger-payload")})
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID, "re-uploading the same filename keeps the asset id (upsert by brand+filename)")
	require.NotEqual(t, first.Size, second.Size, "the bytes/size are replaced")

	list, err := svc.List(ctx, "b1")
	require.NoError(t, err)
	require.Len(t, list, 1, "a re-upload does not create a second catalog row")
}

func TestService_Download_ReturnsStoredBytes(t *testing.T) {
	repo := &mocks.FakeRepository{}
	blobs := &mocks.FakeBlobStore{}
	svc := newService(repo, blobs, "b1")
	ctx := context.Background()

	up, err := svc.Upload(ctx, assets.UploadInput{BrandID: "b1", Filename: "icon.webp", Content: []byte("RIFFwebp")})
	require.NoError(t, err)

	content, err := svc.Download(ctx, up.ID)
	require.NoError(t, err)
	require.Equal(t, "icon.webp", content.Filename)
	require.Equal(t, "image/webp", content.MimeType)
	require.Equal(t, "RIFFwebp", string(content.Bytes))
}

func TestService_Download_NotFound(t *testing.T) {
	svc := newService(&mocks.FakeRepository{}, &mocks.FakeBlobStore{}, "b1")
	_, err := svc.Download(context.Background(), "ghost")
	require.Error(t, err)
	var nf assets.ErrAssetNotFound
	require.True(t, errors.As(err, &nf))
}

func TestService_Delete_IsIdempotent(t *testing.T) {
	repo := &mocks.FakeRepository{}
	blobs := &mocks.FakeBlobStore{}
	svc := newService(repo, blobs, "b1")
	ctx := context.Background()

	up, err := svc.Upload(ctx, assets.UploadInput{BrandID: "b1", Filename: "logo.png", Content: pngBytes()})
	require.NoError(t, err)

	require.NoError(t, svc.Delete(ctx, up.ID))
	require.NoError(t, svc.Delete(ctx, up.ID), "deleting a missing asset is a success")
	require.NoError(t, svc.Delete(ctx, "never-existed"), "deleting an unknown id is a success")
	require.GreaterOrEqual(t, blobs.RemoveCalls.Load(), int64(1), "delete removes the on-disk bytes")
}

func TestService_Delete_BlobRemovalFailureIsNotFatal(t *testing.T) {
	repo := &mocks.FakeRepository{}
	blobs := &mocks.FakeBlobStore{RemoveErr: errors.New("disk gone")}
	svc := newService(repo, blobs, "b1")
	ctx := context.Background()

	up, err := svc.Upload(ctx, assets.UploadInput{BrandID: "b1", Filename: "logo.png", Content: pngBytes()})
	require.NoError(t, err)

	require.NoError(t, svc.Delete(ctx, up.ID), "a best-effort blob removal failure must not fail the delete")
	_, err = svc.Get(ctx, up.ID)
	require.Error(t, err, "the catalog row is still removed")
}

func TestService_Upload_SurfacesResolverOutageAsError(t *testing.T) {
	repo := &mocks.FakeRepository{}
	blobs := &mocks.FakeBlobStore{}
	resolver := mocks.FakeBrandResolver{Err: errors.New("brands db down")}
	svc := assets.NewService(repo, blobs, resolver, nil)

	_, err := svc.Upload(context.Background(), assets.UploadInput{BrandID: "b1", Filename: "logo.png", Content: pngBytes()})
	require.Error(t, err)
	var inv assets.ErrInvalidAsset
	require.False(t, errors.As(err, &inv), "a brand-lookup outage is an internal error, not a validation failure")
	require.True(t, strings.Contains(err.Error(), "brands db down"))
}
