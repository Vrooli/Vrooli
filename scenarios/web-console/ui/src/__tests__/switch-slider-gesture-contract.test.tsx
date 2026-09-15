import { describe, expect, it } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Switch } from "@vrooli/react-component-library/Switch";
import { Slider } from "@vrooli/react-component-library/Slider";

/**
 * These two controls own a horizontal drag. The rules below are the difference
 * between a gesture that completes and one the browser reclaims mid-flight, and
 * neither is expressible as a story assertion — a story contract can check
 * attributes and roles, not computed style. So they are pinned here.
 */
function injectedCss(): string {
  return Array.from(document.querySelectorAll("style"))
    .map((el) => el.textContent ?? "")
    .join("\n");
}

describe("switch and slider gesture contract", () => {
  it("gives the switch track sole ownership of a horizontal drag", () => {
    render(<Switch aria-label="Adaptive chrome" defaultChecked />);
    const css = injectedCss();

    // `pan-y` only asks the browser to *prefer* vertical panning; on a target
    // this small it takes every drag that drifts off-axis, and a drag ending
    // outside the switch gets cancelled rather than committed.
    expect(css).toMatch(/\[data-kind="switch"\][^{]*\[data-rcl-selection-indicator\][^{]*\{[^}]*touch-action:\s*none/s);
    expect(css).not.toMatch(/\[data-kind="switch"\][^{]*\[data-rcl-selection-indicator\][^{]*\{[^}]*touch-action:\s*pan-y/s);
  });

  it("renders no tick on a switch", () => {
    render(<Switch aria-label="Adaptive chrome" defaultChecked />);
    // The checked-state mark is the checkbox's. On a track it reads as a defect.
    expect(injectedCss()).toMatch(
      /\[data-kind="switch"\]\s*\[data-rcl-selection-indicator\]::after\s*\{\s*content:\s*none/,
    );
  });

  it("gives the slider sole ownership of a horizontal drag", () => {
    render(<Slider aria-label="Volume" defaultValue={40} />);
    // The area owns the gesture, not the native input: the input's own drag
    // only engages on the native thumb, a moving ~20px target, so grabbing the
    // thumb worked only some of the time.
    const css = injectedCss();
    expect(css).toMatch(/\[data-rcl-slider-area\]\s*\{[^}]*touch-action:\s*none/s);
    expect(css).toMatch(/\[data-rcl-slider-input\]\s*\{[^}]*pointer-events:\s*none/s);
  });

  it("centres the switch thumb on the axis rather than by a computed offset", () => {
    render(<Switch aria-label="Adaptive chrome" defaultChecked />);
    const css = injectedCss();
    // Positioning it by an inset requires knowing whether the declared track
    // height is a content or a border box; the host's reset decides that, and
    // under border-box the thumb sat low.
    expect(css).toMatch(
      /\[data-kind="switch"\]\s*\[data-rcl-selection-indicator\]::before\s*\{[^}]*inset-block-start:\s*50%/s,
    );
    expect(css).toMatch(
      /\[data-kind="switch"\]\s*\[data-rcl-selection-indicator\]\s*\{[^}]*box-sizing:\s*border-box/s,
    );
  });

  it("keeps the switch operable by tap after the drag handlers are attached", () => {
    render(<Switch aria-label="Adaptive chrome" />);
    const control = screen.getByRole("switch");
    expect(control).toHaveAttribute("aria-checked", "false");
    fireEvent.click(control);
    expect(control).toHaveAttribute("aria-checked", "true");
  });
});
