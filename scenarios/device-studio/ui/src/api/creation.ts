import { fileURL, getFile } from "./client";

export type GeometryCreationJob = {
  job_id: string;
  project_id: string;
  candidate_id: string;
  state: "queued" | "running" | "checkpointed" | "cancel_requested" | "succeeded" | "failed" | "cancelled" | string;
  version: number;
  review_status: "unreviewed" | string;
  message?: string;
};

async function decodeResponse(response: Response): Promise<GeometryCreationJob> {
  const body = await response.json() as GeometryCreationJob | { error?: string };
  if (!response.ok) throw new Error("error" in body ? body.error || "The creation request failed." : "The creation request failed.");
  if (!("job_id" in body) || !body.job_id || body.review_status !== "unreviewed") {
    throw new Error("The creation service returned an incomplete job status.");
  }
  return body;
}

export async function startGeometryCreation(projectId: string, request: {
  candidateId: string;
  idempotencyKey: string;
  brief: string;
  envelopeMM: [number, number, number];
  referenceMediaIDs: string[];
}): Promise<GeometryCreationJob> {
  if (!projectId || !request.brief.trim() || request.referenceMediaIDs.length < 1 || request.referenceMediaIDs.length > 4 || request.envelopeMM.some((value) => !Number.isFinite(value) || value <= 0 || value > 1000)) {
    throw new Error("Provide a brief, supported dimensions, and one to four project images.");
  }
  const response = await fetch(fileURL(`/projects/${encodeURIComponent(projectId)}/geometry-creations`), {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({
      candidate_id: request.candidateId,
      idempotency_key: request.idempotencyKey,
      brief: request.brief,
      envelope_mm: request.envelopeMM,
      reference_media_ids: request.referenceMediaIDs,
      confirm_local_ai: true,
    }),
    cache: "no-store",
  });
  return decodeResponse(response);
}

export async function getGeometryCreation(jobId: string): Promise<GeometryCreationJob> {
  const response = await getFile(`/geometry-creations/${encodeURIComponent(jobId)}`);
  return decodeResponse(response);
}

export async function findGeometryCreationOperation(projectId: string, operationKey: string): Promise<GeometryCreationJob | null> {
  const response = await fetch(fileURL(`/projects/${encodeURIComponent(projectId)}/geometry-creations/operations/${encodeURIComponent(operationKey)}`), { cache: "no-store" });
  if (response.status === 404) return null;
  return decodeResponse(response);
}

export async function cancelGeometryCreation(job: GeometryCreationJob): Promise<GeometryCreationJob> {
  const response = await fetch(fileURL(`/geometry-creations/${encodeURIComponent(job.job_id)}/cancel`), {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ version: job.version }),
    cache: "no-store",
  });
  return decodeResponse(response);
}
