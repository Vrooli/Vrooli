import type { JSX } from "react";

export interface VoiceSettingsPanelProps {
  ariaLabel?: string;
  heading?: string;
  body?: string;
}

/**
 * Voice settings panel. Renders the consumer-supplied heading + slots so
 * each consumer composes its own GetProviderConfig / BYOK credentials /
 * voice-override surface around it. Keeps the embed package free of
 * data-shape opinions while exposing a consistent visual frame.
 */
export function VoiceSettingsPanel(props: VoiceSettingsPanelProps = {}): JSX.Element {
  const ariaLabel = props.ariaLabel ?? "Voice settings";
  const heading = props.heading ?? "Voice Input";
  const body = props.body ?? "Voice settings panel — implemented in audio-tools Configuration Console.";
  return (
    <section aria-label={ariaLabel} className="audio-tools-embed-voice-settings">
      <h3>{heading}</h3>
      <p>{body}</p>
    </section>
  );
}
