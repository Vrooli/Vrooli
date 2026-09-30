package checks

import (
	"embed"

	"ui-health/internal/uiinterop"
)

//go:embed *.go
var ruleFiles embed.FS

func init() {
	uiinterop.RegisterEmbedFS("interop", ruleFiles)
}
