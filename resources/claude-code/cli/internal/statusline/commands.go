package statusline

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/vrooli/cli-core/cliapp"
)

// Handlers binds the statusline commands to one settings file and sample path.
type Handlers struct {
	Settings    Settings
	SamplesPath string
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
}

// Commands registers `statusline run|install|uninstall|status`.
func Commands(h *Handlers) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "statusline",
		Description: "Claude Code statusline quota feed for Agent Manager",
		Subcommands: []cliapp.Command{
			{Name: "run", Description: "Statusline command Claude Code invokes: record a quota sample and print the status line", Usage: "resource-claude-code statusline run < statusline.json", Run: func([]string) error { return h.Run() }},
			{Name: "install", Description: "Reconcile settings.json statusLine to the feed, chaining any previous command (mutating)", Usage: "resource-claude-code statusline install", Run: func([]string) error { return h.Install() }},
			{Name: "uninstall", Description: "Restore the previous statusLine and remove the feed (mutating)", Usage: "resource-claude-code statusline uninstall", Run: func([]string) error { return h.Uninstall() }},
			{Name: "status", Description: "Show the installed statusLine command and sample path", Usage: "resource-claude-code statusline status", Run: func([]string) error { return h.Status() }},
		},
	}
}

func (h *Handlers) Run() error {
	return Feed{SamplesPath: h.SamplesPath, StatePath: h.Settings.StatePath, Stderr: h.Stderr}.Run(h.Stdin, h.Stdout)
}

func (h *Handlers) Install() error {
	result, err := h.Settings.Install(SelfCommand())
	if err != nil {
		return err
	}
	return h.print(result)
}

func (h *Handlers) Uninstall() error {
	result, err := h.Settings.Uninstall()
	if err != nil {
		return err
	}
	return h.print(result)
}

func (h *Handlers) Status() error {
	command, ours, err := h.Settings.Installed()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(h.Stdout, "statusLine: %q (feed installed: %v)\nsamples: %s\nsettings: %s\n", command, ours, h.SamplesPath, h.Settings.SettingsPath)
	return err
}

func (h *Handlers) print(result Result) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(h.Stdout, string(data))
	return err
}

// DefaultHandlers resolves the user's settings file and sample path.
func DefaultHandlers(settingsPath, hookStateDir string) *Handlers {
	samples, _ := DefaultSamplesPath()
	return &Handlers{Settings: DefaultSettings(hookStateDir, settingsPath), SamplesPath: samples, Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr}
}
