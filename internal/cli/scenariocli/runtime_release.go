package scenariocli

import (
	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"strings"
)

// RuntimeReleaseRequest never selects or launches a runtime.
type RuntimeReleaseRequest struct {
	Name, CustomPath, ReleaseID string
	JSON                        bool
}

func runtimeReleaseSpec(id CommandID) commandtree.Spec[CommandID] {
	spec := commandtree.Spec[CommandID]{Name: string(id), Handler: id, Suggestable: true, Group: "Lifecycle and Utility Commands", RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}, Args: commandtree.ArgSchema{Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true}}, Options: []commandtree.OptionArg{commandtree.JSONOption()}}}
	if id == CommandCaptureRuntime {
		spec.Summary = "Retain the declared runtime closure without starting or stopping anything (not a health proof)"
		spec.Args.Options = append(spec.Args.Options, commandtree.OptionArg{Name: "--path", ValueName: "path"})
	} else {
		spec.Summary = "Verify an exact retained runtime's manifest, files, and interpreters without launching it"
		spec.Args.Positionals = append(spec.Args.Positionals, commandtree.PositionalArg{Name: "release ID", Required: true})
	}
	return spec
}

func ParseRuntimeReleaseRequest(id CommandID, globalsJSON bool, args []string) (RuntimeReleaseRequest, error) {
	parsed, err := commandtree.ParseArgs("scenario "+string(id), commandHelpText(id), commandSpec(id).Args, args)
	if err != nil {
		return RuntimeReleaseRequest{}, err
	}
	name, err := resolveAddressArg(string(id), parsed.Positionals[0], "", "")
	if err != nil {
		return RuntimeReleaseRequest{}, err
	}
	if strings.Contains(name, "/") {
		return RuntimeReleaseRequest{}, clipolicy.UsageErrorf("scenario "+string(id), "retained runtime commands support local scenario instances only")
	}
	req := RuntimeReleaseRequest{Name: name, CustomPath: parsed.FlagValue("--path"), JSON: globalsJSON || parsed.HasFlag("--json")}
	if id == CommandVerifyRuntime {
		req.ReleaseID = parsed.Positionals[1]
		if strings.TrimSpace(req.ReleaseID) == "" || req.ReleaseID == "source" {
			return RuntimeReleaseRequest{}, clipolicy.UsageErrorf("scenario "+string(id), "verification requires an exact retained release ID, not source")
		}
	}
	return req, nil
}
