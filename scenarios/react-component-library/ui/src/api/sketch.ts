import { createClient } from "@connectrpc/connect";
import { create, fromJson, type JsonValue } from "@bufbuild/protobuf";
import {
  ImportPageResponseSchema,
  SketchService,
  type ImportPageResponse,
} from "@vrooli/proto-types/react-component-library/v1/sketch/sketch_pb";
import { API_BASE, boundedFetch, decodeApiError, transport, uploadFile } from "./client";

// All workspace reads and writes use the same generated, routed service as CLI.
export const sketchClient = createClient(SketchService, transport);

/**
 * ImportPage is catalog-backed and can outlive the browser proxy's ordinary
 * Connect request. Keep the JSON service boundary and generated response
 * schema, but own the longer bounded request so a successful write cannot
 * remain visually pending forever.
 */
export async function importSketchPage(
  target: { scenario: string; page: string },
  write: boolean,
  expectedContentHash: string,
  onWriteAccepted?: () => void,
): Promise<ImportPageResponse> {
  const response = await boundedFetch(
    `${API_BASE}/vrooli.react_component_library.v1.sketch.SketchService/ImportPage`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ target, write, expectedContentHash }),
    },
  );
  if (!response.ok) throw await decodeApiError(response);
  // A successful write is the authoritative adoption acknowledgement. The
  // browser recording proxy has been observed to leave large write response
  // bodies open after delivering the 200 headers; do not make the operator
  // wait on a redundant decomposition payload after the server has committed.
  if (write) {
    onWriteAccepted?.();
    void response.body?.cancel();
    return create(ImportPageResponseSchema, { written: true, contentHash: expectedContentHash });
  }
  return fromJson(ImportPageResponseSchema, (await response.json()) as JsonValue, {
    ignoreUnknownFields: true,
  });
}

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
