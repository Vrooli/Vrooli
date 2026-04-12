package runtime

func newJQTool() handler {
	return newToolHandler("jq", []string{"jq"}, []string{"--version"}, "jq", nil, "Install jq for JSON processing in setup and resource flows")
}
