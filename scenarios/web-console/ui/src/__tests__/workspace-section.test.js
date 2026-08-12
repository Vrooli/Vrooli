import { jsx as _jsx } from "react/jsx-runtime";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { i18n } from "../i18n";
import WorkspaceSection from "../components/settings/WorkspaceSection";
const mockStoreState = {
    isMinimapVisible: true,
    setMinimapVisible: vi.fn(),
    displayMode: "grid",
    setDisplayMode: vi.fn(),
    toolbarLayout: "expanded",
    setToolbarLayout: vi.fn(),
    keepScreenAwake: true,
    setKeepScreenAwake: vi.fn(),
};
let mockWakeLockStatus = "active";
vi.mock("../stores/useWorkspaceStore", () => ({
    useWorkspaceStore: (selector) => selector(mockStoreState),
}));
vi.mock("../stores/useWakeLockStatus", () => ({
    useWakeLockStatus: (selector) => selector({ status: mockWakeLockStatus }),
}));
describe("WorkspaceSection", () => {
    beforeEach(async () => {
        vi.clearAllMocks();
        mockStoreState.keepScreenAwake = true;
        mockWakeLockStatus = "active";
        await i18n.changeLanguage("en");
    });
    it("renders keep-screen-awake toggle checked when enabled", () => {
        render(_jsx(WorkspaceSection, {}));
        const toggle = screen.getByTestId("keep-screen-awake-toggle");
        expect(toggle).toHaveAttribute("aria-checked", "true");
    });
    it("renders keep-screen-awake toggle unchecked when disabled", () => {
        mockStoreState.keepScreenAwake = false;
        render(_jsx(WorkspaceSection, {}));
        const toggle = screen.getByTestId("keep-screen-awake-toggle");
        expect(toggle).toHaveAttribute("aria-checked", "false");
    });
    it("calls setKeepScreenAwake when toggle is clicked", () => {
        render(_jsx(WorkspaceSection, {}));
        const toggle = screen.getByTestId("keep-screen-awake-toggle");
        fireEvent.click(toggle);
        expect(mockStoreState.setKeepScreenAwake).toHaveBeenCalledWith(false);
    });
    it("selects sidebar display mode", () => {
        render(_jsx(WorkspaceSection, {}));
        fireEvent.click(screen.getByTestId("display-mode-sidebar"));
        expect(mockStoreState.setDisplayMode).toHaveBeenCalledWith("sidebar");
    });
    it("shows unsupported hint when wake lock API is not available", () => {
        mockWakeLockStatus = "unsupported";
        render(_jsx(WorkspaceSection, {}));
        expect(screen.getByText(/doesn't support screen wake lock/i)).toBeTruthy();
    });
    it("shows denied hint when wake lock is denied", () => {
        mockWakeLockStatus = "denied";
        render(_jsx(WorkspaceSection, {}));
        expect(screen.getByText(/was denied/i)).toBeTruthy();
    });
    it("shows active hint with accent color when status is active", () => {
        mockWakeLockStatus = "active";
        render(_jsx(WorkspaceSection, {}));
        const hint = screen.getByText(/Screen is being kept awake/i);
        expect(hint).toBeTruthy();
        expect(hint.className).toContain("text-wc-accent");
    });
    it("shows default hint when toggle is off regardless of status", () => {
        mockStoreState.keepScreenAwake = false;
        mockWakeLockStatus = "unsupported";
        render(_jsx(WorkspaceSection, {}));
        expect(screen.getByText(/Prevent the device from dimming/i)).toBeTruthy();
    });
    it("shows denied hint with error color", () => {
        mockWakeLockStatus = "denied";
        render(_jsx(WorkspaceSection, {}));
        const hint = screen.getByText(/was denied/i);
        expect(hint.className).toContain("text-wc-error");
    });
    it("shows re-acquiring hint with amber color when released", () => {
        mockWakeLockStatus = "released";
        render(_jsx(WorkspaceSection, {}));
        const hint = screen.getByText(/Re-acquiring/i);
        expect(hint).toBeTruthy();
        expect(hint.className).toContain("text-yellow-500");
    });
});
