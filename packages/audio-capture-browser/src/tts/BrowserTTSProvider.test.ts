/**
 * Unit tests for BrowserTTSProvider.
 *
 * BrowserTTSProvider drives window.speechSynthesis and SpeechSynthesisUtterance.
 * Neither exists in jsdom, so we stub them globally before each test.
 *
 * The trick to driving async paths is capturing the utterance passed to
 * speechSynthesis.speak() and then firing its .onend / .onerror callbacks
 * from the test.
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { BrowserTTSProvider } from "./BrowserTTSProvider";

// ─── Fake Web Speech API ──────────────────────────────────────────────────────

interface FakeSpeechSynthesisVoice {
  name: string;
}

/** A minimal SpeechSynthesisUtterance whose callbacks the test can fire. */
class FakeSpeechSynthesisUtterance {
  /** Tracks the most recently constructed utterance. */
  static last: FakeSpeechSynthesisUtterance | null = null;

  text: string;
  rate = 1.0;
  pitch = 1.0;
  voice: FakeSpeechSynthesisVoice | null = null;
  onend: (() => void) | null = null;
  onerror: ((e: { error: string }) => void) | null = null;

  constructor(text: string) {
    this.text = text;
    // Use a static property to avoid a module-level `this` alias.
    FakeSpeechSynthesisUtterance.last = this;
  }
}

const mockSpeechSynthesis = {
  cancel: vi.fn(),
  speak: vi.fn().mockImplementation((u: FakeSpeechSynthesisUtterance) => {
    FakeSpeechSynthesisUtterance.last = u;
  }),
  getVoices: vi.fn().mockReturnValue([] as FakeSpeechSynthesisVoice[]),
  pause: vi.fn(),
  resume: vi.fn(),
};

// ─── suite ────────────────────────────────────────────────────────────────────

