package repositorymigration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"data-backup-manager/internal/destinationreadiness"
	destinations "data-backup-manager/internal/destinations"
	"data-backup-manager/internal/drills"
	"data-backup-manager/internal/engine"
	"data-backup-manager/internal/plans"
	"data-backup-manager/internal/runs"
	"data-backup-manager/internal/sources"
	"data-backup-manager/internal/targets"
)

type MigrationRuns interface {
	CaptureMigrationTarget(context.Context, string, string, string, string) (runs.TargetOutcome, error)
}
type MigrationDrills interface {
	RunMigrationSnapshot(context.Context, string, string, string, string, string) (drills.Drill, error)
	Get(context.Context, string) (drills.Drill, error)
}

// DBMEffects composes DBM's existing target, destination, run, drill and plan
// owners into the repositorymigration contract. No effect uses the damaged
// repository; the destination ID is explicit on every capture and restore.
type DBMEffects struct {
	Plans        plans.Service
	Destinations destinations.Service
	Targets      targets.Service
	Readiness    *destinationreadiness.Service
	Engine       engine.KopiaEngine
	Runs         MigrationRuns
	Drills       MigrationDrills
	Now          func() time.Time
	DrillTimeout time.Duration
	PollInterval time.Duration
}

func (e *DBMEffects) now() time.Time {
	if e.Now != nil {
		return e.Now().UTC()
	}
	return time.Now().UTC()
}

// PlanForIncident accepts only one exact 14-target plan that currently binds
// the damaged repository. Ambiguous plan membership is never guessed.
func (e *DBMEffects) PlanForIncident(ctx context.Context, planIDs []string, oldDestinationID string) (string, []string, error) {
	var selected plans.Plan
	for _, id := range planIDs {
		p, err := e.Plans.Get(ctx, id)
		if err != nil {
			return "", nil, fmt.Errorf("resolve affected plan %s: %w", id, err)
		}
		if len(p.TargetIDs) != RequiredTargetCount || !contains(p.DestinationIDs, oldDestinationID) {
			continue
		}
		if selected.ID != "" {
			return "", nil, errors.New("more than one affected plan has exactly 14 targets; migration plan must be selected explicitly")
		}
		selected = p
	}
	if selected.ID == "" {
		return "", nil, errors.New("no affected 14-target plan is available for fresh migration")
	}
	targets := canonical(selected.TargetIDs)
	if !unique(targets) {
		return "", nil, errors.New("source plan target set is not 14 unique registered IDs")
	}
	return selected.ID, targets, nil
}

