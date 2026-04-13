package contractcli

const (
	ValidateHelpText        = "Usage: vrooli contract validate [--json]\n\nRuns schema validation plus in-process semantic and live drift checks."
	ShowHelpText            = "Usage: vrooli contract show [--json]"
	ResolveHelpText         = "Usage: vrooli contract resolve scenario <name> [--file <key>] [--json]"
	ResolveScenarioHelpText = "Usage: vrooli contract resolve scenario <name> [--file <key>] [--json]\n\nKnown keys: service, docs, requirements, api, ui, cli, initialization"
	MatchGlobHelpText       = "Usage: vrooli contract match-glob <pattern> <path> [--json]"
)
