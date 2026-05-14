import { afterEach, describe, expect, it, vi } from "vitest";
import { fireEvent, renderHook } from "@testing-library/react";

const bridge = vi.hoisted(() => ({
  emitShortcutIntent: vi.fn(),
}));

vi.mock("@vrooli/iframe-bridge", () => ({
  emitShortcutIntent: bridge.emitShortcutIntent,
}));

import { useMobileDrawerShortcuts } from "./useMobileDrawerShortcuts";

describe("useMobileDrawerShortcuts", () => {
  afterEach(() => {
    bridge.emitShortcutIntent.mockReset();
  });

  it("closes and emits a handled shortcut intent on Escape", () => {
    const onClose = vi.fn();
    renderHook(() => useMobileDrawerShortcuts({ open: true, onClose }));

    fireEvent.keyDown(window, { key: "Escape" });

    expect(onClose).toHaveBeenCalledOnce();
    expect(bridge.emitShortcutIntent).toHaveBeenCalledWith(
      expect.objectContaining({
        action: "react-component-library.mobile-drawer.close",
        chord: "Escape",
        outcome: "handled",
      }),
    );
  });

  it("does nothing while closed", () => {
    const onClose = vi.fn();
    renderHook(() => useMobileDrawerShortcuts({ open: false, onClose }));

    fireEvent.keyDown(window, { key: "Escape" });

    expect(onClose).not.toHaveBeenCalled();
    expect(bridge.emitShortcutIntent).not.toHaveBeenCalled();
  });
});
