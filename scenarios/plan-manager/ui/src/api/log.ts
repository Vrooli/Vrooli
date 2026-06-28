import { createClient } from "@connectrpc/connect";
import { LogService } from "@vrooli/proto-types/plan-manager/v1/log/log_pb";
import {
  FindingTriage,
  LogEntryType,
  LogSeverity,
  LogSyncStatus,
  type GuidedStep,
  type LogEntry,
  type LogSummary,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";

import { transport } from "./client";

/**
 * Connect-Web client for the LogService — the single typed ledger for execution
 * log entries (decisions, candidate findings, bug reports, reusable records, and
 * notes). The operator console captures entries in-flow, lists/triages findings,
 * promotes them to bugs/records, and retries downstream sync. Each helper returns
 * the proto-typed shape. This replaces the decision/finding RPCs that used to live
 * on the ExecutionService.
 */
export const logClient = createClient(LogService, transport);

/** Optional fields shared by every Add* RPC. */
export interface AddEntryOptions {
  detail?: string;
  evidence?: string[];
  sourceCommand?: string;
  idempotencyKey?: string;
  runId?: string;
}

/** Add* options for finding/bug entries, which also carry a severity. */
export interface AddSeverityOptions extends AddEntryOptions {
  severity?: LogSeverity;
}

export interface AddEntryResult {
  entry: LogEntry | undefined;
  step: GuidedStep | undefined;
  deduplicated: boolean;
}

function baseFields(
  planOrExecution: string,
  phaseId: string,
  title: string,
  opts: AddEntryOptions,
) {
  return {
    planOrExecution,
    phaseId,
    title,
    detail: opts.detail ?? "",
    evidence: opts.evidence ?? [],
    sourceCommand: opts.sourceCommand ?? "",
    idempotencyKey: opts.idempotencyKey ?? "",
    runId: opts.runId ?? "",
  };
}

export async function addDecision(
  planOrExecution: string,
  phaseId: string,
  title: string,
  opts: AddEntryOptions = {},
): Promise<AddEntryResult> {
  const resp = await logClient.addDecision(baseFields(planOrExecution, phaseId, title, opts));
  return { entry: resp.entry, step: resp.step, deduplicated: resp.deduplicated };
}

export async function addFinding(
  planOrExecution: string,
  phaseId: string,
  title: string,
  opts: AddSeverityOptions = {},
): Promise<AddEntryResult> {
  const resp = await logClient.addFinding({
    ...baseFields(planOrExecution, phaseId, title, opts),
    severity: opts.severity ?? LogSeverity.UNSPECIFIED,
  });
  return { entry: resp.entry, step: resp.step, deduplicated: resp.deduplicated };
}

export async function addBug(
  planOrExecution: string,
  phaseId: string,
  title: string,
  opts: AddSeverityOptions = {},
): Promise<AddEntryResult> {
  const resp = await logClient.addBug({
    ...baseFields(planOrExecution, phaseId, title, opts),
    severity: opts.severity ?? LogSeverity.UNSPECIFIED,
  });
  return { entry: resp.entry, step: resp.step, deduplicated: resp.deduplicated };
}

export async function addRecord(
  planOrExecution: string,
  phaseId: string,
  title: string,
  opts: AddEntryOptions = {},
): Promise<AddEntryResult> {
  const resp = await logClient.addRecord(baseFields(planOrExecution, phaseId, title, opts));
  return { entry: resp.entry, step: resp.step, deduplicated: resp.deduplicated };
}

export async function addNote(
  planOrExecution: string,
  phaseId: string,
  title: string,
  opts: AddEntryOptions = {},
): Promise<AddEntryResult> {
  const resp = await logClient.addNote(baseFields(planOrExecution, phaseId, title, opts));
  return { entry: resp.entry, step: resp.step, deduplicated: resp.deduplicated };
}

export interface ListEntriesFilter {
  planOrExecution?: string;
  phaseId?: string;
  type?: LogEntryType;
  triage?: FindingTriage;
  syncStatus?: LogSyncStatus;
}

export async function listEntries(
  filter: ListEntriesFilter = {},
): Promise<{ entries: LogEntry[]; summary: LogSummary | undefined; step: GuidedStep | undefined }> {
  const resp = await logClient.listEntries({
    planOrExecution: filter.planOrExecution ?? "",
    phaseId: filter.phaseId ?? "",
    type: filter.type ?? LogEntryType.UNSPECIFIED,
    triage: filter.triage ?? FindingTriage.UNSPECIFIED,
    syncStatus: filter.syncStatus ?? LogSyncStatus.UNSPECIFIED,
  });
  return { entries: resp.entries, summary: resp.summary, step: resp.step };
}

export async function getEntry(
  id: string,
): Promise<{ entry: LogEntry | undefined; step: GuidedStep | undefined }> {
  const resp = await logClient.getEntry({ id });
  return { entry: resp.entry, step: resp.step };
}

export interface UpdateEntryArgs {
  id: string;
  title?: string;
  detail?: string;
  severity?: LogSeverity;
  triage?: FindingTriage;
  addEvidence?: string[];
}

export async function updateEntry(
  args: UpdateEntryArgs,
): Promise<{ entry: LogEntry | undefined; step: GuidedStep | undefined }> {
  const resp = await logClient.updateEntry({
    id: args.id,
    title: args.title ?? "",
    detail: args.detail ?? "",
    severity: args.severity ?? LogSeverity.UNSPECIFIED,
    triage: args.triage ?? FindingTriage.UNSPECIFIED,
    addEvidence: args.addEvidence ?? [],
  });
  return { entry: resp.entry, step: resp.step };
}

export interface PromoteEntryArgs {
  id: string;
  toType: LogEntryType;
  title?: string;
  detail?: string;
  severity?: LogSeverity;
}

export async function promoteEntry(
  args: PromoteEntryArgs,
): Promise<{ entry: LogEntry | undefined; source: LogEntry | undefined; step: GuidedStep | undefined }> {
  const resp = await logClient.promoteEntry({
    id: args.id,
    toType: args.toType,
    title: args.title ?? "",
    detail: args.detail ?? "",
    severity: args.severity ?? LogSeverity.UNSPECIFIED,
  });
  return { entry: resp.entry, source: resp.source, step: resp.step };
}

export async function syncEntry(
  id: string,
): Promise<{ entry: LogEntry | undefined; step: GuidedStep | undefined }> {
  const resp = await logClient.syncEntry({ id });
  return { entry: resp.entry, step: resp.step };
}
