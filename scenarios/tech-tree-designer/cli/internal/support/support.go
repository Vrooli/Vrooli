package support

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

const (
	DefaultUIPort = "35000"
	CLIName       = "tech-tree-designer"
)

type Dependencies struct {
	Core     *cliapp.ScenarioApp
	Selector *TreeSelector
}

func (d Dependencies) Get(path string, query url.Values) ([]byte, error) {
	return d.Core.Get(path, d.Selector.Append(query))
}

func (d Dependencies) Request(method, path string, query url.Values, body interface{}) ([]byte, error) {
	return d.Core.Request(method, path, d.Selector.Append(query), body)
}

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

func BuildQuery(params map[string]string) url.Values {
	values := url.Values{}
	for key, value := range params {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		values.Set(key, value)
	}
	return values
}

func ResolveUIPort() string {
	if port := strings.TrimSpace(cliutil.DetectPortFromVrooli(CLIName, "UI_PORT")()); port != "" {
		return port
	}
	return DefaultUIPort
}

func DashboardURL() string {
	return fmt.Sprintf("http://localhost:%s", ResolveUIPort())
}

func OpenBrowser(target string) (bool, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return false, fmt.Errorf("url is required")
	}
	candidates := [][]string{
		{"xdg-open", target},
		{"open", target},
		{"cmd", "/c", "start", target},
	}
	for _, candidate := range candidates {
		if _, err := exec.LookPath(candidate[0]); err != nil {
			continue
		}
		cmd := exec.Command(candidate[0], candidate[1:]...)
		if err := cmd.Start(); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func TrimmedCSV(value string) []string {
	return cliutil.ParseCSV(value)
}

func TreeScopeLine(selector *TreeSelector) string {
	if selector == nil {
		return "Tree scope: active default tree"
	}
	return selector.ScopeLine()
}

func FormatPercent(value float64) string {
	return fmt.Sprintf("%.0f%%", value)
}

func FormatRatio(value float64) string {
	return fmt.Sprintf("%.2f", value)
}

func FormatDate(value *time.Time) string {
	if value == nil || value.IsZero() {
		return "n/a"
	}
	return value.Format("2006-01-02")
}

func FormatDateTime(value time.Time) string {
	if value.IsZero() {
		return "unknown"
	}
	return value.Format(time.RFC3339)
}
