import type { AskQuestion } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

/**
 * The question types this reader draws. `type` is a free string on the wire so
 * a newer sender never breaks an older reader: anything not listed here is
 * drawn as a prompt with a text box and answered with `text`
 * (docs/concepts/DECISION_PACKETS.md, robustness rule 3).
 */
export type QuestionKind = "single_choice" | "multi_choice" | "variant" | "section_review" | "text";

const KNOWN_KINDS: ReadonlySet<string> = new Set<QuestionKind>(["single_choice", "multi_choice", "variant", "section_review", "text"]);

export function questionKind(question: Pick<AskQuestion, "type">): QuestionKind {
  return KNOWN_KINDS.has(question.type) ? (question.type as QuestionKind) : "text";
}

export type ChoiceKind = "single_choice" | "multi_choice" | "variant";

export function isChoiceKind(kind: QuestionKind): kind is ChoiceKind {
  return kind === "single_choice" || kind === "multi_choice" || kind === "variant";
}

export const SECTION_VERDICTS = ["approve", "change", "drop"] as const;
export type SectionVerdict = (typeof SECTION_VERDICTS)[number];

export function isSectionVerdict(value: string): value is SectionVerdict {
  return (SECTION_VERDICTS as readonly string[]).includes(value);
}

/** The verdict a section takes when the operator leaves it alone. */
export function unansweredVerdict(question: Pick<AskQuestion, "unansweredVerdict">): SectionVerdict {
  return isSectionVerdict(question.unansweredVerdict) ? question.unansweredVerdict : "approve";
}

/** Limits the Hub enforces on an answer (docs/concepts/DECISION_PACKETS.md). */
export const ANSWER_LIMITS = {
  responseTextChars: 4000,
  attachmentNotes: 24,
  attachmentNoteChars: 2000,
  contextQuestionChars: 2000,
} as const;
