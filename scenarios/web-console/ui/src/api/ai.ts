import { createClient } from "@connectrpc/connect";
import { AIService } from "@vrooli/proto-types/web-console/v1/ai/ai_pb";

import { transport } from "./client";

// aiClient is the Connect-Web client for AIService.
// UI code imports this directly; the legacy fetch helpers in lib/api.ts
// are shims that delegate here and normalize the camelCase proto shape
// to the existing snake_case wire types.
export const aiClient = createClient(AIService, transport);
