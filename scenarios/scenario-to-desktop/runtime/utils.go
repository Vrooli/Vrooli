// Package bundleruntime provides the desktop bundle runtime supervisor.
package bundleruntime

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"scenario-to-desktop-runtime/manifest"
)

// topoSort performs topological sort on services based on their dependencies.
// Returns service IDs in dependency order (dependencies first).
func topoSort(services []manifest.Service) ([]string, error) {
	graph := make(map[string][]string)
	inDegree := make(map[string]int)

	for _, svc := range services {
		graph[svc.ID] = append(graph[svc.ID], svc.Dependencies...)
		if _, ok := inDegree[svc.ID]; !ok {
			inDegree[svc.ID] = 0
		}
		for _, dep := range svc.Dependencies {
			inDegree[svc.ID]++
			if _, ok := inDegree[dep]; !ok {
				inDegree[dep] = 0
			}
		}
	}

	queue := make([]string, 0)
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	order := make([]string, 0, len(inDegree))
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		order = append(order, id)
		for _, svc := range services {
			for _, dep := range svc.Dependencies {
				if dep == id {
					inDegree[svc.ID]--
					if inDegree[svc.ID] == 0 {
						queue = append(queue, svc.ID)
					}
				}
			}
		}
	}

	if len(order) != len(inDegree) {
		return nil, errors.New("cycle detected in dependencies")
	}
	return order, nil
}

// findService looks up a service by ID from a slice.
func findService(services []manifest.Service, id string) *manifest.Service {
	for i := range services {
		if services[i].ID == id {
			return &services[i]
		}
	}
	return nil
}

// sanitizeAppName normalizes an application name for filesystem use.
func sanitizeAppName(name string) string {
	out := strings.TrimSpace(name)
	if out == "" {
		return "desktop-app"
	}
	out = strings.ReplaceAll(out, " ", "-")
	out = strings.ToLower(out)
	return out
}

// parsePositiveInt parses a string as a positive integer.
func parsePositiveInt(s string) (int, error) {
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, err
	}
	if v <= 0 {
		return 0, errors.New("must be positive")
	}
	return v, nil
}

// tailFile returns the last N lines from a file.
func tailFile(path string, lines int) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(string(data), "\n")
	if lines >= len(parts) {
		return []byte(strings.Join(parts, "\n")), nil
	}
	return []byte(strings.Join(parts[len(parts)-lines:], "\n")), nil
}

// boolToString converts a boolean to "true" or "false".
func boolToString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

// exitCode extracts the exit code from an error, if available.
func exitCode(err error) *int {
	if err == nil {
		code := 0
		return &code
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		code := ee.ExitCode()
		return &code
	}
	return nil
}

// intersection returns elements present in both slices.
func intersection(a []string, b []string) []string {
	set := map[string]bool{}
	for _, v := range b {
		set[v] = true
	}
	var out []string
	for _, v := range a {
		if set[v] {
			out = append(out, v)
		}
	}
	return out
}

// copyStringMap creates a shallow copy of a string map.
func copyStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// envMapToList converts a map of environment variables to KEY=VALUE format.
func envMapToList(env map[string]string) []string {
	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	return out
}
