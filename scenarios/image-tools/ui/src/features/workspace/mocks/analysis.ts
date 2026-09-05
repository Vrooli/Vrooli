/**
 * Mock builders for the Analyze boundary — co-located with the workspace
 * feature so deleting `features/workspace/` takes them with it.
 *
 * `makeAnalysisMocks()` substitutes the `api/analysis` discovery function
 * (mirrors `mocks/ai.ts`); `makeAnalyzeClient()` is the injected `AnalyzeClient`
 * seam fake so the whole analysis flow runs without the network. The result
 * builders return the normalized `AnalyzeResult` shapes the panel renders.
 */
import { vi } from "vitest";
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  AnalysisOperationInfoSchema,
  ListAnalysisOperationsResponseSchema,
  type AnalysisOperationInfo,
  type ListAnalysisOperationsResponse,
} from "@vrooli/proto-types/image-tools/v1/analysis/analysis_pb";

import type {
  AnalyzeDuplicate,
  AnalyzeNsfw,
  AnalyzeOcr,
  AnalyzeProbe,
  AnalyzeQuality,
  AnalyzeResult,
} from "../../../api/analysis";
import type { AnalyzeClient } from "../useAnalyze";
import { makeSelectedModel } from "./ai";

export const makeAnalysisOperationInfo = (
  overrides: MessageInitShape<typeof AnalysisOperationInfoSchema> = {},
): AnalysisOperationInfo =>
  create(AnalysisOperationInfoSchema, {
    name: "probe",
    summary: "Read image metadata",
    modelBacked: false,
    defaultModelId: "",
    ...overrides,
  });

export const makeListAnalysisOperationsResponse = (
  overrides: MessageInitShape<typeof ListAnalysisOperationsResponseSchema> = {},
): ListAnalysisOperationsResponse =>
  create(ListAnalysisOperationsResponseSchema, {
    operations: [
      makeAnalysisOperationInfo({ name: "probe", summary: "Read image metadata", modelBacked: false }),
      makeAnalysisOperationInfo({
        name: "ocr",
        summary: "Extract text",
        modelBacked: true,
        defaultModelId: "tesseract",
      }),
      makeAnalysisOperationInfo({
        name: "nsfw_classify",
        summary: "Safety check",
        modelBacked: true,
        defaultModelId: "nsfw-classifier",
      }),
      makeAnalysisOperationInfo({
        name: "duplicate_detect",
        summary: "Find duplicates",
        modelBacked: false,
      }),
      makeAnalysisOperationInfo({
        name: "quality_assessment",
        summary: "Assess quality",
        modelBacked: false,
      }),
    ],
    ...overrides,
  });

export interface AnalysisMocks {
  listAnalysisOperations: ReturnType<typeof vi.fn>;
}

export const makeAnalysisMocks = (): AnalysisMocks => ({
  listAnalysisOperations: vi.fn().mockResolvedValue(makeListAnalysisOperationsResponse()),
});

export const makeProbeResult = (overrides: Partial<AnalyzeProbe> = {}): AnalyzeProbe => ({
  kind: "probe",
  jobId: "job-probe",
  width: 640,
  height: 480,
  format: "png",
  colorModel: "rgba",
  hasAlpha: true,
  frameCount: 1,
  megapixels: 0.31,
  sizeBytes: 12_345,
  hasExif: false,
  hasGps: false,
  orientation: 0,
  dominantColors: [{ hex: "#ff8800", fraction: 0.42 }],
  ...overrides,
});

export const makeOcrResult = (overrides: Partial<AnalyzeOcr> = {}): AnalyzeOcr => ({
  kind: "ocr",
  jobId: "job-ocr",
  fullText: "Hello world",
  language: "eng",
  blocks: [{ text: "Hello", confidence: 0.95, box: { x: 10, y: 12, width: 80, height: 20 } }],
  ...overrides,
});

export const makeNsfwResult = (overrides: Partial<AnalyzeNsfw> = {}): AnalyzeNsfw => ({
  kind: "nsfw",
  jobId: "job-nsfw",
  flagged: false,
  score: 0.03,
  label: "sfw",
  threshold: 0.5,
  categories: [
    { label: "sfw", score: 0.97 },
    { label: "nsfw", score: 0.03 },
  ],
  ...overrides,
});

export const makeDuplicateResult = (
  overrides: Partial<AnalyzeDuplicate> = {},
): AnalyzeDuplicate => ({
  kind: "duplicate",
  jobId: "job-dup",
  phashHex: "f0e1d2c3b4a59687",
  ahashHex: "0011223344556677",
  hashBits: 64,
  ...overrides,
});

export const makeQualityResult = (overrides: Partial<AnalyzeQuality> = {}): AnalyzeQuality => ({
  kind: "quality",
  jobId: "job-quality",
  overallScore: 0.82,
  sharpness: 0.71,
  blurry: false,
  brightness: 0.55,
  contrast: 0.48,
  exposure: "balanced",
  notes: ["well exposed"],
  ...overrides,
});

/**
 * A fully-stubbed `AnalyzeClient`. By default it walks the pure-Go probe happy
 * path (no model gate → result). Override `selectModel` (installed:false) +
 * `analyze` to exercise the model-install gate, or `analyze` to reject for the
 * failure path.
 */
export const makeAnalyzeClient = (overrides: Partial<AnalyzeClient> = {}): AnalyzeClient => ({
  selectModel: vi.fn(() => Promise.resolve(makeSelectedModel({ id: "tesseract", name: "tesseract" }))),
  analyze: vi.fn((): Promise<AnalyzeResult> => Promise.resolve(makeProbeResult())),
  install: vi.fn(() => Promise.resolve({ jobId: "install-1", alreadyInstalled: false })),
  waitJob: vi.fn(() => Promise.resolve({ ok: true, error: "" })),
  ...overrides,
});
