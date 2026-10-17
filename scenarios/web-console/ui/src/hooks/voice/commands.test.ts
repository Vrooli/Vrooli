import { describe, expect, it, vi } from "vitest";
import { VOICE_COMMANDS, type CommandContext } from "./commands";

describe("persistent voice command vocabulary", () => {
  it("executes every command through its injected action handle", () => {
    const ctx: CommandContext = {
      createTab: vi.fn(),
      switchToTab: vi.fn(),
      closeTab: vi.fn(),
      sendToTerminal: vi.fn(),
      copySelection: vi.fn(),
      pasteFromClipboard: vi.fn(),
      scrollTerminal: vi.fn(),
      exitVoiceMode: vi.fn(),
    };
    for (const command of VOICE_COMMANDS) {
      command.execute(ctx, command.id === "switch-tab" ? { number: 3 } : {});
    }
    expect(ctx.createTab).toHaveBeenCalledOnce();
    expect(ctx.switchToTab).toHaveBeenCalledWith(3);
    expect(ctx.closeTab).toHaveBeenCalledOnce();
    expect(ctx.sendToTerminal).toHaveBeenCalledTimes(4);
    expect(ctx.copySelection).toHaveBeenCalledOnce();
    expect(ctx.pasteFromClipboard).toHaveBeenCalledOnce();
    expect(ctx.scrollTerminal).toHaveBeenCalledWith(-5);
    expect(ctx.scrollTerminal).toHaveBeenCalledWith(5);
    expect(ctx.exitVoiceMode).toHaveBeenCalledOnce();
  });

  it("defaults an invalid tab argument to the first tab", () => {
    const switchToTab = vi.fn();
    const ctx = { switchToTab } as unknown as CommandContext;
    VOICE_COMMANDS.find((command) => command.id === "switch-tab")?.execute(ctx, { number: "3" });
    expect(switchToTab).toHaveBeenCalledWith(1);
  });
});
