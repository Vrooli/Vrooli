import { act, fireEvent, render, screen } from "@testing-library/react";
import { useRef, useState } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ContextMenu } from "@vrooli/react-component-library/ContextMenu/1.3.2";

const originalMatchMedia = window.matchMedia;

function useDesktopViewport() {
  window.matchMedia = vi.fn((query: string) => ({
    matches: query.includes("min-width"),
    media: query,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })) as unknown as typeof window.matchMedia;
}

describe("ContextMenu 1.3.2 dismissal and triggers", () => {
  beforeEach(() => {
    useDesktopViewport();
  });
  afterEach(() => {
    window.matchMedia = originalMatchMedia;
    vi.useRealTimers();
  });

  it("closes the anchored menu on a press outside its surface", () => {
    const onOpenChange = vi.fn();
    render(
      <>
        <button type="button">Elsewhere</button>
        <ContextMenu
          open
          onOpenChange={onOpenChange}
          position={{ x: 40, y: 40 }}
          title="Actions"
          closeLabel="Close actions"
          items={[{ id: "open", label: "Open", onSelect: vi.fn() }]}
        />
      </>,
    );
    expect(document.querySelector("[data-rcl-context-menu]")).toHaveAttribute(
      "data-presentation",
      "menu",
    );
    fireEvent.pointerDown(screen.getByRole("button", { name: "Elsewhere" }));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("lets a consumer anchor toggle the menu closed instead of reopening it", () => {
    function AnchoredToggle() {
      const anchor = useRef<HTMLButtonElement>(null);
      const [open, setOpen] = useState(true);
      return (
        <>
          <button ref={anchor} type="button" onClick={() => setOpen((value) => !value)}>
            Toggle
          </button>
          <ContextMenu
            open={open}
            onOpenChange={setOpen}
            anchorRef={anchor}
            triggers={[]}
            title="Actions"
            closeLabel="Close actions"
            items={[{ id: "open", label: "Open", onSelect: vi.fn() }]}
          />
        </>
      );
    }
    render(<AnchoredToggle />);
    const toggle = screen.getByRole("button", { name: "Toggle" });
    fireEvent.pointerDown(toggle);
    fireEvent.click(toggle);
    // The surface stays mounted for its exit transition; closed, not reopened.
    expect(screen.getByRole("menu")).toHaveAttribute("data-state", "closed");
  });

  it("closes when another row inside the cloned child trigger is pressed", () => {
    const onOpenChange = vi.fn();
    render(
      <ContextMenu
        open
        onOpenChange={onOpenChange}
        position={{ x: 40, y: 40 }}
        title="Actions"
        closeLabel="Close actions"
        items={[{ id: "open", label: "Open", onSelect: vi.fn() }]}
      >
        <div>
          <button type="button">Other row</button>
        </div>
      </ContextMenu>,
    );
    fireEvent.pointerDown(screen.getByRole("button", { name: "Other row" }));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("keeps the menu open for presses inside it and still selects items", () => {
    const onOpenChange = vi.fn();
    const onSelect = vi.fn();
    render(
      <ContextMenu
        open
        onOpenChange={onOpenChange}
        position={{ x: 40, y: 40 }}
        title="Actions"
        closeLabel="Close actions"
        items={[{ id: "open", label: "Open", onSelect }]}
      />,
    );
    const item = screen.getByRole("menuitem", { name: "Open" });
    fireEvent.pointerDown(item);
    expect(onOpenChange).not.toHaveBeenCalled();
    fireEvent.click(item);
    expect(onSelect).toHaveBeenCalledOnce();
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("does not open from a held mouse button", () => {
    vi.useFakeTimers();
    const onOpenAt = vi.fn();
    render(
      <ContextMenu open={false} title="Actions" closeLabel="Close actions" items={[]} onOpenAt={onOpenAt}>
        <button type="button">Row</button>
      </ContextMenu>,
    );
    fireEvent.pointerDown(screen.getByRole("button", { name: "Row" }), {
      pointerId: 1,
      pointerType: "mouse",
      button: 0,
    });
    act(() => vi.advanceTimersByTime(600));
    expect(onOpenAt).not.toHaveBeenCalled();
  });

  it("opens from a touch long press", () => {
    vi.useFakeTimers();
    const onOpenAt = vi.fn();
    render(
      <ContextMenu open={false} title="Actions" closeLabel="Close actions" items={[]} onOpenAt={onOpenAt}>
        <button type="button">Row</button>
      </ContextMenu>,
    );
    fireEvent.pointerDown(screen.getByRole("button", { name: "Row" }), {
      pointerId: 1,
      pointerType: "touch",
      clientX: 10,
      clientY: 20,
      button: 0,
    });
    act(() => vi.advanceTimersByTime(450));
    expect(onOpenAt).toHaveBeenCalledWith({ x: 10, y: 20, pointerType: "touch" });
  });
});
