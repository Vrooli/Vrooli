import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../../test-utils";
import { strings } from "../../../consts/strings";

vi.mock("../../../services/diagnostics", () => ({ getLastSuiteRun: vi.fn(), runSuite: vi.fn() }));

import { SuiteCard } from "./SuiteCard";
import { getLastSuiteRun, runSuite, type SuiteRun } from "../../../services/diagnostics";

const run: SuiteRun = {
  runId: "run-1", startedAtUnixMs: 1_000, finishedAtUnixMs: 1_010,
  overall: "partial", passCount: 1, failCount: 1, totalCount: 2,
  steps: [
    { capability: "stt", ok: false, errorCode: "provider_unavailable", errorMessage: "offline", startedAtUnixMs: 1_000, finishedAtUnixMs: 1_005, providerTier: "", providerId: "", modelId: "", latencyMs: 5, details: {} },
    { capability: "tts", ok: true, errorCode: "", errorMessage: "", startedAtUnixMs: 1_005, finishedAtUnixMs: 1_010, providerTier: "local", providerId: "voice", modelId: "v1", latencyMs: 4.7, details: {} },
  ],
};

beforeEach(() => {
  vi.mocked(getLastSuiteRun).mockResolvedValue({ ok: true, data: { ...run, runId: "", overall: "never" } });
});
afterEach(() => { cleanup(); vi.clearAllMocks(); });

describe("diagnostics SuiteCard", () => {
  it("shows last-run failures expanded and lets the operator collapse the log and select a capability", async () => {
    vi.mocked(getLastSuiteRun).mockResolvedValue({ ok: true, data: run });
    const onTileClick = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<SuiteCard onTileClick={onTileClick} />);

    expect(await screen.findByText(strings.diagnostics.suite.overallPartial)).toBeInTheDocument();
    expect(screen.getByTestId("suite-last-run")).toBeInTheDocument();
    expect(screen.getByText(strings.diagnostics.suite.errorCodeProviderUnavailable)).toBeInTheDocument();
    expect(screen.getByText(/provider_unavailable: offline/)).toBeInTheDocument();
    expect(screen.getByText(strings.diagnostics.suite.logToggleClose)).toBeInTheDocument();

    await user.click(screen.getByTestId("suite-tile-stt"));
    expect(onTileClick).toHaveBeenCalledWith("stt", true);
    await user.click(screen.getByText(strings.diagnostics.suite.logToggleClose));
    await waitFor(() => expect(screen.getByText(strings.diagnostics.suite.logToggleOpen)).toBeInTheDocument());
  });

  it("reports a failed request and disables capability tiles without a click handler", async () => {
    vi.mocked(runSuite).mockResolvedValue({ ok: false, error: { message: "network down" } } as never);
    const user = userEvent.setup();
    renderWithProviders(<SuiteCard />);
    expect(screen.getByTestId("suite-tile-stt")).toBeDisabled();
    await user.click(screen.getByTestId("suite-run"));
    expect(await screen.findByText(strings.diagnostics.suite.runFailed)).toBeInTheDocument();
    expect(screen.getByText(strings.diagnostics.suite.runAction)).toBeInTheDocument();
  });

  it("forwards only steps with provider metadata on a successful run", async () => {
    const noProviderRun = { ...run, overall: "pass" as const, steps: [
      { ...run.steps[0]!, ok: true, errorCode: "", errorMessage: "", providerTier: "", providerId: "" }, run.steps[1]!,
    ] };
    vi.mocked(runSuite).mockResolvedValue({ ok: true, data: noProviderRun });
    const onTrace = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<SuiteCard onTrace={onTrace} />);
    await user.click(screen.getByTestId("suite-run"));
    await waitFor(() => expect(screen.getByText(strings.diagnostics.suite.overallPass)).toBeInTheDocument());
    expect(onTrace).toHaveBeenCalledTimes(1);
    expect(onTrace).toHaveBeenCalledWith("tts", expect.objectContaining({ providerId: "voice", latencyMs: 4.7 }));
  });

  it("uses the error message when a failure has no recognized code", async () => {
    const unknownFailure = { ...run, steps: [
      { ...run.steps[0]!, errorCode: "future_error", errorMessage: "try again" },
    ] };
    vi.mocked(getLastSuiteRun).mockResolvedValue({ ok: true, data: unknownFailure });
    renderWithProviders(<SuiteCard />);

    expect(await screen.findByText(/future_error: try again/)).toBeInTheDocument();
  });

  it("shows quality fixture outcomes and keeps quality failures distinct from readiness", async () => {
    const qualityRun: SuiteRun = {
      ...run,
      steps: [{ ...run.steps[0]!, quality: {
        assessed: true,
        status: "fail",
        hallucinationDetected: true,
        fixtures: [
          { fixtureId: "speech-case", expectedKind: "speech", status: "fail", wer: 0.4, werThreshold: 0.2, hallucinationDetected: true, filtered: false, filterReason: "", preview: "bad transcript", note: "hallucinated" },
          { fixtureId: "silence-case", expectedKind: "no_speech", status: "warn", wer: 0, werThreshold: 0.2, hallucinationDetected: false, filtered: true, filterReason: "empty", preview: "", note: "filtered" },
        ],
      } }],
    };
    vi.mocked(getLastSuiteRun).mockResolvedValue({ ok: true, data: qualityRun });
    renderWithProviders(<SuiteCard />);

    expect(await screen.findByText(strings.diagnostics.suite.qualityReadinessReachable)).toBeInTheDocument();
    expect(screen.getByTitle(qualityRun.steps[0]!.quality!.fixtures[0]!.note)).toHaveTextContent(strings.diagnostics.suite.qualityTagHallucination);
    expect(screen.getByTitle(qualityRun.steps[0]!.quality!.fixtures[1]!.note)).toHaveTextContent(strings.diagnostics.suite.qualityTagFiltered);
    expect(screen.getByText(strings.diagnostics.suite.qualityDeeperHint)).toBeInTheDocument();
  });
});
