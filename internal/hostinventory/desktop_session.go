package hostinventory

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DesktopSessionFacts are live host observations, never an execution grant or
// evidence of capture/input permission. No cached platform facts are used.
type DesktopSessionFacts struct {
	SessionID                       string
	UID                             uint32
	PeerPID                         int
	PeerUID                         uint32
	Type                            string
	sessionUIDVerified              bool
	Active, Locked, Remote, Matched bool
	Reason                          string
	ObservedAt                      time.Time
}

var desktopSessionID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

// InspectDesktopSessionOnDisplay obtains the X server peer from the selected
// local Unix socket, then applies the same live logind/process checks as the
// explicit peer-PID form. It never performs an X11 protocol exchange.
func (c Collector) InspectDesktopSessionOnDisplay(ctx context.Context, id string, display uint16) (DesktopSessionFacts, error) {
	c = c.withDefaults()
	if !desktopSessionID.MatchString(id) {
		return DesktopSessionFacts{SessionID: id, Reason: "unverified", ObservedAt: c.Clock.Now()}, fmt.Errorf("invalid desktop session identity")
	}
	facts := DesktopSessionFacts{SessionID: id, Reason: "unverified", ObservedAt: c.Clock.Now()}
	if c.GOOS != "linux" {
		facts.Reason = "unsupported_platform"
		return facts, nil
	}
	peer, err := desktopPeerForDisplay(ctx, display)
	if err != nil {
		facts.Reason = "peer_unavailable"
		return facts, nil
	}
	return inspectDesktopSessionPeer(ctx, id, peer, func(ctx context.Context, sessionID string, pid int) (DesktopSessionFacts, error) {
		return c.InspectDesktopSession(ctx, sessionID, pid)
	})
}

// inspectDesktopSessionPeer binds the kernel peer identity to the existing host
// verifier. The injected verifier exists to test the binding without probing a
// real host session.
func inspectDesktopSessionPeer(ctx context.Context, id string, peer desktopPeer, inspect func(context.Context, string, int) (DesktopSessionFacts, error)) (DesktopSessionFacts, error) {
	facts, err := inspect(ctx, id, peer.PID)
	if err != nil {
		return facts, err
	}
	if facts.SessionID != id || facts.PeerPID != peer.PID {
		facts.Matched = false
		facts.Reason = "session_facts_unverified"
		return facts, nil
	}
	if facts.Matched {
		// A matched host result must carry both verified UIDs, including UID 0.
		if !facts.sessionUIDVerified || facts.UID != peer.UID || facts.PeerUID != peer.UID {
			facts.Matched = false
			facts.Reason = "peer_user_mismatch"
			return facts, nil
		}
		return facts, nil
	}
	if (facts.PeerUID != 0 && facts.PeerUID != peer.UID) || (facts.sessionUIDVerified && facts.UID != peer.UID) {
		facts.Reason = "peer_user_mismatch"
		return facts, nil
	}
	if facts.PeerUID == 0 {
		// Negative observations may return before /proc UID inspection. The
		// kernel credential remains useful as observed metadata only.
		facts.PeerUID = peer.UID
	}
	return facts, nil
}

