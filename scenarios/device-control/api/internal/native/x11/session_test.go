package x11

import (
	"github.com/stretchr/testify/require"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	"testing"
	"time"
)

func TestSessionGuardRequiresFreshExactControlPlaneFacts(t *testing.T) { // NAT-02
	now := time.Now().UTC()
	peer := Peer{PID: 42, UID: 1000}
	for _, change := range []string{"none", "stale", "future", "uid", "peer", "session", "locked", "unmatched"} {
		t.Run(change, func(t *testing.T) {
			f := &cliv1.CliDesktopSessionFacts{SessionId: "2", Uid: 1000, PeerUid: 1000, PeerPid: 42, Type: "x11", Active: true, Matched: true, ObservedAt: now.Format(time.RFC3339Nano)}
			switch change {
			case "stale":
				f.ObservedAt = now.Add(-3 * time.Second).Format(time.RFC3339Nano)
			case "future":
				f.ObservedAt = now.Add(time.Second).Format(time.RFC3339Nano)
			case "uid":
				f.Uid = 2000
			case "peer":
				f.PeerPid = 43
			case "session":
				f.SessionId = "3"
			case "locked":
				f.Locked = true
			case "unmatched":
				f.Matched = false
			}
			err := verifySessionFacts(f, "2", peer, now)
			if change == "none" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
