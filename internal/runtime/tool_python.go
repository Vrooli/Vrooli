package runtime

func newPythonTool() handler {
	return newToolHandler("python", []string{"python3", "python"}, []string{"--version"}, "python3", map[string]string{
		"brew": "python",
	}, "Install Python 3.10+")
}
