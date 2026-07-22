/**
 * Backlog domain types.
 */

import type {
  BacklogItem as ProtoBacklogItem,
  BacklogFile as ProtoBacklogFile,
} from "@vrooli/proto-types/swarm-manager/v1/domain/backlog_pb";
import type { ProtoMessage } from "./shared";
import type { PlanRef } from "./shared";
import type { AgentSessionAttribution } from "./agent-session";

/**
 * Valid lifecycle states for a backlog item.
 *
 * Lifecycle:
 *   backlog/researching/ready → queued → in_progress → in_review → review_pending → completed | failed | needs_followup
 *
 * - `in_review`: execution completed; review agent is gathering evidence.
 * - `review_pending`: review complete; awaiting user decision via review-decide.
 * - Terminal transitions (`completed`, `failed`, `needs_followup`) are user-only.
 */
export type BacklogStatus =
  | "suggested"
  | "backlog"
  | "researching"
  | "ready"
  | "queued"
  | "in_progress"
  | "in_review"
  | "review_pending"
  | "completed"
  | "failed"
  | "needs_followup";


/**
 * Main backlog categories.
 */
export type BacklogKind = "idea" | "research" | "fix" | "execute" | "chore";

/**
 * A backlog item represents a unit of work for the swarm.
 */
export type BacklogItem = Omit<ProtoMessage<ProtoBacklogItem>, "status" | "kind" | "dependsOn" | "milestone" | "acceptanceAllow" | "acceptanceDeny" | "creates" | "createdBy" | "stale"> & {
  /** Current lifecycle state */
  status: BacklogStatus;
  /** ISO timestamp when the item was archived, or undefined if not archived. */
  archivedAt?: string;
  /** Backlog category */
  kind: BacklogKind;
  /** Items this depends on, as "kind/name" refs. Empty array from API, optional in client code. */
  dependsOn?: string[];
  /** Milestone this item belongs to. */
  milestone?: string;
  /** Glob patterns for expected file modifications. */
  acceptanceAllow?: string[];
  /** Glob patterns for forbidden file modifications. */
  acceptanceDeny?: string[];
  /** Glob patterns for paths the work plans to create (forward-looking acceptance). */
  creates?: string[];
  /** Verified provenance for the actor/session that created this item. */
  createdBy?: AgentSessionAttribution;
  /** Canonical plan-manager plan backing this work item. */
  planRef?: PlanRef;
  /** Explicit authorization for the currently bound canonical plan revision. */
  planAcceptance?: {
    actor: string;
    acceptedAt: string;
    planContentHash: string;
    subjectVersion: string;
  };
  /** Read-time lifecycle signal; never persisted in an item spec. */
  stale?: boolean;
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
  dependsOn?: string[];
  milestone?: string;
  effort?: string;
  acceptanceAllow?: string[];
  acceptanceDeny?: string[];
}

/**
 * A single blocking reason with forceability metadata.
 * Mirrors the proto BlockingReason message.
 */
export interface BlockingReason {
  message: string;
  forceable: boolean;
}

/**
 * Per-item blocking summary from the list endpoint.
 * Mirrors the proto ItemBlockingInfo message.
 */
export interface ItemBlockingInfo {
  blocked: boolean;
  blockingDepKeys: string[];
  allForceable: boolean;
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
