import {
  SafetyFeedbackLabel,
  type ListSafetyFeedbackResponse,
  type SafetyCorpus,
  type SafetyFeedbackCase,
  type SafetyFeedbackStats,
} from "@vrooli/proto-types/image-tools/v1/safety/safety_pb";

import { safetyClient } from "./safety";

export { SafetyFeedbackLabel };
export type {
  ListSafetyFeedbackResponse,
  SafetyCorpus,
  SafetyFeedbackCase,
  SafetyFeedbackStats,
};

/**
 * Who the UI records as the judge. The API trusts the local caller; the value is
 * provenance for the corpus (the CLI sends the agent or user name instead).
 */
export const FEEDBACK_ACTOR = "operator";

/** The UI's three judgements, mapped onto the wire enum. */
export type FeedbackChoice = "legitimate" | "confirmed" | "uncertain";

const CHOICE_LABEL: Record<FeedbackChoice, SafetyFeedbackLabel> = {
  legitimate: SafetyFeedbackLabel.FALSE_POSITIVE,
  confirmed: SafetyFeedbackLabel.CONFIRMED,
  uncertain: SafetyFeedbackLabel.UNCERTAIN,
};

const LABEL_CHOICE: Record<SafetyFeedbackLabel, FeedbackChoice | null> = {
  [SafetyFeedbackLabel.UNSPECIFIED]: null,
  [SafetyFeedbackLabel.FALSE_POSITIVE]: "legitimate",
  [SafetyFeedbackLabel.CONFIRMED]: "confirmed",
  [SafetyFeedbackLabel.UNCERTAIN]: "uncertain",
};

/** The choice a stored label represents, or null for an unlabelled case. */
export const feedbackChoice = (label: SafetyFeedbackLabel): FeedbackChoice | null => LABEL_CHOICE[label] ?? null;

/** True when the stored label is decisive (counts toward the FP rate). */
export const isDecisive = (label: SafetyFeedbackLabel): boolean =>
  label === SafetyFeedbackLabel.FALSE_POSITIVE || label === SafetyFeedbackLabel.CONFIRMED;

/**
 * Record (or replace) the operator's judgement of one reviewed output. An empty
 * `outputRef` labels the job; a ref labels one variation.
 */
export async function recordSafetyFeedback(
  jobId: string,
  choice: FeedbackChoice,
  options: { outputRef?: string; note?: string } = {},
): Promise<SafetyFeedbackCase | undefined> {
  const resp = await safetyClient.recordSafetyFeedback({
    jobId,
    outputRef: options.outputRef?.trim() ?? "",
    label: CHOICE_LABEL[choice],
    note: options.note?.trim() ?? "",
    actor: FEEDBACK_ACTOR,
  });
  return resp;
}

/** Remove one judgement. */
export async function clearSafetyFeedback(jobId: string, outputRef = ""): Promise<boolean> {
  const resp = await safetyClient.clearSafetyFeedback({ jobId, outputRef });
  return resp.cleared;
}

/** List the labelled corpus newest first. */
export async function listSafetyFeedback(
  filters: { label?: FeedbackChoice; reasonCode?: string; modelId?: string; operation?: string; jobId?: string; limit?: number; offset?: number } = {},
): Promise<ListSafetyFeedbackResponse> {
  return safetyClient.listSafetyFeedback({
    label: filters.label ? CHOICE_LABEL[filters.label] : SafetyFeedbackLabel.UNSPECIFIED,
    reasonCode: filters.reasonCode ?? "",
    modelId: filters.modelId ?? "",
    operation: filters.operation ?? "",
    jobId: filters.jobId ?? "",
    limit: filters.limit ?? 0,
    offset: filters.offset ?? 0,
  });
}

/** Aggregate the labelled corpus. */
export async function getSafetyFeedbackStats(since?: Date): Promise<SafetyFeedbackStats> {
  return safetyClient.getSafetyFeedbackStats(
    since ? { since: { seconds: BigInt(Math.floor(since.getTime() / 1000)), nanos: 0 } } : {},
  );
}

/** Export the corpus as JSON Lines. */
export async function exportSafetyCorpus(
  options: { label?: FeedbackChoice; since?: Date; limit?: number } = {},
): Promise<SafetyCorpus> {
  return safetyClient.exportSafetyCorpus({
    label: options.label ? CHOICE_LABEL[options.label] : SafetyFeedbackLabel.UNSPECIFIED,
    since: options.since
      ? { seconds: BigInt(Math.floor(options.since.getTime() / 1000)), nanos: 0 }
      : undefined,
    limit: options.limit ?? 0,
  });
}
