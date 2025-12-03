package main

const (
	defaultDeploymentMode = "external-server"
	defaultServerType     = "external"
	defaultTemplateType   = "universal"
)

var (
	validFrameworks      = []string{"electron", "tauri", "neutralino"}
	validTemplateTypes   = []string{"universal", "basic", "advanced", "multi_window", "kiosk"} // basic is alias for universal
	validDeploymentModes = []string{defaultDeploymentMode, "cloud-api", "bundled"}
	validServerTypes     = []string{"node", "static", "external", "executable"}
)

func isValidFramework(framework string) bool {
	return contains(validFrameworks, framework)
}

func isValidTemplateType(templateType string) bool {
	return contains(validTemplateTypes, templateType)
}

func isValidDeploymentMode(mode string) bool {
	return contains(validDeploymentModes, mode)
}

func isValidServerType(serverType string) bool {
	return contains(validServerTypes, serverType)
}
