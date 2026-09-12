package main

import _ "embed"

// manifestBytes binds the deployed command tree to the reviewed manifest.
//
//go:embed manifest.json
var manifestBytes []byte
