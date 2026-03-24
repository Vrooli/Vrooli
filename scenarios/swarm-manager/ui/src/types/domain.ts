/**
 * Domain types for Swarm Manager
 *
 * This module contains the core domain types that represent the business concepts.
 * These types are shared across the UI and should match the API contract.
 *
 * DOC: docs/concepts/ARCHITECTURE.md#domain-concepts
 * DOC: docs/internal/SEAMS.md#module-boundaries
 * DOC: docs/internal/INTENT.md#module-responsibilities
 */

import type { Message } from "@bufbuild/protobuf";
import type {
  BacklogItem as ProtoBacklogItem,
  BacklogFile as ProtoBacklogFile,
} from "@vrooli/proto-types/swarm-manager/v1/domain/backlog_pb";
import type {
  Initiative as ProtoInitiative,
  InitiativeRollup as ProtoInitiativeRollup,
} from "@vrooli/proto-types/swarm-manager/v1/domain/initiative_pb";
import type { Scenario as ProtoScenario } from "@vrooli/proto-types/swarm-manager/v1/domain/scenario_pb";
import type { Settings as ProtoSettings } from "@vrooli/proto-types/swarm-manager/v1/domain/settings_pb";
import type {
  ScenarioFile as ProtoScenarioFile,
  PreserveFilesRequest as ProtoPreserveFilesRequest,
  DeleteScenarioRequest as ProtoDeleteScenarioRequest,
  DeleteScenarioResponse as ProtoDeleteScenarioResponse,
  UpdateScenarioMetadataRequest as ProtoUpdateScenarioMetadataRequest,
} from "@vrooli/proto-types/swarm-manager/v1/api/scenarios_pb";
import type { AgentManagerStatusResponse as ProtoAgentManagerStatusResponse } from "@vrooli/proto-types/swarm-manager/v1/api/agent_manager_pb";
import type { BacklogResearchResponse as ProtoBacklogResearchResponse } from "@vrooli/proto-types/swarm-manager/v1/api/backlog_pb";
import type {
  ExecutionRecord as ProtoExecutionRecord,
  ExecutionPolicy as ProtoExecutionPolicy,
} from "@vrooli/proto-types/swarm-manager/v1/domain/execution_pb";

type ProtoMessage<T extends Message> = Omit<T, "$typeName" | "$unknown">;

// ============================================================================
// Backlog Domain
// ============================================================================

/**
 * Valid lifecycle states for a backlog item
 */
export type BacklogStatus =
  | "backlog"
  | "researching"
  | "ready"
  | "queued"
  | "in_progress"
  | "completed"
  | "failed"
  | "archived";

/**
 * Main backlog categories.
 */
export type BacklogKind = "idea" | "research" | "fix" | "execute" | "chore";

/**
 * Optional target kind for research items.
 */
export type BacklogResearchTarget = "idea" | "fix" | "execute" | "chore" | "unspecified";

/**
 * A backlog item represents a unit of work for the swarm.
 */
export type BacklogItem = Omit<ProtoMessage<ProtoBacklogItem>, "status" | "kind" | "researchTarget" | "dependsOn" | "initiative"> & {
  /** Current lifecycle state */
  status: BacklogStatus;
  /** Backlog category */
  kind: BacklogKind;
  /** Research target (research items only) */
  researchTarget?: BacklogResearchTarget;
  /** Items this depends on, as "kind/name" refs. Empty array from API, optional in client code. */
  dependsOn?: string[];
  /** Initiative this item belongs to. */
  initiative?: string;
};

/**
 * Form values for creating or editing a backlog item.
 */
export interface BacklogFormValues {
  name: string;
  title: string;
  description: string;
  status: BacklogStatus;
  priority: number;
  tags: string[];
  kind: BacklogKind;
  researchTarget?: BacklogResearchTarget;
  dependsOn?: string[];
  initiative?: string;
}

/**
 * File type in the backlog file tree
 */
export type BacklogFileType = "file" | "directory";

/**
 * Represents a file or directory within a backlog folder.
 */
