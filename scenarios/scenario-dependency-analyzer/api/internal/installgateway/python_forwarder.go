package installgateway

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// pythonLockSurface recognizes the stable requirements.txt entry point used by
// Go APIs whose Python closure is embedded beside its owner code. It follows
// only a single lock include; ordinary pip manifests retain their own route.
func pythonLockSurface(surface string) (string, error) {
	data, err := os.ReadFile(filepath.Join(surface, "requirements.txt"))
	if os.IsNotExist(err) {
		return surface, nil
	}
	if err != nil {
		return "", err
	}
	var entries []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			entries = append(entries, line)
		}
	}
	if len(entries) != 1 {
		return surface, nil
	}
	fields := strings.Fields(entries[0])
	if len(fields) != 2 || (fields[0] != "-r" && fields[0] != "--requirement") || filepath.Base(fields[1]) != "requirements.lock" {
		return surface, nil
	}
	if filepath.IsAbs(fields[1]) {
		return "", fmt.Errorf("Python lock forwarder must be relative to its surface")
	}
	root, err := filepath.EvalSymlinks(surface)
	if err != nil {
		return "", err
	}
	target, err := filepath.EvalSymlinks(filepath.Join(surface, fields[1]))
	if err != nil {
		return "", fmt.Errorf("resolve forwarded Python lock: %w", err)
	}
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("forwarded Python lock escapes its declared surface")
	}
	owner := filepath.Dir(target)
	if _, err := pythonInputBytes(filepath.Join(owner, "requirements.in")); err != nil {
		return "", fmt.Errorf("forwarded Python lock requires its canonical regular requirements.in: %w", err)
	}
	return owner, nil
}

var pythonInputName = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9_.-]*)(\[[A-Za-z0-9_,.-]+\])?\s*([<>=!~@;].*)?$`)

func updatePythonInput(contents, name, version string) (string, error) {
	normalized := pythonName(name)
	lines := strings.Split(strings.TrimRight(contents, "\n"), "\n")
	found := false
	for i, line := range lines {
		declaration := strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		parts := pythonInputName.FindStringSubmatch(declaration)
		if len(parts) == 0 || pythonName(parts[1]) != normalized {
			continue
		}
		if found {
			return "", fmt.Errorf("package %q appears more than once in Python inputs", name)
		}
		found = true
		// An explicit install updates this package's version only. Keep any
		// requested extras and comment instead of silently discarding them.
		if strings.Contains(parts[3], ";") {
			return "", fmt.Errorf("conditional Python input %q requires an explicit reviewed update", name)
		}
		comment := ""
		if n := strings.Index(line, "#"); n >= 0 {
			comment = " " + line[n:]
		}
		lines[i] = name + parts[2] + "==" + version + comment
	}
	if !found {
		lines = append(lines, name+"=="+version)
	}
	return strings.Join(lines, "\n") + "\n", nil
}
