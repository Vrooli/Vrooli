import { ConnectError } from "@connectrpc/connect";
import { createClient } from "@connectrpc/connect";
import {
  ProjectsService,
  ProjectConflictSchema,
  UploadSourceMediaResponseSchema,
  type Project,
  type SourceMedia,
} from "@vrooli/proto-types/device-studio/v1/projects/projects_pb";
import { transport } from "./client";
import { PROTO_READ_OPTIONS, decodeApiError, fromJson, getFile, makeApiError, uploadFile, type JsonValue } from "./client";

export const projectsClient = createClient(ProjectsService, transport);

export function projectConflict(error: unknown) {
  if (!(error instanceof ConnectError)) return undefined;
  return error.findDetails(ProjectConflictSchema)[0];
}

export async function uploadSourceMedia(projectId: string, file: File): Promise<SourceMedia> {
  const form = new FormData();
  form.set("file", file, file.name);
  const response = await uploadFile(`/projects/${encodeURIComponent(projectId)}/media`, form);
  if (!response.ok) throw await decodeApiError(response);
  const decoded = fromJson(UploadSourceMediaResponseSchema, (await response.json()) as JsonValue, PROTO_READ_OPTIONS);
  if (!decoded.sourceMedia) throw makeApiError("internal", "media upload returned no metadata");
  return decoded.sourceMedia;
}

export async function getSourceMediaContent(projectId: string, mediaId: string): Promise<Blob> {
  const response = await getFile(`/projects/${encodeURIComponent(projectId)}/media/${encodeURIComponent(mediaId)}/content`);
  if (!response.ok) throw await decodeApiError(response);
  return response.blob();
}

export type { Project, SourceMedia };
