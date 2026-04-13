package scenariocli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/process"
)

type LogOptions struct {
	Follow      bool
	ForceFollow bool
	StepName    string
	Runtime     bool
	Lifecycle   bool
	Previous    bool
	Clean       bool
}

var ErrScenarioLogsUsage = errors.New("scenario logs requires a scenario name")

func ParseLogsArgs(args []string) (string, LogOptions, error) {
	name := ""
	opts := LogOptions{}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch arg {
		case "--follow", "-f":
			opts.Follow = true
		case "--force-follow":
			opts.Follow = true
			opts.ForceFollow = true
		case "--step":
			if index+1 >= len(args) {
				return "", LogOptions{}, fmt.Errorf("scenario logs --step requires a step name")
			}
			index++
			opts.StepName = args[index]
		case "--runtime":
			opts.Runtime = true
		case "--lifecycle":
			opts.Lifecycle = true
		case "--previous":
			opts.Previous = true
		case "--clean":
			opts.Clean = true
		case "--help", "-h":
			return "", LogOptions{}, ErrScenarioLogsUsage
		default:
			if strings.HasPrefix(arg, "-") {
				return "", LogOptions{}, fmt.Errorf("unknown option for scenario logs: %s", arg)
			}
			if name != "" {
				return "", LogOptions{}, fmt.Errorf("scenario logs accepts exactly one scenario name")
			}
			name = arg
		}
	}
	return name, opts, nil
}

func ShowLogsUsage(w io.Writer) error {
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
		return ErrScenarioLogsUsage
	}
	logsRoot := filepath.Join(home, ".vrooli", "logs", "scenarios")
	entries, err := os.ReadDir(logsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrScenarioLogsUsage
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
	return ErrScenarioLogsUsage
}
