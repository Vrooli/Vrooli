import { describe, expect, it } from "vitest";
import { renderWithProviders, screen } from "../test-utils/renderWithProviders";
import { makeReading } from "../test-utils/readings";
import { LeaderboardReadout } from "./LeaderboardReadout";

describe("LeaderboardReadout", () => {
  it("shows verdicts, control, denominators, and CTA secondary data", () => {
    renderWithProviders(<LeaderboardReadout reading={makeReading({ kind: "leaderboard", value: 2, rows: [
      { key: "control", label: "Control", value: 2, share: 1, denominator: 10, rate: .2, verdict: "CONTROL", is_control: true, cta_clicks: 3, cta_trials: 10 },
      { key: "b", label: "Variant B", value: 4, share: 1, denominator: 10, rate: .4, verdict: "LEADING", probability: .91, cta_clicks: 5, cta_trials: 10 },
    ] })} />);
    expect(screen.getAllByText("CONTROL").length).toBeGreaterThan(0);
    expect(screen.getByText("LEADING")).toBeInTheDocument();
    expect(screen.getByText(/CTA 5/)).toBeInTheDocument();
    expect(screen.getByText(/91% probability/)).toBeInTheDocument();
  });
});
