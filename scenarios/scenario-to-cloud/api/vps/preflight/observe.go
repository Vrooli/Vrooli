package preflight

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
)

// observer is the read-only view of one target before enrollment: every
// probe is an observation program from reach.ObservationPrograms, so the
// bare host is inspected without a vrooli binary and without any shell.
type observer struct {
	reach  reach.Reach
	target identity.TargetRef
}

// observationTimeout bounds one probe.
const observationTimeout = 20 * time.Second

func (o observer) observe(ctx context.Context, program string, args ...string) (reach.Result, error) {
	if o.reach == nil {
		return reach.Result{}, &reach.Error{Kind: reach.KindUnavailable, Target: o.target.Key(), Detail: "no reach bound for preflight"}
	}
	cmd, err := reach.NewObservation(program, args...)
	if err != nil {
		return reach.Result{}, err
	}
	cmd.Timeout = observationTimeout
	return o.reach.Exec(ctx, o.target, cmd)
}

// ok reports a probe that ran and exited zero.
func ok(res reach.Result, err error) bool { return err == nil && res.ExitCode == 0 }

// ssEdgeFilter is the socket filter for the edge ports, as separate argv
// words (ss concatenates its trailing arguments into one filter expression).
func ssEdgeFilter(ports ...int) []string {
	args := []string{"("}
	for i, port := range ports {
		if i > 0 {
			args = append(args, "or")
		}
		args = append(args, "sport", "=", ":"+strconv.Itoa(port))
	}
	return append(args, ")")
}

// listeningSockets lists the listening TCP sockets on the given ports, with
// owning processes when the target user may see them.
func (o observer) listeningSockets(ctx context.Context, ports ...int) (reach.Result, error) {
	res, err := o.observe(ctx, "ss", append([]string{"-ltnpH"}, ssEdgeFilter(ports...)...)...)
	if ok(res, err) {
		return res, nil
	}
	return o.observe(ctx, "ss", append([]string{"-ltnH"}, ssEdgeFilter(ports...)...)...)
}

// dfRoot parses `df -Pk /` into (total, used, available) kilobytes and the
// used percentage.
func dfRoot(res reach.Result) (totalKB, usedKB, availKB int64, usedPct int, found bool) {
	lines := strings.Split(strings.TrimSpace(res.Stdout), "\n")
	if len(lines) < 2 {
		return 0, 0, 0, 0, false
	}
	fields := strings.Fields(lines[len(lines)-1])
	if len(fields) < 5 {
		return 0, 0, 0, 0, false
	}
	totalKB, _ = strconv.ParseInt(fields[1], 10, 64)
	usedKB, _ = strconv.ParseInt(fields[2], 10, 64)
	availKB, _ = strconv.ParseInt(fields[3], 10, 64)
	usedPct, _ = strconv.Atoi(strings.TrimSuffix(fields[4], "%"))
	return totalKB, usedKB, availKB, usedPct, true
}

// memTotalKB parses the MemTotal line of /proc/meminfo.
func memTotalKB(res reach.Result) (int64, bool) {
	for _, line := range strings.Split(res.Stdout, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "MemTotal:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			kb, err := strconv.ParseInt(fields[1], 10, 64)
			return kb, err == nil
		}
	}
	return 0, false
}

// toolDirs are the directories a bootstrap tool is looked for in; find
// reports what is present in one observation.
var toolDirs = []string{"/usr/bin", "/usr/local/bin", "/bin", "/usr/sbin", "/sbin", "/snap/bin"}

// findTools reports which of the named programs exist in toolDirs, keyed by
// name with the first path found.
func (o observer) findTools(ctx context.Context, names ...string) map[string]string {
	args := append([]string{}, toolDirs...)
	args = append(args, "-maxdepth", "1", "(")
	for i, name := range names {
		if i > 0 {
			args = append(args, "-o")
		}
		args = append(args, "-name", name)
	}
	args = append(args, ")")
	res, _ := o.observe(ctx, "find", args...)
	found := map[string]string{}
	for _, line := range strings.Split(res.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		base := line[strings.LastIndex(line, "/")+1:]
		if _, seen := found[base]; !seen {
			found[base] = line
		}
	}
	return found
}

// exists reports whether a path exists on the target.
func (o observer) exists(ctx context.Context, path string) bool {
	res, err := o.observe(ctx, "stat", "--", path)
	return ok(res, err)
}

// hostRepair runs one privilege-broker action through the target owner
// (`vrooli cloud-target host repair`) without an operation: preflight and
// management repairs are operator-driven and unfenced, but they still go
// through the owner's argv policy rather than a shell string.
func hostRepair(ctx context.Context, r reach.Reach, target identity.TargetRef, action string, subject any) (reach.Result, error) {
	raw, err := json.Marshal(subject)
	if err != nil {
		return reach.Result{}, err
	}
	encoded := "b64:" + base64.RawURLEncoding.EncodeToString(raw)
	return r.Exec(ctx, target, reach.Command{
		Verb: "cloud-target host repair", Args: []string{"--action", action, "--subject", encoded, "--json"},
		RequiredScope: "vrooli:write", Effectful: true, Timeout: 2 * time.Minute,
	})
}

// verbFailure extracts the typed target answer from a host repair reply.
func verbFailure(res reach.Result, err error) string {
	if err != nil {
		return err.Error()
	}
	if res.ExitCode == 0 {
		return ""
	}
	var reply struct {
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal([]byte(strings.TrimSpace(res.Stdout)), &reply) == nil && reply.Error != nil {
		return reply.Error.Code + ": " + reply.Error.Message
	}
	return fmt.Sprintf("host repair exited %d: %s", res.ExitCode, strings.TrimSpace(res.Stderr))
}
