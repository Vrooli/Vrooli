import { decodeApiError, fileURL, getFile } from "./client";

export type GeometryCreationJob = {
  job_id: string;
  project_id: string;
  candidate_id: string;
  state: "queued" | "running" | "checkpointed" | "cancel_requested" | "succeeded" | "failed" | "cancelled" | string;
  version: number;
  review_status: "unreviewed" | string;
  message?: string;
};

export type GeometryCorrectionDetails = {
  job_id: string;
  project_id: string;
  attempt_id?: string;
  source_input_fingerprint?: string;
  source_brief_sha256?: string;
  disposition: "available" | "unavailable" | string;
  issue?: { code: string; part_id: string; axis: "x" | "y" | "z"; message: string };
  prior_recipe_json?: string;
  recipe_sha256?: string;
  recipe_bytes?: number;
  envelope_mm: [number, number, number];
};

export async function getGeometryCorrection(jobId: string): Promise<GeometryCorrectionDetails> {
  const response = await getFile(`/geometry-creations/${encodeURIComponent(jobId)}/correction`);
  if (!response.ok) throw await decodeApiError(response);
  const details = await response.json() as GeometryCorrectionDetails;
  if (details.job_id !== jobId || !Array.isArray(details.envelope_mm) || details.envelope_mm.length !== 3) throw new Error("Correction details do not match the failed job.");
  if (details.disposition !== "available") return details;
  if (!details.issue || !["part_outside_envelope", "rotated_part_outside_envelope"].includes(details.issue.code) || !/^[a-z0-9_-]{1,48}$/.test(details.issue.part_id) || !["x", "y", "z"].includes(details.issue.axis) || !details.prior_recipe_json || !details.recipe_sha256 || !details.recipe_bytes || new TextEncoder().encode(details.prior_recipe_json).length !== details.recipe_bytes) {
    throw new Error("Verified prior geometry response is incomplete.");
  }
  const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(details.prior_recipe_json));
  const sha256 = [...new Uint8Array(digest)].map((value) => value.toString(16).padStart(2, "0")).join("");
  if (sha256 !== details.recipe_sha256) throw new Error("Verified prior geometry response digest differs.");
  JSON.parse(details.prior_recipe_json);
  return details;
}

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
  fundingRoute: "local" | "byok" | "vrooli_credits";
  correctionSourceJobId?: string;
  correctionRecipeSha256?: string;
  correctionFeedback?: string;
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
      funding_route: request.fundingRoute,
      confirm_local_ai: request.fundingRoute === "local",
      confirm_remote_ai: request.fundingRoute !== "local",
      correction_source_job_id: request.correctionSourceJobId,
      correction_recipe_sha256: request.correctionRecipeSha256,
      correction_feedback: request.correctionFeedback,
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
