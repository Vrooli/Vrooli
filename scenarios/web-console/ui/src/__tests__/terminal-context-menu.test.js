import { jsx as _jsx } from "react/jsx-runtime";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, act } from "@testing-library/react";
import TerminalContextMenu from "../components/TerminalContextMenu";
import { strings } from "../consts/strings";
import { i18n } from "../i18n";
const defaultProps = () => ({
    position: { x: 200, y: 300 },
    hasSelection: false,
    onCopy: vi.fn(),
    // Default onPaste resolves to {status: "ok"} — tests override for
    // failure / pending scenarios.
    onPaste: vi.fn((_text) => Promise.resolve({ status: "ok" })),
    onSelectAll: vi.fn(),
    onClear: vi.fn(),
    onUploadImage: vi.fn(),
    onClose: vi.fn(),
});
describe("TerminalContextMenu", () => {
    beforeEach(() => {
        // Default: clipboard.readText succeeds
        Object.assign(navigator, {
            clipboard: {
                readText: vi.fn().mockResolvedValue("pasted text"),
                writeText: vi.fn().mockResolvedValue(undefined),
            },
        });
    });
    afterEach(() => {
        vi.restoreAllMocks();
    });
    it("renders Paste, Select All, and Clear items", () => {
        render(_jsx(TerminalContextMenu, { ...defaultProps() }));
        expect(screen.getByTestId("ctx-paste")).toBeInTheDocument();
        expect(screen.getByTestId("ctx-select-all")).toBeInTheDocument();
        expect(screen.getByTestId("ctx-clear")).toBeInTheDocument();
    });
    it("hides Copy when hasSelection is false", () => {
        render(_jsx(TerminalContextMenu, { ...defaultProps() }));
        expect(screen.queryByTestId("ctx-copy")).not.toBeInTheDocument();
    });
    it("shows Copy when hasSelection is true", () => {
        render(_jsx(TerminalContextMenu, { ...defaultProps(), hasSelection: true }));
        expect(screen.getByTestId("ctx-copy")).toBeInTheDocument();
    });
    it("calls onCopy when Copy is clicked", () => {
        const props = defaultProps();
        render(_jsx(TerminalContextMenu, { ...props, hasSelection: true }));
        fireEvent.click(screen.getByTestId("ctx-copy"));
        expect(props.onCopy).toHaveBeenCalledOnce();
    });
    it("reads clipboard, calls onPaste, shows 'Pasting…' then 'Pasted' and auto-closes", async () => {
        vi.useFakeTimers();
        const props = defaultProps();
        render(_jsx(TerminalContextMenu, { ...props }));
        await act(async () => {
            fireEvent.click(screen.getByTestId("ctx-paste"));
            // Let the clipboard + onPaste promises resolve.
            await Promise.resolve();
            await Promise.resolve();
        });
        expect(navigator.clipboard.readText).toHaveBeenCalled();
        expect(props.onPaste).toHaveBeenCalledWith("pasted text");
        // After settle, the button transitions to "Pasted" before close.
        expect(screen.getByTestId("ctx-paste").textContent).toBe(strings.terminalContextMenu.pasted);
        // Advance through the success-flash window; onClose fires.
        await act(async () => {
            vi.advanceTimersByTime(700);
        });
        expect(props.onClose).toHaveBeenCalled();
        vi.useRealTimers();
    });
    it("shows typed failure reason when onPaste resolves to failed", async () => {
        // Opt into the real `en` locale so the {{reason}} token in the
        // pasteFailed string actually gets interpolated. cimode would
        // otherwise return the raw key path.
        await i18n.changeLanguage("en");
        vi.useFakeTimers();
        const props = defaultProps();
        props.onPaste = vi.fn().mockResolvedValue({
            status: "failed",
            reason: "tmux_write_failed",
        });
        render(_jsx(TerminalContextMenu, { ...props }));
        await act(async () => {
            fireEvent.click(screen.getByTestId("ctx-paste"));
            await Promise.resolve();
            await Promise.resolve();
        });
        expect(screen.getByTestId("ctx-paste").textContent).toBe("Paste failed: tmux_write_failed");
        // Menu stays open for the failure-hold window, then closes.
        await act(async () => {
            vi.advanceTimersByTime(3100);
        });
        expect(props.onClose).toHaveBeenCalled();
        vi.useRealTimers();
    });
    it("marks the paste button as disabled while pending", async () => {
        // Never-resolving onPaste — keeps the pending state visible.
        const props = defaultProps();
        props.onPaste = vi.fn().mockReturnValue(new Promise(() => { }));
        render(_jsx(TerminalContextMenu, { ...props }));
        await act(async () => {
            fireEvent.click(screen.getByTestId("ctx-paste"));
            await Promise.resolve();
            await Promise.resolve();
        });
        const btn = screen.getByTestId("ctx-paste");
        expect(btn.textContent).toBe(strings.terminalContextMenu.pasting);
        expect(btn).toBeDisabled();
        expect(btn.getAttribute("data-paste-state")).toBe("pending");
    });
    it("shows fallback text when clipboard read fails", async () => {
        vi.spyOn(navigator.clipboard, "readText").mockRejectedValue(new DOMException("denied"));
        const props = defaultProps();
        render(_jsx(TerminalContextMenu, { ...props }));
        await act(async () => {
            fireEvent.click(screen.getByTestId("ctx-paste"));
        });
        expect(props.onPaste).not.toHaveBeenCalled();
        expect(screen.getByTestId("ctx-paste").textContent).toBe(strings.terminalContextMenu.useCtrlVHint);
    });
    it("calls onSelectAll when Select All is clicked", () => {
        const props = defaultProps();
        render(_jsx(TerminalContextMenu, { ...props }));
        fireEvent.click(screen.getByTestId("ctx-select-all"));
        expect(props.onSelectAll).toHaveBeenCalledOnce();
    });
    it("calls onClear when Clear Terminal is clicked", () => {
        const props = defaultProps();
        render(_jsx(TerminalContextMenu, { ...props }));
        fireEvent.click(screen.getByTestId("ctx-clear"));
        expect(props.onClear).toHaveBeenCalledOnce();
    });
    it("dismisses on Escape key", () => {
        const props = defaultProps();
        render(_jsx(TerminalContextMenu, { ...props }));
        fireEvent.keyDown(window, { key: "Escape" });
        expect(props.onClose).toHaveBeenCalledOnce();
    });
    it("dismisses on backdrop click", () => {
        const props = defaultProps();
        render(_jsx(TerminalContextMenu, { ...props }));
        fireEvent.click(screen.getByTestId("ctx-backdrop"));
        expect(props.onClose).toHaveBeenCalledOnce();
    });
    it("renders all four actions when hasSelection is true", () => {
        render(_jsx(TerminalContextMenu, { ...defaultProps(), hasSelection: true }));
        expect(screen.getByTestId("ctx-copy")).toBeInTheDocument();
        expect(screen.getByTestId("ctx-paste")).toBeInTheDocument();
        expect(screen.getByTestId("ctx-select-all")).toBeInTheDocument();
        expect(screen.getByTestId("ctx-clear")).toBeInTheDocument();
    });
    it("does not call onPaste when clipboard returns empty string", async () => {
        vi.spyOn(navigator.clipboard, "readText").mockResolvedValue("");
        const props = defaultProps();
        render(_jsx(TerminalContextMenu, { ...props }));
        await act(async () => {
            fireEvent.click(screen.getByTestId("ctx-paste"));
            await Promise.resolve();
        });
        expect(props.onPaste).not.toHaveBeenCalled();
        expect(props.onClose).toHaveBeenCalled();
    });
    it("positions the menu at the given coordinates", () => {
        render(_jsx(TerminalContextMenu, { ...defaultProps() }));
        const menu = screen.getByTestId("terminal-context-menu");
        // Before measurement, menu renders at position with opacity 0
        expect(menu.style.left).toBe("200px");
        expect(menu.style.top).toBe("300px");
    });
    it("renders Upload Image button when onUploadImage is provided", () => {
        render(_jsx(TerminalContextMenu, { ...defaultProps() }));
        expect(screen.getByTestId("ctx-upload-image")).toBeInTheDocument();
    });
    it("calls onUploadImage when Upload Image is clicked", () => {
        const props = defaultProps();
        render(_jsx(TerminalContextMenu, { ...props }));
        fireEvent.click(screen.getByTestId("ctx-upload-image"));
        expect(props.onUploadImage).toHaveBeenCalledOnce();
    });
    it("hides Upload Image when onUploadImage is undefined", () => {
        const props = defaultProps();
        const { onUploadImage: _unused, ...propsWithout } = props;
        void _unused;
        render(_jsx(TerminalContextMenu, { ...propsWithout }));
        expect(screen.queryByTestId("ctx-upload-image")).not.toBeInTheDocument();
    });
    it("renders Speak button when hasSelection and onSpeak provided", () => {
        render(_jsx(TerminalContextMenu, { ...defaultProps(), hasSelection: true, onSpeak: vi.fn() }));
        expect(screen.getByTestId("ctx-speak")).toBeInTheDocument();
    });
    it("hides Speak when no selection", () => {
        render(_jsx(TerminalContextMenu, { ...defaultProps(), hasSelection: false, onSpeak: vi.fn() }));
        expect(screen.queryByTestId("ctx-speak")).not.toBeInTheDocument();
    });
    it("hides Speak when onSpeak is undefined", () => {
        render(_jsx(TerminalContextMenu, { ...defaultProps(), hasSelection: true }));
        expect(screen.queryByTestId("ctx-speak")).not.toBeInTheDocument();
    });
    it("calls onSpeak when clicked", () => {
        const onSpeak = vi.fn();
        render(_jsx(TerminalContextMenu, { ...defaultProps(), hasSelection: true, onSpeak: onSpeak }));
        fireEvent.click(screen.getByTestId("ctx-speak"));
        expect(onSpeak).toHaveBeenCalledTimes(1);
    });
});
