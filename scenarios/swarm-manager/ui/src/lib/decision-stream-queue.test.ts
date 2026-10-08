import { describe, expect, it } from "vitest";
import {
  decisionParentKey,
  decisionQuestionKey,
  getUnresolvedDecisionQuestions,
  normalizeDecisionIndex,
} from "./decision-stream-queue";
import type { CrossItemQuestion } from "./command-post-utils";

function makeQuestion(parentName: string, id: string): CrossItemQuestion {
  return {
    parentKind: "idea",
    parentName,
    parentTitle: parentName,
    question: {
      id,
      source: "workshop",
      item_kind: "idea",
      item_name: parentName,
      topic: id,
    },
  };
}

describe("decision-stream-queue", () => {
  it("prunes answered, deleted, and snoozed questions while preserving remaining order", () => {
    const q1 = makeQuestion("first", "q1");
    const q2 = makeQuestion("first", "q2");
    const q3 = makeQuestion("second", "q3");
    const q4 = makeQuestion("third", "q4");

    const unresolved = getUnresolvedDecisionQuestions({
      questions: [q1, q2, q3, q4],
      answeredKeys: new Set([decisionQuestionKey(q1)]),
      deletedIds: new Set(["q2"]),
      snoozedParentKeys: new Set([decisionParentKey("idea", "second")]),
    });

    expect(unresolved).toEqual([q4]);
  });

  it("normalizes indexes after queue shape changes", () => {
    expect(normalizeDecisionIndex(3, 2)).toBe(1);
    expect(normalizeDecisionIndex(-2, 2)).toBe(0);
    expect(normalizeDecisionIndex(4, 0)).toBe(0);
  });
});
