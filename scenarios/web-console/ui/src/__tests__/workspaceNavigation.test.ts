import { describe, expect, it } from "vitest";
import type { ConversationEvent } from "../api/conversation";
import { buildWorkspaceNavigationItems, countWorkspaceUnreadMessages, sortPanesForView, type PaneSortMetrics } from "../lib/workspaceNavigation";
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

describe("sortPanesForView", () => {
  const metrics = new Map<string, PaneSortMetrics>([
    ["a", { name: "Charlie", activityMs: 100, unread: 0 }],
    ["b", { name: "alpha", activityMs: 300, unread: 5 }],
    ["c", { name: "Bravo", activityMs: 200, unread: 2 }],
  ]);

  it("manual mode is a stable identity passthrough", () => {
    const panes = [pane("a"), pane("b"), pane("c")];
    expect(sortPanesForView(panes, "manual", metrics)).toBe(panes);
  });

  it("name mode sorts case-insensitively by locale", () => {
    const result = sortPanesForView([pane("a"), pane("b"), pane("c")], "name", metrics);
    expect(result.map((p) => p.sessionId)).toEqual(["b", "c", "a"]); // alpha, Bravo, Charlie
  });

  it("activity mode sorts most-recent first", () => {
    const result = sortPanesForView([pane("a"), pane("b"), pane("c")], "activity", metrics);
    expect(result.map((p) => p.sessionId)).toEqual(["b", "c", "a"]); // 300, 200, 100
  });

  it("unread mode sorts by unread desc, then activity", () => {
    const result = sortPanesForView([pane("a"), pane("b"), pane("c")], "unread", metrics);
    expect(result.map((p) => p.sessionId)).toEqual(["b", "c", "a"]); // 5, 2, 0
  });

  it("sorts within group partitions, preserving block order", () => {
    // Group g1 (a, c) appears first, then ungrouped (b). Sorting by name must
    // keep g1's panes together and ahead of the ungrouped block.
    const panes = [pane("a", "g1"), pane("c", "g1"), pane("b")];
    const result = sortPanesForView(panes, "name", metrics);
    expect(result.map((p) => p.sessionId)).toEqual(["c", "a", "b"]); // Bravo, Charlie | alpha
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
      groupPosition: "first",
      isActive: true,
      unreadCount: 1,
      viewMode: "messages",
      activityLabel: "3m",
      previewText: "Hello from event",
    });
    expect(items[2]).toMatchObject({ kind: "pane", globalIndex: 1, groupPosition: "last", isActive: false });
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

describe("countWorkspaceUnreadMessages", () => {
  it("aggregates assistant unread counts across message-capable panes", () => {
    const panes = [
      pane("one"),
      pane("two"),
      { ...pane("terminal-only"), supportsMessagesView: false },
    ];

    expect(countWorkspaceUnreadMessages(panes, {
      one: {
        events: [
          event(1, "2026-05-13T12:00:00Z", "user"),
          event(2, "2026-05-13T12:01:00Z"),
          event(3, "2026-05-13T12:02:00Z"),
        ],
        cursor: { lastSeenSequence: 1, lastListenedSequence: 0 },
      },
      two: {
        events: [
          { ...event(4, "2026-05-13T12:03:00Z"), sessionId: "two" },
        ],
        cursor: { lastSeenSequence: 3, lastListenedSequence: 0 },
      },
      "terminal-only": {
        events: [
          { ...event(5, "2026-05-13T12:04:00Z"), sessionId: "terminal-only" },
        ],
        cursor: { lastSeenSequence: 0, lastListenedSequence: 0 },
      },
    })).toBe(3);
  });
});
