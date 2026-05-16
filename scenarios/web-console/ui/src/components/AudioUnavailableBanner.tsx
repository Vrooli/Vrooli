import { AlertCircle } from "lucide-react";

/**
 * AudioUnavailableBanner renders when discovery reported audio-tools is
 * unreachable. The `reason` token comes from
 * window.__AUDIO_TOOLS_UNAVAILABLE_REASON__ (populated by main.tsx) and
 * is mapped to an operator-readable message here.
 *
 * Consumers should mount this in voice-eligible regions (workspace
 * voice-input area, TTS playback bar) so a user clicking the disabled
 * voice control sees why it's disabled instead of a silent no-op.
 */
export interface AudioUnavailableBannerProps {
  reason?: string;
  className?: string;
}

const REASON_MESSAGES: Record<string, string> = {
  discovery_failed: "Could not reach the audio-tools service. Voice input and synthesis are disabled.",
  scenario_not_running: "The audio-tools scenario is not running. Start it to enable voice features.",
  env_misconfigured: "audio-tools URL is not configured. Set AUDIO_TOOLS_URL or start the audio-tools scenario.",
  resolver_not_configured: "Audio-tools discovery is not wired in this build of web-console.",
};

export function AudioUnavailableBanner({ reason, className }: AudioUnavailableBannerProps) {
  if (!reason) return null;
  const message = REASON_MESSAGES[reason] ?? `Audio-tools is unavailable (${reason}).`;
  return (
    <div
      role="status"
      aria-live="polite"
      className={[
        "flex items-start gap-2 rounded-control border border-app-warning/40 bg-app-warning/10 px-3 py-2 text-sm text-app-warning-foreground",
        className ?? "",
      ].join(" ").trim()}
    >
      <AlertCircle className="mt-0.5 h-4 w-4 flex-shrink-0" aria-hidden="true" />
      <span>{message}</span>
    </div>
  );
}

/**
 * Hook-level convenience: returns the current unavailable-reason
 * snapshot from the window global, or undefined when audio-tools is
 * available. Re-reads on every render (cheap; just a property access)
 * since the reason is set once at bootstrap and never mutates.
 */
export function useAudioToolsUnavailableReason(): string | undefined {
  if (typeof window === "undefined") return undefined;
  const r = window.__AUDIO_TOOLS_UNAVAILABLE_REASON__;
  return r && r.length > 0 ? r : undefined;
}
