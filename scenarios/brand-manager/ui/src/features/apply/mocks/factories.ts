/**
 * Test data factories for the apply domain. Co-located with the feature so
 * deleting `features/apply/` takes the factories with it (no central residue).
 *
 * The response/action types are GENERATED proto messages; factories use
 * `create(<Schema>, overrides)` so field defaults match proto3 semantics and
 * adding a proto field is instantly available without editing this file.
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  ApplyActionSchema,
  ApplyResponseSchema,
  SkipReasonSchema,
  type ApplyAction,
  type ApplyResponse,
  type SkipReason,
} from "@vrooli/proto-types/brand-manager/v1/apply/apply_pb";

export type { ApplyAction, ApplyResponse, SkipReason };

export const makeApplyAction = (
  overrides: MessageInitShape<typeof ApplyActionSchema> = {},
): ApplyAction =>
  create(ApplyActionSchema, {
    type: "css",
    file: "ui/src/styles/brand.css",
    element: "colors",
    ...overrides,
  });

export const makeSkipReason = (
  overrides: MessageInitShape<typeof SkipReasonSchema> = {},
): SkipReason =>
  create(SkipReasonSchema, {
    element: "logo",
    reason: "no logo asset",
    ...overrides,
  });

export const makeApplyResponse = (
  overrides: MessageInitShape<typeof ApplyResponseSchema> = {},
): ApplyResponse =>
  create(ApplyResponseSchema, {
    scenario: "web-console",
    brandId: "brand-1",
    brandVersion: 1,
    dryRun: true,
    applied: [],
    skipped: [],
    ...overrides,
  });
