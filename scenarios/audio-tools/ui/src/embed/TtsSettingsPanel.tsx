import type { JSX } from "react";

export interface TtsSettingsPanelProps {
  ariaLabel?: string;
  heading?: string;
  body?: string;
}

export function TtsSettingsPanel(props: TtsSettingsPanelProps = {}): JSX.Element {
  const ariaLabel = props.ariaLabel ?? "TTS settings";
  const heading = props.heading ?? "Voice Output";
  const body = props.body ?? "TTS settings panel — implemented in audio-tools Configuration Console.";
  return (
    <section aria-label={ariaLabel} className="audio-tools-embed-tts-settings">
      <h3>{heading}</h3>
      <p>{body}</p>
    </section>
  );
}
