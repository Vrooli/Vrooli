package hostinventory

import (
	"context"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/shell/shelltest"
)

func TestDesktopSessionRequiresExactLivePeerAssociation(t *testing.T) {
	base := "Id=2\nUser=1000\nType=x11\nActive=yes\nState=active\nLockedHint=no\nRemote=no\n"
	stat := "42 (Xorg name) " + strings.Repeat("0 ", 19) + "12345"
	for _, tc := range []struct{ name, replace, with, cgroup, uid, want string }{
		{name: "matched", want: "session_peer_matched"},
		{name: "locked", replace: "LockedHint=no", with: "LockedHint=yes", want: "locked"},
		{name: "lock unknown", replace: "LockedHint=no", with: "LockedHint=", want: "incomplete_session_facts"},
		{name: "inactive", replace: "Active=yes", with: "Active=no", want: "inactive"},
		{name: "remote", replace: "Remote=no", with: "Remote=yes", want: "remote_session"},
		{name: "wrong type", replace: "Type=x11", with: "Type=wayland", want: "not_x11"},
		{name: "wrong session", replace: "Id=2", with: "Id=3", want: "session_mismatch"},
		{name: "other user", uid: "2000 2000 2000 2000", want: "peer_user_mismatch"},
		{name: "elevated peer", uid: "1000 0 0 1000", want: "peer_user_mismatch"},
		{name: "other login", cgroup: "0::/user.slice/session-3.scope", want: "peer_session_mismatch"},
		{name: "scope substring", cgroup: "0::/user.slice/session-22.scope", want: "peer_session_mismatch"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output := base
			if tc.replace != "" {
				output = strings.Replace(output, tc.replace, tc.with, 1)
			}
			uid := tc.uid
			if uid == "" {
				uid = "1000 1000 1000 1000"
			}
			group := tc.cgroup
			if group == "" {
				group = "0::/user.slice/user-1000.slice/session-2.scope"
			}
			c := Collector{GOOS: "linux", Commands: &shelltest.Fake{Outputs: map[string][]byte{"loginctl show-session 2 --no-pager -p Id -p User -p Type -p Active -p State -p LockedHint -p Remote": []byte(output)}}, Files: fakeFileReader{"/proc/42/stat": []byte(stat), "/proc/42/status": []byte("Uid:\t" + uid), "/proc/42/cgroup": []byte(group)}}
			facts, err := c.InspectDesktopSession(context.Background(), "2", 42)
			if err != nil {
				t.Fatal(err)
			}
			if facts.Reason != tc.want || facts.Matched != (tc.want == "session_peer_matched") {
				t.Fatalf("facts = %+v, want %s", facts, tc.want)
			}
		})
	}
}

func TestDesktopSessionRejectsInvalidIDsWithoutProbing(t *testing.T) {
	c := Collector{GOOS: "linux", Commands: &shelltest.Fake{}}
	for _, id := range []string{"", "../2", "--all", "a/b"} {
		// A leading hyphen must not become an option to loginctl.
		_, err := c.InspectDesktopSession(context.Background(), id, 42)
		if err == nil {
			t.Fatalf("accepted invalid ID %q", id)
		}
	}
}

func TestDisplayPeerIsBoundToExactHostVerifierIdentity(t *testing.T) {
	cases := []struct {
		name        string
		facts       DesktopSessionFacts
		wantMatched bool
		wantReason  string
		peerUID     uint32
		rootPeer    bool
	}{
		{name: "matched peer", facts: DesktopSessionFacts{SessionID: "2", UID: 1000, PeerPID: 42, PeerUID: 1000, Matched: true, Reason: "session_peer_matched", sessionUIDVerified: true}, wantMatched: true, wantReason: "session_peer_matched"},
		{name: "matched root peer uid zero", facts: DesktopSessionFacts{SessionID: "2", UID: 0, PeerPID: 42, PeerUID: 0, Matched: true, Reason: "session_peer_matched", sessionUIDVerified: true}, wantMatched: true, wantReason: "session_peer_matched", peerUID: 0, rootPeer: true},
		{name: "fresh locked negative gets kernel uid", facts: DesktopSessionFacts{SessionID: "2", UID: 1000, PeerPID: 42, Reason: "locked", sessionUIDVerified: true}, wantReason: "locked"},
		{name: "early negative reason preserved", facts: DesktopSessionFacts{SessionID: "2", PeerPID: 42, Reason: "session_unavailable"}, wantReason: "session_unavailable"},
		{name: "claimed match without host peer uid", facts: DesktopSessionFacts{SessionID: "2", UID: 1000, PeerPID: 42, Matched: true, sessionUIDVerified: true}, wantReason: "peer_user_mismatch"},
		{name: "conflicting observed peer uid", facts: DesktopSessionFacts{SessionID: "2", UID: 1000, PeerPID: 42, PeerUID: 2000, Reason: "locked", sessionUIDVerified: true}, wantReason: "peer_user_mismatch"},
		{name: "session uid differs", facts: DesktopSessionFacts{SessionID: "2", UID: 2000, PeerPID: 42, Reason: "locked", sessionUIDVerified: true}, wantReason: "peer_user_mismatch"},
		{name: "wrong pid", facts: DesktopSessionFacts{SessionID: "2", UID: 1000, PeerPID: 43, Reason: "locked"}, wantReason: "session_facts_unverified"},
		{name: "wrong session", facts: DesktopSessionFacts{SessionID: "3", UID: 1000, PeerPID: 42, Reason: "locked"}, wantReason: "session_facts_unverified"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calledPID := 0
			peerUID := tc.peerUID
			if !tc.rootPeer {
				peerUID = 1000
			}
			facts, err := inspectDesktopSessionPeer(context.Background(), "2", desktopPeer{PID: 42, UID: peerUID}, func(_ context.Context, session string, pid int) (DesktopSessionFacts, error) {
				if session != "2" {
					t.Fatalf("session passed to verifier = %q", session)
				}
				calledPID = pid
				return tc.facts, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if calledPID != 42 {
				t.Fatalf("verifier PID = %d, want exact kernel peer PID 42", calledPID)
			}
			if facts.Matched != tc.wantMatched || facts.Reason != tc.wantReason {
				t.Fatalf("facts = %+v", facts)
			}
			if (tc.name == "fresh locked negative gets kernel uid" || tc.name == "early negative reason preserved") && facts.PeerUID != peerUID {
				t.Fatalf("negative observation peer UID = %d, want %d", facts.PeerUID, peerUID)
			}
		})
	}
}

func TestDisplayInspectionRejectsInvalidSessionBeforePeerLookup(t *testing.T) {
	c := Collector{GOOS: "linux"}
	for _, id := range []string{"", "../2"} {
		_, err := c.InspectDesktopSessionOnDisplay(context.Background(), id, 0)
		if err == nil || !strings.Contains(err.Error(), "invalid desktop session identity") {
			t.Fatalf("session %q error = %v", id, err)
		}
	}
}

func TestDisplayInspectionIsUnsupportedOffLinux(t *testing.T) {
	c := Collector{GOOS: "darwin"}
	facts, err := c.InspectDesktopSessionOnDisplay(context.Background(), "2", 0)
	if err != nil || facts.Reason != "unsupported_platform" || facts.Matched {
		t.Fatalf("facts=%+v err=%v", facts, err)
	}
}
