package evidence

import (
	"context"
	"testing"
	"time"

	"deployment-manager/internal/testutil"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func evidenceVerdict() *commonv1.TargetVerdict {
	return &commonv1.TargetVerdict{
		Target: &commonv1.EvidenceTarget{
			Ramp: "scenario-to-desktop", Platform: "desktop", Os: "linux",
			DeviceKind: commonv1.DeviceKind_DEVICE_KIND_HOST,
		},
		Disposition: commonv1.Disposition_DISPOSITION_PASSED,
		RunId:       "run-1",
		Detail:      "journey completed",
		Refs: []*commonv1.EvidenceRef{{
			Producer: "scenario-to-desktop", ArtifactId: "capture-1", Kind: "recording",
			Checksum: "sha256:abc", SizeBytes: 12, CreatedAt: timestamppb.New(time.Unix(10, 0)),
		}},
	}
}

func TestSQLRepositorySaveAndListRoundTrip(t *testing.T) {
	db := testutil.OpenSQLite(t)
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	repo := NewSQLRepository(db, "sqlite")
	ctx := context.Background()
	if err := repo.Save(ctx, "profile-1", "commit-1", evidenceVerdict()); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := repo.List(ctx, "profile-1", "commit-1", 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || len(got[0].Refs) != 1 || got[0].Refs[0].ArtifactId != "capture-1" {
		t.Fatalf("unexpected round trip: %+v", got)
	}
	if got[0].Target.BridgeNodeId != nil {
		t.Fatal("unexpected bridge node")
	}
}

func TestSQLRepositoryRejectsInvalidInput(t *testing.T) {
	db := testutil.OpenSQLite(t)
	repo := NewSQLRepository(db, "sqlite")
	cases := []struct {
		name    string
		profile string
		commit  string
		verdict *commonv1.TargetVerdict
	}{
		{"nil repository value", "p", "c", nil},
		{"missing profile", "", "c", evidenceVerdict()},
		{"missing commit", "p", "", evidenceVerdict()},
		{"missing run", "p", "c", func() *commonv1.TargetVerdict { v := evidenceVerdict(); v.RunId = ""; return v }()},
		{"missing target", "p", "c", func() *commonv1.TargetVerdict { v := evidenceVerdict(); v.Target = nil; return v }()},
		{"nil ref", "p", "c", func() *commonv1.TargetVerdict {
			v := evidenceVerdict()
			v.Refs = []*commonv1.EvidenceRef{nil}
			return v
		}()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := repo.Save(context.Background(), tc.profile, tc.commit, tc.verdict); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
	if _, err := repo.List(context.Background(), "p", "c", 0); err == nil {
		// The schema is intentionally absent, so this exercises the repository
		// error boundary rather than silently treating an unconfigured store as empty.
		t.Fatal("expected list error for uninitialized store")
	}
	var nilRepo *SQLRepository
	if err := nilRepo.Save(context.Background(), "p", "c", evidenceVerdict()); err == nil {
		t.Fatal("expected nil repository error")
	}
}

func TestOptionalString(t *testing.T) {
	if optionalString(nil) != nil {
		t.Fatal("nil pointer should remain SQL NULL")
	}
	value := "node-1"
	if got := optionalString(&value); got != value {
		t.Fatalf("got %v, want %q", got, value)
	}
}
