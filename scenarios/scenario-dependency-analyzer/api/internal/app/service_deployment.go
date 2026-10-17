package app

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/deployment"
	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

type deploymentService struct {
	workspace *scenarioWorkspace
}

func (d *deploymentService) GetDeploymentReport(name string, refresh bool) (*types.DeploymentAnalysisReport, error) {
	return d.getDeploymentReport(name, refresh, false)
}

func (d *deploymentService) GetDeploymentReportWithOptions(name string, refresh, includeProgramBindings bool) (*types.DeploymentAnalysisReport, error) {
	return d.getDeploymentReport(name, refresh, includeProgramBindings)
}

func (d *deploymentService) getDeploymentReport(name string, refresh, includeProgramBindings bool) (*types.DeploymentAnalysisReport, error) {
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
		options := deployment.DependencyBuildOptions{IncludeProgramBindings: includeProgramBindings}
		if includeProgramBindings {
			options.ProgramBindings = deployment.CompositeProgramBindingSource{ScenariosDir: d.workspace.root}
		}
		report = deployment.BuildReportWithOptions(name, scenarioPath, d.workspace.root, cfg, options)
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
