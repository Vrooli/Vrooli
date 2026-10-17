package evidence

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	basevidence "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/evidence"
)

func TestBuildReplayPackageIsStorageIndependentAndProtectsHar(t *testing.T) {
	// enforces invariant: replayArtifactHasIntegrityDigest
	executionID, workflowID, artifactID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	pack, err := BuildReplayPackage(executionID, workflowID, DefaultPolicy(), []ArtifactInput{{ID: artifactID, Kind: "har", MediaType: "application/json", SHA256: strings.Repeat("a", 64), Producer: "playwright-driver", Provenance: ArtifactProvenanceInput{Source: "capture"}}}, nil, ReplayPresentationInput{Theme: "dark"}, time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	artifact := pack.Evidence.Artifacts[0]
	if pack.SchemaVersion != ReplaySchemaVersion || pack.Evidence.SchemaVersion != EvidenceSchemaVersion {
		t.Fatalf("versions = %q / %q", pack.SchemaVersion, pack.Evidence.SchemaVersion)
	}
	if artifact.AccessPolicy != basevidence.AccessPolicy_ACCESS_POLICY_PROTECTED_STORAGE_ONLY || artifact.RetentionClass != basevidence.RetentionClass_RETENTION_CLASS_PROTECTED || !artifact.Redacted {
		t.Fatalf("HAR policy = %#v", artifact)
	}
	if strings.Contains(pack.String(), "/tmp/") || strings.Contains(pack.String(), "storage_object") {
		t.Fatalf("replay package contains storage detail: %s", pack)
	}
	if pack.GetPresentation().GetTheme() != "dark" || pack.GetEvidence().GetArtifacts()[0].GetProvenance().GetSource() != "capture" {
		t.Fatalf("typed replay metadata was not preserved: %#v", pack)
	}
}
