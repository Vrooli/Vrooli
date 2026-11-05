import { describe, it, expect } from "vitest";
import { screen, fireEvent } from "@testing-library/react";
import { renderWithProviders as renderWithCanonicalProviders } from "../../../test-utils/renderWithProviders";
import { EdgeLegend } from "./EdgeLegend";

describe("EdgeLegend", () => {
  it("renders 4 edge type entries", () => {
    renderWithCanonicalProviders(<EdgeLegend edgeTypes={["depends_on", "member_of", "classified_as", "targets"]} />);
    const items = screen.getByTestId("edge-legend-items");
    expect(items.children).toHaveLength(4);
  });

  it("collapses and expands", () => {
    renderWithCanonicalProviders(<EdgeLegend edgeTypes={["depends_on", "member_of"]} />);
    // Initially expanded
    expect(screen.getByTestId("edge-legend-items")).toBeInTheDocument();

    // Collapse
    fireEvent.click(screen.getByTestId("edge-legend-toggle"));
    expect(screen.queryByTestId("edge-legend-items")).not.toBeInTheDocument();

    // Expand
    fireEvent.click(screen.getByTestId("edge-legend-toggle"));
    expect(screen.getByTestId("edge-legend-items")).toBeInTheDocument();
  });

  it("shows correct edge type labels", () => {
    renderWithCanonicalProviders(<EdgeLegend edgeTypes={["depends_on", "member_of", "classified_as", "targets"]} />);
    expect(screen.getByText("Depends on")).toBeInTheDocument();
    expect(screen.getByText("Member of")).toBeInTheDocument();
    expect(screen.getByText("Classified as")).toBeInTheDocument();
    expect(screen.getByText("Targets")).toBeInTheDocument();
  });

  it("renders nothing when no known edge types are visible", () => {
    const { container } = renderWithCanonicalProviders(<EdgeLegend edgeTypes={["unknown"]} />);
    expect(container).toBeEmptyDOMElement();
  });
});
