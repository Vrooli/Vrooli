package closureshipment

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/api-core/coreset"
	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	"google.golang.org/protobuf/proto"
)

// ServerFrame JSON carries chunk bytes as base64 and must stay below the
// agent's 1 MiB SSE scanner line limit.
const MaxChunkBytes = 1 << 19

var ErrIncompleteStage = errors.New("closure shipment stage is incomplete")

type metadata struct {
	Request               []byte                                `json:"request"`
	State                 channelv1.ClosureShipmentReceiptState `json:"state"`
	StageRef              string                                `json:"stage_ref"`
	ActualFiles           []*channelv1.ClosureFile              `json:"actual_files,omitempty"`
	ActualInventoryDigest string                                `json:"actual_inventory_digest,omitempty"`
	ActiveRevision        string                                `json:"active_revision,omitempty"`
	Result                string                                `json:"result,omitempty"`
	Error                 string                                `json:"error,omitempty"`
}

// revisionBinding prevents two different shipment intents from claiming the
// same destination revision, including after a process restart.
type revisionBinding struct {
	TargetID        string `json:"target_id"`
	Destination     string `json:"destination"`
	Revision        string `json:"revision"`
	ShipmentID      string `json:"shipment_id"`
	IdempotencyKey  string `json:"idempotency_key"`
	ClosureDigest   string `json:"closure_digest"`
	InventoryDigest string `json:"inventory_digest"`
	RequestDigest   string `json:"request_digest"`
}

type Receiver struct{ WorkDir, TargetID string }

