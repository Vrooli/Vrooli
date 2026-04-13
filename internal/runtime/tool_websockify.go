package runtime

func newWebsockifyTool() handler {
	return newToolHandler("websockify", []string{"websockify"}, []string{"--version"}, "websockify", map[string]string{
		"apt-get": "websockify",
	}, "Install websockify to bridge VNC desktop sessions into browser WebSocket endpoints")
}
