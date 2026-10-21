import { fromJson } from "@bufbuild/protobuf";
import type { IntakeCandidateRevision } from "@vrooli/proto-types/device-studio/v1/library/library_pb";
import { GetIntakeCandidateResponseSchema, LibraryCatalogService } from "@vrooli/proto-types/device-studio/v1/library/library_pb";
import { createClient } from "@connectrpc/connect";
import { decodeApiError, getFile, PROTO_READ_OPTIONS, transport, uploadFile } from "./client";

const catalog = createClient(LibraryCatalogService, transport);
export const MAX_LOCAL_GEOMETRY_BYTES = 128 * 1024 * 1024;

export function listLocalGeometryCandidates(afterCandidateId = "") {
  if (afterCandidateId && !/^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/.test(afterCandidateId)) {
    throw new Error("The intake cursor must be a stable candidate ID.");
  }
  return catalog.listIntakeCandidates({ afterCandidateId, pageSize: 100 });
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
