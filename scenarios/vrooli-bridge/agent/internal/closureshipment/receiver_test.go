package closureshipment

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/coreset"
	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	"google.golang.org/protobuf/proto"
)

func shipmentRequest(t *testing.T) *channelv1.ClosureShipmentRequest {
	t.Helper()
	cl, err := coreset.Resolve(coreset.DependencyGraph{Components: []coreset.ComponentID{{Kind: "scenario", Name: "demo"}}}, []coreset.ComponentID{{Kind: "scenario", Name: "demo"}})
	require.NoError(t, err)
	member := cl.Members[0]
	protoMember := &channelv1.ClosureMember{Component: &channelv1.ClosureComponentID{Kind: member.Component.Kind, Name: member.Component.Name}, Selected: member.Selected, Root: &channelv1.ClosureComponentID{Kind: member.Root.Kind, Name: member.Root.Name}}
	file := []byte("abc")
	sum := sha256.Sum256(file)
	sha := hex.EncodeToString(sum[:])
	entry := coreset.FileInventoryEntry{Path: "scenarios/demo/main.go", SHA256: sha, Mode: 0o644, Size: int64(len(file))}
	inventory, err := coreset.FileInventoryDigest([]coreset.FileInventoryEntry{entry})
	require.NoError(t, err)
	return &channelv1.ClosureShipmentRequest{ShipmentId: "shipment-1", IdempotencyKey: "key-1", TargetId: "node-1", SourceRef: "hub://bundle/1", Destination: "checkout", Closure: &channelv1.DependencyClosure{SchemaVersion: int32(cl.SchemaVersion), Members: []*channelv1.ClosureMember{protoMember}, Digest: cl.Digest}, InventoryDigest: inventory, PriorRevision: "", NewRevision: "12", SelectionReceiptRef: "vrooli-onboarding://handoffs/hand-off", SelectionDigest: "sha256:selection", DeploymentId: "deployment-1", MachineId: "machine-1", NodeId: "node-1", EnrollmentGeneration: 7, DesiredRevision: 12, Files: []*channelv1.ClosureFile{{Path: entry.Path, Sha256: sha, Mode: entry.Mode, SizeBytes: uint64(entry.Size)}}}
}

func command(action channelv1.ClosureShipmentAction, req *channelv1.ClosureShipmentRequest) *channelv1.ClosureShipmentCommand {
	return &channelv1.ClosureShipmentCommand{Action: action, ShipmentId: req.GetShipmentId(), IdempotencyKey: req.GetIdempotencyKey(), TargetId: req.GetTargetId(), SourceRef: req.GetSourceRef(), Destination: req.GetDestination(), ClosureDigest: req.GetClosure().GetDigest(), InventoryDigest: req.GetInventoryDigest(), PriorRevision: req.GetPriorRevision(), NewRevision: req.GetNewRevision(), SelectionReceiptRef: req.GetSelectionReceiptRef(), SelectionDigest: req.GetSelectionDigest(), DeploymentId: req.GetDeploymentId(), MachineId: req.GetMachineId(), NodeId: req.GetNodeId(), EnrollmentGeneration: req.GetEnrollmentGeneration(), DesiredRevision: req.GetDesiredRevision()}
}

