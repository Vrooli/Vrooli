package main

import _ "embed"

// manifestBytes carries cli/manifest.json baked into the binary so the
// runtime CLI has no filesystem dependency on the manifest file. As the
// BAS proto+Connect migration progresses (see plans:
// bas-migration-to-proto-connect-rpc), each domain's Register function
// will pull its group out of these bytes via cliapp.LoadFromManifest.
// Hand-coded domains continue to compile independently of the manifest
// until they are migrated.
//
//go:embed manifest.json
var manifestBytes []byte

// Ensure the embed is reachable; suppress unused warnings while
// manifest-driven dispatch is still being adopted incrementally.
var _ = manifestBytes
