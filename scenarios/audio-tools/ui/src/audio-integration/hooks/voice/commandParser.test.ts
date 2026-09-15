import { describe, it, expect } from "vitest";

import { levenshtein, parseCommandDirect } from "./commandParser";

// ---------------------------------------------------------------------------
// levenshtein
// ---------------------------------------------------------------------------

describe("levenshtein", () => {
  it("returns 0 for identical strings", () => {
    expect(levenshtein("", "")).toBe(0);
    expect(levenshtein("abc", "abc")).toBe(0);
    expect(levenshtein("hello world", "hello world")).toBe(0);
  });

  it("returns n when a is empty (pure insertions)", () => {
    expect(levenshtein("", "abc")).toBe(3);
    expect(levenshtein("", "x")).toBe(1);
  });

  it("returns m when b is empty (pure deletions)", () => {
    expect(levenshtein("abc", "")).toBe(3);
    expect(levenshtein("x", "")).toBe(1);
  });

  it("counts a single substitution", () => {
    expect(levenshtein("a", "b")).toBe(1);
    expect(levenshtein("cat", "bat")).toBe(1);
  });

  it("counts a single insertion", () => {
    expect(levenshtein("ab", "abc")).toBe(1);
    expect(levenshtein("tab", "tabs")).toBe(1);
  });

  it("counts a single deletion", () => {
    expect(levenshtein("abc", "ab")).toBe(1);
    expect(levenshtein("stop", "top")).toBe(1);
  });

  it("handles the classic kitten→sitting example (distance=3)", () => {
    expect(levenshtein("kitten", "sitting")).toBe(3);
  });

  it("handles transpositions as two edits", () => {
    // "ab" → "ba" requires 2 edits with standard Levenshtein
    expect(levenshtein("ab", "ba")).toBe(2);
  });

  it("is symmetric", () => {
    const pairs = [
      ["new tab", "add tab"],
      ["stop", "stopping"],
      ["clear", "clean"],
    ];
    for (const [a, b] of pairs) {
      expect(levenshtein(a!, b!)).toBe(levenshtein(b!, a!));
    }
  });

  it("handles multi-word strings", () => {
    expect(levenshtein("new tab", "new tab")).toBe(0);
    expect(levenshtein("new tab", "nwe tab")).toBe(2);
    expect(levenshtein("close tab", "close tab")).toBe(0);
    expect(levenshtein("close tab", "close tob")).toBe(1);
  });
});

// ---------------------------------------------------------------------------
// parseCommandDirect
// ---------------------------------------------------------------------------

