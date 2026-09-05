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
  ArchitectureInferenceSchema,
  ImportModelResponseSchema,
  InspectModelSourceResponseSchema,
  BackendReadinessSchema,
  BackendStatusSchema,
  BlocklistEntrySchema,
  CandidateModelSchema,
  CapabilityLabelsSchema,
  CommercialUse,
  DoctorBackendsResponseSchema,
  EnsureBackendResponseSchema,
  HardwareSchema,
  HostSummarySchema,
  InstallModelResponseSchema,
  InstallStateSchema,
  ListBlocklistResponseSchema,
  ListDefaultsResponseSchema,
  ListModelsResponseSchema,
  ListOperationModelsResponseSchema,
  ListOperationsResponseSchema,
  ModelFitSchema,
  ExplainResolutionResponseSchema,
  ModelSchema,
  OpDefaultSchema,
  RemoveModelResponseSchema,
  ResolutionSchema,
  SelectModelResponseSchema,
  SetDefaultModelResponseSchema,
  SetModelEnabledResponseSchema,
  type AddCustomModelResponse,
  type ImportModelResponse,
  type InspectModelSourceResponse,
  type BackendReadiness,
  type BackendStatus,
  type BlocklistEntry,
  type CandidateModel,
  type DoctorBackendsResponse,
  type EnsureBackendResponse,
  type HostSummary,
  type InstallModelResponse,
  type ListBlocklistResponse,
  type ListDefaultsResponse,
  type ListModelsResponse,
  type ListOperationModelsResponse,
  type ListOperationsResponse,
  type Model,
  type ModelFit,
  type OpDefault,
  type ExplainResolutionResponse,
  type RemoveModelResponse,
  type Resolution,
  type SelectModelResponse,
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
  BackendStatus,
  DoctorBackendsResponse,
  EnsureBackendResponse,
  ListOperationModelsResponse,
  CandidateModel,
  ModelFit,
  BackendReadiness,
  HostSummary,
  SelectModelResponse,
};

export const makeBackendStatus = (
  overrides: MessageInitShape<typeof BackendStatusSchema> = {},
): BackendStatus =>
  create(BackendStatusSchema, {
    name: "realesrgan-ncnn-vulkan",
    operations: ["upscale", "denoise"],
    available: false,
    standalone: true,
    cloud: false,
    gpuCapable: true,
    detail: "not found on PATH",
    provision: "vrooli host install realesrgan-ncnn-vulkan",
    hostTool: "realesrgan-ncnn-vulkan",
    hostToolReady: false,
    remediation: "vrooli host install realesrgan-ncnn-vulkan",
    ...overrides,
  });

export const makeDoctorBackendsResponse = (
  overrides: MessageInitShape<typeof DoctorBackendsResponseSchema> = {},
): DoctorBackendsResponse =>
  create(DoctorBackendsResponseSchema, { ok: false, backends: [], ...overrides });

export const makeEnsureBackendResponse = (
  overrides: MessageInitShape<typeof EnsureBackendResponseSchema> = {},
): EnsureBackendResponse =>
  create(EnsureBackendResponseSchema, {
    tool: "realesrgan-ncnn-vulkan",
    jobId: "ensure-job-1",
    etaSeconds: 90,
    alreadyInstalled: false,
    manual: false,
    state: "would_install",
    detail: "",
    ...overrides,
  });

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

// --- Model-picker fixtures (host-aware candidate menu behind every AI action) ---

export const makeHostSummary = (
  overrides: MessageInitShape<typeof HostSummarySchema> = {},
): HostSummary =>
  create(HostSummarySchema, {
    hasGpu: true,
    gpuName: "NVIDIA RTX 4090",
    gpuCount: 1,
    vramTotalGb: 24,
    vramFreeGb: 20,
    vramKnown: true,
    ramGb: 64,
    cpuCores: 16,
    os: "linux",
    arch: "amd64",
    ...overrides,
  });

