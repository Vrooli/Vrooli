/**
 * Test data factories for the workspace domain. Co-located with the feature
 * so deleting `features/workspace/` takes the factories with it (no central
 * residue).
 *
 * `ListOperationsResponse` is a GENERATED proto message; the factory uses
 * `create(<Schema>, overrides)` so it carries proto's reflection state and
 * proto3 field defaults. `runOp`'s return is a plain TS shape (object URL +
 * metadata), so its factories are plain objects.
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  ListOperationsResponseSchema,
  OperationInfoSchema,
  type ListOperationsResponse,
  type OperationInfo,
} from "@vrooli/proto-types/image-tools/v1/ops/ops_pb";

import type { RunOpImageResult, RunOpMetadataResult } from "../../../api/ops";

export type { ListOperationsResponse, OperationInfo };

export const makeOperationInfo = (
  overrides: MessageInitShape<typeof OperationInfoSchema> = {},
): OperationInfo =>
  create(OperationInfoSchema, {
    name: "resize",
    category: "geometry",
    summary: "Resize the image",
    ...overrides,
  });

export const makeListOperationsResponse = (
  overrides: MessageInitShape<typeof ListOperationsResponseSchema> = {},
): ListOperationsResponse =>
  create(ListOperationsResponseSchema, {
    operations: [
      makeOperationInfo({ name: "resize" }),
      makeOperationInfo({ name: "filter", category: "color", summary: "Apply a filter" }),
      makeOperationInfo({ name: "overlay", category: "compose", summary: "Overlay a watermark" }),
      makeOperationInfo({ name: "metadata", category: "metadata", summary: "Read metadata" }),
    ],
    decodableFormats: ["png", "jpeg", "webp"],
    encodableFormats: ["png", "jpeg", "webp"],
    ...overrides,
  });

export const makeRunOpImageResult = (
  overrides: Partial<RunOpImageResult> = {},
): RunOpImageResult => ({
  kind: "image",
  url: "blob:result",
  width: 256,
  height: 128,
  format: "png",
  jobId: "job-1",
  ...overrides,
});

export const makeRunOpMetadataResult = (
  overrides: Partial<RunOpMetadataResult> = {},
): RunOpMetadataResult => ({
  kind: "metadata",
  json: '{"width":256,"height":128}',
  ...overrides,
});
