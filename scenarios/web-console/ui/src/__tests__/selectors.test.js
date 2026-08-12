import { describe, it, expect } from "vitest";
import { selectors, selectorsManifest } from "../consts/selectors";
// Cast to access nested properties - the complex generic types resolve to never
// at type-check time but work correctly at runtime.
const sel = selectors;
// [REQ:P0-001a] Responsive Pane Grid Layout - selector registry
describe("selectors registry", () => {
    it("exposes workspace paneGrid selector", () => {
        expect(sel.workspace?.paneGrid).toBe("pane-grid");
    });
    it("exposes workspace paneContainer selector", () => {
        expect(sel.workspace?.paneContainer).toBe("terminal-pane-container");
    });
    it("exposes workspace newTerminalButton selector", () => {
        expect(sel.workspace?.newTerminalButton).toBe("new-terminal-button");
    });
    it("exposes terminal pane selector", () => {
        expect(sel.terminal?.pane).toBe("terminal-pane");
    });
    it("manifest contains all literal selectors", () => {
        const keys = Object.keys(selectorsManifest.selectors);
        expect(keys).toContain("workspace.paneGrid");
        expect(keys).toContain("workspace.paneContainer");
        expect(keys).toContain("workspace.newTerminalButton");
        expect(keys).toContain("terminal.pane");
    });
    it("manifest selectors have testId and selector properties", () => {
        const entry = selectorsManifest.selectors["workspace.paneGrid"];
        expect(entry?.testId).toBe("pane-grid");
        expect(entry?.selector).toBe('[data-testid="pane-grid"]');
    });
    // [REQ:P0-006a] Terminal Launch Flow UI - selector registry
    it("exposes launcher selectors", () => {
        expect(sel.launcher?.dialog).toBe("terminal-launcher");
        expect(sel.launcher?.emptyShell).toBe("launcher-empty-shell");
        expect(sel.launcher?.customInput).toBe("launcher-custom-input");
    });
    // [REQ:P0-007a] Floating Toolbar Component - selector registry
    it("exposes mobile toolbar selector", () => {
        expect(sel.toolbar?.container).toBe("mobile-toolbar");
    });
    it("manifest contains new component selectors", () => {
        const keys = Object.keys(selectorsManifest.selectors);
        expect(keys).toContain("launcher.dialog");
        expect(keys).toContain("toolbar.container");
        expect(keys).toContain("settings.error");
        expect(keys).toContain("nav.settings");
    });
    it("exposes dynamic terminal/session selectors for BAS", () => {
        const dynamic = selectorsManifest.dynamicSelectors;
        expect(dynamic["terminal.paneBySession"]?.selectorPattern).toContain('[data-session-id="${sessionId}"]');
        expect(dynamic["workspace.paneContainerBySession"]?.selectorPattern).toContain("terminal-pane-container");
        expect(dynamic["workspace.closeButtonBySession"]?.testIdPattern).toBe("terminal-close-${sessionId}");
    });
    // Message navigator surface — BAS workflows must reference these, not raw
    // data-testid literals.
    it("exposes literal message navigator selectors", () => {
        expect(sel.messages?.navPanel).toBe("msg-jump-list");
        expect(sel.messages?.searchTrigger).toBe("messages-search-btn");
        expect(sel.messages?.navTrigger).toBe("msg-jump-trigger");
        expect(sel.messages?.searchInput).toBe("msg-nav-search");
        expect(sel.messages?.clearSearch).toBe("msg-nav-clear");
        expect(sel.messages?.resultCount).toBe("msg-nav-count");
        expect(sel.messages?.moreFilters).toBe("msg-nav-more");
        expect(sel.messages?.advancedPanel).toBe("msg-nav-advanced");
        expect(sel.messages?.emptyState).toBe("msg-nav-empty");
    });
    it("manifest contains message navigator literal selectors", () => {
        const keys = Object.keys(selectorsManifest.selectors);
        expect(keys).toContain("messages.navPanel");
        expect(keys).toContain("messages.searchInput");
        expect(keys).toContain("messages.advancedPanel");
    });
    it("exposes dynamic message navigator selectors with enum params", () => {
        const dynamic = selectorsManifest.dynamicSelectors;
        expect(dynamic["messages.navResultRow"]?.testIdPattern).toBe("msg-jump-item-${eventId}");
        expect(dynamic["messages.navChip"]?.testIdPattern).toBe("msg-nav-chip-${id}");
        expect(dynamic["messages.navChip"]?.params[0]?.values).toContain("user");
        expect(dynamic["messages.navSourceOption"]?.params[0]?.values).toContain("grok");
        expect(dynamic["messages.navStatusOption"]?.params[0]?.values).toContain("summarized");
        expect(dynamic["messages.navContentOption"]?.params[0]?.values).toContain("fileReference");
        expect(dynamic["messages.navSortOption"]?.params[0]?.values).toContain("relevance");
        expect(dynamic["messages.navGroupOption"]?.params[0]?.values).toContain("turn");
    });
});
