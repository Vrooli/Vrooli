import type { JSX } from "react";

export interface VoiceSettingsPanelProps {
  ariaLabel?: string;
  heading?: string;
  body?: string;
}

/**
 * Skeleton settings panel. Real version renders the audio-tools
 * SettingsService.GetProviderConfig + BYOK credentials matrix + canonical
 * voice overrides; this stub exists so consumers can mount the component
 * during early Phase F and replace contents in a follow-up.
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
