import { jsx as _jsx } from "react/jsx-runtime";
import { render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useTabLikeNavigationShortcuts } from "../useTabLikeNavigationShortcuts";
const panes = [
    {
        sessionId: "one",
        name: "One",
        headerColor: "transparent",
        themeId: "default",
        fontSize: 14,
        groupId: null,
        supportsMessagesView: false, manuallyUnread: false,
    },
    {
        sessionId: "two",
        name: "Two",
        headerColor: "transparent",
        themeId: "default",
        fontSize: 14,
        groupId: null,
        supportsMessagesView: false, manuallyUnread: false,
    },
];
function Harness({ enabled, activePane = "one", onActivatePane, onClosePane, }) {
    useTabLikeNavigationShortcuts({
        enabled,
        panes,
        activePane,
        onActivatePane,
        onClosePane,
    });
    return null;
}
describe("useTabLikeNavigationShortcuts", () => {
    it("cycles, jumps, and closes when enabled", () => {
        const onActivatePane = vi.fn();
        const onClosePane = vi.fn();
        render(_jsx(Harness, { enabled: true, onActivatePane: onActivatePane, onClosePane: onClosePane }));
        window.dispatchEvent(new KeyboardEvent("keydown", { key: "Tab", ctrlKey: true }));
        expect(onActivatePane).toHaveBeenLastCalledWith("two");
        window.dispatchEvent(new KeyboardEvent("keydown", { key: "2", ctrlKey: true }));
        expect(onActivatePane).toHaveBeenLastCalledWith("two");
        window.dispatchEvent(new KeyboardEvent("keydown", { key: "w", ctrlKey: true }));
        expect(onClosePane).toHaveBeenCalledWith("one");
    });
    it("does not handle shortcuts when disabled", () => {
        const onActivatePane = vi.fn();
        const onClosePane = vi.fn();
        render(_jsx(Harness, { enabled: false, onActivatePane: onActivatePane, onClosePane: onClosePane }));
        window.dispatchEvent(new KeyboardEvent("keydown", { key: "Tab", ctrlKey: true }));
        window.dispatchEvent(new KeyboardEvent("keydown", { key: "w", ctrlKey: true }));
        expect(onActivatePane).not.toHaveBeenCalled();
        expect(onClosePane).not.toHaveBeenCalled();
    });
});
