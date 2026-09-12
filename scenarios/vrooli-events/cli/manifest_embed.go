package main

import _ "embed"

// manifestBytes is embedded so command registration is independent of the
// process working directory. The same bytes feed runtime registration and the
// committed primitive-evidence test.
//
//go:embed manifest.json
var manifestBytes []byte
