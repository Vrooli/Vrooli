import { AlertCircle } from "lucide-react";

/**
 * AudioUnavailableBanner renders when audio-tools is unreachable.
 * Consumers query web-console's health endpoint and pass the resulting
 * dependency-status token in via `reason`. This component is purely
 * presentational; it does not subscribe to any source on its own.
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
