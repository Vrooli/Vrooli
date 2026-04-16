package settings

import (
	"strconv"

	"agent-inbox/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

type suggestionsSettings struct {
	AutoSuggest struct {
		Enabled         bool `json:"enabled"`
		DebounceMS      int  `json:"debounceMs"`
		ThrottleMS      int  `json:"throttleMs"`
		MinInputLength  int  `json:"minInputLength"`
		MinScorePercent int  `json:"minScorePercent"`
		MaxSuggestions  int  `json:"maxSuggestions"`
	} `json:"autoSuggest"`
}

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "settings",
		Description: "Operational settings for agent inbox",
		Subcommands: []cliapp.Command{
			{Name: "yolo", NeedsAPI: true, Description: "Get or set YOLO tool approval mode", Run: func(args []string) error { return runYolo(core, args) }},
			{Name: "suggestions", NeedsAPI: true, Description: "Get or update suggestions settings", Run: func(args []string) error { return runSuggestions(core, args) }},
		},
	}
}

func runYolo(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("settings yolo")
	setValue := fs.String("set", "", "Set to true or false; omit to only view")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	if value, err := support.ParseOptionalBool(*setValue); err != nil {
		return err
	} else if value != nil {
		if _, err := core.Request("POST", "/settings/yolo-mode", nil, map[string]bool{"enabled": *value}); err != nil {
			return err
		}
	}

	var resp struct {
		Enabled bool `json:"enabled"`
	}
	if err := support.GetJSON(core, "/settings/yolo-mode", &resp); err != nil {
		return err
	}

	report := cliapp.OperationalReport{
		Status: []string{
			"YOLO mode: " + support.BoolLabel(resp.Enabled),
			"Meaning: skip manual approval for tools when enabled",
		},
		NextSteps: []string{support.CLIName + " settings yolo --set false", support.CLIName + " tool list"},
	}
	return support.PrintOperational(*jsonOutput, report)
}

func runSuggestions(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("settings suggestions")
	enabled := fs.String("enabled", "", "Set suggestions enabled to true or false")
	debounce := fs.Int("debounce-ms", -1, "Set debounce in milliseconds")
	throttle := fs.Int("throttle-ms", -1, "Set throttle in milliseconds")
	minInput := fs.Int("min-input-length", -1, "Set minimum input length")
	minScore := fs.Int("min-score-percent", -1, "Set minimum score percent")
	maxSuggestions := fs.Int("max-suggestions", -1, "Set maximum suggestions")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var current suggestionsSettings
	if err := support.GetJSON(core, "/settings/suggestions", &current); err != nil {
		return err
	}

	changed := false
	if value, err := support.ParseOptionalBool(*enabled); err != nil {
		return err
	} else if value != nil {
		current.AutoSuggest.Enabled = *value
		changed = true
	}
	if *debounce >= 0 {
		current.AutoSuggest.DebounceMS = *debounce
		changed = true
	}
	if *throttle >= 0 {
		current.AutoSuggest.ThrottleMS = *throttle
		changed = true
	}
	if *minInput >= 0 {
		current.AutoSuggest.MinInputLength = *minInput
		changed = true
	}
	if *minScore >= 0 {
		current.AutoSuggest.MinScorePercent = *minScore
		changed = true
	}
	if *maxSuggestions >= 0 {
		current.AutoSuggest.MaxSuggestions = *maxSuggestions
		changed = true
	}

	if changed {
		body, err := core.Request("POST", "/settings/suggestions", nil, current)
		if err != nil {
			return err
		}
		if err := support.Decode(body, &current); err != nil {
			return err
		}
	}

	status := []string{
		"Auto-suggest: " + support.BoolLabel(current.AutoSuggest.Enabled),
		"Debounce: " + strconv.Itoa(current.AutoSuggest.DebounceMS) + "ms",
		"Throttle: " + strconv.Itoa(current.AutoSuggest.ThrottleMS) + "ms",
		"Min input length: " + strconv.Itoa(current.AutoSuggest.MinInputLength),
		"Min score percent: " + strconv.Itoa(current.AutoSuggest.MinScorePercent),
		"Max suggestions: " + strconv.Itoa(current.AutoSuggest.MaxSuggestions),
	}
	report := cliapp.OperationalReport{
		Status:    status,
		NextSteps: []string{support.CLIName + " settings suggestions --enabled true", support.CLIName + " settings suggestions --min-score-percent 65"},
	}
	if changed {
		report.Triage = []cliapp.TriageGroup{{
			Heading: "Applied",
			Items:   []string{"Suggestions settings were updated and persisted."},
		}}
	}
	return support.PrintOperational(*jsonOutput, report)
}
