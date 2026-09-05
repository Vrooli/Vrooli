/**
 * Backlog Service — Re-export shim for backwards compatibility
 *
 * The implementation has been decomposed into focused modules under ./backlog/.
 * This file re-exports everything so existing import sites continue to work.
 *
 * @see ./backlog/crud-service.ts    — list, get, create, update, delete
 * @see ./backlog/file-service.ts    — file tree operations
 * @see ./backlog/queue-service.ts   — queue & research
 * @see ./backlog/archive-service.ts — archive targets, modules, review
 * @see ./backlog/bulk-service.ts    — export, import, summaries
 */

export {
  backlogService,
  createBacklogService,
} from "./backlog";

export type {
  IBacklogService,
  QueueResponse,
  BacklogFileOperationResult,
  BacklogUpdatePatch,
  ImportBacklogResponse,
} from "./backlog";
