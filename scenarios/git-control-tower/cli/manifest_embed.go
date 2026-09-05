package main

import _ "embed"

// manifestBytes carries cli/manifest.json baked into the binary so the
// runtime CLI has no filesystem dependency on the manifest file. The
// manifest covers only the Connect-RPC-aligned `worktree` domain today;
// the remaining REST-backed domains (repo/branch/review/audit) are not
// modelled because cli-manifest/v1 only supports binding.kind=connect-rpc.
//
//go:embed manifest.json
var manifestBytes []byte
