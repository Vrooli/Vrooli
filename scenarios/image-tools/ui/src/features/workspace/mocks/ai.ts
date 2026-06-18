/**
 * Mock builders for the AI/Enhance boundary — co-located with the workspace
 * feature so deleting `features/workspace/` takes them with it.
 *
 * `makeAIMocks()` substitutes the `api/ai` discovery function (mirrors
 * `mocks/ops.ts`); `makeEnhanceClient()` is the injected `EnhanceClient` seam
 * fake so the whole AI job lifecycle runs without the network.
 */
import { vi } from "vitest";
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  AIOperationInfoSchema,
  ListAIOperationsResponseSchema,
  type AIOperationInfo,
  type ListAIOperationsResponse,
} from "@vrooli/proto-types/image-tools/v1/ai/ai_pb";

import type { AIImageResult } from "../../../api/ai";
import type { EnhanceClient, SelectedModel } from "../useEnhance";
import type { CreateClient } from "../useCreate";

export const makeAIOperationInfo = (
  overrides: MessageInitShape<typeof AIOperationInfoSchema> = {},
): AIOperationInfo =>
  create(AIOperationInfoSchema, {
    name: "background_removal",
    category: "enhancement",
    summary: "Remove the background to transparency",
    requiresImage: true,
    ...overrides,
  });

export const makeListAIOperationsResponse = (
  overrides: MessageInitShape<typeof ListAIOperationsResponseSchema> = {},
): ListAIOperationsResponse =>
  create(ListAIOperationsResponseSchema, {
    operations: [
      makeAIOperationInfo({ name: "background_removal" }),
      makeAIOperationInfo({ name: "upscale", summary: "Super-resolve / enlarge" }),
      makeAIOperationInfo({ name: "denoise", summary: "Reduce noise / deblur" }),
      makeAIOperationInfo({
        name: "naturalize",
        summary: "Reintroduce realistic texture/grain to over-smoothed images",
      }),
      makeAIOperationInfo({
        name: "text_to_image",
        category: "generation",
        summary: "Generate an image from a text prompt",
        requiresImage: false,
        promptDriven: true,
      }),
      makeAIOperationInfo({
        name: "image_to_image",
        category: "generation",
        summary: "Transform an input image guided by a prompt",
        requiresImage: true,
        promptDriven: true,
      }),
      makeAIOperationInfo({
        name: "inpaint",
        category: "generation",
        summary: "Regenerate a masked region from a prompt",
        requiresImage: true,
        requiresMask: true,
        promptDriven: true,
      }),
      makeAIOperationInfo({
        name: "object_removal",
        category: "generation",
        summary: "Remove a masked object and fill the gap",
        requiresImage: true,
        requiresMask: true,
      }),
    ],
    ...overrides,
  });

export interface AIMocks {
  listAIOperations: ReturnType<typeof vi.fn>;
}

export const makeAIMocks = (): AIMocks => ({
  listAIOperations: vi.fn().mockResolvedValue(makeListAIOperationsResponse()),
});

export const makeSelectedModel = (overrides: Partial<SelectedModel> = {}): SelectedModel => ({
  id: "rembg",
  name: "rembg",
  installed: true,
  sizeMb: 45,
  cpuCapable: true,
  gpuRequired: false,
  minVramGb: 0,
  speedNote: "~30s",
  gpuViable: false,
  reason: "CPU default",
  warnings: [],
  ...overrides,
});

export const makeAIImageResult = (overrides: Partial<AIImageResult> = {}): AIImageResult => ({
  url: "blob:enhanced",
  width: 200,
  height: 100,
  format: "png",
  outputFile: new File(["x"], "enhanced.png", { type: "image/png" }),
  ...overrides,
});

/**
 * A fully-stubbed `EnhanceClient`. By default it walks the happy path: model
 * installed → submit → one progress tick → terminal success → image result.
 * Override individual methods to exercise the install gate, failure, or cancel.
 */
export const makeEnhanceClient = (overrides: Partial<EnhanceClient> = {}): EnhanceClient => ({
  selectModel: vi.fn(() => Promise.resolve(makeSelectedModel())),
  submit: vi.fn(() => Promise.resolve({ jobId: "job-1", tier: "local-cpu", warnings: [] })),
  watch: vi.fn<EnhanceClient["watch"]>((_jobId, _signal, onEvent) => {
    onEvent({ percent: 60, message: "working", state: "running" });
    return Promise.resolve();
  }),
  result: vi.fn(() => Promise.resolve({ ok: true, resultRef: "out/result.png", error: "" })),
  fetchResult: vi.fn(() => Promise.resolve(makeAIImageResult())),
  install: vi.fn(() => Promise.resolve({ jobId: "install-1", alreadyInstalled: false })),
  waitJob: vi.fn(() => Promise.resolve({ ok: true, error: "" })),
  cancel: vi.fn(() => Promise.resolve()),
  ...overrides,
});

/**
 * A fully-stubbed `CreateClient`. By default it walks the happy path for a
 * single text-to-image variation: model installed → submit → one progress tick
 * → terminal success (`message` carries no `variations:` marker, so the primary
 * ref is the sole result) → image result. Override `result` with a
 * `variations: [k0 k1 …]` message + a per-key `fetchResult` to exercise the
 * N-variation grid, or override individual methods for the install gate /
 * failure / cancel paths.
 */
export const makeCreateClient = (overrides: Partial<CreateClient> = {}): CreateClient => ({
  selectModel: vi.fn(() => Promise.resolve(makeSelectedModel({ id: "sd-1.5", name: "sd-1.5" }))),
  submit: vi.fn(() => Promise.resolve({ jobId: "gen-1", tier: "local-cpu", warnings: [] })),
  watch: vi.fn<CreateClient["watch"]>((_jobId, _signal, onEvent) => {
    onEvent({ percent: 50, message: "produced 1/1", state: "running" });
    return Promise.resolve();
  }),
  result: vi.fn(() =>
    Promise.resolve({ ok: true, resultRef: "out/gen-0.png", message: "produced 1/1", error: "" }),
  ),
  fetchResult: vi.fn((ref: string) =>
    Promise.resolve(makeAIImageResult({ url: `blob:${ref}` })),
  ),
  install: vi.fn(() => Promise.resolve({ jobId: "install-1", alreadyInstalled: false })),
  waitJob: vi.fn(() => Promise.resolve({ ok: true, error: "" })),
  cancel: vi.fn(() => Promise.resolve()),
  ...overrides,
});
