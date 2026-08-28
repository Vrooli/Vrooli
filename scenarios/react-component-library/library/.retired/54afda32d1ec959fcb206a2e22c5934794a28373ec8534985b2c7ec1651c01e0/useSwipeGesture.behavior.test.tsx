import { useRef, useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { fireEvent, screen } from "@testing-library/react";

import { renderWithProviders } from "../../../../../ui/src/test-utils";
import {
  useSwipeGesture,
  type SwipeGestureFrame,
  type SwipeGestureRelease,
  type UseSwipeGestureOptions,
} from "./useSwipeGesture";

const STAGES = [60, 140] as const;

interface HarnessProps extends Partial<UseSwipeGestureOptions> {
  onFrames?: (frames: SwipeGestureFrame[]) => void;
}

/**
 * The hook reports through callbacks rather than state by design, so the
 * harness records frames into a ref and exposes only the low-frequency facts
 * (the last release, the armed stage) through the DOM.
 */
function Harness({ onFrames, ...overrides }: HarnessProps) {
  const frames = useRef<SwipeGestureFrame[]>([]);
  const [stage, setStage] = useState(0);
  const [release, setRelease] = useState<SwipeGestureRelease | null>(null);
  const restingRef = useRef(0);

  const { onPointerDown } = useSwipeGesture({
    direction: "right",
    stages: STAGES,
    startOffset: () => restingRef.current,
    onMove: (frame) => {
      frames.current.push(frame);
      onFrames?.(frames.current);
    },
    onStageChange: setStage,
    onRelease: (result) => {
      setRelease(result);
      restingRef.current = result.outcome === "rest" ? STAGES[STAGES.length - 1] : 0;
    },
    ...overrides,
  });

  return (
    <div
      data-testid="surface"
      onPointerDown={onPointerDown}
      data-stage={stage}
      data-outcome={release?.outcome ?? ""}
      data-distance={release ? Math.round(release.distance) : ""}
      data-offset={release ? Math.round(release.offset) : ""}
      data-translate={release ? Math.round(release.translate) : ""}
    />
  );
}

/**
 * jsdom has no pointer capture and no coalesced pointer events, so a gesture is
 * driven as the browser would deliver it: one down on the element, moves and the
 * release on the window, each carrying a monotonic timeStamp.
 */
function pointer(overrides: { x?: number; y?: number; t?: number; id?: number } = {}) {
  return {
    pointerId: overrides.id ?? 1,
    clientX: overrides.x ?? 0,
    clientY: overrides.y ?? 0,
    button: 0,
    pointerType: "touch",
    // fireEvent does not forward timeStamp, so velocity is driven through the
    // property the hook actually reads.
    ...(overrides.t === undefined ? {} : { timeStamp: overrides.t }),
  };
}

function drag(points: Array<{ x: number; y?: number; t?: number }>, end: "up" | "cancel" = "up") {
  const surface = screen.getByTestId("surface");
  const first = points[0];
  fireEvent.pointerDown(surface, pointer({ x: first.x, y: first.y ?? 0, t: first.t ?? 0 }));
  for (const point of points.slice(1)) {
    fireEvent.pointerMove(window, pointer({ x: point.x, y: point.y ?? 0, t: point.t ?? 0 }));
  }
  const last = points[points.length - 1];
  const event = pointer({ x: last.x, y: last.y ?? 0, t: last.t ?? 0 });
  if (end === "up") fireEvent.pointerUp(window, event);
  else fireEvent.pointerCancel(window, event);
  return surface;
}

describe("axis locking", () => {
  it("ignores movement inside the slop radius", () => {
    const frames: SwipeGestureFrame[] = [];
    renderWithProviders(<Harness onMove={(f) => frames.push(f)} />);
    drag([{ x: 0 }, { x: 5 }, { x: 7 }]);
    expect(frames).toHaveLength(0);
  });

  it("locks to the inline axis and reports pixels", () => {
    const frames: SwipeGestureFrame[] = [];
    renderWithProviders(<Harness onMove={(f) => frames.push(f)} />);
    drag([{ x: 0 }, { x: 40 }, { x: 80 }]);
    expect(frames.map((f) => f.distance)).toEqual([40, 80]);
  });

  // The predecessor hook reported distance/threshold clamped to 1, which cannot
  // drive a surface that tracks a finger past its own threshold.
  it("keeps reporting beyond the last threshold instead of clamping", () => {
    const frames: SwipeGestureFrame[] = [];
    renderWithProviders(<Harness onMove={(f) => frames.push(f)} />);
    drag([{ x: 0 }, { x: 400 }]);
    expect(frames.at(-1)?.distance).toBe(400);
  });

  it("abandons a gesture that starts vertically", () => {
    const frames: SwipeGestureFrame[] = [];
    const surface = (renderWithProviders(<Harness onMove={(f) => frames.push(f)} />), drag([
      { x: 0, y: 0 },
      { x: 4, y: 40 },
      { x: 90, y: 60 },
    ]));
    expect(frames).toHaveLength(0);
    expect(surface.getAttribute("data-outcome")).toBe("abort");
  });

  it("does not re-decide the axis once locked", () => {
    const frames: SwipeGestureFrame[] = [];
    renderWithProviders(<Harness onMove={(f) => frames.push(f)} />);
    drag([{ x: 0, y: 0 }, { x: 40, y: 0 }, { x: 45, y: 300 }]);
    expect(frames).toHaveLength(2);
  });

  it("ignores travel opposite the configured direction", () => {
    const frames: SwipeGestureFrame[] = [];
    renderWithProviders(<Harness onMove={(f) => frames.push(f)} />);
    drag([{ x: 0 }, { x: -80 }]);
    expect(frames.at(-1)?.distance).toBe(0);
  });
});

describe("staged thresholds", () => {
  it("arms each stage in turn", () => {
    const changes: number[] = [];
    renderWithProviders(<Harness onStageChange={(stage) => changes.push(stage)} />);
    drag([{ x: 0 }, { x: 30 }, { x: 70 }, { x: 150 }]);
    expect(changes).toEqual([1, 2]);
  });

  it("disarms when the finger comes back", () => {
    const changes: number[] = [];
    renderWithProviders(<Harness onStageChange={(stage) => changes.push(stage)} />);
    drag([{ x: 0 }, { x: 150 }, { x: 20 }]);
    expect(changes).toEqual([2, 0]);
  });

  it("reports the armed stage on release", () => {
    renderWithProviders(<Harness />);
    const surface = drag([{ x: 0 }, { x: 150 }]);
    expect(surface.getAttribute("data-stage")).toBe("2");
  });
});

describe("resistance", () => {
  it("moves the surface one to one below the last threshold", () => {
    const frames: SwipeGestureFrame[] = [];
    renderWithProviders(<Harness onMove={(f) => frames.push(f)} />);
    drag([{ x: 0 }, { x: 100 }]);
    expect(frames.at(-1)?.offset).toBe(100);
  });

  it("damps overtravel past the last threshold", () => {
    const frames: SwipeGestureFrame[] = [];
    renderWithProviders(<Harness onMove={(f) => frames.push(f)} resistance={0.5} />);
    drag([{ x: 0 }, { x: 240 }]);
    // 140 of real travel, then half of the remaining 100.
    expect(frames.at(-1)?.offset).toBe(190);
  });

  it("pins the surface at the ceiling when resistance is zero", () => {
    const frames: SwipeGestureFrame[] = [];
    renderWithProviders(<Harness onMove={(f) => frames.push(f)} resistance={0} />);
    drag([{ x: 0 }, { x: 400 }]);
    expect(frames.at(-1)?.offset).toBe(140);
  });
});

describe("direction", () => {
  it("signs the translation leftward for a left gesture", () => {
    const frames: SwipeGestureFrame[] = [];
    renderWithProviders(<Harness direction="left" onMove={(f) => frames.push(f)} />);
    drag([{ x: 0 }, { x: -100 }]);
    expect(frames.at(-1)?.distance).toBe(100);
    expect(frames.at(-1)?.translate).toBe(-100);
  });

  it("signs the translation rightward for a right gesture", () => {
    const frames: SwipeGestureFrame[] = [];
    renderWithProviders(<Harness onMove={(f) => frames.push(f)} />);
    drag([{ x: 0 }, { x: 100 }]);
    expect(frames.at(-1)?.translate).toBe(100);
  });
});

describe("release", () => {
  it("returns when the gesture stops short of every threshold", () => {
    renderWithProviders(<Harness />);
    const surface = drag([{ x: 0, t: 0 }, { x: 30, t: 400 }]);
    expect(surface.getAttribute("data-outcome")).toBe("return");
  });

  it("rests open past a threshold by default", () => {
    renderWithProviders(<Harness />);
    const surface = drag([{ x: 0, t: 0 }, { x: 100, t: 400 }]);
    expect(surface.getAttribute("data-outcome")).toBe("rest");
  });

  it("commits past a threshold when asked to", () => {
    renderWithProviders(<Harness releaseMode="commit" />);
    const surface = drag([{ x: 0, t: 0 }, { x: 100, t: 400 }]);
    expect(surface.getAttribute("data-outcome")).toBe("commit");
  });

  it("accepts a fast flick that never reached a threshold", () => {
    renderWithProviders(<Harness />);
    const surface = drag([{ x: 0, t: 0 }, { x: 40, t: 10 }]);
    expect(surface.getAttribute("data-outcome")).toBe("rest");
  });

  it("measures velocity over the tail, so a pause then a flick still counts", () => {
    renderWithProviders(<Harness />);
    const surface = drag([
      { x: 0, t: 0 },
      { x: 20, t: 900 },
      { x: 50, t: 940 },
    ]);
    expect(surface.getAttribute("data-outcome")).toBe("rest");
  });

  it("does not treat a slow short drag as a flick", () => {
    renderWithProviders(<Harness />);
    const surface = drag([
      { x: 0, t: 0 },
      { x: 20, t: 500 },
      { x: 30, t: 1000 },
    ]);
    expect(surface.getAttribute("data-outcome")).toBe("return");
  });
});

describe("cancellation", () => {
  // Wiring pointercancel to the pointerup path performs the action the user
  // just stopped asking for. This is the regression that shipped once already.
  it("aborts rather than committing when the browser cancels", () => {
    renderWithProviders(<Harness releaseMode="commit" />);
    const surface = drag([{ x: 0, t: 0 }, { x: 300, t: 100 }], "cancel");
    expect(surface.getAttribute("data-outcome")).toBe("abort");
  });

  it("aborts even from past the final threshold", () => {
    renderWithProviders(<Harness />);
    const surface = drag([{ x: 0, t: 0 }, { x: 500, t: 100 }], "cancel");
    expect(surface.getAttribute("data-outcome")).toBe("abort");
  });
});

describe("lifecycle", () => {
  it("ignores a second pointer while one gesture is live", () => {
    const frames: SwipeGestureFrame[] = [];
    renderWithProviders(<Harness onMove={(f) => frames.push(f)} />);
    const surface = screen.getByTestId("surface");
    fireEvent.pointerDown(surface, pointer({ id: 1, x: 0 }));
    fireEvent.pointerDown(surface, pointer({ id: 2, x: 0 }));
    fireEvent.pointerMove(window, pointer({ id: 2, x: 200 }));
    expect(frames).toHaveLength(0);
    fireEvent.pointerMove(window, pointer({ id: 1, x: 90 }));
    expect(frames.at(-1)?.distance).toBe(90);
  });

  it("does not start while disabled", () => {
    const frames: SwipeGestureFrame[] = [];
    renderWithProviders(<Harness disabled onMove={(f) => frames.push(f)} />);
    drag([{ x: 0 }, { x: 200 }]);
    expect(frames).toHaveLength(0);
  });

  it("ignores a non-primary mouse button", () => {
    const frames: SwipeGestureFrame[] = [];
    renderWithProviders(<Harness onMove={(f) => frames.push(f)} />);
    const surface = screen.getByTestId("surface");
    fireEvent.pointerDown(surface, { pointerId: 3, clientX: 0, clientY: 0, button: 2, pointerType: "mouse" });
    fireEvent.pointerMove(window, pointer({ id: 3, x: 200 }));
    expect(frames).toHaveLength(0);
  });

  // A finger that leaves the row must keep driving it; element handlers alone
  // would strand the surface mid-travel.
  it("keeps tracking after the pointer leaves the element", () => {
    const frames: SwipeGestureFrame[] = [];
    renderWithProviders(<Harness onMove={(f) => frames.push(f)} />);
    const surface = screen.getByTestId("surface");
    fireEvent.pointerDown(surface, pointer({ x: 0 }));
    fireEvent.pointerLeave(surface, pointer({ x: 50 }));
    fireEvent.pointerMove(window, pointer({ x: 120 }));
    expect(frames.at(-1)?.distance).toBe(120);
  });

  it("resumes from a rested offset instead of jumping to zero", () => {
    renderWithProviders(<Harness />);
    drag([{ x: 0, t: 0 }, { x: 200, t: 400 }]);
    const frames: SwipeGestureFrame[] = [];
    const surface = screen.getByTestId("surface");
    fireEvent.pointerDown(surface, pointer({ x: 0 }));
    fireEvent.pointerMove(window, pointer({ x: -40 }));
    // Resting at 140, dragged back 40, so 100 of travel remains.
    expect(surface.getAttribute("data-outcome")).toBe("rest");
    fireEvent.pointerUp(window, pointer({ x: -40 }));
    expect(frames).toHaveLength(0);
  });

  it("stops listening once the consumer unmounts mid-gesture", () => {
    const onMove = vi.fn();
    const view = renderWithProviders(<Harness onMove={onMove} />);
    fireEvent.pointerDown(screen.getByTestId("surface"), pointer({ x: 0 }));
    view.unmount();
    fireEvent.pointerMove(window, pointer({ x: 200 }));
    expect(onMove).not.toHaveBeenCalled();
  });
});
