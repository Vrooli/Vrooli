package storagemigrate

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"scenario-to-desktop-api/storagepaths"
	"strings"

	sharedpath "scenario-to-desktop-api/shared/path"
)

// Options configures one storage relocation run.
type Options struct {
	RepoRoot string
	HomeDir  string
	Locator  *storagepaths.Locator
}

// Entry describes one source to destination move.
type Entry struct {
	Kind        string `json:"kind"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

// Result summarizes one storage relocation run.
type Result struct {
	Moved   []Entry `json:"moved"`
	Skipped []Entry `json:"skipped"`
}

type moveSpec struct {
	kind string
	src  string
	dst  string
}

// Run relocates scenario-to-desktop runtime data from legacy repo/home paths into
// classed runtime storage roots. Runtime code must read only the new locations.
func Run(opts Options) (*Result, error) {
	repoRoot, homeDir, locator, err := resolveRunInputs(opts)
	if err != nil {
		return nil, err
	}
	if _, err := locator.EnsureAll(); err != nil {
		return nil, fmt.Errorf("prepare storage roots: %w", err)
	}

	specs, err := buildMoveSpecs(repoRoot, homeDir, locator)
	if err != nil {
		return nil, err
	}

	return applyMoveSpecs(specs)
}

func resolveRunInputs(opts Options) (string, string, *storagepaths.Locator, error) {
	repoRoot := strings.TrimSpace(opts.RepoRoot)
	if repoRoot == "" {
		repoRoot = sharedpath.DetectVrooliRoot()
	}
	if repoRoot == "" {
		return "", "", nil, fmt.Errorf("detect repo root: not found")
	}

	homeDir := strings.TrimSpace(opts.HomeDir)
	if homeDir == "" {
		resolved, err := os.UserHomeDir()
		if err != nil {
			return "", "", nil, fmt.Errorf("resolve home dir: %w", err)
		}
		homeDir = resolved
	}

	locator := opts.Locator
	if locator == nil {
		created, err := storagepaths.NewLocator()
		if err != nil {
			return "", "", nil, fmt.Errorf("create storage locator: %w", err)
		}
		locator = created
	}
	return repoRoot, homeDir, locator, nil
}

func buildMoveSpecs(repoRoot, homeDir string, locator *storagepaths.Locator) ([]moveSpec, error) {
	resolvers := []struct {
		label string
		fn    func() (string, error)
	}{
		{"data root", locator.DataRoot},
		{"deploy targets path", locator.DeployTargetsPath},
		{"telemetry dir", locator.TelemetryDir},
		{"records path", locator.RecordsPath},
		{"smoke test path", locator.SmokeTestsPath},
		{"pipeline dir", locator.PipelineStateDir},
		{"index dir", locator.PipelineIndexDir},
		{"investigations dir", locator.InvestigationsDir},
		{"scenario state dir", locator.ScenarioStateDir},
		{"live desktop dir", locator.LiveDesktopDir},
	}
	resolved := make(map[string]string, len(resolvers))
	for _, r := range resolvers {
		value, err := r.fn()
		if err != nil {
			return nil, fmt.Errorf("resolve %s: %w", r.label, err)
		}
		resolved[r.label] = value
	}

	scenarioData := filepath.Join(repoRoot, "scenarios", "scenario-to-desktop", "data")
	return []moveSpec{
		{kind: "file", src: filepath.Join(repoRoot, ".vrooli", "deploy-targets.json"), dst: resolved["deploy targets path"]},
		{kind: "dir", src: filepath.Join(repoRoot, ".vrooli", "deployment", "telemetry"), dst: resolved["telemetry dir"]},
		{kind: "file", src: filepath.Join(scenarioData, "desktop_records_v2.json"), dst: resolved["records path"]},
		{kind: "file", src: filepath.Join(scenarioData, "smoke_tests_v2.json"), dst: resolved["smoke test path"]},
		{kind: "file", src: filepath.Join(scenarioData, "smoke_tests.json"), dst: filepath.Join(resolved["data root"], "smoke_tests.json")},
		{kind: "dir", src: filepath.Join(scenarioData, "pipelines"), dst: resolved["pipeline dir"]},
		{kind: "dir", src: filepath.Join(scenarioData, "indexes"), dst: resolved["index dir"]},
		{kind: "dir", src: filepath.Join(scenarioData, "investigations"), dst: resolved["investigations dir"]},
		{kind: "dir", src: filepath.Join(scenarioData, "livedesktop"), dst: resolved["live desktop dir"]},
		{kind: "dir", src: filepath.Join(homeDir, ".vrooli", "scenario-to-desktop", "state"), dst: resolved["scenario state dir"]},
	}, nil
}

func applyMoveSpecs(specs []moveSpec) (*Result, error) {
	result := &Result{}
	for _, spec := range specs {
		exists, err := pathExists(spec.src)
		if err != nil {
			return nil, fmt.Errorf("check source %s: %w", spec.src, err)
		}
		entry := Entry{Kind: spec.kind, Source: spec.src, Destination: spec.dst}
		if !exists {
			result.Skipped = append(result.Skipped, entry)
			continue
		}
		if err := applyMove(spec); err != nil {
			return nil, err
		}
		result.Moved = append(result.Moved, entry)
	}
	return result, nil
}

func applyMove(spec moveSpec) error {
	switch spec.kind {
	case "file":
		return moveFile(spec.src, spec.dst)
	case "dir":
		return moveDir(spec.src, spec.dst)
	default:
		return fmt.Errorf("unsupported move kind %q", spec.kind)
	}
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func moveFile(src, dst string) error {
	if err := ensureFileDestination(dst); err != nil {
		return fmt.Errorf("prepare destination %s: %w", dst, err)
	}
	if err := copyFile(src, dst); err != nil {
		return fmt.Errorf("copy file %s -> %s: %w", src, dst, err)
	}
	if err := os.Remove(src); err != nil {
		return fmt.Errorf("remove source file %s: %w", src, err)
	}
	return nil
}

func moveDir(src, dst string) error {
	if err := ensureDirDestination(dst); err != nil {
		return fmt.Errorf("prepare destination %s: %w", dst, err)
	}
	if err := copyDir(src, dst); err != nil {
		return fmt.Errorf("copy dir %s -> %s: %w", src, dst, err)
	}
	if err := os.RemoveAll(src); err != nil {
		return fmt.Errorf("remove source dir %s: %w", src, err)
	}
	return nil
}

func ensureFileDestination(dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	exists, err := pathExists(dst)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("destination already exists: %s", dst)
	}
	return nil
}

func ensureDirDestination(dst string) error {
	exists, err := pathExists(dst)
	if err != nil {
		return err
	}
	if !exists {
		return os.MkdirAll(filepath.Dir(dst), 0o755)
	}
	entries, err := os.ReadDir(dst)
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("destination already contains data: %s", dst)
	}
	return nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := dst
		if rel != "." {
			target = filepath.Join(dst, rel)
		}
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinks are not supported: %s", path)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	tmpPath := dst + ".tmp-relocate"
	if err := os.RemoveAll(tmpPath); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	copied, err := os.Stat(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if copied.Size() != info.Size() {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("size mismatch after copy: src=%d dst=%d", info.Size(), copied.Size())
	}
	return os.Rename(tmpPath, dst)
}
