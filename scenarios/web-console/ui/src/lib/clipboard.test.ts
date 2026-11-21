import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { isClipboardSupported, readText, writeText } from "./clipboard";

const navigatorClipboard = Object.getOwnPropertyDescriptor(navigator, "clipboard");
const documentExecCommand = Object.getOwnPropertyDescriptor(document, "execCommand");

function setClipboard(value: Clipboard | undefined) {
  Object.defineProperty(navigator, "clipboard", {
    configurable: true,
    value,
  });
}

function setExecCommand(value: (command: string) => boolean) {
  Object.defineProperty(document, "execCommand", {
    configurable: true,
    value,
  });
}

afterEach(() => {
  if (navigatorClipboard) Object.defineProperty(navigator, "clipboard", navigatorClipboard);
  else delete (navigator as { clipboard?: Clipboard }).clipboard;
  if (documentExecCommand) Object.defineProperty(document, "execCommand", documentExecCommand);
  else delete (document as { execCommand?: typeof document.execCommand }).execCommand;
  vi.restoreAllMocks();
});

describe("clipboard capability boundary", () => {
  beforeEach(() => {
    setClipboard({
      writeText: vi.fn().mockResolvedValue(undefined),
      readText: vi.fn().mockResolvedValue("copied text"),
    } as unknown as Clipboard);
  });

  it("uses the async clipboard API and returns typed results", async () => {
    await expect(writeText("hello")).resolves.toEqual({ ok: true });
    await expect(readText()).resolves.toEqual({ ok: true, text: "copied text" });
    expect(isClipboardSupported()).toBe(true);
  });

  it("falls back to execCommand when the async API is absent", async () => {
    setClipboard(undefined);
    const execCommand = vi.fn().mockReturnValue(true);
    setExecCommand(execCommand);

    await expect(writeText("legacy copy")).resolves.toEqual({ ok: true });
    expect(execCommand).toHaveBeenCalledWith("copy");
    expect(isClipboardSupported()).toBe(true);
  });

  it("reports denied reads without throwing", async () => {
    setClipboard({ readText: vi.fn().mockRejectedValue(new Error("permission denied")) } as unknown as Clipboard);

    await expect(readText()).resolves.toEqual({ ok: false, reason: "denied" });
  });
});
