import { createClient } from "@connectrpc/connect";
import { buildApiUrl } from "@vrooli/api-base";
import {
  CandidatesService,
  CandidateOrigin,
  CandidateStatus,
  type ExploreCandidatesResponse,
  type ListCandidatesResponse,
  type LogoCandidate,
  type PickCandidateResponse,
  type RefineCandidateResponse,
  type VectorizeOptions,
} from "@vrooli/proto-types/brand-manager/v1/candidates/candidates_pb";

import { API_BASE, transport } from "./client";

export const candidatesClient = createClient(CandidatesService, transport);

/** listCandidates returns a brand's candidates, newest-first. */
export async function listCandidates(
  brandId: string,
  options: { status?: CandidateStatus; limit?: number; offset?: number } = {},
): Promise<LogoCandidate[]> {
  const resp: ListCandidatesResponse = await candidatesClient.listCandidates({
    brandId,
    status: options.status ?? CandidateStatus.UNSPECIFIED,
    limit: options.limit ?? 48,
    offset: options.offset ?? 0,
  });
  return resp.candidates;
}

/**
 * exploreCandidates fans out N concepts × M variations in one request. A style
 * reference brand (id or slug) renders every concept to match that brand's
 * approved mark, which keeps a product line one family.
 */
export async function exploreCandidates(input: {
  brandId: string;
  brief: string;
  concepts: string[];
  variations: number;
  preferVector: boolean;
  styleReferenceBrand?: string;
}): Promise<ExploreCandidatesResponse> {
  return candidatesClient.exploreCandidates({
    brandId: input.brandId,
    brief: input.brief,
    concepts: input.concepts,
    variations: input.variations,
    preferVector: input.preferVector,
    styleReferenceBrand: input.styleReferenceBrand ?? "",
  });
}

/** refineCandidate derives a NEW candidate from a source and returns it. */
export async function refineCandidate(
  candidateId: string,
  action:
    | { instruction: string }
    | { maskAssetId: string }
    | { removeBackground: true }
    | { vectorize: Partial<VectorizeOptions> },
): Promise<LogoCandidate> {
  let request: Parameters<typeof candidatesClient.refineCandidate>[0] = { candidateId };
  if ("instruction" in action) {
    request = { candidateId, action: { case: "instruction", value: action.instruction } };
  } else if ("maskAssetId" in action) {
    request = { candidateId, action: { case: "maskAssetId", value: action.maskAssetId } };
  } else if ("removeBackground" in action) {
    request = { candidateId, action: { case: "removeBackground", value: true } };
  } else {
    request = {
      candidateId,
      action: {
        case: "vectorize",
        value: {
          colors: action.vectorize.colors ?? 0,
          keepColors: action.vectorize.keepColors ?? [],
          dropBackgroundLayers: action.vectorize.dropBackgroundLayers ?? false,
          clipToLargestRoundedRegion: action.vectorize.clipToLargestRoundedRegion ?? false,
          insetPx: action.vectorize.insetPx ?? 0,
          tolerancePx: action.vectorize.tolerancePx ?? 0,
          smoothing: action.vectorize.smoothing ?? false,
          minAreaPx: action.vectorize.minAreaPx ?? 0,
        },
      },
    };
  }
  const resp: RefineCandidateResponse = await candidatesClient.refineCandidate(request);
  if (!resp.candidate) {
    throw new Error("refine returned no candidate");
  }
  return resp.candidate;
}

/** pickCandidate promotes a candidate; a raster pick is vectorized server-side. */
export async function pickCandidate(candidateId: string): Promise<PickCandidateResponse> {
  return candidatesClient.pickCandidate({ candidateId });
}

/** rejectCandidate marks a candidate rejected with an optional note. */
export async function rejectCandidate(candidateId: string, note = ""): Promise<LogoCandidate> {
  const resp = await candidatesClient.rejectCandidate({ candidateId, note });
  if (!resp.candidate) {
    throw new Error("reject returned no candidate");
  }
  return resp.candidate;
}

/** restoreCandidate returns a rejected candidate to proposed. */
export async function restoreCandidate(candidateId: string): Promise<LogoCandidate> {
  const resp = await candidatesClient.restoreCandidate({ candidateId });
  if (!resp.candidate) {
    throw new Error("restore returned no candidate");
  }
  return resp.candidate;
}

/** candidateThumbnailUrl is the raw endpoint that serves the candidate's bytes. */
export function candidateThumbnailUrl(candidateId: string): string {
  return buildApiUrl(
    `/api/v1/brand-manager/candidates/thumbnail?id=${encodeURIComponent(candidateId)}`,
    { baseUrl: API_BASE },
  );
}

/** brandMarkUrl is the raw endpoint that serves the brand's picked mark bytes. */
export function brandMarkUrl(brandId: string): string {
  return buildApiUrl(
    `/api/v1/brand-manager/brands/mark?brand_id=${encodeURIComponent(brandId)}`,
    { baseUrl: API_BASE },
  );
}

export { CandidateOrigin, CandidateStatus };
export type { ExploreCandidatesResponse, ListCandidatesResponse, LogoCandidate, PickCandidateResponse, VectorizeOptions };
