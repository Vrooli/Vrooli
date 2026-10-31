// Package devrouting wires the dev-only RoutingService proto contract to a
// RoutedDB, so an external orchestrator (test-genie) can install and clear a
// runtime test DB pool on a live scenario.
//
// The service is gated by projectmeta.IsDevelopment(): Register is a no-op in
// production mode, so the RPC path returns 404 to any caller. The
// VROOLI_TEST_MODE_FORCE_ENABLE escape hatch from package apihttp also opens
// this service.
package devrouting

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/projectmeta"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing"
	"github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing/routing_v1connect"
)

// Service implements the generated routing_v1connect.RoutingServiceHandler
// by forwarding calls into a *database.RoutedDB.
type Service struct {
	db          *database.RoutedDB
	roots       *filerouting.RoutedRoots
	opMu        sync.Mutex
	instanceID  string
	checkpoints map[string]checkpointRef
	bindings    map[string]context.Context
	ownerLocks  map[string]checkpointOwnerLock
}

// FileService implements the routing lease for scenarios that persist only
// files. It deliberately does not manufacture a database pool: its sole
// authority is the leased RoutedRoots instance.
type FileService struct {
	roots *filerouting.RoutedRoots
}

// NewService returns a Service bound to db.
func NewService(db *database.RoutedDB, roots ...*filerouting.RoutedRoots) *Service {
	instanceID, _ := randomToken()
	service := &Service{db: db, instanceID: instanceID, checkpoints: make(map[string]checkpointRef), bindings: make(map[string]context.Context), ownerLocks: make(map[string]checkpointOwnerLock)}
	if len(roots) > 0 {
		service.roots = roots[0]
	}
	return service
}

// NewFileService returns a routing service for a file-only scenario.
func NewFileService(roots *filerouting.RoutedRoots) *FileService {
	return &FileService{roots: roots}
}

func (s *FileService) InstallTestPool(_ context.Context, req *connect.Request[routingv1.InstallTestPoolRequest]) (*connect.Response[routingv1.InstallTestPoolResponse], error) {
	if s == nil || s.roots == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("file routing roots are unavailable"))
	}
	ttl := time.Duration(req.Msg.GetLeaseTtlMs()) * time.Millisecond
	if _, err := s.roots.InstallLeasedTestRoots(req.Msg.GetLeaseId(), ttl, req.Msg.GetEmptyConfig()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("install test file roots: %w", err))
	}
	return connect.NewResponse(&routingv1.InstallTestPoolResponse{ActiveLeaseId: req.Msg.GetLeaseId(), FileRootsInstalled: true}), nil
}

func (s *FileService) ClearTestPool(_ context.Context, req *connect.Request[routingv1.ClearTestPoolRequest]) (*connect.Response[routingv1.ClearTestPoolResponse], error) {
	if s == nil || s.roots == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("file routing roots are unavailable"))
	}
	snapshot := s.roots.LeaseStats()
	if err := s.roots.ClearTestRoots(req.Msg.GetLeaseId()); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&routingv1.ClearTestPoolResponse{Stats: &routingv1.LeaseStats{TestRootWrites: snapshot.TestRootWrites, PrimaryRootWritesDuringTestMode: snapshot.PrimaryWritesDuringTestMode}}), nil
}

func (s *FileService) HeartbeatTestPool(_ context.Context, req *connect.Request[routingv1.HeartbeatTestPoolRequest]) (*connect.Response[routingv1.HeartbeatTestPoolResponse], error) {
	if s == nil || s.roots == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("file routing roots are unavailable"))
	}
	expiresAt, err := s.roots.HeartbeatTestRoots(req.Msg.GetLeaseId(), 0)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&routingv1.HeartbeatTestPoolResponse{ExpiresAtUnixMs: expiresAt.UnixMilli()}), nil
}

