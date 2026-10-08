import { describe, it, expect, vi } from "vitest";
import { screen, fireEvent } from "@testing-library/react";
import { renderWithProviders as renderWithCanonicalProviders } from "../../test-utils/renderWithProviders";
import { ScenarioChips } from "./scenario-chips";

describe("ScenarioChips", () => {
  it("renders a chip for each scenario with label", () => {
    renderWithCanonicalProviders(<ScenarioChips scenarios={["alpha", "beta"]} onSelect={vi.fn()} />);
    expect(screen.getByText("Target Scenarios")).toBeInTheDocument();
    expect(screen.getByText("alpha")).toBeInTheDocument();
    expect(screen.getByText("beta")).toBeInTheDocument();
  });

  it("fires onSelect with the correct scenario name on click", () => {
    const onSelect = vi.fn();
    renderWithCanonicalProviders(<ScenarioChips scenarios={["alpha", "beta"]} onSelect={onSelect} />);
    fireEvent.click(screen.getByText("beta"));
    expect(onSelect).toHaveBeenCalledWith("beta");
  });

  it("renders nothing when scenarios is empty", () => {
    const { container } = renderWithCanonicalProviders(<ScenarioChips scenarios={[]} onSelect={vi.fn()} />);
    expect(container.firstChild).toBeNull();
  });
});
