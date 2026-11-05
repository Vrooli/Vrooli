// Custom hook for live wake word testing.
// Encapsulates the state machine: idle → recording → comparing → result
// Uses the same MediaRecorder → decode → MFCC → DTW pipeline as enrollment.

import { useCallback, useEffect, useRef, useState } from "react";
import { acquireMicStream, releaseMicLease, type MicLease, type MicReleaseReason } from "../micOwnership";
import { WAKE_WORD_AUDIO_CONSTRAINTS } from "./types";
import type { AudioFeatures, EngineCalibration, WakeWordEngine } from "./types";

/** Lease reasons that mean the page/OS pulled the mic out from under us, so an
 *  in-flight test recording must be cancelled rather than processed. */
const LIFECYCLE_CANCEL_REASONS: ReadonlySet<MicReleaseReason> = new Set(["hidden", "pagehide", "freeze", "ended"]);

/** Single test attempt result. */
export interface TestAttempt {
  score: number;
  isMatch: boolean;
  timestamp: number;
}

export type TestStatus = "idle" | "recording" | "comparing" | "result";

export interface WakeWordTestState {
  status: TestStatus;
  currentResult: TestAttempt | null;
  history: TestAttempt[];
  error: string | null;
  recordingSeconds: number;
}

export interface UseWakeWordTestOpts {
  engine: WakeWordEngine;
  samples: AudioFeatures[];
  threshold: number;
  /** When true, startRecording is a no-op (mutual exclusion with enrollment). */
  disabled: boolean;
}

const MAX_HISTORY = 10;
const MIN_DURATION_MS = 500;
const MAX_DURATION_MS = 3000;
const MFCC_SAMPLE_RATE = 16000;

export interface UseWakeWordTestReturn {
  state: WakeWordTestState;
  startRecording: () => void;
  stopRecording: () => void;
  clearHistory: () => void;
}

