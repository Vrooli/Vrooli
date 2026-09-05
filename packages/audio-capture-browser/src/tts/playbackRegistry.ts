// Process-wide TTS playback registry.
//
// Problem it solves (Phase 6, streaming tail-durability): the TTS provider —
// and the single HTMLAudioElement inside KokoroProvider — is created and
// disposed by useTextToSpeechCore, which is instantiated once per mounted
// pane. web-console only keeps a warm set of panes mounted, so switching
// sessions unmounts the evicted pane and disposes its provider, truncating
// in-progress speech mid-tail.
//
// This registry decouples provider lifetime from component (pane) lifetime.
// A pane that is speaking when it unmounts HANDS OFF its provider here instead
// of disposing it; the audio keeps playing. When the pane remounts it ADOPTS
// the same live provider, so there is exactly one owner (no duplicate provider,
// no leaked audio element). A handed-off provider that finishes its tail while
// still orphaned is disposed automatically via its onSettled hook. Genuine stop
// intents (explicit stop, session-end, or a new session starting playback) tear
// it down immediately.
//
// The registry is opt-in: only callers that pass a stable owner key and request
// persistence use it, so the generic hook's default behavior is unchanged.

import type { TTSBackend, TTSProvider } from "./types";

interface RegistryEntry {
  provider: TTSProvider;
  backend: TTSBackend;
  /** True when no live component currently holds this provider — it is being
   *  kept alive only until its in-progress tail settles. */
  orphaned: boolean;
}

class TtsPlaybackRegistry {
  private entries = new Map<string, RegistryEntry>();

  /** Install a freshly-created provider as the live owner for `key`. Replaces
   *  (and disposes) any prior entry for the key. */
  install(key: string, provider: TTSProvider, backend: TTSBackend): void {
    const prior = this.entries.get(key);
    if (prior && prior.provider !== provider) {
      this.disposeEntry(prior);
    }
    provider.onSettled = null;
    this.entries.set(key, { provider, backend, orphaned: false });
  }

  /** Adopt the live provider for `key` (e.g. on pane remount) when one exists
   *  and its backend matches. Marks it re-owned and cancels any pending
   *  settle-dispose. Returns null when there is nothing to adopt. */
  adopt(key: string, backend: TTSBackend): TTSProvider | null {
    const entry = this.entries.get(key);
    if (!entry || entry.backend !== backend) return null;
    entry.orphaned = false;
    entry.provider.onSettled = null;
    return entry.provider;
  }

  /** The component holding the provider for `key` is unmounting. When it is
   *  still speaking and `keepAliveIfSpeaking` is set, keep the provider alive
   *  until its tail settles; otherwise dispose it now. Identity-guarded so a
   *  stale release for a replaced provider is ignored. */
  release(key: string, provider: TTSProvider, opts: { keepAliveIfSpeaking: boolean }): void {
    const entry = this.entries.get(key);
    if (!entry || entry.provider !== provider) return;
    if (opts.keepAliveIfSpeaking && provider.isSpeaking) {
      entry.orphaned = true;
      // Dispose once the tail completes (or is stopped) while still orphaned.
      provider.onSettled = () => {
        const current = this.entries.get(key);
        if (current === entry && entry.orphaned && !provider.isSpeaking) {
          this.remove(key);
        }
      };
      return;
    }
    this.remove(key);
  }

  /** Explicit teardown for `key` (user stop, session-end): dispose and drop. */
  stop(key: string): void {
    if (this.entries.has(key)) this.remove(key);
  }

  /** Enforce single-owner audio when a new playback starts on `key`: tear down
   *  any orphaned tail still playing under a different key. Live (non-orphaned)
   *  owners are left alone — only one pane is ever the active speaker. */
  stopOrphansExcept(key: string): void {
    for (const [k, entry] of this.entries) {
      if (k !== key && entry.orphaned) this.remove(k);
    }
  }

  has(key: string): boolean {
    return this.entries.has(key);
  }

  backendOf(key: string): TTSBackend | undefined {
    return this.entries.get(key)?.backend;
  }

  /** Test/inspection helpers. */
  isOrphaned(key: string): boolean {
    return this.entries.get(key)?.orphaned ?? false;
  }
  get size(): number {
    return this.entries.size;
  }
  _resetForTests(): void {
    for (const key of [...this.entries.keys()]) this.remove(key);
  }

  private remove(key: string): void {
    const entry = this.entries.get(key);
    if (!entry) return;
    this.entries.delete(key);
    this.disposeEntry(entry);
  }

  private disposeEntry(entry: RegistryEntry): void {
    entry.provider.onSettled = null;
    entry.provider.dispose();
  }
}

/** Process-wide singleton. */
export const ttsPlaybackRegistry = new TtsPlaybackRegistry();
