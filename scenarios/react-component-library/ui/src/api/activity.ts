import { createClient } from "@connectrpc/connect";
import {
  ActivityService,
  type ActivityEvent,
  type GetActivitySummaryResponse,
} from "@vrooli/proto-types/react-component-library/v1/activity/activity_pb";

import { transport } from "./client";

const client = createClient(ActivityService, transport);

export type { ActivityEvent, GetActivitySummaryResponse };

export async function listActivity(input: {
  subjectKind?: string;
  subjectId?: string;
  actor?: string;
  limit?: number;
  since?: string;
} = {}): Promise<ActivityEvent[]> {
  const response = await client.listActivity({
    subjectKind: input.subjectKind ?? "",
    subjectId: input.subjectId ?? "",
    actor: input.actor ?? "",
    limit: input.limit ?? 100,
    since: input.since ?? "",
  });
  return response.events;
}

export async function getActivitySummary(days = 0): Promise<GetActivitySummaryResponse> {
  return client.getActivitySummary({ days });
}
