/**
 * Backlog domain types.
 */

import type {
  BacklogItem as ProtoBacklogItem,
  BacklogFile as ProtoBacklogFile,
} from "@vrooli/proto-types/swarm-manager/v1/domain/backlog_pb";
import type { ProtoMessage } from "./shared";

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
 * A backlog item represents a unit of work for the swarm.
 */
export type BacklogItem = Omit<ProtoMessage<ProtoBacklogItem>, "status" | "kind" | "dependsOn" | "initiative" | "acceptanceAllow" | "acceptanceDeny"> & {
  /** Current lifecycle state */
  status: BacklogStatus;
  /** Backlog category */
  kind: BacklogKind;
  /** Items this depends on, as "kind/name" refs. Empty array from API, optional in client code. */
  dependsOn?: string[];
  /** Initiative this item belongs to. */
  initiative?: string;
  /** Glob patterns for expected file modifications. */
  acceptanceAllow?: string[];
  /** Glob patterns for forbidden file modifications. */
  acceptanceDeny?: string[];
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
  initiative?: string;
  effort?: string;
  acceptanceAllow?: string[];
  acceptanceDeny?: string[];
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
