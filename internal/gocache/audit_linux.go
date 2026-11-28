//go:build linux

package gocache

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// auditSupported reports whether this platform can audit live go commands.
const auditSupported = true

const (
	// clockTicks is USER_HZ, the fixed unit of /proc/<pid>/stat start times.
	clockTicks = 100
	// startSlack covers the one-second precision of the boot time that
	// anchors process start times.
	startSlack = 2 * time.Second
	// commLength is the kernel's task-name limit, including the terminator.
	commLength = 16
)

type auditor struct {
	procRoot   string
	store      os.FileInfo
	program    string
	uid        int
	now        func() time.Time
	readExe    func(path string) (executable, error)
	executable map[[2]uint64]executable
	bootTime   time.Time
}

func newAuditor(store Store, program string, now func() time.Time) (*auditor, error) {
	info, err := os.Stat(store.Root)
	if err != nil {
		return nil, err
	}
	return &auditor{
		procRoot:   "/proc",
		store:      info,
		program:    program,
		uid:        os.Getuid(),
		now:        now,
		readExe:    readExecutable,
		executable: map[[2]uint64]executable{},
	}, nil
}

// run classifies every live go command. An error means the host could not be
// observed completely, which callers treat as a refusal.
func (a *auditor) run() (AuditReport, error) {
	report := AuditReport{ObservedAt: a.now().UTC()}
	if err := a.requireHostPIDNamespace(); err != nil {
		return report, err
	}
	if a.bootTime.IsZero() {
		boot, err := readBootTime(a.procRoot)
		if err != nil {
			return report, err
		}
		a.bootTime = boot
	}
	entries, err := os.ReadDir(a.procRoot)
	if err != nil {
		return report, fmt.Errorf("list processes: %w", err)
	}
	helperComm := ProgramName[:commLength-1]
	var goCommands []int
	sessionParents := map[int]bool{}
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		comm, err := os.ReadFile(a.proc(pid, "comm"))
		if err != nil {
			continue // exited, or a kernel thread being torn down
		}
		switch strings.TrimSuffix(string(comm), "\n") {
		case "go":
			goCommands = append(goCommands, pid)
		case helperComm:
			if exe, err := a.identify(pid); err == nil && exe.mainPath == ProgramMainPath {
				if stat, err := a.readStat(pid); err == nil {
					sessionParents[stat.ppid] = true
				}
			}
		}
	}
	for _, pid := range goCommands {
		a.classify(pid, sessionParents, &report)
	}
	return report, nil
}

func (a *auditor) classify(pid int, sessionParents map[int]bool, report *AuditReport) {
	finding := func(reason string) {
		report.Unmediated = append(report.Unmediated, ProcessFinding{PID: pid, Reason: reason, Command: a.commandLine(pid)})
	}
	uid, err := a.realUID(pid)
	if err != nil {
		if !vanished(err) {
			finding("process owner unreadable: " + err.Error())
		}
		return
	}
	if uid != a.uid {
		finding(fmt.Sprintf("go command of uid %d cannot be inspected", uid))
		return
	}
	exe, err := a.identify(pid)
	if err != nil {
		if !vanished(err) {
			finding("executable unreadable: " + err.Error())
		}
		return
	}
	if exe.mainPath != goCommandMainPath {
		return
	}
	report.GoCommands++
	if sessionParents[pid] {
		// A go command that has started a helper routes every cache access
		// through it from then on.
		report.Mediated++
		return
	}
	environ, err := a.environ(pid)
	if err != nil {
		if !vanished(err) {
			finding("environment unreadable: " + err.Error())
		}
		return
	}
	stat, err := a.readStat(pid)
	if err != nil {
		if !vanished(err) {
			finding("start time unreadable: " + err.Error())
		}
		return
	}
	lookup := func(key string) string { return environ[key] }
	settings := resolveGoSettings(lookup, processDirs(lookup))
	// Values from the env file are what the process read at start only if
	// the file has not changed since.
	fileTrusted := false
	if settings.EnvFile != "" {
		if info, err := os.Stat(settings.EnvFile); err == nil {
			fileTrusted = a.startTime(stat).After(info.ModTime().Add(startSlack))
		}
	}
	sameStore := true
	switch {
	case settings.CacheFromEnv || fileTrusted:
		sameStore = a.isStore(settings.Cache)
	}
	if !sameStore {
		report.OtherStores++
		return
	}
	switch {
	case !supportsCacheProg(exe.goVersion):
		finding(exe.goVersion + " predates GOCACHEPROG")
	case settings.Program == "":
		finding("does not use the build-cache protocol (GOCACHEPROG unset)")
	case settings.Program != a.program:
		finding(fmt.Sprintf("uses GOCACHEPROG %q", settings.Program))
	case !settings.ProgramFromEnv && !fileTrusted:
		finding("started before its go env file last changed")
	default:
		// It will start the helper on first cache use and has not touched the
		// cache yet.
		report.Mediated++
	}
}

