package main

func (a *App) cmdPlan(args []string) error {
	return a.postManifestOnly(args, "plan", "/api/v1/plan")
}
