package runtime

func newX11VNCTool() handler {
	return newToolHandler("x11vnc", []string{"x11vnc"}, []string{"-version"}, "x11vnc", map[string]string{
		"apt-get": "x11vnc",
	}, "Install x11vnc to expose the virtual Linux desktop over VNC")
}
