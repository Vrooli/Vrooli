import { describe, expect, it } from "vitest";

import { makeAsk, makeAttachment, makeQuestion } from "../mocks/askFactory";
import { blockerFor, buildAnswer, missingRequired, withPrefill, withRecommendations } from "./answer";
import { emptyDraft, withAttachmentNote, type PacketDraft } from "./draft";

const variant = makeQuestion({
  id: "set",
  type: "variant",
  prompt: "Which set?",
  required: true,
  choices: [{ key: "a", label: "A" }, { key: "b", label: "B" }],
  recommended: { selected: ["b"] },
});
const shots = makeQuestion({
  id: "shots",
  type: "multi_choice",
  prompt: "Which shots?",
  choices: [{ key: "intro", label: "Intro" }, { key: "demo", label: "Demo" }, { key: "outro", label: "Outro" }],
  recommended: { selected: ["intro", "demo"] },
});
const review = makeQuestion({
  id: "brief",
  type: "section_review",
  prompt: "Review",
  unansweredVerdict: "approve",
  sections: [{ key: "goal", title: "Goal" }, { key: "timeline", title: "Timeline" }],
  recommended: { sections: [{ sectionKey: "timeline", verdict: "change", note: "shorter" }] },
});
const note = makeQuestion({ id: "why", type: "free-form-from-the-future", prompt: "Anything else?", required: true });

const ask = makeAsk({
  recommended: "re-aim",
  questions: [variant, shots, review, note],
  attachments: [makeAttachment({ name: "character.glb" })],
});
const draftWith = (patch: Partial<PacketDraft>): PacketDraft => ({ ...emptyDraft(), ...patch });

describe("buildAnswer", () => {
  it("sends only what the ask still has, in question order, without empty responses", () => {
    const payload = buildAnswer(
      ask,
      draftWith({
        answer: "close",
        note: "  keep it short ",
        responses: {
          shots: { selected: ["demo", "gone", "demo"], text: "", sections: {} },
          set: { selected: ["b", "a"], text: " better ", sections: {} },
          removed: { selected: ["x"], text: "", sections: {} },
          why: { selected: [], text: "  because ", sections: {} },
        },
      }),
    );
    expect(payload.note).toBe("keep it short");
    expect(payload.changes).toEqual({});
    expect(payload.images).toEqual([]);
    expect(payload.responses.map((response) => response.questionId)).toEqual(["set", "shots", "brief", "why"]);
    expect(payload.responses[0]).toMatchObject({ selected: ["b"], text: "better" });
    expect(payload.responses[1]?.selected).toEqual(["demo"]);
  });

  it("gives every section a verdict, so a skipped heading takes the question's default", () => {
    const payload = buildAnswer(ask, draftWith({ answer: "close", responses: { brief: { selected: [], text: "", sections: { timeline: { verdict: "drop", note: "no" } } } } }));
    expect(payload.responses.find((response) => response.questionId === "brief")?.sections).toEqual([
      { sectionKey: "goal", verdict: "approve", note: "" },
      { sectionKey: "timeline", verdict: "drop", note: "no" },
    ]);
  });

  it("leaves an untouched review out of an Other answer, which bypasses the form", () => {
    const payload = buildAnswer(ask, draftWith({ answer: "other", note: "Something else" }));
    expect(payload.responses).toEqual([]);
  });

  it("drops notes on attachments the ask no longer has and empty notes", () => {
    let draft = draftWith({ answer: "close" });
    draft = withAttachmentNote(draft, "character.glb", "", "Hair is too glossy");
    draft = withAttachmentNote(draft, "removed.png", "", "stale");
    expect(buildAnswer(ask, draft).attachmentNotes).toEqual([{ attachment: "character.glb", path: "", note: "Hair is too glossy" }]);
  });

  it("returns a redesign region ID and correction text in the packet answer payload", () => {
    const captureAsk = makeAsk({ attachments: [makeAttachment({ name: "capture-review", folder: true })] });
    const draft = withAttachmentNote(emptyDraft(), "capture-review", "regions/nav-main.json", "Keep the selected destination visible.");
    expect(buildAnswer(captureAsk, draft).attachmentNotes).toEqual([
      { attachment: "capture-review", path: "regions/nav-main.json", note: "Keep the selected destination visible." },
    ]);
  });
});

describe("required questions", () => {
  it("binds when the operator picks one of the requester's options", () => {
    const missing = missingRequired(ask, draftWith({ answer: "close" })).map((question) => question.id);
    expect(missing).toEqual(["set", "why"]);
    expect(blockerFor(ask, draftWith({ answer: "close" }))).toBe("required-question");
  });

  it("does not bind when the operator answers Other, but Other needs its note", () => {
    expect(missingRequired(ask, draftWith({ answer: "other" }))).toEqual([]);
    expect(blockerFor(ask, draftWith({ answer: "other" }))).toBe("other-needs-note");
    expect(blockerFor(ask, draftWith({ answer: "other", note: "x" }))).toBeNull();
  });

  it("treats an unknown question type as text that must be written", () => {
    const draft = draftWith({
      answer: "close",
      responses: { set: { selected: ["a"], text: "", sections: {} }, why: { selected: [], text: "ok", sections: {} } },
    });
    expect(blockerFor(ask, draft)).toBeNull();
  });

  it("asks for an answer before anything else", () => {
    expect(blockerFor(ask, emptyDraft())).toBe("no-answer");
    expect(blockerFor(ask, draftWith({ answer: "not-an-option" }))).toBe("no-answer");
  });
});

describe("withRecommendations", () => {
  it("copies the main and every question recommendation without sending anything", () => {
    const next = withRecommendations(ask, draftWith({ note: "mine" }));
    expect(next.answer).toBe("re-aim");
    expect(next.note).toBe("mine");
    expect(next.responses.set?.selected).toEqual(["b"]);
    expect(next.responses.shots?.selected).toEqual(["intro", "demo"]);
    expect(next.responses.brief?.sections.timeline).toEqual({ verdict: "change", note: "shorter" });
  });
});

describe("withPrefill (from an explainer page)", () => {
  it("fills the main answer only with an option the requester offered", () => {
    expect(withPrefill(ask, emptyDraft(), "answer", "close").answer).toBe("close");
    const unchanged = emptyDraft();
    expect(withPrefill(ask, unchanged, "answer", "nope")).toBe(unchanged);
  });

  it("accepts a bare choice key, either JSON spelling of a response, and ignores the rest", () => {
    expect(withPrefill(ask, emptyDraft(), "set", "a").responses.set?.selected).toEqual(["a"]);
    expect(withPrefill(ask, emptyDraft(), "shots", { selected: ["outro"] }).responses.shots?.selected).toEqual(["outro"]);
    const reviewed = withPrefill(ask, emptyDraft(), "brief", { sections: [{ section_key: "goal", verdict: "drop", note: "x" }] });
    expect(reviewed.responses.brief?.sections.goal).toEqual({ verdict: "drop", note: "x" });
    const unchanged = emptyDraft();
    expect(withPrefill(ask, unchanged, "missing", "a")).toBe(unchanged);
    expect(withPrefill(ask, unchanged, "set", "not-a-choice")).toBe(unchanged);
    expect(withPrefill(ask, unchanged, "set", 42)).toBe(unchanged);
  });
});
