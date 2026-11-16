import type { MutationPreviewResponse } from "./api";

/** True only when the reviewed repository mutation still describes current state. */
export function commitPreviewMatchesCurrent(reviewed: MutationPreviewResponse, current: MutationPreviewResponse): boolean {
  return reviewed.repositoryId === current.repositoryId &&
    reviewed.operation === current.operation &&
    reviewed.expectedRevision === current.expectedRevision &&
    reviewed.subjectDigest === current.subjectDigest;
}
