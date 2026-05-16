import type { JSX } from "react";

export interface VoiceRejectionBannerProps {
  reason: string;
  onDismiss?: () => void;
  dismissLabel?: string;
}

export function VoiceRejectionBanner(props: VoiceRejectionBannerProps): JSX.Element {
  const dismissLabel = props.dismissLabel ?? "Dismiss";
  return (
    <div role="alert" className="audio-tools-embed-voice-rejection-banner">
      <span>{props.reason}</span>
      {props.onDismiss ? (
        <button type="button" onClick={props.onDismiss}>{dismissLabel}</button>
      ) : null}
    </div>
  );
}
