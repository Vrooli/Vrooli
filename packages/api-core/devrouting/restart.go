package devrouting

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/platform-go"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing"
)

const checkpointNamespace = "vrooli/api-core/devrouting-checkpoints"

type checkpointOwnerLock struct {
	path    string
	release func()
}

type restartCheckpoint struct {
	Version              int                               `json:"version"`
	TokenHash            string                            `json:"token_hash"`
	LeaseID              string                            `json:"lease_id"`
	ExpiresAt            time.Time                         `json:"expires_at"`
	TargetFingerprint    string                            `json:"target_fingerprint"`
	RootsFingerprint     string                            `json:"roots_fingerprint"`
	PrimaryDBFingerprint string                            `json:"primary_db_fingerprint"`
	Database             database.TestPoolRestartState     `json:"database"`
	Roots                filerouting.TestRootsRestartState `json:"roots"`
	PreparedInstance     string                            `json:"prepared_instance"`
	ResumeInstance       string                            `json:"resume_instance,omitempty"`
	Phase                string                            `json:"phase"`
}

type rootMarker struct {
	TokenHash         string `json:"token_hash"`
	LeaseID           string `json:"lease_id"`
	TargetFingerprint string `json:"target_fingerprint"`
}

type checkpointRef struct {
	ticket string
	path   string
}

func (s *Service) PrepareTestPoolRestart(_ context.Context, req *connect.Request[routingv1.PrepareTestPoolRestartRequest]) (*connect.Response[routingv1.PrepareTestPoolRestartResponse], error) {
	if s == nil || s.db == nil || s.roots == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("paired database and file routing are required"))
	}
	s.opMu.Lock()
	defer s.opMu.Unlock()
	leaseID := req.Msg.GetLeaseId()
	if leaseID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("lease_id is required"))
	}
	release, lockErr := acquireLeaseOperationLock(leaseID)
	if lockErr != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("another routing operation is in progress"))
	}
	defer release()
	if ref, ok := s.checkpoints[leaseID]; ok {
		var existing restartCheckpoint
		if readCheckpoint(ref.path, ref.ticket, &existing) == nil && existing.LeaseID == leaseID && existing.Phase == "prepared" && s.checkpointBindingMatches(existing) {
			return connect.NewResponse(prepareResponse(ref.ticket, existing)), nil
		}
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("restart checkpoint state is unavailable"))
	}
	bound, err := s.db.PinTestPool(context.Background(), leaseID)
	if err == nil {
		bound, err = s.roots.PinTestRoots(bound, leaseID)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("paired routing binding changed during checkpoint"))
	}
	s.bindings[leaseID] = bound
	dbState, err := s.db.TestPoolRestartState(leaseID)
	if err != nil || !dbState.ExpiresAt.After(time.Now()) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("active database lease is unavailable"))
	}
	rootState, err := s.roots.TestRootsRestartState(leaseID)
	if err != nil || rootState.ExpiresAt.IsZero() || !rootState.ExpiresAt.After(time.Now()) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("paired file lease is unavailable or inconsistent"))
	}
	if s.db.PrimaryFingerprint() == "" || s.roots.PrimaryFingerprint() == "" || rootState.PrimaryFingerprint != s.roots.PrimaryFingerprint() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("restart target identity is unavailable"))
	}
	ticket, err := randomToken()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("create restart checkpoint"))
	}
	tokenHash := hashToken(ticket)
	target := targetFingerprint(dbState.PrimaryFingerprint, rootState.PrimaryFingerprint)
	path, err := checkpointPathFor(target, leaseID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("create restart checkpoint"))
	}
	if _, err := os.Lstat(path); err == nil || !os.IsNotExist(err) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("a restart checkpoint already exists for this lease"))
	}
	ownerLock, err := acquireCheckpointOwnerLock(path)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("restart checkpoint owner lock is unavailable"))
	}
	keepOwnerLock := false
	defer func() {
		if !keepOwnerLock {
			ownerLock.release()
			_ = os.Remove(ownerLock.path)
		}
	}()
	expiresAt := dbState.ExpiresAt
	if rootState.ExpiresAt.Before(expiresAt) {
		expiresAt = rootState.ExpiresAt
	}
	checkpoint := restartCheckpoint{
		Version: 1, TokenHash: tokenHash, LeaseID: leaseID, ExpiresAt: expiresAt,
		TargetFingerprint: target, RootsFingerprint: rootState.PrimaryFingerprint,
		PrimaryDBFingerprint: dbState.PrimaryFingerprint, Database: dbState, Roots: rootState,
		PreparedInstance: s.instanceID, Phase: "prepared",
	}
	if err := writeRootMarker(rootState.LeaseRoot, rootMarker{TokenHash: tokenHash, LeaseID: leaseID, TargetFingerprint: target}); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("leased root cannot be checkpointed"))
	}
	if err := writeCheckpoint(path, checkpoint); err != nil {
		_ = os.Remove(filepath.Join(rootState.LeaseRoot, rootMarkerName))
		return nil, connect.NewError(connect.CodeInternal, errors.New("persist restart checkpoint"))
	}
	s.ownerLocks[leaseID] = ownerLock
	keepOwnerLock = true
	s.checkpoints[leaseID] = checkpointRef{ticket: ticket, path: path}
	return connect.NewResponse(prepareResponse(ticket, checkpoint)), nil
}

