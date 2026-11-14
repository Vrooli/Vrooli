package notes

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	aisearch "github.com/vrooli/ai-go/search"
)

type FleetMigrationRequest struct {
	ArmID     string
	PeakBytes int64
	Config    FleetGenerationConfig
	Source    NotesFleetSourceContract
}

type FleetMigrationValidation struct {
	GenerationID  string
	OwnerID       string
	Alias         string
	Collection    string
	RecipeID      string
	EmbeddingRole string
	Dimensions    int
	SnapshotID    string
	Watermark     string
	PointCount    uint64
	ReceiptDigest string
}

type PreparedFleetMigration struct {
	ArmID         string
	Config        FleetGenerationConfig
	Snapshot      FleetSnapshot
	Watermark     string
	PointCount    uint64
	ReceiptDigest string
	AdmissionIDs  []string
}

// NotesFleetMigrationAdapter is implemented by the shared, admitted fleet
// owner. It owns durable operation state and physical vector-store actions;
// this Notes coordinator only supplies the source, identity, fence, and gates.
type NotesFleetMigrationAdapter interface {
	Stage(context.Context, FleetMigrationRequest, FleetSnapshot) error
	CatchUp(context.Context, FleetMigrationRequest, FleetSnapshot, string) (string, error)
	Validate(context.Context, FleetMigrationRequest, FleetSnapshot, string, uint64) (FleetMigrationValidation, error)
	Promote(context.Context, FleetMigrationRequest, FleetMigrationValidation) error
	Rollback(context.Context, string) error
	Recover(context.Context, string) error
}

type FleetAdmissionLease interface {
	aisearch.FleetAdmissionScope
	ReceiptID() string
	Release(context.Context) error
}

// FleetOperationAdmission is the only route to physical fleet work. A stage
// request requires a positive peak-byte estimate; every later phase obtains
// its own operation lease and receipt.
type FleetOperationAdmission interface {
	Acquire(context.Context, string, string, []string, int64) (FleetAdmissionLease, error)
	// Reconcile releases an arm's retained leases only after the provider has
	// proven that the interrupted physical effect is resolved.
	Reconcile(context.Context, string) error
}

type NotesFleetMigrationCoordinator struct {
	Adapter   NotesFleetMigrationAdapter
	Admission FleetOperationAdmission
}

const fleetAdmissionReleaseTimeout = 5 * time.Second

func fleetLeaseReleaseContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), fleetAdmissionReleaseTimeout)
}

func (c NotesFleetMigrationCoordinator) validateRequest(request FleetMigrationRequest) error {
	if c.Adapter == nil || c.Admission == nil || request.Source == nil {
		return errors.New("Notes migration adapter, operation admission, and source are required")
	}
	if strings.TrimSpace(request.ArmID) == "" || request.Config.GenerationID != request.ArmID || request.PeakBytes <= 0 {
		return errors.New("Notes migration arm and positive peak-byte estimate are required")
	}
	config := request.Config
	if strings.TrimSpace(config.GenerationID) == "" || strings.TrimSpace(config.OwnerID) == "" || strings.TrimSpace(config.Alias) == "" || strings.TrimSpace(config.Collection) == "" || strings.TrimSpace(config.RecipeID) == "" || strings.TrimSpace(config.EmbeddingRole) == "" || config.EmbeddingDimensions <= 0 {
		return errors.New("Notes migration requires explicit generation, owner, alias, collection, recipe, role, and dimensions")
	}
	return nil
}

func (c NotesFleetMigrationCoordinator) withAdmission(ctx context.Context, operation, armID string, targetIDs []string, peakBytes int64, action func(context.Context, FleetAdmissionLease) error) (string, error) {
	if peakBytes <= 0 || len(targetIDs) == 0 {
		return "", fmt.Errorf("%s requires a positive provider peak-byte estimate and explicit target collection scope", operation)
	}
	lease, err := c.Admission.Acquire(ctx, operation, armID, append([]string(nil), targetIDs...), peakBytes)
	if err != nil {
		return "", err
	}
	if lease == nil || strings.TrimSpace(lease.ReceiptID()) == "" {
		if lease != nil {
			releaseCtx, cancel := fleetLeaseReleaseContext(ctx)
			defer cancel()
			if releaseErr := lease.Release(releaseCtx); releaseErr != nil {
				return lease.ReceiptID(), fmt.Errorf("%s admission returned no receipt; release lease: %w", operation, releaseErr)
			}
		}
		return "", errors.New("Notes fleet operation admission returned no durable receipt")
	}
	operationCtx, err := aisearch.WithFleetAdmissionScope(ctx, lease)
	if err != nil {
		releaseCtx, cancel := fleetLeaseReleaseContext(ctx)
		defer cancel()
		return lease.ReceiptID(), errors.Join(err, lease.Release(releaseCtx))
	}
	result := action(operationCtx, lease)
	if result != nil {
		// The provider may have completed some of the physical effect before
		// returning an error (or losing its response). Keep the durable capacity
		// reservation live so the owner can reconcile the same arm before it is
		// made available to unrelated work. Recovery has its own admitted lease.
		return lease.ReceiptID(), fmt.Errorf("%s failed; admission retained for provider recovery: %w", operation, result)
	}
	releaseCtx, cancel := fleetLeaseReleaseContext(ctx)
	defer cancel()
	releaseErr := lease.Release(releaseCtx)
	if releaseErr != nil {
		return lease.ReceiptID(), fmt.Errorf("release %s admission: %w", operation, releaseErr)
	}
	return lease.ReceiptID(), nil
}

