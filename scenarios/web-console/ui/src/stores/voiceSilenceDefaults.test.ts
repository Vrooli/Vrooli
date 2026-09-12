import { describe, it, expect, beforeEach } from "vitest";
import {
  VAD_FALLBACK_SILENCE_TIMEOUT_MS,
  VAD_FALLBACK_SEGMENT_SILENCE_MS,
} from "../audio-integration/hooks/voice/vad";
import { useWorkspaceStore } from "./useWorkspaceStore";

// Single-source-of-truth guard for the auto-stop silence timeout.
//
// There used to be FOUR independent copies of this number (server Defaults,
// server defaultStreamCfg, the client vad.ts fallback, and this store's
// default). When they drifted, the mic button could leave the recording state
// while the server kept streaming text ("button off but still transcribing").
//
// The server is the single source of truth (audio-tools internal/stt.
// DefaultVADSilenceMs, currently 1200). These tests pin the two client-side
// copies to it; the Go TestVADSilenceDefaultsSingleSource pins the server
// copies. Changing the value on one side without the other now fails a build.
const SERVER_DEFAULT_VAD_SILENCE_MS = 1200;

describe("voice silence-timeout single source of truth", () => {
  it("client VAD fallback matches the audio-tools server default", () => {
    expect(VAD_FALLBACK_SILENCE_TIMEOUT_MS).toBe(SERVER_DEFAULT_VAD_SILENCE_MS);
    expect(VAD_FALLBACK_SEGMENT_SILENCE_MS).toBe(SERVER_DEFAULT_VAD_SILENCE_MS);
  });

  describe("store defaults derive from the fallback constants", () => {
    beforeEach(() => {
      // Drop any persisted override so we read the code-defined defaults.
      localStorage.clear();
    });

    it("vadSilenceTimeoutMs / segmentSilenceMs default to the constants", () => {
      const s = useWorkspaceStore.getState();
      expect(s.vadSilenceTimeoutMs).toBe(VAD_FALLBACK_SILENCE_TIMEOUT_MS);
      expect(s.segmentSilenceMs).toBe(VAD_FALLBACK_SEGMENT_SILENCE_MS);
    });
  });
});
