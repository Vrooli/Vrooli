import { fireEvent, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

const { emitShortcutIntent } = vi.hoisted(() => ({ emitShortcutIntent: vi.fn() }));
vi.mock("@vrooli/iframe-bridge", () => ({ emitShortcutIntent }));

import { useHostShortcutRelay } from "./useHostShortcutRelay";

describe("useHostShortcutRelay", () => {
  afterEach(() => emitShortcutIntent.mockReset());

  it("relays modified shortcuts and ignores ordinary key presses", () => {
    renderHook(() => useHostShortcutRelay());
    fireEvent.keyDown(window, { key: "k" });
    expect(emitShortcutIntent).not.toHaveBeenCalled();

    fireEvent.keyDown(window, { key: "ArrowUp", ctrlKey: true, shiftKey: true });
    expect(emitShortcutIntent).toHaveBeenCalledWith(
      expect.objectContaining({ chord: "ctrl+shift+ArrowUp" }),
    );
  });

  it("does not relay events already handled or targeted at editable controls", () => {
    renderHook(() => useHostShortcutRelay());
    const input = document.createElement("input");
    document.body.appendChild(input);
    fireEvent.keyDown(input, { key: "k", metaKey: true });
    emitShortcutIntent.mockClear();
    const handled = new KeyboardEvent("keydown", { key: "k", metaKey: true, cancelable: true });
    handled.preventDefault();
    window.dispatchEvent(handled);
    expect(emitShortcutIntent).not.toHaveBeenCalled();
    input.remove();
  });
});
