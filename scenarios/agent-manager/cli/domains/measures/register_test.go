package measures

import (
	"strings"
	"testing"
)

func TestRegisterDeclaresMeasureCommands(t *testing.T) {
	group := Register()
	if len(group.Subcommands) == 0 {
		t.Fatal("measure command group must expose its declared measures")
	}
}

func TestTokenAttributionCommandDeclaresSupportedDimensionsAndViews(t *testing.T) {
	group := Register()
	var commandName string
	for _, command := range group.Subcommands {
		if command.Name == "token-attribution" {
			commandName = command.Name
			if err := command.Args.Validate(); err != nil {
				t.Fatalf("token attribution args invalid: %v", err)
			}
			if got := command.Args.Flags[1].Values; strings.Join(got, ",") != strings.Join(tokenAttributionByValues, ",") {
				t.Fatalf("unexpected --by values: %v", got)
			}
			if got := command.Args.Flags[2].Values; strings.Join(got, ",") != strings.Join(tokenAttributionViewValues, ",") {
				t.Fatalf("unexpected --view values: %v", got)
			}
		}
	}
	if commandName == "" {
		t.Fatal("token-attribution command was not registered")
	}
}

func TestNormalizeTokenAttributionByMapsCLIValuesToAPIValues(t *testing.T) {
	tests := map[string]string{
		"capability":         "capability",
		"executable":         "executable",
		"command-path":       "command_path",
		"scenario-operation": "target_scenario_operation",
	}
	for input, expected := range tests {
		got, err := normalizeTokenAttributionBy(input)
		if err != nil || got != expected {
			t.Fatalf("normalizeTokenAttributionBy(%q) = %q, %v; want %q", input, got, err, expected)
		}
	}
	if _, err := normalizeTokenAttributionBy("bad"); err == nil || !strings.Contains(err.Error(), "accepted values") {
		t.Fatalf("invalid grouping error should enumerate accepted values: %v", err)
	}
}
