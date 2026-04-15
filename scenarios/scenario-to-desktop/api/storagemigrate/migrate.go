package storagemigrate

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	sharedpath "scenario-to-desktop-api/shared/path"
	"scenario-to-desktop-api/storagepaths"
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
	repoRoot := strings.TrimSpace(opts.RepoRoot)
	if repoRoot == "" {
		repoRoot = sharedpath.DetectVrooliRoot()
	}
	if repoRoot == "" {
		return nil, fmt.Errorf("detect repo root: not found")
	}

	homeDir := strings.TrimSpace(opts.HomeDir)
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolve home dir: %w", err)
		}
	}

	locator := opts.Locator
	if locator == nil {
		var err error
		locator, err = storagepaths.NewLocator()
		if err != nil {
			return nil, fmt.Errorf("create storage locator: %w", err)
		}
	}
	if _, err := locator.EnsureAll(); err != nil {
		return nil, fmt.Errorf("prepare storage roots: %w", err)
	}

	dataRoot, err := locator.DataRoot()
	if err != nil {
		return nil, fmt.Errorf("resolve data root: %w", err)
	}
	deployTargetsPath, err := locator.DeployTargetsPath()
	if err != nil {
		return nil, fmt.Errorf("resolve deploy targets path: %w", err)
	}
	telemetryDir, err := locator.TelemetryDir()
	if err != nil {
		return nil, fmt.Errorf("resolve telemetry dir: %w", err)
	}
	recordsPath, err := locator.RecordsPath()
	if err != nil {
		return nil, fmt.Errorf("resolve records path: %w", err)
	}
	smokeTestsPath, err := locator.SmokeTestsPath()
	if err != nil {
		return nil, fmt.Errorf("resolve smoke test path: %w", err)
	}
	pipelineDir, err := locator.PipelineStateDir()
	if err != nil {
		return nil, fmt.Errorf("resolve pipeline dir: %w", err)
	}
	indexDir, err := locator.PipelineIndexDir()
	if err != nil {
		return nil, fmt.Errorf("resolve index dir: %w", err)
	}
	investigationsDir, err := locator.InvestigationsDir()
	if err != nil {
		return nil, fmt.Errorf("resolve investigations dir: %w", err)
	}
	stateDir, err := locator.ScenarioStateDir()
	if err != nil {
		return nil, fmt.Errorf("resolve scenario state dir: %w", err)
	}
	liveDesktopDir, err := locator.LiveDesktopDir()
	if err != nil {
		return nil, fmt.Errorf("resolve live desktop dir: %w", err)
	}

	specs := []moveSpec{
		{kind: "file", src: filepath.Join(repoRoot, ".vrooli", "deploy-targets.json"), dst: deployTargetsPath},
		{kind: "dir", src: filepath.Join(repoRoot, ".vrooli", "deployment", "telemetry"), dst: telemetryDir},
		{kind: "file", src: filepath.Join(repoRoot, "scenarios", "scenario-to-desktop", "data", "desktop_records_v2.json"), dst: recordsPath},
		{kind: "file", src: filepath.Join(repoRoot, "scenarios", "scenario-to-desktop", "data", "smoke_tests_v2.json"), dst: smokeTestsPath},
		{kind: "file", src: filepath.Join(repoRoot, "scenarios", "scenario-to-desktop", "data", "smoke_tests.json"), dst: filepath.Join(dataRoot, "smoke_tests.json")},
		{kind: "dir", src: filepath.Join(repoRoot, "scenarios", "scenario-to-desktop", "data", "pipelines"), dst: pipelineDir},
		{kind: "dir", src: filepath.Join(repoRoot, "scenarios", "scenario-to-desktop", "data", "indexes"), dst: indexDir},
		{kind: "dir", src: filepath.Join(repoRoot, "scenarios", "scenario-to-desktop", "data", "investigations"), dst: investigationsDir},
		{kind: "dir", src: filepath.Join(repoRoot, "scenarios", "scenario-to-desktop", "data", "livedesktop"), dst: liveDesktopDir},
		{kind: "dir", src: filepath.Join(homeDir, ".vrooli", "scenario-to-desktop", "state"), dst: stateDir},
	}

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

		switch spec.kind {
		case "file":
			if err := moveFile(spec.src, spec.dst); err != nil {
				return nil, err
			}
		case "dir":
			if err := moveDir(spec.src, spec.dst); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unsupported move kind %q", spec.kind)
		}
		result.Moved = append(result.Moved, entry)
	}

	return result, nil
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
