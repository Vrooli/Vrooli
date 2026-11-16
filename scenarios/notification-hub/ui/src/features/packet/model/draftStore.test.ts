import { afterEach, describe, expect, it } from "vitest";

import { emptyDraft } from "./draft";
import { clearAskDraft, clearAskDrafts, readAskAttempt, readAskDraft, writeAskAttempt, writeAskDraft } from "./draftStore";

afterEach(() => {
  clearAskDrafts();
  window.sessionStorage.clear();
});

describe("draft store", () => {
  it("keeps the unsent answer per signed-in subject and ask", () => {
    writeAskDraft("owner-1", "ask-1", { ...emptyDraft(), answer: "close", note: "n", responses: { q: { selected: ["a"], text: "", sections: {} } } });
    expect(readAskDraft("owner-1", "ask-1").responses.q?.selected).toEqual(["a"]);
    expect(readAskDraft("owner-2", "ask-1")).toEqual(emptyDraft());
    expect(readAskDraft("owner-1", "ask-2")).toEqual(emptyDraft());
  });

  it("restores from browser storage after the in-memory copy is gone", () => {
    writeAskDraft("owner-1", "ask-1", { ...emptyDraft(), answer: "close" });
    const key = window.sessionStorage.key(0) ?? "";
    clearAskDrafts();
    window.sessionStorage.setItem(key, JSON.stringify({ draft: { answer: "close", note: "kept" } }));
    // A draft written before questions existed has no responses or notes.
    expect(readAskDraft("owner-1", "ask-1")).toEqual({ answer: "close", note: "kept", responses: {}, attachmentNotes: [], answerChanges: {}, answerImages: [] });
  });

  it("ignores a corrupt stored draft instead of failing to read the ask", () => {
    writeAskDraft("owner-1", "ask-1", { ...emptyDraft(), answer: "close" });
    const key = window.sessionStorage.key(0) ?? "";
    clearAskDrafts();
    window.sessionStorage.setItem(key, "{not json");
    expect(readAskDraft("owner-1", "ask-1")).toEqual(emptyDraft());
  });

  it("returns an attempt only to the subject that sent it, with its responses intact", () => {
    const attempt = { subject: "owner-1", answer: "close", note: "", responses: [{ questionId: "q", selected: ["a"], text: "", sections: [] }], attachmentNotes: [], changes: {}, images: [] };
    writeAskAttempt("owner-1", "ask-1", attempt);
    expect(readAskAttempt("owner-1", "ask-1")).toEqual(attempt);
    expect(readAskAttempt("owner-2", "ask-1")).toBeNull();
  });

  it("clears one draft without touching another", () => {
    writeAskDraft("owner-1", "ask-1", { ...emptyDraft(), answer: "a" });
    writeAskDraft("owner-1", "ask-2", { ...emptyDraft(), answer: "b" });
    clearAskDraft("owner-1", "ask-1");
    expect(readAskDraft("owner-1", "ask-1").answer).toBe("");
    expect(readAskDraft("owner-1", "ask-2").answer).toBe("b");
  });
});
