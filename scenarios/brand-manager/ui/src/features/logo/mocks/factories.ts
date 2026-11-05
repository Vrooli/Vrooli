/**
 * Test data factories for the logo feature. Co-located so deleting
 * `features/logo/` removes them. Proto messages are built with `create(Schema)`
 * so proto3 defaults match and new fields flow through automatically.
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import {
  CandidateOrigin,
  CandidateStatus,
  type LogoCandidate,
  LogoCandidateSchema,
  type ListCandidatesResponse,
  ListCandidatesResponseSchema,
  type PickCandidateResponse,
  PickCandidateResponseSchema,
  type RefineCandidateResponse,
  RefineCandidateResponseSchema,
} from "@vrooli/proto-types/brand-manager/v1/candidates/candidates_pb";

export type { LogoCandidate, ListCandidatesResponse, PickCandidateResponse, RefineCandidateResponse };

export const makeCandidate = (
  overrides: MessageInitShape<typeof LogoCandidateSchema> = {},
): LogoCandidate =>
  create(LogoCandidateSchema, {
    id: "candidate-1",
    brandId: "brand-1",
    assetId: "asset-1",
    mediaType: "image/svg+xml",
    concept: "constellation eagle line art",
    prompt: "white eagle line art on a transparent background",
    role: "image.vector.default",
    model: "recraft-v4.1-vector",
    origin: CandidateOrigin.GENERATED,
    status: CandidateStatus.PROPOSED,
    createdAt: timestampFromDate(new Date("2026-09-15T12:00:00.000Z")),
    ...overrides,
  });

export const makeListCandidatesResponse = (
  overrides: MessageInitShape<typeof ListCandidatesResponseSchema> = {},
): ListCandidatesResponse =>
  create(ListCandidatesResponseSchema, { candidates: [makeCandidate()], ...overrides });

export const makePickCandidateResponse = (
  overrides: MessageInitShape<typeof PickCandidateResponseSchema> = {},
): PickCandidateResponse =>
  create(PickCandidateResponseSchema, {
    candidate: makeCandidate({ status: CandidateStatus.PICKED }),
    markAssetId: "asset-mark",
    vectorized: false,
    ...overrides,
  });

export const makeRefineCandidateResponse = (
  overrides: MessageInitShape<typeof RefineCandidateResponseSchema> = {},
): RefineCandidateResponse =>
  create(RefineCandidateResponseSchema, {
    candidate: makeCandidate({ id: "candidate-2", parentId: "candidate-1", origin: CandidateOrigin.EDITED }),
    ...overrides,
  });
