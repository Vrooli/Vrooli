import { describe, expect, it, vi } from "vitest";
import {
  getRecentTerminalText,
  terminalContainsCandidate,
  waitForTerminalCandidateMatch,
} from "./terminalTtsMatch";

function terminalWith(lines: string[]) {
  return {
    buffer: {
      active: {
        length: lines.length,
        getLine: (index: number) => (lines[index] ? { translateToString: () => lines[index] } : undefined),
      },
    },
  } as never;
}

describe("terminal TTS matching", () => {
  it("reads recent lines and normalizes presentation markup", () => {
    const terminal = terminalWith(["old", "**Build** complete", ""]);
    expect(getRecentTerminalText(terminal)).toBe("old\n**Build** complete");
    expect(terminalContainsCandidate(terminal, "build complete")).toBe(true);
    expect(terminalContainsCandidate(terminal, "missing output")).toBe(false);
    expect(terminalContainsCandidate(terminal, "")).toBe(false);
  });

  it("limits the candidate prefix and waits for delayed output", async () => {
    vi.useFakeTimers();
    const lines = ["waiting"];
    const terminal = terminalWith(lines);
    const result = waitForTerminalCandidateMatch(terminal, "ready", { intervalMs: 10, timeoutMs: 100 });
    lines[0] = "ready";
    await vi.advanceTimersByTimeAsync(20);
    await expect(result).resolves.toBe(true);
    vi.useRealTimers();
  });
});
