import { operatorAnswerValue } from "./operatorAnswer";

export type DraftableQuestion = {
  id: string;
  kind: string;
  defaultValue?: unknown;
};

export type OperatorQuestionDraft = {
  values: Record<string, unknown>;
  declined: Record<string, boolean>;
};

const questionKeyPrefix = "question_";
const declinedKeyPrefix = "declined_";

function encodedID(id: string): string {
  return Array.from(new TextEncoder().encode(id), (byte) => byte.toString(16).padStart(2, "0")).join("");
}

function questionKey(id: string): string {
  return `${questionKeyPrefix}${encodedID(id)}`;
}

function declinedKey(id: string): string {
  return `${declinedKeyPrefix}${encodedID(id)}`;
}

function isBooleanQuestion(question: DraftableQuestion): boolean {
  return ["boolean", "confirm", "confirmation"].includes(question.kind)
    || typeof question.defaultValue === "boolean";
}

export function operatorQuestionDefaultValue(question: DraftableQuestion): unknown {
  if (isBooleanQuestion(question)) return question.defaultValue === true || question.defaultValue === "true";
  return question.defaultValue ?? "";
}

function sameValue(left: unknown, right: unknown): boolean {
  return JSON.stringify(left) === JSON.stringify(right);
}

/** Draft only non-secret setup answers; secret input values never enter this map. */
export function encodeOperatorQuestionDraft(
  questions: DraftableQuestion[],
  values: Record<string, unknown>,
  declined: Record<string, boolean>,
): Record<string, string> {
  const choices: Record<string, string> = {};
  for (const question of questions) {
    if (question.kind === "secret") continue;
    const value = values[question.id] ?? operatorQuestionDefaultValue(question);
    const defaultValue = operatorQuestionDefaultValue(question);
    if (!sameValue(value, defaultValue)) choices[questionKey(question.id)] = operatorAnswerValue(value);
    if (declined[question.id]) choices[declinedKey(question.id)] = "true";
  }
  return choices;
}

/** Restore values only for questions that are present in the current owner response. */
export function decodeOperatorQuestionDraft(
  questions: DraftableQuestion[],
  choices: Record<string, string> = {},
): OperatorQuestionDraft {
  const values: Record<string, unknown> = {};
  const declined: Record<string, boolean> = {};
  for (const question of questions) {
    const saved = choices[questionKey(question.id)];
    if (question.kind !== "secret" && saved !== undefined) {
      values[question.id] = isBooleanQuestion(question) ? saved === "true" : saved;
    }
    declined[question.id] = choices[declinedKey(question.id)] === "true";
  }
  return { values, declined };
}
