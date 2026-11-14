import { describe, it, expect, beforeEach } from "vitest";
import { ttsPlaybackRegistry } from "./playbackRegistry";
import type { TTSPlaybackCapabilities, TTSProvider } from "./types";

// Minimal TTSProvider stand-in for registry unit tests. `finish()` simulates a
// natural tail-completion (what KokoroProvider.cleanup / BrowserTTSProvider.settle
// do): flip idle and fire onSettled.
class FakeProvider implements TTSProvider {
  disposed = false;
  private speaking: boolean;
  onSettled: (() => void) | null = null;
  readonly capabilities: TTSPlaybackCapabilities = {
    canPause: false,
    canSeek: false,
    canAdjustSpeed: false,
    canAdjustVolume: false,
  };

  constructor(speaking = true) {
    this.speaking = speaking;
  }

  get isSpeaking(): boolean {
    return this.speaking;
  }
  async speak(): Promise<void> {}
  stop(): void {
    this.speaking = false;
    this.onSettled?.();
  }
  dispose(): void {
    this.disposed = true;
  }
  async unlock(): Promise<boolean> {
    return true;
  }
  isUnlocked(): boolean {
    return true;
  }

  /** Simulate the audio tail finishing on its own. */
  finish(): void {
    this.speaking = false;
    this.onSettled?.();
  }
}

describe("ttsPlaybackRegistry", () => {
  beforeEach(() => {
    ttsPlaybackRegistry._resetForTests();
  });

  it("installs a provider as the live owner for a key", () => {
    const p = new FakeProvider();
    ttsPlaybackRegistry.install("s1", p, "kokoro");
    expect(ttsPlaybackRegistry.has("s1")).toBe(true);
    expect(ttsPlaybackRegistry.backendOf("s1")).toBe("kokoro");
    expect(ttsPlaybackRegistry.isOrphaned("s1")).toBe(false);
    expect(p.disposed).toBe(false);
  });

  it("keeps a speaking provider alive on release, then disposes it when its tail settles", () => {
    const p = new FakeProvider(true);
    ttsPlaybackRegistry.install("s1", p, "kokoro");

    // Pane unmounts mid-playback → hand off, do not dispose.
    ttsPlaybackRegistry.release("s1", p, { keepAliveIfSpeaking: true });
    expect(p.disposed).toBe(false);
    expect(ttsPlaybackRegistry.isOrphaned("s1")).toBe(true);
    expect(ttsPlaybackRegistry.has("s1")).toBe(true);

    // Tail finishes while still orphaned → registry disposes it.
    p.finish();
    expect(p.disposed).toBe(true);
    expect(ttsPlaybackRegistry.has("s1")).toBe(false);
  });

  it("disposes immediately on release when the provider is idle", () => {
    const p = new FakeProvider(false);
    ttsPlaybackRegistry.install("s1", p, "kokoro");
    ttsPlaybackRegistry.release("s1", p, { keepAliveIfSpeaking: true });
    expect(p.disposed).toBe(true);
    expect(ttsPlaybackRegistry.has("s1")).toBe(false);
  });

  it("adopts a live provider on remount and cancels the pending settle-dispose", () => {
    const p = new FakeProvider(true);
    ttsPlaybackRegistry.install("s1", p, "kokoro");
    ttsPlaybackRegistry.release("s1", p, { keepAliveIfSpeaking: true });
    expect(ttsPlaybackRegistry.isOrphaned("s1")).toBe(true);

    const adopted = ttsPlaybackRegistry.adopt("s1", "kokoro");
    expect(adopted).toBe(p);
    expect(ttsPlaybackRegistry.isOrphaned("s1")).toBe(false);

    // After adoption, a natural tail-finish must NOT dispose (a live owner holds it).
    p.finish();
    expect(p.disposed).toBe(false);
    expect(ttsPlaybackRegistry.has("s1")).toBe(true);
  });

  it("does not adopt when the backend differs", () => {
    const p = new FakeProvider(true);
    ttsPlaybackRegistry.install("s1", p, "kokoro");
    expect(ttsPlaybackRegistry.adopt("s1", "browser")).toBeNull();
  });

  it("stop() tears down even a speaking, orphaned provider (session-end)", () => {
    const p = new FakeProvider(true);
    ttsPlaybackRegistry.install("s1", p, "kokoro");
    ttsPlaybackRegistry.release("s1", p, { keepAliveIfSpeaking: true });
    expect(ttsPlaybackRegistry.isOrphaned("s1")).toBe(true);

    ttsPlaybackRegistry.stop("s1");
    expect(p.disposed).toBe(true);
    expect(ttsPlaybackRegistry.has("s1")).toBe(false);
  });

  it("stopOrphansExcept disposes other orphaned tails but spares the active owner and live owners", () => {
    const active = new FakeProvider(true);
    const orphanOther = new FakeProvider(true);
    const liveOther = new FakeProvider(true);

    ttsPlaybackRegistry.install("active", active, "kokoro");
    ttsPlaybackRegistry.install("orphan", orphanOther, "kokoro");
    ttsPlaybackRegistry.install("live", liveOther, "kokoro");

    // "orphan" was handed off; "live" is still owned by a mounted pane.
    ttsPlaybackRegistry.release("orphan", orphanOther, { keepAliveIfSpeaking: true });

    ttsPlaybackRegistry.stopOrphansExcept("active");

    expect(orphanOther.disposed).toBe(true);
    expect(ttsPlaybackRegistry.has("orphan")).toBe(false);
    expect(active.disposed).toBe(false);
    expect(liveOther.disposed).toBe(false);
  });

  it("install replaces and disposes a prior provider for the same key", () => {
    const p1 = new FakeProvider(true);
    const p2 = new FakeProvider(true);
    ttsPlaybackRegistry.install("s1", p1, "kokoro");
    ttsPlaybackRegistry.install("s1", p2, "kokoro");
    expect(p1.disposed).toBe(true);
    expect(ttsPlaybackRegistry.has("s1")).toBe(true);
    expect(p2.disposed).toBe(false);
  });

  it("release is identity-guarded against a stale provider after replacement", () => {
    const p1 = new FakeProvider(true);
    const p2 = new FakeProvider(true);
    ttsPlaybackRegistry.install("s1", p1, "kokoro");
    ttsPlaybackRegistry.install("s1", p2, "kokoro"); // p1 disposed, p2 current

    // A late release for the replaced p1 must not touch p2.
    ttsPlaybackRegistry.release("s1", p1, { keepAliveIfSpeaking: true });
    expect(p2.disposed).toBe(false);
    expect(ttsPlaybackRegistry.has("s1")).toBe(true);
  });
});
