/**
 * Test data factories for the generation domain. Co-located with the feature so
 * deleting `features/generation/` takes the factories with it (no central
 * residue).
 *
 * The response/status types are GENERATED proto messages; factories use
 * `create(<Schema>, overrides)` so field defaults match proto3 semantics and
 * adding a proto field is instantly available without editing this file.
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  GetImageBackendStatusResponseSchema,
  GetProviderStatusResponseSchema,
  ImageOperationStatusSchema,
  ProviderStatusSchema,
  type GetImageBackendStatusResponse,
  type GetProviderStatusResponse,
  type ImageOperationStatus,
  type ProviderStatus,
} from "@vrooli/proto-types/brand-manager/v1/generation/generation_pb";

export type { GetImageBackendStatusResponse, GetProviderStatusResponse, ImageOperationStatus, ProviderStatus };

export const makeProviderStatus = (
  overrides: MessageInitShape<typeof ProviderStatusSchema> = {},
): ProviderStatus => create(ProviderStatusSchema, { name: "ollama", available: true, ...overrides });

export const makeProviderStatusResponse = (
  overrides: MessageInitShape<typeof GetProviderStatusResponseSchema> = {},
): GetProviderStatusResponse =>
  create(GetProviderStatusResponseSchema, {
    available: false,
    providers: [],
    ...overrides,
  });

export const makeImageOperationStatus = (
  overrides: MessageInitShape<typeof ImageOperationStatusSchema> = {},
): ImageOperationStatus =>
  create(ImageOperationStatusSchema, {
    operation: "generate",
    ready: true,
    modelId: "sd-1.5",
    tier: "local-gpu",
    hint: "",
    ...overrides,
  });

export const makeImageBackendStatusResponse = (
  overrides: MessageInitShape<typeof GetImageBackendStatusResponseSchema> = {},
): GetImageBackendStatusResponse =>
  create(GetImageBackendStatusResponseSchema, {
    available: false,
    detail: "",
    operations: [],
    ...overrides,
  });