func (s *Service) ResumeTestPool(ctx context.Context, req *connect.Request[routingv1.ResumeTestPoolRequest]) (*connect.Response[routingv1.ResumeTestPoolResponse], error) {
	if s == nil || s.db == nil || s.roots == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("paired database and file routing are required"))
	}
	s.opMu.Lock()
	defer s.opMu.Unlock()
	leaseID, ticket := req.Msg.GetLeaseId(), req.Msg.GetRestartTicket()
	if leaseID == "" || ticket == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("lease_id and restart_ticket are required"))
	}
	release, lockErr := acquireLeaseOperationLock(leaseID)
	if lockErr != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("another routing operation is in progress"))
	}
	defer release()
	target := targetFingerprint(s.db.PrimaryFingerprint(), s.roots.PrimaryFingerprint())
	path, err := checkpointPathFor(target, leaseID)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("restart checkpoint is invalid"))
	}
	var checkpoint restartCheckpoint
	if err := readCheckpoint(path, ticket, &checkpoint); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("restart checkpoint is unavailable"))
	}
	if checkpoint.LeaseID != leaseID || checkpoint.Phase == "clearing" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("restart checkpoint does not match the lease"))
	}
	if checkpoint.Phase != "prepared" && checkpoint.Phase != "resuming" && checkpoint.Phase != "resumed" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("restart checkpoint phase is invalid"))
	}
	if target != checkpoint.TargetFingerprint || s.db.PrimaryFingerprint() != checkpoint.PrimaryDBFingerprint || s.roots.PrimaryFingerprint() != checkpoint.RootsFingerprint {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("restart target identity mismatch"))
	}
	if !checkpoint.ExpiresAt.After(time.Now()) || !checkpoint.Database.ExpiresAt.After(time.Now()) || !checkpoint.Roots.ExpiresAt.After(time.Now()) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("restart checkpoint expired"))
	}
	if err := verifyRootMarker(checkpoint.Roots.LeaseRoot, rootMarker{TokenHash: checkpoint.TokenHash, LeaseID: leaseID, TargetFingerprint: checkpoint.TargetFingerprint}); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("retained leased root does not match checkpoint"))
	}
	if checkpoint.Phase == "resumed" {
		if checkpoint.ResumeInstance != s.instanceID || !s.checkpointBindingMatches(checkpoint) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("restart checkpoint was already consumed"))
		}
		s.checkpoints[leaseID] = checkpointRef{ticket: ticket, path: path}
		return connect.NewResponse(resumeResponse(checkpoint)), nil
	}
	if checkpoint.Phase == "resuming" {
		if checkpoint.ResumeInstance != s.instanceID {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("restart checkpoint has an unresolved prior resume"))
		}
		if s.checkpointBindingMatches(checkpoint) {
			checkpoint.Phase = "resumed"
			if err := writeCheckpoint(path, checkpoint); err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.New("persist resumed checkpoint"))
			}
			s.checkpoints[leaseID] = checkpointRef{ticket: ticket, path: path}
			return connect.NewResponse(resumeResponse(checkpoint)), nil
		}
	}
	if checkpoint.PreparedInstance == s.instanceID || s.db.HasTestPool() || s.roots.HasTestRoots() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("the preparing instance or another lease is still active"))
	}
	checkpoint.Phase, checkpoint.ResumeInstance = "resuming", s.instanceID
	if err := writeCheckpoint(path, checkpoint); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("persist resume intent"))
	}
	if err := s.db.ResumeTestPool(ctx, checkpoint.Database); err != nil {
		checkpoint.Phase, checkpoint.ResumeInstance = "prepared", ""
		_ = writeCheckpoint(path, checkpoint)
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("retained database binding could not be reopened"))
	}
	if err := s.roots.ResumeTestRoots(checkpoint.Roots, checkpoint.RootsFingerprint); err != nil {
		// Keep the durable resuming state if rollback cannot prove both halves
		// are clear. A retry must diagnose this exact partial attachment.
		if clearErr := s.db.ClearTestPool(leaseID); clearErr == nil {
			checkpoint.Phase, checkpoint.ResumeInstance = "prepared", ""
			_ = writeCheckpoint(path, checkpoint)
		}
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("retained file binding could not be reopened"))
	}
	if err := s.pinLeaseBinding(leaseID); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("resumed route could not be pinned"))
	}
	checkpoint.Phase = "resumed"
	if err := writeCheckpoint(path, checkpoint); err != nil {
		// Keep the live route and resuming record. A same-process retry can
		// finish the durable receipt without opening another pool generation.
		return nil, connect.NewError(connect.CodeInternal, errors.New("persist resumed checkpoint"))
	}
	s.checkpoints[leaseID] = checkpointRef{ticket: ticket, path: path}
	return connect.NewResponse(resumeResponse(checkpoint)), nil
}

