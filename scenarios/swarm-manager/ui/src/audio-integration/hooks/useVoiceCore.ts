import {
  useVoiceInput as useAdoptedVoiceInput,
  type VoiceInputOptions as AdoptedVoiceInputOptions,
  type VoiceCapabilityProbe,
} from "@vrooli/react-component-library/useVoiceInput/3.0.0";
import { voiceCoreServices } from "../voiceCoreServices";

// HOST DIFFERENCE: this hook only injects swarm's service bundle; state
// transitions and ownership policy are implemented by the package.
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
