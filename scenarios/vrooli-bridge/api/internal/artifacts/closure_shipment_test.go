package artifacts

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/coreset"
	_ "modernc.org/sqlite"
)

func shipmentFixture(t *testing.T) (ClosureShipmentRequest, []ClosureFile) {
	t.Helper()
	c, err := coreset.Resolve(coreset.DependencyGraph{Components: []coreset.ComponentID{{Kind: "scenario", Name: "demo"}}}, []coreset.ComponentID{{Kind: "scenario", Name: "demo"}})
	require.NoError(t, err)
	b := []byte("selected closure file")
	sum := sha256.Sum256(b)
	files := []ClosureFile{{Path: "scenarios/demo/main.go", SHA256: hex.EncodeToString(sum[:]), Mode: 0o644, Data: b}}
	digest, err := ClosureInventoryDigest(files)
	require.NoError(t, err)
	req := ClosureShipmentRequest{IdempotencyKey: "ship-demo-1", TargetID: "node-test", SourceRef: "bridge-test://closure/1", Destination: "demo", Closure: c, InventoryDigest: digest, PriorRevision: "", NewRevision: "12", Files: files, SelectionReceiptRef: "vrooli-onboarding://handoffs/handoff-1", SelectionDigest: "sha256:selection", DeploymentID: "deployment-1", MachineID: "machine-1", NodeID: "node-test", EnrollmentGeneration: 7, DesiredRevision: 12}
	return req, files
}

func shipmentService(t *testing.T, root string) *ClosureShipmentService {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "bridge.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(schemaSQL)
	require.NoError(t, err)
	repo := NewSQLiteClosureShipmentRepository(db)
	svc := NewClosureShipmentService(repo, DirectoryClosureDestination{Root: root}, func() time.Time { return time.Unix(42, 0).UTC() })
	svc.TargetPolicy = func(context.Context, ClosureShipment) error { return nil }
	svc.AuthorizeSelection = func(context.Context, ClosureShipmentRequest) error { return nil }
	return svc
}

func TestClosureShipmentPersistsAndResumesAfterStageInterruption(t *testing.T) {
	root := t.TempDir()
	unrelated := filepath.Join(root, "unrelated.db")
	require.NoError(t, os.WriteFile(unrelated, []byte("keep"), 0o600))
	svc := shipmentService(t, root)
	req, files := shipmentFixture(t)
	svc.Fault = func(stage string) error {
		if stage == "staged" {
			return errors.New("simulated process interruption")
		}
		return nil
	}
	first, err := svc.Ship(context.Background(), req)
	require.Error(t, err)
	require.Equal(t, ShipmentStaged, first.State)
	require.NotEmpty(t, first.ID)
	svc.Fault = nil
	resumed, err := svc.Resume(context.Background(), first.ID, files)
	require.NoError(t, err)
	require.Equal(t, ShipmentActivated, resumed.State)
	require.Equal(t, req.Closure.Digest, resumed.ClosureDigest)
	loaded, err := svc.repo.GetShipment(context.Background(), first.ID)
	require.NoError(t, err)
	require.Equal(t, ShipmentActivated, loaded.State)
	require.Equal(t, req.InventoryDigest, loaded.InventoryDigest)
	receipt := ClosureNodeReceipt{ShipmentID: first.ID, IdempotencyKey: first.IdempotencyKey, TargetID: first.TargetID, SourceRef: first.SourceRef, Destination: first.Destination, ClosureDigest: first.ClosureDigest, ExpectedInventoryDigest: first.InventoryDigest, ActualInventoryDigest: first.InventoryDigest, PriorRevision: first.PriorRevision, NewRevision: first.NewRevision, ActiveRevision: first.NewRevision, State: ShipmentActivated, SelectionReceiptRef: first.SelectionReceiptRef, SelectionDigest: first.SelectionDigest, DeploymentID: first.DeploymentID, MachineID: first.MachineID, NodeID: first.NodeID, EnrollmentGeneration: first.EnrollmentGeneration, DesiredRevision: first.DesiredRevision, ActualFiles: []StagedFile{{Path: files[0].Path, SHA256: files[0].SHA256, Mode: normalizedMode(files[0].Mode), Size: int64(len(files[0].Data))}}}
	require.NoError(t, svc.RecordNodeReceipt(context.Background(), receipt))
	receipt.ActualFiles = append(receipt.ActualFiles, StagedFile{Path: "extra", SHA256: "bad", Size: 1})
	require.Error(t, svc.RecordNodeReceipt(context.Background(), receipt))
	loaded, err = svc.repo.GetShipment(context.Background(), first.ID)
	require.NoError(t, err)
	require.Len(t, loaded.ActualFiles, 1)
	require.Equal(t, files[0].Path, loaded.ActualFiles[0].Path)
	data, err := os.ReadFile(unrelated)
	require.NoError(t, err)
	require.Equal(t, "keep", string(data))
	active, err := os.ReadFile(filepath.Join(root, ".bridge-closures", "node-test", "demo", "current"))
	require.NoError(t, err)
	require.Equal(t, "12\n", string(active))
	duplicate := req
	duplicate.IdempotencyKey = "ship-demo-replay"
	duplicate.Destination = "other-destination"
	_, err = svc.Prepare(context.Background(), duplicate)
	require.Error(t, err, "one onboarding receipt cannot authorize a second shipment identity")
}