// Prepare pins a source snapshot, stages the candidate, catches it up until a
// stable watermark is observed, and validates an exact point-count receipt.
// No alias change happens here.
func (c NotesFleetMigrationCoordinator) Prepare(ctx context.Context, request FleetMigrationRequest) (PreparedFleetMigration, error) {
	if err := c.validateRequest(request); err != nil {
		return PreparedFleetMigration{}, err
	}
	prepared := PreparedFleetMigration{ArmID: request.ArmID, Config: request.Config}
	stageReceipt, err := c.withAdmission(ctx, "fleet-stage", request.ArmID, []string{request.Config.Collection}, request.PeakBytes, func(operationCtx context.Context, lease FleetAdmissionLease) error {
		snapshot, err := request.Source.PinFleetSnapshot(operationCtx, "notes-"+request.ArmID)
		if err != nil {
			return err
		}
		if strings.TrimSpace(snapshot.ID) == "" || strings.TrimSpace(snapshot.Watermark) == "" {
			return errors.New("Notes source returned an incomplete pinned snapshot")
		}
		prepared.Snapshot = snapshot
		return c.Adapter.Stage(operationCtx, request, snapshot)
	})
	if err != nil {
		return PreparedFleetMigration{}, err
	}
	prepared.AdmissionIDs = append(prepared.AdmissionIDs, stageReceipt)
	if prepared.Snapshot.ID == "" {
		return PreparedFleetMigration{}, errors.New("Notes source did not pin a snapshot")
	}

	for {
		target, err := request.Source.FleetWatermark(ctx)
		if err != nil {
			return PreparedFleetMigration{}, err
		}
		var caught string
		receipt, err := c.withAdmission(ctx, "fleet-catch-up", request.ArmID, []string{request.Config.Collection}, request.PeakBytes, func(operationCtx context.Context, lease FleetAdmissionLease) error {
			var catchErr error
			caught, catchErr = c.Adapter.CatchUp(operationCtx, request, prepared.Snapshot, target)
			return catchErr
		})
		if err != nil {
			return PreparedFleetMigration{}, err
		}
		prepared.AdmissionIDs = append(prepared.AdmissionIDs, receipt)
		order, err := request.Source.CompareFleetSequence(ctx, caught, target)
		if err != nil || order != 0 {
			return PreparedFleetMigration{}, errors.New("Notes migration candidate did not catch up exactly to its requested source watermark")
		}
		latest, err := request.Source.FleetWatermark(ctx)
		if err != nil {
			return PreparedFleetMigration{}, err
		}
		order, err = request.Source.CompareFleetSequence(ctx, latest, target)
		if err != nil || order < 0 {
			return PreparedFleetMigration{}, errors.New("Notes source watermark moved backwards during candidate catch-up")
		}
		if order == 0 {
			prepared.Watermark = target
			break
		}
	}

	prepared.PointCount, err = request.Source.ExpectedFleetPointCount(ctx, request.Config.RecipeID, prepared.Snapshot.ID, prepared.Watermark)
	if err != nil {
		return PreparedFleetMigration{}, err
	}
	var validation FleetMigrationValidation
	receipt, err := c.withAdmission(ctx, "fleet-validate", request.ArmID, []string{request.Config.Collection}, request.PeakBytes, func(operationCtx context.Context, lease FleetAdmissionLease) error {
		var validateErr error
		validation, validateErr = c.Adapter.Validate(operationCtx, request, prepared.Snapshot, prepared.Watermark, prepared.PointCount)
		return validateErr
	})
	if err != nil {
		return PreparedFleetMigration{}, err
	}
	prepared.AdmissionIDs = append(prepared.AdmissionIDs, receipt)
	if validation.GenerationID != request.Config.GenerationID || validation.OwnerID != request.Config.OwnerID || validation.Alias != request.Config.Alias || validation.Collection != request.Config.Collection || validation.RecipeID != request.Config.RecipeID || validation.EmbeddingRole != request.Config.EmbeddingRole || validation.Dimensions != request.Config.EmbeddingDimensions || validation.SnapshotID != prepared.Snapshot.ID || validation.Watermark != prepared.Watermark || validation.PointCount != prepared.PointCount || strings.TrimSpace(validation.ReceiptDigest) == "" {
		return PreparedFleetMigration{}, errors.New("Notes migration validation receipt does not bind owner, generation, encoder, snapshot, watermark, and exact point count")
	}
	prepared.ReceiptDigest = validation.ReceiptDigest
	return prepared, nil
}

