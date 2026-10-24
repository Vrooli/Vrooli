import {
  type ExecutionExportPreview as ProtoExecutionExportPreview,
  ExecutionExportPreviewSchema,
} from "@vrooli/proto-types/browser-automation-studio/v1/execution/execution_pb";
import type { ReplaySpec as ReplayMovieSpec } from "@vrooli/generated-proto/browser-automation-studio/v1/exports/exports_pb";
import { ReplaySpecSchema } from "@vrooli/generated-proto/browser-automation-studio/v1/exports/exports_pb";
import {
  mapExportPreviewStatus,
  type ExportPreviewStatusLabel,
} from "@/domains/exports/presentation";
import { getConfig } from "@/config";
import { parseProtoStrict } from "@/utils/proto";

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value);

export interface ExportPreviewMetrics {
  capturedFrames: number;
  assetCount: number;
  totalDurationMs: number;
}

export const parseExportPreviewPayload = (
  raw: unknown,
): {
  preview: ProtoExecutionExportPreview;
  status: ExportPreviewStatusLabel;
  metrics: ExportPreviewMetrics;
  movieSpec: ReplayMovieSpec | null;
} => {
  const rawRecord = isRecord(raw) ? raw : null;

  const rawMovieSpec = rawRecord?.package ?? null;
  const rawForProto = rawRecord
    ? Object.fromEntries(Object.entries(rawRecord).filter(([key]) => key !== "package"))
    : raw;

  const preview = parseProtoStrict<ProtoExecutionExportPreview>(
    ExecutionExportPreviewSchema,
    rawForProto,
  );

  const status = mapExportPreviewStatus(preview.status);
  const metrics: ExportPreviewMetrics = {
    capturedFrames: typeof preview.capturedFrameCount === "number"
      ? preview.capturedFrameCount
      : 0,
    assetCount: typeof preview.availableAssetCount === "number"
      ? preview.availableAssetCount
      : 0,
    totalDurationMs: typeof preview.totalDurationMs === "number"
      ? preview.totalDurationMs
      : 0,
  };

  let movieSpec: ReplayMovieSpec | null = null;
  if (rawMovieSpec && typeof rawMovieSpec === "object" && !Array.isArray(rawMovieSpec)) {
    try {
      movieSpec = parseProtoStrict<ReplayMovieSpec>(ReplaySpecSchema, rawMovieSpec);
    } catch {
      movieSpec = null;
    }
  }

  return { preview, status, metrics, movieSpec };
};

export const fetchExecutionExportPreview = async (
  executionId: string,
  options?: { signal?: AbortSignal },
) => {
  // Validate at service boundary - prevents /executions//export requests
  if (!executionId || executionId.trim() === '') {
    throw new Error(JSON.stringify({
      code: 'INVALID_EXECUTION_ID',
      message: 'Execution ID is required',
    }));
  }

  const { API_URL } = await getConfig();
  const response = await fetch(
    `${API_URL}/executions/${executionId}/export`,
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      body: JSON.stringify({ format: "json" }),
      signal: options?.signal,
    },
  );
  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || `Export request failed (${response.status})`);
  }
  const raw: unknown = await response.json();
  return parseExportPreviewPayload(raw);
};
