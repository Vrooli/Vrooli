package health

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/vrooli/vrooli/internal/tuning"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/shell"
)

type Result struct {
	// Healthy is true only when every readiness check passed and no liveness
	// check failed.
	Healthy bool
	Message string
	// Serving is true when every readiness check passed, whether or not a
	// liveness check failed. A resource whose liveness check fails still
	// answers requests, so a consumer must be able to tell it from a resource
	// that is down.
	Serving bool
	// LivenessFailed names the first liveness check that failed. Empty means
	// none did.
	LivenessFailed string
}

type Config struct {
	Root       string
	Env        []string
	Runner     func(context.Context, *exec.Cmd) ([]byte, error)
	HTTPClient *http.Client
}

const (
	openRGBProtocolVersion       uint32 = 6
	openRGBSDKPacketAck          uint32 = 10
	openRGBProtocolRequest       uint32 = 40
	openRGBServerNamePacket      uint32 = 51
	openRGBControllerUpdate      uint32 = 1150
	openRGBSDKHeaderSize                = 16
	openRGBSDKMaxHandshakeBody          = 4096
	openRGBSDKMaxUpdateBody             = 4 << 20
	openRGBSDKMaxHandshakeFrames        = 32
)

// RunChecks runs both declared check kinds and combines them.
//
// A failing readiness check means the resource cannot answer: healthy and
// serving are both false. A failing liveness check means the resource answers
// but is not meeting its contract — running below its declared accelerator
// backend is the canonical case — so healthy is false while serving stays true.
// Collapsing the two would either hide a degradation or turn a working resource
// into an outage.
func RunChecks(ctx context.Context, checks []manifestpkg.ResourceHealthCheck, cfg Config) (Result, error) {
	if len(checks) == 0 {
		return Result{}, nil
	}
	var liveness []manifestpkg.ResourceHealthCheck
	for _, check := range checks {
		if isLivenessCheck(check) {
			liveness = append(liveness, check)
			continue
		}
		result, err := RunCheck(ctx, check, cfg)
		if err != nil {
			return result, err
		}
		if !result.Healthy {
			return result, nil
		}
	}

	combined := Result{Healthy: true, Serving: true, Message: "healthy"}
	for _, check := range liveness {
		result, err := RunCheck(ctx, check, cfg)
		if err != nil {
			// A liveness check that could not run is not a failing liveness
			// check. Report the resource as serving and say what could not be
			// determined rather than inventing a verdict.
			combined.Message = fmt.Sprintf("liveness check %q could not run: %v", checkName(check), err)
			return combined, nil
		}
		if result.Healthy {
			continue
		}
		combined.Healthy = false
		combined.LivenessFailed = checkName(check)
		combined.Message = fmt.Sprintf("degraded: liveness check %q failed", combined.LivenessFailed)
		if strings.TrimSpace(result.Message) != "" {
			combined.Message += ": " + result.Message
		}
		return combined, nil
	}
	return combined, nil
}

func isLivenessCheck(check manifestpkg.ResourceHealthCheck) bool {
	return strings.EqualFold(strings.TrimSpace(check.Kind), "liveness")
}

// checkName gives a liveness check a stable name for the operator message.
func checkName(check manifestpkg.ResourceHealthCheck) string {
	if len(check.Command) > 0 {
		return strings.Join(check.Command, " ")
	}
	if target := strings.TrimSpace(check.Target); target != "" {
		return target
	}
	return check.Type
}

