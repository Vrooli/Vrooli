// HOST DIFFERENCE: web-console renders its own readiness copy and tooltip.
import type { JSX } from "react";

export interface MicReadinessIndicatorProps {
  state: "unknown" | "granted" | "denied" | "prompt";
  /** Caller-supplied labels per state (translated). */
  labels?: Partial<Record<MicReadinessIndicatorProps["state"], string>>;
}

const DEFAULT_LABELS: Record<MicReadinessIndicatorProps["state"], string> = {
  granted: "Microphone ready",
  denied: "Microphone denied",
  prompt: "Microphone permission required",
  unknown: "Microphone status unknown",
};

export function MicReadinessIndicator(props: MicReadinessIndicatorProps): JSX.Element {
  const label = props.labels?.[props.state] ?? DEFAULT_LABELS[props.state];
  return (
    <span
      role="status"
      aria-live="polite"
      data-state={props.state}
      className="audio-tools-embed-mic-readiness"
    >
      {label}
    </span>
  );
}
