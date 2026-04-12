package runtime

func newNodeTool() handler {
	return newToolHandler("node", []string{"node"}, []string{"--version"}, "nodejs", map[string]string{
		"brew": "node",
	}, "Install Node.js 20+")
}
