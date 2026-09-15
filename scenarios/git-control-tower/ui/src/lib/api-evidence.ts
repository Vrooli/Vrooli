import { buildApiUrl } from "@vrooli/api-base";
import type {
  GetRunResponse,
  ListEvidenceResponse,
  ListRunsResponse,
} from "@vrooli/proto-types/git-control-tower/v1/evidence/evidence_pb";
import type { StartRunResponse } from "@vrooli/proto-types/test-genie/v1/runs/runs_pb";
import { evidenceClient } from "./connect";
import { API_BASE } from "./api-internals";

export interface RunFilters {
  status?: string;
  search?: string;
  provider?: string;
  phaseClass?: string;
  dimension?: string;
  limit?: number;
  offset?: number;
}

export interface ArtifactFilters {
  kinds?: string[];
  search?: string;
  limit?: number;
  offset?: number;
}

export interface EvidenceFilters extends ArtifactFilters {
  runStatus?: string;
  runLimit?: number;
}

export interface StartRunInput {
  scenario: string;
  preset?: string;
  phases?: string[];
  skip?: string[];
  failFast?: boolean;
  diagnosticsPreset?: string;
  captureProfile?: string;
}

export function startRun(input: StartRunInput): Promise<StartRunResponse> {
  return evidenceClient.startRun(input);
}

export function listRuns(scenario: string, filters: RunFilters = {}): Promise<ListRunsResponse> {
  return evidenceClient.listRuns({
    scenario,
    status: filters.status,
    search: filters.search,
    provider: filters.provider,
    phaseClass: filters.phaseClass,
    dimension: filters.dimension,
    limit: filters.limit,
    offset: filters.offset,
  });
}

export function getRun(scenario: string, runId: string, filters: ArtifactFilters = {}): Promise<GetRunResponse> {
  return evidenceClient.getRun({
    scenario,
    runId,
    artifactKinds: filters.kinds,
    artifactSearch: filters.search,
    artifactLimit: filters.limit,
    artifactOffset: filters.offset,
  });
}

export function listEvidence(scenario: string, filters: EvidenceFilters = {}): Promise<ListEvidenceResponse> {
  return evidenceClient.listEvidence({
    scenario,
    kinds: filters.kinds,
    search: filters.search,
    runStatus: filters.runStatus,
    limit: filters.limit,
    offset: filters.offset,
    runLimit: filters.runLimit,
  });
}

/** Same-origin access to opaque Test Genie artifact bytes. */
export function runArtifactUrl(scenario: string, runId: string, artifactId: string): string {
  const query = new URLSearchParams({ scenario }).toString();
  return buildApiUrl(
    `/repo/test-runs/${encodeURIComponent(runId)}/artifacts/${encodeURIComponent(artifactId)}?${query}`,
    { baseUrl: API_BASE },
  );
}
