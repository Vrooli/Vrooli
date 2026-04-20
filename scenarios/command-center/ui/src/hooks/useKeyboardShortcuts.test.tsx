// [REQ:CC-UI-004] Keyboard shortcut hook relays intent to host frame and navigates.
// focus-visible styles are provided globally in styles.css (*:focus-visible) for keyboard/D-pad kiosk navigation.
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { render, fireEvent } from "@testing-library/react";
import { MemoryRouter, Routes, Route, useLocation } from "react-router-dom";
import { KIOSK_SHORTCUTS, useKeyboardShortcuts } from "./useKeyboardShortcuts";

const emitShortcutIntentMock = vi.fn();

vi.mock("@vrooli/iframe-bridge", () => ({
  emitShortcutIntent: (...args: unknown[]) => emitShortcutIntentMock(...args),
}));

function TestHarness() {
  useKeyboardShortcuts();
  const location = useLocation();
  return <div data-testid="path">{location.pathname}</div>;
}

function renderAt(initialPath: string) {
  return render(
    <MemoryRouter initialEntries={[initialPath]}>
      <Routes>
        <Route path="*" element={<TestHarness />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("useKeyboardShortcuts", () => {
  beforeEach(() => {
    emitShortcutIntentMock.mockReset();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("defines six kiosk shortcuts (one per dashboard)", () => {
    expect(KIOSK_SHORTCUTS).toHaveLength(6);
    const paths = KIOSK_SHORTCUTS.map((s) => s.path);
    expect(paths).toEqual([
      "/mission-control",
      "/hive",
      "/forge",
      "/ledger",
      "/broadcast",
      "/panorama",
    ]);
  });

  it("navigates on digit keys and emits shortcut intent", () => {
    const { getByTestId } = renderAt("/mission-control");
    fireEvent.keyDown(window, { key: "2" });
    expect(getByTestId("path").textContent).toBe("/hive");
    expect(emitShortcutIntentMock).toHaveBeenCalledTimes(1);
    expect(emitShortcutIntentMock).toHaveBeenCalledWith(
      expect.objectContaining({
        action: "command-center.nav.hive",
        outcome: "handled",
        chord: "2",
        source: "keyboard",
      }),
    );
  });

  it("ignores keys when modifier is held", () => {
    const { getByTestId } = renderAt("/mission-control");
    fireEvent.keyDown(window, { key: "3", ctrlKey: true });
    expect(getByTestId("path").textContent).toBe("/mission-control");
    expect(emitShortcutIntentMock).not.toHaveBeenCalled();
  });

  it("ignores keys when target is an editable element", () => {
    const input = document.createElement("input");
    document.body.appendChild(input);
    try {
      const { getByTestId } = renderAt("/mission-control");
      fireEvent.keyDown(input, { key: "4" });
      expect(getByTestId("path").textContent).toBe("/mission-control");
      expect(emitShortcutIntentMock).not.toHaveBeenCalled();
    } finally {
      input.remove();
    }
  });

  it("ignores keys not in the shortcut set", () => {
    const { getByTestId } = renderAt("/mission-control");
    fireEvent.keyDown(window, { key: "z" });
    expect(getByTestId("path").textContent).toBe("/mission-control");
    expect(emitShortcutIntentMock).not.toHaveBeenCalled();
  });
});
