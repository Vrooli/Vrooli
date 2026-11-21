import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { SwipeActions, swipeActionsStyles } from "@vrooli/react-component-library/SwipeActions/1.4.0";

describe("SwipeActions current release contract", () => {
  it("keeps a revealed action available and exposes the gesture claim", async () => {
    const onSelect = vi.fn();
    render(<SwipeActions defaultOpen actions={[{ id: "archive", label: "Archive", onSelect }]}><span>Message</span></SwipeActions>);
    expect(screen.getByTestId("patterns.swipe-actions")).toHaveAttribute("data-rcl-gesture-claim");
    fireEvent.click(screen.getByRole("button", { name: "Archive" }), { detail: 0 });
    expect(onSelect).toHaveBeenCalledOnce();
  });

  // --color-danger-foreground mixes the danger hue into the foreground: it is
  // danger-toned text for a NEUTRAL surface. On a danger background it paints
  // the label nearly the colour it sits on, which reads as a disabled control.
  it("paints destructive labels with the on-danger token, not the on-surface one", () => {
    const destructive = swipeActionsStyles
      .split("\n")
      .find((line) => line.includes('[data-tone="destructive"]'));
    expect(destructive).toBeDefined();
    expect(destructive).toContain("--color-danger-foreground-inverse");
    expect(destructive).not.toMatch(/color:\s*var\(--color-danger-foreground[,)]/);
  });

  it("hides action words when asked, keeping them as the accessible name", () => {
    render(
      <SwipeActions
        defaultOpen
        actionLabels="hidden"
        actions={[{ id: "close", label: "Close", icon: <svg data-testid="close-mark" />, onSelect: vi.fn() }]}
      >
        <span>Message</span>
      </SwipeActions>,
    );
    const action = screen.getByRole("button", { name: "Close" });
    expect(screen.getByTestId("close-mark")).toBeInTheDocument();
    expect(action).not.toHaveTextContent("Close");
  });

  // Hiding both the word and the mark would leave a coloured slab that says
  // nothing, so an action with no icon keeps its label whatever the prop says.
  it("keeps the word for an action that has no icon", () => {
    render(
      <SwipeActions defaultOpen actionLabels="hidden" actions={[{ id: "close", label: "Close", onSelect: vi.fn() }]}>
        <span>Message</span>
      </SwipeActions>,
    );
    expect(screen.getByRole("button", { name: "Close" })).toHaveTextContent("Close");
  });
});
