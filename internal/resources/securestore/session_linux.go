//go:build linux

package securestore

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/vrooli/vrooli/internal/tuning"
)

// The Secret Service is reached over a session bus, so libsecret depends on two
// environment variables that a login can get wrong. The most common way is a
// non-login user switch — `su someone` from a root shell keeps root's
// XDG_RUNTIME_DIR — which leaves a perfectly healthy keyring unreachable and
// reports as a bare "Could not connect: Permission denied".
//
// This file does two things with that condition. It repairs it when the repair
// is provably safe, and it explains it when it is not.

// runtimeRoot is where logind creates per-user runtime directories.
const runtimeRoot = "/run/user"

// sessionRepair is a correction applied to the credential subprocess only. It
// is never written back to this process's environment: a broken login usually
// breaks more than credentials, and silently patching our own environment would
// hide that from every other tool the operator runs.
type sessionRepair struct {
	runtimeDir string
	busAddress string
	// note explains, in operator terms, what was wrong and what was used
	// instead. It is safe to print: it carries no credential material.
	note string
}

// pathFact is the subset of a path's identity the repair decision depends on.
// Taking facts rather than paths is what makes the refusal cases testable: a
// test cannot create a root-owned socket, but it can state that one exists.
type pathFact struct {
	exists bool
	uid    int
	mode   os.FileMode
}

// planSessionRepair decides whether this process can safely reach its own
// session bus despite the environment naming another one.
//
// The guardrails are the whole point of the function, and each one is load
// bearing:
//
//   - The only destination ever considered is runtimeRoot/<this uid>. There is
//     no input, environment variable, or configuration that can aim this at
//     another user's session, so it cannot become a path to another user's
//     secrets.
//   - That directory must exist, be a directory, be owned by this uid, and
//     grant nothing to group or other. A directory someone else can write to is
//     not evidence of our session.
//   - Its bus must exist, be a socket, and be owned by this uid.
//   - A setting that already resolves correctly is never touched.
//
// When any check fails the answer is "no repair", which leaves the caller in
// today's behavior: report the condition and stay degraded. Refusing to guess
// is always available; guessing wrong is not recoverable.
func planSessionRepair(uid int, getenv func(string) string, stat func(string) pathFact) (sessionRepair, bool) {
	ownDir := filepath.Join(runtimeRoot, strconv.Itoa(uid))
	ownBus := filepath.Join(ownDir, "bus")

	dirFact := stat(ownDir)
	if !dirFact.exists || !dirFact.mode.IsDir() || dirFact.uid != uid || dirFact.mode.Perm()&tuning.PermGroupAndOtherMask != 0 {
		return sessionRepair{}, false
	}
	busFact := stat(ownBus)
	if !busFact.exists || busFact.mode&os.ModeSocket == 0 || busFact.uid != uid {
		return sessionRepair{}, false
	}

	var problems []string
	if problem := runtimeDirProblem(getenv("XDG_RUNTIME_DIR"), ownDir, uid, stat); problem != "" {
		problems = append(problems, problem)
	}
	if problem := busProblem(getenv("DBUS_SESSION_BUS_ADDRESS"), ownBus, uid, stat); problem != "" {
		problems = append(problems, problem)
	}
	if len(problems) == 0 {
		return sessionRepair{}, false
	}

	// Both variables are set together even when only one was wrong. They are
	// read as a pair — libsecret derives a bus from the runtime directory when
	// the address is unset — so repairing one and leaving the other is how a
	// half-corrected session ends up pointing at two different places.
	return sessionRepair{
		runtimeDir: ownDir,
		busAddress: "unix:path=" + ownBus,
		note: fmt.Sprintf("%s; used this user's own session at %s instead",
			strings.Join(problems, " and "), ownDir),
	}, true
}

// runtimeDirProblem reports why XDG_RUNTIME_DIR cannot be trusted, or "" when
// it is fine. A directory that is not the logind one but is genuinely ours is
// left alone: an operator who set a custom runtime directory meant it.
func runtimeDirProblem(current, ownDir string, uid int, stat func(string) pathFact) string {
	current = strings.TrimSpace(current)
	if current == "" {
		return "XDG_RUNTIME_DIR was unset"
	}
	if filepath.Clean(current) == ownDir {
		return ""
	}
	fact := stat(current)
	switch {
	case !fact.exists:
		// A private directory belonging to another user fails to stat with a
		// permission error, so "missing" and "invisible" are the same
		// observation from here. Claiming the stronger one would be a guess.
		return fmt.Sprintf("XDG_RUNTIME_DIR=%s is not reachable from this process", current)
	case fact.uid != uid:
		return fmt.Sprintf("XDG_RUNTIME_DIR=%s is owned by uid %d but this process runs as uid %d",
			current, fact.uid, uid)
	default:
		return ""
	}
}

// busProblem reports why DBUS_SESSION_BUS_ADDRESS cannot be trusted, or "" when
// it is fine. A non-unix transport is left alone; ownership is only a
// meaningful question for a socket on this filesystem.
func busProblem(current, ownBus string, uid int, stat func(string) pathFact) string {
	current = strings.TrimSpace(current)
	if current == "" {
		return "DBUS_SESSION_BUS_ADDRESS was unset"
	}
	path := unixSocketPath(current)
	if path == "" || filepath.Clean(path) == ownBus {
		return ""
	}
	fact := stat(path)
	switch {
	case !fact.exists:
		return fmt.Sprintf("DBUS_SESSION_BUS_ADDRESS pointed at %s, which is not reachable from this process", path)
	case fact.uid != uid:
		return fmt.Sprintf("DBUS_SESSION_BUS_ADDRESS pointed at %s owned by uid %d but this process runs as uid %d",
			path, fact.uid, uid)
	default:
		return ""
	}
}

