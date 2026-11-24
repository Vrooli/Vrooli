import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { act, cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";

interface FakeProvider {
  onResult: ((s: string) => void) | null;
  onError: ((s: string) => void) | null;
  onPartial: ((s: string) => void) | null;
  onSegmentFinal: ((s: string, index: number) => void) | null;
  onStatus: ((status: { code: string; message: string }) => void) | null;
  onDiagnostic: ((value: { state: string; durability: string; capturedSequence: number; sentSequence: number; processedSequence: number; doneSent?: boolean; retainedBytes: number; firstPartialLatencyMs: number | null; committedTextLagMs: number | null; terminalReason?: string }) => void) | null;
  lastTurn: { blob: Blob; mimeType: string; durationMs: number; capturedAt: number } | null;
  start: ReturnType<typeof vi.fn>;
  preConnect: ReturnType<typeof vi.fn>;
  markStopRequested: ReturnType<typeof vi.fn>;
  stop: ReturnType<typeof vi.fn>;
  dispose: ReturnType<typeof vi.fn>;
  getStream: () => MediaStream | null;
  getLastTurnAudio: () => FakeProvider["lastTurn"];
  exportDiagnostic: () => string;
  recordVisibleTranscriptRevision: ReturnType<typeof vi.fn>;
}

interface FakeProviderOptions {
  getUserMedia?: () => Promise<MediaStream>;
  captureFactory?: (
    stream: MediaStream,
    onFrame: (samples: Float32Array, sampleRate: number) => void,
  ) => Promise<{ stop(): void }>;
}

const constructed: FakeProvider[] = [];
let fakeStream: MediaStream | null = null;
let extractedWav = { pcm: new Uint8Array([1, 2, 3, 4]), sampleRateHz: 16_000 };
const originalAudioContext = window.AudioContext;
const originalRequestAnimationFrame = window.requestAnimationFrame;
const originalCancelAnimationFrame = window.cancelAnimationFrame;

vi.mock("../../audio-integration", () => ({
  PcmVoiceStreamProvider: class {
    onResult: ((s: string) => void) | null = null;
    onError: ((s: string) => void) | null = null;
    onPartial: ((s: string) => void) | null = null;
    onSegmentFinal: ((s: string, index: number) => void) | null = null;
    onStatus: ((status: { code: string; message: string }) => void) | null = null;
    onDiagnostic: FakeProvider["onDiagnostic"] = null;
    lastTurn: FakeProvider["lastTurn"] = null;
    private readonly options: FakeProviderOptions | undefined;
    start = vi.fn(async () => {
      if (!this.options) return;
      const stream = await this.options.getUserMedia?.();
      if (stream) await this.options.captureFactory?.(stream, () => {});
    });
    preConnect = vi.fn();
    markStopRequested = vi.fn();
    stop = vi.fn();
    dispose = vi.fn();
    recordVisibleTranscriptRevision = vi.fn();
    getStream = vi.fn(() => fakeStream);
    getLastTurnAudio() {
      return this.lastTurn;
    }
    exportDiagnostic() {
      return JSON.stringify({ schemaVersion: 1, state: "failed" });
    }
    constructor(options?: FakeProviderOptions) {
      this.options = options;
      constructed.push(this);
    }
  },
  MicReadinessIndicator: ({ state }: { state: string }) => (
    <span data-testid="mic-readiness" data-state={state} />
  ),
}));

vi.mock("../diagnostics/useMicPermission", () => ({ useMicPermission: () => "granted" }));

vi.mock("./audioWav", () => ({
  extractPcm16FromWav: () => extractedWav,
  pcm16DurationMs: () => 1_000,
}));

import { DictationRecorder } from "./DictationRecorder";

beforeEach(() => {
  constructed.length = 0;
  fakeStream = null;
  extractedWav = { pcm: new Uint8Array([1, 2, 3, 4]), sampleRateHz: 16_000 };
});
afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  vi.useRealTimers();
  Object.defineProperty(window, "AudioContext", { configurable: true, value: originalAudioContext });
  Object.defineProperty(window, "requestAnimationFrame", { configurable: true, value: originalRequestAnimationFrame });
  Object.defineProperty(window, "cancelAnimationFrame", { configurable: true, value: originalCancelAnimationFrame });
});

