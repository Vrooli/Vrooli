import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ScenarioChips } from "./scenario-chips";

describe("ScenarioChips", () => {
  it("renders a chip for each scenario", () => {
    render(<ScenarioChips scenarios={["alpha", "beta"]} onSelect={vi.fn()} />);
    expect(screen.getByText("alpha")).toBeInTheDocument();
    expect(screen.getByText("beta")).toBeInTheDocument();
  });

  it("fires onSelect with the correct scenario name on click", () => {
    const onSelect = vi.fn();
    render(<ScenarioChips scenarios={["alpha", "beta"]} onSelect={onSelect} />);
    fireEvent.click(screen.getByText("beta"));
    expect(onSelect).toHaveBeenCalledWith("beta");
  });

  it("renders nothing when scenarios is empty", () => {
    const { container } = render(<ScenarioChips scenarios={[]} onSelect={vi.fn()} />);
    expect(container.firstChild).toBeNull();
  });
});
