import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Activity, Network } from "lucide-react";
import { LensBar } from "./LensBar";
import type { LensOption } from "./lens-options";

const testLenses: LensOption[] = [
  { lens: "plan", label: "View Plan", icon: Network, iconColorClass: "text-indigo-400" },
  { lens: "focus", label: "View Focus", icon: Activity, iconColorClass: "text-amber-400" },
];

describe("LensBar", () => {
  it("renders correct number of buttons", () => {
    render(<LensBar nodeId="node-1" lenses={testLenses} onDrillToLens={vi.fn()} />);

    expect(screen.getByTestId("lens-bar")).toBeInTheDocument();
    expect(screen.getByTestId("lens-bar-plan")).toBeInTheDocument();
    expect(screen.getByTestId("lens-bar-focus")).toBeInTheDocument();
  });

  it("renders nothing when lenses array is empty", () => {
    const { container } = render(<LensBar nodeId="node-1" lenses={[]} onDrillToLens={vi.fn()} />);

    expect(container.innerHTML).toBe("");
  });

  it("calls onDrillToLens with correct nodeId and lens on click", async () => {
    const onDrill = vi.fn();
    const user = userEvent.setup();
    render(<LensBar nodeId="node-42" lenses={testLenses} onDrillToLens={onDrill} />);

    await user.click(screen.getByTestId("lens-bar-plan"));
    expect(onDrill).toHaveBeenCalledWith("node-42", "plan");

    await user.click(screen.getByTestId("lens-bar-focus"));
    expect(onDrill).toHaveBeenCalledWith("node-42", "focus");

    expect(onDrill).toHaveBeenCalledTimes(2);
  });

  it("renders button labels", () => {
    render(<LensBar nodeId="node-1" lenses={testLenses} onDrillToLens={vi.fn()} />);

    expect(screen.getByText("View Plan")).toBeInTheDocument();
    expect(screen.getByText("View Focus")).toBeInTheDocument();
  });

  it("renders subset of lenses", () => {
    const subset = testLenses.slice(0, 1);
    render(<LensBar nodeId="node-1" lenses={subset} onDrillToLens={vi.fn()} />);

    expect(screen.getByTestId("lens-bar-plan")).toBeInTheDocument();
    expect(screen.queryByTestId("lens-bar-focus")).not.toBeInTheDocument();
  });
});
