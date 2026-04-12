package runtime

func newGoTool() handler {
	return newToolHandler("go", []string{"go"}, []string{"version"}, "golang-go", map[string]string{
		"brew": "go",
	}, "Install Go 1.22+")
}
