import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Activity, History, Network } from "lucide-react";
import { LensBar } from "./LensBar";
import type { LensOption } from "./lens-options";

const testLenses: LensOption[] = [
  { lens: "topology", label: "View Topology", icon: Network, iconColorClass: "text-indigo-400" },
  { lens: "flow", label: "View History", icon: History, iconColorClass: "text-cyan-400" },
  { lens: "operations", label: "View Operations", icon: Activity, iconColorClass: "text-amber-400" },
];

describe("LensBar", () => {
  it("renders correct number of buttons", () => {
    render(<LensBar nodeId="node-1" lenses={testLenses} onDrillToLens={vi.fn()} />);

    expect(screen.getByTestId("lens-bar")).toBeInTheDocument();
    expect(screen.getByTestId("lens-bar-topology")).toBeInTheDocument();
    expect(screen.getByTestId("lens-bar-flow")).toBeInTheDocument();
    expect(screen.getByTestId("lens-bar-operations")).toBeInTheDocument();
  });

  it("renders nothing when lenses array is empty", () => {
    const { container } = render(<LensBar nodeId="node-1" lenses={[]} onDrillToLens={vi.fn()} />);

    expect(container.innerHTML).toBe("");
  });

  it("calls onDrillToLens with correct nodeId and lens on click", async () => {
    const onDrill = vi.fn();
    const user = userEvent.setup();
    render(<LensBar nodeId="node-42" lenses={testLenses} onDrillToLens={onDrill} />);

    await user.click(screen.getByTestId("lens-bar-topology"));
    expect(onDrill).toHaveBeenCalledWith("node-42", "topology");

    await user.click(screen.getByTestId("lens-bar-flow"));
    expect(onDrill).toHaveBeenCalledWith("node-42", "flow");

    await user.click(screen.getByTestId("lens-bar-operations"));
    expect(onDrill).toHaveBeenCalledWith("node-42", "operations");

    expect(onDrill).toHaveBeenCalledTimes(3);
  });

  it("renders button labels", () => {
    render(<LensBar nodeId="node-1" lenses={testLenses} onDrillToLens={vi.fn()} />);

    expect(screen.getByText("View Topology")).toBeInTheDocument();
    expect(screen.getByText("View History")).toBeInTheDocument();
    expect(screen.getByText("View Operations")).toBeInTheDocument();
  });

  it("renders subset of lenses", () => {
    const subset = testLenses.slice(0, 2);
    render(<LensBar nodeId="node-1" lenses={subset} onDrillToLens={vi.fn()} />);

    expect(screen.getByTestId("lens-bar-topology")).toBeInTheDocument();
    expect(screen.getByTestId("lens-bar-flow")).toBeInTheDocument();
    expect(screen.queryByTestId("lens-bar-operations")).not.toBeInTheDocument();
  });
});