// InstallTestPool implements routing_v1connect.RoutingServiceHandler.
//
// Lease conflicts map to connect.CodeAlreadyExists; the response carries
// the active lease so callers can surface a useful error. Other failures
// map to CodeInternal.
func (s *Service) InstallTestPool(ctx context.Context, req *connect.Request[routingv1.InstallTestPoolRequest]) (*connect.Response[routingv1.InstallTestPoolResponse], error) {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	leaseID := req.Msg.GetLeaseId()
	release, lockErr := acquireLeaseOperationLock(leaseID)
	if lockErr != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("another routing operation is in progress"))
	}
	defer release()
	if _, prepared := s.checkpoints[leaseID]; prepared {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("lease is checkpointed for restart"))
	}
	if s.roots != nil {
		checkpointPath, pathErr := checkpointPathForService(s, leaseID)
		if pathErr != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("restart checkpoint namespace is unavailable"))
		}
		if _, statErr := os.Lstat(checkpointPath); statErr == nil || !os.IsNotExist(statErr) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("lease has a pending restart checkpoint"))
		}
	}
	ttl := time.Duration(req.Msg.GetLeaseTtlMs()) * time.Millisecond
	if err := s.db.InstallTestPool(ctx, req.Msg.GetDsn(), leaseID, ttl); err != nil {
		var conflict *database.ErrLeaseConflict
		if errors.As(err, &conflict) {
			return nil, connect.NewError(connect.CodeAlreadyExists, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if s.roots != nil {
		if _, err := s.roots.InstallLeasedTestRoots(leaseID, ttl, req.Msg.GetEmptyConfig()); err != nil {
			_ = s.db.ClearTestPool(leaseID)
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("install test file roots: %w", err))
		}
	}
	return connect.NewResponse(&routingv1.InstallTestPoolResponse{ActiveLeaseId: leaseID, FileRootsInstalled: s.roots != nil}), nil
}

// ClearTestPool implements routing_v1connect.RoutingServiceHandler.
//
// Lease mismatches map to connect.CodeFailedPrecondition. The response
// carries a LeaseStats snapshot taken just before the clear so callers
// (test-genie) can detect that the routed run actually exercised the test
// pool.
func (s *Service) ClearTestPool(ctx context.Context, req *connect.Request[routingv1.ClearTestPoolRequest]) (*connect.Response[routingv1.ClearTestPoolResponse], error) {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	leaseID := req.Msg.GetLeaseId()
	release, lockErr := acquireLeaseOperationLock(leaseID)
	if lockErr != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("another routing operation is in progress"))
	}
	defer release()
	ref := s.checkpoints[leaseID]
	var checkpoint *restartCheckpoint
	if ticket := req.Msg.GetRestartTicket(); ticket != "" {
		path, pathErr := checkpointPathForService(s, leaseID)
		if pathErr != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("restart checkpoint is invalid"))
		}
		var loaded restartCheckpoint
		if err := readCheckpoint(path, ticket, &loaded); err != nil || loaded.LeaseID != leaseID {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("restart checkpoint does not match the lease"))
		}
		ref = checkpointRef{ticket: ticket, path: path}
		checkpoint = &loaded
	} else if ref.ticket != "" {
		var loaded restartCheckpoint
		if err := readCheckpoint(ref.path, ref.ticket, &loaded); err != nil || loaded.LeaseID != leaseID {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("restart checkpoint state is unavailable"))
		}
		checkpoint = &loaded
	}
	if checkpoint != nil {
		if checkpoint.Phase != "clearing" {
			if checkpoint.Phase == "prepared" && checkpointExpired(*checkpoint, time.Now()) {
				if err := s.validateExpiredPreparedCleanup(*checkpoint); err != nil {
					return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("expired prepared checkpoint is not safe to clear"))
				}
			} else if !s.checkpointBindingMatches(*checkpoint) || (checkpoint.Phase == "prepared" && checkpointExpired(*checkpoint, time.Now())) {
				return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("restart lease binding is not active on this instance"))
			}
			checkpoint.Phase = "clearing"
			if err := writeCheckpoint(ref.path, *checkpoint); err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.New("persist restart cleanup intent"))
			}
		} else {
			if err := s.checkpointClearBindings(*checkpoint); err != nil {
				return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("restart cleanup binding does not match the checkpoint"))
			}
			if !s.ownsCheckpointLock(leaseID) && !checkpointOwnerExited(ref.path) {
				return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("prepared checkpoint owner process is still active"))
			}
		}
	}
	snapshot := s.db.LeaseStats()
	fileSnapshot := filerouting.LeaseStatsSnapshot{}
	if s.roots != nil {
		fileSnapshot = s.roots.LeaseStats()
	}
	dbActive := s.db.ActiveLeaseID() == leaseID
	if checkpoint == nil || dbActive {
		if err := s.db.ClearTestPool(leaseID); err != nil {
			var mismatch *database.ErrLeaseMismatch
			if errors.As(err, &mismatch) {
				return nil, connect.NewError(connect.CodeFailedPrecondition, err)
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	if s.roots != nil {
		if checkpoint == nil || s.roots.ActiveLeaseID() == leaseID {
			if err := s.roots.ClearTestRoots(leaseID); err != nil {
				return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("leased file roots could not be removed; retry cleanup"))
			}
		} else if checkpoint.Phase == "clearing" {
			if err := s.roots.ClearRetainedTestRoots(checkpoint.Roots, checkpoint.RootsFingerprint); err != nil {
				return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("detached checkpoint roots could not be removed; retry cleanup"))
			}
		}
	}
	if err := clearCheckpoint(ref.path); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("routing was cleared but its checkpoint record remains; retry cleanup"))
	}
	if lock, ok := s.ownerLocks[leaseID]; ok {
		lock.release()
		delete(s.ownerLocks, leaseID)
	}
	if err := os.Remove(checkpointOwnerLockPath(ref.path)); err != nil && !os.IsNotExist(err) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("routing was cleared but its owner lock remains; retry cleanup"))
	}
	delete(s.checkpoints, leaseID)
	delete(s.bindings, leaseID)
	return connect.NewResponse(&routingv1.ClearTestPoolResponse{
		Stats: &routingv1.LeaseStats{
			TestPoolRequests:                snapshot.TestPoolRequests,
			PrimaryDuringTestModeRequests:   snapshot.PrimaryDuringTestModeRequests,
			TestRootWrites:                  fileSnapshot.TestRootWrites,
			PrimaryRootWritesDuringTestMode: fileSnapshot.PrimaryWritesDuringTestMode,
		},
	}), nil
}

