import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Tooltip } from "./tooltip";

// [REQ:UI-A11Y] Tooltip accessibility and interaction
describe("Tooltip", () => {
  it("renders children without tooltip initially", () => {
    render(<Tooltip content="Help text"><button>Hover me</button></Tooltip>);
    expect(screen.getByText("Hover me")).toBeInTheDocument();
    expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();
  });

  it("shows tooltip on mouse enter", () => {
    render(<Tooltip content="Help text"><button>Hover me</button></Tooltip>);
    fireEvent.mouseEnter(screen.getByText("Hover me").closest("span")!);
    expect(screen.getByRole("tooltip")).toBeInTheDocument();
    expect(screen.getByText("Help text")).toBeInTheDocument();
  });

  it("hides tooltip on mouse leave", () => {
    render(<Tooltip content="Help text"><button>Hover me</button></Tooltip>);
    const wrapper = screen.getByText("Hover me").closest("span")!;
    fireEvent.mouseEnter(wrapper);
    expect(screen.getByRole("tooltip")).toBeInTheDocument();
    fireEvent.mouseLeave(wrapper);
    expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();
  });

  it("shows tooltip on focus", () => {
    render(<Tooltip content="Focus text"><button>Focus me</button></Tooltip>);
    fireEvent.focus(screen.getByText("Focus me").closest("span")!);
    expect(screen.getByRole("tooltip")).toBeInTheDocument();
    expect(screen.getByText("Focus text")).toBeInTheDocument();
  });

  it("hides tooltip on blur", () => {
    render(<Tooltip content="Focus text"><button>Focus me</button></Tooltip>);
    const wrapper = screen.getByText("Focus me").closest("span")!;
    fireEvent.focus(wrapper);
    expect(screen.getByRole("tooltip")).toBeInTheDocument();
    fireEvent.blur(wrapper);
    expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();
  });

  it("has role=tooltip on the tooltip element", () => {
    render(<Tooltip content="Accessible"><button>Target</button></Tooltip>);
    fireEvent.mouseEnter(screen.getByText("Target").closest("span")!);
    expect(screen.getByRole("tooltip")).toHaveTextContent("Accessible");
  });

  it("accepts custom className", () => {
    render(<Tooltip content="Tip" className="custom-class"><span>Test</span></Tooltip>);
    const wrapper = screen.getByText("Test").closest("span.custom-class");
    expect(wrapper).toBeInTheDocument();
  });
});
