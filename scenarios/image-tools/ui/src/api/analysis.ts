import { createClient } from "@connectrpc/connect";
import {
  AnalysisService,
  AnalyzeResponseSchema,
  type AnalysisOperationInfo,
  type ListAnalysisOperationsResponse,
} from "@vrooli/proto-types/image-tools/v1/analysis/analysis_pb";

import {
  decodeApiError,
  fromJson,
  makeApiError,
  PROTO_READ_OPTIONS,
  transport,
  uploadFile,
  type JsonValue,
} from "./client";

/**
 * Typed Connect client for analysis discovery. Execution is the REST multipart
 * edge (`POST /api/v1/analysis/{op}`) — the image bytes can't ride a proto
 * field — so it goes through `analyze` below. Analysis ops run *synchronously*
 * (the response carries the structured result directly, plus a recorded
 * `jobId` for observability), unlike the durable AI job lifecycle.
 */
export const analysisClient = createClient(AnalysisService, transport);

/** Discovery wrapper — lists the analysis op catalog (name/summary/model-backed). */
export const listAnalysisOperations = (): Promise<ListAnalysisOperationsResponse> =>
  analysisClient.listAnalysisOperations({});

/** A pixel-space rectangle (x,y = top-left), in the analyzed image's coords. */
export interface AnalyzeBox {
  x: number;
  y: number;
  width: number;
  height: number;
}

/** One recognized OCR text region with its confidence and (optional) box. */
export interface AnalyzeOcrBlock {
  text: string;
  confidence: number;
  box: AnalyzeBox | null;
}

/** Structured `ocr` output: full text + per-region blocks + detected language. */
export interface AnalyzeOcr {
  kind: "ocr";
  jobId: string;
  fullText: string;
  language: string;
  blocks: AnalyzeOcrBlock[];
}

/** One classifier label with its score. */
export interface AnalyzeCategory {
  label: string;
  score: number;
}

/** Structured `nsfw_classify` output: verdict + score + per-label categories. */
export interface AnalyzeNsfw {
  kind: "nsfw";
  jobId: string;
  flagged: boolean;
  score: number;
  label: string;
  threshold: number;
  categories: AnalyzeCategory[];
}

/** One extracted palette swatch. */
export interface AnalyzeColor {
  hex: string;
  fraction: number;
}

/** Structured pure-Go `probe` output: dimensions, codec facts, EXIF, palette. */
export interface AnalyzeProbe {
  kind: "probe";
  jobId: string;
  width: number;
  height: number;
  format: string;
  colorModel: string;
  hasAlpha: boolean;
  frameCount: number;
  megapixels: number;
  sizeBytes: number;
  hasExif: boolean;
  hasGps: boolean;
  orientation: number;
  dominantColors: AnalyzeColor[];
}

/** Structured pure-Go `duplicate_detect` output: perceptual fingerprints. */
export interface AnalyzeDuplicate {
  kind: "duplicate";
  jobId: string;
  phashHex: string;
  ahashHex: string;
  hashBits: number;
}

/** Structured pure-Go `quality_assessment` output: no-reference quality scores. */
export interface AnalyzeQuality {
  kind: "quality";
  jobId: string;
  overallScore: number;
  sharpness: number;
  blurry: boolean;
  brightness: number;
  contrast: number;
  exposure: string;
  notes: string[];
}

/** The normalized result of one analysis op, discriminated by `kind`. */
export type AnalyzeResult =
  | AnalyzeOcr
  | AnalyzeNsfw
  | AnalyzeProbe
  | AnalyzeDuplicate
  | AnalyzeQuality;

const boxOf = (box: { x: number; y: number; width: number; height: number } | undefined): AnalyzeBox | null =>
  box ? { x: box.x, y: box.y, width: box.width, height: box.height } : null;

/**
 * Run one analysis op against `file`. Builds the multipart request, posts to
 * the sync analyze edge, and parses the `AnalyzeResponse` oneof into the
 * normalized discriminated union the UI renders. A non-2xx (e.g. 409 "model
 * not installed" for the model-backed ocr/nsfw ops) becomes a typed `ApiError`
 * the caller branches on. Validation/parsing lives here at the boundary so the
 * panel only ever sees a well-formed `AnalyzeResult` or a typed error.
 */
export async function analyze(operation: string, file: File): Promise<AnalyzeResult> {
  const formData = new FormData();
  formData.append("file", file);

  const res = await uploadFile(`/analysis/${operation}`, formData);
  if (!res.ok) {
    throw await decodeApiError(res);
  }
  const json = (await res.json()) as JsonValue;
  const parsed = fromJson(AnalyzeResponseSchema, json, PROTO_READ_OPTIONS);
  const jobId = parsed.jobId;

  switch (parsed.result.case) {
    case "ocr": {
      const ocr = parsed.result.value;
      return {
        kind: "ocr",
        jobId,
        fullText: ocr.fullText,
        language: ocr.language,
        blocks: ocr.blocks.map((b) => ({
          text: b.text,
          confidence: b.confidence,
          box: boxOf(b.box),
        })),
      };
    }
    case "nsfw": {
      const nsfw = parsed.result.value;
      return {
        kind: "nsfw",
        jobId,
        flagged: nsfw.nsfw,
        score: nsfw.score,
        label: nsfw.label,
        threshold: nsfw.threshold,
        categories: nsfw.categories.map((c) => ({ label: c.label, score: c.score })),
      };
    }
    case "probe": {
      const probe = parsed.result.value;
      return {
        kind: "probe",
        jobId,
        width: probe.width,
        height: probe.height,
        format: probe.format,
        colorModel: probe.colorModel,
        hasAlpha: probe.hasAlpha,
        frameCount: probe.frameCount,
        megapixels: probe.megapixels,
        sizeBytes: Number(probe.sizeBytes),
        hasExif: probe.hasExif,
        hasGps: probe.hasGps,
        orientation: probe.orientation,
        dominantColors: probe.dominantColors.map((d) => ({ hex: d.hex, fraction: d.fraction })),
      };
    }
    case "duplicate": {
      const dup = parsed.result.value;
      return {
        kind: "duplicate",
        jobId,
        phashHex: dup.phashHex,
        ahashHex: dup.ahashHex,
        hashBits: dup.hashBits,
      };
    }
    case "quality": {
      const q = parsed.result.value;
      return {
        kind: "quality",
        jobId,
        overallScore: q.overallScore,
        sharpness: q.sharpness,
        blurry: q.blurry,
        brightness: q.brightness,
        contrast: q.contrast,
        exposure: q.exposure,
        notes: q.notes,
      };
    }
    default:
      throw makeApiError("internal", `analyze response for ${operation} carried no result`);
  }
}

export type { AnalysisOperationInfo, ListAnalysisOperationsResponse };
