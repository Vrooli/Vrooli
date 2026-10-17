import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { useEffect, useState } from "react";

import {
  candidateImageURL,
  listStyles,
  listSurfaces,
  submitRender,
  type Candidate,
  type RenderJob,
  type RenderRequest,
  type StyleFilter,
  type Style,
  type Surface,
} from "../api/studio";

/**
 * Query keys are exported so a test can seed the cache without repeating the
 * key shape, and so an invalidation in one page cannot miss a key another page
 * spelled differently.
 */
export const studioKeys = {
  styles: (filter: StyleFilter) => ["styles", filter] as const,
  surfaces: () => ["surfaces"] as const,
  render: (req: RenderRequest) =>
    ["render", req.styleId, req.surfaceId, req.placement ?? "", req.seed.toString(), req.candidateCount ?? 1] as const,
};

export function useStyles(filter: StyleFilter = {}): UseQueryResult<Style[]> {
  return useQuery({ queryKey: studioKeys.styles(filter), queryFn: () => listStyles(filter) });
}

export function useSurfaces(): UseQueryResult<Surface[]> {
  return useQuery({ queryKey: studioKeys.surfaces(), queryFn: listSurfaces });
}

/**
 * useRender submits one render and caches it by everything that decides the
 * pixels.
 *
 * A render is deterministic in (style, surface, placement, seed, count), which
 * is what makes caching it correct rather than merely convenient: the same key
 * cannot produce different bytes, so a cached result is not stale, it is the
 * answer. `enabled` lets a page hold the request until the operator has chosen
 * a surface.
 */
export function useRender(req: RenderRequest | null): UseQueryResult<RenderJob> {
  return useQuery({
    queryKey: req ? studioKeys.render(req) : ["render", "idle"],
    queryFn: () => submitRender(req as RenderRequest),
    enabled: req !== null,
    // A model-backed render takes tens of seconds and costs real GPU time.
    // Retrying it on failure spends that again to produce the same failure.
    retry: false,
    staleTime: Infinity,
  });
}

/**
 * useObjectURL owns the lifetime of a candidate's blob URL.
 *
 * Without the revoke, every re-render of a variation grid leaks a
 * megabyte-scale blob, and a session spent comparing seeds walks the tab into
 * hundreds of megabytes of retained image data with nothing visibly wrong.
 */
export function useObjectURL(candidate: Candidate | undefined): string | undefined {
  const [url, setURL] = useState<string | undefined>(undefined);
  useEffect(() => {
    if (!candidate || candidate.imagePng.length === 0) {
      setURL(undefined);
      return;
    }
    const next = candidateImageURL(candidate);
    setURL(next);
    return () => URL.revokeObjectURL(next);
  }, [candidate]);
  return url;
}

/**
 * The distinct values present on each taxonomy axis, derived from the catalog
 * rather than restated from the enum.
 *
 * Deriving them is what keeps a filter honest: an axis value with no style
 * behind it would otherwise appear as a facet that always returns nothing, and
 * this scenario's whole subject-coherence repair was about not offering choices
 * the system cannot honour.
 */
export type AxisValues = {
  role: string[];
  subject: string[];
  treatment: string[];
  lineage: string[];
  placement: string[];
};

export function axisValues(styles: Style[]): AxisValues {
  const collect = (pick: (s: Style) => string[]): string[] =>
    Array.from(new Set(styles.flatMap(pick))).filter(Boolean).sort();
  return {
    role: collect((s) => [s.role]),
    subject: collect((s) => [s.subject]),
    treatment: collect((s) => s.treatments),
    lineage: collect((s) => [s.lineage]),
    placement: collect((s) => s.placements),
  };
}
