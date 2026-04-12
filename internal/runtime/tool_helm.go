package runtime

func newHelmTool() handler {
	return newToolHandler("helm", []string{"helm"}, []string{"version", "--short"}, "helm", nil, "Install Helm for Kubernetes packaging flows")
}
