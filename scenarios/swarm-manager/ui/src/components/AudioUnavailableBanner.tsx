import { AlertCircle } from "lucide-react";

export interface AudioUnavailableBannerProps {
  reason?: string;
  className?: string;
}

const REASON_MESSAGES: Record<string, string> = {
  discovery_failed: "Could not reach the audio-tools service. Voice input and synthesis are disabled.",
  scenario_not_running: "The audio-tools scenario is not running. Start it to enable voice features.",
  env_misconfigured: "audio-tools URL is not configured. Check the scenario environment.",
  resolver_not_configured: "Audio-tools discovery is not wired in this build.",
};

/** Honest, presentational dependency-health state for the swarm surface. */
export function AudioUnavailableBanner({ reason, className }: AudioUnavailableBannerProps) {
  if (!reason) return null;
  return (
    <div
      role="status"
      aria-live="polite"
      data-audio-state="unavailable"
      className={["flex items-start gap-2 rounded border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-sm text-amber-200", className ?? ""].join(" ").trim()}
    >
      <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
      <span>{REASON_MESSAGES[reason] ?? `Audio-tools is unavailable (${reason}).`}</span>
    </div>
  );
}

export default AudioUnavailableBanner;
