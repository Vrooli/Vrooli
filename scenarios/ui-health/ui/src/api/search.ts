// API client for the search domain. Thin wrapper over the generated
// SearchService Connect client; exports plain TS types so callers stay
// decoupled from generated message shapes.
import { createClient } from "@connectrpc/connect";

import {
  Mode as ProtoMode,
  SearchService,
  SurfaceKind as ProtoSurfaceKind,
  type SearchResult as ProtoSearchResult,
  type SearchResponse as ProtoSearchResponse,
  type StatusResponse as ProtoStatusResponse,
} from "@vrooli/proto-types/ui-health/v1/search/search_pb";
import { Provenance as ProtoProvenance } from "@vrooli/proto-types/ui-health/v1/contracts/provenance/provenance_pb";

import { transport } from "./client";

export const searchClient = createClient(SearchService, transport);

export type SearchMode = "ai" | "text" | "unspecified";

export type SurfaceKind =
  | "component"
  | "page"
  | "feature"
  | "hook"
  | "layout"
  | "other"
  | "unspecified";

export const SURFACE_KIND_FILTERS = [
  "all",
  "component",
  "page",
  "feature",
  "hook",
  "layout",
  "other",
] as const;

export type SurfaceKindFilter = (typeof SURFACE_KIND_FILTERS)[number];

export type ProvenanceTag =
  | "custom"
  | "adopted-unmodified"
  | "adopted-modified"
  | "unknown"
  | "unspecified";

export type SearchHit = {
  scenario: string;
  slot: string;
  kind: SurfaceKind;
  displayName: string;
  description: string;
  filePath: string;
  score: number;
  provenance: ProvenanceTag;
  library: string;
  componentName: string;
};

export type SearchResults = {
  hits: SearchHit[];
  modeUsed: SearchMode;
};

export type SearchStatus = {
  available: boolean;
  ollama: boolean;
  qdrant: boolean;
  indexedCount: number;
  lastReconcileAt: string;
  lastReconcileOutcome: string;
};

export async function searchSurfaces(
  query: string,
  limit = 25,
): Promise<SearchResults> {
  const resp = await searchClient.search({
    query,
    limit,
    mode: ProtoMode.UNSPECIFIED,
  });
  return resultsFromProto(resp);
}

export async function searchStatus(): Promise<SearchStatus> {
  const resp = await searchClient.status({});
  return statusFromProto(resp);
}

export function surfaceKindFromProto(k: ProtoSurfaceKind): SurfaceKind {
  switch (k) {
    case ProtoSurfaceKind.COMPONENT:
      return "component";
    case ProtoSurfaceKind.PAGE:
      return "page";
    case ProtoSurfaceKind.FEATURE:
      return "feature";
    case ProtoSurfaceKind.HOOK:
      return "hook";
    case ProtoSurfaceKind.LAYOUT:
      return "layout";
    case ProtoSurfaceKind.OTHER:
      return "other";
    default:
      return "unspecified";
  }
}

function modeFromProto(m: ProtoMode): SearchMode {
  switch (m) {
    case ProtoMode.AI:
      return "ai";
    case ProtoMode.TEXT:
      return "text";
    default:
      return "unspecified";
  }
}

function provenanceFromProto(p: ProtoProvenance | undefined): ProvenanceTag {
  switch (p) {
    case ProtoProvenance.CUSTOM:
      return "custom";
    case ProtoProvenance.ADOPTED_UNMODIFIED:
      return "adopted-unmodified";
    case ProtoProvenance.ADOPTED_MODIFIED:
      return "adopted-modified";
    case ProtoProvenance.UNKNOWN:
      return "unknown";
    default:
      return "unspecified";
  }
}

function hitFromProto(r: ProtoSearchResult): SearchHit {
  return {
    scenario: r.scenario,
    slot: r.slot,
    kind: surfaceKindFromProto(r.kind),
    displayName: r.displayName,
    description: r.description,
    filePath: r.filePath,
    score: r.score,
    provenance: provenanceFromProto(r.provenance?.provenance),
    library: r.provenance?.library ?? "",
    componentName: r.provenance?.componentName ?? "",
  };
}

function resultsFromProto(p: ProtoSearchResponse): SearchResults {
  return {
    hits: p.results.map(hitFromProto),
    modeUsed: modeFromProto(p.modeUsed),
  };
}

function statusFromProto(p: ProtoStatusResponse): SearchStatus {
  return {
    available: p.available,
    ollama: p.ollama,
    qdrant: p.qdrant,
    indexedCount: p.indexedCount,
    lastReconcileAt: p.lastReconcileAt,
    lastReconcileOutcome: p.lastReconcileOutcome,
  };
}

export function filterHits(
  hits: SearchHit[],
  kind: SurfaceKindFilter,
): SearchHit[] {
  if (kind === "all") return hits;
  return hits.filter((h) => h.kind === kind);
}
