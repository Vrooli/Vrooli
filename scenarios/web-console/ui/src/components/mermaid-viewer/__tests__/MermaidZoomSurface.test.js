import { jsx as _jsx } from "react/jsx-runtime";
import { describe, expect, it } from "vitest";
import { render } from "@testing-library/react";
import { MermaidZoomSurface } from "../MermaidZoomSurface";
const SVG = '<svg viewBox="0 0 320 240" style="max-width: 320px;"><rect width="10" height="10" /></svg>';
describe("MermaidZoomSurface", () => {
    it("pins the injected SVG to its intrinsic viewBox size and clears max-width", () => {
        const { getByTestId } = render(_jsx(MermaidZoomSurface, { svgHtml: SVG }));
        const svg = getByTestId("mermaid-zoom-surface").querySelector("svg");
        expect(svg).not.toBeNull();
        expect(svg?.style.width).toBe("320px");
        expect(svg?.style.height).toBe("240px");
        expect(svg?.style.maxWidth).toBe("none");
        expect(svg?.style.display).toBe("block");
    });
    it("uses a non-composited 2D transform so vectors stay crisp when scaled", () => {
        const { getByTestId } = render(_jsx(MermaidZoomSurface, { svgHtml: SVG }));
        const wrapper = getByTestId("mermaid-zoom-surface").firstElementChild;
        expect(wrapper.style.transform).toContain("translate(");
        expect(wrapper.style.transform).not.toContain("translate3d");
        // will-change would promote the SVG to a cached raster layer and blur it.
        expect(wrapper.className).not.toContain("will-change");
    });
    it("disables native touch gestures only on the zoom surface", () => {
        const { getByTestId } = render(_jsx(MermaidZoomSurface, { svgHtml: SVG }));
        expect(getByTestId("mermaid-zoom-surface").className).toContain("touch-none");
    });
});
