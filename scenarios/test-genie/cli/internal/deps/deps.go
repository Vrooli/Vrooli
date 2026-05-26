package deps

import (
	"test-genie/cli/execute"
	"test-genie/cli/generate"
	"test-genie/cli/playbooksseed"
	"test-genie/cli/runlocal"
	"test-genie/cli/status"
	"test-genie/cli/uismoke"

	"github.com/vrooli/cli-core/cliutil"
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
	APIClient  *cliutil.APIClient
}
