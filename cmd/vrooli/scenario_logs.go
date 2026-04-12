package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
)

type scenarioLogOptions struct {
	follow      bool
	forceFollow bool
	stepName    string
	runtime     bool
	lifecycle   bool
	previous    bool
	clean       bool
}

type scenarioStepLogInfo struct {
	Phase string
	Step  string
	Path  string
}

var errScenarioLogsUsage = errors.New("scenario logs requires a scenario name")

func runScenarioLogsCommand(root string, globals globalOptions, args []string, stdout, stderr io.Writer) error {
	app := configuredApp()
	return runScenarioLogsCommandWithApp(app, &commandContext{
		Root:    root,
		Globals: globals,
		Stdout:  stdout,
		Stderr:  stderr,
		app:     app,
	}, args)
}

func runScenarioLogsCommandWithApp(app *App, ctx *commandContext, args []string) error {
	name, opts, err := parseScenarioLogsArgs(args)
	if err != nil {
		var usageErr *showScenarioLogsUsageError
		if errors.As(err, &usageErr) {
			showErr := showScenarioLogsUsage(ctx.Stdout)
			if showErr != nil && !errors.Is(showErr, errScenarioLogsUsage) {
				return showErr
			}
			return nil
		}
		return err
	}
	if name == "" {
		return showScenarioLogsUsage(ctx.Stdout)
	}

	home, err := ctx.HomeDir()
	if err != nil {
		return err
	}

	if opts.clean {
		return cleanScenarioLogs(ctx.Root, home, name, ctx.Stdout)
	}
	if opts.runtime {
		return showScenarioRuntimeLogs(home, name, opts, ctx.Stdout)
	}
	if opts.stepName != "" {
		return showScenarioStepLog(home, name, opts, ctx.Stdout)
	}
	return showScenarioLifecycleLog(ctx.Root, home, name, opts, ctx.Stdout, ctx.Stderr)
}

func parseScenarioLogsArgs(args []string) (string, scenarioLogOptions, error) {
	name := ""
	opts := scenarioLogOptions{}

	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch arg {
		case "--follow", "-f":
			opts.follow = true
		case "--force-follow":
			opts.follow = true
			opts.forceFollow = true
		case "--step":
			if index+1 >= len(args) {
				return "", scenarioLogOptions{}, errors.New("scenario logs --step requires a step name")
			}
			index++
			opts.stepName = args[index]
		case "--runtime":
			opts.runtime = true
		case "--lifecycle":
			opts.lifecycle = true
		case "--previous":
			opts.previous = true
		case "--clean":
			opts.clean = true
		case "--help", "-h":
			return "", scenarioLogOptions{}, &showScenarioLogsUsageError{}
		default:
			if strings.HasPrefix(arg, "-") {
				return "", scenarioLogOptions{}, fmt.Errorf("unknown option for scenario logs: %s", arg)
			}
			if name != "" {
				return "", scenarioLogOptions{}, errors.New("scenario logs accepts exactly one scenario name")
			}
			name = arg
		}
	}

	return name, opts, nil
}

type showScenarioLogsUsageError struct{}

func (*showScenarioLogsUsageError) Error() string { return "usage requested" }

