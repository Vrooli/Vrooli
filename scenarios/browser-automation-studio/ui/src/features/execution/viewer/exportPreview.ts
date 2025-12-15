import {
  type ExecutionExportPreview as ProtoExecutionExportPreview,
  ExecutionExportPreviewSchema,
} from "@vrooli/proto-types/browser-automation-studio/v1/execution/execution_pb";
import type { ReplayMovieSpec } from "@/types/export";
import { mapExportStatus, type ExportStatusLabel } from "../utils/exportHelpers";
import { getConfig } from "@/config";
import { parseProtoStrict } from "@/utils/proto";

export interface ExportPreviewMetrics {
  capturedFrames: number;
  assetCount: number;
  totalDurationMs: number;
}

export const parseExportPreviewPayload = (
  raw: unknown,
): {
  preview: ProtoExecutionExportPreview;
  status: ExportStatusLabel;
  metrics: ExportPreviewMetrics;
  movieSpec: ReplayMovieSpec | null;
} => {
  const rawRecord =
    raw && typeof raw === "object" && !Array.isArray(raw)
      ? (raw as Record<string, unknown>)
      : null;

  const rawMovieSpec = rawRecord?.package ?? null;
  const rawForProto = rawRecord ? { ...rawRecord } : raw;
  if (rawRecord) {
    delete (rawForProto as Record<string, unknown>).package;
  }

  const preview = parseProtoStrict<ProtoExecutionExportPreview>(
    ExecutionExportPreviewSchema,
    rawForProto,
  );

  const status = mapExportStatus(preview.status);
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

  const movieSpec =
    rawMovieSpec && typeof rawMovieSpec === "object" && !Array.isArray(rawMovieSpec)
      ? (rawMovieSpec as ReplayMovieSpec)
      : null;

  return { preview, status, metrics, movieSpec };
};

export const fetchExecutionExportPreview = async (
  executionId: string,
  options?: { signal?: AbortSignal },
) => {
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
  const raw = await response.json();
  return parseExportPreviewPayload(raw);
};
