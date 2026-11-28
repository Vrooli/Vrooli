/**
 * Which images are sensitive.
 *
 * The server flags a job's outputs (`result_meta.sensitive` + `sensitive_refs`)
 * when an NSFW-capable model made them or the output review scored them sexual.
 * The UI blurs every flagged image behind a tap-to-reveal. Two lookups cover every
 * surface:
 *
 *   - Blob-keyed surfaces (Library, Activity, Home) read the flag from the job.
 *   - Surfaces that hold an in-memory image (the Workspace canvas, history,
 *     before/after, mask brush, variation grid, Compare) only have an object URL
 *     and a File. Those are marked here when the result arrives and the mark
 *     follows the image through every derived step (a crop of a sensitive image
 *     is sensitive), so a later edit cannot quietly un-blur it.
 *
 * The marks live in memory for the page's lifetime; a reload re-derives them from
 * the jobs list (Library/Activity) or re-opens the result through the Library,
 * which marks it again.
 */
import type { Job } from "../api/jobs";

const META_SENSITIVE = "sensitive";
const META_SENSITIVE_REFS = "sensitive_refs";

const sensitiveUrls = new Set<string>();
const sensitiveFiles = new WeakSet<File>();

/** The output refs of a job that the server flagged sensitive. */
export function sensitiveRefsOf(meta: Readonly<Record<string, string>> | undefined): string[] {
  if (!meta || meta[META_SENSITIVE] !== "true") {
    return [];
  }
  return (meta[META_SENSITIVE_REFS] ?? "")
    .split(",")
    .map((ref) => ref.trim())
    .filter(Boolean);
}

/** True when the job's output `ref` is flagged sensitive. */
export function isSensitiveRef(job: Pick<Job, "resultMeta">, ref: string): boolean {
  return sensitiveRefsOf(job.resultMeta).includes(ref);
}

/** Mark an in-memory image (its object URL and/or the File behind it). */
export function markSensitive(target: { url?: string; file?: File }): void {
  if (target.url) {
    sensitiveUrls.add(target.url);
  }
  if (target.file) {
    sensitiveFiles.add(target.file);
  }
}

export const isSensitiveUrl = (url: string | null | undefined): boolean => !!url && sensitiveUrls.has(url);
export const isSensitiveFile = (file: File | null | undefined): boolean => !!file && sensitiveFiles.has(file);

/**
 * Carry sensitivity from the image an edit consumed to what it produced. Returns
 * whether the source was sensitive so the caller can branch on it.
 */
export function inheritSensitivity(from: File | null | undefined, to: { url?: string; file?: File }): boolean {
  if (!isSensitiveFile(from)) {
    return false;
  }
  markSensitive(to);
  return true;
}