func TestReceiverVerifiesAndActivatesExactClosureResumably(t *testing.T) {
	root := t.TempDir()
	unrelated := filepath.Join(root, "checkout", "operator.db")
	require.NoError(t, os.MkdirAll(filepath.Dir(unrelated), 0o750))
	require.NoError(t, os.WriteFile(unrelated, []byte("preserve"), 0o600))
	r := Receiver{WorkDir: root, TargetID: "node-1"}
	req := shipmentRequest(t)
	begin := command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_BEGIN, req)
	begin.Request = req
	got, err := r.Handle(context.Background(), begin)
	require.NoError(t, err)
	require.Equal(t, channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_STAGED, got.GetState())
	require.NotContains(t, got.GetStageRef(), root)
	replay, err := r.Handle(context.Background(), begin)
	require.NoError(t, err)
	require.Equal(t, got.GetStageRef(), replay.GetStageRef())
	chunk := command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_FILE_CHUNK, req)
	chunk.FileIndex = 0
	chunk.Content = []byte("abc")
	chunk.ChunkSha256 = hex.EncodeToString(sha256Bytes(chunk.Content))
	got, err = r.Handle(context.Background(), chunk)
	require.NoError(t, err)
	got, err = r.Handle(context.Background(), chunk)
	require.NoError(t, err)
	verify := command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_VERIFY, req)
	got, err = r.Handle(context.Background(), verify)
	require.NoError(t, err)
	require.Equal(t, channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_VERIFIED, got.GetState())
	require.Equal(t, req.GetInventoryDigest(), got.GetActualInventoryDigest())
	// Simulate a crash after the staging directory was moved but before current
	// was advanced. Recovery may reuse this directory only after re-verifying it.
	stage := filepath.Join(root, "checkout", filepath.FromSlash(got.GetStageRef()))
	final := filepath.Join(root, "checkout", ".vrooli-closures", "node-1", "checkout", "revisions", "12")
	destRoot, err := r.openDestinationRoot("checkout")
	require.NoError(t, err)
	require.NoError(t, persistRevisionBinding(destRoot, req, filepath.Join(".vrooli-closures", "node-1", "checkout", "revisions", "12")))
	require.NoError(t, destRoot.Close())
	require.NoError(t, os.MkdirAll(filepath.Dir(final), 0o750))
	require.NoError(t, os.Rename(stage, final))
	activate := command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_ACTIVATE, req)
	got, err = r.Handle(context.Background(), activate)
	require.NoError(t, err)
	require.Equal(t, channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_ACTIVATED, got.GetState())
	got, err = r.Handle(context.Background(), activate)
	require.NoError(t, err)
	require.Equal(t, channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_ACTIVATED, got.GetState())
	data, err := os.ReadFile(unrelated)
	require.NoError(t, err)
	require.Equal(t, "preserve", string(data))
}

func TestReceiverRejectsTamperedAndExtraFilesBeforeActivation(t *testing.T) {
	root := t.TempDir()
	r := Receiver{WorkDir: root, TargetID: "node-1"}
	req := shipmentRequest(t)
	begin := command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_BEGIN, req)
	begin.Request = req
	_, err := r.Handle(context.Background(), begin)
	require.NoError(t, err)
	chunk := command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_FILE_CHUNK, req)
	chunk.FileIndex = 0
	chunk.Content = []byte("bad")
	sum := sha256.Sum256(chunk.Content)
	chunk.ChunkSha256 = hex.EncodeToString(sum[:])
	_, err = r.Handle(context.Background(), chunk)
	require.NoError(t, err)
	stage := filepath.Join(root, "checkout", ".vrooli-closures", "node-1", "shipment-1", "files")
	require.NoError(t, os.WriteFile(filepath.Join(stage, "unexpected"), []byte("extra"), 0o600))
	verify := command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_VERIFY, req)
	got, err := r.Handle(context.Background(), verify)
	require.Error(t, err)
	require.Equal(t, channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_REJECTED, got.GetState())
	_, err = os.Stat(filepath.Join(root, "checkout", ".vrooli-closures", "node-1", "checkout", "current"))
	require.Error(t, err)
}

func TestReceiverRejectsChangedActionBinding(t *testing.T) {
	root := t.TempDir()
	r := Receiver{WorkDir: root, TargetID: "node-1"}
	req := shipmentRequest(t)
	begin := command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_BEGIN, req)
	begin.Request = req
	_, err := r.Handle(context.Background(), begin)
	require.NoError(t, err)
	verify := command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_VERIFY, req)
	verify.NewRevision = "r2"
	_, err = r.Handle(context.Background(), verify)
	require.Error(t, err)
}

func TestReceiverRequiresRevisionToMatchDesiredRevision(t *testing.T) {
	r := Receiver{WorkDir: t.TempDir(), TargetID: "node-1"}
	req := shipmentRequest(t)
	req.DesiredRevision = 13
	begin := command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_BEGIN, req)
	begin.Request = req
	_, err := r.Handle(context.Background(), begin)
	require.ErrorContains(t, err, "must match the owner-selected desired revision")
}

