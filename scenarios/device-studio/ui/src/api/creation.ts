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

export type GeometryCreationSession = {
  session_id: string;
  state: "active" | "closing" | "cancelled" | "completed" | string;
  version: number;
  policy: { project_id: string; candidate_id: string; brief: string; envelope_mm: [number, number, number]; reference_media_ids: string[]; funding_route: "local"; tokens_per_call: 4096; parent_session_id?: string };
  max_provider_calls: number;
  provider_calls_reserved: number;
  max_children: number;
  children_admitted: number;
  children: Array<{ job_id: string; ordinal: number; state: GeometryCreationJob["state"]; version: number; candidate_id: string }>;
  needs_confirmation?: boolean;
  changed_scope?: string[];
};

export type GeometryCorrectionDetails = {
  job_id: string;
  project_id: string;
  attempt_id?: string;
  source_input_fingerprint?: string;
  source_brief_sha256?: string;
  disposition: "available" | "unavailable" | string;
  mode?: "semantic_repair" | "visual_refinement";
  issue?: {
    code: string;
    part_id: string;
    axis?: "x" | "y" | "z";
    bevel_mm?: number;
    minimum_dimension_mm?: number;
    max_bevel_exclusive_mm?: number;
    message: string;
  };
  prior_recipe_json?: string;
  recipe_sha256?: string;
  recipe_bytes?: number;
  envelope_mm: [number, number, number];
  reference_media_ids?: string[];
};

