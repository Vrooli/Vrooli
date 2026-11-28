package tools

import "embed"

//go:embed */tool.json */versions.json
var Manifests embed.FS
