package x11

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

type Peer struct {
	PID int
	UID uint32
}

// LinuxSessionCheck delegates host-state inspection to the control plane. It
// never chooses another session, repairs host state, unlocks or elevates.
func LinuxSessionCheck(sessionID string) SessionCheck {
	return func(ctx context.Context, peer Peer) error {
		if runtime.GOOS != "linux" || peer.PID <= 0 || os.Getuid() < 0 || peer.UID != uint32(os.Getuid()) {
			return ErrUnavailable
		}
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, "vrooli", "host", "desktop-session", "--session-id", sessionID, "--peer-pid", strconv.Itoa(peer.PID), "--json")
		command.WaitDelay = 100 * time.Millisecond
		var output sessionProbeOutput
		command.Stdout = &output
		if command.Run() != nil {
			return ErrUnavailable
		}
		var facts cliv1.CliDesktopSessionFacts
		if protojson.Unmarshal(output.Bytes(), &facts) != nil {
			return ErrUnavailable
		}
		return verifySessionFacts(&facts, sessionID, peer, time.Now())
	}
}

type sessionProbeOutput struct{ bytes.Buffer }

func (b *sessionProbeOutput) Write(p []byte) (int, error) {
	if len(p) > 16*1024-b.Len() {
		return 0, ErrUnavailable
	}
	return b.Buffer.Write(p)
}
func verifySessionFacts(f *cliv1.CliDesktopSessionFacts, id string, peer Peer, now time.Time) error {
	observed, err := time.Parse(time.RFC3339Nano, f.ObservedAt)
	if err != nil || observed.After(now) || now.Sub(observed) > 2*time.Second || !f.Matched || f.SessionId != id || f.PeerPid != int64(peer.PID) || f.Uid != peer.UID || f.PeerUid != peer.UID || f.Type != "x11" || !f.Active || f.Locked || f.Remote {
		return ErrUnavailable
	}
	return nil
}

// NewForSession is the production constructor: callers cannot omit Linux
// login/session verification. The injected-check constructor is package-private
// and used by isolated native conformance fixtures.
func NewForSession(ctx context.Context, display uint16, cookieHex, sessionID string) (*Backend, error) {
	return newBackend(ctx, display, cookieHex, LinuxSessionCheck(sessionID))
}
