import { useState } from "react";
import { describe, expect, it } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";

import { useFocusTrap } from "./useFocusTrap";

function TrapFixture() {
  const [open, setOpen] = useState(false);
  const ref = useFocusTrap<HTMLDivElement>(open, () => setOpen(false));
  return (
    <div>
      <button type="button" onClick={() => setOpen(true)}>
        Open
      </button>
      {open ? (
        <div ref={ref} role="dialog" aria-modal="true" aria-label="Trap">
          <button type="button">First</button>
          <button type="button">Last</button>
        </div>
      ) : null}
    </div>
  );
}

describe("useFocusTrap", () => {
  it("moves focus in, wraps Tab, closes on Escape and restores the trigger", async () => {
    render(<TrapFixture />);
    const trigger = screen.getByRole("button", { name: "Open" });
    trigger.focus();
    fireEvent.click(trigger);

    const first = screen.getByRole("button", { name: "First" });
    const last = screen.getByRole("button", { name: "Last" });
    await waitFor(() => expect(document.activeElement).toBe(first));

    last.focus();
    fireEvent.keyDown(document, { key: "Tab" });
    expect(document.activeElement).toBe(first);

    first.focus();
    fireEvent.keyDown(document, { key: "Tab", shiftKey: true });
    expect(document.activeElement).toBe(last);

    fireEvent.keyDown(document, { key: "Escape" });
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(document.activeElement).toBe(trigger);
  });
});