func checkpointExpired(checkpoint restartCheckpoint, now time.Time) bool {
	return !checkpoint.ExpiresAt.After(now) && !checkpoint.Database.ExpiresAt.After(now) && !checkpoint.Roots.ExpiresAt.After(now)
}

func (s *Service) validateExpiredPreparedCleanup(checkpoint restartCheckpoint) error {
	if checkpoint.Phase != "prepared" || checkpoint.PreparedInstance == "" || checkpoint.PreparedInstance == s.instanceID || !checkpointExpired(checkpoint, time.Now()) {
		return errors.New("checkpoint is live or belongs to this owner")
	}
	if !s.ownsCheckpointLock(checkpoint.LeaseID) && !checkpointOwnerExited(checkpointPathForPrepared(s, checkpoint)) {
		return errors.New("prepared checkpoint owner process is still active")
	}
	if s.db.ActiveLeaseID() != "" || s.roots.ActiveLeaseID() != "" || s.bindings[checkpoint.LeaseID] != nil {
		return errors.New("a database or file lease is still active")
	}
	if s.db.PrimaryFingerprint() == "" || s.roots.PrimaryFingerprint() == "" ||
		checkpoint.PrimaryDBFingerprint != s.db.PrimaryFingerprint() || checkpoint.RootsFingerprint != s.roots.PrimaryFingerprint() ||
		checkpoint.Database.PrimaryFingerprint != checkpoint.PrimaryDBFingerprint || checkpoint.Roots.PrimaryFingerprint != checkpoint.RootsFingerprint ||
		checkpoint.TargetFingerprint != targetFingerprint(s.db.PrimaryFingerprint(), s.roots.PrimaryFingerprint()) {
		return errors.New("checkpoint target identity changed")
	}
	return verifyRootMarker(checkpoint.Roots.LeaseRoot, rootMarker{TokenHash: checkpoint.TokenHash, LeaseID: checkpoint.LeaseID, TargetFingerprint: checkpoint.TargetFingerprint})
}

func checkpointPathForPrepared(s *Service, checkpoint restartCheckpoint) string {
	if ref := s.checkpoints[checkpoint.LeaseID]; ref.path != "" {
		return ref.path
	}
	path, _ := checkpointPathFor(checkpoint.TargetFingerprint, checkpoint.LeaseID)
	return path
}

