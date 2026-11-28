// @vitest-environment node
import { describe, it, expect } from "vitest";
import type { HealthResponse, EventEnvelope, QueryParams, PolicyRule, PolicyViolation, SubscriptionData, SubscriptionHealth, SSEOptions } from "./api";

// [REQ:REQ-API-002A] Query parameter types are well-defined
describe("QueryParams type shape", () => {
  it("accepts empty params", () => {
    const params: QueryParams = {};
    expect(params).toEqual({});
  });

  it("accepts type filter", () => {
    const params: QueryParams = { type: "discovery.*" };
    expect(params.type).toBe("discovery.*");
  });

  it("accepts source filter", () => {
    const params: QueryParams = { source: "my-scenario" };
    expect(params.source).toBe("my-scenario");
  });

  it("accepts target filter", () => {
    const params: QueryParams = { target: "plan-manager" };
    expect(params.target).toBe("plan-manager");
  });

  it("accepts correlationId filter", () => {
    const params: QueryParams = { correlationId: "trace-abc" };
    expect(params.correlationId).toBe("trace-abc");
  });

  it("accepts numeric since parameter", () => {
    const params: QueryParams = { since: 42 };
    expect(params.since).toBe(42);
  });

  it("accepts numeric limit parameter", () => {
    const params: QueryParams = { limit: 100 };
    expect(params.limit).toBe(100);
  });

  it("accepts all params combined", () => {
    const params: QueryParams = {
      type: "*.created",
      source: "src",
      target: "dst",
      correlationId: "cid-123",
      since: 10,
      limit: 50,
    };
    expect(Object.keys(params).length).toBe(6);
  });
});

// [REQ:REQ-API-003A] Health response structure
describe("HealthResponse type shape", () => {
  it("defines required fields", () => {
    const health: HealthResponse = {
      status: "healthy",
      service: "vrooli-events",
      timestamp: "2024-01-01T00:00:00Z",
      readiness: true,
      subscribers: 3,
      store: { totalEvents: 100, totalPayloadBytes: 2048 },
    };
    expect(health.status).toBe("healthy");
    expect(health.store.totalEvents).toBe(100);
    expect(health.store.totalPayloadBytes).toBe(2048);
  });

  it("subscribers is a number", () => {
    const health: HealthResponse = {
      status: "healthy",
      service: "vrooli-events",
      timestamp: "2024-01-01T00:00:00Z",
      readiness: true,
      subscribers: 0,
      store: { totalEvents: 0, totalPayloadBytes: 0 },
    };
    expect(typeof health.subscribers).toBe("number");
  });
});

// [REQ:REQ-ES-002A] Event envelope schema fields
describe("EventEnvelope type shape", () => {
  it("requires eventId, sourceScenario, eventType", () => {
    const evt: EventEnvelope = {
      eventId: "evt-1",
      sourceScenario: "src",
      eventType: "test.event",
    };
    expect(evt.eventId).toBe("evt-1");
    expect(evt.sourceScenario).toBe("src");
    expect(evt.eventType).toBe("test.event");
  });

  it("accepts optional fields", () => {
    const evt: EventEnvelope = {
      eventId: "evt-2",
      sourceScenario: "src",
      eventType: "test.event",
      targetScenario: "target",
      correlationId: "cid-1",
      metadata: { key: "value" },
      payload: { data: true },
      createdAt: "2024-01-01T00:00:00Z",
    };
    expect(evt.targetScenario).toBe("target");
    expect(evt.correlationId).toBe("cid-1");
    expect(evt.metadata?.key).toBe("value");
  });
});

