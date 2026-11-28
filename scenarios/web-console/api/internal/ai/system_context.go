package ai

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// CommandSystemPrompt and SuggestSystemPrompt are the base instructions
// used by BuildCommandSystemPrompt / BuildSuggestSystemPrompt.
const (
	CommandSystemPrompt = "You are a command-line assistant. Given a natural language description, output ONLY the shell command. No explanation, no markdown, no backticks."
	SuggestSystemPrompt = "You are a command-line assistant. Given a natural language description, output 1 to 3 shell commands (one per line) that accomplish the task. If there is only one reasonable command, output just that one. No explanation, no numbering, no markdown, no backticks."
)

// LookPathFunc abstracts exec.LookPath for testability.
type LookPathFunc func(file string) (string, error)

// ToolCategory groups related CLI tools for prompt formatting.
type ToolCategory struct {
	Name  string
	Tools []string
}

// SystemContext holds detected environment info for enriching AI prompts.
type SystemContext struct {
	OS     string
	Arch   string
	Distro string
	Shell  string
	Tools  map[string][]string
}

var defaultToolCategories = []ToolCategory{
	{"Search", []string{"rg", "fd", "fzf", "ag", "grep", "find"}},
	{"File viewing", []string{"bat", "less", "cat", "head", "tail"}},
	{"System", []string{"htop", "top", "ps", "free", "df", "du"}},
	{"Containers", []string{"docker", "kubectl", "podman", "helm"}},
	{"Languages", []string{"go", "node", "python3", "python", "rustc", "java", "ruby"}},
	{"Version control", []string{"git", "gh"}},
	{"Data processing", []string{"jq", "yq", "sqlite3", "xsv"}},
	{"Networking", []string{"curl", "wget", "ssh", "dig"}},
	{"Editors", []string{"vim", "nvim", "nano", "code", "emacs"}},
	{"Archiving", []string{"tar", "zip", "unzip", "gzip", "zstd"}},
	{"Build tools", []string{"make", "cmake", "cargo", "npm", "yarn", "pnpm"}},
}

// DiscoverSystemContext probes the host environment for OS info, shell, and
// available CLI tools.
func DiscoverSystemContext(lookPath LookPathFunc) *SystemContext {
	ctx := &SystemContext{
		OS:     runtime.GOOS,
		Arch:   runtime.GOARCH,
		Distro: readDistro(),
		Shell:  shellBasename(os.Getenv("SHELL")),
		Tools:  make(map[string][]string),
	}

	for _, cat := range defaultToolCategories {
		var found []string
		for _, tool := range cat.Tools {
			if _, err := lookPath(tool); err == nil {
				found = append(found, tool)
			}
		}
		if len(found) > 0 {
			ctx.Tools[cat.Name] = found
		}
	}

	if extra := os.Getenv("WC_EXTRA_TOOLS"); extra != "" {
		var found []string
		for _, tool := range strings.Split(extra, ",") {
			tool = strings.TrimSpace(tool)
			if tool == "" {
				continue
			}
			if _, err := lookPath(tool); err == nil {
				found = append(found, tool)
			}
		}
		if len(found) > 0 {
			ctx.Tools["Custom"] = found
		}
	}

	return ctx
}

// DefaultLookPath is the production LookPathFunc using exec.LookPath.
var DefaultLookPath LookPathFunc = exec.LookPath

func readDistro() string {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			val := strings.TrimPrefix(line, "PRETTY_NAME=")
			return strings.Trim(val, `"`)
		}
	}
	return ""
}

func shellBasename(shell string) string {
	if shell == "" {
		return ""
	}
	if idx := strings.LastIndex(shell, "/"); idx >= 0 {
		return shell[idx+1:]
	}
	return shell
}

// CountFoundTools returns the total number of discovered tools.
func CountFoundTools(tools map[string][]string) int {
	n := 0
	for _, v := range tools {
		n += len(v)
	}
	return n
}

// BuildCommandSystemPrompt constructs the system prompt for single-command
// generation, enriched with environment context when available.
func BuildCommandSystemPrompt(ctx *SystemContext) string {
	if ctx == nil {
		return CommandSystemPrompt
	}
	return buildEnrichedPrompt(ctx, CommandSystemPrompt)
}

// BuildSuggestSystemPrompt constructs the system prompt for multi-command
// suggestion, enriched with environment context when available.
func BuildSuggestSystemPrompt(ctx *SystemContext) string {
	if ctx == nil {
		return SuggestSystemPrompt
	}
	return buildEnrichedPrompt(ctx, SuggestSystemPrompt)
}

func buildEnrichedPrompt(ctx *SystemContext, baseInstruction string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("You are a command-line assistant on %s/%s", ctx.OS, ctx.Arch))
	if ctx.Distro != "" || ctx.Shell != "" {
		b.WriteString(" (")
		parts := make([]string, 0, 2)
		if ctx.Distro != "" {
			parts = append(parts, ctx.Distro)
		}
		if ctx.Shell != "" {
			parts = append(parts, ctx.Shell)
		}
		b.WriteString(strings.Join(parts, ", "))
		b.WriteString(")")
	}
	b.WriteString(".\n")

	if len(ctx.Tools) > 0 {
		b.WriteString("\nAvailable tools:\n")
		emitted := make(map[string]bool)
		for _, cat := range defaultToolCategories {
			if tools, ok := ctx.Tools[cat.Name]; ok {
				b.WriteString(fmt.Sprintf("- %s: %s\n", cat.Name, strings.Join(tools, ", ")))
				emitted[cat.Name] = true
			}
		}
		for name, tools := range ctx.Tools {
			if !emitted[name] {
				b.WriteString(fmt.Sprintf("- %s: %s\n", name, strings.Join(tools, ", ")))
			}
		}
		b.WriteString("\nPrefer modern alternatives when available (rg over grep, fd over find, bat over cat). Use tools you know are installed.\n")
	}

	b.WriteString("\n")
	b.WriteString(baseInstruction)
	return b.String()
}
