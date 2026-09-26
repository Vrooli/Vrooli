// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md
import type { HealthResponse, EventEnvelope } from "../lib/api";

/**
 * Creates a mock HealthResponse with sensible defaults.
 * Override any field via the partial parameter.
 */
export function createMockHealthResponse(
    overrides: Partial<HealthResponse> = {},
): HealthResponse {
    return {
        status: "healthy",
        service: "vrooli-events",
        timestamp: new Date().toISOString(),
        readiness: true,
        subscribers: 0,
        store: {
            totalEvents: 0,
            totalPayloadBytes: 0,
        },
        ...overrides,
    };
}

/**
 * Creates a mock EventEnvelope with sensible defaults.
 * Override any field via the partial parameter.
 */
export function createMockEvent(
    overrides: Partial<EventEnvelope> = {},
): EventEnvelope {
    return {
        eventId: `evt-${Math.random().toString(36).slice(2, 8)}`,
        sourceScenario: "test-source",
        eventType: "test.domain.action.v1",
        createdAt: new Date().toISOString(),
        ...overrides,
    };
}
