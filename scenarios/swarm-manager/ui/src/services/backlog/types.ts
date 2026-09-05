/**
 * Backlog Service Types
 *
 * Shared type definitions used across backlog service modules.
 */

import type { NextActionEffect } from "../next-action-service";

import type { BacklogFile, BacklogItem, BacklogKind, BlockingReason, ItemBlockingInfo, PlanRef } from "../../types";

/**
 * Response from queueing a backlog item for processing.
 * When `dryRun` is true the queue did not execute — check `blockingReasons`.
 */
export interface QueueResponse {
  item?: BacklogItem;
  taskId: string;
  runId: string;
  baseUrl: string;
  created: string;
  dryRun: boolean;
  queued: boolean;
  message: string;
  blockingReasons: BlockingReason[];
  pendingDecisions: number;
  pendingSuggestions: number;
}

export interface BacklogFileOperationResult {
  file?: BacklogFile;
  deletedPath?: string;
}

export type BacklogUpdatePatch = Partial<Pick<
  BacklogItem,
  "title" | "description" | "status" | "priority" | "tags" | "dependsOn" | "milestone" | "effort" | "acceptanceAllow" | "acceptanceDeny" | "acceptanceCriteria" | "note"
>>;

export interface ImportBacklogResponse {
  dryRun: boolean;
  changes: Array<{ item: string; action: string; details: string[] }>;
  errors: string[];
  summary: string;
}

export interface RenderedBacklogPlan {
  path: string;
  markdown: string;
  qualityStatus?: string;
  qualityFindings?: string[];
  planRef?: PlanRef;
}

export type BacklogNextActionID = "none" | "decide" | "accept_suggestion" | "author_plan" | "accept_plan" | "repair_plan" | "resolve_dependencies" | "review" | "view_execution" | "run" | "retry" | "archive" | "dispatch_followup" | "author_followup" | "plan_goal" | "define_criteria" | "close_out" | "chain";

export interface BacklogNextAction {
  id: BacklogNextActionID;
  compactLabel: string;
  expandedLabel: string;
  enabled: boolean;
  reason?: string;
  blockers: BlockingReason[];
  target?: string;
  /**
   * What performing this action does to the system, declared by the server so
   * a control can warn before spending agent time. Optional on the wire: a UI
   * build newer than its API treats absence as unknown, never as harmless.
   */
  effect?: NextActionEffect;
  /** Server's marker for actions that remove or interrupt state. */
  destructive?: boolean;
  followUp?: { steering: string; disposition: "follow_up_run" | "replan" | "new_items"; items?: Array<{ kind: string; name: string; title: string }> };
}

/**
 * Interface for the backlog service.
 * This is the seam - implementations can be swapped for testing.
 */
export interface IBacklogService {
  list(kinds?: BacklogKind[]): Promise<{ items: BacklogItem[]; blocking: Record<string, ItemBlockingInfo> }>;
  listBySpawnedFrom(spawnedFrom: string): Promise<BacklogItem[]>;
  get(kind: BacklogKind, name: string): Promise<BacklogItem>;
  getNextAction(kind: BacklogKind, name: string): Promise<BacklogNextAction>;
  getNextActions(items: Array<{ kind: BacklogKind; name: string }>): Promise<Record<string, BacklogNextAction>>;
  create(item: Omit<BacklogItem, "created" | "updated">): Promise<BacklogItem>;
  update(
    kind: BacklogKind,
    name: string,
    patch: BacklogUpdatePatch
  ): Promise<BacklogItem>;
  delete(kind: BacklogKind, name: string): Promise<void>;
  archiveItem(kind: BacklogKind, name: string): Promise<BacklogItem>;
  unarchiveItem(kind: BacklogKind, name: string): Promise<BacklogItem>;
  getFiles(kind: BacklogKind, name: string): Promise<BacklogFile[]>;
  getFileContent(kind: BacklogKind, name: string, filePath: string): Promise<string>;
  getRenderedPlan(kind: BacklogKind, name: string): Promise<RenderedBacklogPlan>;
  startPlanAuthor(kind: BacklogKind, name: string): Promise<{ executionId: string; definitionDigest: string }>;
  uploadFile(kind: BacklogKind, name: string, file: File, path?: string): Promise<BacklogFile>;
  saveFileContent(
    kind: BacklogKind,
    name: string,
    filePath: string,
    content: string,
    contentType?: string
  ): Promise<BacklogFile>;
  renameFile(kind: BacklogKind, name: string, sourcePath: string, destinationPath: string): Promise<BacklogFileOperationResult>;
  moveFile(kind: BacklogKind, name: string, sourcePath: string, destinationPath: string): Promise<BacklogFileOperationResult>;
  copyFile(kind: BacklogKind, name: string, sourcePath: string, destinationPath: string): Promise<BacklogFileOperationResult>;
  deleteFile(kind: BacklogKind, name: string, sourcePath: string): Promise<BacklogFileOperationResult>;
  queue(
    kind: BacklogKind,
    name: string,
    options?: {
      operation?: "generator" | "improver";
      mode?: "manual" | "yolo";
      startedBy?: string;
      confirm?: boolean;
      force?: boolean;
      strategy?: string;
      maxSlices?: number;
    }
  ): Promise<QueueResponse>;
  retry(
    kind: BacklogKind,
    name: string,
    note?: string,
  ): Promise<{ newExecutionId: string; parentExecutionId: string; status: string }>;
  dispatchFollowUp(kind: BacklogKind, name: string): Promise<void>;
  getArchiveTargets(kind: BacklogKind, name: string): Promise<import("../../types").ArchiveTargetsResponse>;
  createArchiveTarget(kind: string, name: string, target: import("../../types").ArchiveTargetFormValues): Promise<void>;
  updateArchiveTarget(kind: string, name: string, targetId: string, target: import("../../types").ArchiveTargetFormValues): Promise<void>;
  deleteArchiveTarget(kind: string, name: string, targetId: string): Promise<void>;
  updateModuleRequirements(kind: string, name: string, moduleId: string, requirements: import("../../types").ArchiveRequirementRecord[]): Promise<void>;
  createModule(kind: string, name: string, payload: import("../../types").ModuleFormValues & { position?: number }): Promise<void>;
  updateModuleMeta(kind: string, name: string, moduleId: string, payload: { title: string; description: string }): Promise<void>;
  deleteModule(kind: string, name: string, moduleId: string): Promise<void>;
  batchReview(kind: string, name: string, items: import("../../types").ReviewUpdate[]): Promise<void>;
  exportItems(params?: {
    kinds?: string[];
    statuses?: string[];
    names?: string[];
    priorityMax?: number;
    tags?: string[];
    includePrd?: boolean;
    includeRequirements?: boolean;
    includeClarifyQuestions?: boolean;
    includeSuggestions?: boolean;
    includeNotes?: boolean;
    includeTemplate?: boolean;
  }): Promise<Blob>;
  importItems(file: File, apply?: boolean): Promise<ImportBacklogResponse>;
  getBacklogSummary(): Promise<import("../../types").BacklogSummaryResponse>;
  getPendingQuestions(): Promise<import("../../types").PendingQuestionsResponse>;
}
