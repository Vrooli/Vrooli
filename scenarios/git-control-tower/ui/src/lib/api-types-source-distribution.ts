export interface SourceDistribution {
  distribution_id: string;
  scenario: string;
  source_digest: string;
  closure_digest: string;
  recipe_digest: string;
  policy_digest: string;
  artifact_id: string;
  artifact_digest: string;
  verification_status: string;
  verification_receipt: string;
  deployment_manager_decision: string;
  publication_status: string;
  destination: string;
  destination_revision: string;
  readback_receipt: string;
  drift_state: string;
  source_of_truth: string;
  source_timestamp: string;
  freshness: string;
  updated_at: string;
  workflow_url: string;
}

export interface DistributionContent { path: string; source_path: string; category: string; digest: string; size_bytes: number; }
export interface DistributionExclusion { path: string; category: string; safe_reason: string; }
export interface PublicationHandoff { status: string; artifact_id: string; artifact_digest: string; destination: string; human_action: string; readback_oracle: string; preconditions: string[]; approval_reference: string; source_of_truth: string; freshness: string; }
export interface DistributionDrift { state: string; source_changed: boolean; destination_changed: boolean; current_source_digest: string; recorded_source_digest: string; current_artifact_digest: string; recorded_artifact_digest: string; actions: string[]; source_of_truth: string; freshness: string; }
export interface SourceDistributionListResponse { available: boolean; source_of_truth: string; freshness: string; unavailable_reason?: string; distributions: SourceDistribution[]; }
export interface SourceDistributionDetailResponse { available: boolean; unavailable_reason?: string; distribution: SourceDistribution; contents: DistributionContent[]; exclusions: DistributionExclusion[]; unresolved_obligations: string[]; runtime_requirements: string[]; handoff?: PublicationHandoff; drift?: DistributionDrift; }
