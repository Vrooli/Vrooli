package main

import (
	"os"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/cli/topcli"
)

var (
	topLevelCommandTable = buildTopLevelCommandTable()
	scenarioCommandTable = buildScenarioCommandTable()
)

var (
	topLevelCommands           = commandtree.BuildHandlerMap(topLevelCommandTable)
	scenarioCommands           = commandtree.BuildHandlerMap(scenarioCommandTable)
	topLevelCommandDescriptors = commandtree.BuildSpecMap(topLevelCommandTable)
	scenarioCommandDescriptors = commandtree.BuildSpecMap(scenarioCommandTable)
)

func helpOnlyWithoutRoot(args []string) bool {
	return wantsCommandHelp(args)
}

func listOrHelpWithoutRoot(args []string) bool {
	return len(args) == 0 || wantsCommandHelp(args)
}

func scenarioCanRunWithoutRoot(args []string) bool {
	if len(args) == 0 || wantsCommandHelp(args) {
		return true
	}
	descriptor, ok := scenarioCommandDescriptors[commandtree.NormalizeName(args[0])]
	if !ok {
		return true
	}
	if !descriptor.RootPolicy.RequiresRoot {
		return true
	}
	if descriptor.RootPolicy.CanRunWithoutRoot == nil {
		return false
	}
	return descriptor.RootPolicy.CanRunWithoutRoot(args[1:])
}

func topLevelCommandNames() []string {
	return commandtree.SuggestableNames(topLevelCommandTable)
}

func scenarioCommandNames() []string {
	return commandtree.SuggestableNames(scenarioCommandTable)
}

func groupedTopLevelCommands() []commandtree.Entry {
	return commandtree.VisibleEntries(topLevelCommandTable, "")
}

func groupedScenarioCommands() []commandtree.Entry {
	return commandtree.VisibleEntries(scenarioCommandTable, "")
}

func buildScenarioCommandTable() []commandtree.Spec[appCommandHandler] {
	handlerMap := map[scenariocli.CommandID]appCommandHandler{
		scenariocli.CommandList: bindGlobalCommand(
			func(globals globalOptions, args []string) (scenariocli.ListRequest, error) {
				return scenariocli.ParseListRequest(globals.json, args)
			},
			runScenarioListRequest,
			renderScenarioListResponse,
		),
		scenariocli.CommandInfo: bindGlobalCommand(
			func(globals globalOptions, args []string) (scenariocli.InfoRequest, error) {
				return scenariocli.ParseInfoRequest(globals.json, args)
			},
			runScenarioInfoRequest,
			renderScenarioInfoResponse,
		),
		scenariocli.CommandStatus: bindGlobalCommand(
			func(globals globalOptions, args []string) (scenariocli.StatusRequest, error) {
				return scenariocli.ParseStatusRequest(globals.json, args)
			},
			runScenarioStatusRequest,
			renderScenarioStatusResponse,
		),
		scenariocli.CommandValidateEnv: bindGlobalCommand(
			func(globals globalOptions, args []string) (scenariocli.ValidateEnvRequest, error) {
				return scenariocli.ParseValidateEnvRequest(globals.json, args)
			},
			runScenarioValidateEnvRequest,
			scenariocli.RenderValidateEnvResponse,
		),
		scenariocli.CommandRun: runScenarioRunCommand,
		scenariocli.CommandStart: bindGlobalCommand(
			func(globals globalOptions, args []string) (scenariocli.StartRequest, error) {
				return scenariocli.ParseStartRequest(globals.json, args)
			},
			runScenarioStartRequest,
			renderScenarioLifecycleResponse,
		),
		scenariocli.CommandStartAll: bindGlobalCommand(
			func(globals globalOptions, args []string) (scenariocli.StartAllRequest, error) {
				return scenariocli.ParseStartAllRequest(globals.json, args)
			},
			runScenarioStartAllRequest,
			renderScenarioBatchResponse,
		),
		scenariocli.CommandSetup: bindGlobalCommand(
			func(globals globalOptions, args []string) (scenariocli.SetupRequest, error) {
				return scenariocli.ParseSetupRequest(globals.json, args)
			},
			runScenarioSetupRequest,
			renderScenarioSetupResponse,
		),
		scenariocli.CommandRestart: bindGlobalCommand(
			func(globals globalOptions, args []string) (scenariocli.RestartRequest, error) {
				return scenariocli.ParseRestartRequest(globals.json, args)
			},
			runScenarioRestartRequest,
			renderScenarioLifecycleResponse,
		),
		scenariocli.CommandStop: bindGlobalCommand(
			func(globals globalOptions, args []string) (scenariocli.StopRequest, error) {
				return scenariocli.ParseStopRequest(globals.json, args)
			},
			runScenarioStopRequest,
			renderScenarioLifecycleResponse,
		),
		scenariocli.CommandStopAll: bindGlobalCommand(
			func(globals globalOptions, args []string) (scenariocli.StopAllRequest, error) {
				return scenariocli.ParseStopAllRequest(globals.json, args)
			},
			runScenarioStopAllRequest,
			renderScenarioBatchResponse,
		),
		scenariocli.CommandTest: bindGlobalCommand(
			func(globals globalOptions, args []string) (scenariocli.TestRequest, error) {
				return scenariocli.ParseTestRequest(globals.json, globals.verbose, args)
			},
			runScenarioTestRequest,
			renderScenarioTestResponse,
		),
		scenariocli.CommandLogs: runScenarioLogsCommandWithApp,
		scenariocli.CommandOpen: bindGlobalCommand(
			func(globals globalOptions, args []string) (scenariocli.OpenRequest, error) {
				return scenariocli.ParseOpenRequest(globals.json, args)
			},
			runScenarioOpenRequest,
			renderScenarioOpenResponse,
		),
		scenariocli.CommandPort: bindGlobalCommand(
			func(globals globalOptions, args []string) (scenariocli.PortRequest, error) {
				return scenariocli.ParsePortRequest(globals.json, args)
			},
			runScenarioPortRequest,
			renderScenarioPortResponse,
		),
		scenariocli.CommandUISmoke: runScenarioUISmokeCommandWithApp,
		scenariocli.CommandRequirements: bindGlobalCommand(
			func(globals globalOptions, args []string) (scenariocli.RequirementsRequest, error) {
				_ = globals
				return scenariocli.ParseRequirementsRequest(args)
			},
			runScenarioRequirementsRequest,
			scenariocli.RenderRequirementsResponse,
		),
		scenariocli.CommandTemplate:     runScenarioTemplateCommandWithApp,
		scenariocli.CommandGenerate:     runScenarioGenerateCommandWithApp,
		scenariocli.CommandCompleteness: runScenarioCompletenessCommandWithApp,
		scenariocli.CommandHealFromSandbox: bindGlobalCommand(
			func(globals globalOptions, args []string) (scenariocli.HealFromSandboxRequest, error) {
				_ = globals
				return scenariocli.ParseHealFromSandboxRequest(strings.TrimSpace(os.Getenv("SANDBOX_MERGED_DIR")), args)
			},
			runScenarioHealFromSandboxRequest,
			scenariocli.RenderHealFromSandboxResponse,
		),
	}

	source := scenariocli.CommandSpecs()
	specs := make([]commandtree.Spec[appCommandHandler], 0, len(source))
	for _, spec := range source {
		handler, ok := handlerMap[spec.Handler]
		if !ok {
			continue
		}
		specs = append(specs, commandtree.Spec[appCommandHandler]{
			Name:        spec.Name,
			Aliases:     append([]string(nil), spec.Aliases...),
			Group:       spec.Group,
			Summary:     spec.Summary,
			Hidden:      spec.Hidden,
			Suggestable: spec.Suggestable,
			RootPolicy:  spec.RootPolicy,
			Help:        spec.Help,
			Handler:     handler,
		})
	}
	return specs
}

