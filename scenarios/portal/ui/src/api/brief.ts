import { createClient } from "@connectrpc/connect";
import {
  BriefConsumer,
  BriefService,
  BriefUseKind,
  type Brief,
} from "@vrooli/proto-types/portal/v1/brief/brief_pb";

import { transport } from "./client";

const briefClient = createClient(BriefService, transport);

export async function listPortalBriefs(input: {
  consumer?: BriefConsumer;
  chatId?: string;
  limit?: number;
} = {}): Promise<Brief[]> {
  const response = await briefClient.list({
    consumer: input.consumer ?? BriefConsumer.UNSPECIFIED,
    chatId: input.chatId ?? "",
    sessionRef: "",
    limit: input.limit ?? 20,
  });
  return response.briefs;
}

export async function getPortalBrief(id: string): Promise<Brief> {
  const response = await briefClient.get({ id });
  if (!response.brief) throw new Error("brief response did not include a brief");
  return response.brief;
}

export async function recordPortalBriefUse(
  briefId: string,
  itemIndex: number,
  kind: BriefUseKind,
): Promise<boolean> {
  const response = await briefClient.recordUse({ briefId, itemIndex, kind });
  return response.recorded;
}

export { BriefConsumer, BriefUseKind };
export type { Brief };