export const makeModelFit = (
  overrides: MessageInitShape<typeof ModelFitSchema> = {},
): ModelFit =>
  create(ModelFitSchema, {
    runnable: true,
    gpuViable: true,
    fitClass: "gpu",
    vramShortfallGb: 0,
    warnings: [],
    ...overrides,
  });

export const makeBackendReadiness = (
  overrides: MessageInitShape<typeof BackendReadinessSchema> = {},
): BackendReadiness =>
  create(BackendReadinessSchema, {
    backend: "onnx",
    hostTool: "realesrgan-ncnn-vulkan",
    ready: true,
    installTier: "auto",
    remediation: "",
    manualHint: "",
    detail: "",
    ...overrides,
  });

export const makeCandidateModel = (
  overrides: MessageInitShape<typeof CandidateModelSchema> = {},
): CandidateModel =>
  create(CandidateModelSchema, {
    model: makeModel({ id: "cand-1", name: "Real-ESRGAN x4" }),
    fit: makeModelFit(),
    backend: makeBackendReadiness(),
    readyState: "ready",
    selected: true,
    ...overrides,
  });

export const makeListOperationModelsResponse = (
  overrides: MessageInitShape<typeof ListOperationModelsResponseSchema> = {},
): ListOperationModelsResponse =>
  create(ListOperationModelsResponseSchema, {
    operation: "upscale",
    host: makeHostSummary(),
    candidates: [makeCandidateModel()],
    selectedId: "cand-1",
    selectedReason: "best fit for this host",
    ...overrides,
  });

export const makeSelectModelResponse = (
  overrides: MessageInitShape<typeof SelectModelResponseSchema> = {},
): SelectModelResponse =>
  create(SelectModelResponseSchema, {
    model: makeModel({ id: "cand-1", name: "Real-ESRGAN x4" }),
    gpuViable: true,
    reason: "best fit for this host",
    warnings: [],
    ...overrides,
  });

export const makeResolution = (
  overrides: MessageInitShape<typeof ResolutionSchema> = {},
): Resolution =>
  create(ResolutionSchema, {
    operation: "upscale",
    modelId: "cand-1",
    modelName: "Real-ESRGAN x4",
    support: "native",
    technique: "",
    caveat: "",
    weight: "none",
    tier: "local-cpu",
    gpuViable: false,
    warnings: [],
    ...overrides,
  });

export const makeExplainResolutionResponse = (
  overrides: MessageInitShape<typeof ExplainResolutionResponseSchema> = {},
): ExplainResolutionResponse =>
  create(ExplainResolutionResponseSchema, {
    resolution: makeResolution(),
    ...overrides,
  });

export const makeInspectModelSourceResponse = (
  overrides: MessageInitShape<typeof InspectModelSourceResponseSchema> = {},
): InspectModelSourceResponse =>
  create(InspectModelSourceResponseSchema, {
    source: "stabilityai/stable-diffusion-xl-base-1.0",
    repoId: "stabilityai/stable-diffusion-xl-base-1.0",
    revision: "462165984030d82259a11f4367a4eed129e94a7b",
    layout: 2,
    architecture: create(ArchitectureInferenceSchema, {
      architecture: "sdxl",
      confidence: "high",
      evidence: 'pipeline class "StableDiffusionXLPipeline"',
    }),
    license: "openrail++",
    nsfw: false,
    sizeBytes: BigInt(5_167_000_600),
    pipelineClass: "StableDiffusionXLPipeline",
    offeredOperations: ["text_to_image", "image_to_image", "inpaint"],
    proposed: makeModel({ id: "imported-stable-diffusion-xl-base-1-0", backend: "diffusers", custom: true }),
    ...overrides,
  });

export const makeImportModelResponse = (
  overrides: MessageInitShape<typeof ImportModelResponseSchema> = {},
): ImportModelResponse =>
  create(ImportModelResponseSchema, {
    model: makeModel({ id: "imported-sdxl", backend: "diffusers", custom: true }),
    jobId: "job-import-1",
    etaSeconds: 120,
    alreadyInstalled: false,
    ...overrides,
  });

