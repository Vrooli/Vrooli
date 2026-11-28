package corpus

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	corpusv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/corpus"
)

func TestLoadSourceManifestRecordDerivesCanonicalLineage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"corpus_revision":"vctk-v1","records":[{"source_id":"p230_001","speaker_pseudonym":"vctk-p230","accent_group":"uk","split":"held_out","transcript_sha256":"`+strings.Repeat("a", 64)+`","license_reference":"VCTK license"}]}`), 0600))
	got, err := loadSourceManifestRecord(path, "p230_001")
	require.NoError(t, err)
	require.Equal(t, "vctk-v1", got.CorpusRevision)
	require.Equal(t, "vctk-p230", got.SpeakerPseudonym)
	require.Equal(t, "uk", got.AccentGroup)
	require.Equal(t, "held_out", got.Split)
	require.Equal(t, strings.Repeat("a", 64), got.TranscriptSHA256)
	require.Equal(t, "VCTK license", got.LicenseReference)
}

func TestLoadSourceManifestRecordRejectsUnboundLineageAndMissingSource(t *testing.T) {
	_, err := loadSourceManifestRecord("", "p230_001")
	require.ErrorContains(t, err, "must be supplied together")
	path := filepath.Join(t.TempDir(), "manifest.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"corpus_revision":"v1","records":[]}`), 0600))
	_, err = loadSourceManifestRecord(path, "missing")
	require.ErrorContains(t, err, "was not found")
}

func TestResolveSourceManifestRecordRequiresExactAudioImportTuple(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	contentSHA := strings.Repeat("c", 64)
	manifest := `{"corpus_revision":"vctk-v1","records":[{"source_id":"p230_001","speaker_pseudonym":"vctk-p230","accent_group":"uk","split":"held_out","transcript_sha256":"` + strings.Repeat("a", 64) + `","license_reference":"VCTK license"}],"imports":[{"clip_id":"clip-1","source_id":"p230_001","variant":"noisy_20db","content_sha256":"` + contentSHA + `","measured_snr_db":20}]}`
	require.NoError(t, os.WriteFile(path, []byte(manifest), 0600))
	got, err := resolveSourceManifestRecord(path, "p230_001", contentSHA)
	require.NoError(t, err)
	require.Equal(t, "noisy_20db", got.Variant)
	require.Equal(t, contentSHA, got.ContentSHA256)
	require.NotEmpty(t, got.ManifestSHA256)
	require.NotNil(t, got.MeasuredSNRDB)
	require.Equal(t, 20.0, *got.MeasuredSNRDB)
	_, err = resolveSourceManifestRecord(path, "p230_001", strings.Repeat("d", 64))
	require.ErrorContains(t, err, "found 0")
}

func TestLineageBackfillSummaryDistinguishesZeroCountDryRunFromApply(t *testing.T) {
	require.Equal(t, "Dry run: 0 clips would be updated; no rows changed.\n", lineageBackfillSummary(lineageBackfillResult{DryRun: true}))
	require.Equal(t, "Applied source lineage to 0 clips.\n", lineageBackfillSummary(lineageBackfillResult{}))
}

func TestSourceLabel(t *testing.T) {
	if got := sourceLabel(corpusv1.ClipSource_CLIP_SOURCE_SCRIPTED); got != "scripted" {
		t.Fatalf("scripted label = %q", got)
	}
	if got := sourceLabel(corpusv1.ClipSource_CLIP_SOURCE_FREE_FORM); got != "free_form" {
		t.Fatalf("free-form label = %q", got)
	}
}

func TestSourceFromContextUsesBooleanFlagPresence(t *testing.T) {
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "scripted", Bool: true}}}
	app := cliapptest.NewTestApp(t, nil)

	ctx, _ := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		BoolFlags: map[string]bool{"scripted": true},
	})
	require.Equal(t, corpusv1.ClipSource_CLIP_SOURCE_SCRIPTED, sourceFromContext(ctx))

	ctx, _ = cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{})
	require.Equal(t, corpusv1.ClipSource_CLIP_SOURCE_FREE_FORM, sourceFromContext(ctx))
}

func TestRenderImportUsesLengthCapturedDuringOperation(t *testing.T) {
	app := cliapptest.NewTestApp(t, nil)
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "audio-file", Required: true}}}
	ctx, output := cliapptest.NewCapturedRunContext(app, schema, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"audio-file": "this-path-must-not-be-read-again.wav"},
	})
	result := importResult{
		response:   &corpusv1.CreateClipResponse{Clip: &corpusv1.Clip{Id: "clip-1", DurationMs: 90, Format: "pcm_s16le"}},
		audioBytes: 2880,
	}
	require.NoError(t, renderImport(ctx, result))
	require.Equal(t, "Imported clip clip-1 (2880 bytes, 90ms, pcm_s16le).\n", output.String())
	require.NotContains(t, output.String(), "this-path-must-not-be-read-again")
}

func TestParseTimedReferencePreservesReviewProvenance(t *testing.T) {
	input := `{"words":[{"word":"hello","start_seconds":0.1,"end_seconds":0.3}],"provenance":{"status":"human_reviewed","source_reference":"source-1","alignment_tool":"aligner","alignment_model":"model-1","reviewer":"reviewer-1","review_method":"listen-and-correct","reviewed_at":"2026-06-29T12:00:00Z"}}`
	got, err := parseTimedReference([]byte(input))
	require.NoError(t, err)
	require.Equal(t, "reviewer-1", got.GetProvenance().GetReviewer())
	require.Equal(t, 0.1, got.GetWords()[0].GetStartSeconds())
	_, err = parseTimedReference([]byte(strings.Replace(input, `"reviewer":"reviewer-1",`, `"unrecognized":"x",`, 1)))
	require.Error(t, err)
	_, err = parseTimedReference([]byte(input + input))
	require.Error(t, err)

	manual := `{"words":[{"word":"manual","start_seconds":0.2,"end_seconds":0.6}],"provenance":{"status":"human_reviewed","source_reference":"ami-meeting:sha256:abc123","reviewer":"reviewer-1","review_method":"listen-and-correct","reviewed_at":"2026-11-20T22:00:00Z"}}`
	got, err = parseTimedReference([]byte(manual))
	require.NoError(t, err)
	require.Empty(t, got.GetProvenance().GetAlignmentTool())
	require.Empty(t, got.GetProvenance().GetAlignmentModel())
	require.Equal(t, "reviewer-1", got.GetProvenance().GetReviewer())
}

func TestParseReferenceReviewRequiresExplicitAuditorAndProtocol(t *testing.T) {
	got, err := parseReferenceReview([]byte(`{"reviewer":"auditor-1","protocol":"listen-and-correct-v1","reviewed_at":"2026-11-20T22:00:00Z"}`))
	require.NoError(t, err)
	require.Equal(t, "auditor-1", got.GetReviewer())
	_, err = parseReferenceReview([]byte(`{"reviewer":"","protocol":"","reviewed_at":""}`))
	require.Error(t, err)
	_, err = parseReferenceReview([]byte(`{"reviewer":"a","protocol":"p","reviewed_at":"now","unknown":true}`))
	require.Error(t, err)
}
