package cliapp_test

import (
	"os"
	"testing"

	"github.com/vrooli/cli-core/cliapptest"
)

// TestMain detaches the package from any agent session that runs it: cliapp's
// tests send requests to local test servers, which an agent session would
// otherwise redirect to live scenarios and route through the live credential
// broker.
func TestMain(m *testing.M) { os.Exit(cliapptest.RunWithoutAgentSession(m)) }