func (c Collector) InspectDesktopSession(ctx context.Context, id string, peerPID int) (DesktopSessionFacts, error) {
	c = c.withDefaults()
	facts := DesktopSessionFacts{SessionID: id, PeerPID: peerPID, Reason: "unverified", ObservedAt: c.Clock.Now()}
	if !desktopSessionID.MatchString(id) || peerPID <= 0 {
		return facts, fmt.Errorf("invalid desktop session identity")
	}
	if c.GOOS != "linux" {
		facts.Reason = "unsupported_platform"
		return facts, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	raw, err := c.Commands.Run(ctx, "loginctl", "show-session", id, "--no-pager", "-p", "Id", "-p", "User", "-p", "Type", "-p", "Active", "-p", "State", "-p", "LockedHint", "-p", "Remote")
	if err != nil {
		facts.Reason = "session_unavailable"
		return facts, nil
	}
	values := map[string]string{}
	for _, line := range strings.Split(string(raw), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if _, duplicate := values[key]; duplicate {
			facts.Reason = "invalid_session_facts"
			return facts, nil
		}
		values[key] = value
	}
	uid, err := strconv.ParseUint(values["User"], 10, 32)
	if err != nil || values["Id"] != id {
		facts.Reason = "session_mismatch"
		return facts, nil
	}
	facts.UID = uint32(uid)
	facts.sessionUIDVerified = true
	facts.Type = values["Type"]
	for _, key := range []string{"Active", "LockedHint", "Remote"} {
		if values[key] != "yes" && values[key] != "no" {
			facts.Reason = "incomplete_session_facts"
			return facts, nil
		}
	}
	facts.Active = values["Active"] == "yes"
	facts.Locked = values["LockedHint"] == "yes"
	facts.Remote = values["Remote"] == "yes"
	if facts.Type != "x11" {
		facts.Reason = "not_x11"
		return facts, nil
	}
	if facts.Remote {
		facts.Reason = "remote_session"
		return facts, nil
	}
	if !facts.Active || values["State"] != "active" {
		facts.Reason = "inactive"
		return facts, nil
	}
	if facts.Locked {
		facts.Reason = "locked"
		return facts, nil
	}
	prefix := "/proc/" + strconv.Itoa(peerPID) + "/"
	before, err := c.Files.ReadFile(prefix + "stat")
	if err != nil {
		facts.Reason = "peer_unavailable"
		return facts, nil
	}
	start, ok := processStartTime(string(before))
	if !ok {
		facts.Reason = "peer_unavailable"
		return facts, nil
	}
	status, err := c.Files.ReadFile(prefix + "status")
	if err != nil {
		facts.Reason = "peer_unavailable"
		return facts, nil
	}
	foundUID := false
	for _, line := range strings.Split(string(status), "\n") {
		if !strings.HasPrefix(line, "Uid:") {
			continue
		}
		fields := strings.Fields(strings.TrimPrefix(line, "Uid:"))
		if len(fields) != 4 {
			break
		}
		foundUID = true
		for _, value := range fields {
			parsed, err := strconv.ParseUint(value, 10, 32)
			if err != nil || uint32(parsed) != facts.UID {
				facts.Reason = "peer_user_mismatch"
				return facts, nil
			}
		}
		facts.PeerUID = facts.UID
	}
	if !foundUID {
		facts.Reason = "peer_user_unverified"
		return facts, nil
	}
	cgroup, err := c.Files.ReadFile(prefix + "cgroup")
	if err != nil {
		facts.Reason = "peer_session_unverified"
		return facts, nil
	}
	belongs := false
	for _, line := range strings.Split(string(cgroup), "\n") {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 {
			continue
		}
		for _, component := range strings.Split(parts[2], "/") {
			if component == "session-"+id+".scope" {
				belongs = true
			}
		}
	}
	after, err := c.Files.ReadFile(prefix + "stat")
	afterStart, afterOK := processStartTime(string(after))
	if err != nil || !afterOK || afterStart != start {
		facts.Reason = "peer_changed"
		return facts, nil
	}
	if !belongs {
		facts.Reason = "peer_session_mismatch"
		return facts, nil
	}
	facts.Matched = true
	facts.Reason = "session_peer_matched"
	facts.ObservedAt = c.Clock.Now()
	return facts, nil
}

func processStartTime(stat string) (string, bool) {
	// comm may contain spaces and parentheses; fields after its final ')' begin
	// at field 3. starttime is field 22 and protects against PID reuse during probe.
	end := strings.LastIndex(stat, ")")
	if end < 0 {
		return "", false
	}
	fields := strings.Fields(stat[end+1:])
	if len(fields) < 20 {
		return "", false
	}
	_, err := strconv.ParseUint(fields[19], 10, 64)
	return fields[19], err == nil
}