func (a *auditor) isStore(cache string) bool {
	if cache == "" || cache == "off" || !filepath.IsAbs(cache) {
		return false
	}
	info, err := os.Stat(cache)
	return err == nil && os.SameFile(info, a.store)
}

func (a *auditor) proc(pid int, name string) string {
	return filepath.Join(a.procRoot, strconv.Itoa(pid), name)
}

func (a *auditor) identify(pid int) (executable, error) {
	path := a.proc(pid, "exe")
	info, err := os.Stat(path)
	if err != nil {
		return executable{}, err
	}
	device, inode, _ := fileIdentity(info)
	key := [2]uint64{device, inode}
	if exe, ok := a.executable[key]; ok {
		return exe, nil
	}
	exe, err := a.readExe(path)
	if err != nil {
		// A binary without Go build information is simply not a go command.
		if _, statErr := os.Stat(path); statErr != nil {
			return executable{}, statErr
		}
		exe = executable{}
	}
	a.executable[key] = exe
	return exe, nil
}

func (a *auditor) environ(pid int) (map[string]string, error) {
	data, err := os.ReadFile(a.proc(pid, "environ"))
	if err != nil {
		return nil, err
	}
	environ := map[string]string{}
	for _, entry := range bytes.Split(data, []byte{0}) {
		if key, value, ok := bytes.Cut(entry, []byte{'='}); ok && len(key) > 0 {
			environ[string(key)] = string(value)
		}
	}
	return environ, nil
}

func (a *auditor) realUID(pid int) (int, error) {
	file, err := os.Open(a.proc(pid, "status"))
	if err != nil {
		return 0, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if value, ok := strings.CutPrefix(scanner.Text(), "Uid:"); ok {
			fields := strings.Fields(value)
			if len(fields) == 0 {
				break
			}
			return strconv.Atoi(fields[0])
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return 0, errors.New("status has no Uid line")
}

type processStat struct {
	ppid      int
	startTick uint64
}

func (a *auditor) readStat(pid int) (processStat, error) {
	data, err := os.ReadFile(a.proc(pid, "stat"))
	if err != nil {
		return processStat{}, err
	}
	// The command name is parenthesised and may contain spaces or ')'.
	end := bytes.LastIndexByte(data, ')')
	if end < 0 {
		return processStat{}, errors.New("malformed stat")
	}
	fields := strings.Fields(string(data[end+1:]))
	// fields[0] is field 3 (state); ppid is field 4, starttime field 22.
	if len(fields) < 20 {
		return processStat{}, errors.New("malformed stat")
	}
	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return processStat{}, err
	}
	start, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil {
		return processStat{}, err
	}
	return processStat{ppid: ppid, startTick: start}, nil
}

func (a *auditor) startTime(stat processStat) time.Time {
	return a.bootTime.Add(time.Duration(stat.startTick) * time.Second / clockTicks)
}

func (a *auditor) commandLine(pid int) string {
	data, err := os.ReadFile(a.proc(pid, "cmdline"))
	if err != nil {
		return ""
	}
	args := strings.Split(strings.TrimRight(string(data), "\x00"), "\x00")
	if len(args) > 8 {
		args = append(args[:8], "...")
	}
	line := strings.Join(args, " ")
	if len(line) > 200 {
		line = line[:200] + "..."
	}
	return line
}

// requireHostPIDNamespace refuses an audit from inside a nested PID
// namespace, where /proc would hide the host's go commands and report a
// falsely clear host.
func (a *auditor) requireHostPIDNamespace() error {
	file, err := os.Open(filepath.Join(a.procRoot, "self", "status"))
	if err != nil {
		return fmt.Errorf("read own process status: %w", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if value, ok := strings.CutPrefix(scanner.Text(), "NSpid:"); ok {
			if len(strings.Fields(value)) != 1 {
				return errors.New("the go-command audit must run in the host PID namespace; this process is in a nested one and cannot see every build")
			}
			return nil
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return errors.New("cannot determine this process's PID namespace (no NSpid in status)")
}

func readBootTime(procRoot string) (time.Time, error) {
	file, err := os.Open(filepath.Join(procRoot, "stat"))
	if err != nil {
		return time.Time{}, fmt.Errorf("read boot time: %w", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if value, ok := strings.CutPrefix(scanner.Text(), "btime "); ok {
			seconds, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
			if err != nil {
				return time.Time{}, fmt.Errorf("parse boot time: %w", err)
			}
			return time.Unix(seconds, 0), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return time.Time{}, err
	}
	return time.Time{}, errors.New("boot time not found")
}

func vanished(err error) bool {
	return errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ESRCH)
}
