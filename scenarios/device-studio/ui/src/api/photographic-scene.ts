import { decodeApiError, fileURL, getFile } from "./client";

export interface PhotographicOperation {
  job_id: string; operation: string; state: string; progress: number;
  model_id?: string; tier?: string;
  message?: string; error?: string; result_ref?: string; result_meta?: Record<string, string>;
}
export interface PhotographicRequest { operation_id: string; prompt: string; width: 768 | 512; height: 512 | 768 }

async function jsonRequest(path: string, method: "GET" | "POST", body?: unknown): Promise<PhotographicOperation> {
  const response = await fetch(fileURL(path), { method, headers: body ? { "Content-Type": "application/json" } : undefined, body: body ? JSON.stringify(body) : undefined, cache: "no-store" });
  if (!response.ok) throw await decodeApiError(response);
  return await response.json() as PhotographicOperation;
}
const root = (projectId: string) => `/projects/${encodeURIComponent(projectId)}/photographic-scenes`;

export function submitPhotographicScene(projectId: string, request: PhotographicRequest) {
  return jsonRequest(root(projectId), "POST", request);
}
export function readPhotographicOperation(projectId: string, operationId: string) {
  return jsonRequest(`${root(projectId)}/operations/${encodeURIComponent(operationId)}`, "GET");
}
export function waitPhotographicOperation(projectId: string, operationId: string) {
  return jsonRequest(`${root(projectId)}/operations/${encodeURIComponent(operationId)}/wait`, "POST", {});
}
export function cancelPhotographicOperation(projectId: string, operationId: string) {
  return jsonRequest(`${root(projectId)}/operations/${encodeURIComponent(operationId)}/cancel`, "POST", {});
}
export async function getPhotographicPNG(projectId: string, operationId: string, width: number, height: number): Promise<Blob> {
  const response = await getFile(`${root(projectId)}/operations/${encodeURIComponent(operationId)}/content?width=${width}&height=${height}`);
  if (!response.ok) throw await decodeApiError(response);
  const blob = await response.blob();
  if (blob.type.toLowerCase() !== "image/png" || blob.size < 1 || blob.size > 12 * 1024 * 1024) throw new Error("The verified result exceeded the expected PNG bounds.");
  return blob;
}