describe("BrowserTTSProvider", () => {
  beforeEach(() => {
    FakeSpeechSynthesisUtterance.last = null;
    vi.clearAllMocks();
    vi.stubGlobal("speechSynthesis", mockSpeechSynthesis);
    vi.stubGlobal("SpeechSynthesisUtterance", FakeSpeechSynthesisUtterance);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  // ── capabilities ────────────────────────────────────────────────────────────

  it("declares correct capabilities", () => {
    const p = new BrowserTTSProvider();
    expect(p.capabilities).toEqual({
      canPause: true,
      canSeek: false,
      canAdjustSpeed: false,
      canAdjustVolume: false,
    });
  });

  // ── unlock / isUnlocked ──────────────────────────────────────────────────────

  it("unlock() always resolves true", async () => {
    const p = new BrowserTTSProvider();
    expect(await p.unlock()).toBe(true);
    expect(await p.unlock(true)).toBe(true);
  });

  it("isUnlocked() always returns true", () => {
    const p = new BrowserTTSProvider();
    expect(p.isUnlocked()).toBe(true);
  });

  // ── speak – success ─────────────────────────────────────────────────────────

  it("speak() cancels any prior utterance and starts a new one", async () => {
    const p = new BrowserTTSProvider();
    const promise = p.speak("hello");
    expect(mockSpeechSynthesis.cancel).toHaveBeenCalled();
    expect(mockSpeechSynthesis.speak).toHaveBeenCalledTimes(1);
    FakeSpeechSynthesisUtterance.last!.onend!();
    await promise;
  });

  it("speak() resolves when onend fires", async () => {
    const p = new BrowserTTSProvider();
    const promise = p.speak("hello");
    expect(p.isSpeaking).toBe(true);
    FakeSpeechSynthesisUtterance.last!.onend!();
    await promise;
    expect(p.isSpeaking).toBe(false);
  });

  it("speak() rejects when onerror fires", async () => {
    const p = new BrowserTTSProvider();
    const promise = p.speak("hello");
    FakeSpeechSynthesisUtterance.last!.onerror!({ error: "network" });
    await expect(promise).rejects.toThrow("network");
    expect(p.isSpeaking).toBe(false);
  });

  it("speak() applies rate and pitch opts to utterance", async () => {
    const p = new BrowserTTSProvider();
    const promise = p.speak("hi", { rate: 1.5, pitch: 0.8 });
    expect(FakeSpeechSynthesisUtterance.last!.rate).toBe(1.5);
    expect(FakeSpeechSynthesisUtterance.last!.pitch).toBe(0.8);
    FakeSpeechSynthesisUtterance.last!.onend!();
    await promise;
  });

  it("speak() defaults rate to 1.0 and pitch to 1.0 when no opts", async () => {
    const p = new BrowserTTSProvider();
    const promise = p.speak("hi");
    expect(FakeSpeechSynthesisUtterance.last!.rate).toBe(1.0);
    expect(FakeSpeechSynthesisUtterance.last!.pitch).toBe(1.0);
    FakeSpeechSynthesisUtterance.last!.onend!();
    await promise;
  });

  it("speak() assigns matching voice to utterance", async () => {
    const fakeVoice: FakeSpeechSynthesisVoice = { name: "Google UK English Male" };
    mockSpeechSynthesis.getVoices.mockReturnValue([fakeVoice]);
    const p = new BrowserTTSProvider();
    const promise = p.speak("hi", { voice: "Google UK English Male" });
    expect(FakeSpeechSynthesisUtterance.last!.voice).toBe(fakeVoice);
    FakeSpeechSynthesisUtterance.last!.onend!();
    await promise;
  });

  it("speak() does not set voice when no match found", async () => {
    mockSpeechSynthesis.getVoices.mockReturnValue([{ name: "other" }]);
    const p = new BrowserTTSProvider();
    const promise = p.speak("hi", { voice: "nonexistent" });
    expect(FakeSpeechSynthesisUtterance.last!.voice).toBeNull();
    FakeSpeechSynthesisUtterance.last!.onend!();
    await promise;
  });

  it("speak() does not look up voices when no voice opt", async () => {
    const p = new BrowserTTSProvider();
    const promise = p.speak("hi");
    expect(mockSpeechSynthesis.getVoices).not.toHaveBeenCalled();
    FakeSpeechSynthesisUtterance.last!.onend!();
    await promise;
  });

  // ── isSpeaking ───────────────────────────────────────────────────────────────

  it("isSpeaking is false initially", () => {
    const p = new BrowserTTSProvider();
    expect(p.isSpeaking).toBe(false);
  });

  // ── stop ─────────────────────────────────────────────────────────────────────

  it("stop() calls speechSynthesis.cancel and resets state", async () => {
    const p = new BrowserTTSProvider();
    const promise = p.speak("hello");
    p.stop();
    expect(mockSpeechSynthesis.cancel).toHaveBeenCalledTimes(2); // once in speak, once in stop
    expect(p.isSpeaking).toBe(false);
    // Fire onend — promise is NOT resolved (stop already cleared state)
    FakeSpeechSynthesisUtterance.last!.onend!();
    // The speak promise is now orphaned but won't reject; just discard it.
    await promise;
  });

  // ── pause / resume ───────────────────────────────────────────────────────────

  it("pause() calls speechSynthesis.pause when speaking and not paused", async () => {
    const p = new BrowserTTSProvider();
    const promise = p.speak("hello");
    p.pause();
    expect(mockSpeechSynthesis.pause).toHaveBeenCalledTimes(1);
    FakeSpeechSynthesisUtterance.last!.onend!();
    await promise;
  });

  it("pause() is a no-op when not speaking", () => {
    const p = new BrowserTTSProvider();
    p.pause();
    expect(mockSpeechSynthesis.pause).not.toHaveBeenCalled();
  });

  it("pause() is a no-op when already paused", async () => {
    const p = new BrowserTTSProvider();
    const promise = p.speak("hello");
    p.pause(); // first pause
    p.pause(); // second pause — no-op
    expect(mockSpeechSynthesis.pause).toHaveBeenCalledTimes(1);
    FakeSpeechSynthesisUtterance.last!.onend!();
    await promise;
  });

  it("resume() calls speechSynthesis.resume when speaking and paused", async () => {
    const p = new BrowserTTSProvider();
    const promise = p.speak("hello");
    p.pause();
    p.resume();
    expect(mockSpeechSynthesis.resume).toHaveBeenCalledTimes(1);
    FakeSpeechSynthesisUtterance.last!.onend!();
    await promise;
  });

  it("resume() is a no-op when not speaking", () => {
    const p = new BrowserTTSProvider();
    p.resume();
    expect(mockSpeechSynthesis.resume).not.toHaveBeenCalled();
  });

  it("resume() is a no-op when speaking but not paused", async () => {
    const p = new BrowserTTSProvider();
    const promise = p.speak("hello");
    p.resume(); // not paused, no-op
    expect(mockSpeechSynthesis.resume).not.toHaveBeenCalled();
    FakeSpeechSynthesisUtterance.last!.onend!();
    await promise;
  });

  // ── getPlaybackState ─────────────────────────────────────────────────────────

  it("getPlaybackState() reflects paused state", async () => {
    const p = new BrowserTTSProvider();
    const promise = p.speak("hello");
    expect(p.getPlaybackState().isPaused).toBe(false);
    p.pause();
    expect(p.getPlaybackState().isPaused).toBe(true);
    FakeSpeechSynthesisUtterance.last!.onend!();
    await promise;
  });

  it("getPlaybackState() returns minimal fixed values", () => {
    const p = new BrowserTTSProvider();
    const s = p.getPlaybackState();
    expect(s.currentTime).toBe(0);
    expect(s.duration).toBeNull();
    expect(s.playbackRate).toBe(1);
    expect(s.volume).toBe(1);
    expect(s.isMuted).toBe(false);
  });

  // ── onProgress ───────────────────────────────────────────────────────────────

  it("onProgress() is a no-op (Web Speech API has no progress events)", () => {
    const p = new BrowserTTSProvider();
    const cb = vi.fn();
    // Should not throw
    p.onProgress(cb);
    p.onProgress(null);
    expect(cb).not.toHaveBeenCalled();
  });

  // ── dispose ──────────────────────────────────────────────────────────────────

  it("dispose() calls stop()", async () => {
    const p = new BrowserTTSProvider();
    const promise = p.speak("hello");
    p.dispose();
    expect(p.isSpeaking).toBe(false);
    FakeSpeechSynthesisUtterance.last!.onend!();
    await promise;
  });
});
