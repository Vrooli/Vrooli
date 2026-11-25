import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders as render } from "../test-utils";
import type { CompactCandidate } from "../api/usageMode";
import { ContextBudgetSummary } from "./ContextBudgetSummary";

const row = (contextBudget: NonNullable<CompactCandidate["contextBudget"]>): CompactCandidate => ({
  sessionId: "s1", label: "Planner", harness: "codex", state: "idle", contextTokens: 120_000,
  goalActive: false, eligible: true, armed: false, thresholdTokens: 0, contextBudget,
});

describe("context budget summary", () => {
  it("uses the shared provenance record for known meter limits", () => {
    render(<ContextBudgetSummary row={row({
      model: "gpt-6.1-terra", modelState: "reported", modelSource: "native transcript",
      usedTokens: 120_000, usedSource: "native transcript",
      modelWindowTokens: 200_000, modelWindowSource: "codex-models-cache/v1",
      compactionPointTokens: 150_000, compactionPointSource: "session launch descriptor",
      tokensUntilCompaction: 30_000, forecastState: "projected", progressPercent: 60,
    })} />);
    const summary = screen.getByTestId("context-budget");
    expect(summary).toHaveAttribute("data-window-tokens", "200000");
    expect(summary).toHaveAttribute("data-window-source", "codex-models-cache/v1");
    expect(summary).toHaveAttribute("data-compaction-tokens", "150000");
    expect(summary).toHaveAttribute("data-compaction-source", "session launch descriptor");
  });

  it("keeps missing limits unknown even when a source string is stale", () => {
    render(<ContextBudgetSummary row={row({
      modelState: "unknown", modelSource: "unknown", usedTokens: null, usedSource: "unknown",
      modelWindowTokens: null, modelWindowSource: "codex-models-cache/v1",
      compactionPointTokens: null, compactionPointSource: "session launch descriptor",
      tokensUntilCompaction: null, forecastState: "unknown", progressPercent: null,
    })} />);
    const summary = screen.getByTestId("context-budget");
    expect(summary).toHaveAttribute("data-window-tokens", "unknown");
    expect(summary).toHaveAttribute("data-window-source", "unknown");
    expect(summary).toHaveAttribute("data-compaction-tokens", "unknown");
    expect(summary).toHaveAttribute("data-compaction-source", "unknown");
  });

  it("suppresses an old unversioned window and its derived progress meter", () => {
    render(<ContextBudgetSummary row={row({
      model: "gpt-6.1-terra", modelState: "reported", modelSource: "native transcript",
      usedTokens: 120_000, usedSource: "native transcript",
      modelWindowTokens: 200_000, modelWindowSource: "codex-models-cache",
      compactionPointTokens: 150_000, compactionPointSource: "session launch descriptor",
      tokensUntilCompaction: 30_000, forecastState: "projected", progressPercent: 60,
    })} />);
    const summary = screen.getByTestId("context-budget");
    expect(summary).toHaveAttribute("data-window-tokens", "unknown");
    expect(summary).toHaveAttribute("data-window-source", "unknown");
    expect(screen.queryByRole("meter")).toBeNull();
  });
});
