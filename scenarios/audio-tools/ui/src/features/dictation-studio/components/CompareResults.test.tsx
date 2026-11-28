import { screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderWithProviders } from "@vrooli/api-base/testing";
import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { CompareResults } from "./CompareResults";

function row({
  id,
  name = id,
  recipe,
  winner,
  perStrategy = [],
  qualityMeasured = true,
  latencyMeasured = true,
}: {
  id: string;
  name?: string;
  recipe?: Record<string, unknown>;
  winner?: string;
  perStrategy?: Array<Record<string, unknown>>;
  qualityMeasured?: boolean;
  latencyMeasured?: boolean;
}) {
  return {
    experiment: { id, name, status: "completed", recipe },
    report: {
      summary: winner ? { winnerStrategy: winner } : undefined,
      perStrategy,
      qualityMeasured,
      latencyMeasured,
    },
  } as never;
}

describe("CompareResults", () => {
  it("renders the comparison shell without optional recipe or strategy sections", () => {
    renderWithProviders(<CompareResults rows={[]} />);
    expect(screen.getByTestId(selectors.dictationStudio.compareResults)).toBeInTheDocument();
    expect(screen.queryByText(strings.dictationStudio.compareRecipeDiffTitle)).not.toBeInTheDocument();
    expect(screen.queryByText(strings.dictationStudio.compareStrategyAlignmentTitle)).not.toBeInTheDocument();
  });

  it("shows fallbacks when reports lack measurements, strategies, or experiments", () => {
    renderWithProviders(
      <CompareResults
        rows={[
          row({ id: "one", perStrategy: [{ strategy: "", label: "", wer: 0.2, finalizationLatencyP95Ms: 90 }] }),
          { experiment: null, report: { perStrategy: [], qualityMeasured: false, latencyMeasured: false } } as never,
        ]}
      />,
    );
    const comparison = screen.getByTestId(selectors.dictationStudio.compareResults);
    expect(within(comparison).getAllByText(strings.common.dash).length).toBeGreaterThan(0);
    expect(within(comparison).getAllByText(strings.dictationStudio.safetyMeasured)).toHaveLength(2);
    expect(within(comparison).getByRole("heading", { name: strings.dictationStudio.compareStrategyAlignmentTitle })).toBeInTheDocument();
  });

  it("reports two-way recipe changes and flags a recommended unsafe winner", () => {
    const recipe = { clipIds: ["clip-a"], realtimeRepeats: 1, chunkMs: 400, seed: 3, latencyTailSeconds: 2,
      droppedSpanThresholdWords: 4, longFormEnabled: false, targetDurationSeconds: 20, gapMs: 100,
      tagContains: "", sweepDurationsSeconds: [], noiseTypes: [], snrDb: [], competingVoiceIds: [],
      competingText: "", speakerTargetProfileId: "", speakerExtraction: false, speakerVerification: false,
      speakerMode: "strict", speakerThreshold: 0.8, speakerAblation: false, strategyDetails: [] };
    const strategy = { strategy: "fast/v1", label: "Fast", wer: 0.2, finalizationLatencyP95Ms: 125,
      safety: { passed: false } };
    renderWithProviders(
      <CompareResults rows={[
        row({ id: "a", recipe, winner: "fast/v1", perStrategy: [strategy] }),
        row({ id: "b", recipe: { ...recipe, chunkMs: 600, clipIds: ["clip-b"] }, perStrategy: [{ ...strategy, strategy: "accurate/v2", safety: { passed: true } }] }),
      ]} />,
    );
    expect(screen.getByText((content) => content === "chunk_ms: 400 -> 600")).toBeInTheDocument();
    expect(screen.getByText((content) => content === "clip_ids: clip-a -> clip-b")).toBeInTheDocument();
    expect(screen.getByText(strings.dictationStudio.recommendedDespiteSafetyFailure)).toBeInTheDocument();
    expect(screen.getByText(strings.dictationStudio.safetyUnsafe)).toBeInTheDocument();
    expect(screen.getByText((content) => content === "fast")).toBeInTheDocument();
  });

  it("lists multi-experiment recipe values and aligns strategy families with unavailable cells", () => {
    const base = { clipIds: [], realtimeRepeats: 1, chunkMs: 400, seed: 1, latencyTailSeconds: 1,
      droppedSpanThresholdWords: 2, longFormEnabled: false, targetDurationSeconds: 0, gapMs: 0, tagContains: "",
      sweepDurationsSeconds: [], noiseTypes: [], snrDb: [], competingVoiceIds: [], competingText: "",
      speakerTargetProfileId: "", speakerExtraction: false, speakerVerification: false, speakerMode: "", speakerThreshold: 0,
      speakerAblation: false, strategyDetails: [{ kind: "overlap", overlapMaxWindowMs: 500,
        overlapMaxStallRejects: 2, overlapWindowMs: 200, overlapCommitRuns: 3, vadSilenceMs: 250 }] };
    renderWithProviders(
      <CompareResults rows={[
        row({ id: "a", name: "Alpha", recipe: base, perStrategy: [{ strategy: "fast/one", label: "Fast", wer: 0.1, finalizationLatencyP95Ms: 100 }] }),
        row({ id: "b", name: "Beta", recipe: { ...base, seed: 2, strategyDetails: [{ ...base.strategyDetails[0], overlapWindowMs: 300 }] }, perStrategy: [{ strategy: "/alias", label: "Alternate/name", wer: 0.2, finalizationLatencyP95Ms: 200 }] }),
        row({ id: "c", name: "Gamma", recipe: { ...base, seed: 3, strategyDetails: [{ ...base.strategyDetails[0], overlapWindowMs: 400 }] }, perStrategy: [] }),
      ]} />,
    );
    expect(screen.getByText((content) => content === "seed: Alpha=1, Beta=2, Gamma=3")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: strings.dictationStudio.compareStrategyAlignmentTitle })).toBeInTheDocument();
    expect(screen.getByText((content) => content === "10.0% / 100")).toBeInTheDocument();
    expect(screen.getAllByText((content) => content === "-").length).toBeGreaterThan(0);
  });
});
