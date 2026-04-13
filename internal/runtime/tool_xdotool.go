package runtime

func newXDoTool() handler {
	return newToolHandler("xdotool", []string{"xdotool"}, []string{"-v"}, "xdotool", map[string]string{
		"apt-get": "xdotool",
	}, "Install xdotool for X11 window inspection and desktop automation helpers")
}
