import type { ConversationCursor, ConversationEvent } from "../api/conversation";
import { formatRelativeTime, stripMarkdown } from "../components/MessageJumpList.helpers";
import type { PaneViewMode } from "../stores/useConversationStore";
import type { PaneMetadata, TabGroupMeta } from "../stores/useWorkspaceStore";

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
}: BuildWorkspaceNavigationItemsOptions): WorkspaceNavigationItem[] {
  const groupMap = new Map(groups.map((group) => [group.id, group]));
  const items: WorkspaceNavigationItem[] = [];
  let lastGroupId: string | null | undefined = undefined;

  panes.forEach((pane, idx) => {
    const groupId = pane.groupId;
    const group = groupId ? groupMap.get(groupId) : undefined;

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