func RunCheck(ctx context.Context, check manifestpkg.ResourceHealthCheck, cfg Config) (Result, error) {
	timeout := tuning.ServiceHealthTimeout()
	if check.TimeoutSeconds > 0 {
		timeout = time.Duration(check.TimeoutSeconds) * time.Second
	}
	checkCtx, cancel := context.WithTimeout(ctx, tuning.ResourceHealthCheckTimeout(timeout))
	defer cancel()

	switch check.Type {
	case "tcp":
		target := renderTarget(check.Target, cfg.Env)
		conn, err := (&net.Dialer{}).DialContext(checkCtx, "tcp", target)
		if err != nil {
			return Result{Message: fmt.Sprintf("tcp check failed for %s", target)}, nil
		}
		_ = conn.Close()
		return Result{Healthy: true, Message: "healthy"}, nil
	case "openrgb-sdk-v6":
		return runOpenRGBSDKV6Readiness(checkCtx, renderTarget(check.Target, cfg.Env))
	case "postgresql":
		return runPostgreSQLReadiness(checkCtx, renderTarget(check.Target, cfg.Env), cfg.Env), nil
	case "redis":
		return runRedisReadiness(checkCtx, renderTarget(check.Target, cfg.Env)), nil
	case "http":
		target := renderTarget(check.Target, cfg.Env)
		client := cfg.HTTPClient
		if client == nil {
			client = http.DefaultClient
		}
		req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, target, nil)
		if err != nil {
			return Result{}, err
		}
		for name, value := range check.Headers {
			req.Header.Set(name, renderTarget(value, cfg.Env))
		}
		resp, err := client.Do(req)
		if err != nil {
			return Result{Message: fmt.Sprintf("http check failed for %s", target)}, nil
		}
		defer resp.Body.Close()
		if len(check.ExpectedStatus) == 0 {
			if resp.StatusCode >= 200 && resp.StatusCode < 400 {
				return Result{Healthy: true, Message: "healthy"}, nil
			}
		} else if slices.Contains(check.ExpectedStatus, resp.StatusCode) {
			return Result{Healthy: true, Message: "healthy"}, nil
		}
		return Result{Message: fmt.Sprintf("http check returned %d", resp.StatusCode)}, nil
	case "command":
		if len(check.Command) == 0 {
			return Result{}, fmt.Errorf("command health check requires a command")
		}
		runner := cfg.Runner
		if runner == nil {
			runner = func(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
				return cmd.CombinedOutput()
			}
		}
		cmd := shell.Command(shell.Spec{
			Context: checkCtx,
			Name:    check.Command[0],
			Args:    check.Command[1:],
			Dir:     cfg.Root,
			Env:     cfg.Env,
			Stdin:   nil,
		})
		if _, err := runner(checkCtx, cmd); err != nil {
			return Result{Message: fmt.Sprintf("command check failed for %s", strings.Join(check.Command, " "))}, nil
		}
		return Result{Healthy: true, Message: "healthy"}, nil
	default:
		return Result{}, fmt.Errorf("unsupported health check type %q", check.Type)
	}
}

func runOpenRGBSDKV6Readiness(ctx context.Context, target string) (Result, error) {
	address, err := validateOpenRGBSDKAddress(target)
	if err != nil {
		return Result{}, err
	}
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", address)
	if err != nil {
		return Result{Message: fmt.Sprintf("OpenRGB SDK connection failed for %s", address)}, nil
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return Result{Message: "OpenRGB SDK readiness deadline could not be set"}, nil
		}
	}

	request := make([]byte, openRGBSDKHeaderSize+4)
	copy(request[:4], "ORGB")
	binary.LittleEndian.PutUint32(request[8:12], openRGBProtocolRequest)
	binary.LittleEndian.PutUint32(request[12:16], 4)
	binary.LittleEndian.PutUint32(request[16:], openRGBProtocolVersion)
	if err := writeOpenRGBSDKBytes(conn, request); err != nil {
		return Result{Message: "OpenRGB SDK protocol negotiation write failed"}, nil
	}

	var version uint32
	var serverNameSeen bool
	for frame := 0; frame < openRGBSDKMaxHandshakeFrames; frame++ {
		packetID, deviceID, body, err := readOpenRGBSDKHandshakeFrame(conn)
		if err != nil {
			return Result{Message: fmt.Sprintf("OpenRGB SDK protocol negotiation failed: %v", err)}, nil
		}
		switch packetID {
		case openRGBProtocolRequest:
			if deviceID != 0 || len(body) != 4 || version != 0 {
				return Result{Message: "OpenRGB SDK protocol reply is malformed"}, nil
			}
			version = binary.LittleEndian.Uint32(body)
			if version < openRGBProtocolVersion {
				return Result{Message: fmt.Sprintf("OpenRGB SDK protocol %d is below required version %d", version, openRGBProtocolVersion)}, nil
			}
		case openRGBServerNamePacket:
			if deviceID != 0 || serverNameSeen || !validOpenRGBServerName(body) {
				return Result{Message: "OpenRGB SDK server-name reply is malformed"}, nil
			}
			serverNameSeen = true
		case openRGBSDKPacketAck:
			// OpenRGB 1.1 sends the correlated ACK after its version and
			// name replies. A TCP listener that only returns both replies
			// has not completed the v6 negotiation.
			if deviceID != 0 || len(body) != 8 || binary.LittleEndian.Uint32(body[:4]) != openRGBProtocolRequest {
				return Result{Message: "OpenRGB SDK protocol negotiation ACK is malformed"}, nil
			}
			if version == 0 || !serverNameSeen {
				return Result{Message: "OpenRGB SDK protocol negotiation ACK arrived before both replies"}, nil
			}
			if status := binary.LittleEndian.Uint32(body[4:]); status != 0 {
				return Result{Message: fmt.Sprintf("OpenRGB SDK protocol negotiation ACK returned status %d", status)}, nil
			}
			return Result{Healthy: true, Message: "OpenRGB SDK protocol v6 ready"}, nil
		}
	}
	return Result{Message: "OpenRGB SDK handshake exceeded the frame bound"}, nil
}

