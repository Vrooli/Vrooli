import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";

import { selectors } from "../consts/selectors";
import { KeyboardShortcutHint } from "./KeyboardShortcutHint";

describe("KeyboardShortcutHint", () => {
  it("renders multiple Kbd tokens for a chord with modifiers", () => {
    render(<KeyboardShortcutHint chord="mod+k" />);
    const root = screen.getByTestId(selectors.shared.keyboardShortcut.root);
    const kbds = root.querySelectorAll("kbd");
    expect(kbds.length).toBe(2);
  });

  it("renders a single Kbd for a one-key chord", () => {
    render(<KeyboardShortcutHint chord="escape" />);
    const root = screen.getByTestId(selectors.shared.keyboardShortcut.root);
    const kbds = root.querySelectorAll("kbd");
    expect(kbds.length).toBe(1);
  });
});
