import { describe, expect, it } from "vitest";
import type { ConversationEvent } from "../api/conversation";
import { buildWorkspaceNavigationItems } from "../lib/workspaceNavigation";
import { isTabLikeDisplayMode } from "../lib/workspaceDisplayMode";
import type { PaneMetadata, TabGroupMeta } from "../stores/useWorkspaceStore";

const pane = (sessionId: string, groupId: string | null = null): PaneMetadata => ({
  sessionId,
  name: sessionId,
  headerColor: "transparent",
  themeId: "default",
  fontSize: 14,
  groupId,
  supportsMessagesView: true,
});

const event = (sequence: number, createdAt: string, role: "assistant" | "user" = "assistant"): ConversationEvent => ({
  id: `evt-${sequence}`,
  sessionId: "one",
  source: "test",
  role,
  text: "**Hello** from event",
  speechParagraphs: [],
  summarized: false,
  createdAt,
  sequence,
  deliveryState: "delivered",
  ttsState: "idle",
  consumptionState: "new",
});

describe("workspace display mode helpers", () => {
  it("treats tabs and sidebar as tab-like modes", () => {
    expect(isTabLikeDisplayMode("grid")).toBe(false);
    expect(isTabLikeDisplayMode("tabs")).toBe(true);
    expect(isTabLikeDisplayMode("sidebar")).toBe(true);
  });
});

describe("buildWorkspaceNavigationItems", () => {
  it("preserves group labels, pane order, unread counts, and latest activity", () => {
    const group: TabGroupMeta = {
      id: "grp",
      name: "Work",
      color: "#123456",
      isCollapsed: false,
    };
    const items = buildWorkspaceNavigationItems({
      panes: [pane("one", "grp"), pane("two", "grp")],
      groups: [group],
      activePane: "one",
      conversationSessions: {
        one: {
          events: [
            event(1, "2026-05-13T12:00:00Z", "user"),
            event(2, "2026-05-13T12:05:00Z"),
          ],
          cursor: { lastSeenSequence: 1, lastListenedSequence: 0 },
        },
      },
      viewModes: { one: "messages" },
      now: new Date("2026-05-13T12:08:00Z"),
    });

    expect(items[0]).toMatchObject({ kind: "group-label", group, tabCount: 2 });
    expect(items[1]).toMatchObject({
      kind: "pane",
      globalIndex: 0,
      isActive: true,
      unreadCount: 1,
      viewMode: "messages",
      activityLabel: "3m",
      previewText: "Hello from event",
    });
    expect(items[2]).toMatchObject({ kind: "pane", globalIndex: 1, isActive: false });
  });

  it("hides panes in collapsed groups and falls back to visited activity", () => {
    const items = buildWorkspaceNavigationItems({
      panes: [pane("one", "grp"), pane("two")],
      groups: [{ id: "grp", name: "Hidden", color: "#000", isCollapsed: true }],
      activePane: "two",
      conversationSessions: {},
      viewModes: {},
      lastVisitedBySession: { two: "2026-05-13T12:00:00Z" },
      now: new Date("2026-05-13T12:10:00Z"),
    });

    expect(items).toHaveLength(2);
    expect(items[0]).toMatchObject({ kind: "group-label", tabCount: 1 });
    expect(items[1]).toMatchObject({
      kind: "pane",
      globalIndex: 1,
      activityLabel: "Visited 10m",
    });
  });
});
