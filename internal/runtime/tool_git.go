package runtime

func newGitTool() handler {
	return newToolHandler("git", []string{"git"}, []string{"--version"}, "git", nil, "Install Git for repository operations")
}
