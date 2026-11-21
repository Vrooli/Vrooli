/**
 * Additional OneshotTry coverage — loading/result/error states (~41% → ~95%).
 *
 * We stub navigator.mediaDevices so the real MediaRecorder pipeline can be
 * controlled in tests. The component calls getUserMedia, constructs a
 * MediaRecorder, starts it, and uploads on stop.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, fireEvent, act } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { makeApiError } from "../../api/client";
import { strings } from "../../consts/strings";

vi.mock("../../audio-integration", () => ({
  MicReadinessIndicator: ({ state }: { state: string }) => (
    <span data-testid="mic-readiness" data-state={state} />
  ),
}));

vi.mock("./useMicPermission", () => ({
  useMicPermission: () => "granted",
}));

const transcribeMock = vi.fn();
vi.mock("../../services/diagnostics", () => ({
  transcribe: (file: unknown) => transcribeMock(file),
}));

import { OneshotTry } from "./OneshotTry";

// ---------------------------------------------------------------------------
// Fake MediaRecorder + MediaStream
// ---------------------------------------------------------------------------
class FakeMediaRecorder {
  static isTypeSupported() { return true; }
  state: "inactive" | "recording" = "inactive";
  mimeType = "audio/webm";
  ondataavailable: ((ev: { data: { size: number } }) => void) | null = null;
  onstop: (() => void) | null = null;

  start() { this.state = "recording"; }
  stop() {
    this.state = "inactive";
    void Promise.resolve().then(() => this.onstop?.());
  }
}

vi.stubGlobal("MediaRecorder", FakeMediaRecorder);

const originalMR = global.MediaRecorder;

const fakeTrack = { stop: vi.fn() };
const fakeStream = { getTracks: () => [fakeTrack] };

beforeEach(() => {
  vi.clearAllMocks();
  fakeTrack.stop.mockReset();

  vi.stubGlobal("MediaRecorder", FakeMediaRecorder);

  Object.defineProperty(navigator, "mediaDevices", {
    value: { getUserMedia: vi.fn().mockResolvedValue(fakeStream) },
    configurable: true,
    writable: true,
  });
});

afterEach(() => {
  cleanup();
  vi.stubGlobal("MediaRecorder", originalMR);
});

describe("OneshotTry — recording flow", () => {
  it("transitions to recording state (aria-pressed=true) when Start is clicked", async () => {
    renderWithProviders(<OneshotTry onTrace={() => {}} />);
    const btn = screen.getByRole("button");
    await act(async () => { fireEvent.click(btn); await Promise.resolve(); });
    // After getUserMedia resolves, button should show "stop" copy (recording=true)
    expect(btn).toHaveAttribute("aria-pressed", "true");
  });

  it("shows the uploading copy (busy state) while transcribe is in-flight", async () => {
    let resolveTranscribe!: (v: unknown) => void;
    transcribeMock.mockReturnValue(new Promise((res) => { resolveTranscribe = res; }));

    renderWithProviders(<OneshotTry onTrace={() => {}} />);
    // Start recording
    await act(async () => {
      fireEvent.click(screen.getByRole("button"));
      await Promise.resolve();
    });
    // Stop recording → triggers onstop → upload starts
    await act(async () => {
      fireEvent.click(screen.getByRole("button"));
      await Promise.resolve();
    });
    // The busy state shows the uploading copy
    expect(screen.getByText(strings.diagnostics.oneshotUploading)).toBeInTheDocument();

    // Resolve inside act so the resulting state updates are flushed cleanly.
    await act(async () => {
      resolveTranscribe({ ok: true, data: { text: "hello", trace: {} } });
      await Promise.resolve();
    });
  });

  it("shows result text after successful transcription", async () => {
    const onTrace = vi.fn();
    transcribeMock.mockResolvedValue({
      ok: true,
      data: { text: "hello world", trace: { providerTier: "local", providerId: "whisper", modelId: "v3", latencyMs: 100 } },
    });

    renderWithProviders(<OneshotTry onTrace={onTrace} />);
    await act(async () => { fireEvent.click(screen.getByRole("button")); await Promise.resolve(); });
    await act(async () => {
      fireEvent.click(screen.getByRole("button"));
      await Promise.resolve();
    });
    // Drain all microtasks
    await act(async () => { await new Promise((r) => setTimeout(r, 0)); });
    expect(screen.getByText(/hello world/)).toBeInTheDocument();
    expect(onTrace).toHaveBeenCalledWith(
      expect.objectContaining({ providerTier: "local", providerId: "whisper" }),
    );
  });

  it("shows an ApiError via ApiErrorState when transcribe returns ok:false", async () => {
    const apiErr = makeApiError("internal", "server exploded", 500);
    transcribeMock.mockResolvedValue({ ok: false, error: apiErr });

    renderWithProviders(<OneshotTry onTrace={() => {}} />);
    await act(async () => { fireEvent.click(screen.getByRole("button")); await Promise.resolve(); });
    await act(async () => {
      fireEvent.click(screen.getByRole("button"));
      await Promise.resolve();
    });
    await act(async () => { await new Promise((r) => setTimeout(r, 0)); });
    // ApiErrorState renders the error message
    expect(screen.getByText(/server exploded/i)).toBeInTheDocument();
  });

  it("shows a string error message when getUserMedia is denied", async () => {
    (navigator.mediaDevices.getUserMedia as ReturnType<typeof vi.fn>).mockRejectedValue(
      new Error("Permission denied"),
    );
    renderWithProviders(<OneshotTry onTrace={() => {}} />);
    await act(async () => { fireEvent.click(screen.getByRole("button")); await Promise.resolve(); });
    await act(async () => { await Promise.resolve(); });
    expect(screen.getByText(/Permission denied/i)).toBeInTheDocument();
  });
});
