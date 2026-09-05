/**
 * Tests for StatsMetricCard and InsufficientDataCard.
 *
 * These guard the Phase 7 behavior: a stat with zero or under-threshold
 * samples must render as an insufficient-data affordance, never as a
 * misleading numeric zero or rate.
 */

import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { StatsMetricCard } from "./stats-metric-card";
import { InsufficientDataCard } from "./insufficient-data-card";

describe("StatsMetricCard", () => {
  it("renders the value and sample size when sampleSize ≥ minSample", () => {
    render(
      <StatsMetricCard
        label="Avg duration"
        value="4.2 min"
        sampleSize={10}
        minSample={5}
        sampleNoun="runs"
        testId="avg-dur"
      />,
    );
    expect(screen.getByText("Avg duration")).toBeInTheDocument();
    expect(screen.getByText("4.2 min")).toBeInTheDocument();
    expect(screen.getByText(/n = 10/)).toBeInTheDocument();
  });

  it("delegates to InsufficientDataCard when sampleSize is zero", () => {
    render(
      <StatsMetricCard
        label="Avg duration"
        value="NEVER SHOWN"
        sampleSize={0}
        minSample={5}
        testId="avg-dur"
      />,
    );
    expect(screen.queryByText("NEVER SHOWN")).not.toBeInTheDocument();
    expect(screen.getByText(/Not enough data yet/)).toBeInTheDocument();
    expect(screen.getByText(/0 of 5 needed/)).toBeInTheDocument();
  });

  it("delegates to InsufficientDataCard when sampleSize is below threshold", () => {
    render(
      <StatsMetricCard
        label="Success rate"
        value="50%"
        sampleSize={2}
        minSample={5}
      />,
    );
    expect(screen.queryByText("50%")).not.toBeInTheDocument();
    expect(screen.getByText(/Not enough data yet/)).toBeInTheDocument();
    expect(screen.getByText(/2 of 5 needed/)).toBeInTheDocument();
  });

  it("uses the provided reason verbatim when insufficient", () => {
    render(
      <StatsMetricCard
        label="Avg duration"
        value="x"
        sampleSize={0}
        minSample={5}
        insufficientReason="We need at least 5 finished executions."
      />,
    );
    expect(screen.getByText("We need at least 5 finished executions.")).toBeInTheDocument();
  });
});

describe("InsufficientDataCard", () => {
  it("renders reason and 'have of required' when both provided", () => {
    render(
      <InsufficientDataCard
        label="Lead time"
        reason="Need at least 5 completions."
        have={1}
        required={5}
      />,
    );
    expect(screen.getByText("Lead time")).toBeInTheDocument();
    expect(screen.getByText(/Need at least 5 completions/)).toBeInTheDocument();
    expect(screen.getByText(/1 of 5 needed/)).toBeInTheDocument();
  });
});
