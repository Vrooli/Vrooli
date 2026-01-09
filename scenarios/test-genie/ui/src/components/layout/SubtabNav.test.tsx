import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { SubtabNav } from "./SubtabNav";

describe("SubtabNav", () => {
  it("renders scenarios and history subtabs", () => {
    const onSubtabChange = vi.fn();
    render(<SubtabNav activeSubtab="scenarios" onSubtabChange={onSubtabChange} />);

    expect(screen.getByText("Scenarios")).toBeInTheDocument();
    expect(screen.getByText("History")).toBeInTheDocument();
  });

  it("highlights the active subtab", () => {
    const onSubtabChange = vi.fn();
    render(<SubtabNav activeSubtab="scenarios" onSubtabChange={onSubtabChange} />);

    const scenariosTab = screen.getByText("Scenarios");
    expect(scenariosTab).toHaveClass("bg-white/10");
    expect(scenariosTab).toHaveClass("text-white");
  });

  it("inactive subtab has muted styling", () => {
    const onSubtabChange = vi.fn();
    render(<SubtabNav activeSubtab="scenarios" onSubtabChange={onSubtabChange} />);

    const historyTab = screen.getByText("History");
    expect(historyTab).toHaveClass("text-slate-400");
    expect(historyTab).not.toHaveClass("bg-white/10");
  });

  it("calls onSubtabChange when a subtab is clicked", () => {
    const onSubtabChange = vi.fn();
    render(<SubtabNav activeSubtab="scenarios" onSubtabChange={onSubtabChange} />);

    fireEvent.click(screen.getByText("History"));
    expect(onSubtabChange).toHaveBeenCalledWith("history");
  });

  it("has correct data-testid attributes", () => {
    const onSubtabChange = vi.fn();
    render(<SubtabNav activeSubtab="scenarios" onSubtabChange={onSubtabChange} />);

    expect(screen.getByTestId("test-genie-subtab-scenarios")).toBeInTheDocument();
    expect(screen.getByTestId("test-genie-subtab-history")).toBeInTheDocument();
  });

  it("switches active styling when subtab changes", () => {
    const onSubtabChange = vi.fn();
    const { rerender } = render(<SubtabNav activeSubtab="scenarios" onSubtabChange={onSubtabChange} />);

    expect(screen.getByText("Scenarios")).toHaveClass("bg-white/10");
    expect(screen.getByText("History")).not.toHaveClass("bg-white/10");

    rerender(<SubtabNav activeSubtab="history" onSubtabChange={onSubtabChange} />);

    expect(screen.getByText("Scenarios")).not.toHaveClass("bg-white/10");
    expect(screen.getByText("History")).toHaveClass("bg-white/10");
  });
});
