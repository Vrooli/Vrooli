package safeguards

import "embed"

//go:embed */safeguard.json
var Manifests embed.FS
