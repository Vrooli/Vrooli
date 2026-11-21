import { createClient } from "@connectrpc/connect";
import { fromJson } from "@bufbuild/protobuf";

import { ImportArchiveResponseSchema, SourcesService, type AdapterState } from "../../../../../packages/proto/gen/typescript/signal-inbox/v1/sources/sources_pb";
import { decodeApiError, PROTO_READ_OPTIONS, transport, uploadFile } from "./client";

export const sourcesClient = createClient(SourcesService, transport);
export type { AdapterState };

export async function uploadArchive(adapterId: string, file: File) {
  const form = new FormData();
  form.set("adapter_id", adapterId);
  form.set("file", file);
  const response = await uploadFile("/api/v1/sources/archive", form);
  if (!response.ok) throw await decodeApiError(response);
  return fromJson(ImportArchiveResponseSchema, await response.json(), PROTO_READ_OPTIONS);
}
