import { useState } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, screen } from "@testing-library/react";

import { renderWithProviders } from "../../../../../ui/src/test-utils";
import { MorphingIcon } from "./MorphingIcon";
import { clearIconMorphCache } from "../../../../hooks/useIconMorph/versions/1.0.0/useIconMorph";
import { geometryFromElement } from "../../../../foundations/IconGeometry/versions/1.0.0/IconGeometry";

function BubbleIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
      <path d="M13 8H7" />
      <path d="M17 12H7" />
    </svg>
  );
}

function TerminalIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="m7 11 2-2-2-2" />
      <path d="M11 13h4" />
      <rect width="18" height="18" x="3" y="3" rx="2" ry="2" />
    </svg>
  );
}

const root = () => screen.getByTestId("motion.morphing-icon");

beforeEach(() => {
  clearIconMorphCache();
});

describe("rendering arbitrary children", () => {
  it("renders the child icon untouched while idle", () => {
    renderWithProviders(
      <MorphingIcon>
        <BubbleIcon />
      </MorphingIcon>,
    );
    const svg = root().querySelector("[data-rcl-morphing-icon-current] svg");
    expect(svg).not.toBeNull();
    expect(svg!.querySelectorAll("path")).toHaveLength(3);
    // No synthesised frame while nothing is transitioning.
    expect(root().querySelector("[data-rcl-morphing-icon-frame]")).toBeNull();
  });

  it("is hidden from assistive technology unless it is labelled", () => {
    const { rerender } = renderWithProviders(
      <MorphingIcon>
        <BubbleIcon />
      </MorphingIcon>,
    );
    expect(root()).toHaveAttribute("aria-hidden", "true");
    expect(root()).not.toHaveAttribute("role");
    rerender(
      <MorphingIcon label="Messages">
        <BubbleIcon />
      </MorphingIcon>,
    );
    expect(root()).toHaveAttribute("role", "img");
    expect(root()).toHaveAccessibleName("Messages");
  });

  it("sizes from the token scale and from an explicit pixel value", () => {
    const { rerender } = renderWithProviders(
      <MorphingIcon size="lg">
        <BubbleIcon />
      </MorphingIcon>,
    );
    expect(root().style.inlineSize).toBe("var(--icon-size-lg, 1.5rem)");
    rerender(
      <MorphingIcon size={32}>
        <BubbleIcon />
      </MorphingIcon>,
    );
    expect(root().style.inlineSize).toBe("32px");
  });

  it("clamps an out-of-range pixel size", () => {
    const { rerender } = renderWithProviders(
      <MorphingIcon size={4}>
        <BubbleIcon />
      </MorphingIcon>,
    );
    expect(root().style.inlineSize).toBe("12px");
    rerender(
      <MorphingIcon size={999}>
        <BubbleIcon />
      </MorphingIcon>,
    );
    expect(root().style.inlineSize).toBe("64px");
  });
});

describe("the registry path", () => {
  it("still renders a registry glyph", () => {
    renderWithProviders(<MorphingIcon icon="close" />);
    const path = root().querySelector("svg path");
    expect(path).not.toBeNull();
    expect(path!.getAttribute("d")).toBe("M6 6l12 12M18 6L6 18");
  });

  /**
   * The 2.x parser understood no curve commands, so the two arcs that draw the
   * magnifying glass were dropped and the surviving one-point subpath was
   * discarded. `icon="search"` rendered as a bare diagonal line.
   */
  it("recovers the search glyph's circle", () => {
    renderWithProviders(<MorphingIcon icon="search" />);
    const geometry = geometryFromElement(root().querySelector("svg"));
    expect(geometry).not.toBeNull();
    expect(geometry!.subpaths).toHaveLength(2);
    const xs = geometry!.subpaths[0]!.points.map((point) => point.x);
    // The circle spans 14 units; the handle alone would span 4.
    expect(Math.max(...xs) - Math.min(...xs)).toBeGreaterThan(13);
  });

  it("renders the copy glyph, which is not in the shared registry", () => {
    renderWithProviders(<MorphingIcon icon="copy" />);
    expect(root().querySelector("svg path")!.getAttribute("d")).toBe("M9 9h10v10H9zM5 15H4V4h11v1");
  });
});

