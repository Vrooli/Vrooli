package runtime

func newXvfbTool() handler {
	return newToolHandler("Xvfb", []string{"Xvfb"}, []string{"-version"}, "xvfb", map[string]string{
		"apt-get": "xvfb",
	}, "Install Xvfb to provide a virtual Linux display for desktop smoke tests")
}