export type BacklogFile = Omit<ProtoMessage<ProtoBacklogFile>, "type" | "size" | "children"> & {
  /** Whether this is a file or directory */
  type: BacklogFileType;
  /** File size in bytes (only for files) */
  size?: number;
  /** Child files (only for directories) */
  children?: BacklogFile[];
};

// ============================================================================
// Initiative Domain
// ============================================================================

/**
 * A named grouping of related backlog items.
 */
export type Initiative = ProtoMessage<ProtoInitiative>;

/**
 * Aggregated status counts for an initiative's member items.
 */
export type InitiativeRollup = ProtoMessage<ProtoInitiativeRollup>;

/**
 * Initiative with computed rollup from member items.
 */
export interface InitiativeWithRollup {
  initiative: Initiative;
  rollup: InitiativeRollup;
}

// ============================================================================
// Capture Domain
// ============================================================================

/**
 * Lifecycle status of a capture.
 */
export type CaptureStatus = "classifying" | "classified" | "failed";

/**
 * A raw, unclassified thought from the user.
 */
export interface Capture {
  id: string;
  text: string;
  attachments: string[];
  created: string;
  status: CaptureStatus;
  classification: CaptureClassification | null;
}

/**
 * Classification result — contains 1-N suggested backlog items from a single capture.
 */
export interface CaptureClassification {
  items: CaptureClassificationItem[];
  classifiedAt: string;
}

/**
 * One suggested backlog item extracted from a capture.
 */
export interface CaptureClassificationItem {
  kind: BacklogKind;
  title: string;
  description: string;
  priority: number;
  tags: string[];
  confidence: number;
}

// ============================================================================
// Backlog Agent Domain
// ============================================================================

// ============================================================================
// Workshop Domain
// ============================================================================

/**
 * Workshop item types within a round.
 */
export type WorkshopItemType = "decision" | "info";

/**
 * The 5 universal readiness dimensions.
 */
export type ReadinessDimension =
  | "problem_clarity"
  | "scope_defined"
  | "approach_solid"
  | "testable"
  | "risk_awareness";

/**
 * A lettered choice within a decision item.
 */
export interface DecisionOption {
  key: string;
  label: string;
  rationale: string;
}

/**
 * A single item within a workshop round — either a decision point or informational.
 */
export interface WorkshopItem {
  id: string;
  type: WorkshopItemType;
  topic?: string;
  text?: string;
  context?: string;
  options?: DecisionOption[];
  selected?: string | null;
  freeform?: string | null;
  notes?: string | null;
}

/**
 * A single workshop round stored on disk.
 */
export interface WorkshopRound {
  round: number;
  generated_at: string;
  readiness: Record<ReadinessDimension, number>;
  items: WorkshopItem[];
  plan_updates?: string;
}

/**
 * Maturity/readiness data for a single backlog item (from maturity-summary endpoint).
 */
export interface MaturityItemSummary {
  kind: BacklogKind;
  name: string;
  title: string;
  rounds_completed: number;
  raw_scores: Record<ReadinessDimension, number>;
  effective_scores: Record<ReadinessDimension, number>;
  ready: boolean;
  pending_items: number;
  has_plan: boolean;
}

/**
 * Response from the maturity-summary endpoint.
 */
export interface MaturitySummaryResponse {
  items: MaturityItemSummary[];
}

/**
 * Summary of pending feedback for a single backlog item.
 */
export interface FeedbackItemSummary {
  kind: BacklogKind;
  name: string;
  title: string;
  pending_decisions: number;
}

/**
 * Response from the feedback summary endpoint.
 */
export interface FeedbackSummaryResponse {
  items: FeedbackItemSummary[];
  total_pending: number;
  total_items_affected: number;
}

// ============================================================================
// Pending Questions Domain (inline question stepper)
// ============================================================================

/**
 * Source of a pending question — either a workshop decision or a target/requirement review.
 */
export type PendingQuestionSource = "workshop" | "review";

/**
 * Type of review item within a pending question.
 */
export type PendingReviewItemType = "target" | "requirement";

