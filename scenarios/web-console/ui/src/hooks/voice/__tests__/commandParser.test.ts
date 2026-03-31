import { describe, it, expect } from "vitest";
import { parseCommandDirect, levenshtein } from "../commandParser";

// --- levenshtein ---

describe("levenshtein", () => {
  it("returns 0 for identical strings", () => {
    expect(levenshtein("hello", "hello")).toBe(0);
  });

  it("returns length for empty vs non-empty", () => {
    expect(levenshtein("", "abc")).toBe(3);
    expect(levenshtein("abc", "")).toBe(3);
  });

  it("returns 0 for two empty strings", () => {
    expect(levenshtein("", "")).toBe(0);
  });

  it("handles single character substitution", () => {
    expect(levenshtein("cat", "bat")).toBe(1);
  });

  it("handles insertion", () => {
    expect(levenshtein("tab", "tabs")).toBe(1);
  });

  it("handles deletion", () => {
    expect(levenshtein("tabs", "tab")).toBe(1);
  });

  it("handles multiple edits", () => {
    expect(levenshtein("kitten", "sitting")).toBe(3);
  });

  it("is symmetric", () => {
    expect(levenshtein("abc", "xyz")).toBe(levenshtein("xyz", "abc"));
  });
});

// --- parseCommandDirect ---

describe("parseCommandDirect", () => {
  it("returns null for empty text", () => {
    expect(parseCommandDirect("")).toBeNull();
  });

  it("returns null for whitespace-only text", () => {
    expect(parseCommandDirect("   ")).toBeNull();
  });

  // ── Exact matches ──

  it("detects 'new tab' command with exact match", () => {
    const result = parseCommandDirect("new tab");
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("new-tab");
    expect(result?.confidence).toBe(1);
  });

  it("detects 'close tab' command", () => {
    const result = parseCommandDirect("close tab");
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("close-tab");
  });

  it("detects 'enter' command", () => {
    const result = parseCommandDirect("enter");
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("send-enter");
  });

  it("detects 'cancel' command", () => {
    const result = parseCommandDirect("cancel");
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("cancel");
  });

  it("detects 'clear' command", () => {
    const result = parseCommandDirect("clear");
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("clear");
  });

  it("detects 'scroll up' command", () => {
    const result = parseCommandDirect("scroll up");
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("scroll-up");
  });

  it("detects 'stop listening' command", () => {
    const result = parseCommandDirect("stop listening");
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("stop-listening");
  });

  // ── Case insensitivity ──

  it("is case insensitive", () => {
    const result = parseCommandDirect("New Tab");
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("new-tab");
  });

  // ── Fuzzy matching ──

  it("matches Whisper misrecognition 'knew tab' as 'new tab'", () => {
    const result = parseCommandDirect("knew tab");
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("new-tab");
    expect(result?.confidence ?? 0).toBeGreaterThan(0.5);
    expect(result?.confidence ?? 1).toBeLessThan(1);
  });

  it("matches 'cleer' as 'clear' (1 edit)", () => {
    const result = parseCommandDirect("cleer");
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("clear");
  });

  it("rejects gibberish with too many edits", () => {
    const result = parseCommandDirect("xyzzy");
    expect(result).toBeNull();
  });

  it("rejects normal speech that is not a command", () => {
    expect(parseCommandDirect("please open the settings page")).toBeNull();
    expect(parseCommandDirect("what is the weather today")).toBeNull();
  });

  // ── Number extraction ──

  it("extracts numeric digit argument from 'tab 3'", () => {
    const result = parseCommandDirect("tab 3");
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("switch-tab");
    expect(result?.args.number).toBe(3);
  });

  it("extracts number word argument from 'tab three'", () => {
    const result = parseCommandDirect("tab three");
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("switch-tab");
    expect(result?.args.number).toBe(3);
  });

  it("extracts ordinal argument from 'switch tab second'", () => {
    const result = parseCommandDirect("switch tab second");
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("switch-tab");
    expect(result?.args.number).toBe(2);
  });

  // ── Alternative patterns ──

  it("detects 'add tab' as new-tab", () => {
    const result = parseCommandDirect("add tab");
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("new-tab");
  });

  it("detects 'open tab' as new-tab", () => {
    const result = parseCommandDirect("open tab");
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("new-tab");
  });

  it("detects 'interrupt' as cancel", () => {
    const result = parseCommandDirect("interrupt");
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("cancel");
  });

  it("detects 'mic off' as stop-listening", () => {
    const result = parseCommandDirect("mic off");
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("stop-listening");
  });

  // ── Whisper punctuation handling ──

  it("detects command when Whisper inserts period: 'new tab.'", () => {
    const result = parseCommandDirect("new tab.");
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("new-tab");
  });

  it("detects command with trailing punctuation: 'clear screen!'", () => {
    const result = parseCommandDirect("clear screen!");
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("clear");
  });

  it("extracts tab number with Whisper punctuation: 'tab 3.'", () => {
    const result = parseCommandDirect("tab 3.");
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("switch-tab");
    expect(result?.args.number).toBe(3);
  });

  // ── Longer patterns shadow shorter ones ──

  it("prefers 'stop listening' over 'stop' when full match", () => {
    const result = parseCommandDirect("stop listening");
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("stop-listening");
  });

  it("prefers 'clear screen' over 'clear' when full match", () => {
    const result = parseCommandDirect("clear screen");
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("clear");
  });
});