// checkpointClearBindings permits a clearing retry after either owner has
// detached its route, while proving every still-active half remains the exact
// DB/root binding checkpointed before cleanup began.
func (s *Service) checkpointClearBindings(checkpoint restartCheckpoint) error {
	leaseID := checkpoint.LeaseID
	if dbLease := s.db.ActiveLeaseID(); dbLease != "" {
		if dbLease != leaseID {
			return errors.New("another database lease is active")
		}
		state, err := s.db.TestPoolRestartState(leaseID)
		if err != nil || state.DSN != checkpoint.Database.DSN || state.PrimaryFingerprint != checkpoint.PrimaryDBFingerprint {
			return errors.New("active database binding changed")
		}
		ctx, ok := s.bindings[leaseID]
		if !ok {
			return errors.New("database pin is unavailable")
		}
		if _, err := s.db.PinTestPool(ctx, leaseID); err != nil {
			return errors.New("database generation changed")
		}
	}
	if s.roots == nil {
		return errors.New("file routing is unavailable")
	}
	if rootLease := s.roots.ActiveLeaseID(); rootLease != "" {
		if rootLease != leaseID {
			return errors.New("another file lease is active")
		}
		state, err := s.roots.TestRootsRestartState(leaseID)
		if err != nil || state.LeaseRoot != checkpoint.Roots.LeaseRoot || state.PrimaryFingerprint != checkpoint.RootsFingerprint {
			return errors.New("active file binding changed")
		}
		ctx, ok := s.bindings[leaseID]
		if !ok {
			return errors.New("file pin is unavailable")
		}
		if _, err := s.roots.PinTestRoots(ctx, leaseID); err != nil {
			return errors.New("file generation changed")
		}
	} else {
		if _, err := os.Lstat(checkpoint.Roots.LeaseRoot); err == nil {
			if err := verifyRootMarker(checkpoint.Roots.LeaseRoot, rootMarker{TokenHash: checkpoint.TokenHash, LeaseID: leaseID, TargetFingerprint: checkpoint.TargetFingerprint}); err != nil {
				return errors.New("detached retained file tree marker changed")
			}
		} else if !os.IsNotExist(err) {
			return errors.New("retained file tree state is unavailable")
		}
	}
	return nil
}

// HeartbeatTestPool extends the active lease's expiry. Lease mismatches
// map to connect.CodeFailedPrecondition.
func (s *Service) HeartbeatTestPool(ctx context.Context, req *connect.Request[routingv1.HeartbeatTestPoolRequest]) (*connect.Response[routingv1.HeartbeatTestPoolResponse], error) {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	release, lockErr := acquireLeaseOperationLock(req.Msg.GetLeaseId())
	if lockErr != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("another routing operation is in progress"))
	}
	defer release()
	expiresAt, err := s.db.HeartbeatTestPool(req.Msg.GetLeaseId(), 0)
	if err != nil {
		var mismatch *database.ErrLeaseMismatch
		if errors.As(err, &mismatch) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if s.roots != nil {
		rootExpiry, err := s.roots.HeartbeatTestRoots(req.Msg.GetLeaseId(), 0)
		if err != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		if rootExpiry.Before(expiresAt) {
			expiresAt = rootExpiry
		}
	}
	if ref := s.checkpoints[req.Msg.GetLeaseId()]; ref.ticket != "" {
		var checkpoint restartCheckpoint
		if err := readCheckpoint(ref.path, ref.ticket, &checkpoint); err != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("restart checkpoint state is unavailable"))
		}
		dbState, dbErr := s.db.TestPoolRestartState(req.Msg.GetLeaseId())
		rootState, rootErr := s.roots.TestRootsRestartState(req.Msg.GetLeaseId())
		if dbErr != nil || rootErr != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("paired lease checkpoint could not be refreshed"))
		}
		// Keep the original pre-restart counters immutable. Heartbeat only
		// extends the private binding expiries.
		dbState.Stats = checkpoint.Database.Stats
		rootState.Stats = checkpoint.Roots.Stats
		checkpoint.Database, checkpoint.Roots = dbState, rootState
		if rootState.ExpiresAt.Before(expiresAt) {
			expiresAt = rootState.ExpiresAt
		}
		checkpoint.ExpiresAt = expiresAt
		if err := writeCheckpoint(ref.path, checkpoint); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.New("refresh restart checkpoint"))
		}
	}
	return connect.NewResponse(&routingv1.HeartbeatTestPoolResponse{
		ExpiresAtUnixMs: expiresAt.UnixMilli(),
	}), nil
}

