import { createClient } from "@connectrpc/connect";
import { TTSService } from "@vrooli/proto-types/web-console/v1/tts/tts_pb";

import { transport } from "./client";

// ttsClient is the Connect-Web client for TTSService. UI code imports this
// directly; the legacy helpers in lib/api.ts are shims that delegate here
// and normalize the camelCase proto shape to the existing wire types.
export const ttsClient = createClient(TTSService, transport);