// Handle applies one signed, replay-safe shipment command. All durable state
// lives below WorkDir; staging never writes outside the selected destination.
func (r Receiver) Handle(ctx context.Context, c *channelv1.ClosureShipmentCommand) (*channelv1.ClosureShipmentReceipt, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c == nil {
		return nil, errors.New("closure shipment command is nil")
	}
	var req *channelv1.ClosureShipmentRequest
	if c.GetAction() == channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_BEGIN {
		req = c.GetRequest()
		if err := validateRequest(req, r.TargetID); err != nil {
			return rejected(req, r.TargetID, err), err
		}
	} else {
		var err error
		req, err = r.loadRequest(c)
		if err != nil {
			return rejected(nil, r.TargetID, err), err
		}
		if c.GetRequest() != nil && !proto.Equal(c.GetRequest(), req) {
			err = errors.New("closure shipment request mutation")
			return rejected(req, r.TargetID, err), err
		}
		if err = matchBinding(c, req); err != nil {
			return rejected(req, r.TargetID, err), err
		}
	}
	base, err := r.openDestinationRoot(req.GetDestination())
	if err != nil {
		return rejected(req, r.TargetID, err), err
	}
	defer base.Close()
	stateDir := filepath.Join(".vrooli-closures", safe(req.GetTargetId()), safe(req.GetShipmentId()))
	switch c.GetAction() {
	case channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_BEGIN:
		if existing, e := loadMetadata(base, stateDir); e == nil {
			var old channelv1.ClosureShipmentRequest
			if proto.Unmarshal(existing.Request, &old) == nil && proto.Equal(&old, req) {
				return r.response(req, existing, nil), nil
			}
			err = errors.New("shipment id is already bound to another request")
			return rejected(req, r.TargetID, err), err
		}
		cur, err := readRevision(base, filepath.Join(".vrooli-closures", safe(req.GetTargetId()), safe(req.GetDestination()), "current"))
		if err != nil {
			return rejected(req, r.TargetID, err), err
		}
		if cur != req.GetPriorRevision() {
			err = fmt.Errorf("stale prior revision: expected %q, found %q", req.GetPriorRevision(), cur)
			return rejected(req, r.TargetID, err), err
		}
		if cur == req.GetNewRevision() {
			err = errors.New("new revision already active without matching shipment receipt")
			return rejected(req, r.TargetID, err), err
		}
		b, _ := proto.Marshal(req)
		stage := filepath.Join(stateDir, "files")
		m := metadata{Request: b, State: channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_STAGED, StageRef: stage}
		if err := ensureNoSymlinkPath(base, stage); err != nil {
			return rejected(req, r.TargetID, err), err
		}
		if err := base.MkdirAll(stage, 0o750); err != nil {
			return rejected(req, r.TargetID, err), err
		}
		for _, f := range req.GetFiles() {
			if f.GetSizeBytes() != 0 {
				continue
			}
			name := filepath.Join(stage, filepath.FromSlash(f.GetPath()))
			if err := ensureNoSymlinkPath(base, filepath.Dir(name)); err != nil {
				return rejected(req, r.TargetID, err), err
			}
			if err := base.MkdirAll(filepath.Dir(name), 0o750); err != nil {
				return rejected(req, r.TargetID, err), err
			}
			if err := ensureNoSymlinkPath(base, name); err != nil {
				return rejected(req, r.TargetID, err), err
			}
			if err := base.WriteFile(name, nil, os.FileMode(coreset.NormalizeFileMode(f.GetMode()))); err != nil {
				return rejected(req, r.TargetID, err), err
			}
		}
		if err := saveMetadata(base, stateDir, m); err != nil {
			return rejected(req, r.TargetID, err), err
		}
		return r.response(req, m, nil), nil
	case channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_FILE_CHUNK:
		m, err := loadMetadata(base, stateDir)
		if err != nil {
			return rejected(req, r.TargetID, err), err
		}
		if err := validateActive(m); err != nil {
			return rejected(req, r.TargetID, err), err
		}
		if m.State == channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_VERIFIED || m.State == channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_ACTIVATED || m.State == channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_ROLLED_BACK {
			// A replay may arrive after the previous response was lost. Preserve
			// the target's durable state and return it instead of downgrading the
			// operation to a synthetic rejection.
			return r.response(req, m, nil), nil
		}
		if m.State != channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_STAGED {
			err := errors.New("file chunks are accepted only while staged")
			return r.response(req, m, err), err
		}
		if err := writeChunk(base, m.StageRef, req, c); err != nil {
			return r.response(req, m, err), err
		}
		return r.response(req, m, nil), nil
	case channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_VERIFY:
		m, err := loadMetadata(base, stateDir)
		if err != nil {
			return rejected(req, r.TargetID, err), err
		}
		if err := validateActive(m); err != nil {
			return rejected(req, r.TargetID, err), err
		}
		if m.State == channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_VERIFIED || m.State == channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_ACTIVATED || m.State == channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_ROLLED_BACK {
			return receipt(req, m, nil), nil
		}
		if m.State != channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_STAGED {
			return r.response(req, m, errors.New("closure cannot be verified from its current state")), errors.New("closure cannot be verified from its current state")
		}
		files, digest, err := verifyStage(base, m.StageRef, req)
		if err != nil {
			if errors.Is(err, ErrIncompleteStage) {
				m.Error = err.Error()
				_ = saveMetadata(base, stateDir, m)
				return r.response(req, m, err), err
			}
			m.State = channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_REJECTED
			m.Error = err.Error()
			_ = saveMetadata(base, stateDir, m)
			return r.response(req, m, err), err
		}
		m.State = channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_VERIFIED
		m.ActualFiles = files
		m.ActualInventoryDigest = digest
		m.Result = "exact closure file inventory verified"
		if err := saveMetadata(base, stateDir, m); err != nil {
			return nil, err
		}
		return r.response(req, m, nil), nil
	case channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_ACTIVATE:
		m, err := loadMetadata(base, stateDir)
		if err != nil {
			return rejected(req, r.TargetID, err), err
		}
		if err := validateActive(m); err != nil {
			return rejected(req, r.TargetID, err), err
		}
		if m.State == channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_ACTIVATED || m.State == channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_ROLLED_BACK {
			return r.response(req, m, nil), nil
		}
		if m.State != channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_VERIFIED && m.State != channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_ACTIVATED {
			return rejected(req, r.TargetID, errors.New("closure must be verified before activation")), errors.New("closure must be verified before activation")
		}
		if err := r.activate(base, req, m); err != nil {
			m.State = channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_UNCERTAIN
			m.Error = err.Error()
			_ = saveMetadata(base, stateDir, m)
			return r.response(req, m, err), err
		}
		m.State = channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_ACTIVATED
		m.ActiveRevision = req.GetNewRevision()
		m.Result = "verified closure revision activated"
		if err := saveMetadata(base, stateDir, m); err != nil {
			return r.response(req, m, err), err
		}
		return r.response(req, m, nil), nil
	case channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_ROLLBACK:
		m, err := loadMetadata(base, stateDir)
		if err != nil {
			return rejected(req, r.TargetID, err), err
		}
		if err := validateActive(m); err != nil {
			return rejected(req, r.TargetID, err), err
		}
		currentPath := filepath.Join(".vrooli-closures", safe(req.GetTargetId()), safe(req.GetDestination()), "current")
		cur, err := readRevision(base, currentPath)
		if err != nil {
			return rejected(req, r.TargetID, err), err
		}
		if cur != req.GetNewRevision() && cur != req.GetPriorRevision() {
			err = errors.New("cannot roll back after a newer revision activated")
			return rejected(req, r.TargetID, err), err
		}
		if cur == req.GetNewRevision() {
			if err := writeRevision(base, currentPath, req.GetPriorRevision()); err != nil {
				return rejected(req, r.TargetID, err), err
			}
		}
		if err := ensureNoSymlinkPath(base, filepath.Join(stateDir, "files")); err != nil {
			return r.response(req, m, err), err
		}
		if err := base.RemoveAll(filepath.Join(stateDir, "files")); err != nil {
			return r.response(req, m, err), err
		}
		m.State = channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_ROLLED_BACK
		m.ActiveRevision = req.GetPriorRevision()
		m.Result = "prior revision restored"
		if err := saveMetadata(base, stateDir, m); err != nil {
			return r.response(req, m, err), err
		}
		return r.response(req, m, nil), nil
	case channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_RESUME:
		m, err := loadMetadata(base, stateDir)
		if err != nil {
			return rejected(req, r.TargetID, err), err
		}
		if err := validateActive(m); err != nil {
			return rejected(req, r.TargetID, err), err
		}
		return r.response(req, m, nil), nil
	default:
		err := errors.New("unsupported closure shipment action")
		return rejected(req, r.TargetID, err), err
	}
}

