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
	Active, Locked, Remote, Matched bool
	Reason                          string
	ObservedAt                      time.Time
}

var desktopSessionID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

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
