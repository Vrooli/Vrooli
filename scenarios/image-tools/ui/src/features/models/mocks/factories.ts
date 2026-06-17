/**
 * Test data factories for the models domain. Co-located with the feature
 * so deleting `features/models/` takes the factories with it.
 *
 * Each `make<Domain>(overrides?)` returns a stable default instance that
 * tests selectively override via `MessageInitShape<Schema>`.
 *
 * # Wire shape lives in proto, not here
 *
 * The Model / ListModelsResponse types are GENERATED proto messages at
 * `packages/proto/gen/typescript/image-tools/v1/models/...`. Factories
 * use `create(<Schema>, overrides)` so field defaults match proto3
 * semantics and schema additions are instantly available.
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  ListModelsResponseSchema,
  ListOperationsResponseSchema,
  ModelSchema,
  SetModelEnabledResponseSchema,
  type ListModelsResponse,
  type ListOperationsResponse,
  type Model,
  type SetModelEnabledResponse,
} from "@vrooli/proto-types/image-tools/v1/models/models_pb";

export type { Model, ListModelsResponse, ListOperationsResponse, SetModelEnabledResponse };

export const makeModel = (overrides: MessageInitShape<typeof ModelSchema> = {}): Model =>
  create(ModelSchema, {
    id: "model-1",
    name: "Real-ESRGAN x4",
    operations: ["upscale"],
    defaultFor: ["upscale"],
    tier: "default",
    backend: "onnx",
    altBackends: [],
    requiresComfyui: false,
    sizeMbApprox: 64,
    quantVariants: [],
    enabled: true,
    ...overrides,
  });

export const makeListModelsResponse = (
  overrides: MessageInitShape<typeof ListModelsResponseSchema> = {},
): ListModelsResponse =>
  create(ListModelsResponseSchema, {
    models: [],
    ...overrides,
  });

export const makeListOperationsResponse = (
  overrides: MessageInitShape<typeof ListOperationsResponseSchema> = {},
): ListOperationsResponse =>
  create(ListOperationsResponseSchema, {
    operations: [],
    ...overrides,
  });

export const makeSetModelEnabledResponse = (
  overrides: MessageInitShape<typeof SetModelEnabledResponseSchema> = {},
): SetModelEnabledResponse =>
  create(SetModelEnabledResponseSchema, {
    model: makeModel(),
    ...overrides,
  });
