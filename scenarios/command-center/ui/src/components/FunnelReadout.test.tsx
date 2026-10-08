import { describe, expect, it } from "vitest";
import { renderWithProviders, screen } from "../test-utils/renderWithProviders";
import { makeReading } from "../test-utils/readings";
import { FunnelReadout } from "./FunnelReadout";

describe("FunnelReadout", () => {
  it("shows measured counts with their measured denominators", () => {
    renderWithProviders(<FunnelReadout reading={makeReading({ kind: "funnel", value: 100, rows: [{ key: "visitors", label: "Visitors", value: 100, share: 1 }, { key: "paid", label: "Paid", value: 12, share: .12, denominator: 40, rate: .3 }] })} />);
    expect(screen.getByText("12 of 40")).toBeInTheDocument();
    expect(screen.getByText("30.0% of prior step")).toBeInTheDocument();
  });
});
