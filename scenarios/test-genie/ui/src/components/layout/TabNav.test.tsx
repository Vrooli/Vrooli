import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { TabNav } from "./TabNav";

describe("TabNav", () => {
  it("renders all tabs", () => {
    const onTabChange = vi.fn();
    render(<TabNav activeTab="dashboard" onTabChange={onTabChange} />);

    expect(screen.getByText("Dashboard")).toBeInTheDocument();
    expect(screen.getByText("Runs")).toBeInTheDocument();
    expect(screen.getByText("Generate")).toBeInTheDocument();
    expect(screen.getByText("Docs")).toBeInTheDocument();
  });

  it("highlights the active tab", () => {
    const onTabChange = vi.fn();
    render(<TabNav activeTab="runs" onTabChange={onTabChange} />);

    const runsTab = screen.getByText("Runs");
    expect(runsTab).toHaveClass("border-cyan-400");
    expect(runsTab).toHaveClass("bg-cyan-400/20");
  });

  it("calls onTabChange when a tab is clicked", () => {
    const onTabChange = vi.fn();
    render(<TabNav activeTab="dashboard" onTabChange={onTabChange} />);

    fireEvent.click(screen.getByText("Runs"));
    expect(onTabChange).toHaveBeenCalledWith("runs");

    fireEvent.click(screen.getByText("Generate"));
    expect(onTabChange).toHaveBeenCalledWith("generate");

    fireEvent.click(screen.getByText("Docs"));
    expect(onTabChange).toHaveBeenCalledWith("docs");
  });

  it("has correct data-testid attributes", () => {
    const onTabChange = vi.fn();
    render(<TabNav activeTab="dashboard" onTabChange={onTabChange} />);

    expect(screen.getByTestId("test-genie-tab-nav")).toBeInTheDocument();
    expect(screen.getByTestId("test-genie-tab-dashboard")).toBeInTheDocument();
    expect(screen.getByTestId("test-genie-tab-runs")).toBeInTheDocument();
    expect(screen.getByTestId("test-genie-tab-generate")).toBeInTheDocument();
    expect(screen.getByTestId("test-genie-tab-docs")).toBeInTheDocument();
  });

  it("inactive tabs have default styling", () => {
    const onTabChange = vi.fn();
    render(<TabNav activeTab="dashboard" onTabChange={onTabChange} />);

    const runsTab = screen.getByText("Runs");
    expect(runsTab).toHaveClass("border-white/20");
    expect(runsTab).toHaveClass("text-slate-300");
  });
});