/**
 * A single pending question from the pending-questions endpoint.
 * Unifies workshop decisions and unreviewed targets/requirements.
 */
export interface PendingQuestion {
  id: string;
  source: PendingQuestionSource;
  item_kind: BacklogKind;
  item_name: string;
  // Workshop decision fields
  topic?: string;
  text?: string;
  context?: string;
  options?: DecisionOption[];
  selected?: string | null;
  freeform?: string | null;
  notes?: string | null;
  round_number?: number;
  // Review fields
  title?: string;
  description?: string;
  criticality?: string;
  review_status?: ReviewStatus;
  review_comment?: string;
  review_type?: PendingReviewItemType;
  module_id?: string;
}

/**
 * Pending questions grouped by backlog item.
 */
export interface PendingQuestionsItem {
  kind: BacklogKind;
  name: string;
  questions: PendingQuestion[];
}

/**
 * Response from the pending-questions endpoint.
 */
export interface PendingQuestionsResponse {
  items: PendingQuestionsItem[];
}

// ============================================================================
// Review Domain
// ============================================================================

export type ReviewStatus = "approved" | "flagged" | "unreviewed";

export interface ReviewAction {
  review_status: ReviewStatus;
  review_comment?: string;
}

export interface ReviewUpdate {
  id: string;
  type: "target" | "requirement";
  module_id?: string;
  review_status: ReviewStatus;
  review_comment?: string;
}

// ============================================================================
// Archive / Operational Targets Domain
// ============================================================================

export interface ArchiveTarget {
  id: string;
  criticality: string;
  title: string;
  notes: string;
  status: string;
  linked_requirement_ids: string[];
  reviewed_at?: string;
  review_comment?: string;
  review_status?: ReviewStatus;
}

export interface ArchiveRequirement {
  id: string;
  title: string;
  description: string;
  status: string;
  category: string;
  prd_ref: string;
  reviewed_at?: string;
  review_comment?: string;
  review_status?: ReviewStatus;
}

export interface ArchiveRequirementGroup {
  id: string;
  name: string;
  requirements: ArchiveRequirement[];
  children: ArchiveRequirementGroup[];
}

export interface ArchiveRequirementRecord {
  id: string;
  title: string;
  description: string;
  status: string;
  category: string;
  prd_ref: string;
  criticality?: string;
  validation?: Array<{ type: string; phase: string; status: string; ref: string }>;
  dependencies?: string[];
  notes?: string;
}

export interface ModuleFormValues {
  id: string;
  title: string;
  description: string;
}

export interface ArchiveTargetFormValues {
  id: string;
  criticality: string;
  title: string;
  notes: string;
  status: string;
  linked_requirement_ids: string[];
}

export interface ArchiveTargetsResponse {
  targets: ArchiveTarget[];
  requirements: ArchiveRequirementGroup[];
  has_archive: boolean;
}

// ============================================================================
// Scenarios Domain
// ============================================================================

/**
 * Valid runtime states for a scenario
 */
export type ScenarioStatus = "running" | "stopped" | "error" | "unknown";

/**
 * A scenario represents a deployed application in the Vrooli ecosystem.
 * [REQ:REQ-P0-007] Includes metadata for greenfield toggle
 */
export type Scenario = Omit<ProtoMessage<ProtoScenario>, "status"> & {
  /** Current runtime state */
  status: ScenarioStatus;
};

/**
 * Request to update scenario metadata
 * [REQ:REQ-P0-007] Update metadata for greenfield
 */
export type UpdateScenarioMetadataRequest = ProtoMessage<ProtoUpdateScenarioMetadataRequest>;

/**
 * File type in the scenario file tree
 */
export type ScenarioFileType = "file" | "directory";

/**
 * Represents a file or directory within a scenario folder.
 */
export type ScenarioFile = Omit<ProtoMessage<ProtoScenarioFile>, "type" | "size" | "children"> & {
  /** Whether this is a file or directory */
  type: ScenarioFileType;
  /** File size in bytes (only for files) */
  size?: number;
  /** Child files (only for directories) */
  children?: ScenarioFile[];
};

