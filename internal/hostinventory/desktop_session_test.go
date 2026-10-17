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
