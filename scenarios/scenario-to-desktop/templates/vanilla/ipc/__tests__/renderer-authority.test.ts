import { describe, expect, it, vi } from "vitest";
import type { WebContents } from "electron";
import { createRendererAuthority, isOwnedRendererURL, protectRendererNavigation, isExternalBrowserURL } from "../renderer-authority";

const electronNavigationBinding: (contents: WebContents, url: string) => void = protectRendererNavigation;
void electronNavigationBinding;

describe("owned desktop renderer authority", () => {
    it("rejects another loopback origin, window and subframe while preserving the owned main frame", () => {
        const frame = { url: "http://127.0.0.1:20000/" };
        const sender = {};
        const authorize = createRendererAuthority(() => ({ sender, mainFrame: frame, url: "http://127.0.0.1:20000/" }));
        const event = { sender, senderFrame: frame };
        expect(authorize(event)).toBe(true);
        frame.url = "http://127.0.0.1:34335/";
        expect(authorize(event)).toBe(false);
        frame.url = "http://127.0.0.1:20000/";
        expect(authorize({ sender, senderFrame: { url: frame.url } })).toBe(false);
        expect(authorize({ sender: {}, senderFrame: frame })).toBe(false);
        expect(authorize({ sender })).toBe(false);
        expect(authorize(event)).toBe(true);
        expect(createRendererAuthority(() => null)(event)).toBe(false);
    });

    it("prevents foreign top-level navigation and redirects before committing a new document", () => {
        const listeners = new Map<string, (event: { preventDefault(): void; url?: string; isMainFrame?: boolean }, url?: string) => void>();
        protectRendererNavigation({ on: (name, listener) => listeners.set(name, listener) }, "http://127.0.0.1:20000/");
        for (const name of ["will-navigate", "will-redirect", "will-frame-navigate"]) {
            const preventDefault = vi.fn();
            listeners.get(name)!({ preventDefault, url: "http://127.0.0.1:34335/", isMainFrame: true }, name === "will-frame-navigate" ? undefined : "http://127.0.0.1:34335/");
            expect(preventDefault).toHaveBeenCalledOnce();
            preventDefault.mockClear();
            listeners.get(name)!({ preventDefault, url: "http://127.0.0.1:20000/settings", isMainFrame: true }, "http://127.0.0.1:20000/settings");
            expect(preventDefault).not.toHaveBeenCalled();
        }
    });

    it("pins file and custom-scheme owners and opens only browser URLs externally", () => {
        expect(isOwnedRendererURL("file:///app/index.html", "file:///app/index.html#settings")).toBe(true);
        expect(isOwnedRendererURL("file:///app/index.html", "file:///tmp/unrelated.html")).toBe(false);
        expect(isOwnedRendererURL("vrooli-desktop://app/", "vrooli-desktop://unrelated/")).toBe(false);
        expect(isOwnedRendererURL("http://127.0.0.1:20000/", "http://user:secret@127.0.0.1:20000/")).toBe(false);
        expect(isExternalBrowserURL("https://example.com/help")).toBe(true);
        expect(isExternalBrowserURL("file:///tmp/unrelated.html")).toBe(false);
        expect(isExternalBrowserURL("javascript:alert(1)")).toBe(false);
    });
});
