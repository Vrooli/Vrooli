import { describe, expect, it } from "vitest";
import type { ConversationEvent } from "../api/conversation";
import { buildWorkspaceNavigationItems, buildOriginBucketedNavigation, countWorkspaceUnreadMessages, groupIdForDropPosition, orderPanesByGroupBlocks, originBucket, sortPanesForView, type PaneSortMetrics } from "../lib/workspaceNavigation";
import type { SessionOriginName } from "../api/sessions";
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
  manuallyUnread: false,
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

describe("orderPanesByGroupBlocks", () => {
  const g1: TabGroupMeta = { id: "g1", name: "Work", color: "#ff6b6b", isCollapsed: false };

  /** Every group label the sidebar/tab strip would render, in order. */
  const groupLabels = (panes: PaneMetadata[]): string[] =>
    buildWorkspaceNavigationItems({ panes, groups: [g1], activePane: null })
      .filter((item) => item.kind === "group-label")
      .map((item) => (item.kind === "group-label" ? item.group.id : ""));

  it("returns the same array when the invariant already holds", () => {
    const panes = [pane("a", "g1"), pane("b", "g1"), pane("c")];
    expect(orderPanesByGroupBlocks(panes)).toBe(panes);
  });

  it("anchors a group at its first member and keeps ungrouped panes in place", () => {
    const panes = [pane("u1"), pane("a", "g1"), pane("u2"), pane("b", "g1")];
    expect(orderPanesByGroupBlocks(panes).map((p) => p.sessionId)).toEqual(["u1", "a", "b", "u2"]);
  });

  // These four are the ways a group used to come apart on screen. Each one
  // produced two headers for a single group, both claiming its full member
  // count. See the store tests for the write-side half of the same invariant.
  it("renders ONE header when an ungrouped pane sits between two members", () => {
    expect(groupLabels([pane("a", "g1"), pane("intruder"), pane("b", "g1")])).toEqual(["g1"]);
  });

  it("renders ONE header for members scattered across the list", () => {
    expect(groupLabels([pane("a", "g1"), pane("x"), pane("b", "g1"), pane("y"), pane("c", "g1")]))
      .toEqual(["g1"]);
  });

  it("survives a backend order that interleaves a new pane into a group", () => {
    // What `ORDER BY sort_order, created_at` returns when a freshly created
    // pane shares sort_order with the group's first member.
    expect(groupLabels([pane("a", "g1"), pane("new"), pane("b", "g1")])).toEqual(["g1"]);
  });

  it("keeps one header in every sort mode", () => {
    const scattered = [pane("a", "g1"), pane("x"), pane("b", "g1")];
    for (const mode of ["manual", "name", "activity", "unread"] as const) {
      const labels = buildWorkspaceNavigationItems({
        panes: scattered, groups: [g1], activePane: null, sortMode: mode,
      }).filter((item) => item.kind === "group-label");
      expect(labels, `sortMode=${mode}`).toHaveLength(1);
    }
  });
});

describe("groupIdForDropPosition", () => {
  const panes = [pane("a", "g1"), pane("dropped"), pane("b", "g1"), pane("tail")];

  it("joins the group when dropped strictly inside its run", () => {
    expect(groupIdForDropPosition(panes, 1, null)).toBe("g1");
  });

  it("stays in its own group when dropped against the block edge", () => {
    // Reordering to the end of your own block must not eject you from it.
    expect(groupIdForDropPosition([pane("a", "g1"), pane("b", "g1"), pane("t")], 1, "g1")).toBe("g1");
  });

  it("leaves the group when dropped clear of it", () => {
    expect(groupIdForDropPosition([pane("m", "g1"), pane("x"), pane("y")], 0, "g1")).toBeNull();
  });

  it("stays ungrouped when dropped at a block boundary", () => {
    expect(groupIdForDropPosition([pane("x"), pane("a", "g1"), pane("b", "g1")], 0, null)).toBeNull();
  });
});

