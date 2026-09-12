import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { StatusChip } from "./status-chip";

describe("StatusChip", () => {
  const colors = {
    background: "bg-blue-500/20",
    border: "border-blue-400/80",
    text: "text-blue-300",
    dot: "bg-blue-500",
  };

  it("renders the label", () => {
    render(<StatusChip label="Completed" colors={colors} />);
    expect(screen.getByText("Completed")).toBeInTheDocument();
  });

  it("renders as a span when onClick is not provided", () => {
    const { container } = render(<StatusChip label="Queued" colors={colors} />);
    expect(container.querySelector("span")).toBeTruthy();
    expect(container.querySelector("button")).toBeNull();
  });

  it("renders as a button when onClick is provided", () => {
    const onClick = vi.fn();
    render(<StatusChip label="Click me" colors={colors} onClick={onClick} />);
    const btn = screen.getByRole("button", { name: "Click me" });
    fireEvent.click(btn);
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("applies tone classes", () => {
    const { container } = render(<StatusChip label="x" colors={colors} />);
    const el = container.firstChild as HTMLElement;
    expect(el.className).toContain("bg-blue-500/20");
    expect(el.className).toContain("border-blue-400/80");
    expect(el.className).toContain("text-blue-300");
  });

  it("renders a leading dot when leadingDot is true", () => {
    const { container } = render(
      <StatusChip label="x" colors={colors} leadingDot data-testid="chip" />,
    );
    const chip = container.querySelector('[data-testid="chip"]') as HTMLElement;
    const dot = chip.querySelector("[aria-hidden]");
    expect(dot).toBeTruthy();
  });

  it("adds the pulse animation when pulse is true", () => {
    const { container } = render(
      <StatusChip label="Running" colors={colors} leadingDot pulse data-testid="chip" />,
    );
    const ping = container.querySelector(".animate-ping");
    expect(ping).toBeTruthy();
  });

  it("does not render a dot when leadingDot is false", () => {
    const { container } = render(
      <StatusChip label="x" colors={colors} data-testid="chip" />,
    );
    const chip = container.querySelector('[data-testid="chip"]') as HTMLElement;
    expect(chip.querySelector("[aria-hidden]")).toBeNull();
  });

  it("passes through title as tooltip", () => {
    render(<StatusChip label="x" colors={colors} title="Tooltip text" data-testid="chip" />);
    expect(screen.getByTestId("chip")).toHaveAttribute("title", "Tooltip text");
  });

  it("omits border when colors.border is undefined", () => {
    const noBorderColors = { background: "bg-red-500/20", text: "text-red-400" };
    const { container } = render(<StatusChip label="x" colors={noBorderColors} />);
    const el = container.firstChild as HTMLElement;
    expect(el.className).not.toContain("border-");
  });
});
