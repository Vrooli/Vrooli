/**
 * Tests for CollapsibleSection — the standard persisted disclosure.
 *
 * Pins: default open/closed state, toggling, and that each storageKey
 * remembers its own state independently across remounts.
 */

import { beforeEach, describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { CollapsibleSection } from "./collapsible-section";

function renderSection(storageKey: string, defaultOpen = false) {
  return render(
    <CollapsibleSection
      storageKey={storageKey}
      defaultOpen={defaultOpen}
      label="Section"
      toggleTestId={`toggle-${storageKey}`}
      contentTestId={`content-${storageKey}`}
    >
      <p>body</p>
    </CollapsibleSection>,
  );
}

describe("CollapsibleSection", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("respects defaultOpen when nothing is persisted", () => {
    const closed = renderSection("a");
    expect(screen.queryByTestId("content-a")).toBeNull();
    closed.unmount();

    renderSection("b", true);
    expect(screen.getByTestId("content-b")).toBeInTheDocument();
  });

  it("toggles content and aria-expanded", async () => {
    const user = userEvent.setup();
    renderSection("a");

    const toggle = screen.getByTestId("toggle-a");
    expect(toggle).toHaveAttribute("aria-expanded", "false");

    await user.click(toggle);
    expect(toggle).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByTestId("content-a")).toBeInTheDocument();

    await user.click(toggle);
    expect(screen.queryByTestId("content-a")).toBeNull();
  });

  it("remembers state across remounts, independently per storageKey", async () => {
    const user = userEvent.setup();

    const first = renderSection("a");
    await user.click(screen.getByTestId("toggle-a"));
    first.unmount();

    const second = renderSection("b", true);
    await user.click(screen.getByTestId("toggle-b"));
    second.unmount();

    // "a" was opened; "b" was closed — each key restores its own state.
    renderSection("a");
    expect(screen.getByTestId("content-a")).toBeInTheDocument();

    renderSection("b", true);
    expect(screen.queryByTestId("content-b")).toBeNull();
  });
});
