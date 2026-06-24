import { describe, it, expect, beforeEach, vi } from "vitest";
import { useState } from "react";
import { render, screen, fireEvent } from "@testing-library/react";
import HeaderColorPicker from "./HeaderColorPicker";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";

beforeEach(() => {
  useWorkspaceStore.setState({ recentHeaderColors: [] });
});

/** Stateful harness so picks advance `currentColor` like the real parent. */
function Harness({ initial = "transparent", onChange }: { initial?: string; onChange?: (c: string) => void }) {
  const [color, setColor] = useState(initial);
  return (
    <HeaderColorPicker
      currentColor={color}
      onSelectColor={(c) => {
        setColor(c);
        onChange?.(c);
      }}
    />
  );
}

describe("HeaderColorPicker recent row", () => {
  it("does not render the recent row when there are no recents", () => {
    render(<HeaderColorPicker currentColor="transparent" onSelectColor={vi.fn()} />);
    expect(screen.queryByTestId("appearance-recent-row")).not.toBeInTheDocument();
  });

  it("renders recents and records a color on pick", () => {
    useWorkspaceStore.setState({ recentHeaderColors: ["#abcdef"] });
    const onChange = vi.fn();
    render(<Harness onChange={onChange} />);
    expect(screen.getByTestId("appearance-recent-row")).toBeInTheDocument();
    expect(screen.getByTestId("appearance-header-recent-#abcdef")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("appearance-header-color-#ff6b6b"));
    expect(onChange).toHaveBeenCalledWith("#ff6b6b");
    expect(useWorkspaceStore.getState().recentHeaderColors[0]).toBe("#ff6b6b");
  });
});

describe("HeaderColorPicker two-color UX", () => {
  it("serializes a single color when secondary is closed", () => {
    const onChange = vi.fn();
    render(<Harness onChange={onChange} />);
    fireEvent.click(screen.getByTestId("appearance-header-color-#4dabf7"));
    expect(onChange).toHaveBeenLastCalledWith("#4dabf7");
  });

  it("adds a secondary color and serializes a pair, capped at 2", () => {
    const onChange = vi.fn();
    render(<Harness onChange={onChange} />);

    // Pick a primary color first.
    fireEvent.click(screen.getByTestId("appearance-header-color-#4dabf7"));
    expect(onChange).toHaveBeenLastCalledWith("#4dabf7");

    // Open the secondary slot, then pick a second color.
    fireEvent.click(screen.getByTestId("appearance-add-secondary"));
    fireEvent.click(screen.getByTestId("appearance-header-color-#ff6b6b"));
    expect(onChange).toHaveBeenLastCalledWith("#4dabf7|#ff6b6b");

    // No "add secondary" affordance remains (cap of 2 reached).
    expect(screen.queryByTestId("appearance-add-secondary")).not.toBeInTheDocument();
    expect(screen.getByTestId("appearance-remove-secondary")).toBeInTheDocument();
  });

  it("removing the secondary returns to a single color", () => {
    const onChange = vi.fn();
    render(<Harness initial="#4dabf7|#ff6b6b" onChange={onChange} />);
    fireEvent.click(screen.getByTestId("appearance-remove-secondary"));
    expect(onChange).toHaveBeenLastCalledWith("#4dabf7");
  });
});
