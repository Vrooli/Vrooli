/**
 * Skeleton settings panel. Real version renders the audio-tools
 * SettingsService.GetProviderConfig + BYOK credentials matrix + canonical
 * voice overrides; this stub exists so consumers can mount the component
 * during early Phase F and replace contents in a follow-up.
 */
export function VoiceSettingsPanel(): JSX.Element {
  return (
    <section aria-label="Voice settings" className="audio-tools-embed-voice-settings">
      <h3>Voice Input</h3>
      <p>Voice settings panel — implemented in audio-tools Configuration Console.</p>
    </section>
  );
}