describe("sortPanesForView", () => {
  const metrics = new Map<string, PaneSortMetrics>([
    ["a", { name: "Charlie", activityMs: 100, unread: 0, flagged: false }],
    ["b", { name: "alpha", activityMs: 300, unread: 5, flagged: false }],
    ["c", { name: "Bravo", activityMs: 200, unread: 2, flagged: false }],
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

describe("originBucket", () => {
  it("folds unspecified and unknown origins into programmatic", () => {
    expect(originBucket("ui")).toBe("ui");
    expect(originBucket("programmatic")).toBe("programmatic");
    expect(originBucket("remote")).toBe("remote");
    expect(originBucket("unspecified")).toBe("programmatic");
    expect(originBucket(undefined)).toBe("programmatic");
  });
});

describe("buildOriginBucketedNavigation", () => {
  const origins = (map: Record<string, SessionOriginName>) => map;

  it("returns a single ui bucket when only UI sessions exist", () => {
    const buckets = buildOriginBucketedNavigation({
      panes: [pane("a"), pane("b")],
      groups: [],
      activePane: "a",
      originBySession: origins({ a: "ui", b: "ui" }),
    });
    expect(buckets).toHaveLength(1);
    expect(buckets[0]?.bucket).toBe("ui");
    expect(buckets[0]?.items.map((i) => (i.kind === "pane" ? i.pane.sessionId : "grp"))).toEqual(["a", "b"]);
  });

  it("splits sessions into ordered buckets, folding unspecified into programmatic", () => {
    const buckets = buildOriginBucketedNavigation({
      panes: [pane("ui1"), pane("prog1"), pane("rem1"), pane("unspec1")],
      groups: [],
      activePane: "ui1",
      originBySession: origins({ ui1: "ui", prog1: "programmatic", rem1: "remote", unspec1: "unspecified" }),
    });
    // Buckets come back in ORIGIN_BUCKET_ORDER: ui, programmatic, remote.
    expect(buckets.map((b) => b.bucket)).toEqual(["ui", "programmatic", "remote"]);
    const programmatic = buckets.find((b) => b.bucket === "programmatic");
    // unspecified sessions land in the programmatic bucket alongside programmatic ones.
    expect(
      programmatic?.items.flatMap((i) => (i.kind === "pane" ? [i.pane.sessionId] : [])),
    ).toEqual(["prog1", "unspec1"]);
  });

  it("pins globalIndex to the full-list position so drag addresses the store array", () => {
    // Store order is [ui, prog, ui, prog]; the programmatic bucket renders a
    // subsequence but each pane keeps its index in the full array.
    const buckets = buildOriginBucketedNavigation({
      panes: [pane("ui0"), pane("prog1"), pane("ui2"), pane("prog3")],
      groups: [],
      activePane: "ui0",
      originBySession: origins({ ui0: "ui", prog1: "programmatic", ui2: "ui", prog3: "programmatic" }),
    });
    const programmatic = buckets.find((b) => b.bucket === "programmatic");
    const indices = programmatic?.items.flatMap((i) => (i.kind === "pane" ? [{ id: i.pane.sessionId, globalIndex: i.globalIndex }] : []));
    expect(indices).toEqual([
      { id: "prog1", globalIndex: 1 },
      { id: "prog3", globalIndex: 3 },
    ]);
  });

  it("keeps groups and collapsed-group hiding intact within a bucket", () => {
    const buckets = buildOriginBucketedNavigation({
      panes: [pane("p1", "g1"), pane("p2", "g1")],
      groups: [{ id: "g1", name: "Work", color: "#123456", isCollapsed: true }],
      activePane: "p1",
      originBySession: origins({ p1: "programmatic", p2: "programmatic" }),
    });
    const programmatic = buckets.find((b) => b.bucket === "programmatic");
    // The group label survives (tabCount 2), but its collapsed panes are hidden.
    expect(programmatic?.items).toHaveLength(1);
    expect(programmatic?.items[0]).toMatchObject({ kind: "group-label", tabCount: 2 });
  });

  it("keeps a mixed-origin group whole in one bucket", () => {
    // A group is a unit the user built by hand. Splitting it by provenance put
    // half of it behind a tab the user wasn't looking at, so the group appeared
    // to have silently lost members.
    const buckets = buildOriginBucketedNavigation({
      panes: [pane("a", "g1"), pane("b", "g1"), pane("c", "g1"), pane("solo")],
      groups: [{ id: "g1", name: "Work", color: "#123456", isCollapsed: false }],
      activePane: null,
      originBySession: origins({ a: "ui", b: "programmatic", c: "ui", solo: "programmatic" }),
    });

    const ui = buckets.find((b) => b.bucket === "ui");
    expect(ui?.items.filter((i) => i.kind === "group-label")).toHaveLength(1);
    expect(ui?.items.filter((i) => i.kind === "pane").map((i) => i.kind === "pane" && i.pane.sessionId))
      .toEqual(["a", "b", "c"]);

    // Ungrouped sessions still bucket by their own origin.
    const programmatic = buckets.find((b) => b.bucket === "programmatic");
    expect(programmatic?.items.filter((i) => i.kind === "group-label")).toHaveLength(0);
    expect(programmatic?.items.filter((i) => i.kind === "pane").map((i) => i.kind === "pane" && i.pane.sessionId))
      .toEqual(["solo"]);
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
