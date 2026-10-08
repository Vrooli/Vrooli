import { describe, expect, it } from "vitest";

import type { DeploymentAnalysisReport, ScenarioSummary } from "../../types";
import { aggregateDeploymentGaps, buildDeploymentStatus } from "./deploymentStatus";

const scenario: ScenarioSummary = {
  name: "alpha",
  display_name: "Alpha Scenario"
};

function report(overrides: Partial<DeploymentAnalysisReport> = {}): DeploymentAnalysisReport {
  return {
    scenario: "alpha",
    report_version: 1,
    generated_at: "2026-06-14T12:00:00Z",
    dependencies: [],
    aggregates: {},
    ...overrides
  };
}

describe("deployment status helpers", () => {
  it("marks missing-all metadata as critical", () => {
    const status = buildDeploymentStatus(
      scenario,
      report({
        metadata_gaps: {
          total_gaps: 3,
          scenarios_missing_all: 1,
          gaps_by_scenario: {},
          missing_tiers: ["desktop"],
          recommendations: ["Add deployment metadata."]
        }
      })
    );

    expect(status.status).toBe("critical");
    expect(status.missingMetadataCount).toBe(3);
  });

  it("downgrades otherwise ready scenarios when tier blockers exist", () => {
    const status = buildDeploymentStatus(
      scenario,
      report({
        metadata_gaps: {
          total_gaps: 0,
          scenarios_missing_all: 0,
          gaps_by_scenario: {},
          missing_tiers: [],
          recommendations: []
        },
        aggregates: {
          desktop: {
            fitness_score: 0.8,
            dependency_count: 1,
            blocking_dependencies: ["postgres"],
            estimated_requirements: {
              ram_mb: 512,
              disk_mb: 1024,
              cpu_cores: 1
            }
          }
        }
      })
    );

    expect(status.status).toBe("issues");
    expect(status.blockersCount).toBe(1);
    expect(status.tierFitness).toEqual({ best: 0.8, worst: 0.8 });
  });

  it("aggregates metadata gaps without duplicate recommendations", () => {
    const first = buildDeploymentStatus(
      scenario,
      report({
        metadata_gaps: {
          total_gaps: 2,
          scenarios_missing_all: 0,
          gaps_by_scenario: {
            alpha: {
              scenario_name: "alpha",
              scenario_path: "scenarios/alpha",
              has_deployment_block: false,
              missing_dependency_catalog: true,
              missing_tier_definitions: ["desktop"],
              suggested_actions: []
            }
          },
          missing_tiers: ["desktop"],
          recommendations: ["Add deployment metadata."]
        }
      })
    );
    const second = buildDeploymentStatus(
      { name: "beta", display_name: "Beta Scenario" },
      report({
        scenario: "beta",
        metadata_gaps: {
          total_gaps: 1,
          scenarios_missing_all: 1,
          gaps_by_scenario: {},
          missing_tiers: ["desktop"],
          recommendations: ["Add deployment metadata."]
        }
      })
    );

    expect(aggregateDeploymentGaps([first, second])).toMatchObject({
      total_gaps: 3,
      scenarios_missing_all: 1,
      missing_tiers: ["desktop"],
      recommendations: ["Add deployment metadata."]
    });
  });
});
