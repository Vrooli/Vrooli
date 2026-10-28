import { describe, expect, it } from "vitest";
import { mergeContinuationMessage, mergeSuggestionTrailers, parseMessage, serializeMessage } from "./message";

describe("Git message fields", () => {
  it("round-trips description paragraphs and unknown folded trailers in order", () => {
    const text = "fix: retain staged sources\n\nExplain why this changed.\n\nMore detail.\n\nX-Unknown: keep exactly\n  including this continuation\nSigned-off-by: Operator <operator@example.test>";
    expect(parseMessage(text)).toEqual({ title: "fix: retain staged sources", description: "Explain why this changed.\n\nMore detail.", trailers: "X-Unknown: keep exactly\n  including this continuation\nSigned-off-by: Operator <operator@example.test>" });
    expect(serializeMessage(parseMessage(text))).toBe(text);
  });
  it("keeps colon-like body text outside the final trailer paragraph", () => {
    const text = "fix: retain source\n\nReason: inspect this carefully\nNon-trailer prose\n\nA final ordinary paragraph.";
    expect(parseMessage(text).trailers).toBe("");
    expect(serializeMessage(parseMessage(text))).toBe(text);
  });
  it("keeps title paragraph lines as description and accepts Git mixed trailer blocks", () => {
    const fields = parseMessage("fix: title\nextra title-paragraph line\n\nSigned-off-by: A\nexplanation\nmore text\nlast line");
    expect(fields.description).toBe("extra title-paragraph line");
    expect(fields.trailers).toBe("Signed-off-by: A\nexplanation\nmore text\nlast line");
  });
  it("serializes a title and trailers with one empty paragraph", () => {
    expect(serializeMessage({ title: "fix: title", description: "", trailers: "X-Unknown: value" })).toBe("fix: title\n\nX-Unknown: value");
    expect(parseMessage("fix: title\nX-Unknown: body text").trailers).toBe("");
    expect(serializeMessage({ title: "", description: "Description without a title", trailers: "" })).toBe("\n\nDescription without a title");
  });
  it("retains unknown trailers while merging new references and respects value case", () => {
    expect(mergeSuggestionTrailers("X-Unknown: keep\nVrooli-Plan: ABC", [
      { key: "Vrooli-Plan", value: "ABC" }, { key: "Vrooli-Plan", value: "abc" },
    ])).toBe("X-Unknown: keep\nVrooli-Plan: ABC\nVrooli-Plan: abc");
    expect(mergeSuggestionTrailers("X-Unknown: first\n  rest", [{ key: "X-Unknown", value: "first rest" }, { key: "X-Unknown", value: "first" }])).toBe("X-Unknown: first\n  rest\nX-Unknown: first");
  });
});


describe("history continuation messages", () => {
  const continuation = "Related work p2\n\nVrooli-Effort: effort:work\nVrooli-Continues: abc123";
  it("adds continuation references without replacing an existing title or description", () => {
    const existing = "fix: AI drafted summary\n\nKeep the AI description.\n\nX-Custom: preserve\n  folded detail";
    expect(mergeContinuationMessage(existing, continuation)).toBe(existing + "\nVrooli-Effort: effort:work\nVrooli-Continues: abc123");
  });
  it.each(["", " \n\t"])("prefills an empty editor (%j)", existing => {
    expect(mergeContinuationMessage(existing, continuation)).toBe(continuation);
  });
  it("preserves body indentation, line endings and existing trailing whitespace", () => {
    const existing = "Human summary\r\n\r\n    indented code example  \r\n";
    expect(mergeContinuationMessage(existing, "Next p2\n\nVrooli-Continues: abc123")).toBe(existing + "\r\n\r\nVrooli-Continues: abc123");
    const linked = "Human summary\n\n    indented code example\n\nVrooli-Continues: abc123\n\n";
    expect(mergeContinuationMessage(linked, "Next p2\n\nVrooli-Continues: abc123")).toBe(linked);
    expect(mergeContinuationMessage(linked, "Other p2\n\nVrooli-Continues: def456")).toBe(linked.trimEnd() + "\nVrooli-Continues: def456\n\n");
  });
  it("keeps repeated selections idempotent and retains distinct references", () => {
    const once = mergeContinuationMessage("Human summary", continuation);
    expect(mergeContinuationMessage(once, continuation)).toBe(once);
    expect(mergeContinuationMessage(once, "Other work p3\n\nVrooli-Continues: def456")).toBe(once + "\nVrooli-Continues: def456");
  });
  it("preserves an existing description even if its title is still empty", () => {
    expect(mergeContinuationMessage("\n\nDescription being written", continuation)).toBe("\n\nDescription being written\n\nVrooli-Effort: effort:work\nVrooli-Continues: abc123");
  });
});