func TestClosureShipmentResumesAfterVerificationAndActivationBoundaries(t *testing.T) {
	for _, boundary := range []string{"verified", "activated"} {
		t.Run(boundary, func(t *testing.T) {
			root := t.TempDir()
			svc := shipmentService(t, root)
			req, files := shipmentFixture(t)
			interrupted := false
			svc.Fault = func(stage string) error {
				if stage == boundary && !interrupted {
					interrupted = true
					return errors.New("simulated process interruption")
				}
				return nil
			}
			first, err := svc.Ship(context.Background(), req)
			require.Error(t, err)
			require.True(t, interrupted)
			if boundary == "verified" {
				require.Equal(t, ShipmentVerified, first.State)
			} else {
				// Activation may have taken effect even though its terminal receipt
				// was lost. The durable intent remains verified and retry must
				// reattach to the target's idempotent activation result.
				require.Equal(t, ShipmentVerified, first.State)
			}
			svc.Fault = nil
			resumed, err := svc.Resume(context.Background(), first.ID, files)
			require.NoError(t, err)
			require.Equal(t, ShipmentActivated, resumed.State)
			require.Equal(t, req.NewRevision, resumed.ActiveRevision)
		})
	}
}

func TestClosureShipmentRevocationDuringRecoveryDiscardsStagedFiles(t *testing.T) {
	root := t.TempDir()
	svc := shipmentService(t, root)
	req, files := shipmentFixture(t)
	svc.Fault = func(stage string) error {
		if stage == "staged" {
			return errors.New("simulated process interruption")
		}
		return nil
	}
	prepared, err := svc.Ship(context.Background(), req)
	require.Error(t, err)
	require.Equal(t, ShipmentStaged, prepared.State)
	svc.Fault = nil
	svc.TargetPolicy = func(context.Context, ClosureShipment) error { return errors.New("target enrollment revoked") }
	recovered, err := svc.Resume(context.Background(), prepared.ID, files)
	require.Error(t, err)
	require.Equal(t, ShipmentRejected, recovered.State)
	_, err = os.Stat(filepath.Join(root, ".bridge-closures", "node-test", "demo", "current"))
	require.Error(t, err)
	_, err = os.Stat(filepath.Join(root, prepared.StageRef))
	require.Error(t, err)
}

func TestClosureShipmentRejectsTamperedClosureAndExtraStagedMember(t *testing.T) {
	root := t.TempDir()
	svc := shipmentService(t, root)
	req, _ := shipmentFixture(t)
	tampered := req
	tampered.Closure.Members = append([]coreset.ClosureMember(nil), req.Closure.Members...)
	tampered.Closure.Members[0].Component.Name = "changed"
	_, err := svc.Ship(context.Background(), tampered)
	require.Error(t, err)
	req.IdempotencyKey = "ship-demo-extra"
	badDest := extraMemberDestination{DirectoryClosureDestination{Root: root}}
	svc.destination = badDest
	got, err := svc.Ship(context.Background(), req)
	require.Error(t, err)
	require.Equal(t, ShipmentRejected, got.State)
	_, err = os.Stat(filepath.Join(root, ".bridge-closures", "node-test", "demo", "current"))
	require.Error(t, err)
}

func TestClosureShipmentActivationFailureRecordsRollback(t *testing.T) {
	root := t.TempDir()
	svc := shipmentService(t, root)
	req, _ := shipmentFixture(t)
	svc.destination = activationFailureDestination{DirectoryClosureDestination{Root: root}}
	got, err := svc.Ship(context.Background(), req)
	require.Error(t, err)
	require.Equal(t, ShipmentRolledBack, got.State)
	_, err = os.Stat(filepath.Join(root, ".bridge-closures", "node-test", "demo", "current"))
	require.Error(t, err)
}

