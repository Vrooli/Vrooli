// Package actions provides CLI commands for Action discovery and validation.
//
// DOC: docs/concepts/ACTIONS.md
// DOC: docs/reference/cli-commands.md#actions
package actions

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"prompt-manager/cli/internal/appctx"
	"sort"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

type Action struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Description string                  `json:"description,omitempty"`
	Status      string                  `json:"status"`
	Owner       ActionOwner             `json:"owner"`
	Command     ActionCommand           `json:"command"`
	Inputs      map[string]ActionInput  `json:"inputs,omitempty"`
	Outputs     map[string]ActionOutput `json:"outputs,omitempty"`
	Permissions ActionPermissions       `json:"permissions,omitempty"`
	Examples    []ActionExample         `json:"examples,omitempty"`
	Tags        []string                `json:"tags,omitempty"`
	Execution   *ActionExecution        `json:"execution,omitempty"`
	Validation  *ActionValidation       `json:"validation,omitempty"`
	CreatedAt   string                  `json:"createdAt"`
	UpdatedAt   string                  `json:"updatedAt"`
	Pack        string                  `json:"pack,omitempty"`
}

type ActionOwner struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type ActionCommand struct {
	Argv []string `json:"argv"`
}

type ActionInput struct {
	Type           string   `json:"type"`
	Description    string   `json:"description,omitempty"`
	Required       bool     `json:"required,omitempty"`
	Enum           []string `json:"enum,omitempty"`
	Default        any      `json:"default,omitempty"`
	Pattern        string   `json:"pattern,omitempty"`
	Min            *float64 `json:"min,omitempty"`
	Max            *float64 `json:"max,omitempty"`
	MaxLength      *int     `json:"maxLength,omitempty"`
	AllowMultiline bool     `json:"allowMultiline,omitempty"`
}

type ActionOutput struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

type ActionPermissions struct {
	FilesystemRead   bool `json:"filesystemRead,omitempty"`
	FilesystemWrite  bool `json:"filesystemWrite,omitempty"`
	LocalhostNetwork bool `json:"localhostNetwork,omitempty"`
	ExternalNetwork  bool `json:"externalNetwork,omitempty"`
	APIRead          bool `json:"apiRead,omitempty"`
	APIWrite         bool `json:"apiWrite,omitempty"`
	ProcessStart     bool `json:"processStart,omitempty"`
	ProcessStop      bool `json:"processStop,omitempty"`
	HostConfigure    bool `json:"hostConfigure,omitempty"`
	SecretRead       bool `json:"secretRead,omitempty"`
	SecretWrite      bool `json:"secretWrite,omitempty"`
	Destructive      bool `json:"destructive,omitempty"`
}

type ActionExample struct {
	Description string         `json:"description,omitempty"`
	Input       map[string]any `json:"input,omitempty"`
}

type ActionExecution struct {
	TimeoutSeconds *int   `json:"timeoutSeconds,omitempty"`
	OutputMode     string `json:"outputMode,omitempty"`
	RunEligible    *bool  `json:"runEligible,omitempty"`
}

type ActionValidation struct {
	Mode string   `json:"mode,omitempty"`
	Argv []string `json:"argv,omitempty"`
}

type ValidationResponse struct {
	ActionID             string             `json:"actionId"`
	Valid                bool               `json:"valid"`
	Runnable             bool               `json:"runnable"`
	Unvalidated          bool               `json:"unvalidated,omitempty"`
	RequiresConfirmation bool               `json:"requiresConfirmation,omitempty"`
	Status               string             `json:"status"`
	Command              *CommandResolution `json:"command,omitempty"`
	Checks               []ValidationCheck  `json:"checks"`
	Action               *Action            `json:"action,omitempty"`
}

type MutationResponse struct {
	Action     *Action            `json:"action"`
	Validation ValidationResponse `json:"validation"`
}

type RunRequest struct {
	Input  map[string]any `json:"input,omitempty"`
	DryRun bool           `json:"dryRun,omitempty"`
}

