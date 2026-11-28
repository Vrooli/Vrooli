//go:build audio_lab

package bootstrap

import "audio-tools/internal/server"

type labRuntimeDeps = server.LabRuntimeDeps

func labBuildEnabled() bool { return true }

func composeAPI(deps server.CompositionDeps, runtime *labRuntimeDeps) *server.Server {
	if runtime == nil {
		return server.Compose(deps)
	}
	return server.ComposeWithLab(deps, *runtime)
}

func closeExperimentSubsystem(subsystem experimentSubsystem) {
	if subsystem.labRuntime != nil && subsystem.labRuntime.ExperimentManager != nil {
		subsystem.labRuntime.ExperimentManager.Close()
	}
}
