package packagecli

import (
	"io"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/packagegov"
)

type CommandID string

const (
	CommandList       CommandID = "list"
	CommandInfo       CommandID = "info"
	CommandDependents CommandID = "dependents"
	CommandValidate   CommandID = "validate"
	CommandBuild      CommandID = "build"
	CommandGenerate   CommandID = "generate"
	CommandTest       CommandID = "test"
	CommandRefresh    CommandID = "refresh"
	CommandAudit      CommandID = "audit"
)

type (
	ListRequest       struct{}
	InfoRequest       struct{ Name string }
	DependentsRequest struct{ Name string }
	ValidateRequest   struct {
		Name string
		All  bool
	}
)

type RunRequest struct {
	Name   string
	Action string
}
type RefreshRequest struct {
	Name      string
	Target    string
	NoRestart bool
}
type AuditRequest struct {
	Name string
	All  bool
}

type ListResponse struct {
	Packages []packagegov.Package `json:"packages"`
}

type DependentsResponse struct {
	PackageName string                       `json:"package_name"`
	Dependents  []packagegov.Dependent       `json:"dependents"`
	Issues      []packagegov.ValidationIssue `json:"issues,omitempty"`
}

type ValidateResponse struct {
	Report packagegov.ValidationReport `json:"report"`
}

type RunResponse struct {
	PackageName string `json:"package_name"`
	Action      string `json:"action"`
}

type RefreshItem struct {
	Consumer string                       `json:"consumer"`
	Class    packagegov.ConsumerClass     `json:"consumer_class"`
	Classes  []packagegov.ConsumerClass   `json:"consumer_classes,omitempty"`
	Action   packagegov.RefreshActionKind `json:"action"`
	Status   string                       `json:"status"`
}

type RefreshResponse struct {
	PackageName string        `json:"package_name"`
	Items       []RefreshItem `json:"items"`
}

type AuditResponse struct {
	Report packagegov.AuditReport `json:"report"`
}

func CommandSpecs() []commandtree.Spec[CommandID] {
	return []commandtree.Spec[CommandID]{
		{Name: string(CommandList), Summary: "List governed packages", Group: "Package Governance", Args: jsonOnlyArgs(), Handler: CommandList},
		{
			Name:    string(CommandInfo),
			Summary: "Show package manifest metadata",
			Group:   "Package Governance",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "package", Required: true}},
				Options:     []commandtree.OptionArg{commandtree.JSONOption()},
			},
			Handler: CommandInfo,
		},
		{
			Name:    string(CommandDependents),
			Summary: "List package consumers",
			Group:   "Package Governance",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "package", Required: true}},
				Options:     []commandtree.OptionArg{commandtree.JSONOption()},
			},
			Handler: CommandDependents,
		},
		{
			Name:    string(CommandValidate),
			Summary: "Validate package manifests and package adoption policy",
			Group:   "Package Governance",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "package"}},
				Options: []commandtree.OptionArg{
					{Name: "--all", Description: "Validate every governed package"},
					commandtree.JSONOption(),
				},
			},
			Handler: CommandValidate,
		},
		{
			Name:    string(CommandBuild),
			Summary: "Run the package build lifecycle",
			Group:   "Package Governance",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "package", Required: true}},
				Options:     []commandtree.OptionArg{commandtree.JSONOption()},
			},
			Handler: CommandBuild,
		},
		{
			Name:    string(CommandGenerate),
			Summary: "Run the package generation lifecycle",
			Group:   "Package Governance",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "package", Required: true}},
				Options:     []commandtree.OptionArg{commandtree.JSONOption()},
			},
			Handler: CommandGenerate,
		},
		{
			Name:    string(CommandTest),
			Summary: "Run the package test lifecycle",
			Group:   "Package Governance",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "package"}},
				Options:     []commandtree.OptionArg{commandtree.JSONOption()},
			},
			Handler: CommandTest,
		},
		{
			Name:    string(CommandRefresh),
			Summary: "Rebuild/regenerate a package and propagate to affected consumers",
			Group:   "Package Governance",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{
					{Name: "package", Required: true},
					{Name: "target"},
				},
				Options: []commandtree.OptionArg{
					{Name: "--no-restart", Description: "Do not restart affected consumers after refresh"},
					commandtree.JSONOption(),
				},
			},
			Handler: CommandRefresh,
		},
		{
			Name:    string(CommandAudit),
			Summary: "Report governance drift and unsupported package adoption",
			Group:   "Package Governance",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "package"}},
				Options: []commandtree.OptionArg{
					{Name: "--all", Description: "Audit every governed package"},
					commandtree.JSONOption(),
				},
			},
			Handler: CommandAudit,
		},
	}
}

