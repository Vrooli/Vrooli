package interop

import (
	"embed"

	"app-monitor-api/rules"
)

//go:embed *.go
var ruleFiles embed.FS

func init() {
	rules.RegisterEmbedFS("interop", ruleFiles)
}
