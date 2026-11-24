//go:build !audio_lab

package bootstrap

import "audio-tools/internal/server"

type labRuntimeDeps struct{}

func labBuildEnabled() bool { return false }

func composeAPI(deps server.CompositionDeps, _ *labRuntimeDeps) *server.Server {
	return server.ComposeForBuild(deps)
}

func closeExperimentSubsystem(experimentSubsystem) {}
