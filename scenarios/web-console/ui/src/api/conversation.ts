import { createClient } from "@connectrpc/connect";
import { ConversationService } from "@vrooli/proto-types/web-console/v1/conversation/conversation_pb";

import { transport } from "./client";

// conversationClient is the Connect-Web client for ConversationService.
// UI code imports this directly; the legacy fetch helpers in lib/api.ts
// are shims that delegate here and normalize the camelCase proto shape
// to the existing snake_case wire types.
export const conversationClient = createClient(ConversationService, transport);
