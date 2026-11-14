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
 * answer. A null request holds the query until the operator has chosen a
 * surface.
 *
 * `enabled: false` reads the cache without submitting. The key is still the
 * request's own, so a specimen rendered earlier in the session — on the style
 * page, or by an earlier click — shows immediately, while nothing new is
 * spent. The catalog relies on this: a model-backed tile must not start a
 * render just because the page was opened.
 *
 * `retain` keeps a finished render for the rest of the session instead of the
 * default five minutes. It is for model-backed renders, which cost minutes to
 * redo; a procedural specimen is cheaper to redraw than to hold.
 *
 * The query's abort signal reaches the request, so leaving the page cancels
 * the render rather than leaving it to finish for nobody — the API passes the
 * cancellation on to the image-tools job it submitted.
 */
export function useRender(
  req: RenderRequest | null,
  options: { enabled?: boolean; retain?: boolean } = {},
): UseQueryResult<RenderJob> {
  return useQuery({
    queryKey: req ? studioKeys.render(req) : ["render", "idle"],
    queryFn: ({ signal }) => submitRender(req as RenderRequest, signal),
    enabled: req !== null && (options.enabled ?? true),
    ...(options.retain ? { gcTime: Infinity } : {}),
    // A model-backed render takes minutes on a CPU host and costs real compute.
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
