import type { Dispatch, SetStateAction } from "react";
import type { VoiceInputState } from "./types";

const SILENT_STREAM_STATUS = new Set(["processed_acknowledgement", "provider_identity"]);
const NOTICE_CLEARING_STREAM_STATUS = new Set([
  "stream_connected",
  "transcription_complete",
  "buffered_recovery_completed",
  "mic_reacquired",
  "mic_unmuted",
]);
const DEGRADATION_STREAM_STATUS = new Set([
  "backend_degraded",
  "reconnect_exhausted",
  "buffered_recovery",
]);

/** Owns stream notice policy and dismissal state for one voice-core mount. */
export class StreamNoticeController {
  private dismissedNotice: string | null = null;

  constructor(private readonly setState: Dispatch<SetStateAction<VoiceInputState>>) {}

  clearDismissed(): void {
    this.dismissedNotice = null;
  }

  dismissFallback(): void {
    this.setState((state) => {
      this.dismissedNotice = state.fallbackNotice;
      return state.fallbackNotice ? { ...state, fallbackNotice: null } : state;
    });
  }

  clearFallbackOnPartial(): void {
    this.clearDismissed();
    this.setState((state) => state.fallbackNotice
      ? { ...state, fallbackNotice: null }
      : state);
  }

  handleStatus(code: string, message?: string): void {
    if (SILENT_STREAM_STATUS.has(code)) return;

    if (DEGRADATION_STREAM_STATUS.has(code)) {
      this.setState((state) => ({
        ...state,
        streamingDegradationNotice: message || "Streaming degraded — buffered mode is active for this transcription.",
      }));
    }
    if (NOTICE_CLEARING_STREAM_STATUS.has(code)) {
      this.setState((state) => state.fallbackNotice ? { ...state, fallbackNotice: null } : state);
      this.clearDismissed();
      return;
    }
    if (!message || this.dismissedNotice === message) return;
    this.setState((state) => state.fallbackNotice === message
      ? state
      : { ...state, fallbackNotice: message });
  }
}
