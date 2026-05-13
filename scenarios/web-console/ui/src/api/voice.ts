import { createClient } from "@connectrpc/connect";
import { VoiceService } from "@vrooli/proto-types/web-console/v1/voice/voice_pb";

import { transport } from "./client";

// voiceClient is the Connect-Web client for VoiceService. UI code imports
// this directly; the legacy helpers in lib/api.ts are shims that delegate
// here and normalize the camelCase proto shape to the existing wire types.
export const voiceClient = createClient(VoiceService, transport);