export function useWakeWordTest(opts: UseWakeWordTestOpts): UseWakeWordTestReturn {
  const { engine, samples, threshold, disabled } = opts;

  const [status, setStatus] = useState<TestStatus>("idle");
  const [currentResult, setCurrentResult] = useState<TestAttempt | null>(null);
  const [history, setHistory] = useState<TestAttempt[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [recordingSeconds, setRecordingSeconds] = useState(0);

  const recorderRef = useRef<MediaRecorder | null>(null);
  const leaseRef = useRef<MicLease | null>(null);
  const chunksRef = useRef<Blob[]>([]);
  /** Set when the mic lease is pulled by page/OS lifecycle so onstop skips
   *  processing the (now silent / partial) capture. */
  const cancelledRef = useRef(false);
  const tickerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const autoStopRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const startTimeRef = useRef<number>(0);

  // Keep latest opts in refs so the onstop callback sees current values
  const engineRef = useRef(engine);
  const samplesRef = useRef(samples);
  const thresholdRef = useRef(threshold);
  // Calibration derived from the enrollment set, recomputed when samples change,
  // so the test button shows the same calibrated score the passive listener uses.
  const calibrationRef = useRef<EngineCalibration | null>(engine.calibrate?.(samples) ?? null);
  useEffect(() => { engineRef.current = engine; }, [engine]);
  useEffect(() => {
    samplesRef.current = samples;
    calibrationRef.current = engine.calibrate?.(samples) ?? null;
  }, [engine, samples]);
  useEffect(() => { thresholdRef.current = threshold; }, [threshold]);

  const cleanup = useCallback(() => {
    if (tickerRef.current) { clearInterval(tickerRef.current); tickerRef.current = null; }
    if (autoStopRef.current) { clearTimeout(autoStopRef.current); autoStopRef.current = null; }
    releaseMicLease(leaseRef.current, "manual-stop");
    leaseRef.current = null;
    recorderRef.current = null;
  }, []);

  const processRecording = useCallback(async (chunks: Blob[], durationMs: number) => {
    // Lifecycle cancellation (tab hidden / pagehide pulled the mic) — do not
    // process or surface a comparison; just return to idle.
    if (cancelledRef.current) {
      cancelledRef.current = false;
      setStatus("idle");
      return;
    }
    if (durationMs < MIN_DURATION_MS) {
      setError("Hold the button longer (at least 0.5s).");
      setStatus("idle");
      return;
    }

    setStatus("comparing");

    const blob = new Blob(chunks, { type: "audio/webm" });
    if (blob.size === 0) {
      setError("Recording was empty.");
      setStatus("idle");
      return;
    }

    try {
      const arrayBuf = await blob.arrayBuffer();
      const audioCtx = new AudioContext({ sampleRate: MFCC_SAMPLE_RATE });
      const decoded = await audioCtx.decodeAudioData(arrayBuf);
      const pcm = decoded.getChannelData(0);
      await audioCtx.close();

      const candidate = engineRef.current.extractFeatures(pcm, MFCC_SAMPLE_RATE);
      const result = engineRef.current.compareBest(
        candidate,
        samplesRef.current,
        thresholdRef.current,
        calibrationRef.current,
      );

      const attempt: TestAttempt = {
        score: result.score,
        isMatch: result.isMatch,
        timestamp: Date.now(),
      };

      setCurrentResult(attempt);
      setHistory((prev) => [attempt, ...prev].slice(0, MAX_HISTORY));
      setStatus("result");
    } catch (err) {
      setError(`Comparison failed: ${err}`);
      setStatus("idle");
    }
  }, []);

  const stopRecording = useCallback(() => {
    if (!recorderRef.current || recorderRef.current.state !== "recording") return;
    if (tickerRef.current) { clearInterval(tickerRef.current); tickerRef.current = null; }
    if (autoStopRef.current) { clearTimeout(autoStopRef.current); autoStopRef.current = null; }
    recorderRef.current.stop();
  }, []);

  const startRecording = useCallback(async () => {
    if (disabled || samples.length === 0) return;
    if (status === "recording" || status === "comparing") return;

    setError(null);
    setCurrentResult(null);
    setRecordingSeconds(0);
    cancelledRef.current = false;
    chunksRef.current = [];

    try {
      const lease = await acquireMicStream("wake-word-test", { audio: WAKE_WORD_AUDIO_CONSTRAINTS }, {
        onRelease: (reason) => {
          // Page/OS lifecycle pulled the mic mid-test → cancel processing.
          if (!LIFECYCLE_CANCEL_REASONS.has(reason)) return;
          cancelledRef.current = true;
          if (tickerRef.current) { clearInterval(tickerRef.current); tickerRef.current = null; }
          if (autoStopRef.current) { clearTimeout(autoStopRef.current); autoStopRef.current = null; }
          if (recorderRef.current?.state === "recording") recorderRef.current.stop();
          setStatus("idle");
        },
      });
      leaseRef.current = lease;
      const stream = lease.stream;

      const recorder = new MediaRecorder(stream, {
        mimeType: MediaRecorder.isTypeSupported("audio/webm;codecs=opus")
          ? "audio/webm;codecs=opus"
          : "audio/webm",
      });
      recorderRef.current = recorder;

      recorder.ondataavailable = (e) => {
        if (e.data.size > 0) chunksRef.current.push(e.data);
      };

      recorder.onerror = () => {
        cleanup();
        setError("Recording failed.");
        setStatus("idle");
      };

      recorder.onstop = () => {
        const durationMs = performance.now() - startTimeRef.current;
        const chunks = [...chunksRef.current];
        releaseMicLease(leaseRef.current, "manual-stop");
        leaseRef.current = null;
        void processRecording(chunks, durationMs);
      };

      startTimeRef.current = performance.now();
      recorder.start(250);
      setStatus("recording");

      tickerRef.current = setInterval(() => setRecordingSeconds((v) => v + 1), 1000);
      autoStopRef.current = setTimeout(() => stopRecording(), MAX_DURATION_MS);
    } catch (err) {
      cleanup();
      setError(`Mic access failed: ${err}`);
      setStatus("idle");
    }
  }, [disabled, samples.length, status, cleanup, processRecording, stopRecording]);

  const clearHistory = useCallback(() => {
    setHistory([]);
    setCurrentResult(null);
    setError(null);
  }, []);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (tickerRef.current) clearInterval(tickerRef.current);
      if (autoStopRef.current) clearTimeout(autoStopRef.current);
      releaseMicLease(leaseRef.current, "unmount");
      leaseRef.current = null;
    };
  }, []);

  return {
    state: { status, currentResult, history, error, recordingSeconds },
    startRecording: () => void startRecording(),
    stopRecording,
    clearHistory,
  };
}
