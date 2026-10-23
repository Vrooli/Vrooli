import { fromJson } from "@bufbuild/protobuf";
import type { IntakeCandidateRevision } from "@vrooli/proto-types/device-studio/v1/library/library_pb";
import { GetIntakeCandidateResponseSchema, LibraryCatalogService } from "@vrooli/proto-types/device-studio/v1/library/library_pb";
import { createClient } from "@connectrpc/connect";
import { decodeApiError, fileURL, getFile, PROTO_READ_OPTIONS, transport, uploadFile } from "./client";

const catalog = createClient(LibraryCatalogService, transport);
export const MAX_LOCAL_GEOMETRY_BYTES = 128 * 1024 * 1024;

export interface IntakeMaterial {
  index: number; name: string; base_color_factor: [number, number, number, number]; metallic_factor: number;
  roughness_factor: number; emissive_factor: [number, number, number]; editable: boolean;
}
export interface IntakeMaterialInventory {
  candidate_id: string; revision: number; revision_sha256: string; glb_sha256: string; materials: IntakeMaterial[];
  material_correction_lineage?: Record<string, unknown>;
}
export interface IntakeGeometryPreview {
  blob: Blob; preview_sha256: string; vertex_colors: boolean; width: number; height: number;
}
export interface MaterialCorrectionRequest {
  parent_candidate_id: string; parent_revision: number; parent_revision_sha256: string; child_candidate_id: string;
  material_index: number; material_name: string; base_color_factor?: [number, number, number, number];
  metallic_factor?: number; roughness_factor?: number; emissive_factor?: [number, number, number];
}
export interface MaterialCorrectionCandidate {
  id: string; revision: number; revision_sha256: string; original_sha256: string; review_status: string;
  render_status: string; parent_candidate_id: string; parent_revision: number; parent_sha256: string;
  import_method: string; material_correction_lineage?: Record<string, unknown>;
}

export function listLocalGeometryCandidates(afterCandidateId = "") {
  if (afterCandidateId && !/^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/.test(afterCandidateId)) {
    throw new Error("The intake cursor must be a stable candidate ID.");
  }
  return catalog.listIntakeCandidates({ afterCandidateId, pageSize: 100 });
}

export async function getLocalGeometryCreationLineage(candidateId: string, revision = 1) {
  if (!/^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/.test(candidateId) || !Number.isSafeInteger(revision) || revision <= 0) {
    throw new Error("The generated candidate identity must be stable and revisioned.");
  }
  const response = await catalog.getIntakeCandidate({ candidateId, revision });
  if (!response.candidate || response.candidate.id !== candidateId || response.candidate.revision !== revision) {
    throw new Error("The intake service returned a different generated candidate revision.");
  }
  return response.candidate.creationLineage;
}

export async function importLocalGeometryCandidate(file: File, claims: {
  candidateId: string; displayName: string; sourceLabel: string; licenseClaim: string; attribution: string;
}): Promise<IntakeCandidateRevision> {
  if (file.size <= 0 || file.size > MAX_LOCAL_GEOMETRY_BYTES) throw new Error("Choose a non-empty GLB no larger than 128 MiB.");
  const form = new FormData();
  form.append("metadata", JSON.stringify(claims));
  form.append("file", file, file.name);
  const response = await uploadFile("/library/intake-candidates", form);
  if (!response.ok) throw await decodeApiError(response);
  const json = await response.json();
  const decoded = fromJson(GetIntakeCandidateResponseSchema, json, PROTO_READ_OPTIONS);
  const candidate = decoded.candidate;
  if (!candidate || candidate.id !== claims.candidateId || candidate.reviewStatus !== "unreviewed" || candidate.renderStatus !== "not_reviewed") {
    throw new Error("The intake service returned an incomplete or unexpectedly promoted candidate.");
  }
  return candidate;
}

export async function downloadLocalGeometryCandidate(candidate: IntakeCandidateRevision): Promise<Blob> {
  const path = `/library/intake-candidates/${encodeURIComponent(candidate.id)}/revisions/${candidate.revision}/content`;
  const response = await getFile(path);
  if (!response.ok) throw await decodeApiError(response);
  const blob = await response.blob();
  if (blob.size !== Number(candidate.sizeBytes)) throw new Error("Downloaded geometry size differs from candidate metadata.");
  const digest = await crypto.subtle.digest("SHA-256", await blob.arrayBuffer());
  const sha256 = [...new Uint8Array(digest)].map((value) => value.toString(16).padStart(2, "0")).join("");
  if (sha256 !== candidate.originalSha256) throw new Error("Downloaded geometry digest differs from candidate metadata.");
  return blob;
}

