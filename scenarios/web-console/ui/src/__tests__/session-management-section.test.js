import { jsx as _jsx } from "react/jsx-runtime";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import SessionManagementSection from "../components/settings/SessionManagementSection";
import { strings } from "../consts/strings";
let mockUpdateSessionPolicy;
vi.mock("../api/sessions", async () => {
    const actual = await vi.importActual("../api/sessions");
    return {
        ...actual,
        updateSessionPolicy: vi.fn(),
    };
});
vi.mock("../hooks/useCountdown", () => ({
    useCountdown: vi.fn(() => null),
}));
vi.mock("../hooks/useWorkspaceSync", () => ({
    useWorkspaceSync: () => ({
        syncActivePane: vi.fn(),
        syncPaneOrder: vi.fn(),
        syncPaneUpdate: vi.fn(),
        syncCreateGroup: vi.fn(),
        syncUpdateGroup: vi.fn(),
        syncDeleteGroup: vi.fn(),
    }),
}));
const mockStoreState = {
    panes: [],
    movePaneToIndex: vi.fn(),
    setActivePane: vi.fn(),
    setPaneColor: vi.fn(),
    renamePaneById: vi.fn(),
    resetLayout: vi.fn(),
};
vi.mock("../stores/useWorkspaceStore", () => ({
    useWorkspaceStore: (selector) => selector(mockStoreState),
}));
const makeSession = (id) => ({
    id,
    shell: "/bin/bash",
    created_at: "2026-01-15T14:30:00Z",
    cols: 80,
    rows: 24,
    backend: "standard",
    survives_restart: false,
    policy: { mode: "never" },
    busy: false,
    origin: "ui",
    owner: "",
    display_label: "",
});
describe("SessionManagementSection", () => {
    const onDeleteSession = vi.fn();
    const onRequestClose = vi.fn();
    beforeEach(async () => {
        vi.clearAllMocks();
        mockStoreState.panes = [];
        const api = await import("../api/sessions");
        mockUpdateSessionPolicy = api.updateSessionPolicy;
    });
    afterEach(() => {
        vi.useRealTimers();
    });
    it("shows empty state when no panes are open", () => {
        render(_jsx(SessionManagementSection, { sessions: [], onDeleteSession: onDeleteSession, onRequestClose: onRequestClose }));
        expect(screen.getByText(strings.settings.sessionsSection.noTerminalsOpen)).toBeTruthy();
    });
    it("renders pane list when panes exist", () => {
        mockStoreState.panes = [
            { sessionId: "s1", name: "bash", headerColor: "transparent" },
            { sessionId: "s2", name: "zsh", headerColor: "#ff7a7a" },
        ];
        const sessions = [{ session: makeSession("s1") }, { session: makeSession("s2") }];
        render(_jsx(SessionManagementSection, { sessions: sessions, onDeleteSession: onDeleteSession, onRequestClose: onRequestClose }));
        expect(screen.getByTestId("sessions-pane-s1")).toBeTruthy();
        expect(screen.getByTestId("sessions-pane-s2")).toBeTruthy();
    });
    it("moves panes up and down", () => {
        mockStoreState.panes = [
            { sessionId: "s1", name: "bash", headerColor: "transparent" },
            { sessionId: "s2", name: "zsh", headerColor: "transparent" },
        ];
        const sessions = [{ session: makeSession("s1") }, { session: makeSession("s2") }];
        render(_jsx(SessionManagementSection, { sessions: sessions, onDeleteSession: onDeleteSession, onRequestClose: onRequestClose }));
        fireEvent.click(screen.getByTestId("sessions-pane-up-s2"));
        fireEvent.click(screen.getByTestId("sessions-pane-down-s1"));
        expect(mockStoreState.movePaneToIndex).toHaveBeenNthCalledWith(1, "s2", 0);
        expect(mockStoreState.movePaneToIndex).toHaveBeenNthCalledWith(2, "s1", 1);
    });
    it("focuses pane and requests close", () => {
        mockStoreState.panes = [{ sessionId: "s1", name: "bash", headerColor: "transparent" }];
        const sessions = [{ session: makeSession("s1") }];
        render(_jsx(SessionManagementSection, { sessions: sessions, onDeleteSession: onDeleteSession, onRequestClose: onRequestClose }));
        fireEvent.click(screen.getByTestId("sessions-pane-focus-s1"));
        expect(mockStoreState.setActivePane).toHaveBeenCalledWith("s1");
        expect(onRequestClose).toHaveBeenCalledOnce();
    });
    it("calls onDeleteSession when remove button is clicked", () => {
        mockStoreState.panes = [{ sessionId: "s1", name: "bash", headerColor: "transparent" }];
        const sessions = [{ session: makeSession("s1") }];
        render(_jsx(SessionManagementSection, { sessions: sessions, onDeleteSession: onDeleteSession, onRequestClose: onRequestClose }));
        fireEvent.click(screen.getByTestId("sessions-pane-remove-s1"));
        expect(onDeleteSession).toHaveBeenCalledWith("s1");
    });
    it("calls updateSessionPolicy when policy changes", async () => {
        mockUpdateSessionPolicy.mockResolvedValueOnce({});
        mockStoreState.panes = [{ sessionId: "s1", name: "bash", headerColor: "transparent" }];
        const sessions = [{ session: makeSession("s1") }];
        render(_jsx(SessionManagementSection, { sessions: sessions, onDeleteSession: onDeleteSession, onRequestClose: onRequestClose }));
        fireEvent.change(screen.getByTestId("sessions-policy-select-s1"), { target: { value: "preset:1h" } });
        await waitFor(() => {
            expect(mockUpdateSessionPolicy).toHaveBeenCalledWith("s1", {
                mode: "preset",
                duration: "1h",
            });
        });
    });
});
