package main

import _ "embed"

// manifestBytes is embedded so primitive verification uses the exact manifest
// shipped with the CLI rather than a working-tree lookup at runtime.
//
//go:embed manifest.json
var manifestBytes []byte
