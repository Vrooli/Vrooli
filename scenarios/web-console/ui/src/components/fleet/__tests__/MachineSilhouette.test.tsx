import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import MachineSilhouette from "../MachineSilhouette";
import { MACHINE_ARCHETYPES, MACHINE_GEOMETRY, faceBox } from "../../../lib/machineGeometry";

/**
 * The chassis is drawn from a geometry table, so what is worth testing is that
 * the drawing and the table agree: every detail lands inside the face it
 * belongs to, and the viewBox contains the base drawn outside the panel.
 *
 * Both are the failure the hand-rolled silhouette this replaced actually had —
 * its stand was positioned against an ancestor rather than against the chassis,
 * so it floated wherever the card put it.
 */
function viewBoxOf(container: HTMLElement): { minX: number; minY: number; width: number; height: number } {
  const [minX = NaN, minY = NaN, width = NaN, height = NaN] = (container.querySelector("svg")?.getAttribute("viewBox") ?? "")
    .trim()
    .split(/\s+/)
    .map(Number);
  return { minX, minY, width, height };
}

describe("MachineSilhouette", () => {
  it.each(MACHINE_ARCHETYPES)("keeps every %s detail inside its face", (archetype) => {
    const { container } = render(<MachineSilhouette archetype={archetype} state="dispatchable" />);
    const box = faceBox(MACHINE_GEOMETRY[archetype]);

    const slots = Array.from(container.querySelectorAll("g > rect"));
    expect(slots).toHaveLength(MACHINE_GEOMETRY[archetype].vents);
    for (const slot of slots) {
      const x = Number(slot.getAttribute("x"));
      const y = Number(slot.getAttribute("y"));
      expect(x).toBeGreaterThanOrEqual(box.x);
      expect(y).toBeGreaterThanOrEqual(box.y);
      expect(x + Number(slot.getAttribute("width"))).toBeLessThanOrEqual(box.x + box.width + 0.001);
      expect(y + Number(slot.getAttribute("height"))).toBeLessThanOrEqual(box.y + box.height + 0.001);
    }

    for (const lamp of Array.from(container.querySelectorAll("circle"))) {
      const cx = Number(lamp.getAttribute("cx"));
      const cy = Number(lamp.getAttribute("cy"));
      expect(cx).toBeGreaterThan(box.x);
      expect(cx).toBeLessThan(box.x + box.width);
      expect(cy).toBeGreaterThan(box.y);
      expect(cy).toBeLessThan(box.y + box.height);
    }
  });

  it.each(MACHINE_ARCHETYPES)("gives %s a viewBox that contains its base", (archetype) => {
    const { container } = render(<MachineSilhouette archetype={archetype} state="dispatchable" />);
    const geometry = MACHINE_GEOMETRY[archetype];
    const { minX, minY, width, height } = viewBoxOf(container);

    // Feet hang below the panel; ears sit outside its left and right edges.
    const below = geometry.base === "feet" ? geometry.baseHeight : 0;
    const outside = geometry.base === "ears" ? 7 : 0;
    expect(minX).toBeLessThanOrEqual(-outside);
    expect(minY).toBeLessThan(0);
    expect(minX + width).toBeGreaterThanOrEqual(geometry.width + outside);
    expect(minY + height).toBeGreaterThanOrEqual(geometry.height + below);
  });

  it.each([
    ["dispatchable", 3],
    ["offline", 3],
    ["unenrolled", 2],
  ] as const)("draws the %s lamp", (state, circles) => {
    const { container } = render(<MachineSilhouette state={state} />);
    expect(container.querySelector("[data-testid=machine-silhouette]")).toHaveAttribute("data-state", state);
    // A machine that has never answered drops the halo entirely: a glow would
    // imply a reachability it never demonstrated.
    expect(container.querySelectorAll("circle")).toHaveLength(circles);
    expect(container.querySelectorAll(".wc-machine-lamp-halo")).toHaveLength(state === "dispatchable" ? 1 : 0);
  });

  it("washes the face only while the machine is dispatchable", () => {
    const fills = (state: "dispatchable" | "offline") =>
      Array.from(render(<MachineSilhouette state={state} />).container.querySelectorAll("rect"))
        .map((node) => node.getAttribute("fill"));

    expect(fills("dispatchable")).toContain("var(--wc-machine-face-live)");
    expect(fills("offline")).toContain("var(--wc-device-screen)");
    expect(fills("offline")).not.toContain("var(--wc-machine-face-live)");
  });
});
