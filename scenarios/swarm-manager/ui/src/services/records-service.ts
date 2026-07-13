/**
 * Records Service - Data access for the records domain.
 *
 * DOC: docs/internal/SEAMS.md (Records Store Boundary)
 */

import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type {
  RecordCreateInput,
  RecordCaptureInput,
  RecordCaptureResult,
  RecordCaptureMetadata,
  RecordItem,
  RecordKind,
  RecordListFilter,
  RecordNarrativeInput,
  RecordOutcome,
  RecordSearchHit,
} from "../types";

function mapRecord(raw: Record<string, unknown>): RecordItem {
  const captureRaw = raw.capture as Record<string, unknown> | undefined;
  return {
    id: (raw.id as string) ?? "",
    kind: (raw.kind as RecordKind) ?? "fix",
    scenario: (raw.scenario as string) ?? "",
    backlogRef: (raw.backlog_ref as string) || undefined,
    initiativeId: (raw.initiative_id as string) || undefined,
    supersedes: (raw.supersedes as string) || undefined,
    supersededBy: (raw.superseded_by as string) || undefined,
    trigger: (raw.trigger as string) ?? "",
    approach: (raw.approach as string) ?? "",
    ruledOut: (raw.ruled_out as string[]) ?? [],
    evidence: (raw.evidence as string) || undefined,
    commit: (raw.commit as string) || undefined,
    filesChanged: (raw.files_changed as string[]) ?? [],
    outcome: (raw.outcome as RecordOutcome) ?? "shipped",
    stub: Boolean(raw.stub),
    draft: Boolean(raw.draft),
    capture: captureRaw ? mapCaptureMetadata(captureRaw) : undefined,
    createdAt: (raw.created_at as string) ?? "",
    createdBy: (raw.created_by as string) || undefined,
    narrativeAt: (raw.narrative_at as string) || undefined,
  };
}

function mapCaptureMetadata(raw: Record<string, unknown>): RecordCaptureMetadata {
  return {
    raw: raw.raw as Record<string, string> | undefined,
    accepted: raw.accepted as Record<string, string> | undefined,
    needs: (raw.needs as string[]) ?? [],
    invalid: (raw.invalid as RecordCaptureMetadata["invalid"]) ?? [],
    warnings: (raw.warnings as string[]) ?? [],
  };
}

function mapCaptureResult(raw: Record<string, unknown>): RecordCaptureResult {
  return {
    disposition: raw.disposition === "published" ? "published" : "draft",
    record: mapRecord((raw.record as Record<string, unknown>) ?? {}),
    accepted: (raw.accepted as Record<string, string>) ?? {},
    needs: (raw.needs as string[]) ?? [],
    invalid: (raw.invalid as RecordCaptureResult["invalid"]) ?? [],
    warnings: (raw.warnings as string[]) ?? [],
    nextAction: (raw.next_action as string[]) ?? [],
  };
}

export interface IRecordsService {
  list(filter?: RecordListFilter): Promise<RecordItem[]>;
  get(id: string): Promise<RecordItem>;
  create(input: RecordCreateInput): Promise<RecordItem>;
  capture(input: RecordCaptureInput): Promise<RecordCaptureResult>;
  repairCapture(id: string, input: RecordCaptureInput): Promise<RecordCaptureResult>;
  fillNarrative(id: string, input: RecordNarrativeInput): Promise<RecordItem>;
  supersede(id: string, successorId: string, reason?: string): Promise<RecordItem>;
  search(query: string, opts?: { kind?: RecordKind; scenario?: string; limit?: number }): Promise<RecordSearchHit[]>;
}

function buildListPath(filter?: RecordListFilter): string {
  if (!filter) return API_ENDPOINTS.records;
  const params = new URLSearchParams();
  if (filter.scenario) params.set("scenario", filter.scenario);
  if (filter.kind) params.set("kind", filter.kind);
  if (filter.backlogRef) params.set("backlog_ref", filter.backlogRef);
  if (filter.includeStubs) params.set("include_stubs", "true");
  if (filter.limit && filter.limit > 0) params.set("limit", String(filter.limit));
  if (filter.offset && filter.offset > 0) params.set("offset", String(filter.offset));
  const q = params.toString();
  return q ? `${API_ENDPOINTS.records}?${q}` : API_ENDPOINTS.records;
}

export function createRecordsService(client: IApiClient = defaultApiClient): IRecordsService {
  return {
    async list(filter) {
      const resp = await client.get<{ records: Record<string, unknown>[] }>(buildListPath(filter));
      return (resp.records ?? []).map(mapRecord);
    },
    async get(id) {
      const resp = await client.get<{ record: Record<string, unknown> }>(API_ENDPOINTS.recordById(id));
      return mapRecord(resp.record);
    },
    async create(input) {
      const resp = await client.post<{ record: Record<string, unknown> }>(API_ENDPOINTS.records, {
        kind: input.kind,
        scenario: input.scenario,
        backlog_ref: input.backlogRef ?? "",
        initiative_id: input.initiativeId ?? "",
        supersedes: input.supersedes ?? "",
        trigger: input.trigger,
        approach: input.approach,
        ruled_out: input.ruledOut,
        commit: input.commit ?? "",
        files_changed: input.filesChanged ?? [],
        outcome: input.outcome,
        created_by: input.createdBy ?? "",
      });
      return mapRecord(resp.record);
    },
    async capture(input) {
      const resp = await client.post<Record<string, unknown>>(API_ENDPOINTS.recordsCapture, capturePayload(input));
      return mapCaptureResult(resp);
    },
    async repairCapture(id, input) {
      const resp = await client.patch<Record<string, unknown>>(API_ENDPOINTS.recordCapture(id), capturePayload(input));
      return mapCaptureResult(resp);
    },
    async fillNarrative(id, input) {
      const resp = await client.patch<{ record: Record<string, unknown> }>(API_ENDPOINTS.recordNarrative(id), {
        trigger: input.trigger,
        approach: input.approach,
        ruled_out: input.ruledOut,
        commit: input.commit ?? "",
        files_changed: input.filesChanged ?? [],
        outcome: input.outcome,
      });
      return mapRecord(resp.record);
    },
    async supersede(id, successorId, reason) {
      const resp = await client.post<{ record: Record<string, unknown> }>(API_ENDPOINTS.recordSupersede(id), {
        successor_id: successorId,
        reason: reason ?? "",
      });
      return mapRecord(resp.record);
    },
    async search(query, opts) {
      const resp = await client.post<{ hits: Array<{ record: Record<string, unknown>; score: number }> }>(
        API_ENDPOINTS.recordSearch,
        {
          query,
          kind: opts?.kind ?? "",
          scenario: opts?.scenario ?? "",
          limit: opts?.limit ?? 10,
        },
      );
      return (resp.hits ?? []).map((h) => ({ record: mapRecord(h.record), score: h.score }));
    },
  };
}

function capturePayload(input: RecordCaptureInput) {
  return {
    kind: input.kind,
    scenario: input.scenario,
    trigger: input.trigger,
    approach: input.approach,
    evidence: input.evidence ?? "",
    ruled_out: input.ruledOut,
    outcome: input.outcome,
    created_by: input.createdBy ?? "",
    idempotency_key: input.idempotencyKey ?? "",
  };
}

export const recordsService = createRecordsService();
