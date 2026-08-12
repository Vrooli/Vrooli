import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, act, fireEvent, screen } from "@testing-library/react";
import { useState } from "react";
import FullScreenComposer from "../components/FullScreenComposer";
import { composeComposerPayload } from "../lib/composerPayload";
import { useComposerDraft } from "../hooks/useComposerDraft";
function makeSettlement() {
    const subs = new Set();
    const subscribe = (cb) => {
        subs.add(cb);
        return () => subs.delete(cb);
    };
    const fire = (ok) => {
        for (const cb of subs)
            cb(1, ok);
    };
    return { subscribe, fire };
}
function Harness({ onInput = () => ({ status: "sent", seq: 1 }), subscribe, initialOpen = true }) {
    const draft = useComposerDraft("sess-composer");
    const [open, setOpen] = useState(initialOpen);
    return (_jsxs(_Fragment, { children: [_jsx("button", { "data-testid": "ext-open", onClick: () => setOpen(true) }), _jsx(FullScreenComposer, { open: open, onClose: () => setOpen(false), draft: draft, onInput: onInput, subscribeInputSettled: subscribe, onFocusTerminal: vi.fn() })] }));
}
describe("composeComposerPayload", () => {
    it("returns text unchanged when no paths", () => {
        expect(composeComposerPayload("hello", [])).toBe("hello");
    });
    it("space-joins text and paths in order", () => {
        expect(composeComposerPayload("look", ["/a.png", "/b.png"])).toBe("look /a.png /b.png");
    });
    it("returns just paths when text is empty", () => {
        expect(composeComposerPayload("", ["/a.png", "/b.png"])).toBe("/a.png /b.png");
    });
});
describe("FullScreenComposer", () => {
    beforeEach(() => {
        try {
            window.localStorage.clear();
        }
        catch {
            /* no-op */
        }
    });
    it("does not render terminal keys/modifiers", () => {
        render(_jsx(Harness, {}));
        expect(screen.getByTestId("full-screen-composer")).toBeTruthy();
        expect(screen.queryByTestId("toolbar-mod-ctrl")).toBeNull();
        expect(screen.queryByTestId("toolbar-key-esc")).toBeNull();
        expect(screen.queryByTestId(/toolbar-key-/)).toBeNull();
    });
    it("round-trips the draft across minimize/expand (Escape)", () => {
        render(_jsx(Harness, {}));
        const input = screen.getByTestId("composer-input");
        fireEvent.change(input, { target: { value: "a long multi-line prompt" } });
        // Escape minimizes without losing the draft.
        act(() => {
            fireEvent.keyDown(window, { key: "Escape" });
        });
        expect(screen.queryByTestId("full-screen-composer")).toBeNull();
        // Re-open shows the same text.
        fireEvent.click(screen.getByTestId("ext-open"));
        const reopened = screen.getByTestId("composer-input");
        expect(reopened.value).toBe("a long multi-line prompt");
    });
    it("preserves draft when the backdrop is clicked", () => {
        const { container } = render(_jsx(Harness, {}));
        const input = screen.getByTestId("composer-input");
        fireEvent.change(input, { target: { value: "keep me" } });
        const backdrop = container.querySelector(".bg-wc-backdrop");
        expect(backdrop).toBeTruthy();
        act(() => fireEvent.click(backdrop));
        expect(screen.queryByTestId("full-screen-composer")).toBeNull();
        fireEvent.click(screen.getByTestId("ext-open"));
        expect(screen.getByTestId("composer-input").value).toBe("keep me");
    });
    it("sends through onInput and clears+minimizes only on ok settlement", () => {
        const onInput = vi.fn(() => ({ status: "sent", seq: 1 }));
        const settlement = makeSettlement();
        render(_jsx(Harness, { onInput: onInput, subscribe: settlement.subscribe }));
        const input = screen.getByTestId("composer-input");
        fireEvent.change(input, { target: { value: "deploy now" } });
        fireEvent.click(screen.getByTestId("composer-send"));
        expect(onInput).toHaveBeenCalledWith("deploy now", "toolbar-submit");
        // Still open + spinner until settlement.
        expect(screen.getByTestId("composer-sending")).toBeTruthy();
        expect(screen.getByTestId("full-screen-composer")).toBeTruthy();
        act(() => settlement.fire(true));
        // Auto-minimized after ok; draft cleared (reopen shows empty).
        expect(screen.queryByTestId("full-screen-composer")).toBeNull();
        fireEvent.click(screen.getByTestId("ext-open"));
        expect(screen.getByTestId("composer-input").value).toBe("");
    });
    it("keeps draft open and surfaces error on ok=false", () => {
        const onInput = vi.fn(() => ({ status: "sent", seq: 1 }));
        const settlement = makeSettlement();
        render(_jsx(Harness, { onInput: onInput, subscribe: settlement.subscribe }));
        const input = screen.getByTestId("composer-input");
        fireEvent.change(input, { target: { value: "risky payload" } });
        fireEvent.click(screen.getByTestId("composer-send"));
        act(() => settlement.fire(false));
        // Still open, draft preserved, error surfaced.
        expect(screen.getByTestId("full-screen-composer")).toBeTruthy();
        expect(screen.getByTestId("composer-input").value).toBe("risky payload");
        expect(screen.getByTestId("composer-error")).toBeTruthy();
    });
    it("does nothing when send is pressed with an empty draft", () => {
        const onInput = vi.fn(() => ({ status: "sent", seq: 1 }));
        render(_jsx(Harness, { onInput: onInput }));
        fireEvent.click(screen.getByTestId("composer-send"));
        expect(onInput).not.toHaveBeenCalled();
        expect(screen.getByTestId("full-screen-composer")).toBeTruthy();
    });
});
