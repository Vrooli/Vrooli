import { describe, it, expect, beforeEach } from "vitest";
import { LocalEchoController } from "../lib/localEcho";

describe("LocalEchoController", () => {
  let echo: LocalEchoController;

  beforeEach(() => {
    echo = new LocalEchoController();
  });

  describe("handleInput", () => {
    it("echoes a single printable character", () => {
      expect(echo.handleInput("a")).toBe("a");
      expect(echo.pendingCount).toBe(1);
    });

    it("echoes space", () => {
      expect(echo.handleInput(" ")).toBe(" ");
      expect(echo.pendingCount).toBe(1);
    });

    it("echoes tilde (highest printable ASCII)", () => {
      expect(echo.handleInput("~")).toBe("~");
    });

    it("returns null for control characters", () => {
      expect(echo.handleInput("\x03")).toBeNull(); // Ctrl-C
      expect(echo.handleInput("\t")).toBeNull();   // Tab
      expect(echo.handleInput("\r")).toBeNull();   // Enter
      expect(echo.handleInput("\n")).toBeNull();   // Newline
      expect(echo.pendingCount).toBe(0);
    });

    it("returns null for escape sequences", () => {
      expect(echo.handleInput("\x1b")).toBeNull();
      expect(echo.pendingCount).toBe(0);
    });

    it("returns null for DEL (0x7f)", () => {
      expect(echo.handleInput("\x7f")).toBeNull();
      expect(echo.pendingCount).toBe(0);
    });

    it("returns null for multi-char paste", () => {
      expect(echo.handleInput("hello")).toBeNull();
      expect(echo.pendingCount).toBe(0);
    });

    it("returns null when disabled", () => {
      echo.enabled = false;
      expect(echo.handleInput("a")).toBeNull();
      expect(echo.pendingCount).toBe(0);
    });

    it("returns null for multi-byte unicode (surrogate pairs)", () => {
      // Emoji is length > 1 due to surrogate pairs
      expect(echo.handleInput("😀")).toBeNull();
      expect(echo.pendingCount).toBe(0);
    });
  });

  describe("processOutput", () => {
    it("passes through data when no predictions pending", () => {
      expect(echo.processOutput("hello")).toBe("hello");
    });

    it("suppresses matching server echo", () => {
      echo.handleInput("a");
      expect(echo.processOutput("a")).toBe("");
      expect(echo.pendingCount).toBe(0);
    });

    it("suppresses matching prefix and passes through extra output", () => {
      echo.handleInput("l");
      echo.handleInput("s");
      // Server echoes "ls" then sends "\r\nfile1\r\n"
      expect(echo.processOutput("ls\r\nfile1\r\n")).toBe("\r\nfile1\r\n");
      expect(echo.pendingCount).toBe(0);
    });

    it("erases predictions on mismatch and returns full server data", () => {
      echo.handleInput("a");
      echo.handleInput("b");
      // Server sends something completely different (e.g., tab completion)
      const result = echo.processOutput("xyz");
      // Should erase 2 predictions (\b \b\b \b) then show server data
      expect(result).toBe("\b \b\b \bxyz");
      expect(echo.pendingCount).toBe(0);
    });

    it("handles partial match then mismatch", () => {
      echo.handleInput("a");
      echo.handleInput("b");
      echo.handleInput("c");
      // Server echoes "a" then diverges with "XY"
      const result = echo.processOutput("aXY");
      // "a" matches and is consumed, then "X" mismatches "b" — erase "b" and "c"
      expect(result).toBe("\b \b\b \bXY");
      expect(echo.pendingCount).toBe(0);
    });

    it("handles server echoing only some predicted chars", () => {
      echo.handleInput("a");
      echo.handleInput("b");
      echo.handleInput("c");
      // Server only echoes "a"
      expect(echo.processOutput("a")).toBe("");
      expect(echo.pendingCount).toBe(2); // "b" and "c" still pending
    });
  });

  describe("reset", () => {
    it("clears all pending predictions", () => {
      echo.handleInput("a");
      echo.handleInput("b");
      echo.reset();
      expect(echo.pendingCount).toBe(0);
    });
  });

  describe("enabled", () => {
    it("clears predictions when disabled", () => {
      echo.handleInput("a");
      echo.handleInput("b");
      echo.enabled = false;
      expect(echo.pendingCount).toBe(0);
    });

    it("defaults to enabled", () => {
      expect(echo.enabled).toBe(true);
    });

    it("can be re-enabled", () => {
      echo.enabled = false;
      echo.enabled = true;
      expect(echo.handleInput("a")).toBe("a");
    });
  });
});
