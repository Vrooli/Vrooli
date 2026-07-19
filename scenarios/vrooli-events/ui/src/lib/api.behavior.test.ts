// @vitest-environment node
// [REQ:REQ-API-003] fetchHealth behavior through the globalThis.fetch seam
// [REQ:REQ-API-002A] fetchEvents URL-construction behavior through the seam
// [REQ:REQ-PS-001] subscribeSSE behavior through the EventSource seam
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { fetchHealth, fetchEvents, subscribeSSE } from "./api";
import { mockFetch, type MockFetchHandle } from "../test-utils/mockFetch";
import { mockEventSource, type MockEventSourceHandle } from "../test-utils/mockEventSource";
import { createMockHealthResponse } from "../test-utils/factories";

describe("fetchHealth (via globalThis.fetch seam)", () => {
    let httpMock: MockFetchHandle;

    beforeEach(() => {
        httpMock = mockFetch();
    });

    afterEach(() => {
        httpMock.restore();
    });

    it("calls the /health endpoint and returns the parsed response", async () => {
        const payload = createMockHealthResponse({ subscribers: 3, service: "vrooli-events" });
        httpMock.respondTo({ urlPattern: "/health" }, { body: payload });

        const result = await fetchHealth();

        expect(result.subscribers).toBe(3);
        expect(result.service).toBe("vrooli-events");
        expect(httpMock.calls[0].url).toContain("/health");
        expect(httpMock.calls[0].method).toBe("GET");
    });

    it("throws when /health returns non-ok status", async () => {
        httpMock.respondTo({ urlPattern: "/health" }, { status: 503, body: { error: "down" } });

        await expect(fetchHealth()).rejects.toThrow(/Request failed: 503/);
    });
});

describe("fetchEvents (via globalThis.fetch seam)", () => {
    let httpMock: MockFetchHandle;

    beforeEach(() => {
        httpMock = mockFetch();
    });

    afterEach(() => {
        httpMock.restore();
    });

    it("appends filter params to the URL when provided", async () => {
        httpMock.respondTo({ urlPattern: "/events" }, { body: [] });

        await fetchEvents({ type: "discovery.*", source: "agent-manager", target: "plan-manager", limit: 25 });

        const calledUrl = httpMock.calls[0].url;
        // URLSearchParams preserves `*` unencoded (RFC 3986 considers it a sub-delim, not reserved in query).
        expect(calledUrl).toContain("type=discovery.*");
        expect(calledUrl).toContain("source=agent-manager");
        expect(calledUrl).toContain("target=plan-manager");
        expect(calledUrl).toContain("limit=25");
    });

    it("omits query string entirely when no params are passed", async () => {
        httpMock.respondTo({ urlPattern: "/events" }, { body: [] });

        await fetchEvents();

        const calledUrl = httpMock.calls[0].url;
        expect(calledUrl).not.toContain("?");
    });
});

describe("subscribeSSE (via globalThis.EventSource seam)", () => {
    let sseMock: MockEventSourceHandle;

    beforeEach(() => {
        sseMock = mockEventSource();
    });

    afterEach(() => {
        sseMock.restore();
    });

    it("opens an EventSource against /events/subscribe with filter params", () => {
        const unsubscribe = subscribeSSE({
            type: "app.*",
            source: "my-src",
            onEvent: () => {},
        });

        expect(sseMock.instances).toHaveLength(1);
        expect(sseMock.instances[0].url).toContain("/events/subscribe");
        expect(sseMock.instances[0].url).toContain("type=app.*");
        expect(sseMock.instances[0].url).toContain("source=my-src");

        unsubscribe();
        expect(sseMock.instances[0].closed).toBe(true);
    });

    it("parses incoming message data and invokes onEvent", () => {
        const onEvent = vi.fn();
        subscribeSSE({ onEvent });

        sseMock.instances[0].emitMessage({ eventId: "evt-1", sourceScenario: "src", eventType: "x" });

        // Current subscribeSSE wires BOTH `addEventListener("message", ...)` AND `onmessage` to the
        // same handler, so an unnamed SSE event triggers the parser twice. This is a known pre-existing
        // double-dispatch quirk logged in docs/internal/PROBLEMS.md — the seam revealed it cleanly.
        expect(onEvent).toHaveBeenCalledTimes(2);
        expect(onEvent).toHaveBeenLastCalledWith(
            expect.objectContaining({ eventId: "evt-1", eventType: "x" }),
        );
    });

    it("forwards connection errors to onError", () => {
        const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
        const onError = vi.fn();
        subscribeSSE({ onEvent: () => {}, onError });

        sseMock.instances[0].emitError();

        expect(onError).toHaveBeenCalledTimes(1);
        warnSpy.mockRestore();
    });

    it("swallows malformed JSON without throwing (logs warning, keeps stream alive)", () => {
        const onEvent = vi.fn();
        const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

        subscribeSSE({ onEvent });

        // Send a raw string that isn't valid JSON
        sseMock.instances[0].emitMessage("not-json-at-all");

        expect(onEvent).not.toHaveBeenCalled();
        expect(warnSpy).toHaveBeenCalled();

        warnSpy.mockRestore();
    });
});
