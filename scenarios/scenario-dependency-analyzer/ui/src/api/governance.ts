import { createClient } from "@connectrpc/connect";
import {
  DependencyGovernanceService,
  type ApprovedDependencyListResponse,
  type ApprovedDependencyTriageResponse,
  type ApprovedDependencyRecord,
  type SecurityGovernanceGapsResponse,
  type FleetApprovedDependencyValidationResponse,
  type UpsertApprovedDependencyResponse,
  type VulnerabilityRemediationResponse
} from "@vrooli/proto-types/scenario-dependency-analyzer/v1/dependency_governance/dependency_governance_pb";

import { transport } from "./client";

export const governanceClient = createClient(DependencyGovernanceService, transport);

export type GovernancePolicyMode = "advisory" | "strict" | "review_gate";

export async function validateFleetApprovedDependencies(
  policyMode: GovernancePolicyMode = "advisory"
): Promise<FleetApprovedDependencyValidationResponse> {
  return governanceClient.validateFleetApprovedDependencies({ policyMode });
}

export async function listApprovedDependencies(): Promise<ApprovedDependencyListResponse> {
  return governanceClient.listApprovedDependencies({});
}

export async function getApprovedDependencyTriage(options: {
  policyMode?: GovernancePolicyMode;
  section?: string;
  ecosystem?: string;
  packageName?: string;
  limit?: number;
} = {}): Promise<ApprovedDependencyTriageResponse> {
  return governanceClient.getApprovedDependencyTriage({
    policyMode: options.policyMode ?? "advisory",
    section: options.section ?? "",
    ecosystem: options.ecosystem ?? "",
    packageName: options.packageName ?? "",
    limit: options.limit ?? 10
  });
}

export async function listSecurityGovernanceGaps(options: {
  ecosystem?: string;
  packageName?: string;
  severity?: string;
  limit?: number;
} = {}): Promise<SecurityGovernanceGapsResponse> {
  return governanceClient.listSecurityGovernanceGaps({
    ecosystem: options.ecosystem ?? "",
    packageName: options.packageName ?? "",
    severity: options.severity ?? "",
    limit: options.limit ?? 10
  });
}

export async function upsertApprovedDependency(
  record: ApprovedDependencyRecord,
  dryRun = true
): Promise<UpsertApprovedDependencyResponse> {
  return governanceClient.upsertApprovedDependency({ record, dryRun });
}

export async function previewVulnerabilityRemediation(
  ecosystem: string,
  packageName: string,
  vulnerabilityId: string
): Promise<VulnerabilityRemediationResponse> {
  return governanceClient.previewVulnerabilityRemediation({
    ecosystem,
    packageName,
    vulnerabilityId
  });
}

export async function denyVulnerableDependency(options: {
  ecosystem: string;
  packageName: string;
  vulnerabilityId: string;
  affectedRange: string;
  fixedRange: string;
  rationale: string;
  approvedBy: string;
  dryRun?: boolean;
}): Promise<VulnerabilityRemediationResponse> {
  return governanceClient.denyVulnerableDependency({
    ecosystem: options.ecosystem,
    packageName: options.packageName,
    vulnerabilityId: options.vulnerabilityId,
    affectedRange: options.affectedRange,
    fixedRange: options.fixedRange,
    rationale: options.rationale,
    approvedBy: options.approvedBy,
    dryRun: options.dryRun ?? true
  });
}

export type {
  ApprovedDependencyFinding,
  ApprovedDependencyRecord,
  ApprovedDependencyTriageResponse,
  DependencyGovernanceSummary,
  DependencyGovernanceTriageGroup,
  DependencyUsageGroup,
  FleetApprovedDependencyValidationResponse,
  SecurityGovernanceGap,
  SecurityGovernanceGapsResponse,
  SecurityVulnerabilityEvidence,
  UpsertApprovedDependencyResponse,
  VulnerabilityRemediationResponse
} from "@vrooli/proto-types/scenario-dependency-analyzer/v1/dependency_governance/dependency_governance_pb";
