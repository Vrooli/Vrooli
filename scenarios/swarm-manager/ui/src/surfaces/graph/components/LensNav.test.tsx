import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { LensNav } from "./LensNav";

describe("LensNav", () => {
  it("renders Plan, Graph, and Stats as peer surfaces", () => {
    render(<LensNav activeLens="stats" onLensChange={vi.fn()} />);

    expect(screen.getByTestId("lens-nav").className).not.toContain("border");
    expect(screen.getByTestId("lens-stats").className).toContain("after:bg-cyan-400");
    expect(screen.getByTestId("lens-plan")).toBeInTheDocument();
    expect(screen.getByTestId("lens-graph")).toBeInTheDocument();
    expect(screen.getByTestId("lens-stats")).toBeInTheDocument();
    expect(screen.getByTestId("lens-stats")).toHaveAttribute("aria-selected", "true");
  });

  it("maps graph data lenses to the Graph surface", () => {
    render(<LensNav activeLens="focus" onLensChange={vi.fn()} />);

    expect(screen.getByTestId("lens-graph")).toHaveAttribute("aria-selected", "true");
  });

  it("emits the selected surface", () => {
    const onLensChange = vi.fn();
    render(<LensNav activeLens="plan" onLensChange={onLensChange} />);

    fireEvent.click(screen.getByTestId("lens-stats"));

    expect(onLensChange).toHaveBeenCalledWith("stats");
  });

  it("renders a badge on the requested lens", () => {
    render(<LensNav activeLens="graph" onLensChange={vi.fn()} badges={{ plan: 3 }} />);

    expect(screen.getByTestId("lens-plan-badge")).toHaveTextContent("3");
    expect(screen.queryByTestId("lens-graph-badge")).toBeNull();
  });
});
