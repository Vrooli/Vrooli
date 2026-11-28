// Scenario-local adapter that binds the shared browser capture package to
// Audio Tools' own API. Keep this out of the application entry bundle; only
// the lazy routes that use streaming capture need to load it.
// HOST DIFFERENCE: binds the shared transport registration to Audio Tools' RPC API and retry policy.
import { registerVoiceTransport as registerBrowserVoiceTransport } from "@vrooli/audio-capture-browser";
import { openVoiceStream, transcribeAudioWithRetry, transcribeAudioWithRetryDetailed } from "./api/voice";

let registered = false;

export function registerVoiceTransport(): void {
  if (registered) return;
  registerBrowserVoiceTransport({
    retainedRoutePolicy: "exact-model",
    openStream: (start, handlers) => openVoiceStream(start, handlers),
    transcribeRetained: (blob, language, route) => transcribeAudioWithRetry(blob, 2, language, route),
    transcribeRetainedWithIdentity: (blob, language, route) => transcribeAudioWithRetryDetailed(blob, 2, language, route),
  });
  registered = true;
}
