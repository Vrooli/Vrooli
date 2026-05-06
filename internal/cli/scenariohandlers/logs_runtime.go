package scenariohandlers

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	. "github.com/vrooli/vrooli/internal/cli/scenariocli" //nolint:revive // scenariohandlers is a thin glue layer over scenariocli; dot-import keeps wiring readable.
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioexec"
)

type scenarioStepLogInfo struct {
	Phase string
	Step  string
	Path  string
}

func LogsHandler[C any](deps HandlerDeps[C]) func(C, []string) error {
	return func(ctx C, args []string) error {
		name, opts, err := ParseLogsArgs(args)
		if err != nil {
			if err == ErrScenarioLogsUsage {
				showErr := ShowLogsUsage(deps.Stdout(ctx))
				if showErr != nil && showErr != ErrScenarioLogsUsage {
					return showErr
				}
				return nil
			}
			return err
		}
		if name == "" {
			showErr := ShowLogsUsage(deps.Stdout(ctx))
			if showErr != nil && showErr != ErrScenarioLogsUsage {
				return showErr
			}
			return nil
		}
		home, err := deps.HomeDir(ctx)
		if err != nil {
			return err
		}
		if opts.Clean {
			return cleanScenarioLogs(deps.Root(ctx), home, name, deps.Stdout(ctx))
		}
		if opts.Runtime {
			return showScenarioRuntimeLogs(home, name, opts, deps.Stdout(ctx))
		}
		if opts.StepName != "" {
			return showScenarioStepLog(home, name, opts, deps.Stdout(ctx))
		}
		return showScenarioLifecycleLog(deps.Root(ctx), home, name, opts, deps.Stdout(ctx), deps.Stderr(ctx))
	}
}

func cleanScenarioLogs(root, home, name string, stdout io.Writer) error {
	logsDir := process.ScenarioLogsDir(home, name)
	if _, err := os.Stat(logsDir); err != nil {
		if os.IsNotExist(err) {
			_, _ = fmt.Fprintf(stdout, "No log directory found for scenario '%s'. Nothing to clean.\n", name)
			return nil
		}
		return err
	}
	expected := expectedScenarioBackgroundSteps(root, name)
	files, err := filepath.Glob(filepath.Join(logsDir, "vrooli.*."+name+".*.log"))
	if err != nil {
		return err
	}
	sort.Strings(files)
	orphaned := make([]string, 0)
	for _, path := range files {
		info, ok := parseScenarioStepLogInfo(name, path)
		if !ok {
			orphaned = append(orphaned, path)
			continue
		}
		if _, exists := expected[info.Step+":"+info.Phase]; exists {
			continue
		}
		orphaned = append(orphaned, path)
	}
	if len(orphaned) == 0 {
		_, _ = fmt.Fprintln(stdout, "No orphaned logs found.")
		return nil
	}
	_, _ = fmt.Fprintf(stdout, "Removing %d orphaned log(s)\n", len(orphaned))
	for _, path := range orphaned {
		_ = os.Remove(path + ".bak")
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		_, _ = fmt.Fprintf(stdout, "  %s\n", filepath.Base(path))
	}
	return nil
}

func showScenarioRuntimeLogs(home, name string, opts LogOptions, stdout io.Writer) error {
	logsDir := process.ScenarioLogsDir(home, name)
	paths, err := filepath.Glob(filepath.Join(logsDir, "*.log"))
	if err != nil {
		return err
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return fmt.Errorf("no runtime log files found for scenario %q", name)
	}
	if opts.Follow {
		if !opts.ForceFollow && !scenarioexec.WriterSupportsStreaming(stdout) {
			writeScenarioLogSnapshotNotice(stdout)
			return writeScenarioLogTail(stdout, paths, logTailLines(opts, 50))
		}
		_, _ = fmt.Fprintf(stdout, "Following runtime logs for scenario '%s'\n", name)
		return followScenarioLogFiles(stdout, paths, logTailLines(opts, 10))
	}
	_, _ = fmt.Fprintf(stdout, "Recent runtime logs for scenario '%s'\n\n", name)
	return writeScenarioLogTail(stdout, paths, logTailLines(opts, 50))
}