describe("DictationRecorder", () => {
  it("preconnects the stream after microphone permission is granted", async () => {
    renderWithProviders(<DictationRecorder onCaptured={() => {}} />);

    await waitFor(() => expect(constructed[0]!.preConnect).toHaveBeenCalledWith("en"));
  });

  it("does not activate an AudioContext before the provider acquires the microphone", async () => {
    const resume = vi.fn().mockResolvedValue(undefined);
    class SuspendedAudioContext {
      state: AudioContextState = "suspended";
      resume = resume;
      close = vi.fn().mockResolvedValue(undefined);
    }
    Object.defineProperty(window, "AudioContext", { configurable: true, value: SuspendedAudioContext });
    const user = userEvent.setup();
    renderWithProviders(<DictationRecorder onCaptured={() => {}} />);

    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));

    expect(resume).not.toHaveBeenCalled();
  });

  it("starts recording and flips the button to Stop", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DictationRecorder onCaptured={() => {}} />);
    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));
    await waitFor(() =>
      expect(screen.getByRole("button", { name: new RegExp(strings.dictationStudio.recordStop, "i") })).toBeInTheDocument(),
    );
    expect(constructed[0]!.start).toHaveBeenCalledTimes(1);
  });

  it("starts the transport handshake before microphone startup on the first press", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DictationRecorder onCaptured={() => {}} />);
    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));

    await waitFor(() => expect(constructed[0]!.start).toHaveBeenCalledOnce());
    expect(constructed[0]!.preConnect.mock.invocationCallOrder[1]).toBeLessThan(
      constructed[0]!.start.mock.invocationCallOrder[0]!,
    );
  });

  it("keeps unsettled interim text visible while a turn remains recording", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DictationRecorder onCaptured={() => {}} />);
    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));
    await waitFor(() => expect(constructed[0]).toBeTruthy());

    act(() => constructed[0]!.onPartial?.("the quick"));
    expect(screen.getByTestId(selectors.dictationStudio.interimTranscript)).toHaveTextContent("the quick");
    expect(constructed[0]!.recordVisibleTranscriptRevision).toHaveBeenCalledTimes(1);

    act(() => constructed[0]!.onPartial?.("the quick brown"));
    expect(screen.getByTestId(selectors.dictationStudio.interimTranscript)).toHaveTextContent("the quick brown");
    expect(constructed[0]!.recordVisibleTranscriptRevision).toHaveBeenCalledTimes(2);
    expect(screen.queryByText(/^the quick$/)).not.toBeInTheDocument();
  });

  it("stop transitions immediately to transcribing while waiting for the final result", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DictationRecorder onCaptured={() => {}} />);
    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));
    await waitFor(() =>
      expect(screen.getByRole("button", { name: new RegExp(strings.dictationStudio.recordStop, "i") })).toBeInTheDocument(),
    );

    await user.click(screen.getByRole("button", { name: new RegExp(strings.dictationStudio.recordStop, "i") }));

    expect(constructed[0]!.markStopRequested).toHaveBeenCalledTimes(1);
    expect(constructed[0]!.markStopRequested.mock.invocationCallOrder[0]).toBeLessThan(
      constructed[0]!.stop.mock.invocationCallOrder[0]!,
    );
    expect(constructed[0]!.stop).toHaveBeenCalledTimes(1);
    expect(screen.getByText(strings.dictationStudio.transcribing)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.dictationStudio.recordCancel)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.dictationStudio.recordStart)).toBeDisabled();
  });

  it("cancels a stuck transcription and releases the provider", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DictationRecorder onCaptured={() => {}} />);
    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));
    await waitFor(() => expect(constructed[0]).toBeTruthy());
    await user.click(screen.getByRole("button", { name: new RegExp(strings.dictationStudio.recordStop, "i") }));

    await user.click(screen.getByTestId(selectors.dictationStudio.recordCancel));

    expect(constructed[0]!.dispose).toHaveBeenCalledTimes(1);
    expect(screen.getByText(strings.dictationStudio.cancelled)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.dictationStudio.recordStart)).not.toBeDisabled();
  });

  it("ignores a late final result after cancellation", async () => {
    const onCaptured = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<DictationRecorder onCaptured={onCaptured} />);
    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));
    await waitFor(() => expect(constructed[0]).toBeTruthy());
    await user.click(screen.getByRole("button", { name: new RegExp(strings.dictationStudio.recordStop, "i") }));
    await user.click(screen.getByTestId(selectors.dictationStudio.recordCancel));

    const provider = constructed[0]!;
    provider.lastTurn = {
      blob: { arrayBuffer: () => Promise.resolve(new ArrayBuffer(8)) } as unknown as Blob,
      mimeType: "audio/wav",
      durationMs: 1_000,
      capturedAt: 0,
    };
    act(() => provider.onResult?.("late transcript"));
    await Promise.resolve();

    expect(onCaptured).not.toHaveBeenCalled();
    expect(screen.getByText(strings.dictationStudio.cancelled)).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.dictationStudio.finalTranscript)).not.toBeInTheDocument();
  });

  it("on result: extracts the retained PCM and reports the captured clip", async () => {
    const onCaptured = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<DictationRecorder onCaptured={onCaptured} />);
    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));
    await waitFor(() => expect(constructed[0]).toBeTruthy());

    constructed[0]!.lastTurn = {
      // jsdom's Blob lacks arrayBuffer(); stub it (extractPcm16FromWav is mocked).
      blob: { arrayBuffer: () => Promise.resolve(new ArrayBuffer(8)) } as unknown as Blob,
      mimeType: "audio/wav",
      durationMs: 0,
      capturedAt: 0,
    };
    act(() => {
      constructed[0]!.onResult?.("hello world");
    });

    await waitFor(() =>
      expect(onCaptured).toHaveBeenCalledWith(
        expect.objectContaining({ transcript: "hello world", sampleRateHz: 16_000 }),
      ),
    );
    const arg = onCaptured.mock.calls[0]![0] as { audio: Uint8Array };
    expect(Array.from(arg.audio)).toEqual([1, 2, 3, 4]);
  });

  it("preserves committed segment text when the terminal final envelope is empty", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DictationRecorder onCaptured={() => {}} />);
    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));
    await waitFor(() => expect(constructed[0]).toBeTruthy());

    act(() => {
      constructed[0]!.onSegmentFinal?.("a quick brown fox jumps.", 0);
      constructed[0]!.onResult?.("");
    });

    expect(screen.getByTestId(selectors.dictationStudio.finalTranscript)).toHaveTextContent("a quick brown fox jumps.");
  });

  it("joins committed segments without inserting a space before punctuation", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DictationRecorder onCaptured={() => {}} />);
    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));
    await waitFor(() => expect(constructed[0]).toBeTruthy());

    act(() => {
      constructed[0]!.onSegmentFinal?.("hello", 0);
      constructed[0]!.onSegmentFinal?.(".", 1);
    });

    expect(screen.getByTestId(selectors.dictationStudio.finalTranscript)).toHaveTextContent("hello.");
  });

  it("warns when the provider retained no audio for the turn", async () => {
    const onCaptured = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<DictationRecorder onCaptured={onCaptured} />);
    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));
    await waitFor(() => expect(constructed[0]).toBeTruthy());

    constructed[0]!.lastTurn = null;
    act(() => {
      constructed[0]!.onResult?.("ignored");
    });

    expect(await screen.findByText(strings.dictationStudio.captureMissing)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.dictationStudio.recordState)).toHaveAttribute("data-recorder-state", "failed");
    expect(onCaptured).not.toHaveBeenCalled();
  });

  it("passes the transcript to the composer when bounded replay audio is unavailable", async () => {
    const onTranscript = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<DictationRecorder onCaptured={() => {}} onTranscript={onTranscript} />);
    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));
    await waitFor(() => expect(constructed[0]).toBeTruthy());

    constructed[0]!.lastTurn = null;
    act(() => constructed[0]!.onResult?.("long turn transcript"));

    expect(onTranscript).toHaveBeenCalledWith("long turn transcript");
    expect(await screen.findByText(strings.dictationStudio.captureMissing)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.dictationStudio.recordStart)).not.toBeDisabled();
  });

  it("warns when the input stream never produces an audible level", async () => {
    vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout", "Date"] });
    const disconnect = vi.fn();
    class SilentAudioContext {
      resume = vi.fn().mockResolvedValue(undefined);
      createMediaStreamSource = vi.fn(() => ({ connect: vi.fn(), disconnect }));
      createAnalyser = vi.fn(() => ({
        fftSize: 0,
        frequencyBinCount: 4,
        getByteTimeDomainData: (data: Uint8Array) => data.fill(128),
        disconnect,
      }));
    }
    Object.defineProperty(window, "AudioContext", { configurable: true, value: SilentAudioContext });
    let animationFrames = 0;
    Object.defineProperty(window, "requestAnimationFrame", {
      configurable: true,
      value: vi.fn((callback: FrameRequestCallback) => {
        if (animationFrames++ === 0) callback(0);
        return 1;
      }),
    });
    Object.defineProperty(window, "cancelAnimationFrame", { configurable: true, value: vi.fn() });

    fakeStream = {} as MediaStream;
    renderWithProviders(<DictationRecorder onCaptured={() => {}} />);
    fireEvent.click(screen.getByTestId(selectors.dictationStudio.recordStart));
    await act(async () => {
      await Promise.resolve();
    });
    expect(constructed[0]).toBeTruthy();
    act(() => {
      vi.advanceTimersByTime(2_000);
    });

    expect(screen.getByText(strings.dictationStudio.noAudioDetected)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: new RegExp(strings.dictationStudio.recordStop, "i") })).toBeInTheDocument();
  });

  it("keeps per-frame silence analysis while limiting meter renders to 20 Hz", async () => {
    const disconnect = vi.fn();
    let analysisFrames = 0;
    class MeterAudioContext {
      resume = vi.fn().mockResolvedValue(undefined);
      createMediaStreamSource = vi.fn(() => ({ connect: vi.fn(), disconnect }));
      createAnalyser = vi.fn(() => ({
        fftSize: 0,
        frequencyBinCount: 4,
        getByteTimeDomainData: (data: Uint8Array) => {
          analysisFrames += 1;
          data.fill(analysisFrames === 1 ? 255 : 128);
        },
        disconnect,
      }));
    }
    Object.defineProperty(window, "AudioContext", { configurable: true, value: MeterAudioContext });
    const scheduledFrame: { current: FrameRequestCallback | null } = { current: null };
    Object.defineProperty(window, "requestAnimationFrame", {
      configurable: true,
      value: vi.fn((callback: FrameRequestCallback) => {
        scheduledFrame.current = callback;
        return 1;
      }),
    });
    Object.defineProperty(window, "cancelAnimationFrame", { configurable: true, value: vi.fn() });
    fakeStream = {} as MediaStream;

    renderWithProviders(<DictationRecorder onCaptured={() => {}} />);
    fireEvent.click(screen.getByTestId(selectors.dictationStudio.recordStart));
    await act(async () => { await Promise.resolve(); });

    const meter = screen.getByRole("meter");
    const changedValues = [meter.getAttribute("aria-valuenow")];
    for (const atMs of [0, 16, 32, 48, 64, 80, 96, 112]) {
      const frame = scheduledFrame.current;
      expect(frame).toBeTypeOf("function");
      act(() => frame?.(atMs));
      const value = meter.getAttribute("aria-valuenow");
      if (changedValues.at(-1) !== value) changedValues.push(value);
    }

    expect(analysisFrames).toBe(8);
    expect(changedValues).toEqual(["0", "100", "0"]);
  });

  it("surfaces provider errors", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DictationRecorder onCaptured={() => {}} />);
    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));
    await waitFor(() => expect(constructed[0]).toBeTruthy());
    act(() => {
      constructed[0]!.onError?.("mic-failed");
    });
    expect(await screen.findByText(/mic-failed/)).toBeInTheDocument();
		expect(screen.getByTestId(selectors.dictationStudio.recordError)).toHaveTextContent("mic-failed");

    await user.click(screen.getByRole("button", { name: new RegExp(strings.dictationStudio.retryRecording, "i") }));
    await waitFor(() => expect(constructed[0]!.start).toHaveBeenCalledTimes(2));
  });

  it("ignores virtual capture settings with an invalid sample count", async () => {
    const originalURL = window.location.href;
    window.history.replaceState({}, "", "/?stt_test_mode=1&stt_capture_source=virtual&stt_corpus_url=/corpus.wav&stt_virtual_samples=0");
    const fetchMock = vi.spyOn(globalThis, "fetch");
    const user = userEvent.setup();
    renderWithProviders(<DictationRecorder onCaptured={() => {}} />);
    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));
    await waitFor(() => expect(constructed[0]!.start).toHaveBeenCalledOnce());
    expect(fetchMock).not.toHaveBeenCalled();
    fetchMock.mockRestore();
    window.history.replaceState({}, "", originalURL);
  });

  it("ignores virtual capture mode when its corpus URL is missing", async () => {
    const originalURL = window.location.href;
    window.history.replaceState({}, "", "/?stt_test_mode=1&stt_capture_source=virtual&stt_virtual_samples=200");
    const fetchMock = vi.spyOn(globalThis, "fetch");
    const user = userEvent.setup();
    renderWithProviders(<DictationRecorder onCaptured={() => {}} />);
    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));
    await waitFor(() => expect(constructed[0]!.start).toHaveBeenCalledOnce());
    expect(fetchMock).not.toHaveBeenCalled();
    fetchMock.mockRestore();
    window.history.replaceState({}, "", originalURL);
  });

  it("starts the bounded virtual capture from bundle configuration", async () => {
    const originalURL = window.location.href;
    window.history.replaceState({}, "", "/?stt_test_mode=1&stt_capture_source=virtual&stt_corpus_url=/corpus.wav&stt_virtual_samples=200");
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue({
      ok: true,
      status: 200,
      arrayBuffer: () => Promise.resolve(new ArrayBuffer(2)),
    } as Response);
    const user = userEvent.setup();

    renderWithProviders(<DictationRecorder onCaptured={() => {}} />);
    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));
    await waitFor(() => expect(constructed[0]!.start).toHaveBeenCalledTimes(1));
    expect(fetchMock).toHaveBeenCalledWith("/corpus.wav", { credentials: "same-origin" });
    expect(screen.getByRole("button", { name: new RegExp(strings.dictationStudio.recordStop, "i") })).toBeInTheDocument();

    fetchMock.mockRestore();
    window.history.replaceState({}, "", originalURL);
  });

  it("surfaces durable recovery and queue status without developer tools", async () => {
    const queueMessage = "Waiting for the local Kyutai decoder.";
    const user = userEvent.setup();
    renderWithProviders(<DictationRecorder onCaptured={() => {}} />);
    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));
    await waitFor(() => expect(constructed[0]).toBeTruthy());
    act(() => {
      constructed[0]!.onStatus?.({ code: "queued", message: queueMessage });
    });
    expect(await screen.findByText(queueMessage)).toBeInTheDocument();
  });

  it("[REQ:ATD-P1-002] shows coverage and exports a metadata-only turn diagnostic", async () => {
    const user = userEvent.setup();
    const createObjectURL = vi.fn(() => "blob:diagnostic");
    const revokeObjectURL = vi.fn();
    Object.defineProperty(URL, "createObjectURL", { configurable: true, value: createObjectURL });
    Object.defineProperty(URL, "revokeObjectURL", { configurable: true, value: revokeObjectURL });
    const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => undefined);
    renderWithProviders(<DictationRecorder onCaptured={() => {}} />);
    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));
    await waitFor(() => expect(constructed[0]).toBeTruthy());
    act(() => {
      constructed[0]!.onDiagnostic?.({ state: "failed", durability: "persistent", capturedSequence: 4, sentSequence: 4, processedSequence: 3, retainedBytes: 3200, firstPartialLatencyMs: 42, committedTextLagMs: 80, terminalReason: "recovery_failed" });
    });
    expect(await screen.findByTestId(selectors.dictationStudio.turnDetails)).toHaveTextContent("dictationStudio.turnCapturedChunks5");
    expect(screen.getByTestId(selectors.dictationStudio.turnCaptureStatus)).toHaveAttribute("data-has-captured-audio", "true");
    expect(screen.getByTestId(selectors.dictationStudio.turnSentStatus)).toHaveAttribute("data-has-sent-audio", "true");
    expect(screen.getByTestId(selectors.dictationStudio.turnProcessedStatus)).toHaveAttribute("data-has-processed-audio", "true");
    expect(screen.getByTestId(selectors.dictationStudio.turnProcessedStatus)).toHaveAttribute("data-retained-bytes", "3200");
    expect(screen.getByTestId(selectors.dictationStudio.turnProcessedStatus)).toHaveAttribute("data-first-partial-latency-ms", "42");
    expect(screen.getByTestId(selectors.dictationStudio.turnProcessedStatus)).toHaveAttribute("data-committed-text-lag-ms", "80");
    expect(screen.getByTestId(selectors.dictationStudio.turnProcessedReady)).toBeInTheDocument();
    await user.click(screen.getByTestId(selectors.dictationStudio.exportDiagnostic));
    expect(createObjectURL).toHaveBeenCalledOnce();
    expect(click).toHaveBeenCalledOnce();
    expect(revokeObjectURL).toHaveBeenCalledWith("blob:diagnostic");
  });

  it("does not show processed readiness while the diagnostic is still recording", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DictationRecorder onCaptured={() => {}} />);
    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));
    await waitFor(() => expect(constructed[0]).toBeTruthy());

    act(() => constructed[0]!.onDiagnostic?.({
      state: "recording", durability: "persistent", capturedSequence: 2, sentSequence: 2,
      processedSequence: 2, retainedBytes: 800, firstPartialLatencyMs: null, committedTextLagMs: null,
    }));

    expect(screen.getByTestId(selectors.dictationStudio.turnDetails)).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.dictationStudio.turnProcessedReady)).not.toBeInTheDocument();
    expect(screen.getByTestId(selectors.dictationStudio.turnProcessedStatus)).toHaveAttribute("data-first-partial-latency-ms", "");
  });

  it("shows diagnostic readiness flags for an empty terminal turn", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DictationRecorder onCaptured={() => {}} />);
    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));
    await waitFor(() => expect(constructed[0]).toBeTruthy());
    act(() => constructed[0]!.onDiagnostic?.({
      state: "failed", durability: "volatile", capturedSequence: -1, sentSequence: -1,
      processedSequence: -1, doneSent: true, retainedBytes: 0,
      firstPartialLatencyMs: null, committedTextLagMs: null,
    }));
    expect(screen.getByTestId(selectors.dictationStudio.turnCaptureStatus)).toHaveAttribute("data-has-captured-audio", "false");
    expect(screen.getByTestId(selectors.dictationStudio.turnSentStatus)).toHaveAttribute("data-has-sent-audio", "false");
    expect(screen.getByTestId(selectors.dictationStudio.turnDoneStatus)).toHaveAttribute("data-done-sent", "true");
    expect(screen.getByTestId(selectors.dictationStudio.turnDoneReady)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.dictationStudio.turnProcessedStatus)).toHaveAttribute("data-has-processed-audio", "false");
    expect(screen.queryByTestId(selectors.dictationStudio.turnProcessedReady)).not.toBeInTheDocument();
  });

  it("keeps processed acknowledgements out of the connection status", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DictationRecorder onCaptured={() => {}} />);
    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));
    await waitFor(() => expect(constructed[0]).toBeTruthy());
    act(() => constructed[0]!.onStatus?.({ code: "queued", message: "Connected" }));
    expect(screen.getByTestId(selectors.dictationStudio.streamStatus)).toHaveTextContent("Connected");

    act(() => constructed[0]!.onStatus?.({ code: "processed_acknowledgement", message: "internal durability update" }));

    expect(screen.getByTestId(selectors.dictationStudio.streamStatus)).toHaveTextContent("Connected");
    expect(screen.queryByText(/internal durability update/)).not.toBeInTheDocument();
  });

  it("ignores stale provider events after cancelling transcription", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DictationRecorder onCaptured={() => {}} />);
    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));
    await waitFor(() => expect(constructed[0]).toBeTruthy());
    const provider = constructed[0]!;
    await user.click(screen.getByRole("button", { name: new RegExp(strings.dictationStudio.recordStop, "i") }));
    await user.click(screen.getByTestId(selectors.dictationStudio.recordCancel));

    act(() => {
      provider.onError?.("stale error");
      provider.onPartial?.("stale partial");
      provider.onSegmentFinal?.("stale segment", 0);
      provider.onStatus?.({ code: "queued", message: "stale status" });
      provider.onDiagnostic?.({ state: "failed", durability: "none", capturedSequence: -1, sentSequence: -1, processedSequence: -1, retainedBytes: 0, firstPartialLatencyMs: null, committedTextLagMs: null });
    });
    expect(screen.getByTestId(selectors.dictationStudio.recordState)).toHaveAttribute("data-recorder-state", "cancelled");
    expect(screen.queryByTestId(selectors.dictationStudio.recordError)).not.toBeInTheDocument();
    expect(screen.queryByTestId(selectors.dictationStudio.interimTranscript)).not.toBeInTheDocument();
    expect(screen.getByTestId(selectors.dictationStudio.streamStatus)).not.toHaveTextContent(/stale status/);
    expect(screen.queryByTestId(selectors.dictationStudio.turnDetails)).not.toBeInTheDocument();
  });

  it("does not warn when the level monitor observes audible input", async () => {
    vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout", "Date"] });
    class AudibleAudioContext {
      resume = vi.fn().mockResolvedValue(undefined);
      createMediaStreamSource = vi.fn(() => ({ connect: vi.fn(), disconnect: vi.fn() }));
      createAnalyser = vi.fn(() => ({
        fftSize: 0,
        frequencyBinCount: 4,
        getByteTimeDomainData: (data: Uint8Array) => data.fill(255),
        disconnect: vi.fn(),
      }));
    }
    Object.defineProperty(window, "AudioContext", { configurable: true, value: AudibleAudioContext });
    let sampled = false;
    Object.defineProperty(window, "requestAnimationFrame", { configurable: true, value: vi.fn((callback: FrameRequestCallback) => { if (!sampled) { sampled = true; callback(0); } return 1; }) });
    Object.defineProperty(window, "cancelAnimationFrame", { configurable: true, value: vi.fn() });
    fakeStream = {} as MediaStream;
    renderWithProviders(<DictationRecorder onCaptured={() => {}} />);
    fireEvent.click(screen.getByTestId(selectors.dictationStudio.recordStart));
    await act(async () => { await Promise.resolve(); });
    act(() => { vi.advanceTimersByTime(2_000); });
    expect(screen.queryByText(strings.dictationStudio.noAudioDetected)).not.toBeInTheDocument();
  });

  it("reports a capture read failure and permits another recording", async () => {
    const onCaptured = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<DictationRecorder onCaptured={onCaptured} />);
    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));
    await waitFor(() => expect(constructed[0]).toBeTruthy());
    constructed[0]!.lastTurn = {
      blob: { arrayBuffer: () => Promise.reject(new Error("unreadable")) } as unknown as Blob,
      mimeType: "audio/wav", durationMs: 0, capturedAt: 0,
    };

    act(() => constructed[0]!.onResult?.("transcript remains useful"));

    expect(await screen.findByText(strings.dictationStudio.captureMissing)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.dictationStudio.recordState)).toHaveAttribute("data-recorder-state", "failed");
    expect(screen.getByTestId(selectors.dictationStudio.finalTranscript)).toHaveTextContent("transcript remains useful");
    expect(onCaptured).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: new RegExp(strings.dictationStudio.retryRecording, "i") }));
    await waitFor(() => expect(constructed[0]!.start).toHaveBeenCalledTimes(2));
  });

  it("shows the fallback message when microphone startup rejects without an error message", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DictationRecorder onCaptured={() => {}} />);
    await waitFor(() => expect(constructed[0]).toBeTruthy());
    constructed[0]!.start.mockRejectedValueOnce({ message: "" });

    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));

    expect(await screen.findByTestId(selectors.dictationStudio.recordError)).toHaveTextContent(strings.dictationStudio.saveError);
    expect(screen.getByTestId(selectors.dictationStudio.recordState)).toHaveAttribute("data-recorder-state", "failed");
  });

  it("ignores retained audio that resolves after the user cancels", async () => {
    let resolveAudio!: (value: ArrayBuffer) => void;
    const onCaptured = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<DictationRecorder onCaptured={onCaptured} />);
    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));
    await waitFor(() => expect(constructed[0]).toBeTruthy());
    await user.click(screen.getByRole("button", { name: new RegExp(strings.dictationStudio.recordStop, "i") }));
    constructed[0]!.lastTurn = {
      blob: { arrayBuffer: () => new Promise<ArrayBuffer>((resolve) => { resolveAudio = resolve; }) } as unknown as Blob,
      mimeType: "audio/wav", durationMs: 0, capturedAt: 0,
    };
    act(() => constructed[0]!.onResult?.("cancelled transcript"));
    await waitFor(() => expect(resolveAudio).toBeTruthy());
    await user.click(screen.getByTestId(selectors.dictationStudio.recordCancel));
    await act(async () => { resolveAudio(new ArrayBuffer(8)); await Promise.resolve(); });
    expect(onCaptured).not.toHaveBeenCalled();
    expect(screen.getByTestId(selectors.dictationStudio.recordState)).toHaveAttribute("data-recorder-state", "cancelled");
  });

  it("ignores retained audio read failures after cancellation", async () => {
    let rejectAudio!: (reason: Error) => void;
    const onCaptured = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(<DictationRecorder onCaptured={onCaptured} />);
    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));
    await waitFor(() => expect(constructed[0]).toBeTruthy());
    await user.click(screen.getByRole("button", { name: new RegExp(strings.dictationStudio.recordStop, "i") }));
    constructed[0]!.lastTurn = {
      blob: { arrayBuffer: () => new Promise<ArrayBuffer>((_resolve, reject) => { rejectAudio = reject; }) } as unknown as Blob,
      mimeType: "audio/wav", durationMs: 0, capturedAt: 0,
    };
    act(() => constructed[0]!.onResult?.("cancelled transcript"));
    await waitFor(() => expect(rejectAudio).toBeTruthy());
    await user.click(screen.getByTestId(selectors.dictationStudio.recordCancel));
    await act(async () => { rejectAudio(new Error("cancelled read")); await Promise.resolve(); });
    expect(onCaptured).not.toHaveBeenCalled();
    expect(screen.getByTestId(selectors.dictationStudio.recordState)).toHaveAttribute("data-recorder-state", "cancelled");
    expect(screen.queryByText(strings.dictationStudio.captureMissing)).not.toBeInTheDocument();
  });

  it("recovers when creating the input level monitor throws", async () => {
    class BrokenAudioContext {
      resume = vi.fn().mockResolvedValue(undefined);
      createMediaStreamSource() { throw new Error("analyser unavailable"); }
    }
    Object.defineProperty(window, "AudioContext", { configurable: true, value: BrokenAudioContext });
    fakeStream = {} as MediaStream;
    const user = userEvent.setup();
    renderWithProviders(<DictationRecorder onCaptured={() => {}} />);

    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));

    expect(await screen.findByRole("button", { name: new RegExp(strings.dictationStudio.recordStop, "i") })).toBeInTheDocument();
    expect(screen.getByTestId(selectors.dictationStudio.audioMeter)).toHaveAttribute("aria-valuenow", "0");
  });
});
