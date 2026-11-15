package repositorymigration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"data-backup-manager/internal/destinationreadiness"
	"data-backup-manager/internal/destinations"
	destmocks "data-backup-manager/internal/destinations/mocks"
	"data-backup-manager/internal/plans"
	planmocks "data-backup-manager/internal/plans/mocks"
	"data-backup-manager/internal/sources"
	"data-backup-manager/internal/sysmounts"
	"data-backup-manager/internal/targets"
	"data-backup-manager/internal/testutil/mocks"
)

type migrationInspector struct {
	inspection destinationreadiness.Inspection
}

func (i migrationInspector) Inspect(context.Context, string) (destinationreadiness.Inspection, error) {
	return i.inspection, nil
}

func migrationEffectFixture(t *testing.T) (*DBMEffects, *planmocks.FakeService, plans.Plan, []string) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "source.bin"), make([]byte, 1024), 0o600); err != nil {
		t.Fatal(err)
	}
	targetIDs := make([]string, RequiredTargetCount)
	targetRows := make(map[string]targets.Target)
	for i := range targetIDs {
		id := fmtTarget(i)
		targetIDs[i] = id
		targetRows[id] = targets.Target{ID: id, SourceKind: sources.KindFilesystem, Locator: root}
	}
	plan := plans.Plan{ID: "plan-14", TargetIDs: targetIDs, DestinationIDs: []string{"old"}, Enabled: true, KeepLatest: 1, ProtectionTier: plans.TierFullPrimary}
	planSvc := &planmocks.FakeService{GetOut: plan, UpdateOut: plan}
	identity := destinationreadiness.DeviceIdentity{DevicePath: "/dev/backup", Mountpoint: "/mnt/backup", Filesystem: "ext4", UUID: "volume-uuid", Serial: "volume-serial", TotalBytes: 1 << 40}
	engine := &mocks.FakeKopiaEngine{}
	dest := destinations.Destination{ID: "new", Name: "backup-offhost", BackendKind: destinations.BackendFilesystem, Location: "/mnt/backup/dbm", RepositoryLocation: "/mnt/backup/dbm/repositories/kopia", ProvisioningState: destinations.ProvisioningReady, SecretRef: engine.PassphraseRef("backup-offhost"), DeviceIdentity: &identity}
	destSvc := &destmocks.FakeService{GetOut: dest, ResolveOut: dest, ListOut: []destinations.Destination{dest}}
	inspection := destinationreadiness.Inspection{LocationExists: true, LocationIsDirectory: true, Mounted: true, FreeBytes: 1 << 40, Identity: identity, FilesystemState: sysmounts.FilesystemStateClean}
	readiness := destinationreadiness.NewService(migrationInspector{inspection: inspection}, nil)
	// The exact target IDs are still passed to each estimate; this fake returns
	// one registered filesystem source spec per target lookup.
	return &DBMEffects{Plans: planSvc, Destinations: destSvc, Targets: migrationTargetLookup{targets: targetRows}, Readiness: readiness, Engine: engine}, planSvc, plan, targetIDs
}

type migrationTargetLookup struct {
	targets.Service
	targets map[string]targets.Target
}

func (l migrationTargetLookup) Get(_ context.Context, id string) (targets.Target, error) {
	t, ok := l.targets[id]
	if !ok {
		return targets.Target{}, targets.ErrTargetNotFound{ID: id}
	}
	return t, nil
}
func fmtTarget(i int) string { return fmt.Sprintf("target-%02d", i) }

func TestDBMEffectsProvesExactCapacityAndCredentialIdentity(t *testing.T) {
	effects, _, _, targetIDs := migrationEffectFixture(t)
	proof, err := effects.ProveCapacity(context.Background(), "plan-14", "old", "new", targetIDs, false)
	if err != nil {
		t.Fatal(err)
	}
	if proof.PlanID != "plan-14" || proof.OldDestinationID != "old" || proof.DestinationID != "new" || !equal(canonical(proof.TargetIDs), canonical(targetIDs)) || proof.AvailableBytes <= proof.RequiredBytes || proof.DeviceIdentity != "uuid=volume-uuid;serial=volume-serial" || proof.CredentialAuthority != "vrooli/kopia/backup-offhost:repository-passphrase" || proof.Receipt == "" || proof.ExpiresAt.IsZero() {
		t.Fatalf("capacity receipt is incomplete or not bound: %+v", proof)
	}
	if _, err := effects.ProveCapacity(context.Background(), "plan-14", "old", "new", targetIDs[:RequiredTargetCount-1], false); err == nil {
		t.Fatal("accepted incomplete target set")
	}
}

