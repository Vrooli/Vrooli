import { useVoiceInput as useAdoptedVoiceInput, } from "./useVoiceInput";
import { voiceCoreServices } from "../voiceCoreServices";
export { useScenarioVoiceInput, probeWhisperHealth } from "./useScenarioVoiceInput";
export function useScenarioVoiceCore(opts) {
    return useAdoptedVoiceInput({
        ...opts,
        services: voiceCoreServices,
    });
}
