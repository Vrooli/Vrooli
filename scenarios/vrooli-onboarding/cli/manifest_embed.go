package main

import _ "embed"

// manifestBytes is the exact command contract shipped with this binary.
// Runtime registration consumes these bytes, so a stale external manifest
// cannot silently change the command surface.
//
//go:embed manifest.json
var manifestBytes []byte