func TestDBMEffectsOffersOnlyRegisteredReadyFilesystemWithMatchingCredential(t *testing.T) {
	effects, _, plan, targetIDs := migrationEffectFixture(t)
	destinationSvc := effects.Destinations.(*destmocks.FakeService)
	good := destinationSvc.ListOut[0]
	remote := good
	remote.ID, remote.Name, remote.BackendKind = "remote", "remote-store", destinations.BackendS3
	remote.SecretRef = effects.Engine.PassphraseRef(remote.Name)
	noCredential := good
	noCredential.ID, noCredential.Name, noCredential.SecretRef = "no-credential", "unbound", ""
	creating := good
	creating.ID, creating.Name, creating.ProvisioningState = "creating", "not-ready", destinations.ProvisioningCreating
	destinationSvc.ListOut = []destinations.Destination{good, remote, noCredential, creating}
	candidates, err := effects.Candidates(context.Background(), plan.ID, "old", targetIDs)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].ID != "new" {
		t.Fatalf("candidate set included unproven destinations: %+v", candidates)
	}
}

func TestDBMEffectsRegistersOnlyExplicitLocalPathThroughDestinationOwner(t *testing.T) {
	effects, _, plan, targetIDs := migrationEffectFixture(t)
	destinationSvc := effects.Destinations.(*destmocks.FakeService)
	destinationSvc.ListOut = nil
	name := "dbm-fresh-0123456789abcdef0123456789abcdef"
	destinationSvc.CreateOut = destinations.Destination{
		ID: "registered-local", Name: name, BackendKind: destinations.BackendFilesystem,
		Location: "/mnt/recovery", ProvisioningState: destinations.ProvisioningReady,
		SecretRef: effects.Engine.PassphraseRef(name),
	}
	got, err := effects.RegisterLocalDestination(context.Background(), "db4-0123456789abcdef0123456789abcdef", plan.ID, "old", "/mnt/recovery", targetIDs)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "registered-local" || len(destinationSvc.CreateInputs) != 1 {
		t.Fatalf("registration result=%+v create inputs=%+v", got, destinationSvc.CreateInputs)
	}
	input := destinationSvc.CreateInputs[0]
	if input.Backend != destinations.BackendFilesystem || input.Location != "/mnt/recovery" || input.Name != name || input.AcknowledgeReadinessFailure || input.CapBytes != 0 {
		t.Fatalf("destination owner received unbounded or guessed inputs: %+v", input)
	}
	if _, err := effects.RegisterLocalDestination(context.Background(), "db4-0123456789abcdef0123456789abcdef", plan.ID, "old", "s3://bucket/path", targetIDs); err == nil {
		t.Fatal("accepted a provider URI as a local path")
	}
	if len(destinationSvc.CreateInputs) != 1 {
		t.Fatal("invalid provider URI reached destination owner")
	}
}

func TestDBMEffectsSwitchBindingsIsIdempotentAndPreservesOtherDestinations(t *testing.T) {
	effects, planSvc, plan, targetIDs := migrationEffectFixture(t)
	plan.DestinationIDs = []string{"old", "other"}
	planSvc.GetOut = plan
	if err := effects.SwitchBindings(context.Background(), "plan-14", "old", "new", targetIDs, "stable-switch-key"); err != nil {
		t.Fatal(err)
	}
	if len(planSvc.UpdateInputs) != 1 {
		t.Fatalf("plan updates=%d, want one", len(planSvc.UpdateInputs))
	}
	got := planSvc.UpdateInputs[0]
	if equal(got.DestinationIDs, []string{"old", "other"}) || !contains(got.DestinationIDs, "new") || contains(got.DestinationIDs, "old") || !contains(got.DestinationIDs, "other") {
		t.Fatalf("switch binding result = %v", got.DestinationIDs)
	}
	planSvc.GetOut = plans.Plan{ID: "plan-14", TargetIDs: targetIDs, DestinationIDs: []string{"other", "new"}}
	if err := effects.SwitchBindings(context.Background(), "plan-14", "old", "new", targetIDs, "stable-switch-key"); err != nil {
		t.Fatalf("replay after committed switch: %v", err)
	}
	if len(planSvc.UpdateInputs) != 1 {
		t.Fatal("idempotent switch replay wrote plan a second time")
	}
}
