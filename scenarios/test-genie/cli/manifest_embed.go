package main

import _ "embed"

// manifestBytes carries cli/manifest.json baked into tests and future manifest
// registration so validation does not depend on the runtime working directory.
//
//go:embed manifest.json
var manifestBytes []byte
