package model

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"

	testutil "audio-tools/cli/internal/testutil"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/models"
)

// TestModelManifestCoversService proves every RPC on ModelRegistryService is
// either bound to a command or explicitly omitted in cli/manifest.json, and
// that the manifest passes schema validation.
func TestModelManifestCoversService(t *testing.T) {
	cliapp.RequireProtoServiceCoverage(t, testutil.ReadManifest(t), modelsv1.File_audio_tools_v1_models_models_proto, "ModelRegistryService")
}

// TestModelGroupBindsEveryCommand proves the manifest group and the domain's
// handler registry cannot drift: every command resolves to a registered
// handler, every registered handler is used, and both model RPCs are bound.
func TestModelGroupBindsEveryCommand(t *testing.T) {
	raw := testutil.ReadManifest(t)
	manifest, err := cliapp.ParseManifest(raw)
	require.NoError(t, err)

	group := manifest.FindGroup(GroupName)
	require.NotNil(t, group, "manifest group %q must exist", GroupName)

	registered := modelBindings(newHandlers(&cliapp.ScenarioApp{}))
	used := make(map[string]bool, len(group.Commands))
	for _, command := range group.Commands {
		require.Equal(t, "connect-rpc", command.Binding.Kind, "command %s binding kind", command.Name)
		require.Equal(t, "ModelRegistryService", command.Binding.Service, "command %s binding service", command.Name)

		key := command.Binding.BindingKey()
		handler, ok := registered[key]
		require.Truef(t, ok, "command %s/%s binding %s has no registered handler", GroupName, command.Name, key)
		require.NotNilf(t, handler, "command %s/%s binding %s registered a nil handler", GroupName, command.Name, key)
		used[key] = true
	}

	for key := range registered {
		require.Truef(t, used[key], "registered handler %s has no manifest command", key)
	}
	for _, method := range []string{"ListModels", "ResolveModel"} {
		require.Truef(t, used["ModelRegistryService."+method], "ModelRegistryService.%s must be bound to a command", method)
	}
}

// TestModelGroupGovernanceAndHelpMetadata keeps the new group consistent with
// the rest of the CLI: read-only, run-eligible, internal-network only, and
// every command/flag carries help text.
func TestModelGroupGovernanceAndHelpMetadata(t *testing.T) {
	raw := testutil.ReadManifest(t)
	manifest, err := cliapp.ParseManifest(raw)
	require.NoError(t, err)

	group := manifest.FindGroup(GroupName)
	require.NotNil(t, group)
	require.NotEmpty(t, group.Description)

	for _, command := range group.Commands {
		require.NotEmpty(t, command.Description, "command %s must have help text", command.Name)
		require.Equal(t, "read", command.Governance.Effect, "command %s effect", command.Name)
		require.True(t, command.Governance.RunEligible, "command %s run_eligible", command.Name)
		require.Contains(t, command.Governance.Permissions, "network:internal", "command %s permissions", command.Name)

		for _, flag := range command.Flags {
			require.NotEmpty(t, flag.Description, "command %s flag --%s must have help text", command.Name, flag.Name)
		}

		_, err := cliapp.ManifestArgs(command)
		require.NoErrorf(t, err, "command %s must produce a valid ArgSchema", command.Name)
	}

	registered, err := Register(&cliapp.ScenarioApp{}, raw)
	require.NoError(t, err)
	require.Equal(t, GroupName, registered.Name)
	require.Len(t, registered.Subcommands, len(group.Commands))
}
