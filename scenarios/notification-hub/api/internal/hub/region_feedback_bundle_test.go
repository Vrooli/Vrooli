package hub

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func feedbackBytes(label string) []byte {
	return append([]byte("\x89PNG\r\n\x1a\n"), []byte(label)...)
}

func feedbackDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func validRegionFeedbackFolder(t *testing.T) (AskAttachment, map[string][]byte) {
	t.Helper()
	before, after, crop := feedbackBytes("before"), feedbackBytes("after"), feedbackBytes("region")
	manifest := regionFeedbackManifest{
		Schema: regionFeedbackSchema, ID: "capture-set-1", Route: "/settings", StateID: "panel-open", DataID: "fixture-1",
		Before: regionFeedbackCapture{ID: "capture-before", BuildID: "sha256:" + strings.Repeat("a", 64), Route: "/settings", StateID: "panel-open", DataID: "fixture-1", Fidelity: "captured", Viewport: struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		}{Width: 390, Height: 844}, Image: struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		}{Path: "captures/before.png", SHA256: feedbackDigest(before)}},
		After: regionFeedbackCapture{ID: "capture-after", BuildID: "sha256:" + strings.Repeat("b", 64), Route: "/settings", StateID: "panel-open", DataID: "fixture-1", Fidelity: "composited", Viewport: struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		}{Width: 390, Height: 844}, Image: struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		}{Path: "captures/after.png", SHA256: feedbackDigest(after)}},
		Regions: []regionFeedbackRegion{{ID: "main-nav", NodeID: "outline-18", Label: "Main navigation", Component: "BottomNav", EvidenceID: "crop-1", Bounds: struct {
			X      float64 `json:"x"`
			Y      float64 `json:"y"`
			Width  float64 `json:"width"`
			Height float64 `json:"height"`
		}{X: .1, Y: .8, Width: .8, Height: .15}, Crop: struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		}{Path: "regions/crops/nav.png", SHA256: feedbackDigest(crop)}}},
	}
	manifestBytes, err := json.Marshal(manifest)
	require.NoError(t, err)
	contents := map[string][]byte{
		feedbackDigest(manifestBytes): manifestBytes,
		feedbackDigest(before):        before,
		feedbackDigest(after):         after,
		feedbackDigest(crop):          crop,
	}
	folder := AskAttachment{Name: "capture-review", Kind: AttachmentFolder, Entries: []AskAttachmentEntry{
		{Path: regionFeedbackManifestName, SHA256: feedbackDigest(manifestBytes), SizeBytes: int64(len(manifestBytes)), MediaType: "application/json"},
		{Path: "captures/before.png", SHA256: feedbackDigest(before), SizeBytes: int64(len(before)), MediaType: "image/png"},
		{Path: "captures/after.png", SHA256: feedbackDigest(after), SizeBytes: int64(len(after)), MediaType: "image/png"},
		{Path: "regions/crops/nav.png", SHA256: feedbackDigest(crop), SizeBytes: int64(len(crop)), MediaType: "image/png"},
		{Path: "regions/main-nav.json", SHA256: strings.Repeat("f", 64), SizeBytes: 2, MediaType: "application/json"},
	}}
	return folder, contents
}

func TestCheckRegionFeedbackBundlesRequiresBoundCaptureAndRegionEvidence(t *testing.T) {
	folder, contents := validRegionFeedbackFolder(t)
	read := func(_ context.Context, digest string, limit int64) ([]byte, error) {
		data := contents[digest]
		if int64(len(data)) > limit {
			return nil, ErrBlobTooLarge
		}
		return data, nil
	}
	warnings := checkRegionFeedbackBundles(context.Background(), AskSpec{Attachments: []AskAttachment{folder}}, read)
	require.Empty(t, warnings)

	var manifest regionFeedbackManifest
	require.NoError(t, json.Unmarshal(contents[folder.Entries[0].SHA256], &manifest))
	manifest.After.Route = "/profile"
	manifest.After.Fidelity = "mystery"
	changed, err := json.Marshal(manifest)
	require.NoError(t, err)
	folder.Entries[0].SHA256 = feedbackDigest(changed)
	contents[folder.Entries[0].SHA256] = changed
	warnings = checkRegionFeedbackBundles(context.Background(), AskSpec{Attachments: []AskAttachment{folder}}, read)
	require.Len(t, warnings, 1)
	require.Equal(t, WarnRegionFeedbackIncomplete, warnings[0].Code)
	require.Contains(t, warnings[0].Message, "fidelity must be captured")
	require.Contains(t, warnings[0].Message, "manifest route")
}

func TestCheckRegionFeedbackBundlesRejectsMissingCropAndBadImageBytes(t *testing.T) {
	folder, contents := validRegionFeedbackFolder(t)
	folder.Entries[1].MediaType = "image/jpeg"
	read := func(_ context.Context, digest string, _ int64) ([]byte, error) { return contents[digest], nil }
	warnings := checkRegionFeedbackBundles(context.Background(), AskSpec{Attachments: []AskAttachment{folder}}, read)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0].Message, "image bytes do not match")

	folder.Entries[1].MediaType = "image/png"
	folder.Entries = folder.Entries[:len(folder.Entries)-2]
	warnings = checkRegionFeedbackBundles(context.Background(), AskSpec{Attachments: []AskAttachment{folder}}, read)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0].Message, "crop is missing")
}
