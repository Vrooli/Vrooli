import { createClient } from "@connectrpc/connect";
import {
  JournalService,
  type Entry,
} from "@vrooli/proto-types/vrooli-memory/v1/journal/journal_pb";

import { transport } from "./client";

export const journalClient = createClient(JournalService, transport);

/** Lists the newest immutable journal entries for the operator timeline. */
export async function listJournalEntries(limit = 100): Promise<Entry[]> {
  const response = await journalClient.listEntries({ limit });
  return response.entries;
}
