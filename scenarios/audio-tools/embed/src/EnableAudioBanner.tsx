import type { JSX } from "react";

export interface EnableAudioBannerProps {
  onEnable: () => void;
  /** Caller-supplied (translated) banner copy. */
  message?: string;
  /** Caller-supplied (translated) action label. */
  actionLabel?: string;
}

export function EnableAudioBanner(props: EnableAudioBannerProps): JSX.Element {
  const message = props.message ?? "Audio is muted. Click to enable.";
  const actionLabel = props.actionLabel ?? "Enable audio";
  return (
    <div role="status" className="audio-tools-embed-enable-audio-banner">
      <span>{message}</span>
      <button type="button" onClick={props.onEnable}>
        {actionLabel}
      </button>
    </div>
  );
}
