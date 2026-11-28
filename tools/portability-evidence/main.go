// Command portability-evidence writes the durable native-platform evidence
// record consumed by infrastructure-manager's portability ledger.
//
// The command is intentionally small and dependency-free so the scheduled
// Bridge workflow can use the same writer as a local operator run. A failed
// record is still written: the ledger treats the latest applicable failure as
// a decay signal instead of allowing an old qualified claim to remain green.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type record struct {
	SchemaVersion int      `json:"schema_version"`
	Kind          string   `json:"kind"`
	HostOS        string   `json:"host_os"`
	Architecture  string   `json:"architecture"`
	Commit        string   `json:"commit,omitempty"`
	GeneratedAt   string   `json:"generated_at"`
	Passed        bool     `json:"passed"`
	Source        string   `json:"source"`
	RunID         string   `json:"run_id"`
	Host          string   `json:"host"`
	Surface       string   `json:"surface"`
	ArtifactURI   string   `json:"artifact_uri"`
	Capabilities  []string `json:"capabilities,omitempty"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "portability-evidence:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("portability-evidence", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	output := fs.String("output", "", "path to the native-platform JSON record")
	kind := fs.String("kind", "hardware-persistence", "evidence kind")
	hostOS := fs.String("os", "", "host OS (darwin, linux, or windows)")
	arch := fs.String("arch", "", "host architecture")
	commit := fs.String("commit", "", "validated source revision")
	runID := fs.String("run-id", "", "durable validation run identifier")
	host := fs.String("host", "", "named host that was validated")
	surface := fs.String("surface", "", "validation surface")
	artifactURI := fs.String("artifact-uri", "", "durable transcript or artifact URI")
	source := fs.String("source", "bridge-scheduled", "evidence producer")
	claims := fs.String("capabilities", "", "comma-separated capability names; empty means all capabilities")
	passed := fs.Bool("passed", false, "whether the validation passed")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !containsPassedFlag(args) {
		return errors.New("--passed is required (true or false)")
	}
	values := map[string]string{
		"--output":       *output,
		"--kind":         *kind,
		"--os":           *hostOS,
		"--arch":         *arch,
		"--run-id":       *runID,
		"--host":         *host,
		"--surface":      *surface,
		"--artifact-uri": *artifactURI,
		"--source":       *source,
	}
	for name, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	if !isSupportedOS(*hostOS) {
		return fmt.Errorf("--os %q is not darwin, linux, or windows", *hostOS)
	}
	if strings.TrimSpace(*arch) == "" {
		return errors.New("--arch must not be empty")
	}

	record := record{
		SchemaVersion: 1,
		Kind:          strings.TrimSpace(*kind),
		HostOS:        strings.ToLower(strings.TrimSpace(*hostOS)),
		Architecture:  strings.TrimSpace(*arch),
		Commit:        strings.TrimSpace(*commit),
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		Passed:        *passed,
		Source:        strings.TrimSpace(*source),
		RunID:         strings.TrimSpace(*runID),
		Host:          strings.TrimSpace(*host),
		Surface:       strings.TrimSpace(*surface),
		ArtifactURI:   strings.TrimSpace(*artifactURI),
		Capabilities:  splitClaims(*claims),
	}
	return writeRecord(*output, record)
}

func containsPassedFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--passed" || strings.HasPrefix(arg, "--passed=") {
			return true
		}
	}
	return false
}

func isSupportedOS(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "darwin", "macos", "linux", "windows":
		return true
	default:
		return false
	}
}

func splitClaims(raw string) []string {
	seen := make(map[string]struct{})
	claims := make([]string, 0)
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		claims = append(claims, item)
	}
	sort.Strings(claims)
	return claims
}

func writeRecord(path string, value record) error {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "." || filepath.Ext(path) != ".json" {
		return fmt.Errorf("--output must name a .json file: %q", path)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode evidence: %w", err)
	}
	data = append(data, '\n')
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:mnd // evidence directory mode is intentional
		return fmt.Errorf("create evidence directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".native-evidence-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary evidence file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o644); err != nil { //nolint:mnd // evidence artifact permissions are intentional
		_ = tmp.Close()
		return fmt.Errorf("set evidence permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write evidence: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close evidence: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("publish evidence: %w", err)
	}
	return nil
}