func validOpenRGBServerName(body []byte) bool {
	if len(body) < 2 || body[len(body)-1] != 0 || bytes.IndexByte(body[:len(body)-1], 0) >= 0 || !utf8.Valid(body[:len(body)-1]) {
		return false
	}
	return strings.TrimSpace(string(body[:len(body)-1])) != ""
}

func validateOpenRGBSDKAddress(target string) (string, error) {
	host, port, err := net.SplitHostPort(strings.TrimSpace(target))
	if err != nil {
		return "", fmt.Errorf("OpenRGB SDK health target must be a loopback host:port: %w", err)
	}
	ip, err := netip.ParseAddr(strings.Trim(host, "[]"))
	if err != nil || !ip.IsLoopback() {
		return "", fmt.Errorf("OpenRGB SDK health target must use a literal loopback address")
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return "", fmt.Errorf("OpenRGB SDK health target has an invalid port")
	}
	return net.JoinHostPort(ip.String(), strconv.Itoa(portNumber)), nil
}

func writeOpenRGBSDKBytes(w io.Writer, data []byte) error {
	for len(data) > 0 {
		written, err := w.Write(data)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		data = data[written:]
	}
	return nil
}

func readOpenRGBSDKHandshakeFrame(r io.Reader) (packetID, deviceID uint32, body []byte, err error) {
	var header [openRGBSDKHeaderSize]byte
	if _, err = io.ReadFull(r, header[:]); err != nil {
		return 0, 0, nil, err
	}
	if string(header[:4]) != "ORGB" {
		return 0, 0, nil, fmt.Errorf("invalid packet magic")
	}
	deviceID = binary.LittleEndian.Uint32(header[4:8])
	packetID = binary.LittleEndian.Uint32(header[8:12])
	bodySize := binary.LittleEndian.Uint32(header[12:16])
	// OpenRGB broadcasts controller updates to v6 clients independently of
	// request replies. These can exceed the small handshake body limit.
	// Drain a bounded update without decoding or discovering any controller.
	if packetID == openRGBControllerUpdate {
		if bodySize > openRGBSDKMaxUpdateBody {
			return 0, 0, nil, fmt.Errorf("controller update exceeds %d-byte bound", openRGBSDKMaxUpdateBody)
		}
		_, err = io.CopyN(io.Discard, r, int64(bodySize))
		return packetID, deviceID, nil, err
	}
	if bodySize > openRGBSDKMaxHandshakeBody {
		return 0, 0, nil, fmt.Errorf("packet body exceeds %d-byte bound", openRGBSDKMaxHandshakeBody)
	}
	body = make([]byte, int(bodySize))
	if _, err = io.ReadFull(r, body); err != nil {
		return 0, 0, nil, err
	}
	return packetID, deviceID, body, nil
}

func renderTarget(target string, env []string) string {
	values := make(map[string]string, len(env))
	for _, entry := range env {
		if key, value, ok := strings.Cut(entry, "="); ok {
			values[key] = value
		}
	}
	return os.Expand(target, func(key string) string {
		if value, ok := values[key]; ok {
			return value
		}
		return "${" + key + "}"
	})
}
