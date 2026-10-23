import { describe, expect, it } from "vitest";
import { mergeSuggestionTrailers, parseMessage, serializeMessage } from "./message";

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
