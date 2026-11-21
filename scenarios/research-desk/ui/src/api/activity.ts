import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { useQuery } from "@tanstack/react-query";
import {
  ActivityService,
  Baseline,
  ChangeSubject,
  CommissionState,
  MatchKind,
  WebConsoleAccess,
  type ResearchWrite,
  type TopicSummary,
} from "@vrooli/proto-types/research-desk/v1/activity/activity_pb";
import { ReadWhen } from "@vrooli/proto-types/research-desk/v1/shared/commission_pb";

import { serviceClient } from "./client";

/** Activity domain: what is moving, read only (SYSTEM.md §3–§4, §8). */
const client = serviceClient(ActivityService);

/** The projection is cheap (tens of ms); a phone left open keeps itself fresh. */
const LIVE = { refetchInterval: 30_000, refetchOnWindowFocus: true } as const;

export const activityKeys = {
  all: ["activity"] as const,
  changes: (subject: ChangeSubject, id: string) => ["activity", "changes", subject, id] as const,
};

export function useOverview() {
  return useQuery({ queryKey: ["activity", "overview"], queryFn: () => client.getOverview({}), ...LIVE });
}

export function useCommissions() {
  return useQuery({ queryKey: ["activity", "commissions"], queryFn: () => client.listCommissions({}), ...LIVE });
}

export function useCommission(id: string) {
  return useQuery({ queryKey: ["activity", "commission", id], queryFn: () => client.getCommission({ id }), ...LIVE });
}

export function useTopics() {
  return useQuery({ queryKey: ["activity", "topics"], queryFn: () => client.listTopics({}), ...LIVE });
}

/**
 * Reads what changed since the last recorded look, or since `since` when
 * given; recording a look is operatorApi.markSeen.
 */
export function useChanges(subject: ChangeSubject, id: string, since?: Date) {
  return useQuery({
    queryKey: [...activityKeys.changes(subject, id), since?.toISOString() ?? null],
    queryFn: () => client.getChanges({ subject, id, since: since ? timestampFromDate(since) : undefined }),
    ...LIVE,
  });
}

/** Overview lenses the UI shows in its own words rather than as a raw row. */
export const LENS = { courier: "courier", webConsoleAccess: "web-console access" } as const;

/**
 * A topic's last research write: the newest file in its run folders, apart
 * from mission edits and review files, with its source and, when its owner is
 * paused or ended, a note saying so.
 */
export function lastResearchWrite(topic: TopicSummary): ResearchWrite | undefined {
  return topic.lastResearchWrite;
}

export { Baseline, ChangeSubject, CommissionState, MatchKind, ReadWhen, WebConsoleAccess };
export type {
  CommissionSummary,
  GetCommissionResponse,
  GetChangesResponse,
  GetOverviewResponse,
  Overlap,
  ResearchWrite,
  SessionMatch,
  TopicSummary,
} from "@vrooli/proto-types/research-desk/v1/activity/activity_pb";
export type { EntryReading, ReadSet, Workstream } from "@vrooli/proto-types/research-desk/v1/shared/commission_pb";
