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
    runEval.mockResolvedValue({
      perStrategy: [row("batch"), row("overlap_agree")],
      qualityMeasured: true,
      latencyMeasured: false,
    });
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

  it("passes requested real-time repeats when latency is enabled", async () => {
    runEval.mockResolvedValue({
      perStrategy: [row("batch")],
      qualityMeasured: true,
      latencyMeasured: true,
    });
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
    runEval.mockResolvedValue({
      perStrategy: [row("batch")],
      qualityMeasured: true,
      latencyMeasured: false,
    });
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
