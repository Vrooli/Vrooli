import { act, fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useHoverIntent, insideTriangle } from "../../../../components/useHoverIntent/versions/1.0.1/useHoverIntent.tsx";
import { renderWithProviders } from "../../../../../ui/src/test-utils";
import { useRef, useState } from "react";

describe("useHoverIntent safe polygon", () => {
  const last = { x: 10, y: 50 };
  const top = { x: 200, y: 0 };
  const bottom = { x: 200, y: 100 };

  it("holds a pointer travelling toward the child through the background", () => {
    expect(insideTriangle({ x: 120, y: 50 }, last, top, bottom)).toBe(true);
  });

  it("does not hold a pointer travelling away from the child", () => {
    expect(insideTriangle({ x: 40, y: 150 }, last, top, bottom)).toBe(false);
  });

  it("uses the measured child corners rather than a bounding box", () => {
    expect(insideTriangle({ x: 120, y: 95 }, last, top, bottom)).toBe(false);
  });

  it("opens only for a fine pointer and closes when the pointer travels away", () => {
    vi.useFakeTimers();
    const matchMedia = vi.fn(() => ({ matches: true, media: "", onchange: null, addListener: vi.fn(), removeListener: vi.fn(), addEventListener: vi.fn(), removeEventListener: vi.fn(), dispatchEvent: vi.fn() }));
    Object.defineProperty(window, "matchMedia", { configurable: true, value: matchMedia });
    function Harness() {
      const [open, setOpen] = useState(false);
      const child = useRef<HTMLDivElement>(null);
      const hover = useHoverIntent({ childRect: () => child.current?.getBoundingClientRect() ?? null, onOpen: () => setOpen(true), onClose: () => setOpen(false) });
      const { onChildEnter: _onChildEnter, cancel: _cancel, ...pointerHandlers } = hover;
      return <div data-testid="hover" {...pointerHandlers} ref={child} data-open={open} />;
    }
    renderWithProviders(<Harness />);
    const target = screen.getByTestId("hover");
    fireEvent.pointerEnter(target, { pointerType: "mouse", clientX: 10, clientY: 50 });
    act(() => vi.advanceTimersByTime(280));
    expect(target).toHaveAttribute("data-open", "true");
    fireEvent.pointerMove(target, { clientX: 0, clientY: 200 });
    act(() => vi.advanceTimersByTime(100));
    expect(target).toHaveAttribute("data-open", "false");
    vi.useRealTimers();
  });
});
