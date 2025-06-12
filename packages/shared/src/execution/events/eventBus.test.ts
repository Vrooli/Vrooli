import { describe, expect, it, vi, beforeEach } from "vitest";
import { EventBuilder, EventFilterUtils, EventCorrelator } from "./eventBus.js";
import type { BaseEvent, EventFilter } from "../types/events.js";

describe("EventBuilder", () => {
    const createValidEvent = () => new EventBuilder()
        .withId("test-id")
        .withType("TEST_EVENT")
        .withSource(1, "test-component", "test-instance")
        .withCorrelationId("test-correlation");

    it("should throw error when building without required fields", () => {
        const builder = new EventBuilder();
        expect(() => builder.build()).toThrow("Event ID is required");

        builder.withId("test-id");
        expect(() => builder.build()).toThrow("Event type is required");

        builder.withType("TEST_EVENT");
        expect(() => builder.build()).toThrow("Event source is required");

        builder.withSource(1, "component", "instance");
        expect(() => builder.build()).toThrow("Correlation ID is required");
    });

    it("should build valid event with all required fields", () => {
        const event = createValidEvent().build();

        expect(event.id).toBe("test-id");
        expect(event.type).toBe("TEST_EVENT");
        expect(event.source).toEqual({
            tier: 1,
            component: "test-component",
            instanceId: "test-instance",
        });
        expect(event.correlationId).toBe("test-correlation");
        expect(event.timestamp).toBeInstanceOf(Date);
        expect(event.metadata).toEqual({
            version: "1.0.0",
            tags: [],
            priority: "NORMAL",
        });
    });

    it("should set source with cross-cutting tier", () => {
        const event = new EventBuilder()
            .withId("test-id")
            .withType("TEST_EVENT")
            .withSource("cross-cutting", "logger", "logger-456")
            .withCorrelationId("test-correlation")
            .build();

        expect(event.source).toEqual({
            tier: "cross-cutting",
            component: "logger",
            instanceId: "logger-456",
        });
    });

    it("should set causation id", () => {
        const event = createValidEvent()
            .withCausationId("causation-456")
            .build();

        expect(event.causationId).toBe("causation-456");
    });

    it("should set user id in metadata", () => {
        const event = createValidEvent()
            .withUserId("user-789")
            .build();

        expect(event.metadata.userId).toBe("user-789");
    });

    it("should set session id in metadata", () => {
        const event = createValidEvent()
            .withSessionId("session-abc")
            .build();

        expect(event.metadata.sessionId).toBe("session-abc");
    });

    it("should set request id in metadata", () => {
        const event = createValidEvent()
            .withRequestId("request-def")
            .build();

        expect(event.metadata.requestId).toBe("request-def");
    });

    it("should set priority in metadata", () => {
        const lowEvent = createValidEvent().withPriority("LOW").build();
        expect(lowEvent.metadata.priority).toBe("LOW");

        const highEvent = createValidEvent().withPriority("HIGH").build();
        expect(highEvent.metadata.priority).toBe("HIGH");

        const criticalEvent = createValidEvent().withPriority("CRITICAL").build();
        expect(criticalEvent.metadata.priority).toBe("CRITICAL");
    });

    it("should add tags to metadata", () => {
        const event = createValidEvent()
            .withTags("tag1", "tag2")
            .build();

        expect(event.metadata.tags).toEqual(["tag1", "tag2"]);
    });

    it("should accumulate tags when called multiple times", () => {
        const event = createValidEvent()
            .withTags("tag1", "tag2")
            .withTags("tag3")
            .withTags("tag4", "tag5")
            .build();

        expect(event.metadata.tags).toEqual(["tag1", "tag2", "tag3", "tag4", "tag5"]);
    });

    it("should set TTL in metadata", () => {
        const event = createValidEvent()
            .withTTL(3600)
            .build();

        expect(event.metadata.ttl).toBe(3600);
    });

    it("should set payload correctly", () => {
        const payload = { action: "clicked", buttonId: "submit-btn" };
        const event = createValidEvent()
            .withPayload(payload)
            .build();

        expect(event.payload).toEqual(payload);
    });

    it("should create a complete event with chained methods", () => {
        const payload = { message: "Hello, World!" };

        const event = new EventBuilder()
            .withId("event-123")
            .withType("MESSAGE_SENT")
            .withSource(2, "orchestrator", "orch-001")
            .withCorrelationId("corr-456")
            .withCausationId("cause-789")
            .withUserId("user-abc")
            .withSessionId("session-def")
            .withRequestId("req-ghi")
            .withPriority("HIGH")
            .withTags("messaging", "notification")
            .withTTL(1800)
            .withPayload(payload)
            .build();

        expect(event.id).toBe("event-123");
        expect(event.type).toBe("MESSAGE_SENT");
        expect(event.source).toEqual({
            tier: 2,
            component: "orchestrator",
            instanceId: "orch-001",
        });
        expect(event.correlationId).toBe("corr-456");
        expect(event.causationId).toBe("cause-789");
        expect(event.payload).toEqual(payload);
        expect(event.metadata).toEqual({
            version: "1.0.0",
            tags: ["messaging", "notification"],
            priority: "HIGH",
            userId: "user-abc",
            sessionId: "session-def",
            requestId: "req-ghi",
            ttl: 1800,
        });
        expect(event.timestamp).toBeInstanceOf(Date);
    });

    it("should handle edge cases", () => {
        const event = createValidEvent()
            .withTags()
            .withTTL(0)
            .build();

        expect(event.metadata.tags).toEqual([]);
        expect(event.metadata.ttl).toBe(0);
    });
});