func (e *DBMEffects) Candidates(ctx context.Context, planID, oldDestinationID string, targetIDs []string) ([]destinations.Destination, error) {
	if e.Plans == nil || e.Destinations == nil || e.Targets == nil || e.Readiness == nil || e.Engine == nil {
		return nil, errors.New("migration eligibility owners are not configured")
	}
	plan, err := e.Plans.Get(ctx, planID)
	if err != nil {
		return nil, err
	}
	if !equal(canonical(plan.TargetIDs), canonical(targetIDs)) || !contains(plan.DestinationIDs, oldDestinationID) {
		return nil, errors.New("current plan no longer matches the migration's exact source set")
	}
	requiredBytes, err := e.requiredBytes(ctx, plan, targetIDs)
	if err != nil {
		return nil, err
	}
	retentionCopies := retentionCopies(plan)
	neededBytes, err := multiplyBytes(requiredBytes, retentionCopies)
	if err != nil {
		return nil, err
	}
	list, err := e.Destinations.ListDestinations(ctx, 200)
	if err != nil {
		return nil, err
	}
	out := make([]destinations.Destination, 0, len(list))
	for _, candidate := range list {
		if candidate.ID == oldDestinationID || contains(plan.DestinationIDs, candidate.ID) || candidate.BackendKind != destinations.BackendFilesystem || candidate.ProvisioningState != destinations.ProvisioningReady || candidate.DeviceIdentity == nil || !candidate.DeviceIdentity.StrongIdentity() || strings.TrimSpace(candidate.SecretRef) == "" || candidate.SecretRef != e.Engine.PassphraseRef(candidate.Name) {
			continue
		}
		resolved, resolveErr := e.Destinations.ResolveForUse(ctx, candidate.ID)
		if resolveErr != nil || resolved.DeviceIdentity == nil || !candidate.DeviceIdentity.MatchesDevice(*resolved.DeviceIdentity) {
			continue
		}
		obs, observeErr := e.Readiness.ObserveCapacity(ctx, destinationreadiness.AnalyzeInput{
			DestinationID: resolved.ID, Location: resolved.RepositoryLocation, ExpectedDevice: *candidate.DeviceIdentity,
			SelectedTargetBytes: requiredBytes, RetentionCopies: retentionCopies,
		})
		if observeErr != nil || obs.FreeBytes < neededBytes {
			continue
		}
		out = append(out, resolved)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// RegisterLocalDestination delegates a verified, operator-supplied filesystem
// location to DBM's destination owner. The deterministic repository name and
// credential reference are owner-managed; callers cannot supply provider or
// secret values. The owner's readiness gate runs before repository creation.
func (e *DBMEffects) RegisterLocalDestination(ctx context.Context, migrationID, planID, oldDestinationID, location string, targetIDs []string) (destinations.Destination, error) {
	if e.Plans == nil || e.Destinations == nil || e.Engine == nil || !validMigrationID(migrationID) {
		return destinations.Destination{}, errors.New("destination owner or migration identity is invalid")
	}
	plan, err := e.Plans.Get(ctx, planID)
	if err != nil {
		return destinations.Destination{}, err
	}
	if plan.ID != planID || !contains(plan.DestinationIDs, oldDestinationID) || len(targetIDs) != RequiredTargetCount || !equal(canonical(plan.TargetIDs), canonical(targetIDs)) || !unique(canonical(targetIDs)) {
		return destinations.Destination{}, errors.New("local destination registration no longer matches the exact 14-target source plan")
	}
	location = strings.TrimSpace(location)
	if location == "" || len(location) > 512 || !filepath.IsAbs(location) || strings.Contains(location, "://") || strings.ContainsAny(location, "\x00\r\n\t") {
		return destinations.Destination{}, errors.New("local destination must be an absolute filesystem path without a provider URI")
	}
	location = filepath.Clean(location)
	name := "dbm-fresh-" + strings.TrimPrefix(migrationID, "db4-")
	rows, err := e.Destinations.ListDestinations(ctx, 200)
	if err != nil {
		return destinations.Destination{}, err
	}
	if len(rows) == 200 {
		return destinations.Destination{}, errors.New("destination catalog is at the bounded lookup limit; cannot reconcile a new local destination name safely")
	}
	for _, row := range rows {
		if row.Name != name {
			continue
		}
		if row.BackendKind != destinations.BackendFilesystem || filepath.Clean(row.Location) != location {
			return destinations.Destination{}, errors.New("deterministic destination name is already registered for another backend or location")
		}
		if row.ProvisioningState == destinations.ProvisioningReady {
			if row.SecretRef != e.Engine.PassphraseRef(row.Name) {
				return destinations.Destination{}, errors.New("existing deterministic destination credential identity does not match resource-kopia")
			}
			return row, nil
		}
		if row.ProvisioningState != destinations.ProvisioningCreated {
			return destinations.Destination{}, fmt.Errorf("destination owner has a pending provisioning record (%s); reconcile it through the destination owner before retrying", row.ProvisioningState)
		}
		break // Let CreateDestination reconcile a previously confirmed RepoCreate.
	}
	created, err := e.Destinations.CreateDestination(ctx, destinations.CreateInput{
		Name: name, Backend: destinations.BackendFilesystem, Location: location,
		CapPolicy: destinations.CapPolicyAlertBlock,
	})
	if err != nil {
		return destinations.Destination{}, err
	}
	if created.ID == "" || created.Name != name || created.BackendKind != destinations.BackendFilesystem || filepath.Clean(created.Location) != location || created.ProvisioningState != destinations.ProvisioningReady || created.SecretRef == "" || created.SecretRef != e.Engine.PassphraseRef(created.Name) {
		return destinations.Destination{}, errors.New("destination owner did not return a ready repository with its managed credential identity")
	}
	return created, nil
}

func validMigrationID(id string) bool {
	if !strings.HasPrefix(id, "db4-") || len(id) != len("db4-")+32 {
		return false
	}
	for _, r := range strings.TrimPrefix(id, "db4-") {
		if !(r >= '0' && r <= '9') && !(r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

func (e *DBMEffects) ProveCapacity(ctx context.Context, planID, oldDestinationID, destinationID string, targetIDs []string, allowCommittedSwitch bool) (CapacityProof, error) {
	if e.Readiness == nil || e.Destinations == nil || e.Engine == nil {
		return CapacityProof{}, errors.New("capacity or credential owner is unavailable")
	}
	plan, err := e.Plans.Get(ctx, planID)
	if err != nil {
		return CapacityProof{}, err
	}
	if !equal(canonical(plan.TargetIDs), canonical(targetIDs)) {
		return CapacityProof{}, errors.New("capacity target set differs from the exact plan target set")
	}
	oldBound, newBound := contains(plan.DestinationIDs, oldDestinationID), contains(plan.DestinationIDs, destinationID)
	if (!allowCommittedSwitch && (!oldBound || newBound)) || (allowCommittedSwitch && !oldBound && !newBound) {
		return CapacityProof{}, errors.New("plan bindings changed outside the expected migration switch state")
	}
	// The selected destination is intentionally not yet a plan member. Its
	// registration and readiness are independently re-proven below.
	dest, err := e.Destinations.ResolveForUse(ctx, destinationID)
	if err != nil {
		return CapacityProof{}, err
	}
	if dest.BackendKind != destinations.BackendFilesystem || dest.ProvisioningState != destinations.ProvisioningReady || dest.DeviceIdentity == nil || !dest.DeviceIdentity.StrongIdentity() {
		return CapacityProof{}, errors.New("selected destination is not a ready identity-bound filesystem destination")
	}
	credential := strings.TrimSpace(dest.SecretRef)
	if credential == "" || credential != e.Engine.PassphraseRef(dest.Name) {
		return CapacityProof{}, errors.New("selected destination credential authority does not match resource-kopia")
	}
	required, err := e.requiredBytes(ctx, plan, targetIDs)
	if err != nil {
		return CapacityProof{}, err
	}
	copies := retentionCopies(plan)
	required, err = multiplyBytes(required, copies)
	if err != nil {
		return CapacityProof{}, err
	}
	obs, err := e.Readiness.ObserveCapacity(ctx, destinationreadiness.AnalyzeInput{
		DestinationID: dest.ID, Location: dest.RepositoryLocation, ExpectedDevice: *dest.DeviceIdentity,
		SelectedTargetBytes: required, RetentionCopies: 1,
	})
	if err != nil {
		return CapacityProof{}, err
	}
	if obs.FreeBytes < required {
		return CapacityProof{}, fmt.Errorf("free capacity %d is below exact target-set estimate %d", obs.FreeBytes, required)
	}
	device := fmt.Sprintf("uuid=%s;serial=%s", strings.TrimSpace(obs.Identity.UUID), strings.TrimSpace(obs.Identity.Serial))
	measured := obs.MeasuredAt.UTC()
	proof := CapacityProof{PlanID: planID, OldDestinationID: oldDestinationID, DestinationID: destinationID, TargetIDs: canonical(targetIDs), AvailableBytes: obs.FreeBytes, RequiredBytes: required, DeviceIdentity: device, CredentialAuthority: credential, MeasuredAt: measured, ExpiresAt: measured.Add(10 * time.Minute)}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00%s\x00%d\x00%d\x00%s", planID, oldDestinationID, destinationID, strings.Join(proof.TargetIDs, "\x00"), device, obs.Location, obs.MountEpisodeID, proof.AvailableBytes, proof.RequiredBytes, measured.Format(time.RFC3339Nano)+"\x00"+credential)))
	proof.Receipt = "dbm-capacity-" + hex.EncodeToString(sum[:16])
	return proof, nil
}

func (e *DBMEffects) Capture(ctx context.Context, planID, targetID, destinationID, key string) (CaptureReceipt, error) {
	if e.Runs == nil {
		return CaptureReceipt{}, errors.New("DBM run owner is unavailable")
	}
	out, err := e.Runs.CaptureMigrationTarget(ctx, planID, targetID, destinationID, key)
	if err != nil {
		return CaptureReceipt{}, err
	}
	if out.Status != runs.OutcomeSucceeded {
		return CaptureReceipt{}, fmt.Errorf("DBM migration capture ended %s", out.Status)
	}
	return CaptureReceipt{TargetID: targetID, DestinationID: destinationID, SnapshotID: out.SnapshotID, Bytes: out.Bytes, Verified: out.SnapshotID != ""}, nil
}

func (e *DBMEffects) Drill(ctx context.Context, planID, targetID, destinationID, snapshotID, key string) (DrillReceipt, error) {
	if e.Drills == nil {
		return DrillReceipt{}, errors.New("DBM verified restore drill owner is unavailable")
	}
	drill, err := e.Drills.RunMigrationSnapshot(ctx, planID, targetID, destinationID, snapshotID, key)
	if err != nil {
		return DrillReceipt{}, err
	}
	if drill.Status == drills.StatusVerified {
		return migrationDrillReceipt(drill), nil
	}
	if e.DrillTimeout <= 0 {
		e.DrillTimeout = 30 * time.Minute
	}
	if e.PollInterval <= 0 {
		e.PollInterval = time.Second
	}
	deadline := time.NewTimer(e.DrillTimeout)
	defer deadline.Stop()
	tick := time.NewTicker(e.PollInterval)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return DrillReceipt{}, ctx.Err()
		case <-deadline.C:
			return DrillReceipt{}, errors.New("DBM migration scratch drill timed out")
		case <-tick.C:
			drill, err = e.Drills.Get(ctx, drill.ID)
			if err != nil {
				return DrillReceipt{}, err
			}
			if drill.Status == drills.StatusVerified {
				return migrationDrillReceipt(drill), nil
			}
			if drill.Status == drills.StatusFailed {
				return DrillReceipt{}, fmt.Errorf("DBM migration scratch drill failed: %s", drill.Error)
			}
		}
	}
}

func migrationDrillReceipt(d drills.Drill) DrillReceipt {
	return DrillReceipt{TargetID: d.TargetID, DestinationID: d.DestinationID, SnapshotID: d.SnapshotID, RestoreID: d.RestoreID, ScratchReceiptID: "dbm-verified-scratch-restore:" + d.RestoreID, Verified: d.Status == drills.StatusVerified}
}

func (e *DBMEffects) SwitchBindings(ctx context.Context, planID, oldDestinationID, newDestinationID string, targetIDs []string, switchKey string) error {
	if strings.TrimSpace(switchKey) == "" {
		return errors.New("plan switch idempotency key is required")
	}
	p, err := e.Plans.Get(ctx, planID)
	if err != nil {
		return err
	}
	if !equal(canonical(p.TargetIDs), canonical(targetIDs)) {
		return errors.New("plan target set changed before binding switch")
	}
	hasOld, hasNew := contains(p.DestinationIDs, oldDestinationID), contains(p.DestinationIDs, newDestinationID)
	if !hasOld && hasNew {
		return nil
	}
	if !hasOld {
		return errors.New("source plan no longer binds the old destination")
	}
	next := make([]string, 0, len(p.DestinationIDs)+1)
	for _, id := range p.DestinationIDs {
		if id != oldDestinationID {
			next = append(next, id)
		}
	}
	if !hasNew {
		next = append(next, newDestinationID)
	}
	_, err = e.Plans.Update(ctx, plans.UpdateInput{ID: p.ID, Name: p.Name, TargetIDs: p.TargetIDs, DestinationIDs: next, Schedule: p.Schedule, KeepLatest: p.KeepLatest, KeepHourly: p.KeepHourly, KeepDaily: p.KeepDaily, KeepWeekly: p.KeepWeekly, KeepMonthly: p.KeepMonthly, KeepAnnual: p.KeepAnnual, Enabled: p.Enabled, ProtectionTier: p.ProtectionTier, RecoveryDrillSchedule: p.RecoveryDrillSchedule, AllowIncompleteCoverage: true})
	return err
}

func (e *DBMEffects) requiredBytes(ctx context.Context, plan plans.Plan, targetIDs []string) (int64, error) {
	var total int64
	for _, id := range targetIDs {
		t, err := e.Targets.Get(ctx, id)
		if err != nil {
			return 0, fmt.Errorf("resolve target %s: %w", id, err)
		}
		if t.SourceKind != sources.KindFilesystem && t.SourceKind != sources.KindSQLite {
			return 0, fmt.Errorf("target %s has no complete read-only size estimator", id)
		}
		bytes, supported, err := sources.EstimateLocalSourceBytes(t.SourceKind, t.Locator)
		if err != nil || !supported {
			return 0, fmt.Errorf("target %s capacity estimate is unknown: %v", id, err)
		}
		if bytes < 0 || total > int64(^uint64(0)>>1)-bytes {
			return 0, errors.New("target capacity estimate overflows")
		}
		total += bytes
	}
	if total <= 0 {
		total = 1
	}
	return total, nil
}

func retentionCopies(p plans.Plan) int {
	n := int(p.KeepLatest + p.KeepHourly + p.KeepDaily + p.KeepWeekly + p.KeepMonthly + p.KeepAnnual)
	if n < 1 {
		return 1
	}
	return n
}

func multiplyBytes(bytes int64, copies int) (int64, error) {
	if bytes <= 0 || copies < 1 || bytes > int64(^uint64(0)>>1)/int64(copies) {
		return 0, errors.New("capacity estimate is unknown or overflows")
	}
	return bytes * int64(copies), nil
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

var _ Effects = (*DBMEffects)(nil)
