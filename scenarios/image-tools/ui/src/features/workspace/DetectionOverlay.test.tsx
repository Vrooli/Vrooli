/**
 * DetectionOverlay unit test. Verifies the read-only boxes render with their
 * labels and map through the shared crop geometry (a 2× client box doubles the
 * pixel offsets). The overlay is decorative (aria-hidden) and never
 * interactive, so the canvas keeps its pan/zoom/focus.
 */
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";

import { selectors } from "../../consts/selectors";
import { DetectionOverlay, type DetectionBox } from "./DetectionOverlay";

afterEach(cleanup);

const BOXES: DetectionBox[] = [
  { id: "ocr-0", label: "Hello", box: { x: 10, y: 20, width: 30, height: 15 } },
];

describe("DetectionOverlay", () => {
  it("renders a labeled box and maps pixel coords to the displayed box", () => {
    render(
      <DetectionOverlay
        natural={{ width: 100, height: 100 }}
        client={{ width: 200, height: 200 }}
        boxes={BOXES}
      />,
    );

    const overlay = screen.getByTestId(selectors.workspace.analyze.overlay);
    expect(overlay).toHaveAttribute("aria-hidden", "true");
    // The block label ("Hello") is test data, not i18n copy — assert via the
    // overlay's text content rather than a copy-driven query.
    expect(overlay.textContent).toContain("Hello");

    // 100→200 is a 2× contain scale with no letterbox offset: x 10 → 20px.
    const box = overlay.querySelector("div");
    expect(box?.getAttribute("style")).toContain("left: 20px");
    expect(box?.getAttribute("style")).toContain("width: 60px");
  });
});
