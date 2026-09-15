import { createClient, type Client } from "@connectrpc/connect";
import { createScenarioConnectTransport } from "@vrooli/api-base";
import { ConversationSearchService } from "@vrooli/proto-types/agent-manager/v1/domain/conversation_search_pb";
import { resolveAgentManagerApiBase } from "../../../lib/api";

export const conversationSearchClient: Client<typeof ConversationSearchService> = createClient(
  ConversationSearchService,
  createScenarioConnectTransport({ baseUrl: resolveAgentManagerApiBase() }),
);
