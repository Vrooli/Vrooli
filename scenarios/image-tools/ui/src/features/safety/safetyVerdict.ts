import type { Job } from "../../api/jobs";

/** The reviewer's stored verdict for a job, as the job's result_meta records it. */
export interface SafetyVerdict {
  action: string;
  reasonCode: string;
  review: string;
  score: number;
  refusedRefs: string[];
  sensitiveRefs: string[];
}

const LABELABLE_ACTIONS = new Set(["refuse", "flag", "partial_refused", "shadow_refused"]);
const LABELABLE_REVIEWS = new Set(["refused", "partial_refused", "shadow_refused", "flagged"]);

function splitRefs(raw: string | undefined): string[] {
  return (raw ?? "")
    .split(",")
    .map((ref) => ref.trim())
    .filter((ref) => ref !== "");
}

/**
 * Read the verdict worth labelling from a job's result metadata. Returns null
 * when the reviewer passed the output (nothing to judge) or the job predates the
 * structured safety record. A technical reviewer failure is also excluded: there
 * is no verdict to agree or disagree with.
 */
export function safetyVerdictOf(job: Job): SafetyVerdict | null {
  const meta = job.resultMeta;
  const action = (meta.safety_action || "").trim();
  const review = (meta.safety_review || "").trim();
  if (!LABELABLE_ACTIONS.has(action) && !LABELABLE_REVIEWS.has(review)) {
    return null;
  }
  const score = Number.parseFloat(meta.nsfw_score || "0");
  return {
    action,
    reasonCode: (meta.safety_reason || "").trim(),
    review,
    score: Number.isFinite(score) ? score : 0,
    refusedRefs: splitRefs(meta.safety_refused_refs),
    sensitiveRefs: splitRefs(meta.sensitive_refs),
  };
}

/** The short badge label for a verdict's state. */
export function safetyVerdictKind(verdict: SafetyVerdict): "refused" | "partial" | "shadow" | "flagged" {
  if (verdict.review === "partial_refused" || verdict.action === "partial_refused") return "partial";
  if (verdict.review === "shadow_refused" || verdict.action === "shadow_refused") return "shadow";
  if (verdict.action === "flag" || verdict.review === "flagged") return "flagged";
  return "refused";
}
