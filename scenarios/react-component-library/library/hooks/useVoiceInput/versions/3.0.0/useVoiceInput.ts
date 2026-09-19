/**
 * @libraryId react-component-library:useVoiceInput
 * @version 3.0.0
 * @status released
 * @deps {"react":"^18","@vrooli/audio-capture-browser":"*"}
 */
import {
  useVoiceCore,
  type UseVoiceCoreOptions,
} from "@vrooli/audio-capture-browser";

export type { VoiceCapabilityProbe } from "@vrooli/audio-capture-browser";
export type VoiceInputOptions = UseVoiceCoreOptions;

/** Thin presentation-library binding over the governed browser capture core. */
export function useVoiceInput(options: VoiceInputOptions) {
  return useVoiceCore(options);
}
