package main

import (
	"time"

	vroolicli "github.com/vrooli/vrooli-cli-go"
)

const resourceStatusProbeTimeout = 2 * time.Second

// Resource status is a live host probe. Keep a stalled control-plane sweep
// from holding the onboarding health surface for the CLI client's 30-second
// general-purpose deadline; the UI already exposes an explicit error/retry
// state when the probe cannot complete.
var cliClient = vroolicli.New(vroolicli.WithTimeout(resourceStatusProbeTimeout))
