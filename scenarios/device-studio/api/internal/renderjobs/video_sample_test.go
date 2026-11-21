package renderjobs

import (
	"testing"

	"device-studio/internal/jobs"
	"device-studio/internal/rendering"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

func TestVideoSampleManifestComesFromCompletedAttemptOutput(t *testing.T) {
	// Video manifests are staged in the attempt and projected into Completion;
	// they are deliberately not duplicated in the job's immutable input bindings.
	attempt := jobs.Attempt{Completion: &jobs.CompletionEvidence{Outputs: []jobs.OutputRef{
		{Kind: "video_manifest", ArtifactID: "receipts/job/attempt/manifest.json", SHA256: digestBytes([]byte("manifest")), SizeBytes: 42},
		{Kind: "video_receipt", ArtifactID: "receipts/job/attempt/receipt.json", SHA256: digestBytes([]byte("receipt")), SizeBytes: 36},
		{Kind: "video_sample_png", ArtifactID: "outputs/job/attempt/sample.png", SHA256: digestBytes([]byte("png")), SizeBytes: 3},
	}}}
	manifest := uniqueCompletedOutput(attempt, "video_manifest")
	if manifest == nil || manifest.ArtifactID != "receipts/job/attempt/manifest.json" || manifest.SHA256 != digestBytes([]byte("manifest")) {
		t.Fatalf("manifest was not resolved from immutable completion outputs: %+v", manifest)
	}
	if uniqueCompletedOutput(attempt, "video_receipt") == nil || uniqueCompletedOutput(attempt, "video_sample_png") == nil {
		t.Fatal("completed sample receipt/output were not discoverable")
	}
	attempt.Completion.Outputs = append(attempt.Completion.Outputs, jobs.OutputRef{Kind: "video_manifest", ArtifactID: "replacement", SHA256: digestBytes([]byte("replacement")), SizeBytes: 11})
	if uniqueCompletedOutput(attempt, "video_manifest") != nil {
		t.Fatal("ambiguous duplicate manifest outputs must not be accepted")
	}
}

func TestVideoSampleClaimSourceAndTrimMatchFrozenManifest(t *testing.T) {
	sourceSHA := digestBytes([]byte("inspected video source"))
	manifest := rendering.VideoManifest{
		Inspection:   rendering.VideoInspection{Source: rendering.VideoBinding{SHA256: sourceSHA}},
		TrimStartPTS: 120, TrimEndPTS: 900,
	}
	claim := &shotsv1.VideoSampleBinding{SourceSha256: sourceSHA, TrimStartPts: 120, TrimEndPts: 900}
	if !videoSampleSourceTimingMatchesManifest(claim, manifest) {
		t.Fatal("valid claim copied from the verified manifest was rejected")
	}
	for name, mutate := range map[string]func(*shotsv1.VideoSampleBinding){
		"source digest": func(c *shotsv1.VideoSampleBinding) { c.SourceSha256 = digestBytes([]byte("different source")) },
		"trim start":    func(c *shotsv1.VideoSampleBinding) { c.TrimStartPts++ },
		"trim end":      func(c *shotsv1.VideoSampleBinding) { c.TrimEndPts-- },
	} {
		t.Run(name, func(t *testing.T) {
			tampered := *claim
			mutate(&tampered)
			if videoSampleSourceTimingMatchesManifest(&tampered, manifest) {
				t.Fatal("tampered sample claim matched the immutable worker manifest")
			}
		})
	}
}
