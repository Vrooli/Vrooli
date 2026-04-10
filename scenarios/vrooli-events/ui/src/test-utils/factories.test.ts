// @vitest-environment node
import { describe, expect, it } from "vitest";
import { createMockHealthResponse, createMockEvent } from "./factories";

// [REQ:REQ-API-003] Health response factory produces valid defaults
describe("createMockHealthResponse", () => {
  it("returns defaults with all required fields", () => {
    const health = createMockHealthResponse();
    expect(health.status).toBe("healthy");
    expect(health.service).toBe("vrooli-events");
    expect(health.readiness).toBe(true);
    expect(health.subscribers).toBe(0);
    expect(health.store.totalEvents).toBe(0);
    expect(health.store.totalPayloadBytes).toBe(0);
    expect(health.timestamp).toBeTruthy();
  });

  it("allows partial overrides", () => {
    const health = createMockHealthResponse({
      status: "unhealthy",
      subscribers: 5,
    });
    expect(health.status).toBe("unhealthy");
    expect(health.subscribers).toBe(5);
    expect(health.service).toBe("vrooli-events");
  });
});

// [REQ:REQ-ES-002] Event factory produces valid event envelope
describe("createMockEvent", () => {
  it("returns defaults with all required fields", () => {
    const event = createMockEvent();
    expect(event.eventId).toBeTruthy();
    expect(event.sourceScenario).toBe("test-source");
    expect(event.eventType).toBe("test.domain.action.v1");
    expect(event.createdAt).toBeTruthy();
  });

  it("allows partial overrides", () => {
    const event = createMockEvent({
      eventId: "custom-id",
      correlationId: "trace-123",
    });
    expect(event.eventId).toBe("custom-id");
    expect(event.correlationId).toBe("trace-123");
    expect(event.sourceScenario).toBe("test-source");
  });

  it("generates unique event IDs", () => {
    const e1 = createMockEvent();
    const e2 = createMockEvent();
    expect(e1.eventId).not.toBe(e2.eventId);
  });
});
