import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });

export type Coverage = "NOW" | "IN-REACH" | "MISSING" | "UNREGISTERED";
export type Trust = "VALID" | "CACHED" | "UNAVAILABLE" | "UNTRUSTED";
export type Empirical = "NONE" | "PENDING" | "HIT" | "MISS" | "UNMEASURABLE";

export interface SourceBinding {
  team?: string;
  binding?: string;
  read?: string;
  select?: string;
  ttlSeconds?: number;
  instrumentStatus?: string;
  instrumentArchetype?: string;
}

export interface Sample {
  value: number;
  series: number[];
  basis: string;
}

export interface Reading {
  id: string;
  label: string;
  description?: string;
  unit?: string;
  format?: string;
  source: SourceBinding;
  coverage: Coverage;
  trust: Trust;
  trustReason?: string;
  empirical: Empirical;
  value: number | null;
  observedAt: string | null;
  ttlSeconds: number;
  target: { direction: string; bar: number | null; barRef?: string } | null;
  owner: string | null;
  whatIsNeeded: string | null;
  firstObservedMissing: string | null;
  gapOpenDays: number | null;
  sample: Sample | null;
  prediction: { target: number; direction: string; remainingHorizonSeconds: number } | null;
}

export interface SourceMetadata {
  from_cache: boolean;
  staleness_ts: string | null;
}

export interface BoardRoom {
  id: string;
  title: string;
  category?: string;
  theme?: string;
  composition?: string;
  metricIds?: string[];
}

export interface BoardSource {
  name: string;
  team: string;
  instrumentStatus: string;
  instrumentArchetype: string;
  readable: boolean;
  reason: string;
}

export interface BoardResponse {
  schemaVersion: string;
  generatedAt: string;
  rooms: BoardRoom[];
  denominator: { outcomeCategories: number; confidence: string; rationale: string };
  sources: BoardSource[];
}

export interface RoomResponse {
  room: BoardRoom;
  readings: Reading[];
  sources: Record<string, SourceMetadata>;
}

export type FocusKind = "untrusted-reading" | "source-unavailable" | "no-instrument" | "no-pipeline" | "unregistered-outcome";

export interface FocusEntry {
  kind: FocusKind;
  owner: string;
  reason: string;
  metricId?: string;
  rankReason: string;
}

export interface FocusResponse {
  generatedAt: string;
  entries: FocusEntry[];
}

export interface OpenLoopSelfEntry {
  id: string;
  reason: string;
  firstObservedMissing: string;
  gapOpenDays: number;
}

export interface OpenLoopResponse {
  generatedAt: string;
  missing: Reading[];
  unregistered: Reading[];
  self: OpenLoopSelfEntry[];
}

async function getJSON<T>(path: string): Promise<T> {
  const url = buildApiUrl(path, { baseUrl: API_BASE });
  const res = await fetch(url, { headers: { Accept: "application/json" }, cache: "no-store" });
  if (!res.ok) {
    throw new Error(`API request failed: ${res.status.toString()} ${res.statusText}`);
  }
  return res.json() as Promise<T>;
}

export const fetchBoard = (): Promise<BoardResponse> => getJSON("/board");
export const fetchRoom = (id: string, samples: string): Promise<RoomResponse> => getJSON(`/rooms/${id}?samples=${samples}`);
export const fetchFocus = (): Promise<FocusResponse> => getJSON("/focus");
export const fetchOpenLoop = (): Promise<OpenLoopResponse> => getJSON("/open-loop");

export const hasValue = (reading: Pick<Reading, "value">): reading is Reading & { value: number } =>
  typeof reading.value === "number" && Number.isFinite(reading.value);