func (s *FileService) PrepareTestPoolRestart(context.Context, *connect.Request[routingv1.PrepareTestPoolRestartRequest]) (*connect.Response[routingv1.PrepareTestPoolRestartResponse], error) {
	return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("restart checkpoints require a paired database and file lease"))
}

func (s *FileService) ResumeTestPool(context.Context, *connect.Request[routingv1.ResumeTestPoolRequest]) (*connect.Response[routingv1.ResumeTestPoolResponse], error) {
	return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("restart checkpoints require a paired database and file lease"))
}

// Mux is the minimal interface satisfied by *http.ServeMux and the
// gorilla/mux router so callers can pass whichever router they already use.
type Mux interface {
	Handle(pattern string, handler http.Handler)
}

// mounter is implemented by routers, such as chi.Router, that distinguish a
// mounted subtree from a single exact handler path. Connect handlers own a
// service-prefix subtree because every RPC method is a child of that prefix.
// net/http.ServeMux treats a trailing slash as a prefix, while chi requires
// Mount explicitly; keeping this capability optional preserves both callers.
type mounter interface {
	Mount(pattern string, handler http.Handler)
}

// Register mounts the RoutingService Connect handler onto mux, but only if the
// project is running in development mode (or the
// VROOLI_TEST_MODE_FORCE_ENABLE escape hatch is set). In production mode
// Register is a no-op and the RPC path returns whatever 404 the underlying
// mux serves for unknown paths.
//
// Register returns true when it actually mounted the handler. Production
// callers can use the return value to log/observe that the dev surface is
// disabled.
func Register(mux Mux, db *database.RoutedDB, opts ...connect.HandlerOption) bool {
	if !enabled() {
		return false
	}
	path, handler := routing_v1connect.NewRoutingServiceHandler(NewService(db), opts...)
	mountService(mux, path, handler)
	return true
}

// RegisterWithFileRoots mounts the development routing surface with both the
// database and file-storage legs of a test lease. Existing database-only
// adopters remain valid through Register.
func RegisterWithFileRoots(mux Mux, db *database.RoutedDB, roots *filerouting.RoutedRoots, opts ...connect.HandlerOption) bool {
	if !enabled() {
		return false
	}
	path, handler := routing_v1connect.NewRoutingServiceHandler(NewService(db, roots), opts...)
	mountService(mux, path, handler)
	return true
}

// RegisterFileRoots mounts the development routing surface for a file-only
// scenario. It is intentionally separate from RegisterWithFileRoots so a
// scenario never needs an unused SQL database merely to isolate file writes.
func RegisterFileRoots(mux Mux, roots *filerouting.RoutedRoots, opts ...connect.HandlerOption) bool {
	if !enabled() || roots == nil {
		return false
	}
	path, handler := routing_v1connect.NewRoutingServiceHandler(NewFileService(roots), opts...)
	mountService(mux, path, handler)
	return true
}

// RegisterWithFileRootsService registers an explicitly supplied routing
// service while retaining the same development-only guard and leased-file-root
// contract as RegisterWithFileRoots. It supports scenarios whose relational
// engine needs to provision a compatible test database before delegating to
// the shared RoutedDB service.
func RegisterWithFileRootsService(mux Mux, db *database.RoutedDB, roots *filerouting.RoutedRoots, service routing_v1connect.RoutingServiceHandler, opts ...connect.HandlerOption) bool {
	if !enabled() || db == nil || roots == nil || service == nil {
		return false
	}
	path, handler := routing_v1connect.NewRoutingServiceHandler(service, opts...)
	mountService(mux, path, handler)
	return true
}

func mountService(mux Mux, path string, handler http.Handler) {
	if subtree, ok := mux.(mounter); ok {
		subtree.Mount(path, handler)
		return
	}
	mux.Handle(path, handler)
}

func enabled() bool {
	return projectmeta.IsDevelopment() || os.Getenv(apihttp.TestModeForceEnableEnv) == "1"
}
