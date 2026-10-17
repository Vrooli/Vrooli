import { createClient } from "@connectrpc/connect";
import {
  OpsService,
  type ListOperationsResponse,
  type OperationInfo,
} from "@vrooli/proto-types/image-tools/v1/ops/ops_pb";

import { decodeApiError, transport, uploadFile } from "./client";

/**
 * Typed Connect client for the discovery RPC. Execution is a REST
 * multipart endpoint (`POST /api/v1/ops/{operation}`) rather than an RPC,
 * so it goes through `runOp` below instead of this client.
 */
export const opsClient = createClient(OpsService, transport);

/** Discovery wrapper — lists operations plus decodable/encodable formats. */
export const listOperations = (): Promise<ListOperationsResponse> =>
  opsClient.listOperations({});

/**
 * Parameters for a single op, keyed by proto field name (snake_case in
 * protojson). The server wraps these in the matching `OpParams` oneof
 * case, so the shape sent on the wire is `{ "<operation>": { ...params } }`.
 */
export type OpParamValues = Record<string, string | number | boolean>;

/** Image-producing op result: an object URL plus the result metadata. */
export interface RunOpImageResult {
  kind: "image";
  url: string;
  width: number;
  height: number;
  format: string;
  jobId: string;
}

/** Metadata-read result: the server streams JSON instead of image bytes. */
export interface RunOpMetadataResult {
  kind: "metadata";
  json: string;
}

export type RunOpResult = RunOpImageResult | RunOpMetadataResult;

export interface RunOpOptions {
  /** Optional watermark image; only meaningful for the `overlay` op. */
  overlay?: File;
}

const HEADER_WIDTH = "X-Image-Tools-Width";
const HEADER_HEIGHT = "X-Image-Tools-Height";
const HEADER_FORMAT = "X-Image-Tools-Format";
const HEADER_JOB_ID = "X-Image-Tools-Job-Id";

const toNumber = (raw: string | null): number => {
  const value = Number(raw);
  return Number.isFinite(value) ? value : 0;
};

/**
 * Execute a deterministic op against `file`. Builds the multipart request
 * (`file` bytes + `params` protojson + optional `overlay`) and returns an
 * object URL for image results, or the raw JSON for a metadata read.
 *
 * The `metadata` op with no strip flags returns JSON (a metadata read);
 * every other op streams the result image bytes (`output=bytes`).
 */
export async function runOp(
  operation: string,
  file: File,
  params: OpParamValues,
  opts: RunOpOptions = {},
): Promise<RunOpResult> {
  const formData = new FormData();
  formData.append("file", file);
  formData.append("params", JSON.stringify({ [operation]: params }));
  if (opts.overlay) {
    formData.append("overlay", opts.overlay);
  }

  const res = await uploadFile(`/ops/${operation}?output=bytes`, formData);
  if (!res.ok) {
    throw await decodeApiError(res);
  }

  const contentType = res.headers.get("Content-Type") ?? "";
  if (contentType.includes("application/json")) {
    return { kind: "metadata", json: await res.text() };
  }

  const blob = await res.blob();
  return {
    kind: "image",
    url: URL.createObjectURL(blob),
    width: toNumber(res.headers.get(HEADER_WIDTH)),
    height: toNumber(res.headers.get(HEADER_HEIGHT)),
    format: res.headers.get(HEADER_FORMAT) ?? "",
    jobId: res.headers.get(HEADER_JOB_ID) ?? "",
  };
}

export type { ListOperationsResponse, OperationInfo };