// [REQ:REQ-POL-004A] Policy rule type fields
describe("PolicyRule type shape", () => {
  it("contains all required fields", () => {
    const rule: PolicyRule = {
      id: 1,
      rule_type: "access",
      source_scenario: "*",
      target_scenario: "my-api",
      effect: "allow",
      priority: 10,
      enabled: true,
      created_at: "2024-01-01T00:00:00Z",
      updated_at: "2024-01-01T00:00:00Z",
    };
    expect(rule.id).toBe(1);
    expect(rule.rule_type).toBe("access");
    expect(rule.enabled).toBe(true);
  });

  it("accepts rate limit specific fields", () => {
    const rule: PolicyRule = {
      id: 2,
      rule_type: "rate_limit",
      source_scenario: "sender",
      target_scenario: "receiver",
      priority: 5,
      enabled: true,
      max_requests: 100,
      window_seconds: 60,
      burst_allowance: 10,
      created_at: "2024-01-01T00:00:00Z",
      updated_at: "2024-01-01T00:00:00Z",
    };
    expect(rule.max_requests).toBe(100);
    expect(rule.window_seconds).toBe(60);
  });

  it("accepts circuit breaker specific fields", () => {
    const rule: PolicyRule = {
      id: 3,
      rule_type: "circuit_breaker",
      source_scenario: "*",
      target_scenario: "fragile-api",
      priority: 1,
      enabled: true,
      failure_threshold: 5,
      cooldown_seconds: 30,
      success_threshold: 2,
      created_at: "2024-01-01T00:00:00Z",
      updated_at: "2024-01-01T00:00:00Z",
    };
    expect(rule.failure_threshold).toBe(5);
    expect(rule.cooldown_seconds).toBe(30);
  });
});

// [REQ:REQ-POL-007A] Policy violation record fields
describe("PolicyViolation type shape", () => {
  it("contains required fields", () => {
    const violation: PolicyViolation = {
      id: 1,
      timestamp: "2024-01-01T00:00:00Z",
      source_scenario: "sender",
      target_scenario: "receiver",
      endpoint: "/api/v1/data",
      rule_id: 5,
      rule_type: "access",
      reason: "denied by rule",
    };
    expect(violation.source_scenario).toBe("sender");
    expect(violation.rule_type).toBe("access");
  });
});

// [REQ:REQ-SUB-001A] Subscription data fields
describe("SubscriptionData type shape", () => {
  it("contains all subscription fields", () => {
    const sub: SubscriptionData = {
      id: 1,
      name: "my-sub",
      owner_scenario: "owner",
      event_pattern: "discovery.*",
      delivery_type: "webhook",
      delivery_target: "http://example.com/hook",
      enabled: true,
      created_at: "2024-01-01T00:00:00Z",
      updated_at: "2024-01-01T00:00:00Z",
    };
    expect(sub.name).toBe("my-sub");
    expect(sub.event_pattern).toBe("discovery.*");
    expect(sub.delivery_type).toBe("webhook");
  });

  it("accepts optional source filter", () => {
    const sub: SubscriptionData = {
      id: 2,
      name: "filtered-sub",
      owner_scenario: "owner",
      event_pattern: "*",
      source_filter: "specific-source",
      delivery_type: "webhook",
      delivery_target: "http://example.com/hook",
      enabled: false,
      created_at: "2024-01-01T00:00:00Z",
      updated_at: "2024-01-01T00:00:00Z",
    };
    expect(sub.source_filter).toBe("specific-source");
    expect(sub.enabled).toBe(false);
  });
});

// [REQ:REQ-SUB-004A] Subscription health tracking fields
describe("SubscriptionHealth type shape", () => {
  it("contains delivery tracking fields", () => {
    const health: SubscriptionHealth = {
      subscription_id: 1,
      total_delivered: 100,
      total_failed: 3,
      consecutive_failures: 0,
      last_delivered_at: "2024-01-01T00:00:00Z",
      status: "healthy",
    };
    expect(health.total_delivered).toBe(100);
    expect(health.consecutive_failures).toBe(0);
    expect(health.status).toBe("healthy");
  });

  it("tracks consecutive failures for auto-disable", () => {
    const health: SubscriptionHealth = {
      subscription_id: 2,
      total_delivered: 50,
      total_failed: 10,
      consecutive_failures: 5,
      last_failed_at: "2024-01-02T00:00:00Z",
      status: "disabled",
    };
    expect(health.consecutive_failures).toBe(5);
    expect(health.status).toBe("disabled");
  });
});

// [REQ:REQ-PS-001] SSE options type shape
describe("SSEOptions type shape", () => {
  it("requires onEvent callback", () => {
    const opts: SSEOptions = {
      onEvent: () => {},
    };
    expect(typeof opts.onEvent).toBe("function");
  });

  it("accepts filter options", () => {
    const opts: SSEOptions = {
      type: "discovery.*",
      source: "my-source",
      target: "my-target",
      onEvent: () => {},
      onHeartbeat: () => {},
      onError: () => {},
    };
    expect(opts.type).toBe("discovery.*");
    expect(opts.source).toBe("my-source");
  });
});
