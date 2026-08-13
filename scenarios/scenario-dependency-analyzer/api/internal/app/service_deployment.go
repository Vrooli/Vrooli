package app

import (
	"fmt"
	"log"
	"path/filepath"

	"scenario-dependency-analyzer/internal/deployment"
	types "scenario-dependency-analyzer/internal/types"
)

type deploymentService struct {
	workspace *scenarioWorkspace
}

func (d *deploymentService) GetDeploymentReport(name string, refresh bool) (*types.DeploymentAnalysisReport, error) {
	scenarioPath := d.workspace.pathFor(name)
	cfg, err := d.workspace.loadConfig(name)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errScenarioNotFound, err)
	}

	var report *types.DeploymentAnalysisReport
	if !refresh {
		report, err = deployment.LoadReport(scenarioPath)
	}
	if refresh || err != nil {
		report = deployment.BuildReport(name, scenarioPath, d.workspace.root, cfg)
		if report != nil {
			if persistErr := deployment.PersistReport(scenarioPath, report); persistErr != nil {
				log.Printf("Warning: failed to persist deployment report for %s: %v", name, persistErr)
			}
		}
	}

	if report == nil {
		return nil, fmt.Errorf("failed to build deployment report for %s", name)
	}

	return report, nil
}

func (d *deploymentService) ExportTargetDAG(expression string, recursive bool, refresh bool) (*types.TargetDAGResponse, error) {
	if d == nil || d.workspace == nil {
		return nil, fmt.Errorf("deployment workspace unavailable")
	}
	return deployment.BuildTargetDAG(filepath.Dir(d.workspace.root), expression, recursive, refresh)
}
