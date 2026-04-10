import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { BrandCard } from "./brand-card";
import type { Brand } from "../lib/api";

// [REQ:BM-REQ-UI-LIBRARY] [REQ:BM-REQ-UI-DASHBOARD]

const baseBrand: Brand = {
  id: "b1",
  name: "Acme Corp",
  description: "The Acme brand",
  version: 3,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-03-15T10:30:00Z",
  colors: { primary: "#ff0000", secondary: "#00ff00", accent: "#0000ff" },
};

describe("BrandCard", () => {
  it("renders brand name and description", () => {
    render(<BrandCard brand={baseBrand} onClick={() => {}} />);
    expect(screen.getByText("Acme Corp")).toBeTruthy();
    expect(screen.getByText("The Acme brand")).toBeTruthy();
  });

  it("renders version number", () => {
    render(<BrandCard brand={baseBrand} onClick={() => {}} />);
    expect(screen.getByText("v3")).toBeTruthy();
  });

  it("renders color swatches when colors exist", () => {
    const { container } = render(<BrandCard brand={baseBrand} onClick={() => {}} />);
    const swatches = container.querySelectorAll("[style]");
    expect(swatches.length).toBe(3);
  });

  it("omits color swatches when no colors", () => {
    const noColors = { ...baseBrand, colors: undefined };
    const { container } = render(<BrandCard brand={noColors} onClick={() => {}} />);
    const swatches = container.querySelectorAll("[style]");
    expect(swatches.length).toBe(0);
  });

  it("hides description when absent", () => {
    const noDesc = { ...baseBrand, description: undefined };
    render(<BrandCard brand={noDesc} onClick={() => {}} />);
    expect(screen.queryByText("The Acme brand")).toBeNull();
  });

  it("calls onClick when clicked", () => {
    const onClick = vi.fn();
    render(<BrandCard brand={baseBrand} onClick={onClick} />);
    fireEvent.click(screen.getByTestId("brand-card-b1"));
    expect(onClick).toHaveBeenCalledOnce();
  });

  it("sets data-testid with brand id", () => {
    render(<BrandCard brand={baseBrand} onClick={() => {}} />);
    expect(screen.getByTestId("brand-card-b1")).toBeTruthy();
  });
});
