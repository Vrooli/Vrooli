import { describe, expect, it } from "vitest";
import { buildWorkspaceNavigationItems } from "./workspaceNavigation";
import type { PaneMetadata, RoleMeta, TabGroupMeta } from "../stores/useWorkspaceStore";

function pane(sessionId: string, groupId: string | null = null): PaneMetadata {
  return {
    sessionId,
    name: sessionId,
    headerColor: "transparent",
    themeId: "default",
    fontSize: 14,
    groupId,
    supportsMessagesView: false,
    manuallyUnread: false,
  };
}

function role(id: string, groupId: string, overrides: Partial<RoleMeta> = {}): RoleMeta {
  return {
    id,
    groupId,
    label: id,
    command: "agent",
    workingDir: "",
    incomingPrompt: "",
    backend: "",
    targetId: "",
    sessionId: null,
    sortOrder: 0,
    ...overrides,
  };
}

const group = (id: string, isCollapsed = false): TabGroupMeta => ({
  id,
  name: id,
  color: "#22d3ee",
  isCollapsed,
});

describe("buildWorkspaceNavigationItems with roles", () => {
  // Roles are additive: every caller that predates them must get exactly its
  // previous output.
  it("emits nothing new when there are no roles", () => {
    const items = buildWorkspaceNavigationItems({
      panes: [pane("s1", "g1")],
      groups: [group("g1")],
      activePane: "s1",
    });
    expect(items.map((i) => i.kind)).toEqual(["group-label", "pane"]);
  });

  it("emits a waiting role after its group's panes", () => {
    const items = buildWorkspaceNavigationItems({
      panes: [pane("s1", "g1"), pane("s2")],
      groups: [group("g1")],
      roles: [role("r1", "g1")],
      activePane: "s1",
    });
    expect(items.map((i) => i.kind)).toEqual(["group-label", "pane", "waiting-role", "pane"]);
  });

  it("flushes waiting roles for the group the pane list ends inside", () => {
    const items = buildWorkspaceNavigationItems({
      panes: [pane("s0"), pane("s1", "g1")],
      groups: [group("g1")],
      roles: [role("r1", "g1")],
      activePane: "s1",
    });
    expect(items.map((i) => i.kind)).toEqual(["pane", "group-label", "pane", "waiting-role"]);
  });

  // The failure this exists to prevent: a group whose roles have all yet to
  // start owns no pane, so a pane-driven loop never reaches it — and an
  // invisible group cannot be started, renamed, or closed.
  it("renders a group whose roles have all yet to start", () => {
    const items = buildWorkspaceNavigationItems({
      panes: [],
      groups: [group("g1")],
      roles: [role("r1", "g1"), role("r2", "g1", { sortOrder: 1 })],
      activePane: null,
    });
    expect(items.map((i) => i.kind)).toEqual(["group-label", "waiting-role", "waiting-role"]);
    const label = items[0];
    if (label?.kind !== "group-label") throw new Error("expected a group label first");
    expect(label.waitingCount).toBe(2);
    expect(label.tabCount).toBe(0);
  });

  // A running role already appears as its pane. Emitting it again would draw
  // every started agent twice.
  it("never emits a running role as its own row", () => {
    const items = buildWorkspaceNavigationItems({
      panes: [pane("s1", "g1")],
      groups: [group("g1")],
      roles: [role("r1", "g1", { sessionId: "s1" })],
      activePane: "s1",
    });
    expect(items.filter((i) => i.kind === "waiting-role")).toHaveLength(0);
  });

  it("orders waiting roles by sort order, then id, so reads are stable", () => {
    const items = buildWorkspaceNavigationItems({
      panes: [pane("s1", "g1")],
      groups: [group("g1")],
      roles: [
        role("rc", "g1", { sortOrder: 1 }),
        role("ra", "g1", { sortOrder: 0 }),
        role("rb", "g1", { sortOrder: 0 }),
      ],
      activePane: "s1",
    });
    const ids = items.filter((i) => i.kind === "waiting-role").map((i) => (i.kind === "waiting-role" ? i.role.id : ""));
    expect(ids).toEqual(["ra", "rb", "rc"]);
  });

  it("hides waiting roles in a collapsed group, like every other member", () => {
    const items = buildWorkspaceNavigationItems({
      panes: [pane("s1", "g1")],
      groups: [group("g1", true)],
      roles: [role("r1", "g1")],
      activePane: "s1",
    });
    expect(items.filter((i) => i.kind === "waiting-role")).toHaveLength(0);
  });

  it("marks the last waiting role so the group block can close its border", () => {
    const items = buildWorkspaceNavigationItems({
      panes: [pane("s1", "g1")],
      groups: [group("g1")],
      roles: [role("r1", "g1"), role("r2", "g1", { sortOrder: 1 })],
      activePane: "s1",
    });
    const roles = items.filter((i) => i.kind === "waiting-role");
    expect(roles.map((i) => (i.kind === "waiting-role" ? i.isLastInGroup : null))).toEqual([false, true]);
  });

  it("does not emit a group twice when it has both panes and waiting roles", () => {
    const items = buildWorkspaceNavigationItems({
      panes: [pane("s1", "g1"), pane("s2", "g1")],
      groups: [group("g1")],
      roles: [role("r1", "g1")],
      activePane: "s1",
    });
    expect(items.filter((i) => i.kind === "group-label")).toHaveLength(1);
  });

  it("keeps two groups' waiting roles in their own blocks", () => {
    const items = buildWorkspaceNavigationItems({
      panes: [pane("s1", "g1"), pane("s2", "g2")],
      groups: [group("g1"), group("g2")],
      roles: [role("r1", "g1"), role("r2", "g2")],
      activePane: "s1",
    });
    expect(items.map((i) => i.kind)).toEqual([
      "group-label", "pane", "waiting-role",
      "group-label", "pane", "waiting-role",
    ]);
  });
});
