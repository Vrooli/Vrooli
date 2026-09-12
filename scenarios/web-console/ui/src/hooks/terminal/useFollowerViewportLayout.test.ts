import { describe, expect, it, vi } from "vitest";
import { renderHook } from "@testing-library/react";
import { createRef } from "react";
import { FOLLOWER_ENCLOSURE_Z, FOLLOWER_SCREEN_Z, FOLLOWER_TRANSITION, FOLLOWER_TRANSITION_MS, useFollowerViewportLayout } from "./useFollowerViewportLayout";
import type { FollowerFrame } from "./useFollowerPresentation";

function frameFixture(overrides: Partial<FollowerFrame> = {}): FollowerFrame {
  const rect = { x: 10, y: 20, width: 300, height: 600, fontSize: 12, scale: 1 };
  return {
    rect,
    screenRect: { x: 24, y: 34, width: 260, height: 500, fontSize: 12, scale: 1 },
    apertureRect: { x: 20, y: 30, width: 268, height: 540, radius: 16 },
    tier: "full",
    archetype: "phone",
    cols: 46,
    rows: 26,
    kbOpen: false,
    keyboardShare: 0,
    captionOffset: 8,
    ...overrides,
  };
}

function terminalFixture() {
  return { options: {} as { fontSize?: number }, cols: 80, rows: 24, resize: vi.fn() };
}

function mount(frame: FollowerFrame | null, terminal = terminalFixture()) {
  const host = document.createElement("div");
  const screen = document.createElement("div");
  screen.appendChild(host);
  const hostRef = createRef<HTMLDivElement>() as { current: HTMLDivElement | null };
  const screenRef = createRef<HTMLDivElement>() as { current: HTMLDivElement | null };
  hostRef.current = host;
  screenRef.current = screen;
  const refit = vi.fn();
  const fitToHost = vi.fn();
  const view = renderHook(({ frame: current }) => useFollowerViewportLayout({
    frame: current, terminal, hostRef, screenRef, paneFontSize: 15, refit, fitToHost,
  }), { initialProps: { frame } });
  return { host, screen, terminal, refit, fitToHost, view };
}

describe("FOLLOWER_TRANSITION", () => {
  // The frame and the terminal inside it previously carried separate literals
  // that had already drifted: one animated `transform`, the other did not, so
  // a scaled follower's bezel and contents slid apart.
  it("is one declaration covering every property that moves", () => {
    for (const property of ["left", "top", "width", "height", "transform"]) {
      expect(FOLLOWER_TRANSITION).toContain(`${property} ${String(FOLLOWER_TRANSITION_MS)}ms ease`);
    }
  });
});

describe("useFollowerViewportLayout", () => {
  it("places the terminal in the device's screen aperture", () => {
    const { host, terminal, fitToHost } = mount(frameFixture());
    expect(host.style.position).toBe("absolute");
    // Offsets are aperture-relative: the grid sits 4px inside the screen box.
    expect(host.style.left).toBe("4px");
    expect(host.style.top).toBe("4px");
    expect(host.style.width).toBe("260px");
    expect(host.style.height).toBe("500px");
    expect(host.style.transition).toBe(FOLLOWER_TRANSITION);
    expect(terminal.options.fontSize).toBe(12);
    expect(fitToHost).toHaveBeenCalled();
  });

  it("mirrors the leader's grid exactly", () => {
    const { terminal } = mount(frameFixture({ cols: 46, rows: 13 }));
    expect(terminal.resize).toHaveBeenCalledWith(46, 13);
  });

  it("does not resize when the grid already matches", () => {
    const terminal = terminalFixture();
    terminal.cols = 46;
    terminal.rows = 26;
    mount(frameFixture(), terminal);
    expect(terminal.resize).not.toHaveBeenCalled();
  });

  it("scales the surface rather than rendering an illegible font", () => {
    const { host } = mount(frameFixture({
      screenRect: { x: 24, y: 34, width: 130, height: 250, fontSize: 9, scale: 0.5 },
    }));
    expect(host.style.transform).toBe("scale(0.5)");
    expect(host.style.transformOrigin).toBe("top left");
    // The host is laid out at full size and scaled down, so the grid keeps its
    // cell count rather than losing columns.
    expect(host.style.width).toBe("260px");
  });

  it("clips the terminal to the screen opening so it cannot overhang the bezel", () => {
    const { screen } = mount(frameFixture());
    expect(screen.style.position).toBe("absolute");
    expect(screen.style.left).toBe("20px");
    expect(screen.style.top).toBe("30px");
    expect(screen.style.width).toBe("268px");
    expect(screen.style.height).toBe("540px");
    expect(screen.style.overflow).toBe("hidden");
    // A square terminal inside a rounded screen must be clipped by the curve.
    expect(screen.style.borderRadius).toBe("16px");
  });

  it("lifts the terminal above the opaque enclosure drawn behind it", () => {
    const { screen } = mount(frameFixture());
    expect(Number(screen.style.zIndex)).toBe(FOLLOWER_SCREEN_Z);
    expect(FOLLOWER_SCREEN_Z).toBeGreaterThan(FOLLOWER_ENCLOSURE_Z);
  });

  it("restores the pane when this viewer stops following", () => {
    const { host, screen, terminal, refit, view } = mount(frameFixture());
    view.rerender({ frame: null });
    expect(screen.style.position).toBe("");
    expect(screen.style.overflow).toBe("");
    expect(screen.style.borderRadius).toBe("");
    expect(screen.style.zIndex).toBe("");
    expect(host.style.position).toBe("");
    expect(host.style.left).toBe("");
    expect(host.style.transform).toBe("");
    expect(host.style.transition).toBe("");
    expect(terminal.options.fontSize).toBe(15);
    expect(refit).toHaveBeenCalled();
  });
});
