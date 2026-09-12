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
  onDiagnostic: ((value: { state: string; durability: string; capturedSequence: number; sentSequence: number; processedSequence: number; retainedBytes: number; firstPartialLatencyMs: number | null; committedTextLagMs: number | null; terminalReason?: string }) => void) | null;
  lastTurn: { blob: Blob; mimeType: string; durationMs: number; capturedAt: number } | null;
  start: ReturnType<typeof vi.fn>;
  stop: ReturnType<typeof vi.fn>;
  dispose: ReturnType<typeof vi.fn>;
  getStream: () => MediaStream | null;
  getLastTurnAudio: () => FakeProvider["lastTurn"];
  exportDiagnostic: () => string;
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
    stop = vi.fn();
    dispose = vi.fn();
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
  extractPcm16FromWav: () => ({ pcm: new Uint8Array([1, 2, 3, 4]), sampleRateHz: 16_000 }),
  pcm16DurationMs: () => 1_000,
}));

import { DictationRecorder } from "./DictationRecorder";

beforeEach(() => {
  constructed.length = 0;
  fakeStream = null;
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

  it("keeps unsettled interim text visible while a turn remains recording", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DictationRecorder onCaptured={() => {}} />);
    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));
    await waitFor(() => expect(constructed[0]).toBeTruthy());

    act(() => constructed[0]!.onPartial?.("the quick"));
    expect(screen.getByTestId(selectors.dictationStudio.interimTranscript)).toHaveTextContent("the quick");

    act(() => constructed[0]!.onPartial?.("the quick brown"));
    expect(screen.getByTestId(selectors.dictationStudio.interimTranscript)).toHaveTextContent("the quick brown");
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
      durationMs: 1_000,
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
    vi.useFakeTimers();
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

  it("starts the bounded virtual capture from bundle configuration", async () => {
    const originalURL = window.location.href;
    window.history.replaceState({}, "", "/?stt_test_mode=1&stt_capture_source=virtual&stt_corpus_url=/corpus.wav&stt_virtual_samples=200");
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue({
      ok: true,
      status: 200,
      arrayBuffer: async () => new ArrayBuffer(2),
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
    const user = userEvent.setup();
    renderWithProviders(<DictationRecorder onCaptured={() => {}} />);
    await user.click(screen.getByTestId(selectors.dictationStudio.recordStart));
    await waitFor(() => expect(constructed[0]).toBeTruthy());
    act(() => {
      constructed[0]!.onStatus?.({ code: "queued", message: "Waiting for the local Kyutai decoder." });
    });
    expect(await screen.findByText("Waiting for the local Kyutai decoder.")).toBeInTheDocument();
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
});
