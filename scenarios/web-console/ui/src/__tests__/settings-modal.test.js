import { jsx as _jsx } from "react/jsx-runtime";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import SettingsModal from "../components/SettingsModal";
const mockStoreState = {
    settingsModalOpen: true,
    setSettingsModalOpen: vi.fn(),
};
const mediaQueryState = {
    isMobile: false,
};
vi.mock("../stores/useWorkspaceStore", () => ({
    useWorkspaceStore: (selector) => selector(mockStoreState),
}));
vi.mock("../hooks/useMediaQuery", () => ({
    useMediaQuery: () => mediaQueryState.isMobile,
}));
vi.mock("../hooks/useDraggablePosition", () => ({
    useDraggablePosition: () => ({
        elementRef: { current: null },
        floatingStyle: { transform: "translate3d(100px, 100px, 0)" },
        pointerHandlers: {
            onPointerDown: vi.fn(),
            onPointerMove: vi.fn(),
            onPointerUp: vi.fn(),
            onPointerCancel: vi.fn(),
        },
        handleClickCapture: vi.fn(),
    }),
}));
vi.mock("../components/settings/SessionManagementSection", () => ({
    default: () => _jsx("div", { "data-testid": "sessions-section", children: "Sessions section" }),
}));
vi.mock("../components/settings/WorkspaceSection", () => ({
    default: () => _jsx("div", { "data-testid": "workspace-section", children: "Workspace section" }),
}));
vi.mock("../components/settings/VoiceInputSection", () => ({
    default: () => _jsx("div", { "data-testid": "voice-input-section", children: "Voice input section" }),
}));
vi.mock("../components/settings/TtsSettingsSection", () => ({
    default: () => _jsx("div", { "data-testid": "voice-output-section", children: "Voice output section" }),
}));
vi.mock("../components/settings/ShortcutProfilesSection", () => ({
    default: () => _jsx("div", { "data-testid": "shortcuts-section", children: "Shortcuts section" }),
}));
vi.mock("../components/settings/NewPaneDefaultsSection", () => ({
    default: () => _jsx("div", { "data-testid": "defaults-section", children: "Defaults section" }),
}));
vi.mock("../components/settings/IntegrationsSection", () => ({
    default: () => _jsx("div", { "data-testid": "integrations-section", children: "Integrations section" }),
}));
describe("SettingsModal", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        mockStoreState.settingsModalOpen = true;
        mediaQueryState.isMobile = false;
    });
    it("does not render when closed", () => {
        mockStoreState.settingsModalOpen = false;
        render(_jsx(SettingsModal, { sessions: [], onDeleteSession: vi.fn() }));
        expect(screen.queryByTestId("settings-modal")).toBeNull();
    });
    it("renders desktop shell with sidebar by default", () => {
        render(_jsx(SettingsModal, { sessions: [], onDeleteSession: vi.fn() }));
        expect(screen.getByTestId("settings-modal")).toBeTruthy();
        expect(screen.getByTestId("settings-sidebar")).toBeTruthy();
        expect(screen.getByTestId("workspace-section")).toBeTruthy();
    });
    it("switches sections when a desktop tab is clicked", () => {
        render(_jsx(SettingsModal, { sessions: [], onDeleteSession: vi.fn() }));
        fireEvent.click(screen.getByTestId("settings-tab-sessions"));
        expect(screen.getByTestId("sessions-section")).toBeTruthy();
    });
    it("closes on backdrop click, not on panel click", () => {
        render(_jsx(SettingsModal, { sessions: [], onDeleteSession: vi.fn() }));
        const panel = screen.getByTestId("settings-modal");
        fireEvent.click(panel);
        expect(mockStoreState.setSettingsModalOpen).not.toHaveBeenCalled();
        const backdrop = panel.parentElement?.firstElementChild;
        fireEvent.click(backdrop);
        expect(mockStoreState.setSettingsModalOpen).toHaveBeenCalledWith(false);
    });
    it("closes on Escape and renders dialog semantics", () => {
        render(_jsx(SettingsModal, { sessions: [], onDeleteSession: vi.fn() }));
        const panel = screen.getByTestId("settings-modal");
        expect(panel.getAttribute("role")).toBe("dialog");
        expect(panel.getAttribute("aria-modal")).toBe("true");
        fireEvent.keyDown(window, { key: "Escape" });
        expect(mockStoreState.setSettingsModalOpen).toHaveBeenCalledWith(false);
    });
    it("traps focus inside the settings dialog", () => {
        render(_jsx(SettingsModal, { sessions: [], onDeleteSession: vi.fn() }));
        const panel = screen.getByTestId("settings-modal");
        const tab = screen.getByTestId("settings-tab-sessions");
        tab.focus();
        fireEvent.keyDown(panel, { key: "Tab" });
        expect(panel.contains(document.activeElement)).toBe(true);
        fireEvent.keyDown(panel, { key: "Tab", shiftKey: true });
        expect(panel.contains(document.activeElement)).toBe(true);
    });
    it("renders mobile tabs row on mobile", () => {
        mediaQueryState.isMobile = true;
        render(_jsx(SettingsModal, { sessions: [], onDeleteSession: vi.fn() }));
        expect(screen.getByTestId("settings-tabs-row")).toBeTruthy();
        expect(screen.queryByTestId("settings-sidebar")).toBeNull();
    });
    it("keeps the mobile sheet below the top safe area", () => {
        mediaQueryState.isMobile = true;
        render(_jsx(SettingsModal, { sessions: [], onDeleteSession: vi.fn() }));
        expect(screen.getByTestId("settings-modal").className).toContain("--wc-safe-top");
    });
});
