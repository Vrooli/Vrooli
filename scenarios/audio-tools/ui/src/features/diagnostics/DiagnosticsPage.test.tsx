import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";

// TranscribeTryIt pulls in VoiceStreamProvider + MediaRecorder on construct.
// DiagnosticsPage tests focus on the page-level shell + the other three
// try-it panels; the transcribe panel is fully covered in
// TranscribeTryIt.test.tsx.
vi.mock("./TranscribeTryIt", () => ({
  TranscribeTryIt: () => <div data-testid="stub-transcribe-tryit" />,
}));

vi.mock("../../services/diagnostics", () => ({
  summarize: vi.fn(),
}));

vi.mock("../../services/tts", () => ({
  synthesize: vi.fn(),
  listVoices: vi.fn(),
}));

import { DiagnosticsPage } from "./DiagnosticsPage";
import { summarize } from "../../services/diagnostics";
import { listVoices, synthesize } from "../../services/tts";
import { makeApiError } from "../../api/client";

beforeEach(() => {
  vi.mocked(listVoices).mockResolvedValue({ ok: true, data: [] });
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
  it("renders the page header and a tablist for the four try-its", async () => {
    renderWithProviders(<DiagnosticsPage />);
    expect(await screen.findByText("diagnostics.title")).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "diagnostics.tabTranscribe" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "diagnostics.tabSynthesize" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "diagnostics.tabSummarize" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "diagnostics.tabTranscode" })).toBeInTheDocument();
  });

  it("renders the empty provider-trace state on first mount", async () => {
    renderWithProviders(<DiagnosticsPage />);
    expect(await screen.findByText("diagnostics.traceEmpty")).toBeInTheDocument();
  });

  it("renders an error envelope when summarize fails", async () => {
    vi.mocked(summarize).mockResolvedValue({
      ok: false,
      error: makeApiError("internal", "summarize-failed", 500),
    });
    const user = userEvent.setup();
    renderWithProviders(<DiagnosticsPage />);
    await user.click(screen.getByRole("tab", { name: "diagnostics.tabSummarize" }));
    await user.type(
      screen.getByLabelText("diagnostics.summarizeInputLabel"),
      "text to summarize",
    );
    await user.click(screen.getByRole("button", { name: /diagnostics\.summarizeAction/i }));
    await waitFor(() => expect(screen.getByText(/summarize-failed/)).toBeInTheDocument());
  });

  it("calls summarize() exactly once when the primary CTA is invoked with text", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DiagnosticsPage />);
    await user.click(screen.getByRole("tab", { name: "diagnostics.tabSummarize" }));
    await user.type(
      screen.getByLabelText("diagnostics.summarizeInputLabel"),
      "hello world",
    );
    await user.click(screen.getByRole("button", { name: /diagnostics\.summarizeAction/i }));
    await waitFor(() => {
      expect(vi.mocked(summarize)).toHaveBeenCalledTimes(1);
    });
    expect(vi.mocked(summarize)).toHaveBeenCalledWith("hello world", "moderate");
  });
});
