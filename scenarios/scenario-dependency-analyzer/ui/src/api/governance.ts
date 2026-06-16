import { createClient } from "@connectrpc/connect";
import {
  DependencyGovernanceService,
  type ApprovedDependencyListResponse,
  type ApprovedDependencyRecord,
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
  DependencyGovernanceSummary,
  DependencyUsageGroup,
  FleetApprovedDependencyValidationResponse,
  SecurityVulnerabilityEvidence,
  UpsertApprovedDependencyResponse,
  VulnerabilityRemediationResponse
} from "@vrooli/proto-types/scenario-dependency-analyzer/v1/dependency_governance/dependency_governance_pb";
