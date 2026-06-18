import { createClient } from "@connectrpc/connect";
import { create, fromJson, toJsonString } from "@bufbuild/protobuf";
import {
  AIService,
  AIParamsSchema,
  SubmitAIResponseSchema,
  type AIOperationInfo,
  type ListAIOperationsResponse,
  type SubmitAIResponse,
} from "@vrooli/proto-types/image-tools/v1/ai/ai_pb";

import {
  decodeApiError,
  fetchBlob,
  PROTO_READ_OPTIONS,
  transport,
  uploadFile,
  type JsonValue,
} from "./client";

/**
 * Typed Connect client for AI discovery. Execution is the REST multipart submit
 * edge (`POST /api/v1/ai/{op}`) — image/mask bytes can't ride a proto field —
 * so it goes through `submitAI` below, not this client. The op runs on the
 * durable job queue; callers watch progress via `JobsService` and pull the
 * output blob by the job's `result_ref`.
 */
export const aiClient = createClient(AIService, transport);

/** Discovery wrapper — lists the AI op catalog (name/category/summary/flags). */
export const listAIOperations = (): Promise<ListAIOperationsResponse> =>
  aiClient.listAIOperations({});

/**
 * The AIParams fields the UI sets; everything else takes the server default.
 * `seed` is an int64 (protojson string on the wire); the rest map 1:1.
 */
export type AIParamsInput = Partial<{
  prompt: string;
  negativePrompt: string;
  seed: bigint;
  width: number;
  height: number;
  steps: number;
  cfgScale: number;
  variations: number;
  strength: number;
  scale: number;
  realism: number;
  faceAware: boolean;
  modelOverride: string;
  allowByok: boolean;
  autoScanNsfw: boolean;
}>;

/** Parsed `SubmitAIResponse`: the job to watch plus the resolved plan. */
export interface SubmitAIResult {
  jobId: string;
  estimatedSeconds: number;
  modelId: string;
  tier: string;
  warnings: string[];
}

/**
 * Submit one AI op. Builds the multipart request (`file` input bytes when the
 * op needs an image, optional `mask`, and `params` as AIParams protojson) and
 * returns the parsed submit response (job id + ETA + resolved model/tier +
 * fallback warnings). A non-2xx (e.g. 409 "model not installed") becomes a
 * typed `ApiError` the caller branches on.
 */
export async function submitAI(
  operation: string,
  params: AIParamsInput,
  opts: { image?: File; mask?: File } = {},
): Promise<SubmitAIResult> {
  const formData = new FormData();
  if (opts.image) {
    formData.append("file", opts.image);
  }
  if (opts.mask) {
    formData.append("mask", opts.mask);
  }
  formData.append("params", toJsonString(AIParamsSchema, create(AIParamsSchema, params)));

  const res = await uploadFile(`/ai/${operation}`, formData);
  if (!res.ok) {
    throw await decodeApiError(res);
  }
  const json = (await res.json()) as JsonValue;
  const parsed: SubmitAIResponse = fromJson(SubmitAIResponseSchema, json, PROTO_READ_OPTIONS);
  return {
    jobId: parsed.jobId,
    estimatedSeconds: parsed.estimatedSeconds,
    modelId: parsed.modelId,
    tier: parsed.tier,
    warnings: parsed.warnings,
  };
}

/** An AI op's output image, materialized from the job's `result_ref` blob. */
export interface AIImageResult {
  url: string;
  width: number;
  height: number;
  format: string;
  /** The bytes as a File so a follow-up op can compose on top. */
  outputFile: File;
}

const formatFromRef = (ref: string, mime: string): string => {
  const ext = ref.split(".").pop();
  if (ext && ext.length <= 5 && !ext.includes("/")) {
    return ext.toLowerCase();
  }
  const sub = mime.split("/")[1];
  return sub ? sub.toLowerCase() : "png";
};

/** Decode an object URL just far enough to read its natural dimensions. */
const imageSize = (url: string): Promise<{ width: number; height: number }> =>
  new Promise((resolve) => {
    const img = new Image();
    img.onload = () => resolve({ width: img.naturalWidth, height: img.naturalHeight });
    img.onerror = () => resolve({ width: 0, height: 0 });
    img.src = url;
  });

/**
 * Fetch an AI op's result image by its job `result_ref`. Returns an object URL
 * for the canvas, the natural dimensions (so upscale can show the new size),
 * and the bytes as a File so the result composes as the next history step.
 */
export async function fetchAIResult(resultRef: string): Promise<AIImageResult> {
  const blob = await fetchBlob(resultRef);
  const url = URL.createObjectURL(blob);
  const { width, height } = await imageSize(url);
  const format = formatFromRef(resultRef, blob.type);
  const outputFile = new File([blob], `enhanced.${format || "png"}`, {
    type: blob.type || "image/png",
  });
  return { url, width, height, format, outputFile };
}

export type { AIOperationInfo, ListAIOperationsResponse };
