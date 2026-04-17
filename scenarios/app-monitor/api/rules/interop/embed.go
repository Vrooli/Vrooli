package interop

import (
	"app-monitor-api/rules"
	"embed"
)

//go:embed *.go
var ruleFiles embed.FS

func init() {
	rules.RegisterEmbedFS("interop", ruleFiles)
}
