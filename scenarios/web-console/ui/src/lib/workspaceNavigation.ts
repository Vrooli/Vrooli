import type { ConversationCursor, ConversationEvent } from "../api/conversation";
import { formatRelativeTime, stripMarkdown } from "../components/MessageJumpList.helpers";
import type { PaneViewMode } from "../stores/useConversationStore";
import type { PaneMetadata, SidebarSortMode, TabGroupMeta } from "../stores/useWorkspaceStore";

type ConversationSessionSnapshot = {
  events: ConversationEvent[];
  cursor: ConversationCursor;
};

export type WorkspaceNavigationItem =
  | { kind: "group-label"; group: TabGroupMeta; tabCount: number }
  | {
      kind: "pane";
      pane: PaneMetadata;
      globalIndex: number;
      group?: TabGroupMeta;
      groupPosition?: "single" | "first" | "middle" | "last";
      isActive: boolean;
      unreadCount: number;
      viewMode: PaneViewMode;
      latestEventAt: string | null;
      lastVisitedAt: string | null;
      activityLabel: string;
      previewText: string;
    };

export interface BuildWorkspaceNavigationItemsOptions {
  panes: PaneMetadata[];
  groups: TabGroupMeta[];
  activePane: string | null;
  /**
   * Per-session conversation snapshots. Optional: the tab strip renders unread
   * badges via per-tab subscriptions (so a new message re-renders only the one
   * badge), and builds its group/pane structure without conversation data.
   * The sidebar passes this to populate previews/activity/unread.
   */
  conversationSessions?: Record<string, ConversationSessionSnapshot | undefined>;
  viewModes?: Record<string, PaneViewMode | undefined>;
  lastVisitedBySession?: Record<string, string | undefined>;
  now?: Date;
  /**
   * View-only ordering for the sidebar. "manual" (default) preserves the
   * backend sort_order (the `panes` array order); the rest sort *within* each
   * group partition without ever mutating sort_order. The tab strip omits this
   * (always manual).
   */
  sortMode?: SidebarSortMode;
}

/** Per-pane comparison inputs used by the non-manual sidebar sorts. */
export interface PaneSortMetrics {
  name: string;
  /** Activity timestamp in ms (latest event, else last-visited, else 0). */
  activityMs: number;
  unread: number;
}

/**
 * View-only reorder of panes for the sidebar. Groups stay contiguous: panes
 * are bucketed by `groupId` in first-appearance order, each bucket is sorted by
 * the chosen comparator, then buckets are concatenated back in block order.
 * "manual" returns the input untouched (stable passthrough).
 */
export function sortPanesForView(
  panes: PaneMetadata[],
  sortMode: SidebarSortMode,
  metrics: Map<string, PaneSortMetrics>,
): PaneMetadata[] {
  if (sortMode === "manual") return panes;

  const bucketOrder: (string | null)[] = [];
  const buckets = new Map<string | null, PaneMetadata[]>();
  for (const pane of panes) {
    const key = pane.groupId;
    let bucket = buckets.get(key);
    if (!bucket) {
      bucket = [];
      buckets.set(key, bucket);
      bucketOrder.push(key);
    }
    bucket.push(pane);
  }

  const empty: PaneSortMetrics = { name: "", activityMs: 0, unread: 0 };
  const compare = (a: PaneMetadata, b: PaneMetadata): number => {
    const ma = metrics.get(a.sessionId) ?? empty;
    const mb = metrics.get(b.sessionId) ?? empty;
    switch (sortMode) {
      case "name":
        return ma.name.localeCompare(mb.name);
      case "activity":
        return mb.activityMs - ma.activityMs;
      case "unread":
        return mb.unread - ma.unread || mb.activityMs - ma.activityMs;
      default:
        return 0;
    }
  };

  const result: PaneMetadata[] = [];
  for (const key of bucketOrder) {
    const bucket = buckets.get(key);
    if (!bucket) continue;
    result.push(...[...bucket].sort(compare));
  }
  return result;
}

function countUnreadMessages(pane: PaneMetadata, session: ConversationSessionSnapshot | undefined): number {
  if (!pane.supportsMessagesView || !session) return 0;
  return session.events.filter((event) => event.role === "assistant" && event.sequence > session.cursor.lastSeenSequence).length;
}