func showScenarioStepLog(home, name string, opts LogOptions, stdout io.Writer) error {
	logsDir := process.ScenarioLogsDir(home, name)
	suffix := ".log"
	if opts.Previous {
		suffix = ".log.bak"
	}
	pattern := filepath.Join(logsDir, "vrooli.*."+name+"."+opts.StepName+suffix)
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		if opts.Previous {
			return fmt.Errorf("no previous log found for step %q", opts.StepName)
		}
		return fmt.Errorf("no log found for step %q", opts.StepName)
	}
	path := paths[0]
	if opts.Follow {
		if !opts.ForceFollow && !scenarioexec.WriterSupportsStreaming(stdout) {
			writeScenarioLogSnapshotNotice(stdout)
			return writeScenarioLogTail(stdout, []string{path}, logTailLines(opts, 100))
		}
		_, _ = fmt.Fprintf(stdout, "Following log for step '%s' in scenario '%s'\n", opts.StepName, name)
		return followScenarioLogFiles(stdout, []string{path}, logTailLines(opts, 10))
	}
	_, _ = fmt.Fprintf(stdout, "Recent log for step '%s' in scenario '%s'\n\n", opts.StepName, name)
	return writeScenarioLogTail(stdout, []string{path}, logTailLines(opts, 100))
}

func showScenarioLifecycleLog(root, home, name string, opts LogOptions, stdout, stderr io.Writer) error {
	_ = stderr
	path := process.ScenarioLifecycleLogPath(home, name)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no lifecycle log found for scenario %q", name)
		}
		return err
	}
	if opts.Follow {
		if !opts.ForceFollow && !scenarioexec.WriterSupportsStreaming(stdout) {
			writeScenarioLogSnapshotNotice(stdout)
			if err := writeScenarioLogTail(stdout, []string{path}, logTailLines(opts, 100)); err != nil {
				return err
			}
			return writeScenarioLogDiscovery(root, home, name, stdout, stderr)
		}
		_, _ = fmt.Fprintf(stdout, "Following lifecycle log for scenario '%s'\n", name)
		return followScenarioLogFiles(stdout, []string{path}, logTailLines(opts, 10))
	}
	_, _ = fmt.Fprintf(stdout, "Recent lifecycle execution for scenario '%s'\n\n", name)
	if err := writeScenarioLogTail(stdout, []string{path}, logTailLines(opts, 100)); err != nil {
		return err
	}
	return writeScenarioLogDiscovery(root, home, name, stdout, stderr)
}

func logTailLines(opts LogOptions, fallback int) int {
	if opts.Tail > 0 {
		return opts.Tail
	}
	return fallback
}

func writeScenarioLogSnapshotNotice(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Non-interactive environment detected; showing a static snapshot instead of streaming logs")
	_, _ = fmt.Fprintln(w)
}

