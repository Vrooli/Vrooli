import { useLayoutEffect, useRef, useState } from "react";
import { fireEvent, screen } from "@testing-library/react";
import { renderWithProviders } from "../test-utils";
import { describe, expect, it, vi } from "vitest";

import { useDialogFocusTrap } from "./useDialogFocusTrap";

function EmptyDialog({ selector = "[role=dialog]" }: { selector?: string }) {
  const [open] = useState(true);
  const onClose = vi.fn();
  useDialogFocusTrap(open, onClose, selector);
  return <div role="dialog"><p>No focusable controls</p></div>;
}


function FocusableDialog({ onClose }: { onClose: () => void }) {
  const previousSvg = useRef<SVGSVGElement>(null);
  useLayoutEffect(() => previousSvg.current?.focus(), []);
  useDialogFocusTrap(true, onClose);
  return <><svg ref={previousSvg} tabIndex={0} aria-label="previous focus" /><div role="dialog"><button type="button">First action</button><button type="button">Last action</button></div></>;
}

function MissingDialog() {
  useDialogFocusTrap(true, vi.fn(), "#missing-dialog");
  return <p>Dialog has not mounted</p>;
}

describe("useDialogFocusTrap", () => {
  it("prevents Tab from leaving an open dialog with no focusable controls", () => {
    renderWithProviders(<EmptyDialog />);
    const dialog = screen.getByRole("dialog");
    const event = new KeyboardEvent("keydown", { key: "Tab", bubbles: true, cancelable: true });
    fireEvent(dialog, event);
    expect(event.defaultPrevented).toBe(true);
  });

  it("wraps Tab focus at both ends of a dialog and closes on Escape", () => {
    const onClose = vi.fn();
    renderWithProviders(<FocusableDialog onClose={onClose} />);
    const first = screen.getByRole("button", { name: "First action" });
    const last = screen.getByRole("button", { name: "Last action" });
    expect(first).toHaveFocus();
    first.focus();
    fireEvent.keyDown(first, { key: "Tab", shiftKey: true });
    expect(last).toHaveFocus();
    last.focus();
    fireEvent.keyDown(last, { key: "Tab" });
    expect(first).toHaveFocus();
    fireEvent.keyDown(screen.getByRole("dialog"), { key: "Escape" });
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("tolerates an open state before its dialog element is mounted", () => {
    renderWithProviders(<MissingDialog />);
    expect(screen.getByText("Dialog has not mounted")).toBeInTheDocument();
  });
});
