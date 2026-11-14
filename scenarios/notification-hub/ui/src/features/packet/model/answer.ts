import type {
  Ask,
  AskQuestion,
} from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import {
  emptyResponse,
  type AttachmentNoteDraft,
  type PacketDraft,
  type ResponseDraft,
} from "./draft";
import {
  ANSWER_LIMITS,
  isChoiceKind,
  isSectionVerdict,
  questionKind,
  unansweredVerdict,
} from "./questions";

/** One answered question in the shape `Answer` takes; also the immutable record of an attempt. */
export interface ResponsePayload {
  questionId: string;
  selected: string[];
  text: string;
  sections: { sectionKey: string; verdict: string; note: string }[];
}

/** Everything one `Answer` call carries. */
export interface AnswerPayload {
  answer: string;
  note: string;
  responses: ResponsePayload[];
  attachmentNotes: AttachmentNoteDraft[];
}

export const OTHER_ANSWER = "other";

/** The answer keys the Hub accepts: the requester's options, plus "other" which is always allowed. */
export function allowedAnswers(ask: Pick<Ask, "options">): string[] {
  const keys = ask.options.map((option) => option.key);
  return keys.includes(OTHER_ANSWER) ? keys : [...keys, OTHER_ANSWER];
}

const unique = (values: string[]) => [...new Set(values)];

function sectionKeys(question: AskQuestion): string[] {
  return question.sections.map((section) => section.key);
}

function choiceKeys(question: AskQuestion): string[] {
  return question.choices.map((choice) => choice.key);
}

/**
 * The response one question contributes, or null when the operator gave none.
 * A draft is checked against the ask as it is now, so a revised ask can never
 * make a stale draft fail to send: answers to choices or questions that no
 * longer exist are simply left out.
 */
function responseFor(question: AskQuestion, draft: ResponseDraft | undefined, answeredWithOther: boolean): ResponsePayload | null {
  const kind = questionKind(question);
  const given = draft ?? emptyResponse();
  const text = given.text.trim().slice(0, ANSWER_LIMITS.responseTextChars);
  if (isChoiceKind(kind)) {
    const valid = choiceKeys(question);
    const selected = unique(given.selected.filter((key) => valid.includes(key)));
    const kept = kind === "multi_choice" ? selected : selected.slice(0, 1);
    return kept.length || text ? { questionId: question.id, selected: kept, text, sections: [] } : null;
  }
  if (kind === "section_review") {
    const touched = sectionKeys(question).some((key) => given.sections[key]);
    // Skipped headings count as the question's default verdict, so a review
    // that was sent with the requester's options always names every section.
    // An "Other" answer bypasses the form, so an untouched review stays out.
    if (answeredWithOther && !touched && !text) return null;
    const fallback = unansweredVerdict(question);
    return {
      questionId: question.id,
      selected: [],
      text,
      sections: question.sections.map((section) => {
        const entry = given.sections[section.key];
        return {
          sectionKey: section.key,
          verdict: entry && isSectionVerdict(entry.verdict) ? entry.verdict : fallback,
          note: (entry?.note ?? "").trim().slice(0, ANSWER_LIMITS.responseTextChars),
        };
      }),
    };
  }
  return text ? { questionId: question.id, selected: [], text, sections: [] } : null;
}

/** Whether a response counts as answering its question, as the Hub's required rule counts it. */
function answers(question: AskQuestion, response: ResponsePayload | null): boolean {
  if (!response) return false;
  const kind = questionKind(question);
  if (isChoiceKind(kind)) return response.selected.length > 0;
  if (kind === "section_review") return response.sections.length === question.sections.length;
  return response.text !== "";
}

export function buildAnswer(ask: Ask, draft: PacketDraft): AnswerPayload {
  const other = draft.answer === OTHER_ANSWER;
  const responses = ask.questions.flatMap((question) => {
    const response = responseFor(question, draft.responses[question.id], other);
    return response ? [response] : [];
  });
  const known = new Set(ask.attachments.map((attachment) => attachment.name));
  const attachmentNotes = draft.attachmentNotes
    .filter((item) => known.has(item.attachment) && item.note.trim())
    .slice(0, ANSWER_LIMITS.attachmentNotes)
    .map((item) => ({ ...item, note: item.note.trim().slice(0, ANSWER_LIMITS.attachmentNoteChars) }));
  return { answer: draft.answer, note: draft.note.trim(), responses, attachmentNotes };
}

/**
 * Required questions still unanswered. A required question binds only when the
 * operator answers with one of the requester's options: "Other" is the way out
 * of a form that cannot express what they want to say.
 */
export function missingRequired(ask: Ask, draft: PacketDraft): AskQuestion[] {
  if (draft.answer === OTHER_ANSWER) return [];
  const payload = buildAnswer(ask, draft);
  return ask.questions.filter(
    (question) => question.required && !answers(question, payload.responses.find((item) => item.questionId === question.id) ?? null),
  );
}

