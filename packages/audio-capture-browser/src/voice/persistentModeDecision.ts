import type { VoiceBackend } from "./types";

export const PERSISTENT_STREAMING_UNAVAILABLE_MESSAGE =
  "Long-form voice requires the durable streaming audio path; dictation was not started.";

export interface PersistentModeDecision {
  allowed: boolean;
  reason?: string;
}

/**
 * Persistent mode is the unlimited-feeling contract. It may not silently
 * downgrade to one-shot/buffered transcription when streaming is unavailable.
 */
export function decidePersistentMode(
  requested: boolean,
  backend: VoiceBackend,
  streamingAvailable: boolean,
): PersistentModeDecision {
  if (!requested || (backend === "whisper" && streamingAvailable)) return { allowed: true };
  return { allowed: false, reason: PERSISTENT_STREAMING_UNAVAILABLE_MESSAGE };
}