func TestClosureShipmentTerminalReceiptsReattachAfterRepositoryRestart(t *testing.T) {
	for _, tc := range []struct {
		name string
		dest func(string) ClosureDestination
		want ClosureShipmentState
	}{
		{name: "activated", dest: func(root string) ClosureDestination { return DirectoryClosureDestination{Root: root} }, want: ShipmentActivated},
		{name: "rolled back", dest: func(root string) ClosureDestination {
			return activationFailureDestination{DirectoryClosureDestination{Root: root}}
		}, want: ShipmentRolledBack},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			dbPath := filepath.Join(t.TempDir(), "bridge.db")
			db, err := sql.Open("sqlite", dbPath)
			require.NoError(t, err)
			_, err = db.Exec(schemaSQL)
			require.NoError(t, err)
			service := NewClosureShipmentService(NewSQLiteClosureShipmentRepository(db), tc.dest(root), func() time.Time { return time.Unix(84, 0).UTC() })
			service.TargetPolicy = func(context.Context, ClosureShipment) error { return nil }
			service.AuthorizeSelection = func(context.Context, ClosureShipmentRequest) error { return nil }
			req, _ := shipmentFixture(t)
			shipment, shipErr := service.Ship(context.Background(), req)
			if tc.want == ShipmentActivated {
				require.NoError(t, shipErr)
			} else {
				require.Error(t, shipErr)
			}
			require.Equal(t, tc.want, shipment.State)
			require.NoError(t, db.Close())

			// A new repository/service instance reattaches the same durable receipt.
			restartedDB, err := sql.Open("sqlite", dbPath)
			require.NoError(t, err)
			defer restartedDB.Close()
			restarted := NewClosureShipmentService(NewSQLiteClosureShipmentRepository(restartedDB), nil, nil)
			loaded, err := restarted.Get(context.Background(), shipment.ID)
			require.NoError(t, err)
			require.Equal(t, tc.want, loaded.State)
			require.Equal(t, req.Closure.Digest, loaded.ClosureDigest)
			require.Equal(t, req.TargetID, loaded.TargetID)
			require.Equal(t, req.EnrollmentGeneration, loaded.EnrollmentGeneration)
			if tc.want == ShipmentActivated {
				require.Equal(t, req.NewRevision, loaded.ActiveRevision)
				require.Equal(t, req.InventoryDigest, loaded.ActualInventoryDigest)
				require.NotEmpty(t, loaded.ActualFiles)
			} else {
				require.Equal(t, req.PriorRevision, loaded.ActiveRevision)
				require.Equal(t, req.InventoryDigest, loaded.ActualInventoryDigest)
				require.NotEmpty(t, loaded.ActualFiles)
			}

			changed := req
			changed.SourceRef = "bridge-test://closure/changed"
			_, err = restarted.Prepare(context.Background(), changed)
			require.ErrorContains(t, err, "idempotency key already belongs")

			recorder := &closureFrameRecorder{}
			sourceOpened := false
			coordinator := NewClosureShipmentCoordinator(restarted, recorder, func(context.Context, string) (io.ReadCloser, error) {
				sourceOpened = true
				return nil, errors.New("terminal replay must not reopen source")
			})
			replayed, err := coordinator.Start(context.Background(), req)
			require.NoError(t, err)
			require.Equal(t, tc.want, replayed.State)
			require.False(t, sourceOpened)
			require.Empty(t, recorder.commands)
		})
	}
}

func TestClosureShipmentRejectsStaleAndRevokedBeforeStaging(t *testing.T) {
	for _, tc := range []struct {
		name      string
		configure func(*ClosureShipmentService, *ClosureShipmentRequest)
	}{
		{name: "stale", configure: func(_ *ClosureShipmentService, r *ClosureShipmentRequest) { r.PriorRevision = "old-revision" }},
		{name: "revoked", configure: func(s *ClosureShipmentService, _ *ClosureShipmentRequest) {
			s.TargetPolicy = func(context.Context, ClosureShipment) error { return errors.New("target revoked") }
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			svc := shipmentService(t, root)
			req, _ := shipmentFixture(t)
			tc.configure(svc, &req)
			got, err := svc.Ship(context.Background(), req)
			require.Error(t, err)
			require.Equal(t, ShipmentRejected, got.State)
			_, err = os.Stat(filepath.Join(root, ".bridge-closures", "node-test", "demo", "revisions"))
			require.Error(t, err)
		})
	}
}

type extraMemberDestination struct{ DirectoryClosureDestination }

func (d extraMemberDestination) Inspect(ctx context.Context, ref string) ([]StagedFile, error) {
	files, err := d.DirectoryClosureDestination.Inspect(ctx, ref)
	if err == nil {
		files = append(files, StagedFile{Path: "extra", SHA256: "deadbeef", Size: 1})
	}
	return files, err
}

type activationFailureDestination struct{ DirectoryClosureDestination }

func (d activationFailureDestination) Activate(context.Context, ClosureShipment, string) error {
	return errors.New("simulated activation interruption")
}
