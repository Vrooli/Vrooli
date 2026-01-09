package main

func (a *App) cmdBundleBuild(args []string) error {
	return a.postManifestOnly(args, "bundle-build", "/api/v1/bundle/build")
}
