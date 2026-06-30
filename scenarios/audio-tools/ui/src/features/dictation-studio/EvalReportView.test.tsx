import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";

const runEval = vi.fn();
vi.mock("../../services/corpus", () => ({
  runEval: (args: unknown) => runEval(args),
}));

import { EvalReportView } from "./EvalReportView";

function row(strategy: string, over: Record<string, number> = {}) {
  return {
    strategy,
    label: strategy,
    wer: 0.1,
    substitutions: 1,
    insertions: 0,
    deletions: 0,
    refWords: 10,
    whisperCalls: 2,
    whisperAudioSeconds: 3,
    rtf: 0.5,
    finalizationLatencyP50Ms: 100,
    finalizationLatencyP95Ms: 200,
    partialRevisions: 1,
    perClip: [
      {
        clipId: "clip-1",
        reference: "Hello world",
        hypothesis: "Hello word",
        wer: 0.5,
        whisperCalls: 1,
        whisperAudioSeconds: 1.5,
        rtf: 0.5,
        segmentCount: 1,
        partialRevisions: 1,
        finalizationLatencyP50Ms: 90,
        finalizationLatencyP95Ms: 120,
        error: "",
        substitutions: 1,
        insertions: 0,
        deletions: 0,
        refWords: 2,
        hypWords: 2,
        normalizedReference: "hello world",
        normalizedHypothesis: "hello word",
        editOperations: [
          { kind: "match", referenceToken: "hello", hypothesisToken: "hello", referenceIndex: 0, hypothesisIndex: 0 },
          { kind: "substitution", referenceToken: "world", hypothesisToken: "word", referenceIndex: 1, hypothesisIndex: 1 },
        ],
      },
    ],
    werDeltaVsWinner: 0,
    p95DeltaMsVsWinner: 0,
    callMultiplierVsWinner: 1,
    verdict: "winner",
    reasons: ["Lowest WER after deterministic normalization."],
    warnings: [],
    ...over,
  };
}

function report(over = {}) {
  return {
    perStrategy: [row("batch"), row("vad_segment", { wer: 0.05, finalizationLatencyP95Ms: 100 })],
    qualityMeasured: true,
    latencyMeasured: false,
    summary: {
      winnerStrategy: "vad_segment",
      winnerLabel: "vad_segment",
      recommendation: "Prefer vad_segment for this corpus.",
      confidence: "low",
      reasons: ["vad_segment has 5.0% WER on this corpus."],
      confidenceNotes: ["Fewer than 10 clips makes per-strategy differences easy to overfit."],
    },
    warnings: [{ code: "tiny_corpus", severity: "warning", message: "Only 2 clips were evaluated." }],
    normalizationPolicy: {
      werPolicy: "WER lowercases text and removes punctuation.",
      overlapAgreementPolicy: "Overlap-agree strips Unicode punctuation for agreement only.",
    },
    ...over,
  };
}

beforeEach(() => vi.clearAllMocks());
afterEach(cleanup);

describe("EvalReportView", () => {
  it("shows the empty prompt before a run", () => {
    renderWithProviders(<EvalReportView />);
    expect(screen.getByText(strings.dictationStudio.reportEmpty)).toBeInTheDocument();
  });

  it("runs the eval and renders the comparison table", async () => {
    runEval.mockResolvedValue(report({ perStrategy: [row("batch"), row("overlap_agree")] }));
    const user = userEvent.setup();
    renderWithProviders(<EvalReportView />);
    await user.click(screen.getByTestId(selectors.dictationStudio.runEval));

    const table = await screen.findByTestId(selectors.dictationStudio.evalTable);
    const batchRow = within(table).getByTestId(selectors.dictationStudio.evalRow({ strategy: "batch" }));
    // WER renders as a percentage (0.1 -> "10.0%").
    expect(within(batchRow).getByText(/10\.0%/)).toBeInTheDocument();
    expect(within(table).getByTestId(selectors.dictationStudio.evalRow({ strategy: "overlap_agree" }))).toBeInTheDocument();
    expect(runEval).toHaveBeenCalledWith(expect.objectContaining({ realtimeRepeats: 0 }));
  });

  it("renders recommendation, glossary, warnings, deltas, and per-clip diffs", async () => {
    runEval.mockResolvedValue(report());
    const user = userEvent.setup();
    renderWithProviders(<EvalReportView />);

    await user.click(screen.getByTestId(selectors.dictationStudio.runEval));

    expect(await screen.findByTestId(selectors.dictationStudio.evalSummary)).toHaveTextContent("Prefer vad_segment");
    expect(screen.getByText(/Only 2 clips/)).toBeInTheDocument();
    expect(screen.getByText(strings.dictationStudio.metricGlossary)).toBeInTheDocument();
    expect(screen.getByText(/WER lowercases/)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.dictationStudio.evalClips)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.dictationStudio.evalClip({ strategy: "batch", clipId: "clip-1" }))).toHaveTextContent("50.0%");
  });

  it("passes requested real-time repeats when latency is enabled", async () => {
    runEval.mockResolvedValue(report({ perStrategy: [row("batch")], latencyMeasured: true }));
    const user = userEvent.setup();
    renderWithProviders(<EvalReportView />);

    const repeats = screen.getByTestId(selectors.dictationStudio.repeatsInput);
    await user.clear(repeats);
    await user.type(repeats, "2");
    await user.click(screen.getByTestId(selectors.dictationStudio.runEval));

    await screen.findByTestId(selectors.dictationStudio.evalTable);
    expect(runEval).toHaveBeenCalledWith(expect.objectContaining({ realtimeRepeats: 2 }));
  });

  it("dashes the latency columns when latency was not measured", async () => {
    runEval.mockResolvedValue(report({ perStrategy: [row("batch")], latencyMeasured: false }));
    const user = userEvent.setup();
    renderWithProviders(<EvalReportView />);
    await user.click(screen.getByTestId(selectors.dictationStudio.runEval));

    await screen.findByTestId(selectors.dictationStudio.evalTable);
    expect(screen.getByText(strings.dictationStudio.latencyNotMeasured)).toBeInTheDocument();
    // p50 + p95 both collapse to the em-dash.
    expect(screen.getAllByText("—").length).toBeGreaterThanOrEqual(2);
  });

  it("surfaces a run error", async () => {
    runEval.mockRejectedValue(new Error("boom"));
    const user = userEvent.setup();
    renderWithProviders(<EvalReportView />);
    await user.click(screen.getByTestId(selectors.dictationStudio.runEval));
    await waitFor(() => expect(screen.getByText(strings.dictationStudio.reportError)).toBeInTheDocument());
  });
});