describe("parseCommandDirect", () => {
  // ── null / no-match cases ─────────────────────────────────────────────────

  it("returns null for empty string", () => {
    expect(parseCommandDirect("")).toBeNull();
  });

  it("returns null for punctuation-only text that normalizes to empty", () => {
    expect(parseCommandDirect("...")).toBeNull();
    expect(parseCommandDirect("!?!")).toBeNull();
    expect(parseCommandDirect(".,;:")).toBeNull();
  });

  it("returns null for text that matches no command", () => {
    expect(parseCommandDirect("bananas on the moon")).toBeNull();
    expect(parseCommandDirect("zxqwerty abcdef")).toBeNull();
  });

  // ── exact matches ──────────────────────────────────────────────────────────

  it("detects 'new tab' exactly", () => {
    const result = parseCommandDirect("new tab");
    expect(result).not.toBeNull();
    expect(result!.command.id).toBe("new-tab");
    expect(result!.confidence).toBeCloseTo(1.0, 5);
    expect(result!.rawText).toBe("new tab");
  });

  it("detects 'add tab' as an alternate pattern for new-tab", () => {
    const result = parseCommandDirect("add tab");
    expect(result).not.toBeNull();
    expect(result!.command.id).toBe("new-tab");
  });

  it("detects 'close tab' exactly", () => {
    const result = parseCommandDirect("close tab");
    expect(result).not.toBeNull();
    expect(result!.command.id).toBe("close-tab");
    expect(result!.confidence).toBeCloseTo(1.0, 5);
  });

  it("detects 'send' exactly", () => {
    const result = parseCommandDirect("send");
    expect(result).not.toBeNull();
    expect(result!.command.id).toBe("send-enter");
  });

  it("detects 'enter' exactly", () => {
    const result = parseCommandDirect("enter");
    expect(result).not.toBeNull();
    expect(result!.command.id).toBe("send-enter");
  });

  it("detects 'cancel' exactly", () => {
    const result = parseCommandDirect("cancel");
    expect(result).not.toBeNull();
    expect(result!.command.id).toBe("cancel");
  });

  it("detects 'copy' exactly", () => {
    const result = parseCommandDirect("copy");
    expect(result).not.toBeNull();
    expect(result!.command.id).toBe("copy");
  });

  it("detects 'paste' exactly", () => {
    const result = parseCommandDirect("paste");
    expect(result).not.toBeNull();
    expect(result!.command.id).toBe("paste");
  });

  it("detects 'clear' exactly", () => {
    const result = parseCommandDirect("clear");
    expect(result).not.toBeNull();
    expect(result!.command.id).toBe("clear");
  });

  it("detects 'clear screen' as clear command", () => {
    const result = parseCommandDirect("clear screen");
    expect(result).not.toBeNull();
    expect(result!.command.id).toBe("clear");
  });

  it("detects 'scroll up' exactly", () => {
    const result = parseCommandDirect("scroll up");
    expect(result).not.toBeNull();
    expect(result!.command.id).toBe("scroll-up");
  });

  it("detects 'scroll down' exactly", () => {
    const result = parseCommandDirect("scroll down");
    expect(result).not.toBeNull();
    expect(result!.command.id).toBe("scroll-down");
  });

  it("detects 'stop listening' exactly and prefers it over 'stop'", () => {
    const result = parseCommandDirect("stop listening");
    expect(result).not.toBeNull();
    expect(result!.command.id).toBe("stop-listening");
  });

  it("detects 'mic off' as stop-listening", () => {
    const result = parseCommandDirect("mic off");
    expect(result).not.toBeNull();
    expect(result!.command.id).toBe("stop-listening");
  });

  it("detects 'tab key' as tab-key command", () => {
    const result = parseCommandDirect("tab key");
    expect(result).not.toBeNull();
    expect(result!.command.id).toBe("tab-key");
  });

  it("detects 'autocomplete' as tab-key command", () => {
    const result = parseCommandDirect("autocomplete");
    expect(result).not.toBeNull();
    expect(result!.command.id).toBe("tab-key");
  });

  // ── case insensitivity ────────────────────────────────────────────────────

  it("is case-insensitive", () => {
    const result = parseCommandDirect("NEW TAB");
    expect(result).not.toBeNull();
    expect(result!.command.id).toBe("new-tab");
  });

  it("strips trailing punctuation before matching", () => {
    const result = parseCommandDirect("new tab.");
    expect(result).not.toBeNull();
    expect(result!.command.id).toBe("new-tab");
  });

  // ── fuzzy matching ─────────────────────────────────────────────────────────

  it("fuzzy-matches a 1-char typo in a short command ('cancle' → cancel)", () => {
    const result = parseCommandDirect("cancle");
    expect(result).not.toBeNull();
    expect(result!.command.id).toBe("cancel");
    expect(result!.confidence).toBeGreaterThan(0.5);
  });

  it("fuzzy-matches a 1-char typo in a long pattern ('nwe tab' → new-tab)", () => {
    const result = parseCommandDirect("nwe tab");
    // levenshtein("nwe tab", "new tab") = 2 which exceeds maxDist(7)=2, but
    // levenshtein("nwe tab", "add tab") may not match either. Let's check if any pattern matches.
    // Actually "nwe tab" to "new tab": distance = 2 (swap n/e). maxEditDistance("new tab") = 2 → match!
    expect(result).not.toBeNull();
    expect(result!.command.id).toBe("new-tab");
  });

  // ── numeric argument extraction ───────────────────────────────────────────

  it("extracts a digit argument from switch-tab commands", () => {
    const result = parseCommandDirect("tab 3");
    expect(result).not.toBeNull();
    expect(result!.command.id).toBe("switch-tab");
    expect(result!.args.number).toBe(3);
  });

  it("extracts a word-number argument ('tab three')", () => {
    const result = parseCommandDirect("tab three");
    expect(result).not.toBeNull();
    expect(result!.command.id).toBe("switch-tab");
    expect(result!.args.number).toBe(3);
  });

  it("extracts ordinal word numbers ('go to tab second')", () => {
    const result = parseCommandDirect("go to tab second");
    expect(result).not.toBeNull();
    expect(result!.command.id).toBe("switch-tab");
    expect(result!.args.number).toBe(2);
  });

  it("does not set args.number when no number is present in trailing text", () => {
    const result = parseCommandDirect("tab");
    expect(result).not.toBeNull();
    expect(result!.args.number).toBeUndefined();
  });

  it("does not set args.number for digits > 100", () => {
    const result = parseCommandDirect("tab 200");
    // trailing text "200" → digit 200 > 100 → not returned; word search also fails
    if (result) {
      expect(result.args.number).toBeUndefined();
    }
  });

  // ── confidence field ──────────────────────────────────────────────────────

  it("returns confidence = 1.0 for exact match", () => {
    const result = parseCommandDirect("copy");
    expect(result).not.toBeNull();
    expect(result!.confidence).toBe(1.0);
  });

  it("returns confidence < 1.0 for a fuzzy match", () => {
    const result = parseCommandDirect("cancle"); // 1 edit from "cancel"
    expect(result).not.toBeNull();
    expect(result!.confidence).toBeLessThan(1.0);
    expect(result!.confidence).toBeGreaterThan(0.5);
  });

  // ── rawText preservation ──────────────────────────────────────────────────

  it("preserves the original rawText (before normalization)", () => {
    const input = "NEW TAB.";
    const result = parseCommandDirect(input);
    expect(result).not.toBeNull();
    expect(result!.rawText).toBe(input);
  });
});
