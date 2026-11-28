package release

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"backdrop-studio/internal/catalog"
	internal "backdrop-studio/internal/release"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	releasev1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/release"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/shared"
)

type candidateSource map[string]internal.CandidateEvidence

func (s candidateSource) CandidateEvidence(id string) (internal.CandidateEvidence, bool) {
	candidate, ok := s[id]
	return candidate, ok
}

type provenanceSource struct{}

func (provenanceSource) CandidateProvenance(string) (internal.Provenance, bool) {
	return internal.Provenance{Strategy: "procedural", ModelBacked: false}, true
}

func releaseTestPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, color.Black)
		}
	}
	var out bytes.Buffer
	require.NoError(t, png.Encode(&out, img))
	return out.Bytes()
}

func TestAssetEndpointReturnsQualifiedCandidateBytes(t *testing.T) {
	pngBytes := releaseTestPNG(t)
	store := internal.NewStoreWithPublisherAndRoots(nil, provenanceSource{}, filerouting.New(storage.Paths{DataDir: t.TempDir()}), candidateSource{"candidate-1": {
		ID: "candidate-1", JobID: "job-candidate-1", ImagePNG: pngBytes, Width: 8, Height: 8,
		StyleID: "style-1", Strategy: "procedural", SurfaceID: "web.hero", Placement: "full_bleed",
		Regions: []catalog.Region{{X: 0, Y: 0, Width: 1, Height: 1, Kind: "overlay", TextColor: "#FFFFFF"}}, ContrastThreshold: 4.5,
	}})
	released, err := store.Release(internal.Request{CandidateID: "candidate-1", StyleID: "style-1", Strategy: "procedural", SurfaceID: "web.hero", AltText: "A black ambient backdrop"})
	require.NoError(t, err)

	router := mux.NewRouter()
	Module(store).Mount(router)
	req := httptest.NewRequest("GET", "/api/v1/backdrops/"+released.ID+"/asset", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	require.Equal(t, 200, res.Code)
	require.Equal(t, pngBytes, res.Body.Bytes())
	require.Equal(t, "image/png", res.Header().Get("Content-Type"))
	require.Equal(t, released.ContentHash, res.Header().Get("X-Content-SHA256"))
	require.NotEmpty(t, res.Header().Get("ETag"))
	proto := toProto(released)
	require.Equal(t, released.JobID, proto.GetJobId())
	require.Equal(t, released.MIMEType, proto.GetMimeType())
	require.Equal(t, released.ContentHash, proto.GetContentHash())
}

func TestReleaseReferenceAndAssetEndpointPreserveQualifiedFactsInLeasedRoot(t *testing.T) {
	pngBytes := releaseTestPNG(t)
	candidate := internal.CandidateEvidence{
		ID: "candidate-qualified", JobID: "job-qualified", ImagePNG: pngBytes, Width: 8, Height: 8,
		StyleID: "style-qualified", Strategy: "procedural", SurfaceID: "web.hero", Placement: "full_bleed",
		Regions: []catalog.Region{{X: 0, Y: 0, Width: 1, Height: 1, Kind: "overlay", TextColor: "#FFFFFF"}}, ContrastThreshold: 4.5,
	}
	primary := storage.Paths{ConfigDir: t.TempDir(), DataDir: t.TempDir(), CacheDir: t.TempDir(), LogsDir: t.TempDir(), StateDir: t.TempDir()}
	roots := filerouting.New(primary)
	const leaseID = "release-handler-qualification"
	_, err := roots.InstallLeasedTestRoots(leaseID, time.Minute, true)
	require.NoError(t, err)
	defer func() { require.NoError(t, roots.ClearTestRoots(leaseID)) }()

	store := internal.NewStoreWithPublisherAndRoots(nil, provenanceSource{}, roots, candidateSource{candidate.ID: candidate})
	h := &handler{store: store}
	requestCtx := database.WithTestMode(context.Background())
	releaseResponse, err := h.Release(requestCtx, connect.NewRequest(&releasev1.ReleaseRequest{
		CandidateId: candidate.ID,
		StyleId:     candidate.StyleID,
		Strategy:    candidate.Strategy,
		SurfaceId:   candidate.SurfaceID,
		Placement:   candidate.Placement,
		AltText:     "A qualified test backdrop",
	}))
	require.NoError(t, err)
	require.NotNil(t, releaseResponse)

	referenceResponse, err := h.GetReference(requestCtx, connect.NewRequest(&releasev1.GetReferenceRequest{Id: releaseResponse.Msg.GetId()}))
	require.NoError(t, err)
	ref := referenceResponse.Msg
	released, err := store.GetContext(requestCtx, ref.GetId())
	require.NoError(t, err)

	hash := sha256.Sum256(pngBytes)
	expectedHash := hex.EncodeToString(hash[:])
	require.Equal(t, released.ID, ref.GetId())
	require.Equal(t, candidate.ID, ref.GetCandidateId())
	require.Equal(t, candidate.JobID, ref.GetJobId())
	require.Equal(t, candidate.StyleID, ref.GetStyleId())
	require.Equal(t, candidate.SurfaceID, ref.GetSurfaceId())
	require.Equal(t, candidate.Placement, ref.GetPlacement())
	require.Equal(t, int32(candidate.Width), ref.GetWidth())
	require.Equal(t, int32(candidate.Height), ref.GetHeight())
	require.Equal(t, "image/png", ref.GetMimeType())
	require.Equal(t, expectedHash, ref.GetContentHash())
	require.Equal(t, released.ContrastRatio, ref.GetContrastRatio())
	require.Equal(t, released.ContrastThreshold, ref.GetContrastThreshold())
	require.Len(t, ref.GetReservedRegions(), len(candidate.Regions))
	for i, want := range candidate.Regions {
		got := ref.GetReservedRegions()[i]
		require.Equal(t, want.X, got.GetX())
		require.Equal(t, want.Y, got.GetY())
		require.Equal(t, want.Width, got.GetWidth())
		require.Equal(t, want.Height, got.GetHeight())
		require.Equal(t, want.Kind, got.GetKind())
		require.Equal(t, want.TextColor, got.GetTextColor())
	}

	router := mux.NewRouter()
	Module(store).Mount(router)
	req := httptest.NewRequest("GET", ref.GetUri(), nil).WithContext(requestCtx)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	require.Equal(t, 200, res.Code)
	require.Equal(t, pngBytes, res.Body.Bytes())
	require.Equal(t, ref.GetContentHash(), res.Header().Get("X-Content-SHA256"))
	require.Equal(t, ref.GetMimeType(), res.Header().Get("Content-Type"))
	require.Equal(t, "\""+released.ContentHash+"\"", res.Header().Get("ETag"))
	config, format, err := image.DecodeConfig(bytes.NewReader(res.Body.Bytes()))
	require.NoError(t, err)
	require.Equal(t, "png", format)
	require.Equal(t, int(ref.GetWidth()), config.Width)
	require.Equal(t, int(ref.GetHeight()), config.Height)
	// The same ID is deliberately invisible without the lease context. This
	// catches a handler that accidentally falls back to the primary root.
	plainReq := httptest.NewRequest("GET", ref.GetUri(), nil)
	plainRes := httptest.NewRecorder()
	router.ServeHTTP(plainRes, plainReq)
	require.Equal(t, 404, plainRes.Code)
	require.Zero(t, roots.LeaseStats().PrimaryWritesDuringTestMode)
	require.NotZero(t, roots.LeaseStats().TestRootWrites)
	writeQualificationReceiptIfRequested(t, ref, res.Body.Bytes(), roots.LeaseStats())
}

type qualificationReceipt struct {
	Kind              string                     `json:"kind"`
	CandidateID       string                     `json:"candidate_id"`
	ReleaseID         string                     `json:"release_id"`
	JobID             string                     `json:"job_id"`
	StyleID           string                     `json:"style_id"`
	SurfaceID         string                     `json:"surface_id"`
	Placement         string                     `json:"placement"`
	Width             int32                      `json:"width"`
	Height            int32                      `json:"height"`
	MIMEType          string                     `json:"mime_type"`
	ContentHash       string                     `json:"content_hash"`
	ContrastRatio     float64                    `json:"contrast_ratio"`
	ContrastThreshold float64                    `json:"contrast_threshold"`
	ReservedRegions   []*sharedv1.ReservedRegion `json:"reserved_regions"`
	AssetPNGBase64    string                     `json:"asset_png_base64"`
	TestRootWrites    int64                      `json:"test_root_writes"`
	PrimaryWrites     int64                      `json:"primary_writes_during_test_mode"`
}

func writeQualificationReceiptIfRequested(t *testing.T, ref *releasev1.ReleasedBackdrop, body []byte, stats filerouting.LeaseStatsSnapshot) {
	t.Helper()
	path := os.Getenv("BACKDROP_QUALIFICATION_RECEIPT_PATH")
	if path == "" {
		return
	}
	receipt := qualificationReceipt{
		Kind:              "backdrop-studio/isolated-owner-qualification",
		CandidateID:       ref.GetCandidateId(),
		ReleaseID:         ref.GetId(),
		JobID:             ref.GetJobId(),
		StyleID:           ref.GetStyleId(),
		SurfaceID:         ref.GetSurfaceId(),
		Placement:         ref.GetPlacement(),
		Width:             ref.GetWidth(),
		Height:            ref.GetHeight(),
		MIMEType:          ref.GetMimeType(),
		ContentHash:       ref.GetContentHash(),
		ContrastRatio:     ref.GetContrastRatio(),
		ContrastThreshold: ref.GetContrastThreshold(),
		ReservedRegions:   ref.GetReservedRegions(),
		AssetPNGBase64:    base64.StdEncoding.EncodeToString(body),
		TestRootWrites:    stats.TestRootWrites,
		PrimaryWrites:     stats.PrimaryWritesDuringTestMode,
	}
	data, err := json.MarshalIndent(receipt, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	require.NoError(t, err, "qualification receipts must not overwrite earlier evidence")
	_, writeErr := file.Write(append(data, '\n'))
	closeErr := file.Close()
	require.NoError(t, writeErr)
	require.NoError(t, closeErr)
}
