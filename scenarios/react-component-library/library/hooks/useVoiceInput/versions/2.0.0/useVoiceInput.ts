/**
 * @libraryId react-component-library:useVoiceInput
 * @version 2.0.0
 * @status released
 * @deps {"react":"^18","@vrooli/audio-capture-browser":"*"}
 */
import { useVoiceCore, type UseVoiceCoreOptions } from "@vrooli/audio-capture-browser";

export type { VoiceCapabilityProbe } from "@vrooli/audio-capture-browser";

/** The library hook now binds the shared live PCM lifecycle directly. */
export type VoiceInputOptions = UseVoiceCoreOptions;

export function useVoiceInput(options: VoiceInputOptions) {
  return useVoiceCore(options);
}
