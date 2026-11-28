package artifacts

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"io"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/coreset"
	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	_ "modernc.org/sqlite"
)

type closureFrameRecorder struct {
	target   string
	commands []*channelv1.ClosureShipmentCommand
}

func openClosureShipmentTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "shipment.db"))
	require.NoError(t, err)
	_, err = db.Exec(schemaSQL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func (r *closureFrameRecorder) PushClosureShipment(_ context.Context, target string, command *channelv1.ClosureShipmentCommand) (int, error) {
	r.target = target
	r.commands = append(r.commands, command)
	return 1, nil
}

func closureTar(t *testing.T, name string, mode int64, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := tar.NewWriter(&buf)
	require.NoError(t, w.WriteHeader(&tar.Header{Name: name, Mode: mode, Size: int64(len(data)), Typeflag: tar.TypeReg}))
	_, err := w.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}

func TestClosureShipmentCoordinatorValidatesArchiveAndDispatchesBoundFrames(t *testing.T) {
	req, _ := shipmentFixture(t)
	data := req.Files[0].Data
	req.Files[0].Size = int64(len(data))
	digest, err := ClosureInventoryDigest(req.Files)
	require.NoError(t, err)
	req.InventoryDigest = digest
	archive := closureTar(t, req.Files[0].Path, int64(req.Files[0].Mode), data)
	repoDB := openClosureShipmentTestDB(t)
	service := NewClosureShipmentService(NewSQLiteClosureShipmentRepository(repoDB), nil, nil)
	service.TargetPolicy = func(context.Context, ClosureShipment) error { return nil }
	service.AuthorizeSelection = func(context.Context, ClosureShipmentRequest) error { return nil }
	recorder := &closureFrameRecorder{}
	coordinator := NewClosureShipmentCoordinator(service, recorder, func(context.Context, string) (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(archive)), nil
	})
	shipment, err := coordinator.Start(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, req.TargetID, recorder.target)
	require.Equal(t, ShipmentPrepared, shipment.State)
	require.Len(t, recorder.commands, 1)
	require.Equal(t, channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_BEGIN, recorder.commands[0].GetAction())
	// The target's authenticated staged receipt is the gate for transferring bytes.
	require.NoError(t, service.RecordNodeReceipt(context.Background(), ClosureNodeReceipt{
		ShipmentID: shipment.ID, IdempotencyKey: shipment.IdempotencyKey, TargetID: shipment.TargetID,
		SourceRef: shipment.SourceRef, Destination: shipment.Destination, ClosureDigest: shipment.ClosureDigest,
		ExpectedInventoryDigest: shipment.InventoryDigest, PriorRevision: shipment.PriorRevision, NewRevision: shipment.NewRevision,
		SelectionReceiptRef: shipment.SelectionReceiptRef, SelectionDigest: shipment.SelectionDigest,
		DeploymentID: shipment.DeploymentID, MachineID: shipment.MachineID, NodeID: shipment.NodeID,
		EnrollmentGeneration: shipment.EnrollmentGeneration, DesiredRevision: shipment.DesiredRevision,
		State: ShipmentStaged, StageRef: "stage-1",
	}))
	shipment, err = coordinator.Start(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, ShipmentStaged, shipment.State)
	require.Len(t, recorder.commands, 4)
	require.Equal(t, channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_FILE_CHUNK, recorder.commands[1].GetAction())
	require.Equal(t, channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_VERIFY, recorder.commands[2].GetAction())
	require.Equal(t, channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_ACTIVATE, recorder.commands[3].GetAction())
	for _, command := range recorder.commands {
		require.Equal(t, shipment.ID, command.GetShipmentId())
		require.Equal(t, shipment.IdempotencyKey, command.GetIdempotencyKey())
		require.Equal(t, shipment.TargetID, command.GetTargetId())
		require.Equal(t, shipment.ClosureDigest, command.GetClosureDigest())
		require.Equal(t, shipment.InventoryDigest, command.GetInventoryDigest())
		require.Equal(t, shipment.PriorRevision, command.GetPriorRevision())
		require.Equal(t, shipment.NewRevision, command.GetNewRevision())
	}
	chunkHash := sha256.Sum256(recorder.commands[1].GetContent())
	require.Equal(t, hex.EncodeToString(chunkHash[:]), recorder.commands[1].GetChunkSha256())
}

func TestClosureShipmentCoordinatorRejectsArchiveMembershipMismatchBeforeIntent(t *testing.T) {
	req, _ := shipmentFixture(t)
	archive := closureTar(t, "unexpected.txt", 0o644, []byte("wrong"))
	db := openClosureShipmentTestDB(t)
	service := NewClosureShipmentService(NewSQLiteClosureShipmentRepository(db), nil, nil)
	service.TargetPolicy = func(context.Context, ClosureShipment) error { return nil }
	service.AuthorizeSelection = func(context.Context, ClosureShipmentRequest) error { return nil }
	pusher := &closureFrameRecorder{}
	coordinator := NewClosureShipmentCoordinator(service, pusher, func(context.Context, string) (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(archive)), nil
	})
	_, err := coordinator.Start(context.Background(), req)
	require.Error(t, err)
	require.Empty(t, pusher.commands)
	_, err = service.Get(context.Background(), req.IdempotencyKey)
	require.Error(t, err)
}

func TestClosureShipmentFixtureUsesCanonicalGraph(t *testing.T) {
	req, _ := shipmentFixture(t)
	require.NoError(t, coreset.VerifyDigest(req.Closure))
}
