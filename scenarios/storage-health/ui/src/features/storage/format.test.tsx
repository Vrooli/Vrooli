import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";
import { EngineChips, FitnessMeter, IsolationBadge } from "./format";

afterEach(() => cleanup());

describe("storage format primitives", () => {
  it("renders the isolation-ready badge with icon + text", () => {
    renderWithProviders(<IsolationBadge ready />);
    expect(screen.getByText(strings.isolation.ready)).toBeInTheDocument();
  });

  it("renders the isolation-unready badge with the alert text", () => {
    renderWithProviders(<IsolationBadge ready={false} />);
    expect(screen.getByText(strings.isolation.unready)).toBeInTheDocument();
  });

  it("renders engine chips for each engine", () => {
    renderWithProviders(<EngineChips engines={["sqlite", "redis"]} />);
    expect(screen.getByText(/sqlite/)).toBeInTheDocument();
    expect(screen.getByText(/redis/)).toBeInTheDocument();
  });

  it("renders an em dash for an empty engine list", () => {
    renderWithProviders(<EngineChips engines={[]} />);
    expect(screen.getByText("—")).toBeInTheDocument();
  });

  it("renders the fitness meter with a clamped numeric value and role", () => {
    renderWithProviders(<FitnessMeter score={0.8} label="Fitness" />);
    const meter = screen.getByRole("meter");
    expect(meter).toHaveAttribute("aria-valuenow", "0.8");
    expect(screen.getByText("0.80")).toBeInTheDocument();
  });

  it("clamps out-of-range fitness scores into [0,1]", () => {
    renderWithProviders(<FitnessMeter score={1.7} label="Fitness" />);
    expect(screen.getByRole("meter")).toHaveAttribute("aria-valuenow", "1");
  });
});
