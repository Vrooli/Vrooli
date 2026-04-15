package deps

import (
	"github.com/vrooli/cli-core/cliutil"

	"test-genie/cli/execute"
	"test-genie/cli/generate"
	"test-genie/cli/playbooksseed"
	"test-genie/cli/runlocal"
	"test-genie/cli/status"
	"test-genie/cli/uismoke"
)

// Runtime contains the concrete clients needed by the CLI domains.
type Runtime struct {
	Generate   *generate.Client
	Execute    *execute.Client
	RunLocal   *runlocal.Client
	UISmoke    *uismoke.Client
	Seed       *playbooksseed.Client
	Status     *status.Client
	HTTPClient *cliutil.HTTPClient
}
