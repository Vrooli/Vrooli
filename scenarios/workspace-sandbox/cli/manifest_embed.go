package main

import _ "embed"

// manifestBytes is the single CLI contract embedded into the executable.
// The generated Connect handlers keep the runtime command surface and the
// published manifest in lockstep.
//
//go:embed manifest.json
var manifestBytes []byte
