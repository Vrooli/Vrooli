import type { TFunction } from "i18next";
import type { AskQuestion } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { strings } from "../../../consts/strings";
import { VERDICT_LABEL } from "../form/verdictLabels";
import type { ResponsePayload } from "./answer";
import { isChoiceKind, questionKind, unansweredVerdict } from "./questions";

/**
 * What was said to one question, as lines of plain text in the operator's own
 * words: the labels of the choices (never their keys), each section's verdict,
 * or the typed text. Shared by the review before sending and the record after.
 */
export function describeResponse(question: AskQuestion, response: ResponsePayload | undefined, t: TFunction): string[] {
  if (!response) return [];
  const kind = questionKind(question);
  if (isChoiceKind(kind)) {
    const labels = response.selected.map((key) => question.choices.find((choice) => choice.key === key)?.label ?? key);
    return [labels.join(", "), response.text].filter(Boolean);
  }
  if (kind === "section_review") {
    const fallback = unansweredVerdict(question);
    const differing = response.sections.filter((section) => section.verdict !== fallback || section.note);
    const lines = differing.map((section) => {
      const title = question.sections.find((item) => item.key === section.sectionKey)?.title ?? section.sectionKey;
      return `${title}: ${t(VERDICT_LABEL[section.verdict === "change" || section.verdict === "drop" ? section.verdict : "approve"])}${section.note ? ` — ${section.note}` : ""}`;
    });
    const rest = response.sections.length - differing.length;
    if (rest === 0) return lines;
    const verdict = t(VERDICT_LABEL[fallback]);
    return [...lines, t(differing.length > 0 ? strings.packet.form.otherSections : strings.packet.form.allSections, { verdict })];
  }
  return [response.text];
}
