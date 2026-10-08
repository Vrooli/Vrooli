import { describe, expect, it } from "vitest";
import { levenshtein, parseCommandDirect } from "./commandParser";

describe("voice command parser", () => {
  it("computes edit distance including empty strings", () => {
    expect(levenshtein("", "abc")).toBe(3);
    expect(levenshtein("abc", "")).toBe(3);
    expect(levenshtein("kitten", "sitting")).toBe(3);
    expect(levenshtein("tab", "tab")).toBe(0);
  });

  it("normalizes punctuation and extracts digit and word arguments", () => {
    expect(parseCommandDirect("NEW TAB!")?.command.id).toBe("new-tab");
    expect(parseCommandDirect("tab 3")?.args.number).toBe(3);
    expect(parseCommandDirect("go to tab three")?.args.number).toBe(3);
    expect(parseCommandDirect("tab 0")?.args.number).toBeUndefined();
    expect(parseCommandDirect("tab 101")?.args.number).toBeUndefined();
  });

  it("prefers the longer command and rejects empty or low-confidence text", () => {
    expect(parseCommandDirect("stop listening")?.command.id).toBe("stop-listening");
    expect(parseCommandDirect("")).toBeNull();
    expect(parseCommandDirect("!!!")).toBeNull();
    expect(parseCommandDirect("zzzzzzzzzzzzzz")).toBeNull();
  });
});
