import { describe, expect, it } from "vitest";

import { attachmentNoteFor, emptyDraft, emptyResponse, normalizeDraft, withAttachmentNote, withResponse } from "./draft";

describe("normalizeDraft", () => {
  it("restores a stored draft exactly", () => {
    const stored = {
      answer: "approve",
      note: "ship it",
      responses: { q1: { selected: ["a"], text: "because", sections: { s1: { verdict: "ok", note: "fine" } } } },
      attachmentNotes: [{ attachment: "plan", path: "a.md", note: "see this" }],
    };
    expect(normalizeDraft(stored)).toEqual(stored);
  });

  it("accepts a draft saved before questions and attachment notes existed", () => {
    expect(normalizeDraft({ answer: "hold", note: "" })).toEqual({ answer: "hold", note: "", responses: {}, attachmentNotes: [] });
  });

  it("refuses anything that is not a draft, so a corrupt entry never reaches the form", () => {
    for (const value of [null, undefined, "text", 3, [], { answer: 1, note: "" }, { answer: "", note: null }]) {
      expect(normalizeDraft(value)).toBeNull();
    }
  });

  it("keeps what is valid in a damaged draft and drops the rest", () => {
    const draft = normalizeDraft({
      answer: "approve",
      note: "",
      responses: {
        good: { selected: ["a", 7, "b"], text: "t", sections: { s: { verdict: "ok", note: 4 }, bad: "no" } },
        broken: "not a response",
      },
      attachmentNotes: [{ attachment: "plan", note: "n" }, { attachment: 3, note: "x" }, "junk"],
    });
    expect(draft?.responses.good).toEqual({ selected: ["a", "b"], text: "t", sections: { s: { verdict: "ok", note: "" } } });
    expect(draft?.responses.broken).toEqual(emptyResponse());
    expect(draft?.attachmentNotes).toEqual([{ attachment: "plan", path: "", note: "n" }]);
  });
});

describe("withResponse", () => {
  it("replaces one question's response without touching the others", () => {
    const draft = withResponse(withResponse(emptyDraft(), "q1", { ...emptyResponse(), text: "one" }), "q2", { ...emptyResponse(), text: "two" });
    const changed = withResponse(draft, "q1", { ...emptyResponse(), text: "uno" });
    expect(changed.responses.q1?.text).toBe("uno");
    expect(changed.responses.q2?.text).toBe("two");
    expect(draft.responses.q1?.text).toBe("one");
  });
});

describe("attachment notes", () => {
  it("adds a note for a file, and a separate note for a file inside a folder", () => {
    let draft = withAttachmentNote(emptyDraft(), "plan", "", "whole file");
    draft = withAttachmentNote(draft, "folder", "docs/a.md", "inside");
    expect(attachmentNoteFor(draft, "plan")).toBe("whole file");
    expect(attachmentNoteFor(draft, "folder", "docs/a.md")).toBe("inside");
    expect(attachmentNoteFor(draft, "folder")).toBe("");
  });

  it("edits a note in place, keeping the order the operator wrote them in", () => {
    let draft = withAttachmentNote(emptyDraft(), "a", "", "first");
    draft = withAttachmentNote(draft, "b", "", "second");
    draft = withAttachmentNote(draft, "a", "", "first, revised");
    expect(draft.attachmentNotes.map((note) => note.note)).toEqual(["first, revised", "second"]);
  });

  it("removes a note when it is cleared or only whitespace", () => {
    const draft = withAttachmentNote(withAttachmentNote(emptyDraft(), "a", "", "keep"), "b", "", "drop");
    const cleared = withAttachmentNote(withAttachmentNote(draft, "b", "", "   "), "c", "", "");
    expect(cleared.attachmentNotes).toEqual([{ attachment: "a", path: "", note: "keep" }]);
  });
});