func buildTopLevelCommandTable() []commandtree.Spec[appCommandHandler] {
	handlerMap := map[topcli.CommandID]appCommandHandler{
		topcli.CommandSetup:        runTopLevelSetupCommand,
		topcli.CommandDevelop:      runTopLevelDevelopCommand,
		topcli.CommandBuild:        runTopLevelBuildCommand,
		topcli.CommandClean:        runTopLevelCleanCommand,
		topcli.CommandStatus:       runTopLevelStatusCommand,
		topcli.CommandStop:         runTopLevelStopCommand,
		topcli.CommandBackup:       runTopLevelBackupCommand,
		topcli.CommandRestore:      runTopLevelRestoreCommand,
		topcli.CommandInfo:         runInfoTopLevelCommand,
		topcli.CommandScenario:     runScenarioRootCommand,
		topcli.CommandPackage:      runPackageRootCommand,
		topcli.CommandResource:     runTopLevelResourceCommandWithApp,
		topcli.CommandCleanup:      runTopLevelCleanupCommand,
		topcli.CommandDoctor:       runTopLevelDoctorCommand,
		topcli.CommandOrphans:      runTopLevelOrphansCommand,
		topcli.CommandLocks:        runTopLevelLocksCommand,
		topcli.CommandDiagnosePort: runTopLevelDiagnosePortCommand,
		topcli.CommandContract:     runContractCommandWithApp,
		topcli.CommandLifecycle:    runTopLevelLifecycleCommandWithApp,
	}

	source := topcli.CommandSpecs()
	specs := make([]commandtree.Spec[appCommandHandler], 0, len(source))
	for _, spec := range source {
		handler, ok := handlerMap[spec.Handler]
		if !ok {
			continue
		}
		rootPolicy := spec.RootPolicy
		if spec.Handler == topcli.CommandScenario {
			rootPolicy.CanRunWithoutRoot = scenarioCanRunWithoutRoot
		}
		specs = append(specs, commandtree.Spec[appCommandHandler]{
			Name:        spec.Name,
			Aliases:     append([]string(nil), spec.Aliases...),
			Group:       spec.Group,
			Summary:     spec.Summary,
			Hidden:      spec.Hidden,
			Suggestable: spec.Suggestable,
			RootPolicy:  rootPolicy,
			Help:        spec.Help,
			Handler:     handler,
		})
	}
	return specs
}
