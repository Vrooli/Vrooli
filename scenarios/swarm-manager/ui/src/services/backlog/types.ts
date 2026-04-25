/**
 * Backlog Service Types
 *
 * Shared type definitions used across backlog service modules.
 */

import type { BacklogFile, BacklogItem, BacklogKind, BlockingReason, ClarificationThread, ItemBlockingInfo } from "../../types";

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
  "title" | "description" | "status" | "priority" | "tags" | "dependsOn" | "initiative" | "effort" | "acceptanceAllow" | "acceptanceDeny" | "note"
>>;

/** Result of auto-advance decision from the workshop save endpoint. */
export interface WorkshopAutoAdvance {
  triggered: boolean;
  runId?: string;
  taskId?: string;
  reason: string;
  nextMode?: "workshop" | "finalize";
  /** Whether an advance is pending (countdown active, not yet spawned). */
  pending?: boolean;
  /** When the pending advance will fire (RFC 3339 timestamp). */
  advanceAt?: string;
  /** Configured delay in seconds. */
  delaySeconds?: number;
}

/** Response from saving a workshop round via the dedicated endpoint. */
export interface WorkshopSaveResponse {
  file: BacklogFile;
  autoAdvance: WorkshopAutoAdvance;
}

export interface WorkshopDeleteRoundResponse {
  deletedRound: number;
  remainingRounds: number;
}

export interface WorkshopResetResponse {
  deletedRounds: number;
  statusReverted: boolean;
}

export interface ImportBacklogResponse {
  dryRun: boolean;
  changes: Array<{ item: string; action: string; details: string[] }>;
  errors: string[];
  summary: string;
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
  research(
    kind: BacklogKind,
    name: string,
    payload?: {
      prompt?: string;
      projectRoot?: string;
      mode?: string;
      contextPaths?: string[];
      contextTargetIds?: string[];
      contextRequirementIds?: string[];
      confirm?: boolean;
      force?: boolean;
    }
  ): Promise<import("../../types").ResearchResponse>;
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
  getFeedbackSummary(): Promise<import("../../types").FeedbackSummaryResponse>;
  getMaturitySummary(): Promise<import("../../types").MaturitySummaryResponse>;
  getPendingQuestions(): Promise<import("../../types").PendingQuestionsResponse>;
  workshopSave(
    kind: BacklogKind,
    name: string,
    roundNumber: number,
    content: string,
  ): Promise<WorkshopSaveResponse>;
  workshopDeleteRound(
    kind: BacklogKind,
    name: string,
    roundNumber: number,
  ): Promise<WorkshopDeleteRoundResponse>;
  workshopReset(
    kind: BacklogKind,
    name: string,
  ): Promise<WorkshopResetResponse>;
  workshopCancelPendingAdvance(
    kind: BacklogKind,
    name: string,
  ): Promise<{ cancelled: boolean }>;
  createClarification(
    kind: BacklogKind,
    name: string,
    roundNumber: number,
    itemId: string,
    message?: string,
    files?: File[],
  ): Promise<{ thread: ClarificationThread }>;
  getClarification(
    kind: BacklogKind,
    name: string,
    threadId: string,
  ): Promise<{ thread: ClarificationThread }>;
  continueClarification(
    kind: BacklogKind,
    name: string,
    threadId: string,
    message: string,
    files?: File[],
  ): Promise<{ thread: ClarificationThread }>;
  clarificationAction(
    kind: BacklogKind,
    name: string,
    threadId: string,
    action: string,
    updatedItemJson?: string,
  ): Promise<{ action: string; success: boolean; message: string; run_id?: string; task_id?: string }>;
}
