package runtime

func newYQTool() handler {
	return newToolHandler("yq", []string{"yq"}, []string{"--version"}, "yq", map[string]string{
		"apt-get": "yq",
		"brew":    "yq",
	}, "Install yq for YAML-aware shell workflows")
}
