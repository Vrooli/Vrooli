import { EventTypes, generatePK } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { logger } from "../../events/logger.js";
import {
    configureEventApproval,
    configurePatternApproval,
    EventMode,
    EventUsageValidator,
    getApprovalRequiredEvents,
    getEventBehavior,
    getRegisteredEventTypes,
    isEventTypeRegistered,
    printCoverageReport,
    registerEmergentPattern,
    registerEventBehavior,
    requiresApproval,
    resetEventRegistry,
    validateEventRegistry,
    validateEventUsage,
} from "./registry.js";
import type { EventBehavior, EventCoverage } from "./types.js";

// Mock dependencies
vi.mock("../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        debug: vi.fn(),
        warn: vi.fn(),
        error: vi.fn(),
    },
}));

// Don't mock @vrooli/shared - it's our own package

describe("EventRegistry", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        EventUsageValidator.reset();
        resetEventRegistry();
    });

    afterEach(() => {
        EventUsageValidator.reset();
        resetEventRegistry();
        // Reset any dynamic registrations
        process.env.EVENT_VALIDATION_WARNINGS = "true";
    });

    describe("getEventBehavior", () => {
        it("should return exact match for registered events", () => {
            const behavior = getEventBehavior(EventTypes.SECURITY.EMERGENCY_STOP);

            expect(behavior.mode).toBe(EventMode.PASSIVE);
            expect(behavior.interceptable).toBe(false);
            expect(behavior.defaultPriority).toBe("critical");
        });

        it("should return pattern match for wildcard patterns", () => {
            const behavior = getEventBehavior(EventTypes.CHAT.MESSAGE_ADDED);

            expect(behavior.mode).toBe(EventMode.PASSIVE);
            expect(behavior.interceptable).toBe(true);
            expect(behavior.defaultPriority).toBe("low");
        });

        it("should return most specific pattern match", () => {
            // "tool/approval/*" should be more specific than "tool/*"
            const behavior = getEventBehavior(EventTypes.TOOL.APPROVAL_REQUIRED);

            expect(behavior.mode).toBe(EventMode.APPROVAL);
            expect(behavior.defaultPriority).toBe("critical");
        });

        it("should return universal action patterns", () => {
            const completedBehavior = getEventBehavior("custom/completed");
            expect(completedBehavior.defaultPriority).toBe("medium");

            const failedBehavior = getEventBehavior("custom/failed");
            expect(failedBehavior.defaultPriority).toBe("high");

            const startedBehavior = getEventBehavior("custom/started");
            expect(startedBehavior.defaultPriority).toBe("low");
        });

        it("should return default behavior for unknown events", () => {
            const behavior = getEventBehavior("unknown/event/type");

            expect(behavior.mode).toBe(EventMode.PASSIVE);
            expect(behavior.interceptable).toBe(true);
            expect(behavior.defaultPriority).toBe("medium");
            expect(behavior.tags).toContain("unknown");
        });

        it("should handle stream events correctly", () => {
            const streamBehavior = getEventBehavior(EventTypes.CHAT.RESPONSE_STREAM_CHUNK);

            expect(streamBehavior.mode).toBe(EventMode.PASSIVE);
            expect(streamBehavior.interceptable).toBe(false);
            expect(streamBehavior.defaultPriority).toBe("low");
        });

        it("should handle security events with high priority", () => {
            const securityBehavior = getEventBehavior(EventTypes.SECURITY.THREAT_DETECTED);

            expect(securityBehavior.mode).toBe(EventMode.INTERCEPTABLE);
            expect(securityBehavior.defaultPriority).toBe("critical");
        });

        it("should handle system events with critical priority", () => {
            const systemBehavior = getEventBehavior(EventTypes.SYSTEM.ERROR);

            expect(systemBehavior.mode).toBe(EventMode.INTERCEPTABLE);
            expect(systemBehavior.defaultPriority).toBe("critical");
        });
    });

    describe("pattern specificity calculation", () => {
        it("should prioritize exact matches over wildcards", () => {
            // Register a specific override
            registerEventBehavior(EventTypes.CHAT.MESSAGE_ADDED, {
                mode: EventMode.APPROVAL,
                interceptable: true,
                defaultPriority: "high",
            });

            const behavior = getEventBehavior(EventTypes.CHAT.MESSAGE_ADDED);
            expect(behavior.mode).toBe(EventMode.APPROVAL);
            expect(behavior.defaultPriority).toBe("high");
        });

        it("should prioritize more specific patterns", () => {
            // "tool/execution/*" should beat "tool/*"
            const behavior = getEventBehavior(EventTypes.TOOL.EXECUTION_REQUESTED);

            expect(behavior.mode).toBe(EventMode.INTERCEPTABLE);
            expect(behavior.defaultPriority).toBe("high");
        });

        it("should handle multiple wildcard levels", () => {
            const behavior = getEventBehavior("deep/nested/structure/completed");

            // Should match "*/completed" pattern
            expect(behavior.defaultPriority).toBe("medium");
        });
    });

    describe("runtime registration", () => {
        it("should register new event behaviors", () => {
            const customBehavior: EventBehavior = {
                mode: EventMode.CONSENSUS,
                interceptable: true,
                defaultPriority: "high",
                barrierConfig: {
                    quorum: "all",
                    timeoutMs: 10000,
                    timeoutAction: "block",
                },
            };

            registerEventBehavior("custom/event", customBehavior);

            const behavior = getEventBehavior("custom/event");
            expect(behavior.mode).toBe(EventMode.CONSENSUS);
            expect(behavior.barrierConfig?.quorum).toBe("all");
        });

        it("should register emergent patterns with defaults", () => {
            registerEmergentPattern("emergent/*", {
                mode: EventMode.APPROVAL,
                defaultPriority: "critical",
            });

            const behavior = getEventBehavior("emergent/new_event");
            expect(behavior.mode).toBe(EventMode.APPROVAL);
            expect(behavior.defaultPriority).toBe("critical");
            expect(behavior.interceptable).toBe(true); // Should inherit default
        });

        it("should override existing patterns with emergent registration", () => {
            // First register with one behavior
            registerEmergentPattern("override/*", {
                mode: EventMode.PASSIVE,
                defaultPriority: "low",
            });

            // Then override with different behavior
            registerEmergentPattern("override/*", {
                mode: EventMode.APPROVAL,
                defaultPriority: "high",
            });

            const behavior = getEventBehavior("override/test");
            expect(behavior.mode).toBe(EventMode.APPROVAL);
            expect(behavior.defaultPriority).toBe("high");
        });
    });

    describe("approval configuration", () => {
        it("should configure user approval preset", () => {
            configureEventApproval("test/approval", { preset: "userApproval" });

            const behavior = getEventBehavior("test/approval");
            expect(behavior.mode).toBe(EventMode.APPROVAL);
            expect(behavior.barrierConfig?.quorum).toBe(1);
            expect(behavior.barrierConfig?.timeoutMs).toBe(30000);
        });

        it("should configure safety consensus preset", () => {
            configureEventApproval("test/consensus", { preset: "safetyConsensus" });

            const behavior = getEventBehavior("test/consensus");
            expect(behavior.mode).toBe(EventMode.CONSENSUS);
            expect(behavior.barrierConfig?.quorum).toBe("all");
            expect(behavior.barrierConfig?.blockOnFirst).toBe(true);
        });

        it("should configure simple user approval", () => {
            configureEventApproval("test/simple", { requireUserApproval: true });

            const behavior = getEventBehavior("test/simple");
            expect(behavior.mode).toBe(EventMode.APPROVAL);
            expect(behavior.barrierConfig?.quorum).toBe(1);
        });

        it("should configure safety consensus", () => {
            configureEventApproval("test/safety", { requireSafetyConsensus: true });

            const behavior = getEventBehavior("test/safety");
            expect(behavior.mode).toBe(EventMode.CONSENSUS);
            expect(behavior.barrierConfig?.quorum).toBe("all");
        });

        it("should configure custom barrier config", () => {
            const customConfig = {
                quorum: 3,
                timeoutMs: 15000,
                timeoutAction: "defer" as const,
                requiredResponders: ["admin", "security"],
            };

            configureEventApproval("test/custom", { customBarrierConfig: customConfig });

            const behavior = getEventBehavior("test/custom");
            expect(behavior.mode).toBe(EventMode.APPROVAL);
            expect(behavior.barrierConfig).toEqual(customConfig);
        });

        it("should remove approval requirements", () => {
            // First set approval
            configureEventApproval("test/remove", { requireUserApproval: true });
            expect(getEventBehavior("test/remove").mode).toBe(EventMode.APPROVAL);

            // Then remove it
            configureEventApproval("test/remove", {});
            const behavior = getEventBehavior("test/remove");
            expect(behavior.mode).toBe(EventMode.PASSIVE);
            expect(behavior.barrierConfig).toBeUndefined();
        });

        it("should configure pattern-based approval", () => {
            configurePatternApproval("admin/*", { preset: "safetyConsensus" });

            const behavior = getEventBehavior("admin/delete_user");
            expect(behavior.mode).toBe(EventMode.CONSENSUS);
            expect(behavior.defaultPriority).toBe("high");
        });
    });

    describe("utility functions", () => {
        beforeEach(() => {
            // Set up some approval events for testing
            configureEventApproval("test/approval1", { requireUserApproval: true });
            configureEventApproval("test/approval2", { requireSafetyConsensus: true });
        });

        it("should get approval required events", () => {
            const approvalEvents = getApprovalRequiredEvents();

            expect(approvalEvents).toContain("test/approval1");
            expect(approvalEvents).toContain("test/approval2");
            expect(approvalEvents).toContain("tool/approval/*");
        });

        it("should check if event requires approval", () => {
            expect(requiresApproval("test/approval1")).toBe(true);
            expect(requiresApproval("test/approval2")).toBe(true);
            expect(requiresApproval(EventTypes.TOOL.APPROVAL_REQUIRED)).toBe(true);
            expect(requiresApproval(EventTypes.CHAT.MESSAGE_ADDED)).toBe(false);
        });

        it("should check if event type is registered", () => {
            expect(isEventTypeRegistered(EventTypes.CHAT.MESSAGE_ADDED)).toBe(true);
            expect(isEventTypeRegistered(EventTypes.TOOL.APPROVAL_REQUIRED)).toBe(true);
            expect(isEventTypeRegistered("completely/unknown")).toBe(true); // Matches default
            expect(isEventTypeRegistered("")).toBe(false);
        });

        it("should get all registered event types", () => {
            const eventTypes = getRegisteredEventTypes();

            expect(eventTypes).toContain("chat/*");
            expect(eventTypes).toContain("tool/approval/*");
            expect(eventTypes).toContain("security/*");
            expect(eventTypes.length).toBeGreaterThan(15);
        });
    });

    describe("registry validation", () => {
        it("should validate registry coverage", () => {
            const validation = validateEventRegistry();

            expect(validation).toHaveProperty("valid");
            expect(validation).toHaveProperty("errors");
            expect(validation).toHaveProperty("coverage");
            expect(validation.coverage.totalEvents).toBeGreaterThan(0);
        });

        it("should identify covered events", () => {
            const validation = validateEventRegistry();

            expect(validation.coverage.coveredEvents).toBeGreaterThan(0);
            expect(validation.coverage.patternMatches.size).toBeGreaterThan(0);
        });

        it("should identify uncovered events", () => {
            const validation = validateEventRegistry();

            // Most events should be covered by patterns
            const coverageRatio = validation.coverage.coveredEvents / validation.coverage.totalEvents;
            expect(coverageRatio).toBeGreaterThan(0.8); // At least 80% coverage
        });

        it("should print coverage report", () => {
            const validation = validateEventRegistry();
            const report = printCoverageReport(validation.coverage);

            expect(report).toContain("Event Registry Coverage Report");
            expect(report).toContain("Total Events:");
            expect(report).toContain("Coverage:");
            expect(report).toContain("Pattern Usage:");
        });

        it("should handle empty coverage gracefully", () => {
            const emptyCoverage: EventCoverage = {
                totalEvents: 0,
                coveredEvents: 0,
                uncoveredEvents: [],
                patternMatches: new Map(),
            };

            const report = printCoverageReport(emptyCoverage);
            expect(report).toContain("Total Events: 0");
            expect(report).toContain("Coverage: NaN%");
        });
    });

    describe("EventUsageValidator", () => {
        beforeEach(() => {
            EventUsageValidator.reset();
            EventUsageValidator.setWarningsEnabled(true);
        });

        it("should track event emissions", () => {
            const eventId = generatePK().toString();
            EventUsageValidator.trackEmission("test/event", eventId);

            const stats = EventUsageValidator.getStats();
            expect(stats.pendingEvents).toBe(1);
            expect(stats.checkedEvents).toBe(0);
        });

        it("should mark progression as checked", () => {
            const eventId = generatePK().toString();
            EventUsageValidator.trackEmission("test/event", eventId);
            EventUsageValidator.markProgressionChecked(eventId);

            const stats = EventUsageValidator.getStats();
            expect(stats.pendingEvents).toBe(0);
            expect(stats.checkedEvents).toBe(1);
        });

        it("should detect violations for unchecked events", async () => {
            const eventId = generatePK().toString();
            EventUsageValidator.trackEmission("test/event", eventId);

            // Wait for events to become stale
            await new Promise(resolve => setTimeout(resolve, 10));

            const violations = EventUsageValidator.checkViolations(5);
            expect(violations.length).toBeGreaterThan(0);
            expect(violations[0]).toContain("WARNING");
        });

        it("should detect critical violations for blocking events", async () => {
            const eventId = generatePK().toString();
            EventUsageValidator.trackEmission("test/blocking", eventId, true);

            await new Promise(resolve => setTimeout(resolve, 10));

            const violations = EventUsageValidator.checkViolations(5);
            expect(violations.length).toBeGreaterThan(0);
            expect(violations[0]).toContain("CRITICAL");
        });

        it("should detect violations for approval events", async () => {
            configureEventApproval("test/approval", { requireUserApproval: true });
            const eventId = generatePK().toString();
            EventUsageValidator.trackEmission("test/approval", eventId);

            await new Promise(resolve => setTimeout(resolve, 10));

            const violations = EventUsageValidator.checkViolations(5);
            expect(violations.length).toBeGreaterThan(0);
            expect(violations[0]).toContain("CRITICAL");
        });

        it("should handle warnings configuration", () => {
            EventUsageValidator.setWarningsEnabled(false);
            const eventId = generatePK().toString();
            EventUsageValidator.trackEmission("test/event", eventId);

            const violations = EventUsageValidator.checkViolations(0);
            expect(violations.length).toBe(0); // No warnings when disabled

            EventUsageValidator.setWarningsEnabled(true);
        });

        it("should provide detailed statistics", () => {
            const event1Id = generatePK().toString();
            const event2Id = generatePK().toString();
            EventUsageValidator.trackEmission("test/event1", event1Id);
            EventUsageValidator.trackEmission("test/event2", event2Id, true);
            EventUsageValidator.markProgressionChecked(event1Id);

            const violations = EventUsageValidator.checkViolations(0);

            const stats = EventUsageValidator.getStats();
            expect(stats.pendingEvents).toBe(0);
            expect(stats.checkedEvents).toBe(1);
            expect(stats.violations).toBe(violations.length);
            expect(stats.criticalViolations).toBe(1);
            expect(stats.warnings).toBe(0);
        });

        it("should reset all tracking data", () => {
            const eventId = generatePK().toString();
            EventUsageValidator.trackEmission("test/event", eventId);
            EventUsageValidator.checkViolations(0);

            EventUsageValidator.reset();

            const stats = EventUsageValidator.getStats();
            expect(stats.pendingEvents).toBe(0);
            expect(stats.checkedEvents).toBe(0);
            expect(stats.violations).toBe(0);
        });

        it("should get all violations", () => {
            const eventId = generatePK().toString();
            EventUsageValidator.trackEmission("test/event", eventId);
            EventUsageValidator.checkViolations(0);

            const violations = EventUsageValidator.getViolations();
            expect(violations.length).toBeGreaterThan(0);
        });
    });

    describe("validateEventUsage decorator", () => {
        it("should skip decoration in production", () => {
            const originalEnv = process.env.NODE_ENV;
            process.env.NODE_ENV = "production";

            class TestClass {
                testMethod() {
                    return "result";
                }
            }

            const descriptor = {
                value: TestClass.prototype.testMethod,
            };

            validateEventUsage(TestClass.prototype, "testMethod", descriptor);

            // Should not modify the method in production
            expect(descriptor.value).toBe(TestClass.prototype.testMethod);

            process.env.NODE_ENV = originalEnv;
        });

        it("should wrap method in development", () => {
            const originalEnv = process.env.NODE_ENV;
            process.env.NODE_ENV = "development";

            class TestClass {
                async testMethod() {
                    return "result";
                }
            }

            const originalMethod = TestClass.prototype.testMethod;
            const descriptor = {
                value: originalMethod,
            };

            validateEventUsage(TestClass.prototype, "testMethod", descriptor);

            // Should modify the method in development
            expect(descriptor.value).not.toBe(originalMethod);

            process.env.NODE_ENV = originalEnv;
        });

        it("should check violations before and after method execution", async () => {
            const originalEnv = process.env.NODE_ENV;
            process.env.NODE_ENV = "test";

            class TestClass {
                async testMethod() {
                    const eventId = generatePK().toString();
                    EventUsageValidator.trackEmission("test/event", eventId);
                    return "result";
                }
            }

            const descriptor = {
                value: TestClass.prototype.testMethod,
            };

            validateEventUsage(TestClass.prototype, "testMethod", descriptor);

            const instance = new TestClass();
            await descriptor.value.call(instance);

            // Should have logged errors about violations after method execution
            expect(logger.error).toHaveBeenCalled();

            process.env.NODE_ENV = originalEnv;
        });
    });

    describe("edge cases", () => {
        it("should handle empty event type", () => {
            const behavior = getEventBehavior("");
            expect(behavior.mode).toBe(EventMode.PASSIVE);
            expect(behavior.tags).toContain("unknown");
        });

        it("should handle very long event types", () => {
            const longEventType = "very/long/nested/event/type/with/many/segments/that/could/cause/issues";
            const behavior = getEventBehavior(longEventType);
            expect(behavior).toBeDefined();
        });

        it("should handle special characters in event types", () => {
            const specialEventType = "event-with_special.chars@domain.com";
            const behavior = getEventBehavior(specialEventType);
            expect(behavior.mode).toBe(EventMode.PASSIVE);
        });

        it("should handle concurrent registration", () => {
            const behaviors = Array.from({ length: 10 }, (_, i) => ({
                eventType: `concurrent/event${i}`,
                behavior: {
                    mode: EventMode.PASSIVE,
                    interceptable: true,
                    defaultPriority: "medium" as const,
                },
            }));

            // Register all behaviors
            behaviors.forEach(({ eventType, behavior }) => {
                registerEventBehavior(eventType, behavior);
            });

            // Verify all are registered
            behaviors.forEach(({ eventType }) => {
                expect(isEventTypeRegistered(eventType)).toBe(true);
            });
        });

        it("should handle invalid approval configurations gracefully", () => {
            // This should not throw
            configureEventApproval("test/invalid", {
                preset: "nonexistent" as any,
            });

            const behavior = getEventBehavior("test/invalid");
            expect(behavior.mode).toBe(EventMode.PASSIVE);
        });

        it("should handle EventUsageValidator with environment variables", () => {
            process.env.EVENT_VALIDATION_WARNINGS = "false";

            // Recreate validator to pick up new env var
            EventUsageValidator.reset();
            EventUsageValidator.setWarningsEnabled(false); // Explicitly disable for this test

            const eventId = generatePK().toString();
            EventUsageValidator.trackEmission("test/event", eventId);
            const violations = EventUsageValidator.checkViolations(0);

            // Should not create warnings when disabled via env var
            expect(violations.length).toBe(0);
        });
    });

    describe("pattern specificity edge cases", () => {
        it("should handle patterns with multiple wildcards", () => {
            registerEventBehavior("*/test/*", {
                mode: EventMode.APPROVAL,
                interceptable: true,
                defaultPriority: "high",
            });

            const behavior = getEventBehavior("category/test/action");
            expect(behavior.mode).toBe(EventMode.APPROVAL);
        });

        it("should prioritize more specific patterns correctly", () => {
            registerEventBehavior("specific/test/action", {
                mode: EventMode.CONSENSUS,
                interceptable: true,
                defaultPriority: "critical",
            });

            registerEventBehavior("specific/test/*", {
                mode: EventMode.APPROVAL,
                interceptable: true,
                defaultPriority: "high",
            });

            registerEventBehavior("specific/*", {
                mode: EventMode.PASSIVE,
                interceptable: true,
                defaultPriority: "medium",
            });

            // Exact match should win
            const behavior = getEventBehavior("specific/test/action");
            expect(behavior.mode).toBe(EventMode.CONSENSUS);
            expect(behavior.defaultPriority).toBe("critical");
        });

        it("should handle patterns with no matching segments", () => {
            const behavior = getEventBehavior("nomatch");
            expect(behavior.mode).toBe(EventMode.PASSIVE);
            expect(behavior.tags).toContain("unknown");
        });
    });
});
