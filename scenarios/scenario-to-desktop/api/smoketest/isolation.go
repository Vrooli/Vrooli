package smoketest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

func deriveScreenContentSource(observations []deliveryramp.IsolationObservation) string {
	if len(observations) == 0 {
		return "unknown"
	}
	for _, observation := range observations {
		for _, resolved := range resolvedIsolationPaths(observation) {
			if isLiveSessionPath(resolved) {
				return "unknown"
			}
		}
	}
	for _, observation := range observations {
		if observation.Source == "isolated_instance" {
			return "isolated_instance"
		}
	}
	return "bundled_private"
}

func validateIsolationObservations(observations []deliveryramp.IsolationObservation) error {
	for _, observation := range observations {
		for _, resolved := range resolvedIsolationPaths(observation) {
			if isLiveSessionPath(resolved) {
				return fmt.Errorf("live_state_resolved: %s", resolved)
			}
		}
	}
	return nil
}

func resolvedIsolationPaths(observation deliveryramp.IsolationObservation) []string {
	paths := []string{observation.StateRoot, observation.SocketPath, observation.DatabasePath}
	return append(paths, observation.ResolvedPaths...)
}

func isUnderOperatorHome(path string) bool {
	home, err := os.UserHomeDir()
	return err == nil && isUnderPath(path, home)
}

func isLiveSessionPath(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	return strings.Contains(clean, "/.local/state/vrooli/") || strings.Contains(clean, "/web-console-sessions/") || strings.Contains(clean, "/tmux/")
}

func isUnderPath(path, root string) bool {
	if strings.TrimSpace(path) == "" || strings.TrimSpace(root) == "" {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
