package runtime

func newDockerTool() handler {
	return newToolHandler("docker", []string{"docker"}, []string{"--version"}, "docker.io", map[string]string{
		"brew": "docker",
	}, "Install Docker Engine or Docker Desktop")
}
