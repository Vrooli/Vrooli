import { createClient } from "@connectrpc/connect";
import {
  RenderJobStatus,
  RenderJobsService,
  type RenderOutput,
  type VideoSampleBinding,
} from "@vrooli/proto-types/device-studio/v1/shots/shots_pb";
import { decodeApiError, fileURL, getFile } from "./client";
import { transport } from "./client";

export const rendersClient = createClient(RenderJobsService, transport);

export function isRenderActive(status: RenderJobStatus): boolean {
  return status === RenderJobStatus.QUEUED
    || status === RenderJobStatus.OBSERVED
    || status === RenderJobStatus.NEEDS_RECONCILIATION
    || status === RenderJobStatus.CANCEL_REQUESTED;
}

export function isRenderCancellable(status: RenderJobStatus): boolean {
  return status === RenderJobStatus.QUEUED
    || status === RenderJobStatus.OBSERVED
    || status === RenderJobStatus.NEEDS_RECONCILIATION;
}

function expectedContentPath(projectId: string, renderId: string): string {
  return `/api/v1/projects/${encodeURIComponent(projectId)}/renders/${encodeURIComponent(renderId)}/content`;
}

export function renderOutputContentURL(projectId: string, renderId: string, contentUrl: string): string {
  const expected = expectedContentPath(projectId, renderId);
  if (!contentUrl.startsWith("/") || contentUrl.startsWith("//")) throw new Error("The render output link is not a project-scoped content path.");
  const currentLocation = Reflect.get(globalThis, "location") as { origin?: string } | undefined;
  const parsed = new URL(contentUrl, currentLocation?.origin ?? "http://localhost");
  if (parsed.pathname !== expected || parsed.search || parsed.hash) throw new Error("The render output link does not match this project and job.");
  return fileURL(`/projects/${encodeURIComponent(projectId)}/renders/${encodeURIComponent(renderId)}/content`);
}

export async function getVideoSampleBinding(projectId: string, shotId: string, renderId: string): Promise<VideoSampleBinding> {
  const response = await rendersClient.getVideoSampleBinding({ projectId, shotId, renderId });
  if (!response.binding) throw new Error("The completed sample has no verified settings claim.");
  return response.binding;
}

export async function getRenderOutputContent(projectId: string, renderId: string, contentUrl: string): Promise<Blob> {
  const expected = expectedContentPath(projectId, renderId);
  if (!contentUrl.startsWith("/") || contentUrl.startsWith("//")) {
    throw new Error("The render output link is not a project-scoped content path.");
  }
  const currentLocation = Reflect.get(globalThis, "location") as { origin?: string } | undefined;
  const parsed = new URL(contentUrl, currentLocation?.origin ?? "http://localhost");
  if (parsed.pathname !== expected || parsed.search || parsed.hash) {
    throw new Error("The render output link does not match this project and job.");
  }
  const apiPath = `/projects/${encodeURIComponent(projectId)}/renders/${encodeURIComponent(renderId)}/content`;
  const response = await getFile(apiPath);
  if (!response.ok) throw await decodeApiError(response);
  return response.blob();
}

export async function verifyRenderOutputContent(blob: Blob, output: RenderOutput): Promise<Blob> {
  if (output.mediaType.toLowerCase() !== "image/png" || blob.type.toLowerCase() !== "image/png") {
    throw new Error("The completed render is not a PNG image.");
  }
  if (BigInt(blob.size) !== output.sizeBytes) {
    throw new Error("The render output size did not match its saved metadata.");
  }
  if (!/^[a-f0-9]{64}$/i.test(output.sha256)) {
    throw new Error("The render output digest metadata is invalid.");
  }
  // The project-scoped content endpoint verifies the stored digest before it
  // returns bytes; Web Crypto adds an independent client check when available.
  const runtimeCrypto = Reflect.get(globalThis, "crypto") as { subtle?: SubtleCrypto } | undefined;
  const subtle = runtimeCrypto?.subtle;
  const readArrayBuffer = Reflect.get(blob, "arrayBuffer");
  if (subtle && typeof readArrayBuffer === "function") {
    const bytes = await Reflect.apply(readArrayBuffer, blob, []);
    const digest = await subtle.digest("SHA-256", bytes);
    const sha256 = [...new Uint8Array(digest)].map((value) => value.toString(16).padStart(2, "0")).join("");
    if (sha256 !== output.sha256.toLowerCase()) throw new Error("The render output digest did not match its saved metadata.");
  }

  if (typeof globalThis.createImageBitmap !== "function") {
    throw new Error("This browser cannot verify the rendered PNG dimensions.");
  }
  const bitmap = await globalThis.createImageBitmap(blob);
  try {
    if (bitmap.width !== output.widthPixels || bitmap.height !== output.heightPixels) {
      throw new Error("The rendered PNG dimensions did not match its saved metadata.");
    }
  } finally {
    bitmap.close();
  }
  return blob;
}
