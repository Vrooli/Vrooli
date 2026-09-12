import { createClient } from "@connectrpc/connect";
import {
  RecallService,
  type RecallHit,
} from "@vrooli/proto-types/vrooli-memory/v1/recall/recall_pb";

import { transport } from "./client";

export const recallClient = createClient(RecallService, transport);

export async function recallMemory(query: string, limit = 10): Promise<RecallHit[]> {
  const response = await recallClient.recall({ query, limit });
  return response.hits;
}