func (r Receiver) loadRequest(c *channelv1.ClosureShipmentCommand) (*channelv1.ClosureShipmentRequest, error) {
	if c.GetShipmentId() == "" {
		return nil, errors.New("shipment identity required")
	}
	// The destination is bound in the command and the only durable request copy
	// is found through target/shipment metadata under WorkDir.
	base, err := r.openDestinationRoot(c.GetDestination())
	if err != nil {
		return nil, err
	}
	defer base.Close()
	dir := filepath.Join(".vrooli-closures", safe(c.GetTargetId()), safe(c.GetShipmentId()))
	m, err := loadMetadata(base, dir)
	if err != nil {
		return nil, err
	}
	var req channelv1.ClosureShipmentRequest
	if err := proto.Unmarshal(m.Request, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func (r Receiver) openDestinationRoot(dest string) (*os.Root, error) {
	if strings.TrimSpace(r.WorkDir) == "" {
		return nil, errors.New("agent work directory is not configured")
	}
	if dest == "" || filepath.IsAbs(dest) || strings.Contains(dest, "\\") || path.Clean(dest) != dest || dest == ".." || strings.HasPrefix(dest, "../") {
		return nil, errors.New("closure destination must be relative to the agent work directory")
	}
	work, err := filepath.Abs(r.WorkDir)
	if err != nil {
		return nil, err
	}
	workRoot, err := os.OpenRoot(work)
	if err != nil {
		return nil, err
	}
	if err := ensureNoSymlinkPath(workRoot, filepath.FromSlash(dest)); err != nil {
		workRoot.Close()
		return nil, fmt.Errorf("closure destination contains a symlink: %w", err)
	}
	if err := workRoot.MkdirAll(filepath.FromSlash(dest), 0o750); err != nil {
		workRoot.Close()
		return nil, fmt.Errorf("create closure destination beneath agent work directory: %w", err)
	}
	if err := ensureNoSymlinkPath(workRoot, filepath.FromSlash(dest)); err != nil {
		workRoot.Close()
		return nil, fmt.Errorf("closure destination contains a symlink: %w", err)
	}
	destination, err := workRoot.OpenRoot(filepath.FromSlash(dest))
	closeErr := workRoot.Close()
	if err != nil {
		return nil, fmt.Errorf("closure destination escapes or is unavailable beneath agent work directory: %w", err)
	}
	if closeErr != nil {
		destination.Close()
		return nil, closeErr
	}
	return destination, nil
}

func validateRequest(req *channelv1.ClosureShipmentRequest, target string) error {
	if req == nil || req.GetShipmentId() == "" || req.GetIdempotencyKey() == "" || req.GetTargetId() == "" || req.GetSourceRef() == "" || req.GetNewRevision() == "" || req.GetDestination() == "" {
		return errors.New("closure shipment request is incomplete")
	}
	if req.GetTargetId() != target {
		return errors.New("closure shipment target mismatch")
	}
	if req.GetNodeId() != req.GetTargetId() {
		return errors.New("closure shipment onboarding node binding mismatch")
	}
	if req.GetNewRevision() == req.GetPriorRevision() {
		return errors.New("new revision must differ from prior revision")
	}
	if req.GetNewRevision() != strconv.FormatUint(req.GetDesiredRevision(), 10) {
		return errors.New("new revision must match the owner-selected desired revision")
	}
	if req.GetSelectionReceiptRef() == "" || req.GetSelectionDigest() == "" || req.GetDeploymentId() == "" || req.GetMachineId() == "" || req.GetNodeId() == "" || req.GetEnrollmentGeneration() == 0 || req.GetDesiredRevision() == 0 {
		return errors.New("immutable onboarding selection receipt binding is required")
	}
	if err := validateProtoClosure(req.GetClosure()); err != nil {
		return err
	}
	return validateManifest(req)
}

func validateProtoClosure(c *channelv1.DependencyClosure) error {
	if c == nil {
		return errors.New("canonical dependency closure is required")
	}
	closure := coreset.DependencyClosure{SchemaVersion: int(c.GetSchemaVersion()), Digest: c.GetDigest()}
	for _, m := range c.GetMembers() {
		if m == nil || m.GetComponent() == nil || m.GetRoot() == nil {
			return errors.New("closure member identity is incomplete")
		}
		member := coreset.ClosureMember{Component: coreset.ComponentID{Kind: m.GetComponent().GetKind(), Name: m.GetComponent().GetName()}, Selected: m.GetSelected(), Root: coreset.ComponentID{Kind: m.GetRoot().GetKind(), Name: m.GetRoot().GetName()}}
		for _, p := range m.GetPath() {
			member.Path = append(member.Path, convertReason(p))
		}
		for _, p := range m.GetReasons() {
			member.Reasons = append(member.Reasons, convertReason(p))
		}
		closure.Members = append(closure.Members, member)
	}
	return coreset.VerifyDigest(closure)
}
func convertReason(r *channelv1.ClosureReason) coreset.ClosureReason {
	if r == nil {
		return coreset.ClosureReason{}
	}
	id := func(v *channelv1.ClosureComponentID) coreset.ComponentID {
		if v == nil {
			return coreset.ComponentID{}
		}
		return coreset.ComponentID{Kind: v.GetKind(), Name: v.GetName()}
	}
	return coreset.ClosureReason{From: id(r.GetFrom()), To: id(r.GetTo()), Relation: r.GetRelation(), DeclaredBy: r.GetDeclaredBy(), Detail: r.GetDetail(), Required: r.GetRequired(), Policy: r.GetPolicy()}
}

func validateManifest(req *channelv1.ClosureShipmentRequest) error {
	entries := make([]coreset.FileInventoryEntry, 0, len(req.GetFiles()))
	for _, f := range req.GetFiles() {
		if f == nil {
			return errors.New("nil closure file manifest entry")
		}
		entries = append(entries, coreset.FileInventoryEntry{Path: f.GetPath(), SHA256: f.GetSha256(), Mode: f.GetMode(), Size: int64(f.GetSizeBytes())})
	}
	digest, err := coreset.FileInventoryDigest(entries)
	if err != nil {
		return err
	}
	if digest != req.GetInventoryDigest() {
		return errors.New("closure file inventory digest mismatch")
	}
	return nil
}

func matchBinding(c *channelv1.ClosureShipmentCommand, r *channelv1.ClosureShipmentRequest) error {
	if c.GetShipmentId() != r.GetShipmentId() || c.GetIdempotencyKey() != r.GetIdempotencyKey() || c.GetTargetId() != r.GetTargetId() || c.GetSourceRef() != r.GetSourceRef() || c.GetDestination() != r.GetDestination() || c.GetClosureDigest() != r.GetClosure().GetDigest() || c.GetInventoryDigest() != r.GetInventoryDigest() || c.GetPriorRevision() != r.GetPriorRevision() || c.GetNewRevision() != r.GetNewRevision() || c.GetSelectionReceiptRef() != r.GetSelectionReceiptRef() || c.GetSelectionDigest() != r.GetSelectionDigest() || c.GetDeploymentId() != r.GetDeploymentId() || c.GetMachineId() != r.GetMachineId() || c.GetNodeId() != r.GetNodeId() || c.GetEnrollmentGeneration() != r.GetEnrollmentGeneration() || c.GetDesiredRevision() != r.GetDesiredRevision() {
		return errors.New("closure shipment command binding mismatch")
	}
	if c.GetRequest() != nil && !proto.Equal(c.GetRequest(), r) {
		return errors.New("closure shipment request mutation")
	}
	return nil
}

func writeChunk(root *os.Root, stage string, r *channelv1.ClosureShipmentRequest, c *channelv1.ClosureShipmentCommand) error {
	if int(c.GetFileIndex()) >= len(r.GetFiles()) {
		return errors.New("closure file index outside manifest")
	}
	if len(c.GetContent()) == 0 || len(c.GetContent()) > MaxChunkBytes {
		return errors.New("closure chunk size outside limit")
	}
	sum := sha256.Sum256(c.GetContent())
	if hex.EncodeToString(sum[:]) != strings.ToLower(c.GetChunkSha256()) {
		return errors.New("closure chunk digest mismatch")
	}
	f := r.GetFiles()[c.GetFileIndex()]
	if c.GetOffsetBytes()+uint64(len(c.GetContent())) > f.GetSizeBytes() {
		return errors.New("closure chunk exceeds declared file size")
	}
	name := filepath.Join(stage, filepath.FromSlash(f.GetPath()))
	if err := ensureNoSymlinkPath(root, filepath.Dir(name)); err != nil {
		return err
	}
	if err := root.MkdirAll(filepath.Dir(name), 0o750); err != nil {
		return err
	}
	if err := ensureNoSymlinkPath(root, name); err != nil {
		return err
	}
	file, err := root.OpenFile(name, os.O_CREATE|os.O_RDWR, os.FileMode(coreset.NormalizeFileMode(f.GetMode())))
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	offset := int64(c.GetOffsetBytes())
	if offset > info.Size() {
		return errors.New("closure chunk offset skips bytes")
	}
	if offset < info.Size() {
		prefixSize := info.Size() - offset
		if prefixSize > int64(len(c.GetContent())) {
			prefixSize = int64(len(c.GetContent()))
		}
		prefix := make([]byte, prefixSize)
		n, e := file.ReadAt(prefix, offset)
		if e != nil && e != io.EOF {
			return e
		}
		if int64(n) != prefixSize || !equal(prefix, c.GetContent()[:prefixSize]) {
			return errors.New("closure duplicate chunk does not match staged bytes")
		}
		if prefixSize == int64(len(c.GetContent())) {
			return nil
		}
		rest := c.GetContent()[prefixSize:]
		written, writeErr := file.WriteAt(rest, offset+prefixSize)
		if writeErr != nil {
			return writeErr
		}
		if written != len(rest) {
			return io.ErrShortWrite
		}
		return file.Sync()
	}
	n, err := file.WriteAt(c.GetContent(), offset)
	if err != nil {
		return err
	}
	if n != len(c.GetContent()) {
		return io.ErrShortWrite
	}
	return file.Sync()
}

func verifyStage(root *os.Root, stage string, req *channelv1.ClosureShipmentRequest) ([]*channelv1.ClosureFile, string, error) {
	want := map[string]bool{}
	actual := make([]*channelv1.ClosureFile, 0, len(req.GetFiles()))
	entries := make([]coreset.FileInventoryEntry, 0, len(req.GetFiles()))
	for _, f := range req.GetFiles() {
		want[f.GetPath()] = true
		name := filepath.Join(stage, filepath.FromSlash(f.GetPath()))
		if err := ensureNoSymlinkPath(root, name); err != nil {
			return nil, "", err
		}
		info, err := root.Stat(name)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, "", fmt.Errorf("%w: missing closure file %s", ErrIncompleteStage, f.GetPath())
			}
			return nil, "", fmt.Errorf("missing closure file %s: %w", f.GetPath(), err)
		}
		if !info.Mode().IsRegular() {
			return nil, "", fmt.Errorf("closure member is not a regular file: %s", f.GetPath())
		}
		data, err := root.ReadFile(name)
		if err != nil {
			return nil, "", err
		}
		sum := sha256.Sum256(data)
		if uint64(len(data)) < f.GetSizeBytes() {
			return nil, "", fmt.Errorf("%w: closure file %s has %d of %d bytes", ErrIncompleteStage, f.GetPath(), len(data), f.GetSizeBytes())
		}
		if uint64(len(data)) != f.GetSizeBytes() || hex.EncodeToString(sum[:]) != strings.ToLower(f.GetSha256()) {
			return nil, "", fmt.Errorf("closure file verification failed: %s", f.GetPath())
		}
		mode := uint32(info.Mode().Perm())
		if mode != coreset.NormalizeFileMode(f.GetMode()) {
			return nil, "", fmt.Errorf("closure file mode mismatch: %s", f.GetPath())
		}
		actual = append(actual, &channelv1.ClosureFile{Path: f.GetPath(), Sha256: hex.EncodeToString(sum[:]), Mode: mode, SizeBytes: uint64(len(data))})
		entries = append(entries, coreset.FileInventoryEntry{Path: f.GetPath(), SHA256: hex.EncodeToString(sum[:]), Mode: mode, Size: int64(len(data))})
	}
	var unexpected []string
	err := fs.WalkDir(root.FS(), filepath.ToSlash(stage), func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("staged closure contains a symlink: %s", p)
		}
		if d.IsDir() {
			return nil
		}
		rel, er := filepath.Rel(stage, filepath.FromSlash(p))
		if er != nil {
			return er
		}
		rel = filepath.ToSlash(rel)
		if !want[rel] {
			unexpected = append(unexpected, rel)
		}
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	if len(unexpected) > 0 {
		return nil, "", fmt.Errorf("unexpected staged closure members: %s", strings.Join(unexpected, ", "))
	}
	sort.Slice(actual, func(i, j int) bool { return actual[i].GetPath() < actual[j].GetPath() })
	digest, err := coreset.FileInventoryDigest(entries)
	if err != nil {
		return nil, "", err
	}
	if digest != req.GetInventoryDigest() {
		return nil, "", errors.New("delivered closure inventory digest mismatch")
	}
	return actual, digest, nil
}