func repairSession() (sessionRepair, bool) {
	return planSessionRepair(os.Getuid(), os.Getenv, statPathFact)
}

// sessionRuntimeDir reports this login session's runtime directory, which
// logind creates on a tmpfs and removes at logout. It is where an open
// credential data key may be kept between commands, so the checks are the same
// ones the bus repair applies: only this uid's own directory is ever
// considered, it must be a directory this uid owns, and it must grant nothing
// to group or other. Anything else means no session cache and an unlock that
// lasts one process — never a fallback onto durable disk.
func sessionRuntimeDir() (string, bool) {
	uid := os.Getuid()
	dir := filepath.Join(runtimeRoot, strconv.Itoa(uid))
	fact := statPathFact(dir)
	if !fact.exists || !fact.mode.IsDir() || fact.uid != uid || fact.mode.Perm()&tuning.PermGroupAndOtherMask != 0 {
		return "", false
	}
	return dir, true
}

// sessionEnviron is the environment for a Secret Service subprocess. A nil
// result means "inherit unchanged", which is what exec.Cmd already does for a
// nil Env, so the healthy path costs nothing.
func sessionEnviron() []string {
	repair, ok := repairSession()
	if !ok {
		return nil
	}
	environ := os.Environ()
	environ = withEnv(environ, "XDG_RUNTIME_DIR", repair.runtimeDir)
	environ = withEnv(environ, "DBUS_SESSION_BUS_ADDRESS", repair.busAddress)
	return environ
}

// sessionRepairNote reports the repair the credential path is relying on, so
// `vrooli credentials doctor` can say so. A repair that works but stays
// invisible would leave an operator with no thread to pull when the same broken
// login breaks something that cannot self-correct.
func sessionRepairNote() string {
	repair, ok := repairSession()
	if !ok {
		return ""
	}
	return repair.note
}

func withEnv(environ []string, name, value string) []string {
	prefix := name + "="
	for i, entry := range environ {
		if strings.HasPrefix(entry, prefix) {
			environ[i] = prefix + value
			return environ
		}
	}
	return append(environ, prefix+value)
}

// sessionDiagnosis names the host condition behind a Secret Service failure in
// operator terms. Every branch degrades to an empty string rather than failing
// the caller: a diagnosis is an explanation, never a precondition.
func sessionDiagnosis() string {
	uid := os.Getuid()
	expected := filepath.Join(runtimeRoot, strconv.Itoa(uid))

	// A repaired session already reached this user's own bus, so the
	// environment is not what failed. Reporting a uid mismatch here would send
	// an operator to fix a variable the credential path already stopped using.
	if _, repaired := repairSession(); repaired {
		return ""
	}

	if runtimeDir := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR")); runtimeDir != "" {
		if owner, ok := pathOwnerUID(runtimeDir); ok && owner != uid {
			return fmt.Sprintf(
				"XDG_RUNTIME_DIR=%s is owned by uid %d but this process runs as uid %d, and %s is not a usable session directory",
				runtimeDir, owner, uid, expected)
		}
	}

	busAddress := strings.TrimSpace(os.Getenv("DBUS_SESSION_BUS_ADDRESS"))
	if busPath := unixSocketPath(busAddress); busPath != "" {
		if owner, ok := pathOwnerUID(busPath); ok && owner != uid {
			return fmt.Sprintf(
				"DBUS_SESSION_BUS_ADDRESS points at %s owned by uid %d but this process runs as uid %d, and %s/bus is not a usable session bus",
				busPath, owner, uid, expected)
		}
	}

	if busAddress == "" {
		if _, err := os.Stat(expected + "/bus"); err != nil {
			return fmt.Sprintf(
				"no session bus: DBUS_SESSION_BUS_ADDRESS is unset and %s/bus does not exist, so this host has no Secret Service session",
				expected)
		}
	}

	return ""
}

// unixSocketPath extracts the socket path from a D-Bus address. D-Bus
// addresses are semicolon-separated alternatives of key=value pairs; only the
// unix transport's path is meaningful for ownership.
func unixSocketPath(address string) string {
	for _, alternative := range strings.Split(address, ";") {
		for _, field := range strings.Split(alternative, ",") {
			key, value, ok := strings.Cut(strings.TrimSpace(field), "=")
			if !ok {
				continue
			}
			if strings.TrimPrefix(strings.TrimSpace(key), "unix:") == "path" {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}

// statPathFact reads a path without following a final symlink. A symlinked
// runtime directory or bus is not evidence of our own session, and resolving it
// would let a link created elsewhere decide where we look for secrets.
func statPathFact(path string) pathFact {
	info, err := os.Lstat(path)
	if err != nil {
		return pathFact{}
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return pathFact{}
	}
	return pathFact{exists: true, uid: int(stat.Uid), mode: info.Mode()}
}

func pathOwnerUID(path string) (int, bool) {
	fact := statPathFact(path)
	return fact.uid, fact.exists
}
