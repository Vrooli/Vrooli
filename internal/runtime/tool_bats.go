package runtime

func newBatsTool() handler {
	return newToolHandler("bats", []string{"bats"}, []string{"--version"}, "bats", map[string]string{
		"apt-get": "bats",
		"brew":    "bats-core",
	}, "Install bats for shell-based integration suites")
}
