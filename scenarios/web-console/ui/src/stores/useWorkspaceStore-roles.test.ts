import { beforeEach, describe, expect, it } from "vitest";
import { useWorkspaceStore, type RoleMeta } from "./useWorkspaceStore";
import { captureGroupSnapshot, emptyGroups, isGroupClosable } from "../lib/groupLifecycle";

function role(overrides: Partial<RoleMeta> & { id: string; groupId: string }): RoleMeta {
  return {
    label: "Role",
    command: "",
    workingDir: "",
    incomingPrompt: "",
    backend: "",
    targetId: "",
    sessionId: null,
    sortOrder: 0,
    ...overrides,
  };
}

describe("workspace store roles", () => {
  beforeEach(() => {
    useWorkspaceStore.setState({ panes: [], groups: [], roles: [], closedGroupUndo: null });
  });

  it("adds, updates, and removes roles", () => {
    const store = useWorkspaceStore.getState();
    store.addRole(role({ id: "r1", groupId: "g1", label: "Planner" }));
    store.addRole(role({ id: "r2", groupId: "g1", label: "Implementer", sortOrder: 1 }));
    expect(useWorkspaceStore.getState().roles).toHaveLength(2);

    useWorkspaceStore.getState().updateRole("r2", { label: "Builder" });
    expect(useWorkspaceStore.getState().roles.find((r) => r.id === "r2")?.label).toBe("Builder");

    useWorkspaceStore.getState().removeRole("r1");
    expect(useWorkspaceStore.getState().roles.map((r) => r.id)).toEqual(["r2"]);
  });

  it("derives running and waiting from the session pointer alone", () => {
    const store = useWorkspaceStore.getState();
    store.addRole(role({ id: "r1", groupId: "g1" }));
    expect(useWorkspaceStore.getState().roles[0]?.sessionId).toBeNull();

    useWorkspaceStore.getState().setRoleSession("r1", "sess-1");
    expect(useWorkspaceStore.getState().roles[0]?.sessionId).toBe("sess-1");

    // Clearing returns it to waiting — the same field, no second flag to
    // fall out of sync with.
    useWorkspaceStore.getState().setRoleSession("r1", null);
    expect(useWorkspaceStore.getState().roles[0]?.sessionId).toBeNull();
  });

  it("drops a group's roles when the group is removed, matching the database cascade", () => {
    const store = useWorkspaceStore.getState();
    store.addGroup({ id: "g1", name: "Doomed", color: "#f00", isCollapsed: false });
    store.addGroup({ id: "g2", name: "Kept", color: "#0f0", isCollapsed: false });
    useWorkspaceStore.getState().addRole(role({ id: "r1", groupId: "g1" }));
    useWorkspaceStore.getState().addRole(role({ id: "r2", groupId: "g2" }));

    useWorkspaceStore.getState().removeGroup("g1");
    expect(useWorkspaceStore.getState().roles.map((r) => r.id)).toEqual(["r2"]);
  });

  it("holds an ordered list of any length, not a pair", () => {
    const store = useWorkspaceStore.getState();
    for (let i = 0; i < 5; i++) {
      store.addRole(role({ id: `r${i}`, groupId: "g1", sortOrder: i, label: `Role ${i}` }));
    }
    const roles = useWorkspaceStore.getState().roles.filter((r) => r.groupId === "g1");
    expect(roles).toHaveLength(5);
    expect(roles.map((r) => r.sortOrder)).toEqual([0, 1, 2, 3, 4]);
  });
});

describe("isGroupClosable", () => {
  it("closes a group with no panes and no waiting roles", () => {
    expect(isGroupClosable("g1", [], [])).toBe(true);
  });

  it("keeps a group that still holds a pane", () => {
    const panes = [{ sessionId: "s1", groupId: "g1" }] as never;
    expect(isGroupClosable("g1", panes, [])).toBe(false);
  });

  // The exemption is the whole safety argument: a templated group whose roles
  // have not started yet is empty AND intentional.
  it("keeps a group that holds a waiting role", () => {
    expect(isGroupClosable("g1", [], [role({ id: "r1", groupId: "g1" })])).toBe(false);
  });

  it("ignores waiting roles belonging to another group", () => {
    expect(isGroupClosable("g1", [], [role({ id: "r1", groupId: "g2" })])).toBe(true);
  });

  // A running role has a session, and that session has a pane, so the pane
  // check already covers it — there is no second clause to keep in sync.
  it("closes a group whose only role is running but whose pane is gone", () => {
    expect(isGroupClosable("g1", [], [role({ id: "r1", groupId: "g1", sessionId: "s1" })])).toBe(true);
  });
});

describe("captureGroupSnapshot", () => {
  it("captures the group, its roles, its members, and its position", () => {
    const groups = [
      { id: "g0", name: "First", color: "#000", isCollapsed: false },
      { id: "g1", name: "Ship it", color: "#22d3ee", isCollapsed: false },
    ];
    const panes = [
      { sessionId: "s1", groupId: "g1" },
      { sessionId: "s2", groupId: "g0" },
    ] as never;
    const roles = [role({ id: "r1", groupId: "g1", label: "Implementer" })];

    const snap = captureGroupSnapshot("g1", groups, panes, roles);
    expect(snap).not.toBeNull();
    expect(snap?.group.name).toBe("Ship it");
    expect(snap?.roles.map((r) => r.id)).toEqual(["r1"]);
    expect(snap?.memberSessionIds).toEqual(["s1"]);
    expect(snap?.sortIndex).toBe(1);
  });

  it("returns null for a group that is already gone", () => {
    expect(captureGroupSnapshot("missing", [], [], [])).toBeNull();
  });

  it("copies rather than aliases, so the snapshot survives the delete", () => {
    const groups = [{ id: "g1", name: "Ship it", color: "#22d3ee", isCollapsed: false }];
    const roles = [role({ id: "r1", groupId: "g1", label: "Implementer" })];
    const snap = captureGroupSnapshot("g1", groups, [], roles);
    roles[0]!.label = "mutated after capture";
    expect(snap?.roles[0]?.label).toBe("Implementer");
  });
});

describe("emptyGroups", () => {
  it("lists only groups with no panes", () => {
    const groups = [
      { id: "g1", name: "Busy", color: "#000", isCollapsed: false },
      { id: "g2", name: "Empty", color: "#000", isCollapsed: false },
    ];
    const panes = [{ sessionId: "s1", groupId: "g1" }] as never;
    expect(emptyGroups(groups, panes).map((g) => g.id)).toEqual(["g2"]);
  });
});
