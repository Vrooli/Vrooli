// @vitest-environment node
// [REQ:REQ-API-003] fetchHealth behavior through the globalThis.fetch seam
// [REQ:REQ-API-002A] fetchEvents URL-construction behavior through the seam
// [REQ:REQ-PS-001] subscribeSSE behavior through the EventSource seam
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import {
    createPolicy,
    createSubscription,
    deletePolicy,
    deleteSubscription,
    fetchHealth,
    fetchEvents,
    fetchPolicies,
    fetchPolicy,
    fetchSubscription,
    fetchSubscriptionHealth,
    fetchSubscriptions,
    fetchViolations,
    overrideCircuitBreaker,
    subscribeSSE,
    updatePolicy,
} from "./api";
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
        const call = httpMock.calls[0];
        if (!call) throw new Error("expected one health request");
        expect(call.url).toContain("/health");
        expect(call.method).toBe("GET");
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

        await fetchEvents({
            type: "discovery.*",
            source: "agent-manager",
            target: "plan-manager",
            correlationId: "corr-1",
            since: 10,
            limit: 25,
        });

        const call = httpMock.calls[0];
        if (!call) throw new Error("expected one events request");
        const calledUrl = call.url;
        // URLSearchParams preserves `*` unencoded (RFC 3986 considers it a sub-delim, not reserved in query).
        expect(calledUrl).toContain("type=discovery.*");
        expect(calledUrl).toContain("source=agent-manager");
        expect(calledUrl).toContain("target=plan-manager");
        expect(calledUrl).toContain("correlation_id=corr-1");
        expect(calledUrl).toContain("since=10");
        expect(calledUrl).toContain("limit=25");
    });

    it("omits query string entirely when no params are passed", async () => {
        httpMock.respondTo({ urlPattern: "/events" }, { body: [] });

        await fetchEvents();

        const call = httpMock.calls[0];
        if (!call) throw new Error("expected one events request");
        const calledUrl = call.url;
        expect(calledUrl).not.toContain("?");
    });
});

describe("policy and subscription API operations", () => {
    let httpMock: MockFetchHandle;

    beforeEach(() => {
        httpMock = mockFetch();
        httpMock.respondTo({ urlPattern: "/policies" }, { body: [] });
        httpMock.respondTo({ urlPattern: "/subscriptions" }, { body: [] });
    });

    afterEach(() => {
        httpMock.restore();
    });

    it("covers the CRUD, violation, override, and health request surfaces", async () => {
        await fetchPolicy(1);
        await fetchPolicies();
        await createPolicy({
            rule_type: "access",
            source_scenario: "alpha",
            target_scenario: "beta",
            priority: 1,
            enabled: true,
        });
        await updatePolicy(1, { enabled: false });
        await deletePolicy(1);
        await fetchViolations();
        await overrideCircuitBreaker(1, "open", 60);

        await fetchSubscription(2);
        await fetchSubscriptions();
        await fetchSubscriptionHealth(2);
        await createSubscription({
            name: "alpha",
            owner_scenario: "owner",
            event_pattern: "alpha.*",
            delivery_type: "webhook",
            delivery_target: "http://example.test",
            enabled: true,
        });
        await deleteSubscription(2);

        expect(httpMock.calls.map((call) => call.method)).toEqual([
            "GET", "GET", "POST", "PUT", "DELETE", "GET", "POST",
            "GET", "GET", "GET", "POST", "DELETE",
        ]);
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
            target: "my-target",
            onEvent: () => {},
        });

        expect(sseMock.instances).toHaveLength(1);
        const instance = sseMock.instances[0];
        if (!instance) throw new Error("expected one EventSource instance");
        expect(instance.url).toContain("/events/subscribe");
        expect(instance.url).toContain("type=app.*");
        expect(instance.url).toContain("source=my-src");
        expect(instance.url).toContain("target=my-target");

        unsubscribe();
        expect(instance.closed).toBe(true);
    });

    it("parses incoming message data and invokes onEvent", () => {
        const onEvent = vi.fn();
        subscribeSSE({ onEvent });

        const instance = sseMock.instances[0];
        if (!instance) throw new Error("expected one EventSource instance");
        instance.emitMessage({ eventId: "evt-1", sourceScenario: "src", eventType: "x" });

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

        const instance = sseMock.instances[0];
        if (!instance) throw new Error("expected one EventSource instance");
        instance.emitError();

        expect(onError).toHaveBeenCalledTimes(1);
        warnSpy.mockRestore();
    });

    it("swallows malformed JSON without throwing (logs warning, keeps stream alive)", () => {
        const onEvent = vi.fn();
        const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

        subscribeSSE({ onEvent });

        // Send a raw string that isn't valid JSON
        const instance = sseMock.instances[0];
        if (!instance) throw new Error("expected one EventSource instance");
        instance.emitMessage("not-json-at-all");

        expect(onEvent).not.toHaveBeenCalled();
        expect(warnSpy).toHaveBeenCalled();

        warnSpy.mockRestore();
    });

    it("handles non-string SSE data", () => {
        const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
        subscribeSSE({ onEvent: () => {} });
        const instance = sseMock.instances[0];
        if (!instance) throw new Error("expected one EventSource instance");
        for (const listener of instance.listeners.get("message") ?? []) {
            listener({ data: {} } as MessageEvent);
        }
        expect(warnSpy).toHaveBeenCalled();
        warnSpy.mockRestore();
    });
});
