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
