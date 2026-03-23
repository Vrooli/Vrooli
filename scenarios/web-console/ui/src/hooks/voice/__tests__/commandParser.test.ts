import { describe, it, expect } from "vitest";
import { parseCommand, levenshtein, partialContainsPrefix } from "../commandParser";

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

// --- parseCommand ---

describe("parseCommand", () => {
  const PREFIX = "hey do";

  it("returns null for empty text", () => {
    expect(parseCommand("", PREFIX)).toBeNull();
  });

  it("returns null for empty prefix", () => {
    expect(parseCommand("hey do new tab", "")).toBeNull();
  });

  it("returns null when text does not start with prefix", () => {
    expect(parseCommand("please new tab", PREFIX)).toBeNull();
  });

  it("returns null when only the prefix is spoken", () => {
    expect(parseCommand("hey do", PREFIX)).toBeNull();
  });

  // ── Exact matches ──

  it("detects 'new tab' command with exact match", () => {
    const result = parseCommand("hey do new tab", PREFIX);
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("new-tab");
    expect(result?.confidence).toBe(1);
  });

  it("detects 'close tab' command", () => {
    const result = parseCommand("hey do close tab", PREFIX);
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("close-tab");
  });

  it("detects 'enter' command", () => {
    const result = parseCommand("hey do enter", PREFIX);
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("send-enter");
  });

  it("detects 'cancel' command", () => {
    const result = parseCommand("hey do cancel", PREFIX);
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("cancel");
  });

  it("detects 'clear' command", () => {
    const result = parseCommand("hey do clear", PREFIX);
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("clear");
  });

  it("detects 'scroll up' command", () => {
    const result = parseCommand("hey do scroll up", PREFIX);
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("scroll-up");
  });

  it("detects 'stop listening' command", () => {
    const result = parseCommand("hey do stop listening", PREFIX);
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("stop-listening");
  });

  // ── Case insensitivity ──

  it("is case insensitive for prefix", () => {
    const result = parseCommand("Hey Do new tab", PREFIX);
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("new-tab");
  });

  it("is case insensitive for command", () => {
    const result = parseCommand("hey do New Tab", PREFIX);
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("new-tab");
  });

  // ── Fuzzy matching ──

  it("matches Whisper misrecognition 'knew tab' as 'new tab'", () => {
    const result = parseCommand("hey do knew tab", PREFIX);
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("new-tab");
    expect(result?.confidence ?? 0).toBeGreaterThan(0.5);
    expect(result?.confidence ?? 1).toBeLessThan(1);
  });

  it("matches 'cleer' as 'clear' (1 edit)", () => {
    const result = parseCommand("hey do cleer", PREFIX);
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("clear");
  });

  it("rejects gibberish with too many edits", () => {
    const result = parseCommand("hey do xyzzy", PREFIX);
    expect(result).toBeNull();
  });

  // ── Number extraction ──

  it("extracts numeric digit argument from 'tab 3'", () => {
    const result = parseCommand("hey do tab 3", PREFIX);
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("switch-tab");
    expect(result?.args.number).toBe(3);
  });

  it("extracts number word argument from 'tab three'", () => {
    const result = parseCommand("hey do tab three", PREFIX);
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("switch-tab");
    expect(result?.args.number).toBe(3);
  });

  it("extracts ordinal argument from 'switch tab second'", () => {
    const result = parseCommand("hey do switch tab second", PREFIX);
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("switch-tab");
    expect(result?.args.number).toBe(2);
  });

  // ── Alternative patterns ──

  it("detects 'add tab' as new-tab", () => {
    const result = parseCommand("hey do add tab", PREFIX);
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("new-tab");
  });

  it("detects 'open tab' as new-tab", () => {
    const result = parseCommand("hey do open tab", PREFIX);
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("new-tab");
  });

  it("detects 'interrupt' as cancel", () => {
    const result = parseCommand("hey do interrupt", PREFIX);
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("cancel");
  });

  it("detects 'mic off' as stop-listening", () => {
    const result = parseCommand("hey do mic off", PREFIX);
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("stop-listening");
  });

  // ── Different prefix ──

  it("works with custom prefix", () => {
    const result = parseCommand("run clear", "run");
    expect(result).not.toBeNull();
    expect(result?.command.id).toBe("clear");
  });
});

// --- partialContainsPrefix ---

describe("partialContainsPrefix", () => {
  it("returns true when partial contains prefix", () => {
    expect(partialContainsPrefix("I said hey do something", "hey do")).toBe(true);
  });

  it("is case insensitive", () => {
    expect(partialContainsPrefix("Hey Do something", "hey do")).toBe(true);
  });

  it("returns false when prefix not present", () => {
    expect(partialContainsPrefix("hello world", "hey do")).toBe(false);
  });

  it("returns false for empty partial", () => {
    expect(partialContainsPrefix("", "hey do")).toBe(false);
  });

  it("returns false for empty prefix", () => {
    expect(partialContainsPrefix("hey do something", "")).toBe(false);
  });
});
