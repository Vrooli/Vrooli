package cloudtarget

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/shell"
)

// ObservationRequest is the target-owner input for one bounded read. The
// caller supplies a semantic kind and arguments; it cannot select an
// executable or construct a shell command.
type ObservationRequest struct {
	Kind string
	Args []string
}

// ObservationResult is the stable envelope returned by host observe. Output
// is bounded by the invoked read and remains target-produced evidence.
type ObservationResult struct {
	Kind     string `json:"kind"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr,omitempty"`
	ExitCode int    `json:"exit_code"`
}

// MaxObservationOutputBytes bounds the target-owner evidence returned by one
// observation. Individual operations may impose a smaller semantic bound
// (for example file_head); this is the final envelope bound for every kind.
const MaxObservationOutputBytes = 1 << 20

// Observe executes one of the target owner's closed read-only observation
// operations. Every operation has a fixed executable and argument policy;
// paths are resolved before the read to prevent traversal and symlink escape.
func Observe(ctx context.Context, req ObservationRequest, runner shell.Runner) (ObservationResult, error) {
	if runner == nil {
		runner = shell.OSRunner{}
	}
	kind := strings.TrimSpace(req.Kind)
	if kind == "" {
		return ObservationResult{}, refuse(CodeInvalidArgument, "observation kind is required")
	}
	args := append([]string(nil), req.Args...)
	for _, arg := range args {
		if strings.TrimSpace(arg) == "" || len(arg) > 4096 || strings.ContainsAny(arg, ";&|$`<>\n\r\x00'\"\\") {
			return ObservationResult{}, refuse(CodeInvalidArgument, "observation argument is unsafe")
		}
	}
	program, err := observationProgram(kind, args)
	if err != nil {
		return ObservationResult{}, err
	}
	out, runErr := runner.Run(ctx, program.name, program.args...)
	tooLarge := len(out) > MaxObservationOutputBytes
	if tooLarge {
		out = out[:MaxObservationOutputBytes]
	}
	result := ObservationResult{Kind: kind, Stdout: string(out), ExitCode: 0}
	if runErr != nil {
		result.ExitCode = 1
		result.Stderr = string(out)
	}
	if tooLarge {
		result.ExitCode = 1
		result.Stderr = "observation output exceeded the target-owner limit"
	}
	return result, nil
}

type observationCommand struct {
	name string
	args []string
}

func observationProgram(kind string, args []string) (observationCommand, error) {
	switch kind {
	case "os":
		if !sameArgs(args, "-s") && !sameArgs(args, "-m") {
			return observationCommand{}, refuse(CodeInvalidArgument, "os observation accepts only -s or -m")
		}
		return observationCommand{name: "uname", args: args}, nil
	case "disk":
		if len(args) < 2 || args[0] != "-Pk" {
			return observationCommand{}, refuse(CodeInvalidArgument, "disk observation requires -Pk and one path")
		}
		if err := validateReadPath(args[len(args)-1], true); err != nil {
			return observationCommand{}, err
		}
		return observationCommand{name: "df", args: args}, nil
	case "directory_usage":
		if err := validatePathArgs(args); err != nil {
			return observationCommand{}, err
		}
		return observationCommand{name: "du", args: args}, nil
	case "directory_entries":
		if err := validateFindArgs(args); err != nil {
			return observationCommand{}, err
		}
		return observationCommand{name: "find", args: args}, nil
	case "directory_listing":
		if err := validatePathArgs(args); err != nil {
			return observationCommand{}, err
		}
		return observationCommand{name: "ls", args: args}, nil
	case "file":
		path, err := lastPath(args)
		if err != nil {
			return observationCommand{}, err
		}
		if err := validateReadPath(path, false); err != nil {
			return observationCommand{}, err
		}
		return observationCommand{name: "cat", args: []string{"--", path}}, nil
	case "file_head":
		if len(args) < 4 || args[0] != "-c" || args[2] != "--" {
			return observationCommand{}, refuse(CodeInvalidArgument, "file_head observation requires -c <bytes> -- <path>")
		}
		limit, err := strconv.Atoi(args[1])
		if err != nil || limit < 0 || limit > 1<<20 {
			return observationCommand{}, refuse(CodeInvalidArgument, "file_head byte limit is outside the bounded range")
		}
		if err := validateReadPath(args[3], false); err != nil {
			return observationCommand{}, err
		}
		return observationCommand{name: "head", args: args}, nil
	case "file_stat":
		if len(args) >= 2 && args[0] == "--" {
			for _, path := range args[1:] {
				if err := validateReadPath(path, false); err != nil {
					return observationCommand{}, err
				}
			}
			return observationCommand{name: "stat", args: args}, nil
		}
		if len(args) < 4 || args[0] != "-c" || args[2] != "--" {
			return observationCommand{}, refuse(CodeInvalidArgument, "file_stat observation requires -c <format> -- <path>")
		}
		if args[1] != "%n" && args[1] != "%s" {
			return observationCommand{}, refuse(CodeInvalidArgument, "file_stat format is not allowed")
		}
		for _, path := range args[3:] {
			if err := validateReadPath(path, false); err != nil {
				return observationCommand{}, err
			}
		}
		return observationCommand{name: "stat", args: args}, nil
	case "grep":
		path, err := lastPath(args)
		if err != nil {
			return observationCommand{}, err
		}
		if err := validateReadPath(path, false); err != nil {
			return observationCommand{}, err
		}
		return observationCommand{name: "grep", args: args}, nil
	case "journal":
		if err := validateJournalArgs(args); err != nil {
			return observationCommand{}, err
		}
		return observationCommand{name: "journalctl", args: args}, nil
	case "processes":
		if !sameArgs(args, "aux", "--no-headers") {
			return observationCommand{}, refuse(CodeInvalidArgument, "process observation shape is not allowed")
		}
		return observationCommand{name: "ps", args: args}, nil
	case "sockets":
		if len(args) == 1 && args[0] == "-tlnp" {
			return observationCommand{name: "ss", args: args}, nil
		}
		if len(args) == 5 && args[0] == "-ltnH" && args[1] == "sport" && args[2] == "=" && strings.HasPrefix(args[3], ":") && safeName(strings.TrimPrefix(args[3], ":")) {
			return observationCommand{name: "ss", args: args}, nil
		}
		if validSocketSet(args) {
			return observationCommand{name: "ss", args: args}, nil
		}
		return observationCommand{}, refuse(CodeInvalidArgument, "socket observation shape is not allowed")
	case "process_match":
		if len(args) == 2 && args[0] == "-x" && safeName(args[1]) {
			return observationCommand{name: "pgrep", args: args}, nil
		}
		if len(args) == 3 && args[0] == "-a" && args[1] == "-f" && safeName(args[2]) {
			return observationCommand{name: "pgrep", args: args}, nil
		}
		if len(args) != 2 || args[0] != "-x" || !safeName(args[1]) {
			return observationCommand{}, refuse(CodeInvalidArgument, "process match observation shape is not allowed")
		}
		return observationCommand{name: "pgrep", args: args}, nil
	case "privilege":
		if !sameArgs(args, "-n", "-l") {
			return observationCommand{}, refuse(CodeInvalidArgument, "privilege observation accepts only sudo -n -l")
		}
		return observationCommand{name: "sudo", args: args}, nil
	default:
		return observationCommand{}, refuse(CodeInvalidArgument, "unknown target observation kind %q", kind)
	}
}

