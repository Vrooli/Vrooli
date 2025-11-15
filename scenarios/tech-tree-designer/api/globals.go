package main

import "database/sql"

var (
	db             *sql.DB
	catalogManager *ScenarioCatalogManager

	stageTypeAliases = map[string]string{
		"foundation":      "foundation",
		"foundational":    "foundation",
		"operational":     "operational",
		"operations":      "operational",
		"analytics":       "analytics",
		"analysis":        "analytics",
		"integration":     "integration",
		"integrated":      "integration",
		"digital_twin":    "digital_twin",
		"digital twin":    "digital_twin",
		"twin":            "digital_twin",
		"custom":          "custom",
		"exploratory":     "exploratory",
		"experimentation": "experimentation",
		"governance":      "governance",
		"compliance":      "compliance",
		"platform":        "platform",
		"infrastructure":  "infrastructure",
		"innovation":      "innovation",
		"intelligence":    "analytics",
		"coordination":    "integration",
		"automations":     "operational",
	}

	stageIdeaTemplates = map[string][]ideaTemplate{
		"software": {
			{Name: "Autonomous Refactoring Grid", StageType: "analytics", Description: "Maps codebase entropy and recommends refactors across repositories", Scenarios: []string{"code-smell", "code-tidiness-manager", "graph-studio"}},
			{Name: "Agent Integration Fabric", StageType: "integration", Description: "Unifies agent-to-agent workflows across the platform", Scenarios: []string{"agent-dashboard", "swarm-manager", "scenario-dependency-analyzer"}},
		},
		"manufacturing": {
			{Name: "Adaptive Line Simulation", StageType: "digital_twin", Description: "Predictive simulation for production cells with material + labor constraints", Scenarios: []string{"scenario-to-desktop", "workflow-scheduler"}},
			{Name: "Supply Coordination Mesh", StageType: "integration", Description: "Real-time supplier telemetry fused with MES for constraint solving", Scenarios: []string{"system-monitor", "ecosystem-manager"}},
		},
		"healthcare": {
			{Name: "Clinical Narrative Intelligence", StageType: "analytics", Description: "Transforms unstructured notes into treatment readiness scores", Scenarios: []string{"research-assistant", "privacy-terms-generator"}},
			{Name: "Care Continuity Graph", StageType: "integration", Description: "Connects home, clinic, and hospital data streams for proactive alerts", Scenarios: []string{"notification-hub", "personal-relationship-manager"}},
		},
		"finance": {
			{Name: "Risk Scenario Forge", StageType: "analytics", Description: "Runs Monte Carlo economic shocks across treasury + payments", Scenarios: []string{"financial-calculators-hub", "roi-fit-analysis"}},
			{Name: "Policy Automation Spine", StageType: "operational", Description: "Applies programmable financial controls to new instruments", Scenarios: []string{"secrets-manager", "system-monitor"}},
		},
		"education": {
			{Name: "Adaptive Mastery Engine", StageType: "analytics", Description: "Predicts learner gaps and emits personalized quests", Scenarios: []string{"study-buddy", "recommendation-engine"}},
			{Name: "Skill Graph Weaving", StageType: "integration", Description: "Links curriculum outcomes to workplace skills", Scenarios: []string{"resume-screening-assistant", "personal-digital-twin"}},
		},
		"governance": {
			{Name: "Civic Signal Mesh", StageType: "integration", Description: "Aggregates public sentiment + sensor data for policy readiness", Scenarios: []string{"prompt-manager", "news-aggregator-bias-analysis"}},
			{Name: "Regulation Co-Pilot", StageType: "operational", Description: "Autogenerates compliance playbooks per jurisdiction", Scenarios: []string{"privacy-terms-generator", "scenario-auditor"}},
		},
	}
)