func RenderCommandHelp(w io.Writer) {
	commandtree.RenderHelp(w, commandtree.Help{
		Title:        "Vrooli Package Commands",
		Usage:        "vrooli package <subcommand> [options]",
		DefaultGroup: "Package Governance",
	}, CommandSpecs())
}

func jsonOnlyArgs() commandtree.ArgSchema {
	return commandtree.ArgSchema{
		Options: []commandtree.OptionArg{commandtree.JSONOption()},
	}
}

func ParseListRequest(args []string) (ListRequest, error) {
	if _, err := commandtree.ParseArgs("package list", commandHelpText(CommandList), commandtree.ArgSchema{}, args); err != nil {
		return ListRequest{}, err
	}
	return ListRequest{}, nil
}

func ParseInfoRequest(args []string) (InfoRequest, error) {
	parsed, err := commandtree.ParseArgs("package info", commandHelpText(CommandInfo), commandSpec(CommandInfo).Args, args)
	if err != nil {
		return InfoRequest{}, err
	}
	return InfoRequest{Name: parsed.Positionals[0]}, nil
}

func ParseDependentsRequest(args []string) (DependentsRequest, error) {
	parsed, err := commandtree.ParseArgs("package dependents", commandHelpText(CommandDependents), commandSpec(CommandDependents).Args, args)
	if err != nil {
		return DependentsRequest{}, err
	}
	return DependentsRequest{Name: parsed.Positionals[0]}, nil
}

func ParseValidateRequest(args []string) (ValidateRequest, error) {
	parsed, err := commandtree.ParseArgs("package validate", commandHelpText(CommandValidate), commandSpec(CommandValidate).Args, args)
	if err != nil {
		return ValidateRequest{}, err
	}
	req := ValidateRequest{All: parsed.HasFlag("--all")}
	if len(parsed.Positionals) == 1 {
		req.Name = parsed.Positionals[0]
	}
	if !req.All && req.Name == "" {
		req.All = true
	}
	return req, nil
}

func ParseRunRequest(action string, args []string) (RunRequest, error) {
	commandID := CommandBuild
	if action == string(CommandGenerate) {
		commandID = CommandGenerate
	} else if action == string(CommandTest) {
		commandID = CommandTest
	}
	command := "package " + action
	parsed, err := commandtree.ParseArgs(command, commandHelpText(commandID), commandSpec(commandID).Args, args)
	if err != nil {
		return RunRequest{}, err
	}
	name := ""
	if len(parsed.Positionals) == 1 {
		name = parsed.Positionals[0]
	}
	return RunRequest{Name: name, Action: action}, nil
}

func ParseRefreshRequest(args []string) (RefreshRequest, error) {
	parsed, err := commandtree.ParseArgs("package refresh", commandHelpText(CommandRefresh), commandSpec(CommandRefresh).Args, args)
	if err != nil {
		return RefreshRequest{}, err
	}
	req := RefreshRequest{
		Name:      parsed.Positionals[0],
		Target:    "all",
		NoRestart: parsed.HasFlag("--no-restart"),
	}
	if len(parsed.Positionals) > 1 {
		req.Target = parsed.Positionals[1]
	}
	return req, nil
}

func ParseAuditRequest(args []string) (AuditRequest, error) {
	parsed, err := commandtree.ParseArgs("package audit", commandHelpText(CommandAudit), commandSpec(CommandAudit).Args, args)
	if err != nil {
		return AuditRequest{}, err
	}
	req := AuditRequest{All: parsed.HasFlag("--all")}
	if len(parsed.Positionals) == 1 {
		req.Name = parsed.Positionals[0]
	}
	if req.Name == "" {
		req.All = true
	}
	return req, nil
}

func commandSpec(id CommandID) commandtree.Spec[CommandID] {
	for _, spec := range CommandSpecs() {
		if spec.Handler == id {
			return spec
		}
	}
	panic("unknown package command spec: " + string(id))
}

func commandHelpText(id CommandID) string {
	spec := commandSpec(id)
	return commandtree.SpecHelpText("", "vrooli package "+spec.Name, spec)
}
