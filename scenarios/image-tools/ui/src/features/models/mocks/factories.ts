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
  AddCustomModelResponseSchema,
  BlocklistEntrySchema,
  CapabilityLabelsSchema,
  CommercialUse,
  HardwareSchema,
  InstallModelResponseSchema,
  InstallStateSchema,
  ListBlocklistResponseSchema,
  ListDefaultsResponseSchema,
  ListModelsResponseSchema,
  ListOperationsResponseSchema,
  ModelSchema,
  OpDefaultSchema,
  RemoveModelResponseSchema,
  SetDefaultModelResponseSchema,
  SetModelEnabledResponseSchema,
  type AddCustomModelResponse,
  type BlocklistEntry,
  type InstallModelResponse,
  type ListBlocklistResponse,
  type ListDefaultsResponse,
  type ListModelsResponse,
  type ListOperationsResponse,
  type Model,
  type OpDefault,
  type RemoveModelResponse,
  type SetDefaultModelResponse,
  type SetModelEnabledResponse,
} from "@vrooli/proto-types/image-tools/v1/models/models_pb";

export type {
  Model,
  ListModelsResponse,
  ListOperationsResponse,
  SetModelEnabledResponse,
  InstallModelResponse,
  RemoveModelResponse,
  AddCustomModelResponse,
  SetDefaultModelResponse,
  ListDefaultsResponse,
  ListBlocklistResponse,
  BlocklistEntry,
  OpDefault,
};

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
    custom: false,
    hardware: create(HardwareSchema, {
      cpuCapable: true,
      gpuRequired: false,
      minVramGb: 0,
      minRamGb: 4,
    }),
    capabilityLabels: create(CapabilityLabelsSchema, {
      nsfwCapable: false,
      license: "BSD-3-Clause",
      commercialUse: CommercialUse.YES,
    }),
    install: create(InstallStateSchema, { installed: false }),
    ...overrides,
  });

export const makeInstallModelResponse = (
  overrides: MessageInitShape<typeof InstallModelResponseSchema> = {},
): InstallModelResponse =>
  create(InstallModelResponseSchema, {
    jobId: "job-install-1",
    etaSeconds: 30,
    sizeMbApprox: 64,
    alreadyInstalled: false,
    ...overrides,
  });

export const makeRemoveModelResponse = (
  overrides: MessageInitShape<typeof RemoveModelResponseSchema> = {},
): RemoveModelResponse => create(RemoveModelResponseSchema, { removed: true, ...overrides });

export const makeAddCustomModelResponse = (
  overrides: MessageInitShape<typeof AddCustomModelResponseSchema> = {},
): AddCustomModelResponse =>
  create(AddCustomModelResponseSchema, {
    model: makeModel({ id: "custom-1", custom: true }),
    ...overrides,
  });

export const makeSetDefaultModelResponse = (
  overrides: MessageInitShape<typeof SetDefaultModelResponseSchema> = {},
): SetDefaultModelResponse =>
  create(SetDefaultModelResponseSchema, { operation: "upscale", modelId: "model-1", ...overrides });

export const makeOpDefault = (
  overrides: MessageInitShape<typeof OpDefaultSchema> = {},
): OpDefault =>
  create(OpDefaultSchema, { operation: "upscale", modelId: "model-1", source: "seed", ...overrides });

export const makeListDefaultsResponse = (
  overrides: MessageInitShape<typeof ListDefaultsResponseSchema> = {},
): ListDefaultsResponse => create(ListDefaultsResponseSchema, { defaults: [], ...overrides });

export const makeBlocklistEntry = (
  overrides: MessageInitShape<typeof BlocklistEntrySchema> = {},
): BlocklistEntry =>
  create(BlocklistEntrySchema, {
    id: "blocked-1",
    operations: ["upscale"],
    license: "Proprietary",
    reason: "License forbids redistribution",
    exportingOnnxRemovesRestriction: false,
    ...overrides,
  });

export const makeListBlocklistResponse = (
  overrides: MessageInitShape<typeof ListBlocklistResponseSchema> = {},
): ListBlocklistResponse => create(ListBlocklistResponseSchema, { entries: [], ...overrides });

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
