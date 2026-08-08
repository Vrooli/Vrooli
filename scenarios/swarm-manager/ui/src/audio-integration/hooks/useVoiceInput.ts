/**
 * @vrooliComponentSource react-component-library:useVoiceInput
 * @vrooliComponentVersion 3.0.0
 * @vrooliComponentAdoption 1facf313-4a56-4ab5-8731-0d468b0a929b
 * @vrooliComponentAppliedAt 2026-08-05T04:35:59Z
 * @vrooliComponentSourceSha256 28cd367574b9bac669e54dc163aa14dd6a1a70eff5e19e34b17dee84504aa428
 * @vrooliComponentDriftHash 6dd16bae70dfdfa7ab1a0c6e621b1b6c0c4305f15d984a7846dfa9f8cfdc1ece
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { useVoiceCore, type UseVoiceCoreOptions } from "@vrooli/audio-capture-browser";

export type { VoiceCapabilityProbe } from "@vrooli/audio-capture-browser";
export type VoiceInputOptions = UseVoiceCoreOptions;

/** Thin presentation-library binding over the governed browser capture core. */
export function useVoiceInput(options: VoiceInputOptions) {
  return useVoiceCore(options);
}
