export interface MicReadinessIndicatorProps {
  state: "unknown" | "granted" | "denied" | "prompt";
}

export function MicReadinessIndicator(props: MicReadinessIndicatorProps): JSX.Element {
  const label =
    props.state === "granted"
      ? "Microphone ready"
      : props.state === "denied"
      ? "Microphone denied"
      : props.state === "prompt"
      ? "Microphone permission required"
      : "Microphone status unknown";
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
