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
  RecordItem,
  RecordKind,
  RecordListFilter,
  RecordNarrativeInput,
  RecordOutcome,
  RecordSearchHit,
} from "../types";

function mapRecord(raw: Record<string, unknown>): RecordItem {
  return {
    id: (raw.id as string) ?? "",
    kind: (raw.kind as RecordKind) ?? "fix",
    scenario: (raw.scenario as string) ?? "",
    backlogRef: (raw.backlog_ref as string) || undefined,
    supersedes: (raw.supersedes as string) || undefined,
    supersededBy: (raw.superseded_by as string) || undefined,
    trigger: (raw.trigger as string) ?? "",
    approach: (raw.approach as string) ?? "",
    ruledOut: (raw.ruled_out as string[]) ?? [],
    commit: (raw.commit as string) || undefined,
    filesChanged: (raw.files_changed as string[]) ?? [],
    outcome: (raw.outcome as RecordOutcome) ?? "shipped",
    stub: Boolean(raw.stub),
    createdAt: (raw.created_at as string) ?? "",
    createdBy: (raw.created_by as string) || undefined,
    narrativeAt: (raw.narrative_at as string) || undefined,
  };
}

export interface IRecordsService {
  list(filter?: RecordListFilter): Promise<RecordItem[]>;
  get(id: string): Promise<RecordItem>;
  create(input: RecordCreateInput): Promise<RecordItem>;
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

export const recordsService = createRecordsService();
