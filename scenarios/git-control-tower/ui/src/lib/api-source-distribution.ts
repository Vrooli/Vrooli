import { create } from "@bufbuild/protobuf";
import { GetDistributionDetailsRequestSchema, ListDistributionsRequestSchema } from "@vrooli/proto-types/git-control-tower/v1/source_distributions/source_distributions_pb";
import type { SourceDistributionDetailResponse, SourceDistributionListResponse } from "./api-types-source-distribution";
import { sourceDistributionClient } from "./connect";

export async function fetchSourceDistributions(repoId?: string, scenario?: string): Promise<SourceDistributionListResponse> {
  const response = await sourceDistributionClient.listDistributions(create(ListDistributionsRequestSchema, {
    scenario: scenario ?? "", repositoryContext: repoId ?? "", limit: 50,
  }));
  if (!response.available) throw new Error(response.unavailableReason || "source-ramp unavailable");
  return {
    available: response.available,
    source_of_truth: response.sourceOfTruth,
    freshness: response.freshness,
    distributions: response.distributions.map((d) => ({
      distribution_id: d.distributionId, scenario: d.scenario, source_digest: d.sourceDigest,
      closure_digest: d.closureDigest, recipe_digest: d.recipeDigest, policy_digest: d.policyDigest,
      artifact_id: d.artifactId, artifact_digest: d.artifactDigest, verification_status: d.verificationStatus,
      verification_receipt: d.verificationReceipt, deployment_manager_decision: d.deploymentManagerDecision,
      publication_status: d.publicationStatus, destination: d.destination, destination_revision: d.destinationRevision,
      readback_receipt: d.readbackReceipt, drift_state: d.driftState, source_of_truth: d.sourceOfTruth,
      source_timestamp: d.sourceTimestamp, freshness: d.freshness, updated_at: d.updatedAt, workflow_url: d.workflowUrl,
    })),
  };
}

export async function fetchSourceDistribution(id: string): Promise<SourceDistributionDetailResponse> {
  const d = await sourceDistributionClient.getDistributionDetails(create(GetDistributionDetailsRequestSchema, { distributionId: id }));
  return {
    available: d.available,
    unavailable_reason: d.unavailableReason,
    distribution: {
      distribution_id: d.distribution?.distributionId ?? "", scenario: d.distribution?.scenario ?? "",
      source_digest: d.distribution?.sourceDigest ?? "", closure_digest: d.distribution?.closureDigest ?? "",
      recipe_digest: d.distribution?.recipeDigest ?? "", policy_digest: d.distribution?.policyDigest ?? "",
      artifact_id: d.distribution?.artifactId ?? "", artifact_digest: d.distribution?.artifactDigest ?? "",
      verification_status: d.distribution?.verificationStatus ?? "", verification_receipt: d.distribution?.verificationReceipt ?? "",
      deployment_manager_decision: d.distribution?.deploymentManagerDecision ?? "", publication_status: d.distribution?.publicationStatus ?? "",
      destination: d.distribution?.destination ?? "", destination_revision: d.distribution?.destinationRevision ?? "",
      readback_receipt: d.distribution?.readbackReceipt ?? "", drift_state: d.distribution?.driftState ?? "",
      source_of_truth: d.distribution?.sourceOfTruth ?? "", source_timestamp: d.distribution?.sourceTimestamp ?? "",
      freshness: d.distribution?.freshness ?? "unknown", updated_at: d.distribution?.updatedAt ?? "", workflow_url: d.distribution?.workflowUrl ?? "",
    },
    contents: d.contents.map((x) => ({ path: x.path, source_path: x.sourcePath, category: x.category, digest: x.digest, size_bytes: Number(x.sizeBytes) })),
    exclusions: d.exclusions.map((x) => ({ path: x.path, category: x.category, safe_reason: x.safeReason })),
    unresolved_obligations: d.unresolvedObligations,
    runtime_requirements: d.runtimeRequirements,
    handoff: d.handoff ? { status: d.handoff.status, artifact_id: d.handoff.artifactId, artifact_digest: d.handoff.artifactDigest, destination: d.handoff.destination, human_action: d.handoff.humanAction, readback_oracle: d.handoff.readbackOracle, preconditions: d.handoff.preconditions, approval_reference: d.handoff.approvalReference, source_of_truth: d.handoff.sourceOfTruth, freshness: d.handoff.freshness } : undefined,
    drift: d.drift ? { state: d.drift.state, source_changed: d.drift.sourceChanged, destination_changed: d.drift.destinationChanged, current_source_digest: d.drift.currentSourceDigest, recorded_source_digest: d.drift.recordedSourceDigest, current_artifact_digest: d.drift.currentArtifactDigest, recorded_artifact_digest: d.drift.recordedArtifactDigest, actions: d.drift.actions, source_of_truth: d.drift.sourceOfTruth, freshness: d.drift.freshness } : undefined,
  };
}
