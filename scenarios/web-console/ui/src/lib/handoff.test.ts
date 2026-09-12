import { describe, expect, it } from "vitest";
import { allDelivered, anyUnsent, renderHandoffPrompt } from "./handoff";

describe("renderHandoffPrompt", () => {
  it("returns the payload alone when the template is empty", () => {
    expect(renderHandoffPrompt("", "/tmp/notes.md")).toBe("/tmp/notes.md");
    expect(renderHandoffPrompt("   \n ", "/tmp/notes.md")).toBe("/tmp/notes.md");
  });

  it("substitutes every occurrence of the placeholder", () => {
    expect(renderHandoffPrompt("Read {{payload}} then summarise {{payload}}", "notes.md"))
      .toBe("Read notes.md then summarise notes.md");
  });

  it("appends after a blank line when the template has no placeholder", () => {
    expect(renderHandoffPrompt("Take a look at this", "notes.md"))
      .toBe("Take a look at this\n\nnotes.md");
  });

  it("returns the template unchanged when there is no payload and no placeholder", () => {
    expect(renderHandoffPrompt("Start reviewing", "")).toBe("Start reviewing");
  });

  it("substitutes an empty payload rather than leaving the placeholder visible", () => {
    expect(renderHandoffPrompt("Review {{payload}} now", "")).toBe("Review  now");
  });

  // The generalization requirement, as a test. The function must treat every
  // payload identically: a markdown plan, a source file, a URL, and a prose
  // passage all take the same path, because nothing here may know what a
  // payload is.
  it("treats every payload identically regardless of shape", () => {
    const template = "Handle {{payload}}";
    const payloads = [
      "/home/me/.vrooli/plans/some-plan.md",
      "src/index.ts:42",
      "https://example.com/thing",
      "the third paragraph, verbatim",
      "C:\\Users\\me\\notes.txt",
      "",
    ];
    for (const payload of payloads) {
      expect(renderHandoffPrompt(template, payload)).toBe(`Handle ${payload}`);
    }
  });

  // Prohibition 7: no workflow assumption in substitution. If the function
  // ever grew a branch on extension or path shape, these two would diverge.
  it("does not branch on file extension", () => {
    const plan = renderHandoffPrompt("Do {{payload}}", "a.md");
    const notPlan = renderHandoffPrompt("Do {{payload}}", "a.bin");
    expect(plan).toBe("Do a.md");
    expect(notPlan).toBe("Do a.bin");
  });

  it("does not branch on whether the payload looks like a path", () => {
    expect(renderHandoffPrompt("Do {{payload}}", "/abs/path.md")).toBe("Do /abs/path.md");
    expect(renderHandoffPrompt("Do {{payload}}", "not a path at all")).toBe("Do not a path at all");
  });
});

describe("delivery predicates", () => {
  it("treats queued as not delivered", () => {
    const results = [
      { targetId: "a", label: "A", status: "sent" as const },
      { targetId: "b", label: "B", status: "queued" as const, reason: "not-ready" },
    ];
    expect(allDelivered(results)).toBe(false);
    expect(anyUnsent(results)).toBe(true);
  });

  it("reports delivery only when every target reached a terminal", () => {
    const results = [
      { targetId: "a", label: "A", status: "sent" as const },
      { targetId: "b", label: "B", status: "sent" as const },
    ];
    expect(allDelivered(results)).toBe(true);
    expect(anyUnsent(results)).toBe(false);
  });

  it("does not report delivery for an empty result set", () => {
    expect(allDelivered([])).toBe(false);
  });
});
