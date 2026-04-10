import { describe, it, expect } from "vitest";
import { VOICE_COMMANDS, type CommandContext } from "../commands";

function createMockContext(): CommandContext & {
  calls: Record<string, unknown[][]>;
} {
  const calls: Record<string, unknown[][]> = {};
  const track = (name: string) => (...args: unknown[]) => {
    if (!calls[name]) calls[name] = [];
    calls[name].push(args);
  };
  return {
    calls,
    createTab: track("createTab") as () => void,
    switchToTab: track("switchToTab") as (index: number) => void,
    closeTab: track("closeTab") as () => void,
    sendToTerminal: track("sendToTerminal") as (data: string) => void,
    exitVoiceMode: track("exitVoiceMode") as () => void,
  };
}

function getCommand(id: string) {
  const command = VOICE_COMMANDS.find((candidate) => candidate.id === id);
  expect(command).toBeDefined();
  if (!command) throw new Error(`Missing command ${id}`);
  return command;
}

describe("VOICE_COMMANDS", () => {
  it("has unique command IDs", () => {
    const ids = VOICE_COMMANDS.map((c) => c.id);
    expect(new Set(ids).size).toBe(ids.length);
  });

  it("every command has at least one pattern", () => {
    for (const cmd of VOICE_COMMANDS) {
      expect(cmd.patterns.length).toBeGreaterThan(0);
    }
  });

  it("every command has a description", () => {
    for (const cmd of VOICE_COMMANDS) {
      expect(cmd.description).toBeTruthy();
    }
  });

  it("all patterns are lowercase", () => {
    for (const cmd of VOICE_COMMANDS) {
      for (const pattern of cmd.patterns) {
        expect(pattern).toBe(pattern.toLowerCase());
      }
    }
  });
});

describe("Command execution", () => {
  it("new-tab calls createTab", () => {
    const ctx = createMockContext();
    const cmd = getCommand("new-tab");
    cmd.execute(ctx, {});
    expect(ctx.calls.createTab).toHaveLength(1);
  });

  it("switch-tab calls switchToTab with the right number", () => {
    const ctx = createMockContext();
    const cmd = getCommand("switch-tab");
    cmd.execute(ctx, { number: 3 });
    expect(ctx.calls.switchToTab).toHaveLength(1);
    expect(ctx.calls.switchToTab?.[0]).toEqual([3]);
  });

  it("switch-tab defaults to 1 when no number provided", () => {
    const ctx = createMockContext();
    const cmd = getCommand("switch-tab");
    cmd.execute(ctx, {});
    expect(ctx.calls.switchToTab?.[0]).toEqual([1]);
  });

  it("close-tab calls closeTab", () => {
    const ctx = createMockContext();
    const cmd = getCommand("close-tab");
    cmd.execute(ctx, {});
    expect(ctx.calls.closeTab).toHaveLength(1);
  });

  it("send-enter sends carriage return", () => {
    const ctx = createMockContext();
    const cmd = getCommand("send-enter");
    cmd.execute(ctx, {});
    expect(ctx.calls.sendToTerminal?.[0]).toEqual(["\r"]);
  });

  it("cancel sends Ctrl+C", () => {
    const ctx = createMockContext();
    const cmd = getCommand("cancel");
    cmd.execute(ctx, {});
    expect(ctx.calls.sendToTerminal?.[0]).toEqual(["\x03"]);
  });

  it("clear sends Ctrl+L", () => {
    const ctx = createMockContext();
    const cmd = getCommand("clear");
    cmd.execute(ctx, {});
    expect(ctx.calls.sendToTerminal?.[0]).toEqual(["\x0c"]);
  });

  it("tab-key sends Tab character", () => {
    const ctx = createMockContext();
    const cmd = getCommand("tab-key");
    cmd.execute(ctx, {});
    expect(ctx.calls.sendToTerminal?.[0]).toEqual(["\t"]);
  });

  it("stop-listening calls exitVoiceMode", () => {
    const ctx = createMockContext();
    const cmd = getCommand("stop-listening");
    cmd.execute(ctx, {});
    expect(ctx.calls.exitVoiceMode).toHaveLength(1);
  });
});
