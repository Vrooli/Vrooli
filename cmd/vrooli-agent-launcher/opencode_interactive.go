package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	opencodeversion "github.com/vrooli/vrooli/packages/opencode-version"
)

// opencodeSubcommands are the invocations that are not the interactive TUI.
// The TUI is the default command, and only it rejects the V1 global model flag
// while requiring a private server for deterministic per-pane behavior.
var opencodeSubcommands = map[string]struct{}{
	"acp": {}, "api": {}, "auth": {}, "debug": {}, "mcp": {}, "mini": {},
	"models": {}, "pair": {}, "plugin": {}, "reload": {}, "run": {},
	"service": {}, "session": {}, "serve": {}, "stats": {}, "uninstall": {},
	"update": {}, "upgrade": {},
}

// adaptOpenCodeInteractive translates the OpenCode 1 interactive contract to
// OpenCode 2:
//
//   - OpenCode 2's TUI rejects -m/--model; the same model is supplied through
//     OPENCODE_CONFIG_CONTENT, merged with any provider block a model source
//     already placed there.
//   - The TUI defaults to a shared background service. A Web Console pane needs
//     a private server so panes cannot fight over one port.
//
// It is deliberately fail-open: an unreadable version or malformed content
// leaves the launch exactly as it was.
func adaptOpenCodeInteractive(agent, path string, args, environment []string) (outArgs, outEnvironment []string) {
	defer func() {
		if recover() != nil {
			outArgs, outEnvironment = args, environment
		}
	}()
	if agent != "opencode" || path == "" || !interactiveOpenCode(args) {
		return args, environment
	}
	if openCodeMajor(path) < 2 {
		return args, environment
	}
	model, rest := extractOpenCodeModel(args)
	if model != "" {
		content := mergeOpenCodeModelContent(environmentValue(environment, "OPENCODE_CONFIG_CONTENT"), model)
		environment = withEnvironmentValue(environment, "OPENCODE_CONFIG_CONTENT", content)
	}
	if !containsArg(rest, "--standalone") && !containsArg(rest, "--server") {
		rest = append(rest, "--standalone")
	}
	return rest, environment
}

func interactiveOpenCode(args []string) bool {
	for _, arg := range args {
		if arg == "--" {
			return true
		}
		if arg == "-h" || arg == "--help" || arg == "-v" || arg == "--version" {
			return false
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		_, subcommand := opencodeSubcommands[arg]
		return !subcommand
	}
	return true
}

func openCodeMajor(path string) int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "--version").Output()
	if err != nil {
		return 1
	}
	if major, ok := opencodeversion.ParseMajor(string(out)); ok {
		return major
	}
	return 1
}

func extractOpenCodeModel(args []string) (string, []string) {
	model := ""
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-m" || arg == "--model":
			if i+1 < len(args) {
				model = args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "-m="):
			model = strings.TrimPrefix(arg, "-m=")
		case strings.HasPrefix(arg, "--model="):
			model = strings.TrimPrefix(arg, "--model=")
		default:
			out = append(out, arg)
		}
	}
	return strings.TrimSpace(model), out
}

func mergeOpenCodeModelContent(existing, model string) string {
	doc := map[string]any{}
	if strings.TrimSpace(existing) != "" {
		if err := json.Unmarshal([]byte(existing), &doc); err != nil {
			doc = map[string]any{}
		}
	}
	doc["model"] = model
	encoded, err := json.Marshal(doc)
	if err != nil {
		return fmt.Sprintf(`{"model":%q}`, model)
	}
	return string(encoded)
}

func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

func environmentValue(environment []string, key string) string {
	for _, entry := range environment {
		name, value, ok := strings.Cut(entry, "=")
		if ok && name == key {
			return value
		}
	}
	return ""
}

func withEnvironmentValue(environment []string, key, value string) []string {
	out := make([]string, 0, len(environment)+1)
	replaced := false
	for _, entry := range environment {
		name, _, ok := strings.Cut(entry, "=")
		if ok && name == key {
			if !replaced {
				out = append(out, key+"="+value)
				replaced = true
			}
			continue
		}
		out = append(out, entry)
	}
	if !replaced {
		out = append(out, key+"="+value)
	}
	return out
}
