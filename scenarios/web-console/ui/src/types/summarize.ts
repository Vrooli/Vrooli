export type SummarizeErrorStatus = "idle" | "retrying" | "failed";

/**
 * A failed TTS summarization, surfaced as a top-chrome banner until the user
 * dismisses it or a retry succeeds. Lived in `SummarizeErrorBanner.tsx` until
 * that component was replaced by a banner descriptor.
 */
export interface SummarizeErrorState {
  sessionId: string;
  eventId: string;
  message: string;
  /** "auto" = backend-initiated, "on-demand" = user-initiated from the bar/popover. */
  source: "auto" | "on-demand";
  status: SummarizeErrorStatus;
}
