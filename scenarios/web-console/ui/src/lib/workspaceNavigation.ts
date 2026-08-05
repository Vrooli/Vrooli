import type { ConversationCursor, ConversationEvent } from "../api/conversation";
import type { SessionOriginName } from "../api/sessions";
import { formatRelativeTime, stripMarkdown } from "../components/MessageJumpList.helpers";
import type { PaneViewMode } from "../stores/useConversationStore";
import type { PaneMetadata, SidebarOriginTab, SidebarSortMode, TabGroupMeta } from "../stores/useWorkspaceStore";

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
  /**
   * Optional override for each pane's emitted `globalIndex`. The origin-bucketed
   * sidebar passes each pane's index in the FULL (unbucketed) store array so
   * drag-reorder keeps addressing the backing array even when a bucket renders
   * only a subsequence of it. Omitted for the flat list, where `globalIndex` is
   * simply the pane's position.
   */
  globalIndexBySession?: Record<string, number>;
}

/**
 * Reorder panes so each group occupies ONE contiguous block, anchored at the
 * position of its first member. Ungrouped panes keep their relative positions;
 * a group's members keep their relative order inside the block.
 *
 * This is the invariant the whole navigation layer assumes but nothing used to
 * enforce. Group boundaries are decided purely by adjacency — a group label is
 * emitted whenever `groupId` differs from the previous pane's — so a group
 * whose members are not neighbors renders as several blocks, each with its own
 * header and each claiming the group's full member count. Every ordinary
 * action used to be able to cause that: removing a middle member from a group,
 * dragging any pane through a group's run, or simply reloading (the backend
 * orders by `sort_order, created_at`, and ties are common).
 *
 * Applying this at every write AND at the render boundary makes the split
 * unrepresentable rather than merely unlikely. Returns the input array itself
 * when it already satisfies the invariant, so React sees a stable identity on
 * the overwhelmingly common path.
 */
export function orderPanesByGroupBlocks(panes: PaneMetadata[]): PaneMetadata[] {
  const membersByGroup = new Map<string, PaneMetadata[]>();
  for (const pane of panes) {
    if (!pane.groupId) continue;
    const members = membersByGroup.get(pane.groupId);
    if (members) members.push(pane);
    else membersByGroup.set(pane.groupId, [pane]);
  }
  if (membersByGroup.size === 0) return panes;

  const emitted = new Set<string>();
  const ordered: PaneMetadata[] = [];
  for (const pane of panes) {
    const groupId = pane.groupId;
    if (!groupId) {
      ordered.push(pane);
      continue;
    }
    if (emitted.has(groupId)) continue;
    emitted.add(groupId);
    ordered.push(...(membersByGroup.get(groupId) ?? [pane]));
  }

  return ordered.every((pane, i) => pane === panes[i]) ? panes : ordered;
}

/**
 * The group a pane belongs to after being dropped at `index`, following the
 * tab-group convention users already know from browsers:
 *   - dropped strictly *inside* a group's run → joins that group;
 *   - dropped against an edge of its own group → stays (plain reorder);
 *   - dropped anywhere else → leaves its group.
 *
 * Without this a drop inside a group would be silently undone by
 * `orderPanesByGroupBlocks`, which reads as the drag doing nothing.
 * `panes` must already have the moved pane at `index`.
 */
export function groupIdForDropPosition(
  panes: PaneMetadata[],
  index: number,
  currentGroupId: string | null,
): string | null {
  const before = panes[index - 1]?.groupId ?? null;
  const after = panes[index + 1]?.groupId ?? null;
  if (before !== null && before === after) return before;
  if (currentGroupId !== null && (before === currentGroupId || after === currentGroupId)) {
    return currentGroupId;
  }
  return null;
}

/** Per-pane comparison inputs used by the non-manual sidebar sorts. */
export interface PaneSortMetrics {
  name: string;
  /** Activity timestamp in ms (latest event, else last-visited, else 0). */
  activityMs: number;
  unread: number;
  /** The user's manual "come back to this" flag. */
  flagged: boolean;
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

