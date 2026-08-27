package privilegebroker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/vrooli/vrooli/internal/tuning"
)

const (
	brokerParameterA = 10
)

const (
	brokerParameterB = 16
)

// Config is deliberately small: setup owns all filesystem and systemd policy.
type Config struct {
	SocketPath        string
	AllowedUID        uint32
	SocketGID         int
	AuditPath         string
	Executor          Executor
	RuntimeHomeRepair func(context.Context, RuntimeHomeSubject) Result
}

// Executor is the narrow test seam. Implementations receive only broker-built
// argv; they never receive a command string or shell fragment.
type Executor interface {
	Run(context.Context, string, ...string) ([]byte, error)
}

// Broker serves one validated request per local Unix connection.
type Broker struct {
	config Config
}

func New(config Config) (*Broker, error) {
	if config.SocketPath == "" {
		return nil, fmt.Errorf("socket path is required")
	}
	if config.Executor == nil {
		config.Executor = OSExecutor{}
	}
	return &Broker{config: config}, nil
}

// Serve runs until ctx is cancelled. It only listens on a Unix socket and
// refuses to start when not running as root.
func (b *Broker) Serve(ctx context.Context) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("privilege broker must run as root")
	}
	if err := os.MkdirAll(filepath.Dir(b.config.SocketPath), tuning.PermGroupDir); err != nil {
		return fmt.Errorf("create socket directory: %w", err)
	}
	if err := os.Remove(b.config.SocketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove stale socket: %w", err)
	}
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: b.config.SocketPath, Net: "unix"})
	if err != nil {
		return fmt.Errorf("listen broker socket: %w", err)
	}
	defer func() {
		_ = listener.Close()
		_ = os.Remove(b.config.SocketPath)
	}()
	if err := os.Chmod(b.config.SocketPath, tuning.PermSocket); err != nil {
		return fmt.Errorf("set socket mode: %w", err)
	}
	if b.config.SocketGID >= 0 {
		if err := os.Chown(b.config.SocketPath, 0, b.config.SocketGID); err != nil {
			return fmt.Errorf("set socket ownership: %w", err)
		}
	}
	var closeOnce sync.Once
	go func() {
		<-ctx.Done()
		closeOnce.Do(func() { _ = listener.Close() })
	}()
	for {
		conn, err := listener.AcceptUnix()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("accept broker socket: %w", err)
		}
		go b.handle(ctx, conn)
	}
}

func (b *Broker) handle(ctx context.Context, conn *net.UnixConn) {
	defer conn.Close()
	uid, err := peerUID(conn)
	if err != nil || uid != b.config.AllowedUID {
		_ = json.NewEncoder(conn).Encode(NewFailure("", "", "caller_not_authorized"))
		return
	}
	decoder := json.NewDecoder(io.LimitReader(conn, brokerParameterB<<brokerParameterA))
	decoder.DisallowUnknownFields()
	var req Request
	if err := decoder.Decode(&req); err != nil {
		_ = json.NewEncoder(conn).Encode(NewFailure("", "", "invalid_request"))
		return
	}
	result := b.Execute(ctx, req, uid)
	_ = json.NewEncoder(conn).Encode(result)
}

// Execute validates a typed request, invokes the fixed adapter, and appends a
// redacted audit record. It is exported for controlled integration tests.
func (b *Broker) Execute(ctx context.Context, req Request, callerUID uint32) Result {
	if callerUID != b.config.AllowedUID {
		return NewFailure(req.RequestID, req.Action, "caller_not_authorized")
	}
	if err := Validate(req); err != nil {
		return NewFailure(req.RequestID, req.Action, err.Error())
	}
	if req.Action == ActionRuntimeHomeOwnershipRepair && req.RuntimeHome.ExpectedUID != callerUID {
		return NewFailure(req.RequestID, req.Action, "runtime_home_identity_not_caller")
	}
	result := b.dispatch(ctx, req)
	b.audit(callerUID, req, result)
	return result
}

// dispatch routes a validated request to its action family's adapter. There is
// no default branch that executes anything: an action Validate did not
// recognise never reaches here.
func (b *Broker) dispatch(ctx context.Context, req Request) Result {
	switch req.Action {
	case ActionVolumeFilesystemCheck, ActionVolumeFilesystemRepair:
		return executeVolume(ctx, b.config.Executor, req)
	case ActionRuntimeHomeOwnershipRepair:
		if b.config.RuntimeHomeRepair == nil {
			return NewFailure(req.RequestID, req.Action, "runtime_home_repair_unavailable")
		}
		return b.config.RuntimeHomeRepair(ctx, *req.RuntimeHome)
	default:
		return executeUFW(ctx, b.config.Executor, req)
	}
}

func (b *Broker) audit(uid uint32, req Request, result Result) {
	if b.config.AuditPath == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(b.config.AuditPath), tuning.PermGroupDir); err != nil {
		return
	}
	f, err := os.OpenFile(b.config.AuditPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, tuning.PermSecret)
	if err != nil {
		return
	}
	defer f.Close()
	// No request body, credentials, environment, or command output is logged.
	_ = json.NewEncoder(f).Encode(struct {
		CallerUID uint32 `json:"caller_uid"`
		Action    string `json:"action"`
		Scenario  string `json:"scenario"`
		Candidate string `json:"candidate_ip"`
		Port      int    `json:"port"`
		Status    string `json:"status"`
		Code      string `json:"code,omitempty"`
	}{uid, req.Action, req.Subject.Scenario, req.Subject.CandidateIP, req.Subject.Port, result.Status, result.Code})
}
