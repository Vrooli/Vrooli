import { createClient } from "@connectrpc/connect";

import {
  FleetService,
  type ScanFleetResponse,
  type FleetScenarioEntry,
  type TierDistribution,
  type FleetScanError,
} from "@vrooli/proto-types/performance-health/v1/fleet/fleet_pb";
import {
  ReadinessService,
  type ValidateReadinessResponse,
  type ReadinessFixResponse,
  CaptureTier,
} from "@vrooli/proto-types/performance-health/v1/readiness/readiness_pb";
import {
  AuditService,
  type RunAuditResponse,
  AuditOutcome,
} from "@vrooli/proto-types/performance-health/v1/audit/audit_pb";
import {
  TrendService,
  type GetTrendResponse,
  type TrendSample,
} from "@vrooli/proto-types/performance-health/v1/trend/trend_pb";
import {
  StartupService,
  type GetStartupTrendResponse,
  type StartupMeasurement,
} from "@vrooli/proto-types/performance-health/v1/startup/startup_pb";
import {
  BudgetService,
  type Budget,
  type GetBudgetResponse,
  type SetBudgetResponse,
  type CheckBudgetResponse,
  type BudgetViolation,
} from "@vrooli/proto-types/performance-health/v1/budgets/budgets_pb";
import {
  AnalysisService,
  type AnalyzeTraceResponse,
  type ComponentTiming,
  type PerfFinding,
  type CompareTracesResponse,
  type ComponentDelta,
} from "@vrooli/proto-types/performance-health/v1/analysis/analysis_pb";

import { transport } from "./client";

/**
 * Typed Connect-Web clients for every performance-health product domain the UI
 * speaks to. Each method is a thin wrapper that forwards the request fields the
 * UI actually sets, so screens read strictly off the generated proto contracts
 * (no hand-rolled fetch, no untyped JSON). Verdicts and tiers mirror the gating
 * semantics returned by the backend.
 */
const fleet = createClient(FleetService, transport);
const readiness = createClient(ReadinessService, transport);
const audit = createClient(AuditService, transport);
const trend = createClient(TrendService, transport);
const startup = createClient(StartupService, transport);
const budgets = createClient(BudgetService, transport);
const analysis = createClient(AnalysisService, transport);

export const perfClient = {
  scanFleet: (input: { scenarios?: string[] } = {}): Promise<ScanFleetResponse> =>
    fleet.scanFleet({ scenarios: input.scenarios ?? [] }),

  validateReadiness: (input: { scenario: string }): Promise<ValidateReadinessResponse> =>
    readiness.validateReadiness({ scenario: input.scenario }),
  previewReadinessFix: (input: {
    scenario: string;
    ruleIds?: string[];
  }): Promise<ReadinessFixResponse> =>
    readiness.previewReadinessFix({ scenario: input.scenario, ruleIds: input.ruleIds ?? [] }),
  applyReadinessFix: (input: {
    scenario: string;
    ruleIds?: string[];
  }): Promise<ReadinessFixResponse> =>
    readiness.applyReadinessFix({ scenario: input.scenario, ruleIds: input.ruleIds ?? [] }),

  runAudit: (input: { scenario: string; workflow?: string }): Promise<RunAuditResponse> =>
    audit.runAudit({ scenario: input.scenario, workflow: input.workflow ?? "" }),

  getTrend: (input: { scenario: string; limit?: number }): Promise<GetTrendResponse> =>
    trend.getTrend({ scenario: input.scenario, limit: input.limit ?? 30 }),

  getStartupTrend: (input: { scenario: string; limit?: number }): Promise<GetStartupTrendResponse> =>
    startup.getStartupTrend({ scenario: input.scenario, limit: input.limit ?? 30 }),

  getBudget: (input: { scenario: string }): Promise<GetBudgetResponse> =>
    budgets.getBudget({ scenario: input.scenario }),
  setBudget: (input: { budget: Budget }): Promise<SetBudgetResponse> =>
    budgets.setBudget({ budget: input.budget }),
  checkBudget: (input: { scenario: string }): Promise<CheckBudgetResponse> =>
    budgets.checkBudget({ scenario: input.scenario }),

  analyzeTrace: (input: {
    scenario: string;
    traceArtifact: string;
  }): Promise<AnalyzeTraceResponse> =>
    analysis.analyzeTrace({ scenario: input.scenario, traceArtifact: input.traceArtifact }),
  compareTraces: (input: {
    scenario: string;
    baselineArtifact: string;
    candidateArtifact: string;
  }): Promise<CompareTracesResponse> =>
    analysis.compareTraces({
      scenario: input.scenario,
      baselineArtifact: input.baselineArtifact,
      candidateArtifact: input.candidateArtifact,
    }),
};

export { CaptureTier, AuditOutcome };
export type {
  ScanFleetResponse,
  FleetScenarioEntry,
  TierDistribution,
  FleetScanError,
  ValidateReadinessResponse,
  ReadinessFixResponse,
  RunAuditResponse,
  GetTrendResponse,
  TrendSample,
  GetStartupTrendResponse,
  StartupMeasurement,
  Budget,
  GetBudgetResponse,
  SetBudgetResponse,
  CheckBudgetResponse,
  BudgetViolation,
  AnalyzeTraceResponse,
  ComponentTiming,
  PerfFinding,
  CompareTracesResponse,
  ComponentDelta,
};
