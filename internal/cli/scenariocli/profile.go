package scenariocli

import (
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/runtimeprofile"
)

type ProfileRequest struct {
	runtimeprofile.Request
	JSON bool
}

func ParseProfileRequest(globalsJSON bool, args []string) (ProfileRequest, error) {
	parsed, err := commandtree.ParseArgs("scenario profile", commandHelpText(CommandProfile), commandSpec(CommandProfile).Args, args)
	if err != nil {
		return ProfileRequest{}, err
	}
	name, err := resolveAddressArg("profile", parsed.Positionals[0], parsed.FlagValue("--instance"), parsed.FlagValue("--node"))
	if err != nil {
		return ProfileRequest{}, err
	}
	if strings.Contains(name, "/") {
		return ProfileRequest{}, clipolicy.UsageErrorf("scenario profile", "CPU profiling currently supports local scenario instances only")
	}
	seconds := 10
	if value := parsed.FlagValue("--seconds"); value != "" {
		seconds, err = strconv.Atoi(value)
		if err != nil || seconds < 1 || seconds > 30 {
			return ProfileRequest{}, clipolicy.UsageErrorf("scenario profile", "--seconds must be an integer between 1 and 30")
		}
	}
	step := parsed.FlagValue("--step")
	if step == "" {
		step = "start-api"
	}
	output := parsed.FlagValue("--out")
	if output == "" {
		return ProfileRequest{}, clipolicy.UsageErrorf("scenario profile", "--out is required and must name a new file")
	}
	return ProfileRequest{Request: runtimeprofile.Request{Scenario: name, Step: step, Seconds: seconds, Output: output}, JSON: globalsJSON || parsed.HasFlag("--json")}, nil
}