func (r Receiver) activate(base *os.Root, req *channelv1.ClosureShipmentRequest, m metadata) error {
	currentPath := filepath.Join(".vrooli-closures", safe(req.GetTargetId()), safe(req.GetDestination()), "current")
	cur, err := readRevision(base, currentPath)
	if err != nil {
		return err
	}
	final := filepath.Join(".vrooli-closures", safe(req.GetTargetId()), safe(req.GetDestination()), "revisions", safe(req.GetNewRevision()))
	if cur == req.GetNewRevision() {
		if err := verifyRevisionBinding(base, req); err != nil {
			return err
		}
		if _, digest, err := verifyStage(base, final, req); err != nil || digest != req.GetInventoryDigest() {
			return errors.New("active revision does not match the verified closure inventory")
		}
		return nil
	}
	if cur != req.GetPriorRevision() {
		return fmt.Errorf("stale closure activation: expected %q, found %q", req.GetPriorRevision(), cur)
	}
	revisions := filepath.Join(".vrooli-closures", safe(req.GetTargetId()), safe(req.GetDestination()), "revisions")
	if err := ensureNoSymlinkPath(base, revisions); err != nil {
		return err
	}
	if err := base.MkdirAll(revisions, 0o750); err != nil {
		return err
	}
	if err := persistRevisionBinding(base, req, final); err != nil {
		return err
	}
	if err := ensureNoSymlinkPath(base, final); err != nil {
		return err
	}
	if _, err := base.Stat(final); err == nil {
		if _, digest, verifyErr := verifyStage(base, final, req); verifyErr != nil || digest != req.GetInventoryDigest() {
			return errors.New("existing revision directory does not match verified closure inventory")
		}
	} else if os.IsNotExist(err) {
		if err := ensureNoSymlinkPath(base, m.StageRef); err != nil {
			return err
		}
		if err := base.Rename(m.StageRef, final); err != nil {
			return err
		}
	} else {
		return err
	}
	return writeRevision(base, currentPath, req.GetNewRevision())
}