export async function getLocalGeometryMaterials(candidate: IntakeCandidateRevision): Promise<IntakeMaterialInventory> {
  const path = `/library/intake-candidates/${encodeURIComponent(candidate.id)}/revisions/${candidate.revision}/materials?revision_sha256=${encodeURIComponent(candidate.revisionSha256)}`;
  const response = await getFile(path);
  if (!response.ok) throw await decodeApiError(response);
  const inventory = await response.json() as IntakeMaterialInventory;
  if (inventory.candidate_id !== candidate.id || inventory.revision !== candidate.revision || inventory.revision_sha256 !== candidate.revisionSha256 || inventory.glb_sha256 !== candidate.originalSha256 || !Array.isArray(inventory.materials) || inventory.materials.length > 256) {
    throw new Error("Material inventory does not match the exact imported candidate revision.");
  }
  return inventory;
}

export async function getLocalGeometryPreview(candidate: IntakeCandidateRevision): Promise<IntakeGeometryPreview> {
  const path = `/library/intake-candidates/${encodeURIComponent(candidate.id)}/revisions/${candidate.revision}/preview?revision_sha256=${encodeURIComponent(candidate.revisionSha256)}`;
  const response = await getFile(path);
  if (!response.ok) throw await decodeApiError(response);
  const headers = response.headers;
  if (headers.get("Content-Type")?.split(";")[0] !== "image/png" || headers.get("X-Intake-Candidate-ID") !== candidate.id ||
      headers.get("X-Intake-Revision") !== String(candidate.revision) || headers.get("X-Intake-Revision-SHA256") !== candidate.revisionSha256 ||
      headers.get("X-Intake-GLB-SHA256") !== candidate.originalSha256 || headers.get("X-Intake-Preview-Review") !== "unreviewed" ||
      headers.get("X-Intake-Preview-Scale") !== "not-estimated") {
    throw new Error("Geometry preview metadata does not match the exact unreviewed candidate revision.");
  }
  const previewSha = headers.get("X-Intake-Preview-SHA256") ?? "";
  if (!/^[0-9a-f]{64}$/.test(previewSha)) throw new Error("Geometry preview digest is invalid.");
  const vertexColors = headers.get("X-Intake-Preview-Vertex-Colors");
  if (vertexColors !== "true" && vertexColors !== "false") throw new Error("Geometry preview color mode is invalid.");
  const blob = await response.blob();
  if (blob.size <= 0 || blob.size > 4 * 1024 * 1024) throw new Error("Geometry preview exceeds the supported byte bound.");
  const bytes = new Uint8Array(await blob.arrayBuffer());
  const signature = [137, 80, 78, 71, 13, 10, 26, 10];
  if (bytes.length < 24 || signature.some((value, index) => bytes[index] !== value)) throw new Error("Geometry preview is not a PNG image.");
  const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
  const width = view.getUint32(16, false), height = view.getUint32(20, false);
  if (width !== 768 || height !== 256) throw new Error("Geometry preview dimensions are outside the fixed three-view profile.");
  const digest = new Uint8Array(await crypto.subtle.digest("SHA-256", bytes));
  const actualSha = [...digest].map((value) => value.toString(16).padStart(2, "0")).join("");
  if (actualSha !== previewSha) throw new Error("Geometry preview bytes do not match their response digest.");
  return { blob, preview_sha256: previewSha, vertex_colors: vertexColors === "true", width, height };
}

export async function correctLocalGeometryMaterial(request: MaterialCorrectionRequest): Promise<MaterialCorrectionCandidate> {
  const path = `/library/intake-candidates/${encodeURIComponent(request.parent_candidate_id)}/revisions/${request.parent_revision}/material-corrections`;
  const response = await fetch(fileURL(path), { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(request), cache: "no-store" });
  if (!response.ok) throw await decodeApiError(response);
  const json = await response.json() as { candidate?: MaterialCorrectionCandidate };
  const candidate = json.candidate;
  if (!candidate || candidate.id !== request.child_candidate_id || candidate.parent_candidate_id !== request.parent_candidate_id || candidate.parent_revision !== request.parent_revision || candidate.parent_sha256 !== request.parent_revision_sha256 || candidate.review_status !== "unreviewed" || candidate.render_status !== "not_reviewed" || candidate.import_method !== "local_material_correction") {
    throw new Error("Material correction did not return the expected unreviewed child candidate.");
  }
  return candidate;
}
