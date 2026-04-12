package runtime

func newTmuxTool() handler {
	return newToolHandler("tmux", []string{"tmux"}, []string{"-V"}, "tmux", nil, "Install tmux for detachable operator workflows")
}
