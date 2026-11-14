import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { hasLocalFileReference, renderDecisionText } from "./DecisionText";

describe("decision text evidence", () => {
  it("renders embedded image evidence inside the decision text", () => {
    const source = "data:image/png;base64,iVBORw0KGgo=";
    render(<div>{renderDecisionText(`Before: ![History page without navigation](${source})`)}</div>);

    const image = screen.getByRole("img", { name: "History page without navigation" });
    expect(image).toHaveAttribute("src", source);
    expect(screen.getByText("Before:")).toBeInTheDocument();
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });

  it("keeps image links restricted to embedded raster data and preserves file-link detection", () => {
    const svg = "![Unsafe](data:image/svg+xml;base64,PHN2Zz4=)";
    const parts = renderDecisionText(svg);
    expect(parts.some((part) => typeof part !== "string" && Boolean(part))).toBe(false);
    expect(hasLocalFileReference("See scenarios/notification-hub/README.md")).toBe(true);
    expect(hasLocalFileReference("![Shot](data:image/png;base64,iVBORw0KGgo=)")).toBe(false);
  });
});
