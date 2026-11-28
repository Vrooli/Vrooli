/**
 * Mock builders for `api/candidates` and `api/styles`. Canonical usage:
 *
 *   vi.mock("../../api/candidates", async (importOriginal) => {
 *     const actual = await importOriginal<typeof import("../../api/candidates")>();
 *     return { ...actual, ...makeCandidatesMocks() };
 *   });
 */
import { vi } from "vitest";

import {
  makeListCandidatesResponse,
  makePickCandidateResponse,
  makeRefineCandidateResponse,
  makeCandidate,
  type LogoCandidate,
} from "./factories";

export interface CandidatesMocks {
  candidatesClient: {
    listCandidates: ReturnType<typeof vi.fn>;
    exploreCandidates: ReturnType<typeof vi.fn>;
    refineCandidate: ReturnType<typeof vi.fn>;
    pickCandidate: ReturnType<typeof vi.fn>;
    rejectCandidate: ReturnType<typeof vi.fn>;
    restoreCandidate: ReturnType<typeof vi.fn>;
  };
  listCandidates: ReturnType<typeof vi.fn>;
  exploreCandidates: ReturnType<typeof vi.fn>;
  refineCandidate: ReturnType<typeof vi.fn>;
  pickCandidate: ReturnType<typeof vi.fn>;
  rejectCandidate: ReturnType<typeof vi.fn>;
  restoreCandidate: ReturnType<typeof vi.fn>;
  candidateThumbnailUrl: (id: string) => string;
  brandMarkUrl: (brandId: string) => string;
}

export const makeCandidatesMocks = (candidates: LogoCandidate[] = []): CandidatesMocks => {
  const listResponse = () => makeListCandidatesResponse({ candidates });
  const mocks: CandidatesMocks = {
    candidatesClient: {
      listCandidates: vi.fn().mockResolvedValue(listResponse()),
      exploreCandidates: vi.fn().mockResolvedValue({ candidates, warnings: [] }),
      refineCandidate: vi.fn().mockResolvedValue(makeRefineCandidateResponse()),
      pickCandidate: vi.fn().mockResolvedValue(makePickCandidateResponse()),
      rejectCandidate: vi.fn().mockResolvedValue({ candidate: makeCandidate() }),
      restoreCandidate: vi.fn().mockResolvedValue({ candidate: makeCandidate() }),
    },
    listCandidates: vi.fn().mockResolvedValue(candidates),
    exploreCandidates: vi.fn().mockResolvedValue({ candidates, warnings: [] }),
    refineCandidate: vi.fn().mockResolvedValue(makeRefineCandidateResponse().candidate),
    pickCandidate: vi.fn().mockResolvedValue(makePickCandidateResponse()),
    rejectCandidate: vi.fn().mockResolvedValue(makeCandidate()),
    restoreCandidate: vi.fn().mockResolvedValue(makeCandidate()),
    candidateThumbnailUrl: (id: string) => `/api/v1/brand-manager/candidates/thumbnail?id=${id}`,
    brandMarkUrl: (brandId: string) => `/api/v1/brand-manager/brands/mark?brand_id=${brandId}`,
  };
  return mocks;
};
