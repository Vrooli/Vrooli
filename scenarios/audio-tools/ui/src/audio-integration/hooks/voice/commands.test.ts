import { describe, it, expect, vi } from "vitest";

import { VOICE_COMMANDS } from "./commands";
import type { CommandContext } from "./commands";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeContext(): CommandContext {
  return {
    createTab: vi.fn(),
    switchToTab: vi.fn(),
    closeTab: vi.fn(),
    sendToTerminal: vi.fn(),
    exitVoiceMode: vi.fn(),
  };
}

function findCommand(id: string) {
  const cmd = VOICE_COMMANDS.find(c => c.id === id);
  if (!cmd) throw new Error(`Command not found: ${id}`);
  return cmd;
}

// ---------------------------------------------------------------------------
// VOICE_COMMANDS structure
// ---------------------------------------------------------------------------

describe("VOICE_COMMANDS", () => {
  it("exports a non-empty array", () => {
    expect(Array.isArray(VOICE_COMMANDS)).toBe(true);
    expect(VOICE_COMMANDS.length).toBeGreaterThan(0);
  });

  it("every command has id, description, patterns (non-empty), and execute", () => {
    for (const cmd of VOICE_COMMANDS) {
      expect(typeof cmd.id).toBe("string");
      expect(cmd.id.length).toBeGreaterThan(0);
      expect(typeof cmd.description).toBe("string");
      expect(Array.isArray(cmd.patterns)).toBe(true);
      expect(cmd.patterns.length).toBeGreaterThan(0);
      expect(typeof cmd.execute).toBe("function");
    }
  });

  it("all command ids are unique", () => {
    const ids = VOICE_COMMANDS.map(c => c.id);
    expect(new Set(ids).size).toBe(ids.length);
  });
});

// ---------------------------------------------------------------------------
// Individual command execute functions
// ---------------------------------------------------------------------------

describe("new-tab command", () => {
  it("calls ctx.createTab()", () => {
    const ctx = makeContext();
    findCommand("new-tab").execute(ctx, {});
    expect(ctx.createTab).toHaveBeenCalledTimes(1);
  });
});

describe("switch-tab command", () => {
  it("calls ctx.switchToTab with provided number arg", () => {
    const ctx = makeContext();
    findCommand("switch-tab").execute(ctx, { number: 3 });
    expect(ctx.switchToTab).toHaveBeenCalledWith(3);
  });

  it("defaults to tab 1 when no number arg is provided", () => {
    const ctx = makeContext();
    findCommand("switch-tab").execute(ctx, {});
    expect(ctx.switchToTab).toHaveBeenCalledWith(1);
  });

  it("defaults to tab 1 when args.number is not a number type", () => {
    const ctx = makeContext();
    findCommand("switch-tab").execute(ctx, { number: "three" });
    expect(ctx.switchToTab).toHaveBeenCalledWith(1);
  });
});

describe("close-tab command", () => {
  it("calls ctx.closeTab()", () => {
    const ctx = makeContext();
    findCommand("close-tab").execute(ctx, {});
    expect(ctx.closeTab).toHaveBeenCalledTimes(1);
  });
});

describe("send-enter command", () => {
  it("sends carriage return (\\r) to the terminal", () => {
    const ctx = makeContext();
    findCommand("send-enter").execute(ctx, {});
    expect(ctx.sendToTerminal).toHaveBeenCalledWith("\r");
  });
});

describe("cancel command", () => {
  it("sends Ctrl+C (\\x03) to the terminal", () => {
    const ctx = makeContext();
    findCommand("cancel").execute(ctx, {});
    expect(ctx.sendToTerminal).toHaveBeenCalledWith("\x03");
  });
});

describe("copy command", () => {
  it("sends the Ctrl+Shift+C CSI sequence to the terminal", () => {
    const ctx = makeContext();
    findCommand("copy").execute(ctx, {});
    expect(ctx.sendToTerminal).toHaveBeenCalledWith("\x1b[67;5u");
  });
});

describe("paste command", () => {
  it("sends the Ctrl+Shift+V CSI sequence to the terminal", () => {
    const ctx = makeContext();
    findCommand("paste").execute(ctx, {});
    expect(ctx.sendToTerminal).toHaveBeenCalledWith("\x1b[86;5u");
  });
});

describe("clear command", () => {
  it("sends Ctrl+L (\\x0c) to the terminal", () => {
    const ctx = makeContext();
    findCommand("clear").execute(ctx, {});
    expect(ctx.sendToTerminal).toHaveBeenCalledWith("\x0c");
  });
});

describe("tab-key command", () => {
  it("sends a tab character (\\t) to the terminal", () => {
    const ctx = makeContext();
    findCommand("tab-key").execute(ctx, {});
    expect(ctx.sendToTerminal).toHaveBeenCalledWith("\t");
  });
});

describe("scroll-up command", () => {
  it("sends Shift+PageUp CSI sequence", () => {
    const ctx = makeContext();
    findCommand("scroll-up").execute(ctx, {});
    expect(ctx.sendToTerminal).toHaveBeenCalledWith("\x1b[5~");
  });
});

describe("scroll-down command", () => {
  it("sends Shift+PageDown CSI sequence", () => {
    const ctx = makeContext();
    findCommand("scroll-down").execute(ctx, {});
    expect(ctx.sendToTerminal).toHaveBeenCalledWith("\x1b[6~");
  });
});

describe("stop-listening command", () => {
  it("calls ctx.exitVoiceMode()", () => {
    const ctx = makeContext();
    findCommand("stop-listening").execute(ctx, {});
    expect(ctx.exitVoiceMode).toHaveBeenCalledTimes(1);
  });
});
