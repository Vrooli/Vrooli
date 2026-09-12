import { renderWithProviders as render } from "../../../test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, screen } from "@testing-library/react";
import BannerRegion from "../BannerRegion";
import type { BannerDescriptor } from "../types";
import { INSTANT_DAMPING } from "../damping";

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

describe("library style integration", () => {
  /**
   * The banner's entire appearance now arrives from the library stylesheet.
   * If that sheet stops mounting — a StyleSheet regression, a bad key, an
   * import that resolves to a stub — every banner renders as unstyled markup
   * while the DOM still looks correct to every other assertion in this file.
   * That failure is invisible to markup tests, so it gets its own.
   */
  it("mounts the library banner stylesheet with token-derived tone palettes", () => {
    render(<BannerRegion banners={[warning("styled")]} damping={INSTANT_DAMPING} />);

    const sheet = document.querySelector('style[data-rcl-sheet^="banner-"]');
    expect(sheet).not.toBeNull();
    const css = sheet?.textContent ?? "";

    // Tone palettes resolve through the semantic colour tokens rather than
    // hard-coded literals, which is what keeps them themeable by the host.
    expect(css).toContain('[data-rcl-banner][data-tone="danger"]');
    expect(css).toContain("var(--color-danger");
    expect(css).toContain("var(--color-warning");
    expect(css).toContain("color-mix(in srgb");

    // The close control is a bare icon button, and its touch target is an
    // overlay so a conforming hit area costs the compact row no height.
    expect(css).toMatch(/\[data-rcl-banner-dismiss\]\s*\{[^}]*border:\s*0/);
    expect(css).toContain("var(--tap-target-min, 44px)");
  });
});
