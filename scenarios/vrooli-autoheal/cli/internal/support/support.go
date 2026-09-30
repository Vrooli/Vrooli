package support

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func ParseFlags(fs *flag.FlagSet, args []string) error {
	return cliutil.ParseInterspersed(fs, args)
}

func Decode(body []byte, dest interface{}) error {
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	return nil
}

func PrettyJSON(body []byte) string {
	var out bytes.Buffer
	if err := json.Indent(&out, body, "", "  "); err == nil {
		return out.String()
	}
	return string(body)
}

func StatusGlyph(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "ok", "healthy":
		return "OK"
	case "warning":
		return "WARN"
	case "critical", "unhealthy":
		return "CRIT"
	default:
		return "INFO"
	}
}

func BoolWord(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func ResolveLoopBinary() (string, error) {
	names := []string{"vrooli-autoheal-loop"}
	if runtime.GOOS == "windows" {
		names = []string{"vrooli-autoheal-loop.exe", "vrooli-autoheal-loop"}
	}
	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}

	scenarioRoot := cliutil.ResolveScenarioPath("vrooli-autoheal")
	if strings.TrimSpace(scenarioRoot) == "" {
		return "", errors.New("unable to resolve scenario path for vrooli-autoheal")
	}

	candidates := []string{
		filepath.Join(scenarioRoot, "cli", "vrooli-autoheal-loop"),
		filepath.Join(scenarioRoot, "cli", "loop", "vrooli-autoheal-loop"),
	}
	if runtime.GOOS == "windows" {
		candidates = append([]string{
			filepath.Join(scenarioRoot, "cli", "vrooli-autoheal-loop.exe"),
			filepath.Join(scenarioRoot, "cli", "loop", "vrooli-autoheal-loop.exe"),
		}, candidates...)
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("loop binary not found; build it with 'cd %s/cli/loop && go build -o ../vrooli-autoheal-loop .'", scenarioRoot)
}

func BuildQuery(values map[string]string) url.Values {
	query := url.Values{}
	for key, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		query.Set(key, trimmed)
	}
	return query
}

func Confirm(prompt string) (bool, error) {
	fmt.Fprintf(os.Stdout, "%s [y/N]: ", strings.TrimSpace(prompt))
	var input string
	if _, err := fmt.Fscanln(os.Stdin, &input); err != nil {
		if errors.Is(err, io.EOF) {
			return false, nil
		}
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func TruncateLines(value string, maxLines int) string {
	value = strings.TrimSpace(value)
	if value == "" || maxLines <= 0 {
		return value
	}
	lines := strings.Split(value, "\n")
	if len(lines) <= maxLines {
		return value
	}
	return strings.Join(lines[:maxLines], "\n") + "\n..."
}
