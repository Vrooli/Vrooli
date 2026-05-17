import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";

// TranscribeTryIt pulls in VoiceStreamProvider + MediaRecorder on construct.
// DiagnosticsPage tests focus on the page-level shell — the transcribe
// panel is fully covered in TranscribeTryIt.test.tsx.
vi.mock("./TranscribeTryIt", () => ({
  TranscribeTryIt: () => <div data-testid="stub-transcribe-tryit" />,
}));

vi.mock("../../services/diagnostics", () => ({
  summarize: vi.fn(),
  runSuite: vi.fn(),
  getLastSuiteRun: vi.fn(),
}));

vi.mock("../../services/tts", () => ({
  synthesize: vi.fn(),
  listVoices: vi.fn(),
}));

import { DiagnosticsPage } from "./DiagnosticsPage";
import { getLastSuiteRun, runSuite, summarize } from "../../services/diagnostics";
import { listVoices, synthesize } from "../../services/tts";
import { makeApiError } from "../../api/client";

beforeEach(() => {
  vi.mocked(listVoices).mockResolvedValue({ ok: true, data: [] });
  vi.mocked(getLastSuiteRun).mockResolvedValue({
    ok: true,
    data: {
      runId: "",
      startedAtUnixMs: 0,
      finishedAtUnixMs: 0,
      overall: "never",
      passCount: 0,
      failCount: 0,
      totalCount: 0,
      steps: [],
    },
  });
  vi.mocked(runSuite).mockResolvedValue({
    ok: true,
    data: {
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
    },
  });
  vi.mocked(summarize).mockResolvedValue({
    ok: true,
    data: {
      text: "the summary",
      promptTokens: 10,
      outputTokens: 5,
      trace: { providerTier: "local", providerId: "ollama", modelId: "llama3", latencyMs: 12 },
    },
  });
  vi.mocked(synthesize).mockResolvedValue({
    ok: true,
    data: {
      audio: new Uint8Array([1, 2, 3]),
      contentType: "audio/wav",
      providerTier: "local",
      providerId: "kokoro",
      modelId: "kokoro-v1",
      latencyMs: 8,
    },
  });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("DiagnosticsPage", () => {
  it("renders the page header and the SuiteCard above the per-capability panels", async () => {
    renderWithProviders(<DiagnosticsPage />);
    expect(await screen.findByText(strings.diagnostics.title)).toBeInTheDocument();
    expect(screen.getByText(strings.diagnostics.suite.title)).toBeInTheDocument();
    expect(screen.getByTestId("suite-run")).toBeInTheDocument();
  });

  it("renders the empty provider-trace state on first mount", async () => {
    renderWithProviders(<DiagnosticsPage />);
    expect(await screen.findByText(strings.diagnostics.traceEmpty)).toBeInTheDocument();
  });

  it("running the suite forwards each step's trace into the right-rail timeline", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DiagnosticsPage />);
    await user.click(screen.getByTestId("suite-run"));
    await waitFor(() => expect(vi.mocked(runSuite)).toHaveBeenCalled());
    await waitFor(() =>
      expect(screen.queryByText(strings.diagnostics.traceEmpty)).not.toBeInTheDocument(),
    );
  });

  it("renders an error envelope when summarize fails on the inline panel", async () => {
    vi.mocked(summarize).mockResolvedValue({
      ok: false,
      error: makeApiError("internal", "summarize-failed", 500),
    });
    const user = userEvent.setup();
    renderWithProviders(<DiagnosticsPage />);
    await user.type(
      screen.getByLabelText(strings.diagnostics.summarizeInputLabel),
      "text to summarize",
    );
    await user.click(screen.getByRole("button", { name: /diagnostics\.summarizeAction/i }));
    await waitFor(() => expect(screen.getByText(/summarize-failed/)).toBeInTheDocument());
  });
});
