import { describe, expect, it, beforeEach } from "vitest";
import { EventTypeRegistry, type EventTypeMetadata } from "./eventTypes.js";
import { EventCategory } from "../types/events.js";

describe("EventTypeRegistry", () => {
    let registry: EventTypeRegistry;

    beforeEach(() => {
        // Reset singleton instance for each test
        (EventTypeRegistry as any).instance = undefined;
        registry = EventTypeRegistry.getInstance();
    });

    it("should be a singleton", () => {
        const registry1 = EventTypeRegistry.getInstance();
        const registry2 = EventTypeRegistry.getInstance();
        
        expect(registry1).toBe(registry2);
    });

    it("should register and retrieve event types", () => {
        const metadata: EventTypeMetadata = {
            type: "TEST_EVENT",
            category: EventCategory.LIFECYCLE,
            tier: 1,
            description: "Test event for unit testing",
            schema: { test: "value" },
            examples: [{ action: "test" }],
        };

        registry.register(metadata);
        const retrieved = registry.get("TEST_EVENT");

        expect(retrieved).toEqual(metadata);
    });

    it("should return undefined for non-existent event types", () => {
        const retrieved = registry.get("NON_EXISTENT_EVENT");
        expect(retrieved).toBeUndefined();
    });

    it("should filter events by category", () => {
        const lifecycleEvent: EventTypeMetadata = {
            type: "LIFECYCLE_TEST",
            category: EventCategory.LIFECYCLE,
            tier: 1,
            description: "Lifecycle test event",
        };

        const errorEvent: EventTypeMetadata = {
            type: "ERROR_TEST",
            category: EventCategory.ERROR,
            tier: 2,
            description: "Error test event",
        };

        registry.register(lifecycleEvent);
        registry.register(errorEvent);

        const lifecycleEvents = registry.getByCategory(EventCategory.LIFECYCLE);
        const errorEvents = registry.getByCategory(EventCategory.ERROR);

        expect(lifecycleEvents).toContain(lifecycleEvent);
        expect(errorEvents).toContain(errorEvent);
        expect(lifecycleEvents).not.toContain(errorEvent);
        expect(errorEvents).not.toContain(lifecycleEvent);
    });

    it("should filter events by tier", () => {
        const tier1Event: EventTypeMetadata = {
            type: "TIER1_TEST",
            category: EventCategory.LIFECYCLE,
            tier: 1,
            description: "Tier 1 test event",
        };

        const tier2Event: EventTypeMetadata = {
            type: "TIER2_TEST",
            category: EventCategory.LIFECYCLE,
            tier: 2,
            description: "Tier 2 test event",
        };

        const crossCuttingEvent: EventTypeMetadata = {
            type: "CROSS_CUTTING_TEST",
            category: EventCategory.SECURITY,
            tier: "cross-cutting",
            description: "Cross-cutting test event",
        };

        registry.register(tier1Event);
        registry.register(tier2Event);
        registry.register(crossCuttingEvent);

        const tier1Events = registry.getByTier(1);
        const tier2Events = registry.getByTier(2);
        const crossCuttingEvents = registry.getByTier("cross-cutting");

        expect(tier1Events).toContain(tier1Event);
        expect(tier2Events).toContain(tier2Event);
        expect(crossCuttingEvents).toContain(crossCuttingEvent);

        expect(tier1Events).not.toContain(tier2Event);
        expect(tier2Events).not.toContain(tier1Event);
        expect(tier1Events).not.toContain(crossCuttingEvent);
    });

    it("should return all registered event types", () => {
        const event1: EventTypeMetadata = {
            type: "EVENT_1",
            category: EventCategory.LIFECYCLE,
            tier: 1,
            description: "Event 1",
        };

        const event2: EventTypeMetadata = {
            type: "EVENT_2",
            category: EventCategory.ERROR,
            tier: 2,
            description: "Event 2",
        };

        registry.register(event1);
        registry.register(event2);

        const allTypes = registry.getAllTypes();

        expect(allTypes).toContain("EVENT_1");
        expect(allTypes).toContain("EVENT_2");
    });

    it("should register default types during initialization", () => {
        const allTypes = registry.getAllTypes();

        // Should contain some default event types
        expect(allTypes.length).toBeGreaterThan(0);
        
        // Check for specific cross-cutting events that should be registered
        expect(allTypes).toContain("SECURITY_VIOLATION");
        expect(allTypes).toContain("RESOURCE_EXHAUSTED");
    });

    it("should handle overwriting existing event types", () => {
        const originalMetadata: EventTypeMetadata = {
            type: "OVERWRITE_TEST",
            category: EventCategory.LIFECYCLE,
            tier: 1,
            description: "Original description",
        };

        const updatedMetadata: EventTypeMetadata = {
            type: "OVERWRITE_TEST",
            category: EventCategory.ERROR,
            tier: 2,
            description: "Updated description",
        };

        registry.register(originalMetadata);
        registry.register(updatedMetadata);

        const retrieved = registry.get("OVERWRITE_TEST");
        expect(retrieved).toEqual(updatedMetadata);
        expect(retrieved?.description).toBe("Updated description");
        expect(retrieved?.category).toBe(EventCategory.ERROR);
        expect(retrieved?.tier).toBe(2);
    });

    it("should return empty arrays for non-existent categories and tiers", () => {
        // Assuming there's no event with category PERFORMANCE registered by default
        const performanceEvents = registry.getByCategory(EventCategory.PERFORMANCE);
        expect(performanceEvents).toEqual([]);

        // Tier 3 might not have any events registered by default
        const tier3Events = registry.getByTier(3);
        expect(Array.isArray(tier3Events)).toBe(true);
    });

    it("should handle event metadata with optional fields", () => {
        const minimalMetadata: EventTypeMetadata = {
            type: "MINIMAL_EVENT",
            category: EventCategory.LIFECYCLE,
            tier: 1,
            description: "Minimal event metadata",
        };

        const fullMetadata: EventTypeMetadata = {
            type: "FULL_EVENT",
            category: EventCategory.LIFECYCLE,
            tier: 1,
            description: "Full event metadata",
            schema: {
                type: "object",
                properties: {
                    action: { type: "string" },
                    timestamp: { type: "number" },
                },
            },
            examples: [
                { action: "start", timestamp: 1234567890 },
                { action: "stop", timestamp: 1234567900 },
            ],
        };

        registry.register(minimalMetadata);
        registry.register(fullMetadata);

        const retrievedMinimal = registry.get("MINIMAL_EVENT");
        const retrievedFull = registry.get("FULL_EVENT");

        expect(retrievedMinimal?.schema).toBeUndefined();
        expect(retrievedMinimal?.examples).toBeUndefined();

        expect(retrievedFull?.schema).toEqual(fullMetadata.schema);
        expect(retrievedFull?.examples).toEqual(fullMetadata.examples);
    });

    it("should handle special tier values", () => {
        const crossCuttingEvent: EventTypeMetadata = {
            type: "CROSS_CUTTING_SPECIAL",
            category: EventCategory.SECURITY,
            tier: "cross-cutting",
            description: "Cross-cutting special event",
        };

        registry.register(crossCuttingEvent);

        const crossCuttingEvents = registry.getByTier("cross-cutting");
        expect(crossCuttingEvents).toContain(crossCuttingEvent);

        const tier1Events = registry.getByTier(1);
        expect(tier1Events).not.toContain(crossCuttingEvent);
    });

    describe("Edge cases", () => {
        it("should handle empty string event type", () => {
            const emptyTypeEvent: EventTypeMetadata = {
                type: "",
                category: EventCategory.LIFECYCLE,
                tier: 1,
                description: "Empty type event",
            };

            registry.register(emptyTypeEvent);
            const retrieved = registry.get("");

            expect(retrieved).toEqual(emptyTypeEvent);
        });

        it("should handle very long event type names", () => {
            const longTypeName = "A".repeat(1000);
            const longTypeEvent: EventTypeMetadata = {
                type: longTypeName,
                category: EventCategory.LIFECYCLE,
                tier: 1,
                description: "Long type name event",
            };

            registry.register(longTypeEvent);
            const retrieved = registry.get(longTypeName);

            expect(retrieved).toEqual(longTypeEvent);
        });

        it("should handle special characters in event type names", () => {
            const specialTypeEvent: EventTypeMetadata = {
                type: "SPECIAL_EVENT_@#$%^&*()",
                category: EventCategory.LIFECYCLE,
                tier: 1,
                description: "Special characters event",
            };

            registry.register(specialTypeEvent);
            const retrieved = registry.get("SPECIAL_EVENT_@#$%^&*()");

            expect(retrieved).toEqual(specialTypeEvent);
        });

        it("should handle complex schema objects", () => {
            const complexSchema = {
                type: "object",
                properties: {
                    nested: {
                        type: "object",
                        properties: {
                            deeply: {
                                type: "object",
                                properties: {
                                    value: { type: "string" },
                                },
                            },
                        },
                    },
                    array: {
                        type: "array",
                        items: { type: "string" },
                    },
                },
                required: ["nested"],
            };

            const complexEvent: EventTypeMetadata = {
                type: "COMPLEX_SCHEMA_EVENT",
                category: EventCategory.LIFECYCLE,
                tier: 1,
                description: "Complex schema event",
                schema: complexSchema,
            };

            registry.register(complexEvent);
            const retrieved = registry.get("COMPLEX_SCHEMA_EVENT");

            expect(retrieved?.schema).toEqual(complexSchema);
        });
    });
});