import { describe, expect, it, vi } from "vitest";
import {
    createSecureDesktopSchemeHandler,
    registerSecureDesktopScheme,
    SECURE_DESKTOP_SCHEME,
} from "../secure-custom-scheme";

describe("secure desktop custom scheme", () => {
    it("registers a secure standard fetch-capable scheme only when selected", () => {
        const registerSchemesAsPrivileged = vi.fn();
        registerSecureDesktopScheme({ registerSchemesAsPrivileged }, false);
        expect(registerSchemesAsPrivileged).not.toHaveBeenCalled();

        registerSecureDesktopScheme({ registerSchemesAsPrivileged }, true);
        expect(registerSchemesAsPrivileged).toHaveBeenCalledWith([{
            scheme: SECURE_DESKTOP_SCHEME,
            privileges: { standard: true, secure: true, supportFetchAPI: true },
        }]);
    });

    it("forwards renderer paths, methods, and authorization to the bundled origin", async () => {
        const fetcher = vi.fn(async (_request: Request) => new Response("ok"));
        const handler = createSecureDesktopSchemeHandler("http://127.0.0.1:4312/", fetcher);
        const request = new Request("vrooli-desktop://desktop/api/v1/items?q=one", {
            method: "POST",
            headers: { Authorization: "Bearer local-session", "Content-Type": "application/json" },
            body: JSON.stringify({ item: 1 }),
        });

        const response = await handler(request);
        const forwarded = fetcher.mock.calls[0]?.[0];
        expect(forwarded).toBeDefined();
        if (!forwarded) throw new Error("forwarded request missing");
        expect(response.status).toBe(200);
        expect(forwarded.url).toBe("http://127.0.0.1:4312/api/v1/items?q=one");
        expect(forwarded.method).toBe("POST");
        expect(forwarded.headers.get("Authorization")).toBe("Bearer local-session");
        expect(forwarded.headers.get("Origin")).toBe("http://127.0.0.1:4312");
        expect(await forwarded.text()).toBe(JSON.stringify({ item: 1 }));
    });

    it("rejects arbitrary renderer hosts and non-loopback runtimes", async () => {
        const handler = createSecureDesktopSchemeHandler("http://localhost:4312/", async () => new Response("unexpected"));
        const response = await handler(new Request("vrooli-desktop://attacker/api"));
        expect(response.status).toBe(404);

        expect(() => createSecureDesktopSchemeHandler("https://example.com/", async () => new Response())).toThrow("loopback runtime");
        expect(() => createSecureDesktopSchemeHandler("http://example.com/", async () => new Response())).toThrow("loopback runtime");
    });
});