func sameArgs(got []string, want ...string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func safeName(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if !(r == '-' || r == '_' || r == '.' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func lastPath(args []string) (string, error) {
	for i := len(args) - 1; i >= 0; i-- {
		if args[i] != "--" && !strings.HasPrefix(args[i], "-") {
			return args[i], nil
		}
	}
	return "", refuse(CodeInvalidArgument, "observation path is required")
}

func validatePathArgs(args []string) error {
	seen := false
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		seen = true
		if err := validateReadPath(arg, false); err != nil {
			return err
		}
	}
	if !seen {
		return refuse(CodeInvalidArgument, "observation path is required")
	}
	return nil
}

func validateFindArgs(args []string) error {
	seenPath := false
	for _, arg := range args {
		if strings.HasPrefix(arg, "/") {
			seenPath = true
			if err := validateReadPath(arg, false); err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(arg, "-") {
			switch arg {
			case "-exec", "-execdir", "-delete", "-ok", "-okdir", "-fls", "-fprint", "-fprint0", "-fprintf":
				return refuse(CodeInvalidArgument, "find mutation/output action is not allowed")
			}
			continue
		}
		if !safeName(arg) && arg != "(" && arg != ")" {
			return refuse(CodeInvalidArgument, "find argument is not bounded")
		}
	}
	if !seenPath {
		return refuse(CodeInvalidArgument, "observation path is required")
	}
	return nil
}

func validSocketSet(args []string) bool {
	if len(args) < 8 || args[0] != "-ltnpH" || args[1] != "(" || args[len(args)-1] != ")" {
		return false
	}
	for i := 2; i < len(args)-1; i += 4 {
		if i+2 >= len(args)-1 || args[i] != "sport" || args[i+1] != "=" || !strings.HasPrefix(args[i+2], ":") || !safeName(strings.TrimPrefix(args[i+2], ":")) {
			return false
		}
		if i+3 == len(args)-1 {
			return true
		}
		if args[i+3] != "or" {
			return false
		}
	}
	return false
}

func validateJournalArgs(args []string) error {
	for i, arg := range args {
		if arg == "-n" && i+1 < len(args) {
			n, err := strconv.Atoi(args[i+1])
			if err != nil || n < 0 || n > 2000 {
				return refuse(CodeInvalidArgument, "journal line limit is outside the bounded range")
			}
		}
		if strings.HasPrefix(arg, "/") {
			if err := validateReadPath(arg, false); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateReadPath(value string, allowRoot bool) error {
	if strings.TrimSpace(value) == "" || !filepath.IsAbs(value) || strings.Contains(value, "..") {
		return refuse(CodeInvalidArgument, "observation path must be an absolute, non-traversing path")
	}
	clean := filepath.Clean(value)
	if clean == "/" && allowRoot {
		return nil
	}
	allowed := []string{"/etc", "/home", "/opt", "/proc", "/root", "/run", "/sys", "/tmp", "/usr", "/var"}
	if !underAllowedRoot(clean, allowed) {
		return refuse(CodeInvalidArgument, "observation path is outside the target-owned read roots")
	}
	if resolved, err := filepath.EvalSymlinks(clean); err == nil && !underAllowedRoot(resolved, allowed) {
		return refuse(CodeInvalidArgument, "observation path resolves outside the target-owned read roots")
	}
	return nil
}

func underAllowedRoot(value string, roots []string) bool {
	for _, root := range roots {
		if value == root || strings.HasPrefix(value, root+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