func writeScenarioLogDiscovery(root, home, name string, stdout, stderr io.Writer) error {
	_ = stderr
	paths, err := filepath.Glob(filepath.Join(process.ScenarioLogsDir(home, name), "vrooli.*."+name+".*.log"))
	if err != nil {
		return err
	}
	sort.Strings(paths)
	available := make([]scenarioStepLogInfo, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	orphaned := make([]scenarioStepLogInfo, 0)
	expected := expectedScenarioBackgroundSteps(root, name)
	for _, path := range paths {
		info, ok := parseScenarioStepLogInfo(name, path)
		if !ok {
			orphaned = append(orphaned, scenarioStepLogInfo{Path: path})
			continue
		}
		key := info.Step + ":" + info.Phase
		seen[key] = struct{}{}
		if _, ok := expected[key]; ok {
			available = append(available, info)
		} else {
			orphaned = append(orphaned, info)
		}
	}
	expectedKeys := make([]string, 0, len(expected))
	for key := range expected {
		expectedKeys = append(expectedKeys, key)
	}
	sort.Strings(expectedKeys)
	_, _ = fmt.Fprintln(stdout)
	_, _ = fmt.Fprintln(stdout, "Background step logs:")
	if len(available) == 0 && len(expectedKeys) == 0 {
		_, _ = fmt.Fprintln(stdout, "  (none found)")
	} else {
		sort.Slice(available, func(i, j int) bool {
			if available[i].Phase == available[j].Phase {
				return available[i].Step < available[j].Step
			}
			return available[i].Phase < available[j].Phase
		})
		for _, info := range available {
			_, _ = fmt.Fprintf(stdout, "  %s (%s)\n", info.Step, info.Phase)
		}
		for _, key := range expectedKeys {
			if _, ok := seen[key]; ok {
				continue
			}
			parts := strings.SplitN(key, ":", 2)
			if len(parts) == 2 {
				_, _ = fmt.Fprintf(stdout, "  %s (%s) [missing]\n", parts[0], parts[1])
			}
		}
	}
	if len(orphaned) > 0 {
		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprintln(stdout, "Orphaned background logs:")
		for _, info := range orphaned {
			label := filepath.Base(info.Path)
			if info.Step != "" {
				label = info.Step
				if info.Phase != "" {
					label += " (" + info.Phase + ")"
				}
			}
			_, _ = fmt.Fprintf(stdout, "  %s\n", label)
		}
		_, _ = fmt.Fprintf(stdout, "  Tip: vrooli scenario logs %s --clean\n", name)
	}
	_, _ = fmt.Fprintln(stdout)
	_, _ = fmt.Fprintln(stdout, "Tips:")
	_, _ = fmt.Fprintln(stdout, "  Use --runtime to view all background process logs")
	_, _ = fmt.Fprintln(stdout, "  Use --step <name> to view one background process log")
	_, _ = fmt.Fprintln(stdout, "  Use --follow or -f to stream logs")
	return nil
}

func expectedScenarioBackgroundSteps(root, name string) map[string]struct{} {
	item, err := scenario.Load(root, name, scenario.SandboxEnvFromEnv())
	if err != nil {
		return map[string]struct{}{}
	}
	out := make(map[string]struct{})
	appendPhase := func(phaseName string, phase scenario.Phase) {
		for _, step := range phase.Steps {
			if step.Background {
				out[step.Name+":"+phaseName] = struct{}{}
			}
		}
	}
	appendPhase("setup", item.Manifest.Lifecycle.Setup)
	appendPhase("develop", item.Manifest.Lifecycle.Develop)
	appendPhase("test", item.Manifest.Lifecycle.Test)
	appendPhase("production", item.Manifest.Lifecycle.Production)
	appendPhase("stop", item.Manifest.Lifecycle.Stop)
	return out
}

func parseScenarioStepLogInfo(scenarioName, path string) (scenarioStepLogInfo, bool) {
	base := filepath.Base(path)
	if !strings.HasPrefix(base, "vrooli.") {
		return scenarioStepLogInfo{Path: path}, false
	}
	trimmed := strings.TrimPrefix(base, "vrooli.")
	trimmed = strings.TrimSuffix(trimmed, ".bak")
	trimmed = strings.TrimSuffix(trimmed, ".log")
	parts := strings.Split(trimmed, ".")
	if len(parts) < 3 || parts[1] != scenarioName {
		return scenarioStepLogInfo{Path: path}, false
	}
	return scenarioStepLogInfo{Phase: parts[0], Step: strings.Join(parts[2:], "."), Path: path}, true
}

func writeScenarioLogTail(w io.Writer, paths []string, lines int) error {
	for index, path := range paths {
		if index > 0 {
			_, _ = fmt.Fprintln(w)
		}
		_, _ = fmt.Fprintf(w, "==> %s <==\n", filepath.Base(path))
		data, err := readLastLogLines(path, lines)
		if err != nil {
			return err
		}
		if len(data) > 0 {
			_, _ = w.Write(data)
			if data[len(data)-1] != '\n' {
				_, _ = fmt.Fprintln(w)
			}
		}
	}
	return nil
}

func readLastLogLines(path string, lines int) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if lines <= 0 || len(data) == 0 {
		return data, nil
	}
	split := bytes.Split(data, []byte{'\n'})
	if len(split) > 0 && len(split[len(split)-1]) == 0 {
		split = split[:len(split)-1]
	}
	if len(split) <= lines {
		return data, nil
	}
	result := bytes.Join(split[len(split)-lines:], []byte{'\n'})
	return append(result, '\n'), nil
}

func ReadLastLogLines(path string, lines int) ([]byte, error) {
	return readLastLogLines(path, lines)
}

func followScenarioLogFiles(w io.Writer, paths []string, initialLines int) error {
	if err := writeScenarioLogTail(w, paths, initialLines); err != nil {
		return err
	}
	offsets := make(map[string]int64, len(paths))
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		offsets[path] = info.Size()
	}
	lastHeader := ""
	for {
		progress := false
		for _, path := range paths {
			offset := offsets[path]
			data, nextOffset, err := readScenarioLogDelta(path, offset)
			if err != nil {
				return err
			}
			offsets[path] = nextOffset
			if len(data) == 0 {
				continue
			}
			progress = true
			if len(paths) > 1 && lastHeader != path {
				if lastHeader != "" {
					_, _ = fmt.Fprintln(w)
				}
				_, _ = fmt.Fprintf(w, "==> %s <==\n", filepath.Base(path))
				lastHeader = path
			}
			if _, err := w.Write(data); err != nil {
				return err
			}
		}
		if !progress {
			time.Sleep(250 * time.Millisecond)
		}
	}
}

func readScenarioLogDelta(path string, offset int64) ([]byte, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, offset, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, offset, err
	}
	if offset > info.Size() {
		offset = 0
	}
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return nil, offset, err
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, offset, err
	}
	return data, info.Size(), nil
}

func ReadScenarioLogDelta(path string, offset int64) ([]byte, int64, error) {
	return readScenarioLogDelta(path, offset)
}