/** Why the answer cannot be sent yet, or null when it can. */
export type Blocker = "no-answer" | "other-needs-note" | "required-question";

export function blockerFor(ask: Ask, draft: PacketDraft): Blocker | null {
  if (!draft.answer || !allowedAnswers(ask).includes(draft.answer)) return "no-answer";
  if (draft.answer === OTHER_ANSWER && !draft.note.trim()) return "other-needs-note";
  if (missingRequired(ask, draft).length > 0) return "required-question";
  return null;
}

/** The answer-shaped part of an `AskQuestionResponse`; also what an explainer's pre-fill is read into. */
interface ResponseInput {
  selected: readonly string[];
  text: string;
  sections: readonly { sectionKey: string; verdict: string; note: string }[];
}

const fromInput = (question: AskQuestion, recommended: ResponseInput): ResponseDraft => {
  const kind = questionKind(question);
  const base = emptyResponse();
  if (isChoiceKind(kind)) {
    const valid = choiceKeys(question);
    const selected = unique(recommended.selected.filter((key) => valid.includes(key)));
    return { ...base, selected: kind === "multi_choice" ? selected : selected.slice(0, 1) };
  }
  if (kind === "section_review") {
    const valid = sectionKeys(question);
    for (const section of recommended.sections) {
      if (valid.includes(section.sectionKey) && isSectionVerdict(section.verdict)) {
        base.sections[section.sectionKey] = { verdict: section.verdict, note: section.note };
      }
    }
    return base;
  }
  return { ...base, text: recommended.text };
};

function recommendedAnswer(ask: Ask): string {
  return ask.recommended && ask.options.some((option) => option.key === ask.recommended) ? ask.recommended : "";
}

/**
 * True when "Accept all recommended" saves the operator work: the requester
 * recommended answers to typed questions. A recommended main option alone is
 * one tap already.
 */
export function hasQuestionRecommendations(ask: Ask): boolean {
  return ask.questions.some((question) => question.recommended !== undefined);
}

/**
 * "Accept all recommended": copy the main recommendation and every question's
 * recommendation into the draft. It only fills the form; the operator still
 * reads it and sends it themselves.
 */
export function withRecommendations(ask: Ask, draft: PacketDraft): PacketDraft {
  const responses = { ...draft.responses };
  for (const question of ask.questions) {
    if (question.recommended) responses[question.id] = fromInput(question, question.recommended);
  }
  return { ...draft, answer: recommendedAnswer(ask) || draft.answer, responses };
}

const isRecord = (value: unknown): value is Record<string, unknown> =>
  Boolean(value) && typeof value === "object" && !Array.isArray(value);

const stringList = (value: unknown): string[] =>
  Array.isArray(value) ? value.filter((item): item is string => typeof item === "string") : [];

/** Read an explainer's response in either proto JSON spelling (camelCase or snake_case). */
function parseResponseValue(value: unknown): ResponseInput | null {
  if (!isRecord(value)) return null;
  const sections = Array.isArray(value.sections)
    ? value.sections.flatMap((item) => {
        if (!isRecord(item)) return [];
        const key = item.sectionKey ?? item.section_key;
        return typeof key === "string" && typeof item.verdict === "string"
          ? [{ sectionKey: key, verdict: item.verdict, note: typeof item.note === "string" ? item.note : "" }]
          : [];
      })
    : [];
  return {
    selected: stringList(value.selected),
    text: typeof value.text === "string" ? value.text : "",
    sections,
  };
}

/**
 * Apply a pre-fill sent by an explainer page (`hub.prefill`). Only the form
 * draft changes, only for things the ask really has, and nothing is sent:
 * an invalid or unknown target leaves the draft untouched.
 */
export function withPrefill(ask: Ask, draft: PacketDraft, target: string, value: unknown): PacketDraft {
  if (target === "answer") {
    const key = typeof value === "string" ? value : "";
    return ask.options.some((option) => option.key === key) ? { ...draft, answer: key } : draft;
  }
  const question = ask.questions.find((item) => item.id === target);
  if (!question) return draft;
  // A bare string is the shorthand for "this choice key" (or this text).
  const parsed: ResponseInput | null = typeof value === "string"
    ? isChoiceKind(questionKind(question)) ? { selected: [value], text: "", sections: [] } : { selected: [], text: value, sections: [] }
    : parseResponseValue(value);
  if (!parsed) return draft;
  const next = fromInput(question, parsed);
  const empty = next.selected.length === 0 && !next.text && Object.keys(next.sections).length === 0;
  return empty ? draft : { ...draft, responses: { ...draft.responses, [question.id]: next } };
}
