import { act, cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils";
import { useHoverIntent } from "../../../library/components/useHoverIntent/versions/1.0.3/useHoverIntent";
import { useRef, useState } from "react";

function Harness() {
  const child = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);
  const hover = useHoverIntent({
    childRect: () => child.current?.getBoundingClientRect() ?? null,
    onOpen: () => setOpen(true),
    onClose: () => setOpen(false),
    openDelay: 100,
    closeDelay: 80,
  });
  return (
    <div>
      <p role="status">{open ? "Submenu open" : "Submenu closed"}</p>
      <button
        type="button"
        onPointerEnter={hover.onPointerEnter}
        onPointerMove={hover.onPointerMove}
        onPointerLeave={hover.onPointerLeave}
      >
        Preview submenu
      </button>
      {open ? <div ref={child} onPointerEnter={hover.onChildEnter}>Release notes</div> : null}
    </div>
  );
}

const setFinePointer = () => {
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    value: vi.fn(() => ({
      matches: true,
      media: "(hover: hover) and (pointer: fine)",
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
};

afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

describe("useHoverIntent draft story behavior", () => {
  it("opens after the configured mouse delay and closes after pointer leave", () => {
    vi.useFakeTimers();
    setFinePointer();
    renderWithProviders(<Harness />);
    const trigger = screen.getByRole("button", { name: "Preview submenu" });

    fireEvent.pointerEnter(trigger, { pointerType: "mouse", clientX: 10, clientY: 20 });
    act(() => vi.advanceTimersByTime(99));
    expect(screen.getByRole("status")).toHaveTextContent("Submenu closed");
    act(() => vi.advanceTimersByTime(1));
    expect(screen.getByRole("status")).toHaveTextContent("Submenu open");

    fireEvent.pointerLeave(trigger);
    act(() => vi.advanceTimersByTime(80));
    expect(screen.getByRole("status")).toHaveTextContent("Submenu closed");
  });

  it("ignores touch pointer entry", () => {
    vi.useFakeTimers();
    setFinePointer();
    renderWithProviders(<Harness />);
    fireEvent.pointerEnter(screen.getByRole("button", { name: "Preview submenu" }), {
      pointerType: "touch",
      clientX: 10,
      clientY: 20,
    });
    act(() => vi.advanceTimersByTime(200));
    expect(screen.getByRole("status")).toHaveTextContent("Submenu closed");
  });
});
