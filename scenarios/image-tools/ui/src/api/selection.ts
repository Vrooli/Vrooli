import { createClient } from "@connectrpc/connect";
import { create, fromJson, toJsonString } from "@bufbuild/protobuf";
import {
  SegmentMode,
  SegmentParamsSchema,
  SegmentResultSchema,
  SelectionService,
  type ListRegionClassesResponse,
  type RegionClassInfo,
  type SegmentParams,
  type SegmentResult,
  type SuggestedEdit,
} from "@vrooli/proto-types/image-tools/v1/selection/selection_pb";

import { decodeApiError, PROTO_READ_OPTIONS, transport, uploadFile, type JsonValue } from "./client";

/**
 * Typed Connect client for SelectionService discovery + the pure contextual-edit
 * compiler (ListRegionClasses / SuggestEdits). Segmentation EXECUTION is the
 * REST multipart edge (`POST /api/v1/selection/segment`) — image bytes can't
 * ride a proto field — so it goes through `segment` below. Segmentation runs
 * *synchronously* (the response carries the mask ref + class + edit menu), like
 * the analysis edge.
 */
export const selectionClient = createClient(SelectionService, transport);

export { SegmentMode };
export type { RegionClassInfo, SegmentParams, SegmentResult, SuggestedEdit, ListRegionClassesResponse };

/** Discovery wrapper — lists region classes + their contextual edit menus. */
export const listRegionClasses = (): Promise<ListRegionClassesResponse> =>
  selectionClient.listRegionClasses({});

/** The compose-seam — the contextual edit menu for a region class. */
export const suggestEdits = (regionClass: string): Promise<{ regionClass: string; edits: SuggestedEdit[] }> =>
  selectionClient.suggestEdits({ regionClass }).then((r) => ({ regionClass: r.regionClass, edits: r.edits }));

/** Inputs to a smart-select segmentation (a subset the UI controls). */
export interface SegmentInput {
  image: File | Blob;
  mode?: SegmentMode;
  /** Normalized point(s) (0..1) for POINT mode. */
  points?: { x: number; y: number; negative?: boolean }[];
  /** Normalized box (0..1) for BOX mode. */
  box?: { x: number; y: number; width: number; height: number };
  /** Region-grow colour threshold 0..1 (0 = default). */
  tolerance?: number;
  /** Force a SAM model id (falls back to the built-in region-grow if unwired). */
  modelOverride?: string;
}

/**
 * Run a smart-select segmentation against the image. Builds the multipart
 * request (file + SegmentParams protojson), posts to the sync segment edge, and
 * parses the `SegmentResult`. A non-2xx becomes a typed `ApiError` the caller
 * branches on. The returned `maskRef` is a blob key (fetch via `blobUrl`/
 * `fetchBlob`) and feeds the AI submit edge's `mask` part for a masked edit.
 */
export async function segment(input: SegmentInput): Promise<SegmentResult> {
  const params = create(SegmentParamsSchema, {
    mode: input.mode ?? SegmentMode.UNSPECIFIED,
    points: (input.points ?? []).map((p) => ({ x: p.x, y: p.y, negative: p.negative ?? false })),
    box: input.box,
    tolerance: input.tolerance ?? 0,
    modelOverride: input.modelOverride ?? "",
  });

  const formData = new FormData();
  formData.append("file", input.image, "input.png");
  formData.append("params", toJsonString(SegmentParamsSchema, params));

  const res = await uploadFile(`/selection/segment`, formData);
  if (!res.ok) {
    throw await decodeApiError(res);
  }
  const json = (await res.json()) as JsonValue;
  return fromJson(SegmentResultSchema, json, PROTO_READ_OPTIONS);
}