func TestReceiverRejectsPreexistingUnverifiedRevisionDirectory(t *testing.T) {
	root := t.TempDir()
	r := Receiver{WorkDir: root, TargetID: "node-1"}
	req := shipmentRequest(t)
	begin := command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_BEGIN, req)
	begin.Request = req
	_, err := r.Handle(context.Background(), begin)
	require.NoError(t, err)
	chunk := command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_FILE_CHUNK, req)
	chunk.FileIndex = 0
	chunk.Content = []byte("abc")
	chunk.ChunkSha256 = hex.EncodeToString(sha256Bytes(chunk.Content))
	_, err = r.Handle(context.Background(), chunk)
	require.NoError(t, err)
	_, err = r.Handle(context.Background(), command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_VERIFY, req))
	require.NoError(t, err)

	destRoot, err := r.openDestinationRoot("checkout")
	require.NoError(t, err)
	require.NoError(t, persistRevisionBinding(destRoot, req, filepath.Join(".vrooli-closures", "node-1", "checkout", "revisions", "12")))
	require.NoError(t, destRoot.Close())
	final := filepath.Join(root, "checkout", ".vrooli-closures", "node-1", "checkout", "revisions", "12")
	require.NoError(t, os.MkdirAll(final, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(final, "tampered"), []byte("other bytes"), 0o600))
	got, err := r.Handle(context.Background(), command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_ACTIVATE, req))
	require.ErrorContains(t, err, "existing revision directory")
	require.Equal(t, channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_UNCERTAIN, got.GetState())
	_, err = os.Stat(filepath.Join(root, "checkout", ".vrooli-closures", "node-1", "checkout", "current"))
	require.Error(t, err)
}

func TestReceiverRefusesDifferentShipmentReusingActiveRevision(t *testing.T) {
	root := t.TempDir()
	r := Receiver{WorkDir: root, TargetID: "node-1"}
	req1 := shipmentRequest(t)
	req2 := proto.Clone(req1).(*channelv1.ClosureShipmentRequest)
	req2.ShipmentId = "shipment-2"
	req2.IdempotencyKey = "key-2"
	for _, req := range []*channelv1.ClosureShipmentRequest{req1, req2} {
		begin := command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_BEGIN, req)
		begin.Request = req
		_, err := r.Handle(context.Background(), begin)
		require.NoError(t, err)
		chunk := command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_FILE_CHUNK, req)
		chunk.FileIndex = 0
		chunk.Content = []byte("abc")
		chunk.ChunkSha256 = hex.EncodeToString(sha256Bytes(chunk.Content))
		_, err = r.Handle(context.Background(), chunk)
		require.NoError(t, err)
		_, err = r.Handle(context.Background(), command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_VERIFY, req))
		require.NoError(t, err)
	}
	first, err := r.Handle(context.Background(), command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_ACTIVATE, req1))
	require.NoError(t, err)
	require.Equal(t, channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_ACTIVATED, first.GetState())
	second, err := r.Handle(context.Background(), command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_ACTIVATE, req2))
	require.ErrorContains(t, err, "different closure shipment")
	require.Equal(t, channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_UNCERTAIN, second.GetState())
	current, err := os.ReadFile(filepath.Join(root, "checkout", ".vrooli-closures", "node-1", "checkout", "current"))
	require.NoError(t, err)
	require.Equal(t, "12\n", string(current))
}

func TestReceiverRollbackReceiptSurvivesRestartAndReplay(t *testing.T) {
	root := t.TempDir()
	r := Receiver{WorkDir: root, TargetID: "node-1"}
	req := shipmentRequest(t)
	req.PriorRevision = "11"
	prior := filepath.Join(root, "checkout", ".vrooli-closures", "node-1", "checkout", "revisions", "11")
	require.NoError(t, os.MkdirAll(prior, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(prior, "known-good.txt"), []byte("keep prior"), 0o600))
	current := filepath.Join(root, "checkout", ".vrooli-closures", "node-1", "checkout", "current")
	require.NoError(t, os.MkdirAll(filepath.Dir(current), 0o750))
	require.NoError(t, os.WriteFile(current, []byte("11\n"), 0o600))

	begin := command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_BEGIN, req)
	begin.Request = req
	_, err := r.Handle(context.Background(), begin)
	require.NoError(t, err)
	chunk := command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_FILE_CHUNK, req)
	chunk.FileIndex = 0
	chunk.Content = []byte("abc")
	chunk.ChunkSha256 = hex.EncodeToString(sha256Bytes(chunk.Content))
	_, err = r.Handle(context.Background(), chunk)
	require.NoError(t, err)
	_, err = r.Handle(context.Background(), command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_VERIFY, req))
	require.NoError(t, err)
	activated, err := r.Handle(context.Background(), command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_ACTIVATE, req))
	require.NoError(t, err)
	require.Equal(t, channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_ACTIVATED, activated.GetState())
	rolledBack, err := r.Handle(context.Background(), command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_ROLLBACK, req))
	require.NoError(t, err)
	require.Equal(t, channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_ROLLED_BACK, rolledBack.GetState())
	require.Equal(t, "11", rolledBack.GetActiveRevision())
	require.Equal(t, req.GetInventoryDigest(), rolledBack.GetActualInventoryDigest())

	// A new receiver instance must replay the persisted rollback receipt safely.
	restarted := Receiver{WorkDir: root, TargetID: "node-1"}
	rolledBack, err = restarted.Handle(context.Background(), command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_ROLLBACK, req))
	require.NoError(t, err)
	require.Equal(t, channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_ROLLED_BACK, rolledBack.GetState())
	activeRevision, err := os.ReadFile(current)
	require.NoError(t, err)
	require.Equal(t, "11\n", string(activeRevision))
	priorData, err := os.ReadFile(filepath.Join(prior, "known-good.txt"))
	require.NoError(t, err)
	require.Equal(t, "keep prior", string(priorData))
}