func showScenarioLogsUsage(w io.Writer) error {
	home, _ := process.HomeDir()

	_, _ = fmt.Fprintln(w, "Usage: vrooli scenario logs <name> [options]")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Options:")
	_, _ = fmt.Fprintln(w, "  --follow, -f        Follow log output in real time")
	_, _ = fmt.Fprintln(w, "  --step <name>       View a specific background step log")
	_, _ = fmt.Fprintln(w, "  --runtime           View all background process logs")
	_, _ = fmt.Fprintln(w, "  --lifecycle         View lifecycle log (default)")
	_, _ = fmt.Fprintln(w, "  --previous          View the previous step log backup (.log.bak)")
	_, _ = fmt.Fprintln(w, "  --force-follow      Stream even in non-interactive environments")
	_, _ = fmt.Fprintln(w, "  --clean             Remove orphaned background logs")

	if home == "" {
		return errScenarioLogsUsage
	}

	logsRoot := filepath.Join(home, ".vrooli", "logs", "scenarios")
	entries, err := os.ReadDir(logsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return errScenarioLogsUsage
		}
		return err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Available scenarios with logs:")
	if len(names) == 0 {
		_, _ = fmt.Fprintln(w, "  (none found)")
	} else {
		for _, name := range names {
			_, _ = fmt.Fprintf(w, "  %s\n", name)
		}
	}
	return errScenarioLogsUsage
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

func showScenarioRuntimeLogs(home, name string, opts scenarioLogOptions, stdout io.Writer) error {
	logsDir := process.ScenarioLogsDir(home, name)
	paths, err := filepath.Glob(filepath.Join(logsDir, "*.log"))
	if err != nil {
		return err
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return fmt.Errorf("no runtime log files found for scenario %q", name)
	}

	if opts.follow {
		if !opts.forceFollow && !writerSupportsStreaming(stdout) {
			writeScenarioLogSnapshotNotice(stdout)
			return writeScenarioLogTail(stdout, paths, 50)
		}
		_, _ = fmt.Fprintf(stdout, "Following runtime logs for scenario '%s'\n", name)
		return followScenarioLogFiles(stdout, paths, 10)
	}

	_, _ = fmt.Fprintf(stdout, "Recent runtime logs for scenario '%s'\n\n", name)
	return writeScenarioLogTail(stdout, paths, 50)
}

func showScenarioStepLog(home, name string, opts scenarioLogOptions, stdout io.Writer) error {
	logsDir := process.ScenarioLogsDir(home, name)
	suffix := ".log"
	if opts.previous {
		suffix = ".log.bak"
	}

	pattern := filepath.Join(logsDir, "vrooli.*."+name+"."+opts.stepName+suffix)
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		if opts.previous {
			return fmt.Errorf("no previous log found for step %q", opts.stepName)
		}
		return fmt.Errorf("no log found for step %q", opts.stepName)
	}

	path := paths[0]
	if opts.follow {
		if !opts.forceFollow && !writerSupportsStreaming(stdout) {
			writeScenarioLogSnapshotNotice(stdout)
			return writeScenarioLogTail(stdout, []string{path}, 100)
		}
		_, _ = fmt.Fprintf(stdout, "Following log for step '%s' in scenario '%s'\n", opts.stepName, name)
		return followScenarioLogFiles(stdout, []string{path}, 10)
	}

	_, _ = fmt.Fprintf(stdout, "Recent log for step '%s' in scenario '%s'\n\n", opts.stepName, name)
	return writeScenarioLogTail(stdout, []string{path}, 100)
}

func showScenarioLifecycleLog(root, home, name string, opts scenarioLogOptions, stdout, stderr io.Writer) error {
	_ = stderr

	path := process.ScenarioLifecycleLogPath(home, name)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no lifecycle log found for scenario %q", name)
		}
		return err
	}

	if opts.follow {
		if !opts.forceFollow && !writerSupportsStreaming(stdout) {
			writeScenarioLogSnapshotNotice(stdout)
			if err := writeScenarioLogTail(stdout, []string{path}, 100); err != nil {
				return err
			}
			return writeScenarioLogDiscovery(root, home, name, stdout, stderr)
		}
		_, _ = fmt.Fprintf(stdout, "Following lifecycle log for scenario '%s'\n", name)
		return followScenarioLogFiles(stdout, []string{path}, 10)
	}

	_, _ = fmt.Fprintf(stdout, "Recent lifecycle execution for scenario '%s'\n\n", name)
	if err := writeScenarioLogTail(stdout, []string{path}, 100); err != nil {
		return err
	}
	return writeScenarioLogDiscovery(root, home, name, stdout, stderr)
}

func writeScenarioLogSnapshotNotice(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Non-interactive environment detected; showing a static snapshot instead of streaming logs")
	_, _ = fmt.Fprintln(w)
}

func writeScenarioLogDiscovery(root, home, name string, stdout, stderr io.Writer) error {
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
			if len(parts) != 2 {
				continue
			}
			_, _ = fmt.Fprintf(stdout, "  %s (%s) [missing]\n", parts[0], parts[1])
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
	const prefix = "vrooli."
	if !strings.HasPrefix(base, prefix) {
		return scenarioStepLogInfo{Path: path}, false
	}

	trimmed := strings.TrimPrefix(base, prefix)
	trimmed = strings.TrimSuffix(trimmed, ".bak")
	trimmed = strings.TrimSuffix(trimmed, ".log")
	parts := strings.Split(trimmed, ".")
	if len(parts) < 3 {
		return scenarioStepLogInfo{Path: path}, false
	}
	if parts[1] != scenarioName {
		return scenarioStepLogInfo{Path: path}, false
	}
	return scenarioStepLogInfo{
		Phase: parts[0],
		Step:  strings.Join(parts[2:], "."),
		Path:  path,
	}, true
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
