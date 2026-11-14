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
import { PROTO_READ_OPTIONS, decodeApiError, fileURL, fromJson, getFile, makeApiError, uploadFile, type JsonValue } from "./client";

export const projectsClient = createClient(ProjectsService, transport);

export async function getSourceMediaClockPages(projectId: string, mediaId: string, clockSha256: string) {
  const pages = [];
  let pageToken = "";
  do {
    const page = await projectsClient.getSourceMediaClockPage({
      projectId, mediaId, expectedClockSha256: clockSha256, pageSize: 256, pageToken,
    });
    if (page.clockSha256 !== clockSha256) throw makeApiError("data_loss", "The source clock changed while it was being read.");
    pages.push(page);
    pageToken = page.nextPageToken;
  } while (pageToken);
  return pages;
}

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

// Native media elements stream from the routed content endpoint directly. Do
// not fetch recordings into a Blob: videos may be hundreds of MiB.
export function sourceMediaContentURL(projectId: string, mediaId: string): string {
  return fileURL(`/projects/${encodeURIComponent(projectId)}/media/${encodeURIComponent(mediaId)}/content`);
}

// The content endpoint verifies stored SHA-256 before returning bytes. This client
// helper also checks the returned byte count and MIME metadata, and recomputes the
// digest where Web Crypto is available (for example, secure contexts and localhost).
export async function verifySourceMediaContent(blob: Blob, media: SourceMedia): Promise<Blob> {
  if (BigInt(blob.size) !== media.sizeBytes) throw new Error("Source content size did not match its saved metadata");
  if (blob.type.toLowerCase() !== media.mediaType.toLowerCase()) throw new Error("Source content type did not match its saved metadata");
  const subtle = Reflect.get(globalThis.crypto, "subtle") as SubtleCrypto | undefined;
  if (subtle && typeof blob.arrayBuffer === "function") {
    const digest = await subtle.digest("SHA-256", await blob.arrayBuffer());
    const sha256 = [...new Uint8Array(digest)].map((value) => value.toString(16).padStart(2, "0")).join("");
    if (sha256 !== media.sha256.toLowerCase()) throw new Error("Source content digest did not match its saved metadata");
  }
  return blob;
}

export type { Project, SourceMedia };