func TestReceiverRejectsSymlinkedDestinationAndManifestParent(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	r := Receiver{WorkDir: root, TargetID: "node-1"}
	req := shipmentRequest(t)
	require.NoError(t, os.Symlink(outside, filepath.Join(root, "checkout")))
	begin := command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_BEGIN, req)
	begin.Request = req
	_, err := r.Handle(context.Background(), begin)
	require.ErrorContains(t, err, "symlink")
	_, err = os.Stat(filepath.Join(outside, ".vrooli-closures"))
	require.Error(t, err)

	root, outside = t.TempDir(), t.TempDir()
	r = Receiver{WorkDir: root, TargetID: "node-1"}
	_, err = r.Handle(context.Background(), begin)
	require.NoError(t, err)
	stage := filepath.Join(root, "checkout", ".vrooli-closures", "node-1", "shipment-1", "files")
	require.NoError(t, os.Symlink(outside, filepath.Join(stage, "scenarios")))
	chunk := command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_FILE_CHUNK, req)
	chunk.FileIndex = 0
	chunk.Content = []byte("abc")
	chunk.ChunkSha256 = hex.EncodeToString(sha256Bytes(chunk.Content))
	_, err = r.Handle(context.Background(), chunk)
	require.ErrorContains(t, err, "symlink")
	entries, err := os.ReadDir(outside)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestLoadMetadataRebasesPriorAbsoluteStagePathWithinRoot(t *testing.T) {
	base := filepath.Join(t.TempDir(), "checkout")
	require.NoError(t, os.MkdirAll(filepath.Join(base, ".vrooli-closures", "node-1", "shipment-1"), 0o750))
	root, err := os.OpenRoot(base)
	require.NoError(t, err)
	defer root.Close()
	prior := metadata{StageRef: filepath.Join(base, ".vrooli-closures", "node-1", "shipment-1", "files")}
	b, err := json.Marshal(prior)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(base, ".vrooli-closures", "node-1", "shipment-1", "receipt.json"), b, 0o600))
	loaded, err := loadMetadata(root, filepath.Join(".vrooli-closures", "node-1", "shipment-1"))
	require.NoError(t, err)
	require.Equal(t, filepath.Join(".vrooli-closures", "node-1", "shipment-1", "files"), loaded.StageRef)

	prior.StageRef = filepath.Join(t.TempDir(), "outside")
	b, err = json.Marshal(prior)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(base, ".vrooli-closures", "node-1", "shipment-1", "receipt.json"), b, 0o600))
	_, err = loadMetadata(root, filepath.Join(".vrooli-closures", "node-1", "shipment-1"))
	require.ErrorContains(t, err, "escapes destination root")
}

func TestReceiverKeepsPartialTransferRecoverable(t *testing.T) {
	root := t.TempDir()
	r := Receiver{WorkDir: root, TargetID: "node-1"}
	req := shipmentRequest(t)
	begin := command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_BEGIN, req)
	begin.Request = req
	_, err := r.Handle(context.Background(), begin)
	require.NoError(t, err)
	verify := command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_VERIFY, req)
	got, err := r.Handle(context.Background(), verify)
	require.ErrorIs(t, err, ErrIncompleteStage)
	require.Equal(t, channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_STAGED, got.GetState())
	chunk := command(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_FILE_CHUNK, req)
	chunk.FileIndex = 0
	chunk.Content = []byte("abc")
	chunk.ChunkSha256 = hex.EncodeToString(sha256Bytes(chunk.Content))
	_, err = r.Handle(context.Background(), chunk)
	require.NoError(t, err)
	got, err = r.Handle(context.Background(), verify)
	require.NoError(t, err)
	require.Equal(t, channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_VERIFIED, got.GetState())
}

func sha256Bytes(b []byte) []byte { sum := sha256.Sum256(b); return sum[:] }
