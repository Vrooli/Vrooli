/**
 * Sheet tests — the app's overlay primitive (centered modal on desktop,
 * full-screen drawer on mobile). Pins the modal essentials it owns: closed →
 * renders nothing; open → role="dialog"/aria-modal wired to the title; the
 * backdrop + header close button both fire onClose; Escape closes; the optional
 * subtitle slot; body scroll-lock on open and its restoration on close; initial
 * focus into the panel; and the Tab focus-trap loop at both edges.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { Sheet } from "./sheet";

const renderSheet = (props: Partial<Parameters<typeof Sheet>[0]> = {}) =>
  render(
    <Sheet
      open={props.open ?? true}
      onClose={props.onClose ?? vi.fn()}
      title={props.title ?? "Pick a model"}
      subtitle={props.subtitle}
      closeLabel={props.closeLabel ?? "Close"}
      testId={props.testId ?? "sheet"}
    >
      {props.children ?? (
        <>
          <button type="button">first</button>
          <button type="button">last</button>
        </>
      )}
    </Sheet>,
  );

afterEach(() => {
  cleanup();
  document.body.style.overflow = "";
});

describe("Sheet", () => {
  it("renders nothing when closed", () => {
    renderSheet({ open: false });
    expect(screen.queryByTestId("sheet")).not.toBeInTheDocument();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("renders a labelled modal dialog with the title when open", () => {
    renderSheet();
    const dialog = screen.getByRole("dialog");
    expect(dialog).toHaveAttribute("aria-modal", "true");
    // The title is wired via aria-labelledby → the heading element.
    const labelledBy = dialog.getAttribute("aria-labelledby");
    expect(labelledBy).toBeTruthy();
    expect(document.getElementById(labelledBy ?? "")).toHaveTextContent("Pick a model");
  });

  it("renders the optional subtitle when provided and omits it otherwise", () => {
    const { unmount } = renderSheet({ subtitle: <span>16 cores · 64 GB</span> });
    expect(screen.getByText(/16 cores/)).toBeInTheDocument();
    unmount();

    renderSheet();
    expect(screen.queryByText(/16 cores/)).not.toBeInTheDocument();
  });

  it("fires onClose when the header close button is clicked", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderSheet({ onClose });
    await user.click(screen.getByTestId("sheet-close"));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("fires onClose when the backdrop is clicked", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderSheet({ onClose });
    // The backdrop is a tabIndex=-1 button carrying the close label.
    const backdrops = screen.getAllByRole("button", { name: "Close" });
    const backdrop = backdrops.find((b) => b.getAttribute("tabindex") === "-1");
    expect(backdrop).toBeDefined();
    await user.click(backdrop as HTMLElement);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("fires onClose on Escape", () => {
    const onClose = vi.fn();
    renderSheet({ onClose });
    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("ignores non-trap keys", () => {
    const onClose = vi.fn();
    renderSheet({ onClose });
    fireEvent.keyDown(document, { key: "a" });
    expect(onClose).not.toHaveBeenCalled();
  });

  it("locks body scroll while open and restores it on close", () => {
    const { rerender } = renderSheet();
    expect(document.body.style.overflow).toBe("hidden");
    rerender(
      <Sheet open={false} onClose={vi.fn()} title="Pick a model" closeLabel="Close" testId="sheet">
        <button type="button">x</button>
      </Sheet>,
    );
    expect(document.body.style.overflow).not.toBe("hidden");
  });

  it("moves initial focus into the panel on open", () => {
    renderSheet();
    expect(screen.getByRole("dialog")).toHaveFocus();
  });

  it("wraps Tab focus from the last focusable back to the first", () => {
    renderSheet();
    const dialog = screen.getByRole("dialog");
    const buttons = within(dialog).getAllByRole("button");
    const [first, last] = [buttons.at(0)!, buttons.at(-1)!];

    last.focus();
    fireEvent.keyDown(document, { key: "Tab" });
    expect(first).toHaveFocus();
  });

  it("wraps Shift+Tab focus from the first focusable to the last", () => {
    renderSheet();
    const dialog = screen.getByRole("dialog");
    const buttons = within(dialog).getAllByRole("button");
    const [first, last] = [buttons.at(0)!, buttons.at(-1)!];

    first.focus();
    fireEvent.keyDown(document, { key: "Tab", shiftKey: true });
    expect(last).toHaveFocus();
  });

  it("leaves Tab alone when focus is in the middle of the trap", () => {
    renderSheet({
      children: (
        <>
          <button type="button">a</button>
          <button type="button">b</button>
          <button type="button">c</button>
        </>
      ),
    });
    const dialog = screen.getByRole("dialog");
    const middle = within(dialog).getAllByRole("button").at(1)!;
    middle.focus();
    fireEvent.keyDown(document, { key: "Tab" });
    // Focus is unchanged — the trap only fires at the two edges.
    expect(middle).toHaveFocus();
  });

  it("no-ops the Tab trap when the panel has no focusable children", () => {
    renderSheet({ children: <span>nothing focusable</span> });
    // The close + backdrop buttons live OUTSIDE the panelRef, so the panel's
    // own focusable set is empty — Tab must not throw.
    expect(() => fireEvent.keyDown(document, { key: "Tab" })).not.toThrow();
  });
});