export function countWorkspaceUnreadMessages(
  panes: PaneMetadata[],
  conversationSessions: Record<string, ConversationSessionSnapshot | undefined>,
): number {
  return panes.reduce((sum, pane) => sum + countUnreadMessages(pane, conversationSessions[pane.sessionId]), 0);
}

function latestEvent(events: ConversationEvent[]): ConversationEvent | null {
  return events.reduce<ConversationEvent | null>((latest, event) => {
    if (!latest) return event;
    if (event.sequence !== latest.sequence) {
      return event.sequence > latest.sequence ? event : latest;
    }
    return event.createdAt > latest.createdAt ? event : latest;
  }, null);
}

function eventPreview(event: ConversationEvent | null): string {
  if (!event?.text) return "";
  return stripMarkdown(event.text).replace(/\s+/g, " ").trim();
}

export function buildWorkspaceNavigationItems({
  panes,
  groups,
  activePane,
  conversationSessions = {},
  viewModes = {},
  lastVisitedBySession = {},
  now = new Date(),
  sortMode = "manual",
}: BuildWorkspaceNavigationItemsOptions): WorkspaceNavigationItem[] {
  const groupMap = new Map(groups.map((group) => [group.id, group]));
  const items: WorkspaceNavigationItem[] = [];
  let lastGroupId: string | null | undefined = undefined;

  // Non-manual sidebar sorts reorder a view-only copy (sort_order is never
  // touched). Metrics are computed once up front so the comparator is cheap.
  let orderedPanes = panes;
  if (sortMode !== "manual") {
    const metrics = new Map<string, PaneSortMetrics>();
    for (const pane of panes) {
      const session = conversationSessions[pane.sessionId];
      const latest = latestEvent(session?.events ?? []);
      const activityAt = latest?.createdAt ?? lastVisitedBySession[pane.sessionId] ?? null;
      metrics.set(pane.sessionId, {
        name: pane.name,
        activityMs: activityAt ? Date.parse(activityAt) || 0 : 0,
        unread: countUnreadMessages(pane, session),
      });
    }
    orderedPanes = sortPanesForView(panes, sortMode, metrics);
  }

  orderedPanes.forEach((pane, idx) => {
    const groupId = pane.groupId;
    const group = groupId ? groupMap.get(groupId) : undefined;
    const previousPane = idx > 0 ? orderedPanes[idx - 1] : undefined;
    const nextPane = idx < orderedPanes.length - 1 ? orderedPanes[idx + 1] : undefined;
    const previousInSameGroup = !!group && previousPane?.groupId === groupId;
    const nextInSameGroup = !!group && nextPane?.groupId === groupId;

    if (groupId && groupId !== lastGroupId && group) {
      const tabCount = panes.filter((candidate) => candidate.groupId === groupId).length;
      items.push({ kind: "group-label", group, tabCount });
    }
    lastGroupId = groupId;

    if (group?.isCollapsed) return;

    const session = conversationSessions[pane.sessionId];
    const latest = latestEvent(session?.events ?? []);
    const latestEventAt = latest?.createdAt ?? null;
    const lastVisitedAt = lastVisitedBySession[pane.sessionId] ?? null;
    const activityAt = latestEventAt ?? lastVisitedAt;
    const activityLabel = activityAt
      ? latestEventAt
        ? formatRelativeTime(activityAt, now)
        : `Visited ${formatRelativeTime(activityAt, now)}`
      : "";

    const unreadCount = countUnreadMessages(pane, session);

    items.push({
      kind: "pane",
      pane,
      globalIndex: idx,
      group,
      groupPosition: group
        ? previousInSameGroup && nextInSameGroup
          ? "middle"
          : previousInSameGroup
            ? "last"
            : nextInSameGroup
              ? "first"
              : "single"
        : undefined,
      isActive: pane.sessionId === activePane,
      unreadCount,
      viewMode: pane.supportsMessagesView ? (viewModes[pane.sessionId] ?? "terminal") : "terminal",
      latestEventAt,
      lastVisitedAt,
      activityLabel,
      previewText: eventPreview(latest),
    });
  });

  return items;
}
