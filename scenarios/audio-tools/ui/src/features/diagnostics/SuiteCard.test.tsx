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
      { capability: "stt", ok: true, errorCode: "", errorMessage: "", startedAtUnixMs: 1000, finishedAtUnixMs: 1010, providerTier: "local", providerId: "whisper", modelId: "base", latencyMs: 12, details: { quality_assessed: "false" } },
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

function qualityPassRun(): SuiteRun {
  const r = passRun();
  r.steps[0] = {
    ...r.steps[0]!,
    details: { quality_assessed: "true", quality_status: "pass" },
    quality: {
      assessed: true,
      status: "pass",
      hallucinationDetected: false,
      fixtures: [
        { fixtureId: "no_speech_silence", expectedKind: "no_speech", status: "pass", wer: 0, werThreshold: 0, filtered: true, filterReason: "hallucination", hallucinationDetected: false, preview: "", note: "" },
        { fixtureId: "clean_speech", expectedKind: "speech", status: "pass", wer: 0, werThreshold: 0.34, filtered: false, filterReason: "", hallucinationDetected: false, preview: "the quick brown fox jumps", note: "" },
      ],
    },
  };
  return r;
}

function qualityFailRun(): SuiteRun {
  const r = passRun();
  r.overall = "partial";
  r.passCount = 3;
  r.failCount = 1;
  r.steps[0] = {
    ...r.steps[0]!,
    ok: false,
    errorCode: "quality_smoke_failed",
    errorMessage: "STT quality smoke failed",
    details: { quality_assessed: "true", quality_status: "fail" },
    quality: {
      assessed: true,
      status: "fail",
      hallucinationDetected: true,
      fixtures: [
        { fixtureId: "no_speech_silence", expectedKind: "no_speech", status: "fail", wer: 0, werThreshold: 0, filtered: false, filterReason: "", hallucinationDetected: true, preview: "", note: "" },
        { fixtureId: "clean_speech", expectedKind: "speech", status: "pass", wer: 0, werThreshold: 0.34, filtered: false, filterReason: "", hallucinationDetected: false, preview: "the quick brown fox jumps", note: "" },
      ],
    },
  };
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
    expect(screen.getByText(strings.diagnostics.suite.qualityNotAssessed)).toBeInTheDocument();
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

  it("renders actionable labels for the honest-error codes (model_not_installed, invalid_input)", async () => {
    // Guards the honest-error contract end-to-end: the backend now emits
    // these distinct codes instead of "internal", and the UI must turn them
    // into actionable text rather than the bare code string.
    const r = passRun();
    r.overall = "partial";
    r.passCount = 2;
    r.failCount = 2;
    r.steps[2] = { ...r.steps[2]!, ok: false, errorCode: "model_not_installed", errorMessage: "model role not installed", providerTier: "", providerId: "" };
    r.steps[3] = { ...r.steps[3]!, ok: false, errorCode: "invalid_input", errorMessage: "ffmpeg rejected input", providerTier: "", providerId: "" };
    vi.mocked(runSuite).mockResolvedValue({ ok: true, data: r });
    const user = userEvent.setup();
    renderWithProviders(<SuiteCard />);
    await user.click(screen.getByTestId("suite-run"));
    await waitFor(() => expect(screen.getByText(strings.diagnostics.suite.overallPartial)).toBeInTheDocument());
    expect(screen.getByText(strings.diagnostics.suite.errorCodeModelNotInstalled)).toBeInTheDocument();
    expect(screen.getByText(strings.diagnostics.suite.errorCodeInvalidInput)).toBeInTheDocument();
  });

  it("renders the STT quality-smoke breakdown when quality was assessed", async () => {
    vi.mocked(runSuite).mockResolvedValue({ ok: true, data: qualityPassRun() });
    const user = userEvent.setup();
    renderWithProviders(<SuiteCard />);
    await user.click(screen.getByTestId("suite-run"));
    await waitFor(() => expect(screen.getByText(strings.diagnostics.suite.overallPass)).toBeInTheDocument());
    // The test i18n returns raw keys (no interpolation), so assert on keys.
    expect(screen.getByText(strings.diagnostics.suite.qualityStatusLabel)).toBeInTheDocument();
    expect(screen.getByText(strings.diagnostics.suite.qualityFixtureNoSpeech, { exact: false })).toBeInTheDocument();
    expect(screen.getByText(strings.diagnostics.suite.qualityFixtureSpeech, { exact: false })).toBeInTheDocument();
    // Quality was assessed, so the readiness-only "not assessed" note is gone.
    expect(screen.queryByText(strings.diagnostics.suite.qualityNotAssessed)).not.toBeInTheDocument();
  });

  it("keeps readiness distinct and flags the hallucination leak when quality fails", async () => {
    vi.mocked(runSuite).mockResolvedValue({ ok: true, data: qualityFailRun() });
    const user = userEvent.setup();
    renderWithProviders(<SuiteCard />);
    await user.click(screen.getByTestId("suite-run"));
    await waitFor(() => expect(screen.getByText(strings.diagnostics.suite.overallPartial)).toBeInTheDocument());
    expect(screen.getByText(strings.diagnostics.suite.qualityReadinessReachable)).toBeInTheDocument();
    expect(screen.getByText(strings.diagnostics.suite.qualityStatusLabel)).toBeInTheDocument();
    // The hallucination tag is concatenated into the failing fixture chip.
    expect(screen.getByText(strings.diagnostics.suite.qualityTagHallucination, { exact: false })).toBeInTheDocument();
  });

  it("renders the last run when getLastSuiteRun returns a populated envelope", async () => {
    vi.mocked(getLastSuiteRun).mockResolvedValue({ ok: true, data: passRun() });
    renderWithProviders(<SuiteCard />);
    await waitFor(() => expect(screen.getByText(strings.diagnostics.suite.overallPass)).toBeInTheDocument());
    expect(screen.getByTestId("suite-last-run")).toBeInTheDocument();
  });
});
