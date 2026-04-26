import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { PlaybackModeControl } from "../tts/PlaybackModeControl";

function renderCtl(overrides?: Partial<Parameters<typeof PlaybackModeControl>[0]>) {
  return render(
    <PlaybackModeControl
      testIdPrefix="x"
      isSummarized={false}
      hasOriginalVersion={false}
      canSummarize
      isSummarizing={false}
      currentLevel="moderate"
      onToggleSummarized={vi.fn()}
      onChangeLevel={vi.fn()}
      {...overrides}
    />,
  );
}

describe("PlaybackModeControl", () => {
  it("renders nothing when neither hasOriginalVersion nor canSummarize", () => {
    const { container } = renderCtl({ hasOriginalVersion: false, canSummarize: false });
    expect(container.firstChild).toBeNull();
  });

  it("renders the active summary level label when isSummarized", () => {
    renderCtl({ isSummarized: true, hasOriginalVersion: true, currentLevel: "heavy" });
    expect(screen.getByTestId("x-mode-control").textContent).toMatch(/Heavy/);
  });

  it("renders 'Original' label when !isSummarized && hasOriginalVersion", () => {
    renderCtl({ isSummarized: false, hasOriginalVersion: true });
    expect(screen.getByTestId("x-mode-control").textContent).toMatch(/Original/);
  });

  it("renders 'Summarize' label when !hasOriginalVersion && canSummarize", () => {
    renderCtl({ isSummarized: false, hasOriginalVersion: false, canSummarize: true });
    expect(screen.getByTestId("x-mode-control").textContent).toMatch(/Summarize/);
  });

  it("opens dropdown with all level options + Original when clicked", () => {
    renderCtl({ isSummarized: true, hasOriginalVersion: true });
    fireEvent.click(screen.getByTestId("x-mode-control"));
    expect(screen.getByTestId("x-mode-menu")).toBeInTheDocument();
    expect(screen.getByTestId("x-mode-option-original")).toBeInTheDocument();
    expect(screen.getByTestId("x-mode-option-light")).toBeInTheDocument();
    expect(screen.getByTestId("x-mode-option-moderate")).toBeInTheDocument();
    expect(screen.getByTestId("x-mode-option-heavy")).toBeInTheDocument();
  });

  it("omits Original option when !hasOriginalVersion", () => {
    renderCtl({ isSummarized: false, hasOriginalVersion: false, canSummarize: true });
    fireEvent.click(screen.getByTestId("x-mode-control"));
    expect(screen.queryByTestId("x-mode-option-original")).toBeNull();
    expect(screen.getByTestId("x-mode-option-light")).toBeInTheDocument();
  });

  it("active level has a check icon (lucide svg rendered inside)", () => {
    renderCtl({ isSummarized: true, hasOriginalVersion: true, currentLevel: "heavy" });
    fireEvent.click(screen.getByTestId("x-mode-control"));
    const activeOption = screen.getByTestId("x-mode-option-heavy");
    expect(activeOption.querySelector("svg")).not.toBeNull();
    const inactiveOption = screen.getByTestId("x-mode-option-light");
    expect(inactiveOption.querySelector("svg")).toBeNull();
  });

  it("clicking Original calls onToggleSummarized(false)", () => {
    const onToggleSummarized = vi.fn();
    renderCtl({ isSummarized: true, hasOriginalVersion: true, onToggleSummarized });
    fireEvent.click(screen.getByTestId("x-mode-control"));
    fireEvent.click(screen.getByTestId("x-mode-option-original"));
    expect(onToggleSummarized).toHaveBeenCalledWith(false);
  });

  it("clicking a different level calls onChangeLevel", () => {
    const onChangeLevel = vi.fn();
    renderCtl({ isSummarized: true, hasOriginalVersion: true, currentLevel: "moderate", onChangeLevel });
    fireEvent.click(screen.getByTestId("x-mode-control"));
    fireEvent.click(screen.getByTestId("x-mode-option-light"));
    expect(onChangeLevel).toHaveBeenCalledWith("light");
  });

  it("clicking the currently active level is a no-op", () => {
    const onChangeLevel = vi.fn();
    renderCtl({ isSummarized: true, hasOriginalVersion: true, currentLevel: "moderate", onChangeLevel });
    fireEvent.click(screen.getByTestId("x-mode-control"));
    fireEvent.click(screen.getByTestId("x-mode-option-moderate"));
    expect(onChangeLevel).not.toHaveBeenCalled();
  });

  it("clicking a level when NOT currently summarized always fires onChangeLevel", () => {
    const onChangeLevel = vi.fn();
    renderCtl({ isSummarized: false, hasOriginalVersion: false, canSummarize: true, currentLevel: "moderate", onChangeLevel });
    fireEvent.click(screen.getByTestId("x-mode-control"));
    fireEvent.click(screen.getByTestId("x-mode-option-moderate"));
    expect(onChangeLevel).toHaveBeenCalledWith("moderate");
  });

  it("disables the button and shows spinner when isSummarizing", () => {
    renderCtl({ isSummarizing: true });
    const btn = screen.getByTestId("x-mode-control");
    expect(btn).toBeDisabled();
    expect(btn.querySelector("svg")).not.toBeNull();
  });

  it("disables the button when disabled prop is true (idle state)", () => {
    renderCtl({ isSummarized: true, hasOriginalVersion: true, disabled: true });
    expect(screen.getByTestId("x-mode-control")).toBeDisabled();
  });

  it("closes menu when backdrop is clicked", () => {
    renderCtl({ isSummarized: true, hasOriginalVersion: true });
    fireEvent.click(screen.getByTestId("x-mode-control"));
    expect(screen.getByTestId("x-mode-menu")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("x-mode-menu-backdrop"));
    expect(screen.queryByTestId("x-mode-menu")).toBeNull();
  });
});
