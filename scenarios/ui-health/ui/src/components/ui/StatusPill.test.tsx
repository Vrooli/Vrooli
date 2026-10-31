import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { StatusPill } from "./StatusPill";

describe("StatusPill", () => {
  it("renders the label", () => {
    render(<StatusPill status="ok" label="healthy-x" data-testid="p" />);
    expect(screen.getByTestId("p")).toHaveTextContent("healthy-x");
  });
  it("applies status-specific class", () => {
    render(<StatusPill status="error" label="down-x" data-testid="p" />);
    expect(screen.getByTestId("p").className).toMatch(/text-app-danger/);
  });
  it("spins icon for running status", () => {
    const { container } = render(<StatusPill status="running" label="reindex-x" data-testid="p" />);
    expect(container.querySelector(".animate-spin")).not.toBeNull();
  });
});
