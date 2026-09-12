package main

import _ "embed"

// manifestBytes carries cli/manifest.json baked into the binary so the
// runtime CLI has no filesystem dependency on the manifest file. The
// manifest covers the Connect-RPC-aligned `worktree`, `repo`, and `branch`
// domains; the remaining REST-backed domains (review/audit reads and
// operational compatibility surfaces) are not
// modelled because cli-manifest/v1 only supports binding.kind=connect-rpc.
//
//go:embed manifest.json
var manifestBytes []byte
