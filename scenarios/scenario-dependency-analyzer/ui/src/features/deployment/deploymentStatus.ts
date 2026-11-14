import type { DeploymentAnalysisReport, ScenarioGapInfo, ScenarioSummary } from "../../types";

export type DeploymentStatusKind = "ready" | "issues" | "not-scanned" | "critical";

export interface DeploymentTierOption {
  value: string;
  label: string;
}

export interface ScenarioDeploymentStatus {
  scenario: ScenarioSummary;
  report: DeploymentAnalysisReport | null;
  loading: boolean;
  status: DeploymentStatusKind;
  tierFitness: { best: number; worst: number } | null;
  blockersCount: number;
  missingMetadataCount: number;
  lastReport?: DeploymentAnalysisReport | null;
}

export interface AggregatedDeploymentGaps {
  total_gaps: number;
  scenarios_missing_all: number;
  gaps_by_scenario: Record<string, ScenarioGapInfo>;
  missing_tiers: string[];
  recommendations: string[];
}

export function buildDeploymentStatus(
  scenario: ScenarioSummary,
  report: DeploymentAnalysisReport | null,
  loading = false
): ScenarioDeploymentStatus {
  let status: DeploymentStatusKind = "not-scanned";
  let blockersCount = 0;
  let missingMetadataCount = 0;
  let tierFitness: { best: number; worst: number } | null = null;

  if (report) {
    const gaps = report.metadata_gaps;
    if (gaps) {
      missingMetadataCount = gaps.total_gaps;
      if (gaps.scenarios_missing_all > 0) {
        status = "critical";
      } else if (gaps.total_gaps > 0) {
        status = "issues";
      } else {
        status = "ready";
      }
    }

    if (report.aggregates && Object.keys(report.aggregates).length > 0) {
      const fitnessScores = Object.values(report.aggregates)
        .map((aggregate) => aggregate.fitness_score)
        .filter((score) => typeof score === "number" && !Number.isNaN(score));

      if (fitnessScores.length > 0) {
        tierFitness = {
          best: Math.max(...fitnessScores),
          worst: Math.min(...fitnessScores)
        };
      }

      Object.values(report.aggregates).forEach((aggregate) => {
        blockersCount += aggregate.blocking_dependencies?.length ?? 0;
      });

      if (blockersCount > 0 && status === "ready") {
        status = "issues";
      }
    }
  }

  return {
    scenario,
    report,
    loading,
    status,
    tierFitness,
    blockersCount,
    missingMetadataCount,
    lastReport: report
  };
}

export function aggregateDeploymentGaps(
  statuses: Iterable<ScenarioDeploymentStatus>
): AggregatedDeploymentGaps | null {
  const allGaps: Record<string, ScenarioGapInfo> = {};
  let totalGaps = 0;
  let scenariosMissingAll = 0;
  const missingTiersSet = new Set<string>();
  const recommendations: string[] = [];

  for (const status of statuses) {
    const gaps = status.report?.metadata_gaps;
    if (!gaps) continue;

    totalGaps += gaps.total_gaps;
    scenariosMissingAll += gaps.scenarios_missing_all;

    if (gaps.gaps_by_scenario) {
      Object.assign(allGaps, gaps.gaps_by_scenario);
    }

    gaps.missing_tiers?.forEach((tier) => missingTiersSet.add(tier));
    gaps.recommendations?.forEach((recommendation) => {
      if (!recommendations.includes(recommendation)) {
        recommendations.push(recommendation);
      }
    });
  }

  if (totalGaps === 0) return null;

  return {
    total_gaps: totalGaps,
    scenarios_missing_all: scenariosMissingAll,
    gaps_by_scenario: allGaps,
    missing_tiers: Array.from(missingTiersSet),
    recommendations: recommendations.slice(0, 5)
  };
}
