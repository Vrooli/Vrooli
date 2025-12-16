package main

type PlanStep struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type PlanResponse struct {
	Plan      []PlanStep        `json:"plan"`
	Issues    []ValidationIssue `json:"issues,omitempty"`
	Timestamp string            `json:"timestamp"`
}

func BuildPlan(manifest CloudManifest) []PlanStep {
	return []PlanStep{
		{
			ID:          "preflight",
			Title:       "VPS Preflight",
			Description: "Validate SSH connectivity, OS version, disk/RAM, outbound network, ports 80/443, and that DNS points to the VPS.",
		},
		{
			ID:          "bundle",
			Title:       "Build mini-Vrooli bundle",
			Description: "Create a tarball with required scenarios/resources (from analyzer snapshot), plus packages/ and vrooli-autoheal.",
		},
		{
			ID:          "upload",
			Title:       "Upload bundle",
			Description: "SCP tarball to the VPS target and extract into the deployment workdir.",
		},
		{
			ID:          "setup",
			Title:       "Vrooli setup",
			Description: "Run ./scripts/manage.sh setup --yes yes and apply scoped autoheal config.",
		},
		{
			ID:          "deploy",
			Title:       "Start resources and scenario",
			Description: "Start required resources, then start the scenario with fixed ports (UI 3000 / API 3001 / WS 3002).",
		},
		{
			ID:          "edge",
			Title:       "Configure Caddy + TLS",
			Description: "Install/configure Caddy and obtain a Let’s Encrypt certificate for edge.domain (HTTP-01).",
		},
		{
			ID:          "verify",
			Title:       "Verify health",
			Description: "Verify local readiness and https://<domain>/health returns 200 through Caddy.",
		},
	}
}
