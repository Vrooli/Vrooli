/**
 * LumeMark tests — the SVG product mark. The only branching is the optional
 * `title`, which flips the mark between an accessible `role="img"` (with a
 * `<title>`) and a decorative `aria-hidden` glyph, plus the `size` default.
 */
import { describe, expect, it } from "vitest";
import { render } from "@testing-library/react";

import { LumeMark } from "./lume-mark";

describe("LumeMark", () => {
  it("is decorative (aria-hidden, no role/title) by default", () => {
    const { container } = render(<LumeMark />);
    const svg = container.querySelector("svg");
    expect(svg).not.toBeNull();
    expect(svg).toHaveAttribute("aria-hidden", "true");
    expect(svg).not.toHaveAttribute("role");
    expect(svg?.querySelector("title")).toBeNull();
  });

  it("uses the default size of 24 when none is given", () => {
    const { container } = render(<LumeMark />);
    const svg = container.querySelector("svg");
    expect(svg).toHaveAttribute("width", "24");
    expect(svg).toHaveAttribute("height", "24");
  });

  it("honours a custom size on both axes", () => {
    const { container } = render(<LumeMark size={48} />);
    const svg = container.querySelector("svg");
    expect(svg).toHaveAttribute("width", "48");
    expect(svg).toHaveAttribute("height", "48");
  });

  it("becomes an accessible image with a <title> when titled", () => {
    const { container } = render(<LumeMark title="Lume" />);
    const svg = container.querySelector("svg");
    expect(svg).toHaveAttribute("role", "img");
    expect(svg).not.toHaveAttribute("aria-hidden");
    expect(svg?.querySelector("title")?.textContent).toBe("Lume");
  });

  it("merges a caller className with the base shrink-0 class", () => {
    const { container } = render(<LumeMark className="extra-class" />);
    const svg = container.querySelector("svg");
    expect(svg).toHaveClass("shrink-0");
    expect(svg).toHaveClass("extra-class");
  });
});