export async function getGeometryCorrection(jobId: string): Promise<GeometryCorrectionDetails> {
  const response = await getFile(`/geometry-creations/${encodeURIComponent(jobId)}/correction`);
  if (!response.ok) throw await decodeApiError(response);
  const details = await response.json() as GeometryCorrectionDetails;
  if (details.job_id !== jobId || !Array.isArray(details.envelope_mm) || details.envelope_mm.length !== 3) throw new Error("Correction details do not match the source job.");
  if (details.disposition !== "available") return details;
  const mode = details.mode ?? "semantic_repair";
  const issue = details.issue;
  const validIssue = Boolean(issue && /^[a-z0-9_-]{1,48}$/.test(issue.part_id) && (
    (["part_outside_envelope", "rotated_part_outside_envelope"].includes(issue.code) && ["x", "y", "z"].includes(issue.axis ?? "")) ||
    (issue.code === "bevel_exceeds_part_dimensions" && issue.axis === undefined && Number.isFinite(issue.bevel_mm) && (issue.bevel_mm ?? 0) > 0 && Number.isFinite(issue.minimum_dimension_mm) && (issue.minimum_dimension_mm ?? 0) > 0 && Number.isFinite(issue.max_bevel_exclusive_mm) && issue.max_bevel_exclusive_mm === (issue.minimum_dimension_mm ?? 0) / 2 && (issue.bevel_mm ?? 0) >= (issue.max_bevel_exclusive_mm ?? 0) && issue.message === "bevel must be less than half the smallest part dimension")
  ));
  if (!details.prior_recipe_json || !details.recipe_sha256 || !details.recipe_bytes || !details.reference_media_ids?.length || new TextEncoder().encode(details.prior_recipe_json).length !== details.recipe_bytes || (mode === "semantic_repair" && (!issue || !validIssue)) || (mode === "visual_refinement" && issue !== undefined) || !["semantic_repair", "visual_refinement"].includes(mode)) {
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
  maxCostMicros?: number;
  correctionSourceJobId?: string;
  correctionMode?: "semantic_repair" | "visual_refinement";
  correctionRecipeSha256?: string;
  correctionFeedback?: string;
}): Promise<GeometryCreationJob> {
  if (!projectId || !request.brief.trim() || request.referenceMediaIDs.length < 1 || request.referenceMediaIDs.length > 4 || request.envelopeMM.some((value) => !Number.isFinite(value) || value <= 0 || value > 1000)) {
    throw new Error("Provide a brief, supported dimensions, and one to four project images.");
  }
  if (request.fundingRoute === "local" ? request.maxCostMicros !== undefined : !Number.isSafeInteger(request.maxCostMicros) || (request.maxCostMicros ?? 0) <= 0) {
    throw new Error("Choose a positive maximum spend for remote creation; local creation does not accept a remote ceiling.");
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
      max_cost_micros: request.maxCostMicros,
      confirm_local_ai: request.fundingRoute === "local",
      confirm_remote_ai: request.fundingRoute !== "local",
      correction_source_job_id: request.correctionSourceJobId,
      correction_mode: request.correctionMode,
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

async function decodeSessionResponse(response: Response): Promise<GeometryCreationSession> {
  const body = await response.json() as GeometryCreationSession | { error?: string };
  if (!response.ok && !(response.status === 409 && "session_id" in body)) {
    throw new Error("error" in body ? body.error || "The creation session request failed." : "The creation session request failed.");
  }
  if (!("session_id" in body) || !body.session_id || !Number.isSafeInteger(body.version) || !Array.isArray(body.children)) {
    throw new Error("The creation service returned an incomplete session.");
  }
  return body;
}

export async function createGeometryCreationSession(projectId: string, request: {
  candidateId: string; idempotencyKey: string; brief: string; envelopeMM: [number, number, number]; referenceMediaIDs: string[]; confirmLocalAI: boolean;
}): Promise<GeometryCreationSession> {
  const response = await fetch(fileURL(`/projects/${encodeURIComponent(projectId)}/geometry-creation-sessions`), {
    method: "POST", headers: { "content-type": "application/json" },
    body: JSON.stringify({ candidate_id: request.candidateId, idempotency_key: request.idempotencyKey, brief: request.brief, envelope_mm: request.envelopeMM, reference_media_ids: request.referenceMediaIDs, funding_route: "local", confirm_local_ai: request.confirmLocalAI }),
    cache: "no-store",
  });
  return decodeSessionResponse(response);
}

export async function getGeometryCreationSession(sessionId: string): Promise<GeometryCreationSession> {
  const response = await getFile(`/geometry-creation-sessions/${encodeURIComponent(sessionId)}`);
  return decodeSessionResponse(response);
}

export async function refineGeometryCreationSession(sessionId: string, request: {
  candidateId: string; idempotencyKey: string; brief: string; envelopeMM: [number, number, number]; referenceMediaIDs: string[];
  correctionSourceJobId: string; correctionMode: "semantic_repair" | "visual_refinement"; correctionRecipeSha256: string; correctionFeedback: string; confirmScopeExpansion: boolean; confirmLocalAI: boolean;
}): Promise<GeometryCreationSession> {
  const response = await fetch(fileURL(`/geometry-creation-sessions/${encodeURIComponent(sessionId)}/refinements`), {
    method: "POST", headers: { "content-type": "application/json" },
    body: JSON.stringify({ candidate_id: request.candidateId, idempotency_key: request.idempotencyKey, brief: request.brief, envelope_mm: request.envelopeMM, reference_media_ids: request.referenceMediaIDs, funding_route: "local", confirm_local_ai: request.confirmLocalAI, correction_source_job_id: request.correctionSourceJobId, correction_mode: request.correctionMode, correction_recipe_sha256: request.correctionRecipeSha256, correction_feedback: request.correctionFeedback, confirm_scope_expansion: request.confirmScopeExpansion }),
    cache: "no-store",
  });
  return decodeSessionResponse(response);
}

export async function cancelGeometryCreationSession(sessionId: string, version: number): Promise<GeometryCreationSession> {
  const response = await fetch(fileURL(`/geometry-creation-sessions/${encodeURIComponent(sessionId)}/cancel`), {
    method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify({ version }), cache: "no-store",
  });
  return decodeSessionResponse(response);
}
