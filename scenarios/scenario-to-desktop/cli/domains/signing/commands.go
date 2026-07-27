package signing

import (
	"fmt"
	"scenario-to-desktop/cli/internal/support"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"
)

type Commands struct{ rpc signingRPC }

func New(deps support.Dependencies) *Commands {
	return &Commands{rpc: newSigningRPC(deps.ScenarioApp())}
}

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	c := New(deps)
	return cliapp.SubcommandGroup{Name: "signing", Description: "Code signing configuration", NeedsAPI: true, Subcommands: []cliapp.Command{
		(cliapp.Command{Name: "get", Description: "Get signing config", Args: scenarioSchema("scenario")}).WithPrimitive(c.getPrimitive()),
		(cliapp.Command{Name: "set", Description: "Set signing config from canonical Proto JSON", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true}}, Flags: []cliapp.Flag{{Name: "config", Required: true}}}}).WithPrimitive(c.setPrimitive()),
		(cliapp.Command{Name: "delete", Description: "Delete signing config", Args: scenarioSchema("scenario")}).WithPrimitive(c.deletePrimitive()),
		(cliapp.Command{Name: "validate", Description: "Validate signing config", Args: scenarioSchema("scenario")}).WithPrimitive(c.validatePrimitive()),
		(cliapp.Command{Name: "ready", Description: "Check signing readiness", Args: scenarioSchema("scenario")}).WithPrimitive(c.readyPrimitive()),
		(cliapp.Command{Name: "prerequisites", Description: "List signing tools"}).WithPrimitive(c.prerequisitesPrimitive()),
		(cliapp.Command{Name: "discover", Description: "Discover certificates", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "platform", Required: true}}}}).WithPrimitive(c.discoverPrimitive()),
		(cliapp.Command{Name: "generate-key", Description: "Generate Linux GPG key", Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true}}, Flags: []cliapp.Flag{{Name: "name", Required: true}, {Name: "email", Required: true}, {Name: "passphrase-env"}, {Name: "force", Bool: true}}}}).WithPrimitive(c.generateKeyPrimitive()),
	}}
}

func scenarioSchema(name string) cliapp.ArgSchema {
	return cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: name, Required: true}}}
}

func signingPlatform(value string) (sharedv1.Platform, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "win", "windows":
		return sharedv1.Platform_PLATFORM_WIN, nil
	case "mac", "macos", "darwin":
		return sharedv1.Platform_PLATFORM_MAC, nil
	case "linux":
		return sharedv1.Platform_PLATFORM_LINUX, nil
	default:
		return sharedv1.Platform_PLATFORM_UNSPECIFIED, fmt.Errorf("unsupported platform %q (expected windows, macos, or linux)", value)
	}
}
