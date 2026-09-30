import {
  useVoiceInput as useAdoptedVoiceInput,
  type VoiceInputOptions as AdoptedVoiceInputOptions,
  type VoiceCapabilityProbe,
} from "@vrooli/react-component-library/useVoiceInput/3";
import { voiceCoreServices } from "../voiceCoreServices";

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
// HOST DIFFERENCE: audio-tools binds the shared voice core to its own transport and settings.
