import { afterEach, describe, expect, it, vi } from "vitest";
import { writeClipboardText } from "../../../library/components/CopyIconButton/versions/1.0.2/CopyIconButton";

const originalClipboard = Object.getOwnPropertyDescriptor(navigator, "clipboard");
const originalExecCommand = Object.getOwnPropertyDescriptor(document, "execCommand");

afterEach(() => {
  vi.restoreAllMocks();
  if (originalClipboard) Object.defineProperty(navigator, "clipboard", originalClipboard);
  else Reflect.deleteProperty(navigator, "clipboard");
  if (originalExecCommand) Object.defineProperty(document, "execCommand", originalExecCommand);
  else Reflect.deleteProperty(document, "execCommand");
});

describe("CopyIconButton draft clipboard compatibility", () => {
  it("uses the asynchronous clipboard when available", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });
    const execCommand = vi.fn(() => true);
    Object.defineProperty(document, "execCommand", {
      configurable: true,
      value: execCommand,
    });

    await expect(writeClipboardText("copy me")).resolves.toBe(true);
    expect(writeText).toHaveBeenCalledWith("copy me");
    expect(execCommand).not.toHaveBeenCalled();
  });

  it.each(["missing", "denied"] as const)(
    "keeps the gesture-bound legacy copy fallback when the clipboard API is %s",
    async (availability) => {
      if (availability === "denied") {
        Object.defineProperty(navigator, "clipboard", {
          configurable: true,
          value: { writeText: vi.fn().mockRejectedValue(new Error("permission denied")) },
        });
      }
      const execCommand = vi.fn((command: string) => {
        expect(command).toBe("copy");
        expect(document.querySelector<HTMLTextAreaElement>('textarea[readonly]')?.value).toBe(
          "copy me",
        );
        return true;
      });
      Object.defineProperty(document, "execCommand", {
        configurable: true,
        value: execCommand,
      });

      await expect(writeClipboardText("copy me")).resolves.toBe(true);
      expect(execCommand).toHaveBeenCalledTimes(1);
      expect(document.querySelector('textarea[readonly]')).toBeNull();
    },
  );
});
