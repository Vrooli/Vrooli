package privsep

// This file is the narrow local boundary between the ordinary node agent and
// the privileged provisioning service. The runner can request only the typed
// ProvisionCommand already authorized by Bridge; it cannot send an argv or a
// shell string. The helper owns the StepRunner and streams typed
// ProvisionEvents back over the same connection.

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	provisionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/provision"
	"google.golang.org/protobuf/encoding/protojson"
)

const ipcProtocolVersion = 1

type ipcRequest struct {
	Version   int             `json:"version"`
	Operation string          `json:"operation"`
	Command   json.RawMessage `json:"command,omitempty"`
}

type ipcResponse struct {
	Version int             `json:"version"`
	Event   json.RawMessage `json:"event,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// Serve starts the privileged helper on a local Unix socket and blocks until
// ctx is cancelled. The listener is removed only when it is the socket this
// process created; an existing non-socket path is never overwritten.
func Serve(ctx context.Context, socket, vrooliBin, workDir string, allowedClientUID int) error {
	if filepath.IsAbs(socket) == false {
		return fmt.Errorf("provision IPC socket must be absolute: %q", socket)
	}
	if err := prepareSocketPath(socket); err != nil {
		return err
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		return fmt.Errorf("listen on provisioning IPC socket: %w", err)
	}
	defer func() {
		_ = listener.Close()
		_ = os.Remove(socket)
	}()
	// The peer credential check is the authorization boundary. The mode permits
	// the separately-owned runner to connect; callers are still rejected by
	// peer UID on Linux and macOS before a command is decoded.
	if err := os.Chmod(socket, 0o666); err != nil { // #nosec G302 -- Linux peer credentials, not mode bits, authorize the runner; mode permits the distinct service user to connect.
		return fmt.Errorf("secure provisioning IPC socket: %w", err)
	}

	var provisionMu sync.Mutex
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()
	for {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("accept provisioning IPC client: %w", acceptErr)
		}
		go serveConn(ctx, conn, vrooliBin, workDir, allowedClientUID, &provisionMu)
	}
}

func prepareSocketPath(socket string) error {
	if err := os.MkdirAll(filepath.Dir(socket), 0o750); err != nil {
		return fmt.Errorf("create provisioning IPC directory: %w", err)
	}
	info, err := os.Lstat(socket)
	if err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return fmt.Errorf("refusing to replace non-socket provisioning IPC path %q", socket)
		}
		if err := os.Remove(socket); err != nil {
			return fmt.Errorf("remove stale provisioning IPC socket: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect provisioning IPC path: %w", err)
	}
	return nil
}

func serveConn(ctx context.Context, conn net.Conn, vrooliBin, workDir string, allowedClientUID int, provisionMu *sync.Mutex) {
	defer conn.Close()
	if allowedClientUID >= 0 {
		uid, err := peerUID(conn)
		if err != nil || uid != allowedClientUID {
			_ = writeIPC(conn, ipcResponse{Version: ipcProtocolVersion, Error: "provisioning IPC caller is not the configured runner principal"})
			return
		}
	}
	dec := json.NewDecoder(bufio.NewReader(conn))
	var req ipcRequest
	if err := dec.Decode(&req); err != nil {
		_ = writeIPC(conn, ipcResponse{Version: ipcProtocolVersion, Error: "invalid provisioning IPC request"})
		return
	}
	if req.Version != ipcProtocolVersion || req.Operation != "provision" || len(req.Command) == 0 {
		_ = writeIPC(conn, ipcResponse{Version: ipcProtocolVersion, Error: "unsupported provisioning IPC request"})
		return
	}
	var command channelv1.ProvisionCommand
	if err := protojson.Unmarshal(req.Command, &command); err != nil {
		_ = writeIPC(conn, ipcResponse{Version: ipcProtocolVersion, Error: "invalid typed provision command"})
		return
	}

	// Git checkout/setup mutates one working tree. Serialize calls even if a
	// buggy or malicious runner opens multiple IPC connections.
	provisionMu.Lock()
	defer provisionMu.Unlock()
	reporter := &ipcReporter{conn: conn}
	helper := NewHelper(vrooliBin, workDir, reporter)
	if err := helper.Provision(ctx, &command); err != nil && ctx.Err() == nil {
		_ = writeIPC(conn, ipcResponse{Version: ipcProtocolVersion, Error: "provisioning event transport failed"})
	}
}

type ipcReporter struct{ conn net.Conn }

func (r *ipcReporter) Report(_ context.Context, event *provisionv1.ProvisionEvent) error {
	b, err := protojson.Marshal(event)
	if err != nil {
		return err
	}
	return writeIPC(r.conn, ipcResponse{Version: ipcProtocolVersion, Event: b})
}

// Run calls the privileged helper and forwards each typed event to report. A
// missing helper, a peer mismatch, a malformed response, or an EOF before the
// terminal event is an error; callers must turn it into a typed failed
// provisioning operation rather than silently dropping the operation.
func Run(ctx context.Context, socket string, expectedHelperUID int, command *channelv1.ProvisionCommand, report func(*provisionv1.ProvisionEvent) error) error {
	if filepath.IsAbs(socket) == false {
		return fmt.Errorf("provision IPC socket must be absolute: %q", socket)
	}
	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", socket)
	if err != nil {
		return fmt.Errorf("connect to provisioning helper: %w", err)
	}
	defer conn.Close()
	if expectedHelperUID >= 0 {
		uid, peerErr := peerUID(conn)
		if peerErr != nil || uid != expectedHelperUID {
			return fmt.Errorf("provisioning helper peer uid %d is not expected uid %d", uid, expectedHelperUID)
		}
	}
	commandJSON, err := protojson.Marshal(command)
	if err != nil {
		return fmt.Errorf("marshal provision command: %w", err)
	}
	if err := writeIPC(conn, ipcRequest{Version: ipcProtocolVersion, Operation: "provision", Command: commandJSON}); err != nil {
		return err
	}
	dec := json.NewDecoder(bufio.NewReader(conn))
	for {
		var response ipcResponse
		if err := dec.Decode(&response); err != nil {
			return fmt.Errorf("read provisioning IPC response: %w", err)
		}
		if response.Error != "" {
			return errors.New(response.Error)
		}
		if response.Version != ipcProtocolVersion || len(response.Event) == 0 {
			return errors.New("invalid provisioning IPC response")
		}
		var event provisionv1.ProvisionEvent
		if err := protojson.Unmarshal(response.Event, &event); err != nil {
			return fmt.Errorf("unmarshal provision event: %w", err)
		}
		if err := report(&event); err != nil {
			return err
		}
		if event.GetKind() == provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_EXIT {
			return nil
		}
	}
}

var ipcWriteMu sync.Mutex

func writeIPC(conn net.Conn, value any) error {
	// A helper connection has one reporter, but keeping writes serialized makes
	// the framing invariant explicit if future helpers multiplex events.
	ipcWriteMu.Lock()
	defer ipcWriteMu.Unlock()
	return json.NewEncoder(conn).Encode(value)
}
