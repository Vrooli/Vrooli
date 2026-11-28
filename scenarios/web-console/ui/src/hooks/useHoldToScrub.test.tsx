import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { SCRUB_TIERS, useHoldToScrub } from "./useHoldToScrub";

const HOLD = 400;

/** A list of `size` items stepped forward by one button. */
function Harness({ size, onEnd }: { size: number; onEnd?: () => void }) {
  const [position, setPosition] = useState(0);
  const positionRef = { current: position };
  const { state, pressProps } = useHoldToScrub({
    holdDelay: HOLD,
    canStep: position < size - 1,
    step: (count) => {
      const moved = Math.min(count, size - 1 - positionRef.current);
      positionRef.current += moved;
      setPosition(positionRef.current);
      return moved;
    },
    onScrubEnd: onEnd,
  });
  return (
    <>
      <button type="button" data-testid="step" data-phase={state.phase} data-tier={state.tier} data-edge={state.atEdge} {...pressProps}>next</button>
      <output data-testid="position">{position}</output>
    </>
  );
}

const position = () => Number(screen.getByTestId("position").textContent);
const button = () => screen.getByTestId("step");
const advance = (ms: number) => { act(() => { vi.advanceTimersByTime(ms); }); };

describe("useHoldToScrub", () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout", "requestAnimationFrame", "cancelAnimationFrame", "performance", "Date"] });
  });
  afterEach(() => { vi.useRealTimers(); });

  it("steps once on a tap and never scrubs", () => {
    render(<Harness size={100} />);
    fireEvent.pointerDown(button(), { isPrimary: true, button: 0 });
    advance(100);
    fireEvent.pointerUp(window);
    fireEvent.click(button());
    expect(position()).toBe(1);
    expect(button()).toHaveAttribute("data-phase", "idle");
  });

  it("scrubs on a hold, faster the longer it lasts, and swallows the release click", () => {
    const onEnd = vi.fn();
    render(<Harness size={1000} onEnd={onEnd} />);
    fireEvent.pointerDown(button(), { isPrimary: true, button: 0 });
    advance(HOLD + 20);
    expect(button()).toHaveAttribute("data-phase", "scrubbing");
    expect(position()).toBe(1); // engages with a step

    advance(1000);
    const firstSecond = position() - 1;
    expect(firstSecond).toBeGreaterThanOrEqual(SCRUB_TIERS[0]!.perSecond - 1);

    advance(SCRUB_TIERS[3]!.after);
    expect(Number(button().getAttribute("data-tier"))).toBe(SCRUB_TIERS.length - 1);
    const before = position();
    advance(1000);
    expect(position() - before).toBeGreaterThan(firstSecond * 4);

    const held = position();
    fireEvent.pointerUp(window);
    fireEvent.click(button());
    expect(position()).toBe(held);
    expect(onEnd).toHaveBeenCalledTimes(1);
    expect(button()).toHaveAttribute("data-phase", "idle");
  });

  it("stops at the end and stays engaged until release", () => {
    render(<Harness size={4} />);
    fireEvent.pointerDown(button(), { isPrimary: true, button: 0 });
    advance(HOLD + 3000);
    expect(position()).toBe(3);
    expect(button()).toHaveAttribute("data-edge", "true");
    expect(button()).toHaveAttribute("data-phase", "scrubbing");
    fireEvent.pointerUp(window);
    expect(button()).toHaveAttribute("data-phase", "idle");
  });

  it("abandons a cancelled press without stepping", () => {
    render(<Harness size={100} />);
    fireEvent.pointerDown(button(), { isPrimary: true, button: 0 });
    fireEvent.pointerCancel(window);
    advance(HOLD * 3);
    expect(position()).toBe(0);
  });

  it("holds and taps from the keyboard too", () => {
    render(<Harness size={1000} />);
    fireEvent.keyDown(button(), { key: "Enter" });
    fireEvent.keyUp(window, { key: "Enter" });
    expect(position()).toBe(1);

    fireEvent.keyDown(button(), { key: " " });
    fireEvent.keyDown(button(), { key: " ", repeat: true });
    advance(HOLD + 1000);
    expect(position()).toBeGreaterThan(2);
    const held = position();
    fireEvent.keyUp(window, { key: " " });
    expect(position()).toBe(held);
  });
});
