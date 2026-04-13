package packagecli

import (
	"fmt"
	"io"
	"strings"

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
		{Name: string(CommandList), Summary: "List governed packages", Group: "Package Governance", Handler: CommandList},
		{Name: string(CommandInfo), Summary: "Show package manifest metadata", Group: "Package Governance", Handler: CommandInfo},
		{Name: string(CommandDependents), Summary: "List package consumers", Group: "Package Governance", Handler: CommandDependents},
		{Name: string(CommandValidate), Summary: "Validate package manifests and package adoption policy", Group: "Package Governance", Handler: CommandValidate},
		{Name: string(CommandBuild), Summary: "Run the package build lifecycle", Group: "Package Governance", Handler: CommandBuild},
		{Name: string(CommandGenerate), Summary: "Run the package generation lifecycle", Group: "Package Governance", Handler: CommandGenerate},
		{Name: string(CommandRefresh), Summary: "Rebuild/regenerate a package and propagate to affected consumers", Group: "Package Governance", Handler: CommandRefresh},
		{Name: string(CommandAudit), Summary: "Report governance drift and unsupported package adoption", Group: "Package Governance", Handler: CommandAudit},
	}
}

func RenderCommandHelp(w io.Writer) {
	commandtree.RenderHelp(w, commandtree.Help{
		Title:        "Vrooli Package Commands",
		Usage:        "vrooli package <subcommand> [options]",
		DefaultGroup: "Package Governance",
	}, CommandSpecs())
}

func ParseListRequest(args []string) (ListRequest, error) {
	if len(args) > 0 {
		return ListRequest{}, fmt.Errorf("unknown option for package list: %s", args[0])
	}
	return ListRequest{}, nil
}

func ParseInfoRequest(args []string) (InfoRequest, error) {
	if len(args) != 1 {
		return InfoRequest{}, fmt.Errorf("package info requires exactly one package name")
	}
	return InfoRequest{Name: args[0]}, nil
}

func ParseDependentsRequest(args []string) (DependentsRequest, error) {
	if len(args) != 1 {
		return DependentsRequest{}, fmt.Errorf("package dependents requires exactly one package name")
	}
	return DependentsRequest{Name: args[0]}, nil
}

func ParseValidateRequest(args []string) (ValidateRequest, error) {
	req := ValidateRequest{}
	for _, arg := range args {
		switch arg {
		case "--all":
			req.All = true
		default:
			if strings.HasPrefix(arg, "-") {
				return ValidateRequest{}, fmt.Errorf("unknown option for package validate: %s", arg)
			}
			if req.Name != "" {
				return ValidateRequest{}, fmt.Errorf("package validate accepts at most one package name")
			}
			req.Name = arg
		}
	}
	if !req.All && req.Name == "" {
		req.All = true
	}
	return req, nil
}

func ParseRunRequest(action string, args []string) (RunRequest, error) {
	if len(args) != 1 {
		return RunRequest{}, fmt.Errorf("package %s requires exactly one package name", action)
	}
	return RunRequest{Name: args[0], Action: action}, nil
}

func ParseRefreshRequest(args []string) (RefreshRequest, error) {
	req := RefreshRequest{Target: "all"}
	for _, arg := range args {
		switch arg {
		case "--no-restart":
			req.NoRestart = true
		default:
			if strings.HasPrefix(arg, "-") {
				return RefreshRequest{}, fmt.Errorf("unknown option for package refresh: %s", arg)
			}
			if req.Name == "" {
				req.Name = arg
				continue
			}
			if req.Target == "all" {
				req.Target = arg
				continue
			}
			return RefreshRequest{}, fmt.Errorf("package refresh accepts at most a package name and one target scenario")
		}
	}
	if req.Name == "" {
		return RefreshRequest{}, fmt.Errorf("package refresh requires a package name")
	}
	return req, nil
}

func ParseAuditRequest(args []string) (AuditRequest, error) {
	req := AuditRequest{}
	for _, arg := range args {
		switch arg {
		case "--all":
			req.All = true
		default:
			if strings.HasPrefix(arg, "-") {
				return AuditRequest{}, fmt.Errorf("unknown option for package audit: %s", arg)
			}
			if req.Name != "" {
				return AuditRequest{}, fmt.Errorf("package audit accepts at most one package name")
			}
			req.Name = arg
		}
	}
	if req.Name == "" {
		req.All = true
	}
	return req, nil
}