func prepareResponse(ticket string, checkpoint restartCheckpoint) *routingv1.PrepareTestPoolRestartResponse {
	return &routingv1.PrepareTestPoolRestartResponse{
		RestartTicket: ticket, ExpiresAtUnixMs: checkpoint.ExpiresAt.UnixMilli(),
		PreRestartStats: protoStats(checkpoint.Database.Stats, checkpoint.Roots.Stats),
	}
}

func resumeResponse(checkpoint restartCheckpoint) *routingv1.ResumeTestPoolResponse {
	return &routingv1.ResumeTestPoolResponse{
		ActiveLeaseId: checkpoint.LeaseID, ExpiresAtUnixMs: checkpoint.ExpiresAt.UnixMilli(),
		PreRestartStats: protoStats(checkpoint.Database.Stats, checkpoint.Roots.Stats), FileRootsResumed: true,
	}
}

func (s *Service) pinLeaseBinding(leaseID string) error {
	ctx, err := s.db.PinTestPool(context.Background(), leaseID)
	if err != nil {
		return err
	}
	ctx, err = s.roots.PinTestRoots(ctx, leaseID)
	if err != nil {
		return err
	}
	s.bindings[leaseID] = ctx
	return nil
}

func (s *Service) checkpointBindingMatches(checkpoint restartCheckpoint) bool {
	leaseID := checkpoint.LeaseID
	dbState, dbErr := s.db.TestPoolRestartState(leaseID)
	rootState, rootErr := s.roots.TestRootsRestartState(leaseID)
	if dbErr != nil || rootErr != nil || dbState.DSN != checkpoint.Database.DSN || dbState.PrimaryFingerprint != checkpoint.PrimaryDBFingerprint || rootState.LeaseRoot != checkpoint.Roots.LeaseRoot || rootState.PrimaryFingerprint != checkpoint.RootsFingerprint || targetFingerprint(dbState.PrimaryFingerprint, rootState.PrimaryFingerprint) != checkpoint.TargetFingerprint {
		return false
	}
	if verifyRootMarker(rootState.LeaseRoot, rootMarker{TokenHash: checkpoint.TokenHash, LeaseID: leaseID, TargetFingerprint: checkpoint.TargetFingerprint}) != nil {
		return false
	}
	ctx, ok := s.bindings[leaseID]
	if !ok {
		return false
	}
	if _, err := s.db.PinTestPool(ctx, leaseID); err != nil {
		return false
	}
	if _, err := s.roots.PinTestRoots(ctx, leaseID); err != nil {
		return false
	}
	if checkpoint.Phase == "prepared" {
		return checkpoint.PreparedInstance == s.instanceID
	}
	return (checkpoint.Phase == "resuming" || checkpoint.Phase == "resumed") && checkpoint.ResumeInstance == s.instanceID
}

func protoStats(db database.LeaseStatsSnapshot, roots filerouting.LeaseStatsSnapshot) *routingv1.LeaseStats {
	return &routingv1.LeaseStats{
		TestPoolRequests: db.TestPoolRequests, PrimaryDuringTestModeRequests: db.PrimaryDuringTestModeRequests,
		TestRootWrites: roots.TestRootWrites, PrimaryRootWritesDuringTestMode: roots.PrimaryWritesDuringTestMode,
	}
}

const rootMarkerName = ".vrooli-devrouting-checkpoint"

func writeRootMarker(root string, marker rootMarker) error {
	if root == "" {
		return errors.New("empty retained root")
	}
	path := filepath.Join(root, rootMarkerName)
	body, err := json.Marshal(marker)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(body)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func verifyRootMarker(root string, want rootMarker) error {
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("retained root unavailable")
	}
	markerPath := filepath.Join(root, rootMarkerName)
	markerInfo, err := os.Lstat(markerPath)
	if err != nil || !markerInfo.Mode().IsRegular() || markerInfo.Mode()&os.ModeSymlink != 0 || markerInfo.Size() > 4096 {
		return errors.New("retained root marker unavailable")
	}
	body, err := os.ReadFile(markerPath)
	if err != nil {
		return err
	}
	var got rootMarker
	if err := json.Unmarshal(body, &got); err != nil || got != want {
		return errors.New("retained root marker mismatch")
	}
	return nil
}

func randomToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

func hashToken(ticket string) string {
	digest := sha256.Sum256([]byte(ticket))
	return hex.EncodeToString(digest[:])
}

func checkpointPathFor(targetFingerprint, leaseID string) (string, error) {
	if len(targetFingerprint) != 64 || strings.Trim(targetFingerprint, "0123456789abcdef") != "" || leaseID == "" {
		return "", errors.New("invalid checkpoint identity")
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir, err := checkpointNamespacePath(base)
	if err != nil {
		return "", err
	}
	key := sha256.Sum256([]byte(targetFingerprint + "\x00" + leaseID))
	return filepath.Join(dir, hex.EncodeToString(key[:])+".json"), nil
}

func checkpointPathForService(s *Service, leaseID string) (string, error) {
	if s == nil || s.db == nil || s.roots == nil {
		return "", errors.New("paired routing is unavailable")
	}
	return checkpointPathFor(targetFingerprint(s.db.PrimaryFingerprint(), s.roots.PrimaryFingerprint()), leaseID)
}

func targetFingerprint(databaseFingerprint, rootsFingerprint string) string {
	digest := sha256.Sum256([]byte(databaseFingerprint + "\x00" + rootsFingerprint))
	return hex.EncodeToString(digest[:])
}

func checkpointNamespacePath(base string) (string, error) {
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", err
	}
	info, err := os.Lstat(base)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("user cache root is unavailable")
	}
	dir := base
	for _, segment := range strings.Split(checkpointNamespace, "/") {
		dir = filepath.Join(dir, segment)
		if err := ensurePrivateDir(dir); err != nil {
			return "", err
		}
	}
	return dir, nil
}

func ensurePrivateDir(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("checkpoint namespace is not a private directory")
	}
	return os.Chmod(path, 0o700)
}

func acquireLeaseOperationLock(leaseID string) (func(), error) {
	if leaseID == "" {
		return nil, errors.New("lease id is required")
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	dir, err := checkpointNamespacePath(base)
	if err != nil {
		return nil, err
	}
	key := sha256.Sum256([]byte(leaseID))
	path := filepath.Join(dir, "operation-"+hex.EncodeToString(key[:])+".lock")
	if err := ensurePrivateFile(path); err != nil {
		return nil, err
	}
	return platform.AcquireFileLock(path)
}

func checkpointOwnerLockPath(checkpointPath string) string {
	return checkpointPath + ".owner.lock"
}

func acquireCheckpointOwnerLock(checkpointPath string) (checkpointOwnerLock, error) {
	path := checkpointOwnerLockPath(checkpointPath)
	if err := ensurePrivateFile(path); err != nil {
		return checkpointOwnerLock{}, err
	}
	release, err := platform.AcquireFileLock(path)
	return checkpointOwnerLock{path: path, release: release}, err
}

func (s *Service) ownsCheckpointLock(leaseID string) bool {
	_, ok := s.ownerLocks[leaseID]
	return ok
}

// checkpointOwnerExited probes the kernel lock held for the lifetime of a
// prepared checkpoint. A missing sidecar is a legacy checkpoint; callers
// clearing those must rely on the lifecycle owner having confirmed process
// exit, because the old checkpoint format carried no independently verifiable
// process-liveness proof.
func checkpointOwnerExited(checkpointPath string) bool {
	path := checkpointOwnerLockPath(checkpointPath)
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		return true
	} else if err != nil {
		return false
	}
	if err := ensurePrivateFile(path); err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	release, err := platform.AcquireFileLockContext(ctx, path)
	if err != nil {
		return false
	}
	release()
	return true
}

func ensurePrivateFile(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		file, createErr := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
		if createErr != nil {
			return createErr
		}
		return file.Close()
	}
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("lock path is not a regular file")
	}
	return os.Chmod(path, 0o600)
}

func readCheckpoint(path, ticket string, out *restartCheckpoint) error {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 || info.Size() < 1 || info.Size() > 1<<20 {
		return errors.New("checkpoint file unavailable")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return errors.New("checkpoint file unavailable")
	}
	if err := json.Unmarshal(body, out); err != nil {
		return err
	}
	if out.Version != 1 || len(ticket) != 64 || strings.Trim(ticket, "0123456789abcdef") != "" || out.TokenHash != hashToken(ticket) {
		return errors.New("checkpoint binding mismatch")
	}
	return nil
}

func writeCheckpoint(path string, checkpoint restartCheckpoint) error {
	body, err := json.Marshal(checkpoint)
	if err != nil {
		return err
	}
	if len(body) > 1<<20 {
		return errors.New("checkpoint exceeds size limit")
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".checkpoint-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func clearCheckpoint(path string) error {
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove restart checkpoint: %w", err)
	}
	return nil
}