func revisionBindingPath(req *channelv1.ClosureShipmentRequest) string {
	return filepath.Join(".vrooli-closures", safe(req.GetTargetId()), safe(req.GetDestination()), "revision-bindings", safe(req.GetNewRevision())+".json")
}

func expectedRevisionBinding(req *channelv1.ClosureShipmentRequest) (revisionBinding, error) {
	encoded, err := (proto.MarshalOptions{Deterministic: true}).Marshal(req)
	if err != nil {
		return revisionBinding{}, err
	}
	sum := sha256.Sum256(encoded)
	return revisionBinding{
		TargetID: req.GetTargetId(), Destination: req.GetDestination(), Revision: req.GetNewRevision(),
		ShipmentID: req.GetShipmentId(), IdempotencyKey: req.GetIdempotencyKey(),
		ClosureDigest: req.GetClosure().GetDigest(), InventoryDigest: req.GetInventoryDigest(),
		RequestDigest: hex.EncodeToString(sum[:]),
	}, nil
}

func persistRevisionBinding(root *os.Root, req *channelv1.ClosureShipmentRequest, final string) error {
	want, err := expectedRevisionBinding(req)
	if err != nil {
		return err
	}
	path := revisionBindingPath(req)
	if err := ensureNoSymlinkPath(root, filepath.Dir(path)); err != nil {
		return err
	}
	if err := root.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	if _, err := root.Stat(path); err == nil {
		return verifyRevisionBinding(root, req)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := ensureNoSymlinkPath(root, final); err != nil {
		return err
	}
	if _, err := root.Stat(final); err == nil {
		return errors.New("revision directory exists without its durable request binding")
	} else if !os.IsNotExist(err) {
		return err
	}
	encoded, err := json.Marshal(want)
	if err != nil {
		return err
	}
	return atomicWrite(root, path, encoded, 0o600)
}

func verifyRevisionBinding(root *os.Root, req *channelv1.ClosureShipmentRequest) error {
	path := revisionBindingPath(req)
	if err := ensureNoSymlinkPath(root, path); err != nil {
		return err
	}
	encoded, err := root.ReadFile(path)
	if err != nil {
		return fmt.Errorf("revision has no durable request binding: %w", err)
	}
	var actual revisionBinding
	if err := json.Unmarshal(encoded, &actual); err != nil {
		return fmt.Errorf("decode revision request binding: %w", err)
	}
	want, err := expectedRevisionBinding(req)
	if err != nil {
		return err
	}
	if actual != want {
		return errors.New("destination revision is already bound to a different closure shipment")
	}
	return nil
}

func loadMetadata(root *os.Root, dir string) (metadata, error) {
	var m metadata
	b, err := root.ReadFile(filepath.Join(dir, "receipt.json"))
	if err != nil {
		return m, err
	}
	err = json.Unmarshal(b, &m)
	if err == nil && filepath.IsAbs(m.StageRef) {
		// Earlier receiver versions persisted an absolute staging path. Rebase
		// only paths contained by this already-open destination root, then write
		// the relative form on the next durable state transition.
		rel, relErr := filepath.Rel(root.Name(), m.StageRef)
		if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return metadata{}, errors.New("persisted closure stage path escapes destination root")
		}
		m.StageRef = rel
	}
	return m, err
}
func saveMetadata(root *os.Root, dir string, m metadata) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return atomicWrite(root, filepath.Join(dir, "receipt.json"), b, 0o600)
}
func validateActive(m metadata) error {
	if len(m.Request) == 0 {
		return errors.New("closure shipment has no durable request")
	}
	return nil
}
func readRevision(root *os.Root, p string) (string, error) {
	if err := ensureNoSymlinkPath(root, p); err != nil {
		return "", err
	}
	b, err := root.ReadFile(p)
	if os.IsNotExist(err) {
		return "", nil
	}
	return strings.TrimSpace(string(b)), err
}
func writeRevision(root *os.Root, p, v string) error {
	if err := ensureNoSymlinkPath(root, filepath.Dir(p)); err != nil {
		return err
	}
	if err := root.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return err
	}
	return atomicWrite(root, p, []byte(v+"\n"), 0o600)
}
func atomicWrite(root *os.Root, p string, b []byte, mode fs.FileMode) error {
	tmp := p + ".tmp"
	if err := ensureNoSymlinkPath(root, filepath.Dir(p)); err != nil {
		return err
	}
	if err := ensureNoSymlinkPath(root, p); err != nil {
		return err
	}
	if err := ensureNoSymlinkPath(root, tmp); err != nil {
		return err
	}
	if err := root.WriteFile(tmp, b, mode); err != nil {
		return err
	}
	if err := root.Rename(tmp, p); err != nil {
		_ = root.Remove(tmp)
		return err
	}
	return nil
}

