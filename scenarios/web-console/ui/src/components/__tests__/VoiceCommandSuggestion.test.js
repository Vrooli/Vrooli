import { jsx as _jsx } from "react/jsx-runtime";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, act, fireEvent } from "@testing-library/react";
import VoiceCommandSuggestion from "../VoiceCommandSuggestion";
function makeSuggestion(overrides = {}) {
    return {
        id: "test-cmd-1",
        commandId: "new-tab",
        description: "New Tab",
        confidence: 0.95,
        rawText: "hey do new tab",
        timestamp: Date.now(),
        args: {},
        ...overrides,
    };
}
describe("VoiceCommandSuggestion", () => {
    beforeEach(() => {
        vi.useFakeTimers();
    });
    afterEach(() => {
        vi.useRealTimers();
    });
    it("renders the command description", () => {
        const suggestion = makeSuggestion();
        render(_jsx(VoiceCommandSuggestion, { suggestion: suggestion, onConfirm: vi.fn(), onDismiss: vi.fn() }));
        expect(screen.getByText("New Tab")).toBeInTheDocument();
    });
    it("calls onConfirm when confirm button is clicked", () => {
        const onConfirm = vi.fn();
        const suggestion = makeSuggestion();
        render(_jsx(VoiceCommandSuggestion, { suggestion: suggestion, onConfirm: onConfirm, onDismiss: vi.fn() }));
        fireEvent.click(screen.getByTestId("voice-command-confirm"));
        expect(onConfirm).toHaveBeenCalledWith(suggestion);
    });
    it("calls onDismiss when dismiss button is clicked", () => {
        const onDismiss = vi.fn();
        const suggestion = makeSuggestion();
        render(_jsx(VoiceCommandSuggestion, { suggestion: suggestion, onConfirm: vi.fn(), onDismiss: onDismiss }));
        fireEvent.click(screen.getByTestId("voice-command-dismiss"));
        expect(onDismiss).toHaveBeenCalledWith(suggestion);
    });
    it("auto-dismisses after 5 seconds", () => {
        const onDismiss = vi.fn();
        const suggestion = makeSuggestion();
        render(_jsx(VoiceCommandSuggestion, { suggestion: suggestion, onConfirm: vi.fn(), onDismiss: onDismiss }));
        expect(onDismiss).not.toHaveBeenCalled();
        act(() => { vi.advanceTimersByTime(5000); });
        expect(onDismiss).toHaveBeenCalledWith(suggestion);
    });
    it("clears auto-dismiss timer on confirm", () => {
        const onDismiss = vi.fn();
        const onConfirm = vi.fn();
        const suggestion = makeSuggestion();
        render(_jsx(VoiceCommandSuggestion, { suggestion: suggestion, onConfirm: onConfirm, onDismiss: onDismiss }));
        fireEvent.click(screen.getByTestId("voice-command-confirm"));
        act(() => { vi.advanceTimersByTime(5000); });
        // onDismiss should not be called since confirm cleared the timer
        expect(onDismiss).not.toHaveBeenCalled();
    });
});
