import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";

vi.mock("../../services/diagnostics", () => ({
  runSuite: vi.fn(),
  getLastSuiteRun: vi.fn(),
}));

import { SuiteCard } from "./SuiteCard";
import { getLastSuiteRun, runSuite, type SuiteRun } from "../../services/diagnostics";

function emptyRun(): SuiteRun {
  return {
    runId: "",
    startedAtUnixMs: 0,
    finishedAtUnixMs: 0,
    overall: "never",
    passCount: 0,
    failCount: 0,
    totalCount: 0,
    steps: [],
  };
}

function passRun(): SuiteRun {
  return {
    runId: "run-1",
    startedAtUnixMs: 1000,
    finishedAtUnixMs: 1050,
    overall: "pass",
    passCount: 4,
    failCount: 0,
    totalCount: 4,
    steps: [
      { capability: "stt", ok: true, errorCode: "", errorMessage: "", startedAtUnixMs: 1000, finishedAtUnixMs: 1010, providerTier: "local", providerId: "whisper", modelId: "base", latencyMs: 12, details: {} },
      { capability: "tts", ok: true, errorCode: "", errorMessage: "", startedAtUnixMs: 1010, finishedAtUnixMs: 1020, providerTier: "local", providerId: "kokoro", modelId: "v1", latencyMs: 14, details: {} },
      { capability: "summarize", ok: true, errorCode: "", errorMessage: "", startedAtUnixMs: 1020, finishedAtUnixMs: 1030, providerTier: "local", providerId: "ollama", modelId: "l3", latencyMs: 17, details: {} },
      { capability: "transcode", ok: true, errorCode: "", errorMessage: "", startedAtUnixMs: 1030, finishedAtUnixMs: 1050, providerTier: "local", providerId: "ffmpeg", modelId: "", latencyMs: 4, details: {} },
    ],
  };
}

function partialRun(): SuiteRun {
  const r = passRun();
  r.overall = "partial";
  r.passCount = 3;
  r.failCount = 1;
  const first = r.steps[0]!;
  r.steps[0] = { ...first, ok: false, errorCode: "provider_unavailable", errorMessage: "no provider", providerTier: "", providerId: "" };
  return r;
}

beforeEach(() => {
  vi.mocked(getLastSuiteRun).mockResolvedValue({ ok: true, data: emptyRun() });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("SuiteCard", () => {
  it("renders the never-run state with the Run full diagnostics button", async () => {
    renderWithProviders(<SuiteCard />);
    expect(await screen.findByText(strings.diagnostics.suite.lastRunNever)).toBeInTheDocument();
    expect(screen.getByTestId("suite-run")).toBeInTheDocument();
    expect(screen.getByText(strings.diagnostics.suite.runAction)).toBeInTheDocument();
    // Four capability tiles, all in the never state.
    expect(screen.getByTestId("suite-tile-stt")).toBeInTheDocument();
    expect(screen.getByTestId("suite-tile-tts")).toBeInTheDocument();
    expect(screen.getByTestId("suite-tile-summarize")).toBeInTheDocument();
    expect(screen.getByTestId("suite-tile-transcode")).toBeInTheDocument();
  });

  it("runs the suite, flips overall to PASS, and forwards every step trace", async () => {
    vi.mocked(runSuite).mockResolvedValue({ ok: true, data: passRun() });
    const onTrace = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<SuiteCard onTrace={onTrace} />);
    await user.click(screen.getByTestId("suite-run"));
    await waitFor(() => expect(screen.getByText(strings.diagnostics.suite.overallPass)).toBeInTheDocument());
    expect(onTrace).toHaveBeenCalledTimes(4);
    expect(onTrace).toHaveBeenCalledWith("stt", expect.objectContaining({ providerId: "whisper" }));
  });

  it("renders the partial state when one capability fails", async () => {
    vi.mocked(runSuite).mockResolvedValue({ ok: true, data: partialRun() });
    const user = userEvent.setup();
    renderWithProviders(<SuiteCard />);
    await user.click(screen.getByTestId("suite-run"));
    await waitFor(() => expect(screen.getByText(strings.diagnostics.suite.overallPartial)).toBeInTheDocument());
    // STT tile surfaces the provider_unavailable label.
    expect(screen.getByText(strings.diagnostics.suite.errorCodeProviderUnavailable)).toBeInTheDocument();
  });

  it("renders the last run when getLastSuiteRun returns a populated envelope", async () => {
    vi.mocked(getLastSuiteRun).mockResolvedValue({ ok: true, data: passRun() });
    renderWithProviders(<SuiteCard />);
    await waitFor(() => expect(screen.getByText(strings.diagnostics.suite.overallPass)).toBeInTheDocument());
    expect(screen.getByTestId("suite-last-run")).toBeInTheDocument();
  });
});
