package sessions

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

const apiBase = "/sessions"

type sessionView struct {
	ID            string    `json:"id"`
	ScenarioName  string    `json:"scenario_name"`
	State         string    `json:"state"`
	VNCPort       int       `json:"vnc_port"`
	WSPort        int       `json:"ws_port"`
	Width         int       `json:"width"`
	Height        int       `json:"height"`
	Platform      string    `json:"platform"`
	Headless      bool      `json:"headless"`
	DisplayID     string    `json:"display_id"`
	AppRunning    bool      `json:"app_running"`
	CreatedAt     time.Time `json:"created_at"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	Error         string    `json:"error,omitempty"`
}

func summarize(s sessionView) string {
	mode := "vnc"
	if s.Headless {
		mode = "headless"
	}
	parts := []string{
		fmt.Sprintf("id=%s", s.ID),
		fmt.Sprintf("scenario=%s", s.ScenarioName),
		fmt.Sprintf("state=%s", s.State),
		fmt.Sprintf("mode=%s", mode),
		fmt.Sprintf("size=%dx%d", s.Width, s.Height),
	}
	if !s.Headless {
		parts = append(parts, fmt.Sprintf("vnc_port=%d", s.VNCPort), fmt.Sprintf("ws_port=%d", s.WSPort))
	}
	if s.Headless && s.DisplayID != "" {
		parts = append(parts, fmt.Sprintf("display=%s", s.DisplayID))
	}
	if s.AppRunning {
		parts = append(parts, "app=running")
	}
	if s.Error != "" {
		parts = append(parts, fmt.Sprintf("error=%q", s.Error))
	}
	return strings.Join(parts, " ")
}

// Register builds the "session" subcommand group bound to the given ScenarioApp.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "session",
		Description: "Manage virtual-display sessions",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "list",
				Description: "List sessions",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runList(core, args) },
			},
			{
				Name:        "create",
				Description: "Create a session (--scenario, --headless, --width, --height, --platform)",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runCreate(core, args) },
			},
			{
				Name:        "destroy",
				Description: "Destroy a session by ID",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runDestroy(core, args) },
			},
			{
				Name:        "exec",
				Description: "Execute a control action on a session (exec <id> <action> [--app-path P] [--env K=V]...)",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runExec(core, args) },
			},
			{
				Name:        "logs",
				Description: "Stream session lifecycle state (use -f to follow)",
				NeedsAPI:    true,
				Run:         func(args []string) error { return runLogs(core, args) },
			},
		},
	}
}

func runList(core *cliapp.ScenarioApp, _ []string) error {
	body, err := core.Get(apiBase, nil)
	if err != nil {
		return err
	}
	var sessions []sessionView
	if err := json.Unmarshal(body, &sessions); err != nil {
		return fmt.Errorf("decoding sessions: %w", err)
	}

	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d session(s)", len(sessions))},
		ResultsHeading: "Sessions",
	}
	for _, s := range sessions {
		report.Results = append(report.Results, summarize(s))
	}
	report.RetrievalHints = []string{
		"vrooli-emulator session list",
		"vrooli-emulator session destroy <id>",
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runCreate(core *cliapp.ScenarioApp, args []string) error {
	fs := flag.NewFlagSet("session create", flag.ContinueOnError)
	scenario := fs.String("scenario", "", "Scenario name (required)")
	headless := fs.Bool("headless", false, "Allocate a headless session (no VNC)")
	width := fs.Int("width", 0, "Display width (default 1280)")
	height := fs.Int("height", 0, "Display height (default 720)")
	platform := fs.String("platform", "", "Target platform (default linux)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *scenario == "" {
		return fmt.Errorf("--scenario is required")
	}

	payload := map[string]any{
		"scenario_name": *scenario,
		"headless":      *headless,
	}
	if *width > 0 {
		payload["width"] = *width
	}
	if *height > 0 {
		payload["height"] = *height
	}
	if *platform != "" {
		payload["platform"] = *platform
	}

	body, err := core.Request(http.MethodPost, apiBase, nil, payload)
	if err != nil {
		return err
	}
	var s sessionView
	if err := json.Unmarshal(body, &s); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	report := cliapp.MutationReport{
		Result: []string{summarize(s)},
		Changes: []string{
			fmt.Sprintf("Session %s created on display (%dx%d)", s.ID, s.Width, s.Height),
		},
		NextCommand: []string{
			fmt.Sprintf("vrooli-emulator session exec %s launch_app --app-path /usr/bin/your-app", s.ID),
			fmt.Sprintf("vrooli-emulator session destroy %s", s.ID),
		},
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runDestroy(core *cliapp.ScenarioApp, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: session destroy <id>")
	}
	id := args[0]
	if _, err := core.Request(http.MethodDelete, fmt.Sprintf("%s/%s", apiBase, id), nil, nil); err != nil {
		return err
	}
	report := cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Session %s destroyed", id)},
		Changes: []string{"Display stopped, remote access torn down."},
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runExec(core *cliapp.ScenarioApp, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: session exec <id> <action> [--app-path P] [--env K=V]...")
	}
	id := args[0]
	action := args[1]
	rest := args[2:]

	fs := flag.NewFlagSet("session exec", flag.ContinueOnError)
	appPath := fs.String("app-path", "", "Application path (for launch_app)")
	envs := multiString{}
	fs.Var(&envs, "env", "Environment variable as KEY=VALUE (repeatable)")
	if err := fs.Parse(rest); err != nil {
		return err
	}

	params := map[string]any{}
	if *appPath != "" {
		params["app_path"] = *appPath
	}
	if len(envs) > 0 {
		envMap := map[string]string{}
		for _, kv := range envs {
			k, v, ok := strings.Cut(kv, "=")
			if !ok {
				return fmt.Errorf("invalid --env value %q (expected KEY=VALUE)", kv)
			}
			envMap[k] = v
		}
		params["env"] = envMap
	}

	body, err := core.Request(http.MethodPost, fmt.Sprintf("%s/%s/control", apiBase, id), nil, map[string]any{
		"action": action,
		"params": params,
	})
	if err != nil {
		return err
	}

	report := cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Action %q completed", action)},
		Changes: []string{strings.TrimSpace(string(body))},
	}
	return cliapp.RenderMutationReport(os.Stdout, report)
}

func runLogs(core *cliapp.ScenarioApp, args []string) error {
	follow := false
	var id string
	for _, a := range args {
		switch a {
		case "-f", "--follow":
			follow = true
		default:
			if id == "" {
				id = a
			}
		}
	}
	if id == "" {
		return fmt.Errorf("usage: session logs <id> [-f]")
	}

	fetch := func() error {
		body, err := core.Get(fmt.Sprintf("%s/%s", apiBase, id), nil)
		if err != nil {
			return err
		}
		var s sessionView
		if err := json.Unmarshal(body, &s); err != nil {
			return fmt.Errorf("decoding session: %w", err)
		}
		fmt.Printf("[%s] %s\n", time.Now().UTC().Format(time.RFC3339), summarize(s))
		return nil
	}

	if !follow {
		return fetch()
	}
	for {
		if err := fetch(); err != nil {
			return err
		}
		time.Sleep(2 * time.Second)
	}
}

type multiString []string

func (m *multiString) String() string     { return strings.Join(*m, ",") }
func (m *multiString) Set(v string) error { *m = append(*m, v); return nil }