describe("transitions", () => {
  let now = 0;
  let frames: FrameRequestCallback[] = [];

  beforeEach(() => {
    now = 0;
    frames = [];
    vi.spyOn(performance, "now").mockImplementation(() => now);
    vi.stubGlobal("requestAnimationFrame", (cb: FrameRequestCallback) => {
      frames.push(cb);
      return frames.length;
    });
    vi.stubGlobal("cancelAnimationFrame", () => {});
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  const advance = (ms: number) => {
    now += ms;
    const pending = frames;
    frames = [];
    act(() => {
      for (const frame of pending) frame(now);
    });
  };

  function Swapper({ morph }: { morph?: "auto" | "morph" | "crossfade" | "none" }) {
    const [on, setOn] = useState(false);
    return (
      <>
        <button
          type="button"
          onClick={() => {
            setOn((v) => !v);
          }}
        >
          swap
        </button>
        <MorphingIcon morph={morph}>{on ? <TerminalIcon /> : <BubbleIcon />}</MorphingIcon>
      </>
    );
  }

  const swap = () =>
    act(() => {
      fireEvent.click(screen.getByText("swap"));
    });

  it("detects a swap from the child component's identity alone", () => {
    renderWithProviders(<Swapper />);
    expect(root()).toHaveAttribute("data-rcl-technique", "idle");
    swap();
    expect(root()).toHaveAttribute("data-rcl-technique", "morph");
  });

  it("paints interpolated geometry that changes across frames", () => {
    renderWithProviders(<Swapper />);
    swap();
    const at = () => root().querySelector("[data-rcl-morphing-icon-frame] path")!.getAttribute("d");
    const first = at();
    advance(80);
    const second = at();
    advance(80);
    const third = at();
    expect(first).not.toBe(second);
    expect(second).not.toBe(third);
  });

  it("ends on the incoming icon with no synthesised frame left behind", () => {
    renderWithProviders(<Swapper />);
    swap();
    advance(500);
    expect(root()).toHaveAttribute("data-rcl-technique", "idle");
    expect(root().querySelector("[data-rcl-morphing-icon-frame]")).toBeNull();
    // A rect is unique to the terminal icon, so this proves the swap landed.
    expect(root().querySelector("[data-rcl-morphing-icon-current] rect")).not.toBeNull();
  });

  it("shows both icons at once during a crossfade", () => {
    renderWithProviders(<Swapper morph="crossfade" />);
    swap();
    expect(root().querySelector("[data-rcl-morphing-icon-current]")).not.toBeNull();
    expect(root().querySelector("[data-rcl-morphing-icon-previous]")).not.toBeNull();
    advance(500);
    expect(root().querySelector("[data-rcl-morphing-icon-previous]")).toBeNull();
  });

  it("crossfades opacity in opposite directions", () => {
    renderWithProviders(<Swapper morph="crossfade" />);
    swap();
    advance(160);
    const incoming = Number(
      (root().querySelector("[data-rcl-morphing-icon-current]") as HTMLElement).style.opacity,
    );
    const outgoing = Number(
      (root().querySelector("[data-rcl-morphing-icon-previous]") as HTMLElement).style.opacity,
    );
    expect(incoming).toBeGreaterThan(0);
    expect(incoming).toBeLessThan(1);
    expect(outgoing).toBeCloseTo(1 - incoming, 6);
  });

  it.each([
    // `transform` was never a distinct rendering path in 2.x, only a data
    // attribute, so it resolves to the crossfade it actually performed.
    ["transform", "crossfade"],
    ["crossfade", "crossfade"],
    ["morph", "auto"],
  ] as const)("maps the 2.x strategy=%s onto mode=%s", (strategy, mode) => {
    renderWithProviders(
      <MorphingIcon strategy={strategy}>
        <BubbleIcon />
      </MorphingIcon>,
    );
    expect(root()).toHaveAttribute("data-rcl-transition-mode", mode);
  });

  it("lets an explicit morph prop win over a legacy strategy", () => {
    renderWithProviders(
      <MorphingIcon strategy="transform" morph="morph">
        <BubbleIcon />
      </MorphingIcon>,
    );
    expect(root()).toHaveAttribute("data-rcl-transition-mode", "morph");
  });

  it("swaps instantly under reduced motion", () => {
    const original = window.matchMedia;
    Object.defineProperty(window, "matchMedia", {
      configurable: true,
      writable: true,
      value: (query: string) => ({
        matches: query.includes("reduce"),
        media: query,
        onchange: null,
        addEventListener: () => undefined,
        removeEventListener: () => undefined,
        addListener: () => undefined,
        removeListener: () => undefined,
        dispatchEvent: () => false,
      }),
    });
    try {
      renderWithProviders(<Swapper />);
      swap();
      expect(root()).toHaveAttribute("data-rcl-technique", "idle");
      expect(root().querySelector("[data-rcl-morphing-icon-frame]")).toBeNull();
    } finally {
      Object.defineProperty(window, "matchMedia", {
        configurable: true,
        writable: true,
        value: original,
      });
    }
  });
});
