import { renderHook } from "@testing-library/react";
import type { MouseEvent as ReactMouseEvent, PointerEvent as ReactPointerEvent } from "react";
import { describe, expect, it, vi } from "vitest";
import { usePillGestures } from "./usePillGestures";

function pointerAt(target: HTMLElement, x: number, y: number) {
  return {
    clientX: x,
    clientY: y,
    pointerId: 1,
    pointerType: "touch",
    button: 0,
    target,
    currentTarget: target,
    preventDefault: vi.fn(),
    stopPropagation: vi.fn(),
  } as unknown as ReactPointerEvent<HTMLElement>;
}

function setup({ swipeTracks = true }: { swipeTracks?: boolean } = {}) {
  const callbacks = { onTap: vi.fn(), onDismiss: vi.fn(), onNext: vi.fn(), onPrevious: vi.fn(), onScrub: vi.fn() };
  const pill = document.createElement("div");
  const zone = document.createElement("div");
  pill.appendChild(zone);
  zone.getBoundingClientRect = () => ({ left: 0, width: 200, top: 0, height: 8, right: 200, bottom: 8, x: 0, y: 0, toJSON: () => ({}) });
  const { result } = renderHook(() => usePillGestures({ ...callbacks, progressRef: { current: zone }, swipeTracks }));
  const gesture = (target: HTMLElement, points: Array<[number, number]>) => {
    const [first, ...rest] = points;
    if (!first) return;
    result.current.onPointerDown(pointerAt(target, first[0], first[1]));
    for (const [x, y] of rest) result.current.onPointerMove(pointerAt(target, x, y));
    const last = rest[rest.length - 1] ?? first;
    result.current.onPointerUp(pointerAt(target, last[0], last[1]));
  };
  return { callbacks, pill, zone, result, gesture };
}

describe("usePillGestures", () => {
  it("[REQ:P0-017h] a touch that moves under 8 px is a tap", () => {
    const { callbacks, pill, gesture } = setup();
    gesture(pill, [[100, 100], [104, 105]]);
    expect(callbacks.onTap).toHaveBeenCalledTimes(1);
    expect(callbacks.onDismiss).not.toHaveBeenCalled();
    expect(callbacks.onNext).not.toHaveBeenCalled();
  });

  it("[REQ:P0-017h] a drag more than 48 px down dismisses", () => {
    const { callbacks, pill, gesture } = setup();
    gesture(pill, [[100, 100], [102, 130], [103, 160]]);
    expect(callbacks.onDismiss).toHaveBeenCalledTimes(1);
    expect(callbacks.onTap).not.toHaveBeenCalled();
  });

  it("a drag up does not dismiss", () => {
    const { callbacks, pill, gesture } = setup();
    gesture(pill, [[100, 160], [100, 130], [100, 100]]);
    expect(callbacks.onDismiss).not.toHaveBeenCalled();
    expect(callbacks.onTap).not.toHaveBeenCalled();
  });

  it("[REQ:P0-017h] a horizontal drag over 64 px changes track by direction", () => {
    const { callbacks, pill, gesture } = setup();
    gesture(pill, [[200, 100], [150, 101], [120, 102]]);
    expect(callbacks.onNext).toHaveBeenCalledTimes(1);
    gesture(pill, [[100, 100], [140, 99], [180, 98]]);
    expect(callbacks.onPrevious).toHaveBeenCalledTimes(1);
  });

  it("a horizontal drag under 64 px changes nothing", () => {
    const { callbacks, pill, gesture } = setup();
    gesture(pill, [[100, 100], [150, 100]]);
    expect(callbacks.onNext).not.toHaveBeenCalled();
    expect(callbacks.onPrevious).not.toHaveBeenCalled();
    expect(callbacks.onTap).not.toHaveBeenCalled();
  });

  it("horizontal drags leave the track alone while track swipes are off", () => {
    const { callbacks, pill, gesture } = setup({ swipeTracks: false });
    gesture(pill, [[200, 100], [120, 102]]);
    expect(callbacks.onNext).not.toHaveBeenCalled();
  });

  it("[REQ:P0-017h] a drag inside the progress zone scrubs with a 0..1 ratio", () => {
    const { callbacks, zone, gesture } = setup();
    gesture(zone, [[20, 4], [100, 4], [150, 4]]);
    expect(callbacks.onScrub).toHaveBeenLastCalledWith(0.75);
    expect(callbacks.onNext).not.toHaveBeenCalled();
    expect(callbacks.onPrevious).not.toHaveBeenCalled();
    gesture(zone, [[20, 4], [260, 4]]);
    expect(callbacks.onScrub).toHaveBeenLastCalledWith(1);
  });

  it("[REQ:P0-017h] a drag that starts vertical never becomes horizontal", () => {
    const { callbacks, pill, gesture } = setup();
    gesture(pill, [[200, 100], [200, 120], [100, 125]]);
    expect(callbacks.onNext).not.toHaveBeenCalled();
    expect(callbacks.onDismiss).not.toHaveBeenCalled();
  });

  it("a drag swallows the click that follows it; a tap does not", () => {
    const { pill, result, gesture } = setup();
    const click = () => {
      const stopPropagation = vi.fn();
      return { event: { preventDefault: vi.fn(), stopPropagation } as unknown as ReactMouseEvent<HTMLElement>, stopPropagation };
    };
    gesture(pill, [[100, 100], [100, 170]]);
    const afterDrag = click();
    result.current.onClickCapture(afterDrag.event);
    expect(afterDrag.stopPropagation).toHaveBeenCalled();
    gesture(pill, [[100, 100], [101, 101]]);
    const afterTap = click();
    result.current.onClickCapture(afterTap.event);
    expect(afterTap.stopPropagation).not.toHaveBeenCalled();
  });
});
