import type { CrossItemQuestion } from "./command-post-utils";
import type { BacklogKind } from "../types";

export function decisionQuestionKey(ciq: CrossItemQuestion): string {
  return `${ciq.parentKind}/${ciq.parentName}/${ciq.question.source}/${ciq.question.id}`;
}

export function decisionParentKey(kind: BacklogKind, name: string): string {
  return `backlog:${kind}/${name}`;
}

export interface DecisionQueueState {
  questions: CrossItemQuestion[];
  answeredKeys: Set<string>;
  deletedIds: Set<string>;
  snoozedParentKeys: Set<string>;
}

export function getUnresolvedDecisionQuestions({
  questions,
  answeredKeys,
  deletedIds,
  snoozedParentKeys,
}: DecisionQueueState): CrossItemQuestion[] {
  return questions.filter((ciq) => {
    if (snoozedParentKeys.has(decisionParentKey(ciq.parentKind, ciq.parentName))) return false;
    if (deletedIds.has(ciq.question.id)) return false;
    return !answeredKeys.has(decisionQuestionKey(ciq));
  });
}

export function normalizeDecisionIndex(index: number, total: number): number {
  if (total <= 0) return 0;
  return Math.min(Math.max(0, index), total - 1);
}
