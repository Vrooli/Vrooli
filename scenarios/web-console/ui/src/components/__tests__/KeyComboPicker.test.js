import { jsx as _jsx } from "react/jsx-runtime";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import KeyComboPicker from "../KeyComboPicker";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
// Reset store before each test
beforeEach(() => {
    useWorkspaceStore.setState({
        recentCombos: [],
    });
});
describe("KeyComboPicker", () => {
    const defaultProps = {
        onInput: vi.fn(() => ({ status: "sent", seq: 1 })),
        onFocusTerminal: vi.fn(),
    };
    it("renders trigger button", () => {
        render(_jsx(KeyComboPicker, { ...defaultProps }));
        expect(screen.getByTestId("combo-picker-trigger")).toBeInTheDocument();
    });
    it("trigger button has tabIndex={-1}", () => {
        render(_jsx(KeyComboPicker, { ...defaultProps }));
        expect(screen.getByTestId("combo-picker-trigger")).toHaveAttribute("tabindex", "-1");
    });
    it("clicking trigger opens bottom sheet", () => {
        render(_jsx(KeyComboPicker, { ...defaultProps }));
        fireEvent.click(screen.getByTestId("combo-picker-trigger"));
        expect(screen.getByTestId("combo-picker-panel")).toBeInTheDocument();
        expect(screen.getByTestId("combo-picker-backdrop")).toBeInTheDocument();
    });
    it("clicking backdrop closes bottom sheet", () => {
        render(_jsx(KeyComboPicker, { ...defaultProps }));
        fireEvent.click(screen.getByTestId("combo-picker-trigger"));
        expect(screen.getByTestId("combo-picker-panel")).toBeInTheDocument();
        fireEvent.click(screen.getByTestId("combo-picker-backdrop"));
        expect(screen.queryByTestId("combo-picker-panel")).not.toBeInTheDocument();
    });
    it("combo items are rendered with correct labels", () => {
        render(_jsx(KeyComboPicker, { ...defaultProps }));
        fireEvent.click(screen.getByTestId("combo-picker-trigger"));
        // Check a few known combos
        expect(screen.getByTestId("combo-item-ctrl-c")).toBeInTheDocument();
        expect(screen.getByTestId("combo-item-ctrl-d")).toBeInTheDocument();
        expect(screen.getByTestId("combo-item-ctrl-c-x2")).toBeInTheDocument();
    });
    it("tapping a combo calls onInput and closes sheet", async () => {
        const onInput = vi.fn(() => ({ status: "sent", seq: 1 }));
        render(_jsx(KeyComboPicker, { onInput: onInput, onFocusTerminal: vi.fn() }));
        fireEvent.click(screen.getByTestId("combo-picker-trigger"));
        fireEvent.click(screen.getByTestId("combo-item-ctrl-c"));
        // Sheet should close
        expect(screen.queryByTestId("combo-picker-panel")).not.toBeInTheDocument();
        // onInput should have been called with Ctrl+C data and the
        // toolbar-key source tag.
        await waitFor(() => {
            expect(onInput).toHaveBeenCalledWith("\x03", "toolbar-key");
        });
    });
    it("search filters visible combos", () => {
        render(_jsx(KeyComboPicker, { ...defaultProps }));
        fireEvent.click(screen.getByTestId("combo-picker-trigger"));
        const searchInput = screen.getByTestId("combo-picker-search");
        fireEvent.change(searchInput, { target: { value: "suspend" } });
        // Only Ctrl+Z (Suspend) should remain
        expect(screen.getByTestId("combo-item-ctrl-z")).toBeInTheDocument();
        expect(screen.queryByTestId("combo-item-ctrl-c")).not.toBeInTheDocument();
    });
    it("recent combos section appears when store has entries", () => {
        useWorkspaceStore.setState({ recentCombos: ["ctrl-c", "ctrl-d"] });
        render(_jsx(KeyComboPicker, { ...defaultProps }));
        fireEvent.click(screen.getByTestId("combo-picker-trigger"));
        expect(screen.getByTestId("combo-recent-ctrl-c")).toBeInTheDocument();
        expect(screen.getByTestId("combo-recent-ctrl-d")).toBeInTheDocument();
    });
    it("calls onFocusTerminal after selecting a combo", async () => {
        const onFocusTerminal = vi.fn();
        render(_jsx(KeyComboPicker, { onInput: vi.fn(() => ({ status: "sent", seq: 1 })), onFocusTerminal: onFocusTerminal }));
        fireEvent.click(screen.getByTestId("combo-picker-trigger"));
        fireEvent.click(screen.getByTestId("combo-item-ctrl-c"));
        expect(onFocusTerminal).toHaveBeenCalled();
    });
});
