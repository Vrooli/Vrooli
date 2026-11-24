package inventory

import (
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	provenancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/contracts/provenance"
	"react-component-library/internal/adoptions"
)

func TestBuildProvenanceDistinguishesAuthoritativeDriftAndOrphanTags(t *testing.T) {
	content := []byte("/**\n * @vrooliComponentName Button\n * @vrooliComponentSource react-component-library:Button\n * @vrooliComponentVersion 1.2.0\n * @vrooliComponentAdoption adoption-7\n * @vrooliComponentAppliedAt 2026-01-02T03:04:05Z\n * @vrooliComponentSourceSha256 source-hash\n * @vrooliComponentDriftHash stale-tag\n */\nexport const Button = () => null;\n")
	actualHash := fmt.Sprintf("%x", sha256.Sum256(content))
	appliedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.FixedZone("UTC-5", -5*60*60))
	row := adoptions.Adoption{
		ID: "adoption-7", LibraryID: "react-component-library:Button", AdoptedVersion: "1.2.0",
		AppliedAt: appliedAt, SourceSHA256: actualHash,
	}

	for _, tc := range []struct {
		name       string
		row        map[string]adoptions.Adoption
		wantStatus provenancev1.Provenance
	}{
		{name: "clean authoritative row", row: map[string]adoptions.Adoption{"ui/src/Button.tsx": row}, wantStatus: provenancev1.Provenance_PROVENANCE_ADOPTED_UNMODIFIED},
		{name: "locally modified authoritative row", row: map[string]adoptions.Adoption{"ui/src/Button.tsx": {ID: row.ID, LibraryID: row.LibraryID, AdoptedVersion: row.AdoptedVersion, AppliedAt: row.AppliedAt, SourceSHA256: "older-source-hash"}}, wantStatus: provenancev1.Provenance_PROVENANCE_ADOPTED_MODIFIED},
		{name: "orphaned heal-from tags", row: map[string]adoptions.Adoption{}, wantStatus: provenancev1.Provenance_PROVENANCE_UNKNOWN},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := buildProvenance("ui/src/Button.tsx", content, tc.row)
			if got.GetProvenance() != tc.wantStatus {
				t.Fatalf("provenance = %s, want %s", got.GetProvenance(), tc.wantStatus)
			}
			if got.GetDriftHash() != actualHash {
				t.Fatalf("drift hash = %q, want content hash %q", got.GetDriftHash(), actualHash)
			}
			if tc.name != "orphaned heal-from tags" {
				if got.GetLibrary() != row.LibraryID || got.GetLibraryVersion() != row.AdoptedVersion || got.GetAdoptionId() != row.ID {
					t.Fatalf("authoritative adoption metadata lost: %#v", got)
				}
				if got.GetAppliedAt() != "2026-01-02T08:04:05Z" {
					t.Fatalf("applied_at = %q, want normalized UTC", got.GetAppliedAt())
				}
				return
			}
			if got.GetLibrary() != "react-component-library:Button" || got.GetLibraryVersion() != "1.2.0" || got.GetAdoptionId() != "adoption-7" {
				t.Fatalf("orphaned JSDoc metadata lost: %#v", got)
			}
		})
	}

	custom := buildProvenance("ui/src/Custom.tsx", []byte("export const Custom = () => null;"), nil)
	if custom.GetProvenance() != provenancev1.Provenance_PROVENANCE_CUSTOM || custom.GetDriftHash() == "" {
		t.Fatalf("untagged source should remain custom with a content fingerprint: %#v", custom)
	}
}
