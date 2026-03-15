// DOC: docs/internal/SEAMS.md#text-to-speech-seam
import { useState, useEffect, useCallback, useRef } from "react";

/** Adapter interface for testability — wraps window.speechSynthesis. */
export interface SpeechSynthesisAdapter {
  getVoices(): SpeechSynthesisVoice[];
  speak(utterance: SpeechSynthesisUtterance): void;
  cancel(): void;
  pause(): void;
  resume(): void;
  readonly speaking: boolean;
  readonly paused: boolean;
  onvoiceschanged: (() => void) | null;
}

/** Default adapter wrapping the real browser API. */
export function createDefaultAdapter(): SpeechSynthesisAdapter | null {
  if (typeof window === "undefined" || !window.speechSynthesis) return null;
  const synth = window.speechSynthesis;
  return {
    getVoices: () => synth.getVoices(),
    speak: (u) => synth.speak(u),
    cancel: () => synth.cancel(),
    pause: () => synth.pause(),
    resume: () => synth.resume(),
    get speaking() { return synth.speaking; },
    get paused() { return synth.paused; },
    set onvoiceschanged(fn: (() => void) | null) { synth.onvoiceschanged = fn; },
  };
}

export interface TTSSettings {
  voice: string;  // SpeechSynthesisVoice.name, "" = browser default
  rate: number;   // 0.5 - 2.0
  pitch: number;  // 0.5 - 2.0
}

export interface TTSState {
  supported: boolean;
  isSpeaking: boolean;
  isPaused: boolean;
  voices: SpeechSynthesisVoice[];
  error: string | null;
}

export function useTextToSpeech(
  settings: TTSSettings,
  adapter?: SpeechSynthesisAdapter | null,
) {
  const resolvedAdapter = useRef(adapter ?? createDefaultAdapter());
  const [state, setState] = useState<TTSState>({
    supported: !!resolvedAdapter.current,
    isSpeaking: false,
    isPaused: false,
    voices: resolvedAdapter.current?.getVoices() ?? [],
    error: null,
  });
  const utteranceRef = useRef<SpeechSynthesisUtterance | null>(null);

  // Load voices (some browsers load asynchronously)
  useEffect(() => {
    const a = resolvedAdapter.current;
    if (!a) return;
    const loadVoices = () => {
      const voices = a.getVoices();
      if (voices.length > 0) {
        setState((s) => ({ ...s, voices }));
      }
    };
    loadVoices();
    a.onvoiceschanged = loadVoices;
    return () => { a.onvoiceschanged = null; };
  }, []);

  // Cleanup on unmount
  useEffect(() => {
    const adapter = resolvedAdapter.current;
    return () => { adapter?.cancel(); };
  }, []);

  const speak = useCallback((text: string) => {
    const a = resolvedAdapter.current;
    if (!a) return;

    a.cancel();
    const utterance = new SpeechSynthesisUtterance(text);
    utterance.rate = settings.rate;
    utterance.pitch = settings.pitch;

    if (settings.voice) {
      const match = a.getVoices().find((v) => v.name === settings.voice);
      if (match) utterance.voice = match;
    }

    utterance.onstart = () => setState((s) => ({ ...s, isSpeaking: true, isPaused: false, error: null }));
    utterance.onend = () => setState((s) => ({ ...s, isSpeaking: false, isPaused: false }));
    utterance.onerror = (e) => setState((s) => ({ ...s, isSpeaking: false, isPaused: false, error: (e as SpeechSynthesisErrorEvent).error ?? "Speech synthesis failed" }));
    utterance.onpause = () => setState((s) => ({ ...s, isPaused: true }));
    utterance.onresume = () => setState((s) => ({ ...s, isPaused: false }));

    utteranceRef.current = utterance;
    a.speak(utterance);
  }, [settings.rate, settings.pitch, settings.voice]);

  const pause = useCallback(() => { resolvedAdapter.current?.pause(); }, []);
  const resume = useCallback(() => { resolvedAdapter.current?.resume(); }, []);
  const stop = useCallback(() => {
    resolvedAdapter.current?.cancel();
    setState((s) => ({ ...s, isSpeaking: false, isPaused: false }));
  }, []);

  return { ...state, speak, pause, resume, stop };
}
