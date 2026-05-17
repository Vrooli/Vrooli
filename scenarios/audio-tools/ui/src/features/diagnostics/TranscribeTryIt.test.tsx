import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { TranscribeTryIt } from "./TranscribeTryIt";
import { strings } from "../../consts/strings";

// VoiceStreamProvider pulls in MediaRecorder + WebSocket on construction.
// Stub the audio-integration module so the unit test focuses on the
// three-mode UX rather than the streaming wire protocol (covered by Phase
// F integration tests).
vi.mock("../../audio-integration", () => ({
  VoiceStreamProvider: class {
    onResult: ((text: string) => void) | null = null;
    onError: ((message: string) => void) | null = null;
    onPartial: ((text: string) => void) | null = null;
    start = vi.fn().mockResolvedValue(undefined);
    stop = vi.fn();
    dispose = vi.fn();
  },
  MicReadinessIndicator: ({ state }: { state: string }) => (
    <span data-testid="mic-readiness" data-state={state} />
  ),
}));

// transcribe() hits the multipart REST endpoint; stub the service so the
// test doesn't need msw + a running audio-tools API.
vi.mock("../../services/diagnostics", () => ({
  transcribe: vi.fn().mockResolvedValue({
    ok: true,
    data: {
      text: "hello world",
      trace: { providerTier: "local", providerId: "whisper", modelId: "v3", latencyMs: 42 },
    },
  }),
}));

beforeEach(() => {
  vi.clearAllMocks();
  // jsdom does not implement permissions or MediaRecorder; the component
  // tolerates absence via try/catch — these tests do not exercise the
  // actual recording, only the mode switch and the upload-once flow.
});

describe("TranscribeTryIt", () => {
  it("defaults to Live mode and shows a Start button", () => {
    renderWithProviders(<TranscribeTryIt onTrace={() => {}} />);
    expect(screen.getByRole("button", { name: strings.diagnostics.liveStart })).toBeInTheDocument();
  });

  it("switches to One-shot mode and exposes a Record button", async () => {
    renderWithProviders(<TranscribeTryIt onTrace={() => {}} />);
    const user = userEvent.setup();
    await user.click(screen.getByRole("tab", { name: strings.diagnostics.tabOneshot }));
    expect(screen.getByRole("button", { name: strings.diagnostics.oneshotRecord })).toBeInTheDocument();
  });

  it("switches to File mode and exposes a file input + Transcribe button", async () => {
    renderWithProviders(<TranscribeTryIt onTrace={() => {}} />);
    const user = userEvent.setup();
    await user.click(screen.getByRole("tab", { name: strings.diagnostics.tabFile }));
    expect(screen.getByLabelText(strings.diagnostics.audioFileLabel)).toBeInTheDocument();
    // Transcribe button stays disabled until a file is selected.
    expect(screen.getByRole("button", { name: strings.diagnostics.transcribeAction })).toBeDisabled();
  });

  it("uploads a file and renders the final transcript + emits a trace", async () => {
    const onTrace = vi.fn();
    renderWithProviders(<TranscribeTryIt onTrace={onTrace} />);
    const user = userEvent.setup();
    await user.click(screen.getByRole("tab", { name: strings.diagnostics.tabFile }));

    const file = new File(["dummy"], "sample.wav", { type: "audio/wav" });
    const input = screen.getByLabelText(strings.diagnostics.audioFileLabel);
    await user.upload(input, file);

    await user.click(screen.getByRole("button", { name: strings.diagnostics.transcribeAction }));
    expect(await screen.findByText(/hello world/)).toBeInTheDocument();
    expect(onTrace).toHaveBeenCalledWith(
      expect.objectContaining({ providerTier: "local", providerId: "whisper" }),
    );
  });
});
