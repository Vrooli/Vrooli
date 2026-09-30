import { createClient } from "@connectrpc/connect";
import { create, fromJson, toJsonString } from "@bufbuild/protobuf";
import {
  DiffMode,
  DiffParamsSchema,
  DiffResultSchema,
  DiffService,
  type DiffModeInfo,
  type DiffParams,
  type DiffResult,
  type ListDiffModesResponse,
} from "@vrooli/proto-types/image-tools/v1/diff/diff_pb";

import { decodeApiError, PROTO_READ_OPTIONS, transport, uploadFile, type JsonValue } from "./client";

/**
 * Typed Connect client for DiffService discovery (`ListDiffModes`) plus the
 * REST multipart compare edge. Comparison EXECUTION can't ride a proto field
 * (two image blobs) so it goes through `compare` below, mirroring the selection
 * `segment` edge exactly: a multipart body (`base` + `compare` files + the
 * `params` protojson) posted to a synchronous edge that returns the full
 * `DiffResult` in one round-trip.
 */
export const diffClient = createClient(DiffService, transport);

export { DiffMode };
export type { DiffModeInfo, DiffParams, DiffResult, ListDiffModesResponse };

/** Discovery wrapper — lists the comparison modes the engine supports. */
export const listDiffModes = (): Promise<ListDiffModesResponse> => diffClient.listDiffModes({});

/** Inputs to a visual comparison (the subset the UI controls). */
export interface CompareInput {
  /** The reference ("before") image. */
  base: File | Blob;
  /** The candidate ("after") image. */
  compare: File | Blob;
  /** Verdict driver: pixel (default) or perceptual. */
  mode?: DiffMode;
  /** Per-channel pixel tolerance band 0..1 (0 = exact). */
  tolerance?: number;
  /** Generate + store the heat-map overlay (default true). */
  includeHeatmap?: boolean;
  /** Heat-map highlight colour (#rrggbb); empty = default magenta. */
  highlightHex?: string;
}

/**
 * Run a visual comparison of two images. Builds the multipart request (both
 * files + DiffParams protojson), posts to the sync compare edge, and parses the
 * `DiffResult` (its int64 `changedPixels`/`totalPixels` arrive as bigint via
 * protojson — `fromJson` decodes them like the other int64 fields in this
 * codebase). A non-2xx becomes a typed `ApiError` the caller branches on. The
 * returned `heatmapRef` is a blob key (resolve via `blobUrl`/`fetchBlob`); it is
 * empty when `includeHeatmap` was false.
 */
export async function compare(input: CompareInput): Promise<DiffResult> {
  const params = create(DiffParamsSchema, {
    mode: input.mode ?? DiffMode.UNSPECIFIED,
    tolerance: input.tolerance ?? 0,
    includeHeatmap: input.includeHeatmap ?? true,
    highlightHex: input.highlightHex ?? "",
  });

  const formData = new FormData();
  formData.append("base", input.base, "base.png");
  formData.append("compare", input.compare, "compare.png");
  formData.append("params", toJsonString(DiffParamsSchema, params));

  const res = await uploadFile(`/diff/compare`, formData);
  if (!res.ok) {
    throw await decodeApiError(res);
  }
  const json = (await res.json()) as JsonValue;
  return fromJson(DiffResultSchema, json, PROTO_READ_OPTIONS);
}
