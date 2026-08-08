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
  onStatus: ((status: { code: string; message: string }) => void) | null;
  onDiagnostic: ((value: { state: string; durability: string; capturedSequence: number; sentSequence: number; processedSequence: number; terminalReason?: string }) => void) | null;
  lastTurn: { blob: Blob; mimeType: string; durationMs: number; capturedAt: number } | null;
  start: ReturnType<typeof vi.fn>;
  stop: ReturnType<typeof vi.fn>;
  dispose: ReturnType<typeof vi.fn>;
  getStream: () => MediaStream | null;
  getLastTurnAudio: () => FakeProvider["lastTurn"];
  exportDiagnostic: () => string;
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
    onStatus: ((status: { code: string; message: string }) => void) | null = null;
    onDiagnostic: FakeProvider["onDiagnostic"] = null;
    lastTurn: FakeProvider["lastTurn"] = null;
    start = vi.fn().mockResolvedValue(undefined);
    stop = vi.fn();
    dispose = vi.fn();
    getStream = vi.fn(() => fakeStream);
    getLastTurnAudio() {
      return this.lastTurn;
    }
    exportDiagnostic() {
      return JSON.stringify({ schemaVersion: 1, state: "failed" });
    }
    constructor() {
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
    expect(onCaptured).not.toHaveBeenCalled();
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
    Object.defineProperty(window, "requestAnimationFrame", { configurable: true, value: vi.fn(() => 1) });
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
      constructed[0]!.onDiagnostic?.({ state: "failed", durability: "persistent", capturedSequence: 4, sentSequence: 4, processedSequence: 3, terminalReason: "recovery_failed" });
    });
    expect(await screen.findByTestId(selectors.dictationStudio.turnDetails)).toHaveTextContent("dictationStudio.turnCapturedChunks5");
    expect(screen.getByTestId(selectors.dictationStudio.turnCaptureStatus)).toHaveAttribute("data-has-captured-audio", "true");
    expect(screen.getByTestId(selectors.dictationStudio.turnSentStatus)).toHaveAttribute("data-has-sent-audio", "true");
    expect(screen.getByTestId(selectors.dictationStudio.turnProcessedStatus)).toHaveAttribute("data-has-processed-audio", "true");
    expect(screen.getByTestId(selectors.dictationStudio.turnProcessedReady)).toBeInTheDocument();
    await user.click(screen.getByTestId(selectors.dictationStudio.exportDiagnostic));
    expect(createObjectURL).toHaveBeenCalledOnce();
    expect(click).toHaveBeenCalledOnce();
    expect(revokeObjectURL).toHaveBeenCalledWith("blob:diagnostic");
  });
});
