import {
  useVoiceInput as useAdoptedVoiceInput,
  type VoiceInputOptions as AdoptedVoiceInputOptions,
  type VoiceCapabilityProbe,
} from "./useVoiceInput";
import { voiceCoreServices } from "../voiceCoreServices";
export { useScenarioVoiceInput, probeWhisperHealth } from "./useScenarioVoiceInput";

// HOST DIFFERENCE: web-console additionally exposes its scenario health probe;
// capture core and ownership policy remain package-owned.
export type { VoiceCapabilityProbe };

export type UseVoiceCoreOptions = Omit<AdoptedVoiceInputOptions, "services"> & {
  /** Host compatibility flag; capture lifecycle is shared and does not pre-warm. */
  lowLatencyVoice?: boolean;
};

export function useScenarioVoiceCore(opts: UseVoiceCoreOptions) {
  return useAdoptedVoiceInput({
    ...opts,
    services: voiceCoreServices,
  });
}