/**
 * Available presets for file preservation during archive
 */
export type PreserveFilesPreset = "documentation" | "requirements" | "planning" | "all-planning";

/**
 * Request to specify which files to preserve when archiving
 */
export type PreserveFilesRequest = Partial<
  Omit<ProtoMessage<ProtoPreserveFilesRequest>, "preset">
> & {
  /** Preset name: "documentation", "requirements", "planning", "all-planning" */
  preset?: PreserveFilesPreset;
};

/**
 * Request body for DELETE /api/v1/scenarios/{name}
 */
export type DeleteScenarioRequest = Omit<ProtoMessage<ProtoDeleteScenarioRequest>, "preserveFiles"> & {
  /** Optional file preservation settings when archiving */
  preserveFiles?: PreserveFilesRequest;
};

/**
 * Response from scenario deletion
 * [REQ:REQ-P0-008] Deletion confirmation with archive status
 */
export type DeleteScenarioResponse = ProtoMessage<ProtoDeleteScenarioResponse>;

/**
 * Response from spec-sync-archive
 * Contains execution ID for progress polling
 */
export interface SpecSyncArchiveResponse {
  executionId: string;
  status: string;
  message: string;
}

// Agent Manager Domain
// ============================================================================

export type AgentManagerStatus = ProtoMessage<ProtoAgentManagerStatusResponse>;

export type AgentRunStatus =
  | "pending"
  | "starting"
  | "running"
  | "needs_review"
  | "complete"
  | "failed"
  | "cancelled"
  | "unspecified";

export interface AgentRunState {
  runId: string;
  taskId?: string;
  status: AgentRunStatus;
  startedAt?: string;
  finishedAt?: string;
  errorMessage?: string;
  durationSeconds?: number;
  active: boolean;
}

// ============================================================================
// Settings Domain
// ============================================================================

/**
 * Theme preference for the UI.
 */
export type ThemePreference = "dark" | "light" | "system";

/**
 * User preferences and configuration
 */
export type Settings = Omit<
  ProtoMessage<ProtoSettings>,
  "theme"
> & {
  /** UI theme preference */
  theme: ThemePreference;
};

/**
 * Research agent spawn response
 */
export type ResearchResponse = ProtoMessage<ProtoBacklogResearchResponse>;

// ============================================================================
// Execution Domain
// ============================================================================

export type ExecutionStatus = "pending" | "scheduled" | "starting" | "running" | "needs_review" | "completed" | "failed" | "canceled";

export type ExecutionMode = "manual" | "scheduled" | "yolo";

export type ExecutionRecord = Omit<ProtoMessage<ProtoExecutionRecord>, "status" | "mode" | "operation"> & {
  status: ExecutionStatus;
  mode: ExecutionMode;
  operation?: "generator" | "improver";
};

export type ExecutionPolicy = Omit<ProtoMessage<ProtoExecutionPolicy>, "defaultDelaySeconds"> & {
  defaultMode: ExecutionMode;
  defaultDelaySeconds: number;
};

// ============================================================================
// Prompt Center Domain
// ============================================================================

export interface PromptTrace {
  skill_id: string;
  purpose: string;
  variables?: Record<string, string>;
  prompt: string;
  used_fallback: boolean;
  captured_at: string;
}

export interface PromptBinding {
  area: "research" | "process";
  trigger: string;
  kind?: string;
  mode?: string;
  operation?: string;
  skill_id: string;
  purpose: string;
  output_paths?: string[];
}

export interface PromptSkillSummary {
  id: string;
  name: string;
  description: string;
  default_scope?: string;
  draft: boolean;
  updated_at?: string;
  created_at?: string;
  trigger_count: number;
  impact_summary: string;
  current_content?: string;
  required_missing?: string[];
}

export interface PromptSkillVersion {
  version: number;
  content: string;
  name: string;
  updatedAt: string;
  createdBy?: string;
}


export interface PromptSkillVersions {
  skillId: string;
  current: number;
  versions: PromptSkillVersion[];
}
