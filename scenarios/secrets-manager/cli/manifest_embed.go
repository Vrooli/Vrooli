package main

import _ "embed"

// manifestBytes is embedded so the runtime CLI and CLI-health use the same
// command contract without relying on the checkout being present at runtime.
//
//go:embed manifest.json
var manifestBytes []byte
