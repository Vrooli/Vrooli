/**
 * EvalsPanel tests — the search-quality baseline surface in isolation.
 *
 * Renders <EvalsPanel /> with the ./api/evals boundary mocked, so failures
 * point at eval-feature behaviour rather than transport. Copy is asserted via
 * the strings registry / test ids (cimode), never translated literals.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import {
  EvalSuiteSchema,
  EvalRunSchema,
  CompareRunsResponseSchema,
} from "@vrooli/proto-types/search-hub/v1/eval/eval_pb";

import { renderWithProviders } from "../../test-utils";

vi.mock("../../api/evals", () => ({
  listSuites: vi.fn(),
  listRuns: vi.fn(),
  getRun: vi.fn(),
  compareRuns: vi.fn(),
}));

import { EvalsPanel } from "./EvalsPanel";
import { selectors } from "../../consts/selectors";
import * as evalsApi from "../../api/evals";

const SUITE_ID = "cli-health.commands.primary";
// Case ids are provider data (not user-facing copy); consts satisfy the
// copy-driven-query lint rule while still asserting they thread to the DOM.
const CASE_STRONG = "restart";
const CASE_GIBBERISH = "gibberish-1";

const suite = () =>
  create(EvalSuiteSchema, {
    suiteId: SUITE_ID,
    providerId: "cli-health.commands",
    name: "CLI command discovery — primary",
    cases: [{ caseId: "restart", query: "restart" }],
    state: "active",
  });

const run = (runId: string, tag: string, strong: number, gibberish: number) =>
  create(EvalRunSchema, {
    runId,
    suiteId: SUITE_ID,
    tag,
    createdAt: `2026-06-04T12:0${runId.length}:00Z`,
    config: { rerankerLeg: "cross-encoder:bge", rerankEnabled: true },
    results: [
      { caseId: "restart", outcome: "met", observedTopScore: strong, expectedRank: 1 },
      { caseId: "gibberish-1", outcome: gibberish > 0.4 ? "unexpected_hit" : "met", observedTopScore: gibberish },
    ],
    aggregate: { cases: 2, met: 1, below: 0, meanStrongTop1: strong, maxGibberishScore: gibberish, latencyP95Ms: 12 },
  });

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("EvalsPanel", () => {
  it("lists suites, selects the first, and shows its run history with a trend", async () => {
    vi.mocked(evalsApi.listSuites).mockResolvedValue([suite()]);
    vi.mocked(evalsApi.listRuns).mockResolvedValue([
      run("run-2", "cross-encoder", 0.82, 0.05),
      run("run-1", "rerank-off", 0.8, 0.53),
    ]);

    renderWithProviders(<EvalsPanel />);

    // Suite list rendered + auto-selected → run history appears.
    await waitFor(() => {
      expect(screen.getByTestId(selectors.evals.runHistory)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.evals.suiteItem({ suiteId: SUITE_ID }))).toBeInTheDocument();
    // Two runs ⇒ a trend renders (>= 2 points).
    expect(within(screen.getByTestId(selectors.evals.trend)).getByTestId("evals-trend-inner")).toBeInTheDocument();
    expect(screen.getByTestId(selectors.evals.runRow({ runId: "run-1" }))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.evals.runRow({ runId: "run-2" }))).toBeInTheDocument();
  });

  it("shows the empty state when no suites are registered", async () => {
    vi.mocked(evalsApi.listSuites).mockResolvedValue([]);
    renderWithProviders(<EvalsPanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.evals.noSuites)).toBeInTheDocument();
    });
  });

  it("expands a run to its per-case table", async () => {
    vi.mocked(evalsApi.listSuites).mockResolvedValue([suite()]);
    vi.mocked(evalsApi.listRuns).mockResolvedValue([run("run-2", "cross-encoder", 0.82, 0.05)]);
    const user = userEvent.setup();

    renderWithProviders(<EvalsPanel />);
    const row = await screen.findByTestId(selectors.evals.runRow({ runId: "run-2" }));
    await user.click(within(row).getByRole("button"));

    // The per-case rows surface (caseId is data, asserted directly).
    expect(within(row).getByText(CASE_STRONG)).toBeInTheDocument();
    expect(within(row).getByText(CASE_GIBBERISH)).toBeInTheDocument();
  });

  it("compares two selected runs", async () => {
    vi.mocked(evalsApi.listSuites).mockResolvedValue([suite()]);
    vi.mocked(evalsApi.listRuns).mockResolvedValue([
      run("run-2", "cross-encoder", 0.82, 0.05),
      run("run-1", "rerank-off", 0.8, 0.53),
    ]);
    vi.mocked(evalsApi.compareRuns).mockResolvedValue(
      create(CompareRunsResponseSchema, {
        runA: run("run-1", "rerank-off", 0.8, 0.53),
        runB: run("run-2", "cross-encoder", 0.82, 0.05),
        deltas: [
          { caseId: "gibberish-1", outcomeA: "unexpected_hit", outcomeB: "met", topScoreA: 0.53, topScoreB: 0.05 },
        ],
      }),
    );
    const user = userEvent.setup();

    renderWithProviders(<EvalsPanel />);
    await screen.findByTestId(selectors.evals.runHistory);

    await user.click(screen.getByTestId(selectors.evals.runSelect({ runId: "run-1" })));
    await user.click(screen.getByTestId(selectors.evals.runSelect({ runId: "run-2" })));
    await user.click(screen.getByTestId(selectors.evals.compareButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.evals.compareResult)).toBeInTheDocument();
    });
    expect(vi.mocked(evalsApi.compareRuns)).toHaveBeenCalledWith("run-1", "run-2");
  });
});
