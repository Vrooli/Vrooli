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
