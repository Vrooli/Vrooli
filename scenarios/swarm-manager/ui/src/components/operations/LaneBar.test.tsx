import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { LaneBar } from "./LaneBar";

describe("LaneBar", () => {
  it("renders the lane label and active/capacity counts", () => {
    render(
      <LaneBar status={{ lane: "execute", active: 2, capacity: 3, queue: 0 }} />,
    );
    expect(screen.getByText("Execute")).toBeInTheDocument();
    expect(screen.getByText("2")).toBeInTheDocument();
    expect(screen.getByText("/ 3")).toBeInTheDocument();
  });

  it("shows queue count when non-zero", () => {
    render(
      <LaneBar status={{ lane: "execute", active: 3, capacity: 3, queue: 2 }} />,
    );
    expect(screen.getByText("+2 queued")).toBeInTheDocument();
  });

  it("flags warning state at or above 80% utilization", () => {
    const { container } = render(
      <LaneBar status={{ lane: "execute", active: 3, capacity: 3, queue: 0 }} />,
    );
    const wrapper = container.querySelector('[data-warning="true"]');
    expect(wrapper).not.toBeNull();
  });

  it("does not warn below 80% utilization", () => {
    const { container } = render(
      <LaneBar status={{ lane: "investigate", active: 1, capacity: 6, queue: 0 }} />,
    );
    const wrapper = container.querySelector('[data-warning="false"]');
    expect(wrapper).not.toBeNull();
  });

  it("renders unknown lanes with the fallback palette", () => {
    render(
      <LaneBar status={{ lane: "experimental", active: 0, capacity: 1, queue: 0 }} />,
    );
    expect(screen.getByText("experimental")).toBeInTheDocument();
  });

  it("renders the progressbar accessibility role", () => {
    render(
      <LaneBar status={{ lane: "review", active: 4, capacity: 8, queue: 0 }} />,
    );
    const bar = screen.getByRole("progressbar");
    expect(bar).toHaveAttribute("aria-valuenow", "4");
    expect(bar).toHaveAttribute("aria-valuemax", "8");
  });
});
