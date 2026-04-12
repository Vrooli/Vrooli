package runtime

func newCurlTool() handler {
	return newToolHandler("curl", []string{"curl"}, []string{"--version"}, "curl", nil, "Install curl for HTTP-based setup and diagnostics")
}
