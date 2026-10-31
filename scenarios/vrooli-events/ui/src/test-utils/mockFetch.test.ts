// [REQ:REQ-API-003] Health endpoint seam — mockFetch enables behavioral tests of api.ts
// [REQ:REQ-API-002A] Query parameter seam verification via mockFetch
import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { mockFetch } from "./mockFetch";
import type { MockFetchHandle } from "./mockFetch";

describe("mockFetch helper", () => {
    let handle: MockFetchHandle;

    beforeEach(() => {
        handle = mockFetch();
    });

    afterEach(() => {
        handle.restore();
    });

    it("returns a programmed JSON body", async () => {
        handle.respondTo({ urlPattern: "/health" }, { body: { status: "healthy" } });

        const res = await fetch("http://localhost/api/v1/health");
        const json = await res.json();

        expect(res.status).toBe(200);
        expect(json).toEqual({ status: "healthy" });
    });

    it("matches by URL substring", async () => {
        handle.respondTo({ urlPattern: "/events" }, { body: [{ eventId: "e1" }] });
        handle.respondTo({ urlPattern: "/policies" }, { body: [{ id: 1 }] });

        const events = await (await fetch("http://a/api/v1/events?type=foo")).json();
        const policies = await (await fetch("http://a/api/v1/policies")).json();

        expect(events).toEqual([{ eventId: "e1" }]);
        expect(policies).toEqual([{ id: 1 }]);
    });

    it("matches by RegExp pattern", async () => {
        handle.respondTo({ urlPattern: /\/policies\/\d+$/ }, { body: { id: 7 } });

        const res = await fetch("http://a/api/v1/policies/7");
        const json = await res.json();

        expect(json).toEqual({ id: 7 });
    });

    it("respects method filter on matchers", async () => {
        handle.respondTo({ urlPattern: "/subscriptions", method: "POST" }, { status: 201, body: { id: 42 } });

        // Unmatched GET falls through to the 500 fallback
        const getRes = await fetch("http://a/api/v1/subscriptions");
        expect(getRes.status).toBe(500);

        const postRes = await fetch("http://a/api/v1/subscriptions", {
            method: "POST",
            body: JSON.stringify({ name: "x" }),
        });
        expect(postRes.status).toBe(201);
        expect(await postRes.json()).toEqual({ id: 42 });
    });

    it("records call metadata (url, method, body, headers)", async () => {
        handle.respondTo({ urlPattern: "/policies" }, { status: 201, body: { id: 1 } });

        await fetch("http://a/api/v1/policies", {
            method: "POST",
            headers: { "Content-Type": "application/json", "X-Test": "1" },
            body: JSON.stringify({ priority: 5 }),
        });

        expect(handle.calls).toHaveLength(1);
        expect(handle.calls[0]!.url).toContain("/policies");
        expect(handle.calls[0]!.method).toBe("POST");
        expect(handle.calls[0]!.body).toBe(JSON.stringify({ priority: 5 }));
        expect(handle.calls[0]!.headers?.["content-type"]).toBe("application/json");
        expect(handle.calls[0]!.headers?.["x-test"]).toBe("1");
    });

    it("throws the programmed error when rejectWith matches", async () => {
        handle.rejectWith({ urlPattern: "/events" }, new Error("simulated network failure"));

        await expect(fetch("http://a/api/v1/events")).rejects.toThrow("simulated network failure");
    });

    it("returns 500 fallback for unmatched calls (fail-loud)", async () => {
        const res = await fetch("http://a/unmatched");

        expect(res.status).toBe(500);
        expect(await res.text()).toContain("no response programmed");
    });

    it("first matching program wins (insertion order)", async () => {
        handle.respondTo({ urlPattern: "/events" }, { body: { source: "first" } });
        handle.respondTo({ urlPattern: "/events" }, { body: { source: "second" } });

        const json = await (await fetch("http://a/api/v1/events")).json();
        expect(json).toEqual({ source: "first" });
    });

    it("restore() returns globalThis.fetch to the original implementation", () => {
        const mockedFetch = globalThis.fetch;
        handle.restore();
        expect(globalThis.fetch).not.toBe(mockedFetch);
    });
});