func (c NotesFleetMigrationCoordinator) withSourceFence(ctx context.Context, operation, armID string, targetIDs []string, peakBytes int64, source NotesFleetSourceContract, expected string, action func(context.Context, FleetAdmissionLease) error) error {
	if source == nil {
		return errors.New("Notes source is required for fenced fleet operation")
	}
	if expected == "" {
		var err error
		expected, err = source.FleetWatermark(ctx)
		if err != nil {
			return err
		}
	}
	_, err := c.withAdmission(ctx, operation, armID, targetIDs, peakBytes, func(operationCtx context.Context, lease FleetAdmissionLease) error {
		fence, err := source.AcquireFleetCutover(operationCtx, expected)
		if err != nil {
			return err
		}
		current, actionErr := fence.CurrentWatermark(operationCtx)
		if actionErr == nil && current != expected {
			actionErr = errors.New("Notes source changed while the fleet operation fence was held")
		}
		if actionErr == nil {
			actionErr = action(operationCtx, lease)
		}
		releaseCtx, cancel := fleetLeaseReleaseContext(ctx)
		defer cancel()
		releaseErr := fence.Release(releaseCtx)
		if actionErr != nil {
			if releaseErr != nil {
				return fmt.Errorf("%s failed: %w; release source fence: %v", operation, actionErr, releaseErr)
			}
			return actionErr
		}
		if releaseErr != nil {
			return fmt.Errorf("release Notes source fence after %s: %w", operation, releaseErr)
		}
		return nil
	})
	return err
}

// Commit takes the Notes source fence across the owner's atomic alias switch.
// A stale watermark causes a refusal; prepare and catch-up must then repeat.
func (c NotesFleetMigrationCoordinator) Commit(ctx context.Context, request FleetMigrationRequest, prepared PreparedFleetMigration) error {
	if err := c.validateRequest(request); err != nil {
		return err
	}
	if prepared.ArmID != request.ArmID || prepared.Config != request.Config || prepared.Watermark == "" || prepared.ReceiptDigest == "" {
		return errors.New("Notes commit requires a validation receipt for the same arm and generation")
	}
	validation := FleetMigrationValidation{GenerationID: prepared.Config.GenerationID, OwnerID: prepared.Config.OwnerID, Alias: prepared.Config.Alias, Collection: prepared.Config.Collection, RecipeID: prepared.Config.RecipeID, EmbeddingRole: prepared.Config.EmbeddingRole, Dimensions: prepared.Config.EmbeddingDimensions, SnapshotID: prepared.Snapshot.ID, Watermark: prepared.Watermark, PointCount: prepared.PointCount, ReceiptDigest: prepared.ReceiptDigest}
	return c.withSourceFence(ctx, "fleet-promote", request.ArmID, []string{request.Config.Collection}, request.PeakBytes, request.Source, prepared.Watermark, func(operationCtx context.Context, lease FleetAdmissionLease) error {
		return c.Adapter.Promote(operationCtx, request, validation)
	})
}

func (c NotesFleetMigrationCoordinator) Rollback(ctx context.Context, armID string, targetCollections []string, peakBytes int64, source NotesFleetSourceContract) error {
	if c.Adapter == nil || c.Admission == nil || strings.TrimSpace(armID) == "" || len(targetCollections) == 0 || peakBytes <= 0 || source == nil {
		return errors.New("Notes rollback requires migration owner, operation admission, source, arm identity, target collection scope, and positive peak-byte estimate")
	}
	return c.withSourceFence(ctx, "fleet-rollback", armID, targetCollections, peakBytes, source, "", func(operationCtx context.Context, _ FleetAdmissionLease) error {
		return c.Adapter.Rollback(operationCtx, armID)
	})
}

func (c NotesFleetMigrationCoordinator) Recover(ctx context.Context, armID string, targetCollections []string, peakBytes int64, source NotesFleetSourceContract) error {
	if c.Adapter == nil || c.Admission == nil || strings.TrimSpace(armID) == "" || len(targetCollections) == 0 || peakBytes <= 0 || source == nil {
		return errors.New("Notes recovery requires migration owner, operation admission, source, arm identity, target collection scope, and positive peak-byte estimate")
	}
	if err := c.withSourceFence(ctx, "fleet-recover", armID, targetCollections, peakBytes, source, "", func(operationCtx context.Context, _ FleetAdmissionLease) error {
		return c.Adapter.Recover(operationCtx, armID)
	}); err != nil {
		return err
	}
	if err := c.Admission.Reconcile(ctx, armID); err != nil {
		return fmt.Errorf("reconcile retained Notes fleet capacity leases after provider recovery: %w", err)
	}
	return nil
}
