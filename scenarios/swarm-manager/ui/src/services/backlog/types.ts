/**
 * Backlog Service Types
 *
 * Shared type definitions used across backlog service modules.
 */

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
  "title" | "description" | "status" | "priority" | "tags" | "dependsOn" | "milestone" | "effort" | "acceptanceAllow" | "acceptanceDeny" | "note"
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

/**
 * Interface for the backlog service.
 * This is the seam - implementations can be swapped for testing.
 */
export interface IBacklogService {
  list(kinds?: BacklogKind[]): Promise<{ items: BacklogItem[]; blocking: Record<string, ItemBlockingInfo> }>;
  listBySpawnedFrom(spawnedFrom: string): Promise<BacklogItem[]>;
  get(kind: BacklogKind, name: string): Promise<BacklogItem>;
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
    }
  ): Promise<QueueResponse>;
  retry(
    kind: BacklogKind,
    name: string,
    note?: string,
  ): Promise<{ newExecutionId: string; parentExecutionId: string; status: string }>;
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
