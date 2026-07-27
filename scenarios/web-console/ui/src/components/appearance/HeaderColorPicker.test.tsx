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
    expect(screen.queryByTestId("appearance-header-color-recents")).not.toBeInTheDocument();
  });

  it("renders recents and records a color on pick", () => {
    useWorkspaceStore.setState({ recentHeaderColors: ["#abcdef"] });
    const onChange = vi.fn();
    render(<Harness onChange={onChange} />);
    expect(screen.getByTestId("appearance-header-color-recents")).toBeInTheDocument();
    expect(screen.getByTestId("appearance-header-color-recent-#abcdef")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("appearance-header-color-palette-#ff6b6b"));
    expect(onChange).toHaveBeenCalledWith("#ff6b6b");
    expect(useWorkspaceStore.getState().recentHeaderColors[0]).toBe("#ff6b6b");
  });
});

describe("HeaderColorPicker two-color UX", () => {
  it("serializes a single color when secondary is closed", () => {
    const onChange = vi.fn();
    render(<Harness onChange={onChange} />);
    fireEvent.click(screen.getByTestId("appearance-header-color-palette-#4dabf7"));
    expect(onChange).toHaveBeenLastCalledWith("#4dabf7");
  });

  it("adds a secondary color and serializes a pair, capped at 2", () => {
    const onChange = vi.fn();
    render(<Harness onChange={onChange} />);

    // Pick a primary color first.
    fireEvent.click(screen.getByTestId("appearance-header-color-palette-#4dabf7"));
    expect(onChange).toHaveBeenLastCalledWith("#4dabf7");

    // Open the secondary slot, then pick a second color.
    fireEvent.click(screen.getByTestId("appearance-header-color-add-gradient"));
    fireEvent.click(screen.getByTestId("appearance-header-color-palette-#ff6b6b"));
    expect(onChange).toHaveBeenLastCalledWith("#4dabf7|#ff6b6b");

    // No "add secondary" affordance remains (cap of 2 reached).
    expect(screen.queryByTestId("appearance-header-color-add-gradient")).not.toBeInTheDocument();
    expect(screen.getByTestId("appearance-header-color-remove-gradient")).toBeInTheDocument();
  });

  it("removing the secondary returns to a single color", () => {
    const onChange = vi.fn();
    render(<Harness initial="#4dabf7|#ff6b6b" onChange={onChange} />);
    fireEvent.click(screen.getByTestId("appearance-header-color-remove-gradient"));
    expect(onChange).toHaveBeenLastCalledWith("#4dabf7");
  });
});

describe("HeaderColorPicker custom-input recents", () => {
  it("dragging the native picker applies live without recording recents", () => {
    const onChange = vi.fn();
    render(<Harness onChange={onChange} />);
    const input = screen.getByTestId("appearance-header-color-custom-input");

    // Chromium fires BOTH `input` and `change` per drag tick — neither may
    // record a recent.
    fireEvent.input(input, { target: { value: "#111111" } });
    fireEvent.change(input, { target: { value: "#222222" } });

    expect(onChange).toHaveBeenLastCalledWith("#222222");
    expect(useWorkspaceStore.getState().recentHeaderColors).toEqual([]);
  });

  it("blurring the input records only the final dragged color", () => {
    const onChange = vi.fn();
    render(<Harness onChange={onChange} />);
    const input = screen.getByTestId("appearance-header-color-custom-input");

    fireEvent.change(input, { target: { value: "#111111" } });
    fireEvent.change(input, { target: { value: "#333333" } });
    fireEvent.blur(input);

    expect(useWorkspaceStore.getState().recentHeaderColors).toEqual(["#333333"]);

    // A second blur without new picking must not double-record.
    fireEvent.blur(input);
    expect(useWorkspaceStore.getState().recentHeaderColors).toEqual(["#333333"]);
  });

  it("unmounting flushes the pending custom color into recents", () => {
    const { unmount } = render(<Harness />);
    const input = screen.getByTestId("appearance-header-color-custom-input");

    fireEvent.change(input, { target: { value: "#444444" } });
    expect(useWorkspaceStore.getState().recentHeaderColors).toEqual([]);

    unmount();
    expect(useWorkspaceStore.getState().recentHeaderColors).toEqual(["#444444"]);
  });

  it("swatch clicks still record recents immediately", () => {
    render(<Harness />);
    fireEvent.click(screen.getByTestId("appearance-header-color-palette-#ff6b6b"));
    expect(useWorkspaceStore.getState().recentHeaderColors).toEqual(["#ff6b6b"]);
  });
});
