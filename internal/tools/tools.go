package tools

import "embed"

//go:embed */tool.json
var Manifests embed.FS
