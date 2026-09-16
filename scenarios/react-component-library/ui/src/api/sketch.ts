import { createClient } from "@connectrpc/connect";
import { SketchService } from "@vrooli/proto-types/react-component-library/v1/sketch/sketch_pb";
import { decodeApiError, transport, uploadFile } from "./client";

// All workspace reads and writes use the same generated, routed service as CLI.
export const sketchClient = createClient(SketchService, transport);

export interface ReferenceAssetUpload {
  id: string;
  name: string;
  size: number;
  mime: string;
  url: string;
}

export interface ReferenceAssetGeneration extends ReferenceAssetUpload {
  jobId: string;
  modelId: string;
  tier: string;
  warnings: string[];
}

export async function uploadReferenceAsset(target: { scenario: string; page: string }, file: File): Promise<ReferenceAssetUpload> {
  const form = new FormData();
  form.append("scenario", target.scenario);
  form.append("page", target.page);
  form.append("file", file);
  const response = await uploadFile("/reference-assets", form);
  if (!response.ok) throw await decodeApiError(response);
  const value = (await response.json()) as Partial<ReferenceAssetUpload>;
  if (!value.id || !value.name || !value.url) throw new Error("reference asset upload returned incomplete metadata");
  return { id: value.id, name: value.name, size: value.size ?? 0, mime: value.mime ?? file.type, url: value.url };
}

export async function generateReferenceAsset(
  target: { scenario: string; page: string },
  prompt: string,
): Promise<ReferenceAssetGeneration> {
  const response = await fetch("/api/v1/reference-assets/generate", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ scenario: target.scenario, page: target.page, prompt }),
  });
  if (!response.ok) throw await decodeApiError(response);
  const value = (await response.json()) as Partial<ReferenceAssetGeneration>;
  if (!value.id || !value.name || !value.url || !value.jobId || !value.modelId) {
    throw new Error("reference generation returned incomplete metadata");
  }
  return {
    id: value.id,
    name: value.name,
    size: value.size ?? 0,
    mime: value.mime ?? "image/png",
    url: value.url,
    jobId: value.jobId,
    modelId: value.modelId,
    tier: value.tier ?? "unknown",
    warnings: value.warnings ?? [],
  };
}