type RunResponse struct {
	ActionID        string             `json:"actionId"`
	Status          string             `json:"status"`
	ExitCode        *int               `json:"exitCode,omitempty"`
	DurationMs      int64              `json:"durationMs"`
	Argv            []string           `json:"argv,omitempty"`
	Stdout          string             `json:"stdout,omitempty"`
	Stderr          string             `json:"stderr,omitempty"`
	StdoutTruncated bool               `json:"stdoutTruncated,omitempty"`
	StderrTruncated bool               `json:"stderrTruncated,omitempty"`
	Output          map[string]any     `json:"output,omitempty"`
	Validation      ValidationResponse `json:"validation"`
	Error           string             `json:"error,omitempty"`
}

type ValidationCheck struct {
	Code    string `json:"code"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
}

type CommandResolution struct {
	Certainty            string      `json:"certainty"`
	Owner                ActionOwner `json:"owner"`
	Target               string      `json:"target"`
	CommandPath          []string    `json:"commandPath,omitempty"`
	Effect               string      `json:"effect,omitempty"`
	Permissions          []string    `json:"permissions,omitempty"`
	RunSurfaces          []string    `json:"runSurfaces,omitempty"`
	RequiresConfirmation bool        `json:"requiresConfirmation,omitempty"`
	Message              string      `json:"message,omitempty"`
}

func Commands(ctx appctx.Context) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Actions",
		Commands: []cliapp.Command{
			{
				Name:        "action",
				Aliases:     []string{"actions"},
				NeedsAPI:    true,
				Description: "Manage Actions (list|show|create|update|delete|validate|run)",
				Usage:       "prompt-manager action <subcommand> [args]",
				HelpText:    usageText(),
				Run: func(args []string) error {
					return route(ctx, args)
				},
			},
		},
	}
}

func route(ctx appctx.Context, args []string) error {
	if len(args) == 0 {
		fmt.Println(usageText())
		return nil
	}

	subcommand := args[0]
	subArgs := args[1:]
	switch subcommand {
	case "list", "ls":
		return cmdList(ctx, subArgs)
	case "show", "get":
		return cmdShow(ctx, subArgs)
	case "create", "add":
		return cmdCreate(ctx, subArgs)
	case "update", "edit":
		return cmdUpdate(ctx, subArgs)
	case "delete", "rm":
		return cmdDelete(ctx, subArgs)
	case "validate", "check":
		return cmdValidate(ctx, subArgs)
	case "run":
		return cmdRun(ctx, subArgs)
	default:
		return fmt.Errorf("unknown subcommand: %s\n\n%s", subcommand, usageText())
	}
}

func usageText() string {
	return `Usage: prompt-manager action <subcommand> [args]

Subcommands:
  list, ls             List Actions
  show, get <id>       Show an Action contract
  create, add          Create an Action from an action.json file
  update, edit <id>    Update an Action from an action.json file
  delete, rm <id>      Archive or hard-delete an Action
  validate, check <id> Validate an Action contract without running it
  run <id>             Run an Action through the governed API runtime`
}

func cmdList(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("action list", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	pack := fs.String("pack", "", "Filter by pack (core|local|drafts)")
	status := fs.String("status", "", "Filter by status (active|draft|archived)")
	owner := fs.String("owner", "", "Filter by owner ID")
	tag := fs.String("tag", "", "Filter by tag")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	addQuery(query, "pack", *pack)
	addQuery(query, "status", *status)
	addQuery(query, "owner", *owner)
	addQuery(query, "tag", *tag)

	var actions []Action
	if err := ctx.GetWithQuery("/actions", query, &actions); err != nil {
		return fmt.Errorf("failed to list actions: %w", err)
	}
	if *jsonOut {
		return writeJSON(actions)
	}
	if len(actions) == 0 {
		fmt.Println("No actions found")
		return nil
	}

	fmt.Println("Actions:")
	for _, action := range actions {
		tags := ""
		if len(action.Tags) > 0 {
			tags = " [" + strings.Join(action.Tags, ", ") + "]"
		}
		owner := strings.Trim(action.Owner.Type+":"+action.Owner.ID, ":")
		packText := action.Pack
		if packText == "" {
			packText = "unknown"
		}
		fmt.Printf("  %s - %s (%s, %s)%s [%s]\n", action.Name, packText, action.Status, owner, tags, action.ID)
	}
	return nil
}

func cmdShow(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("action show", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: action show <id> [--json]")
	}

	var action Action
	if err := ctx.Get(actionPath(fs.Arg(0)), &action); err != nil {
		return fmt.Errorf("failed to get action: %w", err)
	}
	if *jsonOut {
		return writeJSON(action)
	}
	printAction(action)
	return nil
}

func cmdCreate(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("action create", flag.ContinueOnError)
	file := fs.String("file", "", "Path to action.json")
	pack := fs.String("pack", "", "Target pack (core|local|drafts); defaults to API policy")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 0 || *file == "" {
		return fmt.Errorf("usage: action create --file=action.json [--pack=core|local|drafts] [--json]")
	}

	action, err := readActionFile(*file)
	if err != nil {
		return err
	}
	if *pack != "" {
		action.Pack = *pack
	}

	var result MutationResponse
	if err := ctx.Post("/actions", action, &result); err != nil {
		return fmt.Errorf("failed to create action: %w", err)
	}
	if *jsonOut {
		return writeJSON(result)
	}
	if result.Action == nil {
		return fmt.Errorf("create response did not include an action")
	}
	fmt.Printf("Created action: %s [%s]\n", result.Action.Name, result.Action.ID)
	printMutationValidation(result.Validation)
	return nil
}

func cmdUpdate(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("action update", flag.ContinueOnError)
	file := fs.String("file", "", "Path to replacement action.json")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 || *file == "" {
		return fmt.Errorf("usage: action update <id> --file=action.json [--json]")
	}
	id := fs.Arg(0)

	action, err := readActionFile(*file)
	if err != nil {
		return err
	}

	var result MutationResponse
	if err := ctx.Put(actionPath(id), action, &result); err != nil {
		return fmt.Errorf("failed to update action: %w", err)
	}
	if *jsonOut {
		return writeJSON(result)
	}
	if result.Action == nil {
		return fmt.Errorf("update response did not include an action")
	}
	fmt.Printf("Updated action: %s [%s]\n", result.Action.Name, result.Action.ID)
	printMutationValidation(result.Validation)
	return nil
}

func cmdDelete(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("action delete", flag.ContinueOnError)
	yes := fs.Bool("yes", false, "Skip confirmation prompt")
	force := fs.Bool("force", false, "Alias for --yes")
	hard := fs.Bool("hard", false, "Hard-delete instead of archiving")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: action delete <id> [--yes] [--hard]")
	}
	id := fs.Arg(0)

	if !*yes && !*force {
		verb := "Archive"
		if *hard {
			verb = "Hard-delete"
		}
		fmt.Printf("%s action %q? [y/N]: ", verb, id)
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	path := actionPath(id)
	if *hard {
		path += "?hard=true"
	}
	if err := ctx.Delete(path); err != nil {
		return fmt.Errorf("failed to delete action: %w", err)
	}
	if *hard {
		fmt.Printf("Deleted action: %s\n", id)
	} else {
		fmt.Printf("Archived action: %s\n", id)
	}
	return nil
}

func cmdValidate(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("action validate", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: action validate <id> [--json]")
	}

	var result ValidationResponse
	if err := ctx.Post(actionPath(fs.Arg(0))+"/validate", map[string]any{}, &result); err != nil {
		return fmt.Errorf("failed to validate action: %w", err)
	}
	if *jsonOut {
		return writeJSON(result)
	}
	printValidation(result)
	if !result.Valid {
		return fmt.Errorf("action %s is invalid", result.ActionID)
	}
	return nil
}

func cmdRun(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("action run", flag.ContinueOnError)
	input := fs.String("input", "", "JSON object containing Action input values")
	inputFile := fs.String("input-file", "", "Path to a JSON object containing Action input values")
	dryRun := fs.Bool("dry-run", false, "Validate and render argv without starting the process")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: action run <id> [--input='{\"key\":\"value\"}'|--input-file=payload.json] [--dry-run] [--json]")
	}
	if *input != "" && *inputFile != "" {
		return fmt.Errorf("use either --input or --input-file, not both")
	}

	values, err := readRunInput(*input, *inputFile)
	if err != nil {
		return err
	}

	req := RunRequest{
		Input:  values,
		DryRun: *dryRun,
	}
	var result RunResponse
	if err := ctx.Post(actionPath(fs.Arg(0))+"/run", req, &result); err != nil {
		return fmt.Errorf("failed to run action: %w", err)
	}
	if *jsonOut {
		if err := writeJSON(result); err != nil {
			return err
		}
	} else {
		printRun(result)
	}
	if result.Status != "completed" && result.Status != "dry-run" {
		if result.Error != "" {
			return fmt.Errorf("action %s %s: %s", result.ActionID, result.Status, result.Error)
		}
		return fmt.Errorf("action %s %s", result.ActionID, result.Status)
	}
	return nil
}

func readActionFile(path string) (Action, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Action{}, fmt.Errorf("failed to read action file: %w", err)
	}
	var action Action
	if err := json.Unmarshal(raw, &action); err != nil {
		return Action{}, fmt.Errorf("failed to parse action file: %w", err)
	}
	if strings.TrimSpace(action.ID) == "" {
		return Action{}, fmt.Errorf("action file is missing id")
	}
	return action, nil
}

func readRunInput(inline, file string) (map[string]any, error) {
	switch {
	case inline != "":
		return parseRunInput([]byte(inline), "--input")
	case file != "":
		raw, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read input file: %w", err)
		}
		return parseRunInput(raw, "--input-file")
	default:
		return map[string]any{}, nil
	}
}

func parseRunInput(raw []byte, source string) (map[string]any, error) {
	var values map[string]any
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("failed to parse %s JSON: %w", source, err)
	}
	if values == nil {
		return nil, fmt.Errorf("%s must be a JSON object", source)
	}
	return values, nil
}

func addQuery(query url.Values, key, value string) {
	if strings.TrimSpace(value) != "" {
		query.Set(key, value)
	}
}

func actionPath(id string) string {
	return "/actions/" + url.PathEscape(id)
}

func writeJSON(value any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}

func printAction(action Action) {
	fmt.Printf("Name: %s\n", action.Name)
	fmt.Printf("ID: %s\n", action.ID)
	fmt.Printf("Status: %s\n", action.Status)
	if action.Pack != "" {
		fmt.Printf("Pack: %s\n", action.Pack)
	}
	fmt.Printf("Owner: %s:%s\n", action.Owner.Type, action.Owner.ID)
	if action.Description != "" {
		fmt.Printf("Description: %s\n", action.Description)
	}
	if len(action.Tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(action.Tags, ", "))
	}
	if len(action.Command.Argv) > 0 {
		fmt.Printf("Command: %s\n", strings.Join(action.Command.Argv, " "))
	}
	printNamedKeys("Inputs", action.Inputs)
	printNamedKeys("Outputs", action.Outputs)
	if permissions := permissionNames(action.Permissions); len(permissions) > 0 {
		fmt.Printf("Permissions: %s\n", strings.Join(permissions, ", "))
	}
	if action.UpdatedAt != "" {
		fmt.Printf("Updated: %s\n", action.UpdatedAt)
	}
}

func printValidation(result ValidationResponse) {
	status := "invalid"
	if result.Valid {
		status = "valid"
	}
	fmt.Printf("Action: %s\n", result.ActionID)
	fmt.Printf("Status: %s\n", status)
	fmt.Printf("Runnable: %v\n", result.Runnable)
	if result.Unvalidated {
		fmt.Println("Unvalidated: owning scenario has not declared cli/manifest.json governance; action runs without cataloged safety properties")
	}
	if result.RequiresConfirmation {
		fmt.Println("Requires confirmation: command governance flags this action as needing operator confirmation before invocation")
	}
	if result.Command != nil {
		fmt.Printf("Command certainty: %s\n", result.Command.Certainty)
		if result.Command.Target != "" {
			fmt.Printf("Command target: %s\n", result.Command.Target)
		}
		if result.Command.Effect != "" {
			fmt.Printf("Command effect: %s\n", result.Command.Effect)
		}
	}
	if len(result.Checks) > 0 {
		fmt.Println("Checks:")
		for _, check := range result.Checks {
			path := ""
			if check.Path != "" {
				path = " (" + check.Path + ")"
			}
			fmt.Printf("  %s: %s%s - %s\n", check.Status, check.Code, path, check.Message)
		}
	}
}

func printMutationValidation(result ValidationResponse) {
	status := "invalid"
	if result.Valid {
		status = "valid"
	}
	runnable := ""
	if result.Runnable {
		runnable = ", runnable"
	}
	unvalidated := ""
	if result.Unvalidated {
		unvalidated = ", unvalidated"
	}
	confirm := ""
	if result.RequiresConfirmation {
		confirm = ", requires-confirmation"
	}
	fmt.Printf("Validation: %s%s%s%s\n", status, runnable, unvalidated, confirm)
}

func printRun(result RunResponse) {
	fmt.Printf("Action: %s\n", result.ActionID)
	fmt.Printf("Status: %s\n", result.Status)
	if result.ExitCode != nil {
		fmt.Printf("Exit code: %d\n", *result.ExitCode)
	}
	fmt.Printf("Duration: %dms\n", result.DurationMs)
	if len(result.Argv) > 0 {
		fmt.Printf("Argv: %s\n", strings.Join(result.Argv, " "))
	}
	if result.Stdout != "" {
		suffix := ""
		if result.StdoutTruncated {
			suffix = " (truncated)"
		}
		fmt.Printf("Stdout%s:\n%s\n", suffix, result.Stdout)
	}
	if result.Stderr != "" {
		suffix := ""
		if result.StderrTruncated {
			suffix = " (truncated)"
		}
		fmt.Printf("Stderr%s:\n%s\n", suffix, result.Stderr)
	}
	if len(result.Output) > 0 {
		fmt.Println("Output:")
		raw, err := json.MarshalIndent(result.Output, "  ", "  ")
		if err == nil {
			fmt.Println("  " + string(raw))
		}
	}
	if result.Error != "" {
		fmt.Printf("Error: %s\n", result.Error)
	}
}

func printNamedKeys[T any](label string, values map[string]T) {
	if len(values) == 0 {
		return
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fmt.Printf("%s: %s\n", label, strings.Join(keys, ", "))
}

func permissionNames(perms ActionPermissions) []string {
	names := make([]string, 0, 12)
	if perms.APIRead {
		names = append(names, "apiRead")
	}
	if perms.APIWrite {
		names = append(names, "apiWrite")
	}
	if perms.FilesystemRead {
		names = append(names, "filesystemRead")
	}
	if perms.FilesystemWrite {
		names = append(names, "filesystemWrite")
	}
	if perms.LocalhostNetwork {
		names = append(names, "localhostNetwork")
	}
	if perms.ExternalNetwork {
		names = append(names, "externalNetwork")
	}
	if perms.ProcessStart {
		names = append(names, "processStart")
	}
	if perms.ProcessStop {
		names = append(names, "processStop")
	}
	if perms.HostConfigure {
		names = append(names, "hostConfigure")
	}
	if perms.SecretRead {
		names = append(names, "secretRead")
	}
	if perms.SecretWrite {
		names = append(names, "secretWrite")
	}
	if perms.Destructive {
		names = append(names, "destructive")
	}
	return names
}
