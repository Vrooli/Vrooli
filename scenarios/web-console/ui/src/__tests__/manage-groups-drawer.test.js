import { jsx as _jsx } from "react/jsx-runtime";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import ManageGroupsDrawer from "../components/ManageGroupsDrawer";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { HEADER_COLORS } from "../consts/config";
const mockUpdateWorkspacePane = vi.fn().mockResolvedValue(undefined);
const mockCreateTabGroup = vi.fn().mockResolvedValue({
    id: "g-new",
    name: "New Group",
    color: HEADER_COLORS[1],
    sort_order: 0,
    is_collapsed: false,
});
const mockUpdateTabGroup = vi.fn().mockResolvedValue(undefined);
const mockDeleteTabGroup = vi.fn().mockResolvedValue(undefined);
vi.mock("../api/workspace", () => ({
    saveWorkspaceLayout: vi.fn().mockResolvedValue(undefined),
    updateWorkspacePane: (...args) => mockUpdateWorkspacePane(...args),
    createTabGroup: (...args) => mockCreateTabGroup(...args),
    updateTabGroup: (...args) => mockUpdateTabGroup(...args),
    deleteTabGroup: (...args) => mockDeleteTabGroup(...args),
}));
const makePanes = (assignments) => Object.entries(assignments).map(([id, groupId]) => ({
    sessionId: id,
    name: id,
    headerColor: "transparent",
    themeId: "default",
    fontSize: 14,
    groupId,
    supportsMessagesView: false,
    manuallyUnread: false,
}));
const groupA = { id: "ga", name: "Alpha", color: HEADER_COLORS[0] ?? "#111111", isCollapsed: false };
const groupB = { id: "gb", name: "Beta", color: HEADER_COLORS[2] ?? "#222222", isCollapsed: false };
function setStore(overrides = {}) {
    useWorkspaceStore.setState({
        panes: makePanes({ s1: "ga", s2: "ga", s3: null }),
        activePane: "s1",
        groups: [groupA, groupB],
        manageGroupsTarget: { sessionId: null },
        ...overrides,
    });
}
describe("ManageGroupsDrawer", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        setStore();
    });
    it("renders nothing when manageGroupsTarget is null", () => {
        setStore({ manageGroupsTarget: null });
        render(_jsx(ManageGroupsDrawer, {}));
        expect(screen.queryByTestId("manage-groups-drawer")).toBeNull();
    });
    it("renders a dialog with one row per group and live client-derived counts", () => {
        render(_jsx(ManageGroupsDrawer, {}));
        const panel = screen.getByTestId("manage-groups-drawer");
        expect(panel.getAttribute("role")).toBe("dialog");
        expect(screen.getByTestId("manage-groups-row-ga")).toBeTruthy();
        expect(screen.getByTestId("manage-groups-row-gb")).toBeTruthy();
        // cimode: t(key, {count}) echoes the key; assert via the count testid text
        // living on the badge plus the store-derived value in the DOM structure.
        expect(screen.getByTestId("manage-groups-count-ga")).toBeTruthy();
        expect(screen.getByTestId("manage-groups-count-gb")).toBeTruthy();
    });
    it("shows the empty state when no groups exist", () => {
        setStore({ groups: [] });
        render(_jsx(ManageGroupsDrawer, {}));
        expect(screen.getByTestId("manage-groups-empty")).toBeTruthy();
    });
    it("renames a group inline and syncs to the backend", () => {
        render(_jsx(ManageGroupsDrawer, {}));
        fireEvent.click(screen.getByTestId("manage-groups-rename-ga"));
        const input = screen.getByTestId("manage-groups-rename-input");
        fireEvent.change(input, { target: { value: "Renamed" } });
        fireEvent.keyDown(input, { key: "Enter" });
        expect(useWorkspaceStore.getState().groups.find((g) => g.id === "ga")?.name).toBe("Renamed");
        expect(mockUpdateTabGroup).toHaveBeenCalledWith("ga", { name: "Renamed" });
    });
    it("recolors a group via the palette", () => {
        render(_jsx(ManageGroupsDrawer, {}));
        fireEvent.click(screen.getByTestId("manage-groups-recolor-ga"));
        const targetColor = HEADER_COLORS[3] ?? HEADER_COLORS[0];
        fireEvent.click(screen.getByTestId(`manage-groups-color-${targetColor}`));
        expect(useWorkspaceStore.getState().groups.find((g) => g.id === "ga")?.color).toBe(targetColor);
        expect(mockUpdateTabGroup).toHaveBeenCalledWith("ga", { color: targetColor });
    });
    it("deletes a group behind a consequence confirm, ungrouping its members", () => {
        render(_jsx(ManageGroupsDrawer, {}));
        fireEvent.click(screen.getByTestId("manage-groups-delete-ga"));
        // Consequence confirm appears on the confirm tier above the drawer.
        const dialog = screen.getByTestId("manage-groups-delete-confirm-dialog");
        expect(dialog).toBeTruthy();
        expect(screen.getByRole("alertdialog")).toBeTruthy();
        fireEvent.click(screen.getByTestId("manage-groups-delete-confirm-confirm"));
        const state = useWorkspaceStore.getState();
        expect(state.groups.find((g) => g.id === "ga")).toBeUndefined();
        expect(state.panes.filter((p) => p.groupId === "ga")).toHaveLength(0);
        expect(mockDeleteTabGroup).toHaveBeenCalledWith("ga");
        expect(mockUpdateWorkspacePane).toHaveBeenCalledWith("s1", { group_id: null });
        expect(mockUpdateWorkspacePane).toHaveBeenCalledWith("s2", { group_id: null });
    });
    it("cancel keeps the group and the drawer open", () => {
        render(_jsx(ManageGroupsDrawer, {}));
        fireEvent.click(screen.getByTestId("manage-groups-delete-ga"));
        fireEvent.click(screen.getByTestId("manage-groups-delete-confirm-cancel"));
        expect(screen.queryByTestId("manage-groups-delete-confirm-dialog")).toBeNull();
        expect(screen.getByTestId("manage-groups-drawer")).toBeTruthy();
        expect(useWorkspaceStore.getState().groups.find((g) => g.id === "ga")).toBeTruthy();
        expect(mockDeleteTabGroup).not.toHaveBeenCalled();
    });
    it("Escape resolves the topmost surface: confirm first, then the drawer", () => {
        render(_jsx(ManageGroupsDrawer, {}));
        fireEvent.click(screen.getByTestId("manage-groups-delete-ga"));
        fireEvent.keyDown(window, { key: "Escape" });
        expect(screen.queryByTestId("manage-groups-delete-confirm-dialog")).toBeNull();
        expect(screen.getByTestId("manage-groups-drawer")).toBeTruthy();
        fireEvent.keyDown(window, { key: "Escape" });
        expect(useWorkspaceStore.getState().manageGroupsTarget).toBeNull();
    });
    it("creates a group server-first and enters inline rename with the server id", async () => {
        render(_jsx(ManageGroupsDrawer, {}));
        fireEvent.click(screen.getByTestId("manage-groups-create"));
        await waitFor(() => {
            // Server-first: existing colors are taken, so the next distinct palette
            // color is requested and the server-generated id is adopted locally.
            expect(mockCreateTabGroup).toHaveBeenCalledWith({ name: "New Group", color: HEADER_COLORS[1] });
            expect(useWorkspaceStore.getState().groups.find((g) => g.id === "g-new")).toBeTruthy();
        });
        await waitFor(() => {
            expect(screen.getByTestId("manage-groups-rename-input")).toBeTruthy();
        });
    });
    it("with a session context, assigns and removes the session per row", () => {
        setStore({ manageGroupsTarget: { sessionId: "s3" } });
        render(_jsx(ManageGroupsDrawer, {}));
        // s3 is ungrouped: both rows offer assign. Joining seeds the group's color
        // onto the pane, and both halves of that must reach the backend — a synced
        // group_id with a stale header_color is what made the group color show up
        // on one surface and one device only.
        fireEvent.click(screen.getByTestId("manage-groups-assign-gb"));
        expect(useWorkspaceStore.getState().panes.find((p) => p.sessionId === "s3")?.groupId).toBe("gb");
        expect(mockUpdateWorkspacePane).toHaveBeenCalledWith("s3", {
            group_id: "gb",
            header_color: groupB.color,
        });
        // Row for gb now offers remove. The seeded color stays: leaving a group
        // must not silently restyle a session the user can see.
        fireEvent.click(screen.getByTestId("manage-groups-unassign-gb"));
        expect(useWorkspaceStore.getState().panes.find((p) => p.sessionId === "s3")?.groupId).toBeNull();
        expect(mockUpdateWorkspacePane).toHaveBeenCalledWith("s3", {
            group_id: null,
            header_color: groupB.color,
        });
    });
    it("hides the assign/remove toggle without a session context", () => {
        render(_jsx(ManageGroupsDrawer, {}));
        expect(screen.queryByTestId("manage-groups-assign-ga")).toBeNull();
        expect(screen.queryByTestId("manage-groups-unassign-ga")).toBeNull();
    });
});