// ensureNoSymlinkPath checks each existing component through the destination
// root. Root methods already prevent escape from that root; rejecting symlinks
// also prevents a destination-local link from redirecting writes into a
// different target tree beneath WorkDir.
func ensureNoSymlinkPath(root *os.Root, name string) error {
	clean := filepath.Clean(name)
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return errors.New("path escapes closure destination")
	}
	parts := strings.Split(clean, string(filepath.Separator))
	current := "."
	for index, part := range parts {
		if part == "." || part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := root.Lstat(current)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			return fmt.Errorf("symlink path component %s is not allowed", current)
		}
		if index < len(parts)-1 && !info.IsDir() {
			return fmt.Errorf("path component %s is not a directory", current)
		}
	}
	return nil
}
func safe(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	if s == "" || s == "." || s == ".." {
		return "_"
	}
	return s
}
func equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := range a {
		v |= a[i] ^ b[i]
	}
	return v == 0
}
func rejected(r *channelv1.ClosureShipmentRequest, target string, err error) *channelv1.ClosureShipmentReceipt {
	receipt := &channelv1.ClosureShipmentReceipt{TargetId: target, State: channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_REJECTED, Error: err.Error()}
	if r != nil {
		receipt.ShipmentId = r.GetShipmentId()
		receipt.IdempotencyKey = r.GetIdempotencyKey()
		receipt.SourceRef = r.GetSourceRef()
		receipt.Destination = r.GetDestination()
		receipt.ClosureDigest = r.GetClosure().GetDigest()
		receipt.InventoryDigest = r.GetInventoryDigest()
		receipt.PriorRevision = r.GetPriorRevision()
		receipt.NewRevision = r.GetNewRevision()
		receipt.SelectionReceiptRef = r.GetSelectionReceiptRef()
		receipt.SelectionDigest = r.GetSelectionDigest()
		receipt.DeploymentId = r.GetDeploymentId()
		receipt.MachineId = r.GetMachineId()
		receipt.NodeId = r.GetNodeId()
		receipt.EnrollmentGeneration = r.GetEnrollmentGeneration()
		receipt.DesiredRevision = r.GetDesiredRevision()
	}
	return receipt
}
func receipt(r *channelv1.ClosureShipmentRequest, m metadata, err error) *channelv1.ClosureShipmentReceipt {
	out := &channelv1.ClosureShipmentReceipt{ShipmentId: r.GetShipmentId(), IdempotencyKey: r.GetIdempotencyKey(), TargetId: r.GetTargetId(), SourceRef: r.GetSourceRef(), Destination: r.GetDestination(), ClosureDigest: r.GetClosure().GetDigest(), InventoryDigest: r.GetInventoryDigest(), ActualInventoryDigest: m.ActualInventoryDigest, PriorRevision: r.GetPriorRevision(), NewRevision: r.GetNewRevision(), ActiveRevision: m.ActiveRevision, StageRef: m.StageRef, State: m.State, Result: m.Result, Error: m.Error, ActualFiles: m.ActualFiles, SelectionReceiptRef: r.GetSelectionReceiptRef(), SelectionDigest: r.GetSelectionDigest(), DeploymentId: r.GetDeploymentId(), MachineId: r.GetMachineId(), NodeId: r.GetNodeId(), EnrollmentGeneration: r.GetEnrollmentGeneration(), DesiredRevision: r.GetDesiredRevision()}
	if err != nil && out.Error == "" {
		out.Error = err.Error()
	}
	return out
}

func (r Receiver) response(req *channelv1.ClosureShipmentRequest, m metadata, err error) *channelv1.ClosureShipmentReceipt {
	return receipt(req, m, err)
}