describe("EventFilterUtils", () => {
    const testEvent: BaseEvent = {
        id: "test-event-1",
        type: "USER_ACTION",
        source: { tier: 1, component: "ui", instanceId: "ui-1" },
        correlationId: "corr-123",
        timestamp: new Date(),
        metadata: {
            version: "1.0.0",
            tags: ["click", "button"],
            priority: "NORMAL",
            userId: "user-456",
        },
        payload: { buttonId: "submit", value: "test-value" },
    };

    it("should match equals operator", () => {
        const filter: EventFilter = {
            field: "type",
            operator: "equals",
            value: "USER_ACTION",
        };

        expect(EventFilterUtils.matches(testEvent, filter)).toBe(true);

        const falseFilter: EventFilter = {
            field: "type",
            operator: "equals",
            value: "SYSTEM_EVENT",
        };

        expect(EventFilterUtils.matches(testEvent, falseFilter)).toBe(false);
    });

    it("should match contains operator", () => {
        const filter: EventFilter = {
            field: "type",
            operator: "contains",
            value: "ACTION",
        };

        expect(EventFilterUtils.matches(testEvent, filter)).toBe(true);

        const falseFilter: EventFilter = {
            field: "type",
            operator: "contains",
            value: "SYSTEM",
        };

        expect(EventFilterUtils.matches(testEvent, falseFilter)).toBe(false);
    });

    it("should match startsWith operator", () => {
        const filter: EventFilter = {
            field: "type",
            operator: "startsWith",
            value: "USER",
        };

        expect(EventFilterUtils.matches(testEvent, filter)).toBe(true);

        const falseFilter: EventFilter = {
            field: "type",
            operator: "startsWith",
            value: "SYSTEM",
        };

        expect(EventFilterUtils.matches(testEvent, falseFilter)).toBe(false);
    });

    it("should match endsWith operator", () => {
        const filter: EventFilter = {
            field: "type",
            operator: "endsWith",
            value: "ACTION",
        };

        expect(EventFilterUtils.matches(testEvent, filter)).toBe(true);

        const falseFilter: EventFilter = {
            field: "type",
            operator: "endsWith",
            value: "EVENT",
        };

        expect(EventFilterUtils.matches(testEvent, falseFilter)).toBe(false);
    });

    it("should match in operator", () => {
        const filter: EventFilter = {
            field: "type",
            operator: "in",
            value: ["USER_ACTION", "SYSTEM_EVENT"],
        };

        expect(EventFilterUtils.matches(testEvent, filter)).toBe(true);

        const falseFilter: EventFilter = {
            field: "type",
            operator: "in",
            value: ["SYSTEM_EVENT", "ERROR_EVENT"],
        };

        expect(EventFilterUtils.matches(testEvent, falseFilter)).toBe(false);
    });

    it("should match regex operator", () => {
        const filter: EventFilter = {
            field: "type",
            operator: "regex",
            value: "^USER_.*",
        };

        expect(EventFilterUtils.matches(testEvent, filter)).toBe(true);

        const falseFilter: EventFilter = {
            field: "type",
            operator: "regex",
            value: "^SYSTEM_.*",
        };

        expect(EventFilterUtils.matches(testEvent, falseFilter)).toBe(false);
    });

    it("should handle nested field access", () => {
        const filter: EventFilter = {
            field: "metadata.userId",
            operator: "equals",
            value: "user-456",
        };

        expect(EventFilterUtils.matches(testEvent, filter)).toBe(true);

        const payloadFilter: EventFilter = {
            field: "payload.buttonId",
            operator: "equals",
            value: "submit",
        };

        expect(EventFilterUtils.matches(testEvent, payloadFilter)).toBe(true);
    });

    it("should return false for unknown operators", () => {
        const filter: EventFilter = {
            field: "type",
            operator: "unknown" as any,
            value: "USER_ACTION",
        };

        expect(EventFilterUtils.matches(testEvent, filter)).toBe(false);
    });

    it("should match all filters", () => {
        const filters: EventFilter[] = [
            { field: "type", operator: "equals", value: "USER_ACTION" },
            { field: "metadata.userId", operator: "equals", value: "user-456" },
            { field: "payload.buttonId", operator: "equals", value: "submit" },
        ];

        expect(EventFilterUtils.matchesAll(testEvent, filters)).toBe(true);

        const filtersWithFalse: EventFilter[] = [
            { field: "type", operator: "equals", value: "USER_ACTION" },
            { field: "metadata.userId", operator: "equals", value: "user-999" },
        ];

        expect(EventFilterUtils.matchesAll(testEvent, filtersWithFalse)).toBe(false);
    });

    it("should handle null/undefined values gracefully", () => {
        const filter: EventFilter = {
            field: "nonexistent.field",
            operator: "equals",
            value: "test",
        };

        expect(EventFilterUtils.matches(testEvent, filter)).toBe(false);
    });
});

