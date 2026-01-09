import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { StatusPill } from "./StatusPill";

describe("StatusPill", () => {
  it("renders the status text", () => {
    render(<StatusPill status="passed" />);
    expect(screen.getByText("passed")).toBeInTheDocument();
  });

  it("applies passed/completed styling for passed status", () => {
    render(<StatusPill status="passed" />);
    const pill = screen.getByText("passed");
    expect(pill).toHaveClass("bg-emerald-500/20");
    expect(pill).toHaveClass("text-emerald-300");
  });

  it("applies passed/completed styling for completed status", () => {
    render(<StatusPill status="completed" />);
    const pill = screen.getByText("completed");
    expect(pill).toHaveClass("bg-emerald-500/20");
  });

  it("applies running styling for running status", () => {
    render(<StatusPill status="running" />);
    const pill = screen.getByText("running");
    expect(pill).toHaveClass("bg-cyan-500/20");
    expect(pill).toHaveClass("text-cyan-200");
  });

  it("applies queued styling for queued status", () => {
    render(<StatusPill status="queued" />);
    const pill = screen.getByText("queued");
    expect(pill).toHaveClass("bg-amber-400/20");
    expect(pill).toHaveClass("text-amber-200");
  });

  it("applies idle styling for idle status", () => {
    render(<StatusPill status="idle" />);
    const pill = screen.getByText("idle");
    expect(pill).toHaveClass("bg-white/10");
    expect(pill).toHaveClass("text-slate-200");
  });

  it("applies failed styling for unknown/failed status", () => {
    render(<StatusPill status="failed" />);
    const pill = screen.getByText("failed");
    expect(pill).toHaveClass("bg-red-500/20");
    expect(pill).toHaveClass("text-red-200");
  });

  it("is case insensitive", () => {
    render(<StatusPill status="PASSED" />);
    const pill = screen.getByText("PASSED");
    expect(pill).toHaveClass("bg-emerald-500/20");
  });

  it("applies custom className", () => {
    render(<StatusPill status="passed" className="custom-class" />);
    const pill = screen.getByText("passed");
    expect(pill).toHaveClass("custom-class");
  });

  it("has correct base styles", () => {
    render(<StatusPill status="passed" />);
    const pill = screen.getByText("passed");
    expect(pill).toHaveClass("rounded-full");
    expect(pill).toHaveClass("text-xs");
    expect(pill).toHaveClass("uppercase");
  });
});
