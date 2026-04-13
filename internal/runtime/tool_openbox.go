package runtime

func newOpenboxTool() handler {
	return newToolHandler("openbox", []string{"openbox"}, []string{"--version"}, "openbox", map[string]string{
		"apt-get": "openbox",
	}, "Install openbox to provide a lightweight Linux window manager for virtual desktop sessions")
}