  const empty: PaneSortMetrics = { name: "", activityMs: 0, unread: 0, flagged: false };
  const compare = (a: PaneMetadata, b: PaneMetadata): number => {
    const ma = metrics.get(a.sessionId) ?? empty;
    const mb = metrics.get(b.sessionId) ?? empty;
    switch (sortMode) {
      case "name":
        return ma.name.localeCompare(mb.name);
      case "activity":
        return mb.activityMs - ma.activityMs;
      case "unread":
        // Real unread messages outrank a manual flag (they carry a count and
        // a reason), but a flagged session still sorts above an untouched one.
        return mb.unread - ma.unread
          || Number(mb.flagged) - Number(ma.flagged)
          || mb.activityMs - ma.activityMs;
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
  globalIndexBySession,
}: BuildWorkspaceNavigationItemsOptions): WorkspaceNavigationItem[] {
  const groupMap = new Map(groups.map((group) => [group.id, group]));
  const items: WorkspaceNavigationItem[] = [];
  let lastGroupId: string | null | undefined = undefined;

  // Group blocks first, in every mode. `sortPanesForView` already partitions
  // by group, so this is a no-op for the non-manual sorts; it is the render
  // boundary's last line of defence for "manual", where the pane array is
  // whatever the store and the backend last agreed on.
  let orderedPanes = orderPanesByGroupBlocks(panes);

  // Non-manual sidebar sorts reorder a view-only copy (sort_order is never
  // touched). Metrics are computed once up front so the comparator is cheap.
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
        flagged: pane.manuallyUnread,
      });
    }
    orderedPanes = sortPanesForView(orderedPanes, sortMode, metrics);
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
      globalIndex: globalIndexBySession?.[pane.sessionId] ?? idx,
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

// ---------------------------------------------------------------------------
// Origin buckets
// ---------------------------------------------------------------------------

/** Display order of the sidebar origin tabs, top to bottom / left to right. */
export const ORIGIN_BUCKET_ORDER: readonly SidebarOriginTab[] = ["ui", "programmatic", "remote"];

/**
 * Fold a session's provenance into a sidebar origin bucket. "unspecified" (and
 * any unknown/absent origin) normalizes to "programmatic", matching the server's
 * normalization of an origin-less create — so a session that reaches the UI
 * untagged still lands in a real bucket rather than a fourth phantom one.
 */
export function originBucket(origin: SessionOriginName | undefined): SidebarOriginTab {
  return origin === "ui" ? "ui" : origin === "remote" ? "remote" : "programmatic";
}

export interface OriginBucketNavigation {
  bucket: SidebarOriginTab;
  items: WorkspaceNavigationItem[];
}

export interface BuildOriginBucketedNavigationOptions extends BuildWorkspaceNavigationItemsOptions {
  /** Provenance per session id. Sessions absent from the map bucket into
   *  "programmatic" via originBucket's unspecified fallback. */
  originBySession: Record<string, SessionOriginName | undefined>;
}

/**
 * Partition the sidebar into origin buckets (UI-owned / Programmatic / Remote),
 * each carrying its own navigation list with groups, sort, and drag intact.
 *
 * Only non-empty buckets are returned, in ORIGIN_BUCKET_ORDER. Each bucket's
 * items are built over just that bucket's panes — so groups, sort modes, and
 * group-position rounding compose *within* the bucket — while `globalIndex` is
 * pinned to the pane's position in the full (unbucketed) list so drag-reorder
 * still addresses the backing store array.
 *
 * A group is atomic here: every member follows the group's FIRST member into a
 * single bucket, whatever its own provenance. A group is a unit the user made
 * by hand, so a mixed-origin group (say, one session opened in the UI and one
 * started by an agent) must not be torn in half across two tabs — where each
 * half looks like the group has silently lost members.
 *
 * With only UI-origin sessions this returns a single "ui" bucket, and the caller
 * renders that list exactly as it did before origin tabs existed.
 */
export function buildOriginBucketedNavigation({
  originBySession,
  ...options
}: BuildOriginBucketedNavigationOptions): OriginBucketNavigation[] {
  const { panes } = options;
  const globalIndexBySession: Record<string, number> = {};
  panes.forEach((pane, index) => {
    globalIndexBySession[pane.sessionId] = index;
  });

  const bucketByGroup = new Map<string, SidebarOriginTab>();
  for (const pane of panes) {
    if (!pane.groupId || bucketByGroup.has(pane.groupId)) continue;
    bucketByGroup.set(pane.groupId, originBucket(originBySession[pane.sessionId]));
  }
  const bucketForPane = (pane: PaneMetadata): SidebarOriginTab =>
    (pane.groupId ? bucketByGroup.get(pane.groupId) : undefined)
    ?? originBucket(originBySession[pane.sessionId]);

  const result: OriginBucketNavigation[] = [];
  for (const bucket of ORIGIN_BUCKET_ORDER) {
    const bucketPanes = panes.filter((pane) => bucketForPane(pane) === bucket);
    if (bucketPanes.length === 0) continue;
    result.push({
      bucket,
      items: buildWorkspaceNavigationItems({ ...options, panes: bucketPanes, globalIndexBySession }),
    });
  }
  return result;
}
