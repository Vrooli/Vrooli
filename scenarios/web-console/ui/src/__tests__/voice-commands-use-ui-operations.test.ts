import { describe, expect, it, vi } from "vitest";
import { VOICE_COMMANDS, type CommandContext } from "../hooks/voice/commands";

describe("voice terminal commands", () => {
  it("routes clipboard and scroll commands to UI operations", () => {
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

    for (const id of ["copy", "paste", "scroll-up", "scroll-down"]) {
      const command = VOICE_COMMANDS.find((candidate) => candidate.id === id);
      if (!command) throw new Error(`missing command ${id}`);
      command.execute(ctx, {});
    }

    expect(ctx.copySelection).toHaveBeenCalledOnce();
    expect(ctx.pasteFromClipboard).toHaveBeenCalledOnce();
    expect(ctx.scrollTerminal).toHaveBeenNthCalledWith(1, -5);
    expect(ctx.scrollTerminal).toHaveBeenNthCalledWith(2, 5);
    expect(ctx.sendToTerminal).not.toHaveBeenCalled();
  });
});
