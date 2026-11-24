import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../../test-utils";
import { CopyableCommand } from "./CopyableCommand";

const clipboardDescriptor = Object.getOwnPropertyDescriptor(navigator, "clipboard");

function setClipboard(value: Clipboard | undefined) {
  if (value === undefined) {
    Reflect.deleteProperty(navigator, "clipboard");
    return;
  }
  Object.defineProperty(navigator, "clipboard", { configurable: true, value });
}

afterEach(() => {
  cleanup();
  if (clipboardDescriptor) {
    Object.defineProperty(navigator, "clipboard", clipboardDescriptor);
  } else {
    Reflect.deleteProperty(navigator, "clipboard");
  }
});

describe("CopyableCommand", () => {
  it("reports copied only after the clipboard accepts the command", async () => {
    const user = userEvent.setup();
    const writeText = vi.fn().mockResolvedValue(undefined);
    setClipboard({ writeText } as unknown as Clipboard);
    renderWithProviders(<CopyableCommand command="audio-tools stt engines" copyLabel="Copy" copiedLabel="Copied" />);

    await user.click(screen.getByRole("button", { name: "Copy" }));
    expect(writeText).toHaveBeenCalledWith("audio-tools stt engines");
    expect(screen.getByRole("button", { name: "Copied" })).toBeInTheDocument();
  });

  it.each([
    ["clipboard is unavailable", undefined],
    ["clipboard write is rejected", { writeText: vi.fn().mockRejectedValue(new Error("denied")) }],
  ])("keeps the copy action label when %s", async (_name, clipboard) => {
    const user = userEvent.setup();
    setClipboard(clipboard as Clipboard | undefined);
    renderWithProviders(<CopyableCommand command="audio-tools stt engines" copyLabel="Copy" copiedLabel="Copied" />);

    await user.click(screen.getByRole("button", { name: "Copy" }));
    expect(screen.getByRole("button", { name: "Copy" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Copied" })).not.toBeInTheDocument();
  });
});
