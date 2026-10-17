//go:build linux

package capacity

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	cgroupFieldCount  = 3
	containerIDLength = 64
)

// procCgroupSource resolves a container ID from /proc/<pid>/cgroup on linux.
type procCgroupSource struct {
	procRoot string // overridable for tests; defaults to /proc
}

func newProcCgroupSource() CgroupSource {
	return procCgroupSource{procRoot: "/proc"}
}

func (s procCgroupSource) ContainerID(pid int) (string, bool) {
	root := s.procRoot
	if root == "" {
		root = "/proc"
	}
	data, err := os.ReadFile(filepath.Join(root, strconv.Itoa(pid), "cgroup"))
	if err != nil {
		return "", false
	}
	return parseCgroupContainerID(string(data))
}

// parseCgroupContainerID extracts a docker/containerd container ID from the
// contents of a /proc/<pid>/cgroup file. Handles cgroup v1
// ("12:devices:/docker/<id>") and v2 ("0::/system.slice/docker-<id>.scope")
// layouts, plus kubepods nesting.
func parseCgroupContainerID(contents string) (string, bool) {
	for _, line := range strings.Split(contents, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// The path is the last colon-separated field.
		parts := strings.SplitN(line, ":", cgroupFieldCount)
		path := line
		if len(parts) == cgroupFieldCount {
			path = parts[2]
		}
		if id, ok := containerIDFromPath(path); ok {
			return id, true
		}
	}
	return "", false
}

func containerIDFromPath(path string) (string, bool) {
	// Split on both "/" and "-" so "docker-<id>.scope" and "/docker/<id>" both
	// surface the id token.
	fields := strings.FieldsFunc(path, func(r rune) bool { return r == '/' || r == '-' })
	for _, f := range fields {
		f = strings.TrimSuffix(f, ".scope")
		if isContainerID(f) {
			return f, true
		}
	}
	return "", false
}

// isContainerID reports whether a token looks like a 64-char hex container ID.
func isContainerID(s string) bool {
	if len(s) != containerIDLength {
		return false
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		default:
			return false
		}
	}
	return true
}
