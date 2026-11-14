package main

import _ "embed"

// manifestBytes carries cli/manifest.json baked into the binary so the
// runtime CLI has no filesystem dependency on the manifest file. The same
// bytes pass through domains.SubcommandGroups → each domain's Register →
// cliapp.LoadFromManifest, where they're parsed once at NewApp time.
//
//go:embed manifest.json
var manifestBytes []byte
