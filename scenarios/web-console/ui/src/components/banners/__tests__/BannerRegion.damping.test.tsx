import { renderWithProviders as render } from "../../../test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, screen } from "@testing-library/react";
import BannerRegion from "../BannerRegion";
import type { BannerDescriptor } from "../types";

/**
 * Proves the region is actually wired to the damping policy, and that it holds
 * exactly one timer to do it. The policy's own behaviour is covered at
 * millisecond resolution in `damping.test.ts`; this suite is about the seam.
 */

function warning(id: string): BannerDescriptor {
  return { id, testId: `${id}-banner`, tone: "warning", priority: 50, title: id };
}

function renderRegion(banners: BannerDescriptor[]) {
  return render(<BannerRegion banners={banners} />);
}

describe("BannerRegion damping wiring", () => {
  beforeEach(() => { vi.useFakeTimers(); });
  afterEach(() => { vi.useRealTimers(); });

  it("does not paint a warning until it outlasts the enter delay", () => {
    const { rerender } = renderRegion([warning("noisy")]);
    expect(screen.queryByTestId("noisy-banner")).toBeNull();

    act(() => { vi.advanceTimersByTime(300); });
    rerender(<BannerRegion banners={[warning("noisy")]} />);
    expect(screen.getByTestId("noisy-banner")).toBeTruthy();
  });

  it("never paints a condition that clears inside the enter delay", () => {
    const { rerender } = renderRegion([warning("blip")]);
    act(() => { vi.advanceTimersByTime(100); });
    rerender(<BannerRegion banners={[]} />);
    act(() => { vi.advanceTimersByTime(2_000); });
    expect(screen.queryByTestId("blip-banner")).toBeNull();
  });

  it("holds a painted banner across a brief clear, marking it inert meanwhile", () => {
    const { rerender } = renderRegion([warning("held")]);
    act(() => { vi.advanceTimersByTime(300); });
    rerender(<BannerRegion banners={[warning("held")]} />);
    expect(screen.getByTestId("held-banner")).toBeTruthy();

    // Condition clears; the banner stays on screen and stops accepting input.
    rerender(<BannerRegion banners={[]} />);
    act(() => { vi.advanceTimersByTime(50); });
    const held = screen.getByTestId("held-banner");
    expect(held).toHaveAttribute("data-settling");

    // It comes back before the hold expires — no remove/add pair was ever shown.
    rerender(<BannerRegion banners={[warning("held")]} />);
    act(() => { vi.advanceTimersByTime(50); });
    expect(screen.getByTestId("held-banner")).not.toHaveAttribute("data-settling");
  });

  it("removes the banner once the condition stays cleared", () => {
    const { rerender } = renderRegion([warning("gone")]);
    act(() => { vi.advanceTimersByTime(300); });
    rerender(<BannerRegion banners={[warning("gone")]} />);
    expect(screen.getByTestId("gone-banner")).toBeTruthy();

    rerender(<BannerRegion banners={[]} />);
    act(() => { vi.advanceTimersByTime(5_000); });
    expect(screen.queryByTestId("gone-banner")).toBeNull();
  });

  it("schedules one timer for the whole region, and none at rest", () => {
    const { rerender, unmount } = render(
      <BannerRegion banners={[warning("a"), warning("b"), warning("c")]} />,
    );
    // Three pending banners share a single wake-up.
    expect(vi.getTimerCount()).toBe(1);

    act(() => { vi.advanceTimersByTime(300); });
    rerender(<BannerRegion banners={[warning("a"), warning("b"), warning("c")]} />);
    // All painted, nothing pending, primary dwell is the only outstanding
    // transition — still one timer, never one per banner.
    expect(vi.getTimerCount()).toBeLessThanOrEqual(1);

    unmount();
    expect(vi.getTimerCount()).toBe(0);
  });
});