describe("EventCorrelator", () => {
    let correlator: EventCorrelator;

    beforeEach(() => {
        correlator = new EventCorrelator();
    });

    it("should add events and track correlations", () => {
        const event1: BaseEvent = {
            id: "event-1",
            type: "USER_ACTION",
            source: { tier: 1, component: "ui", instanceId: "ui-1" },
            correlationId: "corr-123",
            timestamp: new Date(),
            metadata: { version: "1.0.0", tags: [], priority: "NORMAL" },
        };

        const event2: BaseEvent = {
            id: "event-2",
            type: "SYSTEM_RESPONSE",
            source: { tier: 2, component: "api", instanceId: "api-1" },
            correlationId: "corr-123",
            timestamp: new Date(),
            metadata: { version: "1.0.0", tags: [], priority: "NORMAL" },
        };

        correlator.addEvent(event1);
        correlator.addEvent(event2);

        const correlated = correlator.getCorrelatedEvents("corr-123");
        expect(correlated).toEqual(expect.arrayContaining(["event-1", "event-2"]));
        expect(correlated).toHaveLength(2);
    });

    it("should handle events with different correlation IDs", () => {
        const event1: BaseEvent = {
            id: "event-1",
            type: "USER_ACTION",
            source: { tier: 1, component: "ui", instanceId: "ui-1" },
            correlationId: "corr-123",
            timestamp: new Date(),
            metadata: { version: "1.0.0", tags: [], priority: "NORMAL" },
        };

        const event2: BaseEvent = {
            id: "event-2",
            type: "USER_ACTION",
            source: { tier: 1, component: "ui", instanceId: "ui-1" },
            correlationId: "corr-456",
            timestamp: new Date(),
            metadata: { version: "1.0.0", tags: [], priority: "NORMAL" },
        };

        correlator.addEvent(event1);
        correlator.addEvent(event2);

        expect(correlator.getCorrelatedEvents("corr-123")).toEqual(["event-1"]);
        expect(correlator.getCorrelatedEvents("corr-456")).toEqual(["event-2"]);
        expect(correlator.getCorrelatedEvents("corr-789")).toEqual([]);
    });

    it("should handle causation chains", () => {
        const event1: BaseEvent = {
            id: "event-1",
            type: "USER_ACTION",
            source: { tier: 1, component: "ui", instanceId: "ui-1" },
            correlationId: "corr-123",
            timestamp: new Date(),
            metadata: { version: "1.0.0", tags: [], priority: "NORMAL" },
        };

        const event2: BaseEvent = {
            id: "event-2",
            type: "SYSTEM_RESPONSE",
            source: { tier: 2, component: "api", instanceId: "api-1" },
            correlationId: "corr-123",
            causationId: "event-1",
            timestamp: new Date(),
            metadata: { version: "1.0.0", tags: [], priority: "NORMAL" },
        };

        correlator.addEvent(event1);
        correlator.addEvent(event2);

        const correlated = correlator.getCorrelatedEvents("corr-123");
        expect(correlated).toEqual(expect.arrayContaining(["event-1", "event-2"]));
    });
});