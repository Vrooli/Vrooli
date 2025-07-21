/**
 * Event Registry - Defines behavior for all event types
 * 
 * This registry predefines how each event type behaves in the system,
 * enabling emergent capabilities through configuration rather than code.
 */

import { EventTypes, SECONDS_5_MS, type SwarmEventTypeValue } from "@vrooli/shared";
import { logger } from "../../events/logger.js";
import type { EventBehavior } from "./types.js";
import { EventMode } from "./types.js";

export { EventMode };

/**
 * Approval configuration presets for common approval scenarios
 */
const ApprovalPresets = {
    userApproval: {
        mode: EventMode.APPROVAL,
        interceptable: true,
        barrierConfig: {
            quorum: 1,
            timeoutMs: 30000,
            timeoutAction: "block" as const,
            blockOnFirst: false,
        },
    },
    safetyConsensus: {
        mode: EventMode.CONSENSUS,
        interceptable: true,
        barrierConfig: {
            quorum: "all" as const,
            timeoutMs: 5000,
            timeoutAction: "block" as const,
            blockOnFirst: true,
        },
    },
    resourceApproval: {
        mode: EventMode.APPROVAL,
        interceptable: true,
        barrierConfig: {
            quorum: 1,
            timeoutMs: 10000,
            timeoutAction: "defer" as const,
            requiredResponders: ["resource-manager"] as string[],
        },
    },
    sensitiveDataAccess: {
        mode: EventMode.APPROVAL,
        interceptable: true,
        barrierConfig: {
            quorum: "all" as const,
            timeoutMs: 3000,
            timeoutAction: "block" as const,
            requiredResponders: ["data-security-agent"] as string[],
        },
    },
    externalApiCall: {
        mode: EventMode.INTERCEPTABLE,
        interceptable: true,
        barrierConfig: {
            quorum: 1,
            timeoutMs: 2000,
            timeoutAction: "continue" as const,
        },
    },
    highRiskTool: {
        mode: EventMode.APPROVAL,
        interceptable: true,
        barrierConfig: {
            quorum: 2,
            timeoutMs: 5000,
            timeoutAction: "block" as const,
            requiredResponders: ["security-agent"] as string[],
        },
    },
    emergentConsensus: {
        mode: EventMode.CONSENSUS,
        interceptable: true,
        barrierConfig: {
            quorum: "all" as const,
            timeoutMs: 1000,
            timeoutAction: "block" as const,
            blockOnFirst: true,
        },
    },
} as const;

/**
 * Simplified event registry using strategic wildcard patterns.
 * This covers all events with ~15 patterns instead of 75+ individual entries.
 * More specific patterns are matched first by getEventBehavior().
 */
export const EventRegistry: Record<SwarmEventTypeValue | string, EventBehavior> = {
    // ===== Universal Action Patterns (covers ~60% of events) =====
    "*/completed": {
        mode: EventMode.PASSIVE,
        interceptable: true,
        defaultPriority: "medium",
    },

    "*/failed": {
        mode: EventMode.PASSIVE,
        interceptable: true,
        defaultPriority: "high",
    },

    "*/started": {
        mode: EventMode.PASSIVE,
        interceptable: true,
        defaultPriority: "low",
    },

    "*/updated": {
        mode: EventMode.PASSIVE,
        interceptable: true,
        defaultPriority: "low",
    },

    // ===== Security Category (critical priority) =====
    "security/*": {
        mode: EventMode.INTERCEPTABLE,
        interceptable: true,
        defaultPriority: "critical",
    },

    // ===== Tool Category =====
    "tool/approval/*": {
        ...ApprovalPresets.userApproval,
        defaultPriority: "critical",
    },

    "tool/execution/*": {
        mode: EventMode.INTERCEPTABLE,
        interceptable: true,
        defaultPriority: "high",
    },

    "tool/*": {
        mode: EventMode.PASSIVE,
        interceptable: true,
        defaultPriority: "medium",
    },

    // ===== Data Category =====
    "data/*": {
        mode: EventMode.INTERCEPTABLE,
        interceptable: true,
        defaultPriority: "high",
    },

    // ===== API Category =====
    "api/*": {
        mode: EventMode.INTERCEPTABLE,
        interceptable: true,
        defaultPriority: "high",
    },

    // ===== Resource Category =====
    "resource/*": {
        mode: EventMode.INTERCEPTABLE,
        interceptable: true,
        defaultPriority: "high",
    },


    // ===== Stream Events =====
    "*/stream/*": {
        mode: EventMode.PASSIVE,
        interceptable: false,
        defaultPriority: "low",
    },

    // ===== Bot Stream Events =====
    "bot/*/stream": {
        mode: EventMode.PASSIVE,
        interceptable: false,
        defaultPriority: "low",
    },

    // ===== Broader Category Patterns =====
    "swarm/*": {
        mode: EventMode.INTERCEPTABLE,
        interceptable: true,
        defaultPriority: "medium",
    },

    "run/*": {
        mode: EventMode.PASSIVE,
        interceptable: true,
        defaultPriority: "medium",
    },

    "chat/*": {
        mode: EventMode.PASSIVE,
        interceptable: true,
        defaultPriority: "low",
    },

    "user/*": {
        mode: EventMode.PASSIVE,
        interceptable: false,
        defaultPriority: "low",
    },

    "room/*": {
        mode: EventMode.PASSIVE,
        interceptable: true,
        defaultPriority: "low",
    },

    "system/*": {
        mode: EventMode.INTERCEPTABLE,
        interceptable: true,
        defaultPriority: "critical",
    },

    // ===== Specific Overrides for Different Behavior =====
    [EventTypes.SECURITY.EMERGENCY_STOP]: {
        mode: EventMode.PASSIVE,
        interceptable: false, // Direct handling only - no interception
        defaultPriority: "critical",
    },

    [EventTypes.CHAT.CANCELLATION_REQUESTED]: {
        mode: EventMode.PASSIVE,
        interceptable: false, // Direct handling required
        defaultPriority: "critical",
    },

    [EventTypes.RUN.DECISION_REQUESTED]: {
        mode: EventMode.INTERCEPTABLE,
        interceptable: true,
        defaultPriority: "high",
    },

    [EventTypes.RESOURCE.LIMIT_EXCEEDED]: {
        mode: EventMode.INTERCEPTABLE,
        interceptable: true,
        defaultPriority: "critical",
    },

    // Note: All Chat streaming events (response/stream/*, reasoning/stream/*, bot/*/stream)
    // are covered by "*/stream/*" pattern which correctly sets them as non-interceptable
};

/**
 * Registry reset mechanism for tests
 * Saves original registry state and provides reset functionality
 */
const originalRegistryEntries = { ...EventRegistry };

export function resetEventRegistry(): void {
    // Clear all dynamic entries
    Object.keys(EventRegistry).forEach(key => {
        if (!(key in originalRegistryEntries)) {
            delete EventRegistry[key];
        }
    });
    
    // Restore original entries
    Object.entries(originalRegistryEntries).forEach(([key, value]) => {
        EventRegistry[key] = value;
    });
}

/**
 * Get event behavior for a given event type
 * Uses pattern matching with most specific patterns matched first
 */
export function getEventBehavior(eventType: string): EventBehavior & { tags?: string[] } {
    // Check for exact match first
    if (EventRegistry[eventType]) {
        return EventRegistry[eventType];
    }

    // Get all matching patterns sorted by specificity (most specific first)
    const matchingPatterns = Object.entries(EventRegistry)
        .filter(([pattern]) => pattern.includes("*"))
        .map(([pattern, behavior]) => ({
            pattern,
            behavior,
            specificity: calculatePatternSpecificity(pattern, eventType),
        }))
        .filter(item => item.specificity > 0)
        .sort((a, b) => b.specificity - a.specificity);

    // Return the most specific match
    if (matchingPatterns.length > 0) {
        return matchingPatterns[0].behavior;
    }

    // Default behavior for unknown events
    return {
        mode: EventMode.PASSIVE,
        interceptable: true,
        defaultPriority: "medium",
        tags: ["unknown"],
    };
}

/**
 * Calculate pattern specificity for event matching
 * Higher numbers = more specific patterns
 */
function calculatePatternSpecificity(pattern: string, eventType: string): number {
    const EXACT_MATCH_BONUS = 10;
    const WILDCARD_MATCH_BONUS = 1;
    const STRUCTURE_BONUS = 5;

    const regex = new RegExp("^" + pattern.replace(/\*/g, ".*") + "$");
    if (!regex.test(eventType)) {
        return 0; // No match
    }

    // Count non-wildcard segments (more specific = higher score)
    const patternParts = pattern.split("/");
    const eventParts = eventType.split("/");

    let specificity = 0;

    // Exact segment matches count more than wildcard matches
    for (let i = 0; i < Math.min(patternParts.length, eventParts.length); i++) {
        if (patternParts[i] === eventParts[i]) {
            specificity += EXACT_MATCH_BONUS;
        } else if (patternParts[i] === "*") {
            specificity += WILDCARD_MATCH_BONUS;
        }
    }

    // Prefer patterns with more specific structure
    specificity += (patternParts.length - (patternParts.filter(p => p === "*").length)) * STRUCTURE_BONUS;

    return specificity;
}

/**
 * Register a new event type behavior
 * Allows runtime extension of event behaviors
 */
export function registerEventBehavior(eventType: string, behavior: EventBehavior): void {
    EventRegistry[eventType] = behavior;
}

/**
 * Register an emergent pattern for dynamic safety behavior
 * Allows agents to register new patterns at runtime
 */
export function registerEmergentPattern(
    pattern: string,
    behavior: Partial<EventBehavior>,
): void {
    const defaultBehavior = getEventBehavior(pattern);
    EventRegistry[pattern] = {
        ...defaultBehavior,
        ...behavior,
    };
}

/**
 * Get all registered event types
 * Useful for agent discovery and documentation
 */
export function getRegisteredEventTypes(): string[] {
    return Object.keys(EventRegistry);
}

/**
 * Configure approval requirements for an event type
 * This allows runtime configuration of which events need approval
 */
export function configureEventApproval(
    eventType: string,
    approvalConfig: {
        preset?: keyof typeof ApprovalPresets;
        requireUserApproval?: boolean;
        requireSafetyConsensus?: boolean;
        customBarrierConfig?: EventBehavior["barrierConfig"];
    },
): void {
    const existingBehavior = getEventBehavior(eventType);

    let newBehavior: EventBehavior;

    if (approvalConfig.preset) {
        // Use a preset configuration
        newBehavior = {
            ...existingBehavior,
            ...ApprovalPresets[approvalConfig.preset],
        };
    } else if (approvalConfig.requireUserApproval) {
        // Simple user approval
        newBehavior = {
            ...existingBehavior,
            mode: EventMode.APPROVAL,
            interceptable: true,
            barrierConfig: {
                quorum: 1,
                timeoutMs: 30000,
                timeoutAction: "block",
                blockOnFirst: false,
            },
        };
    } else if (approvalConfig.requireSafetyConsensus) {
        // Safety consensus
        newBehavior = {
            ...existingBehavior,
            mode: EventMode.CONSENSUS,
            interceptable: true,
            barrierConfig: {
                quorum: "all",
                timeoutMs: 5000,
                timeoutAction: "block",
                blockOnFirst: true,
            },
        };
    } else if (approvalConfig.customBarrierConfig) {
        // Custom configuration
        newBehavior = {
            ...existingBehavior,
            mode: EventMode.APPROVAL,
            interceptable: true,
            barrierConfig: approvalConfig.customBarrierConfig,
        };
    } else {
        // Remove approval requirements
        newBehavior = {
            ...existingBehavior,
            mode: EventMode.PASSIVE,
            barrierConfig: undefined,
        };
    }

    registerEventBehavior(eventType, newBehavior);
}

/**
 * Create pattern-based approval rules
 * Allows configuring approvals for multiple events at once
 */
export function configurePatternApproval(
    pattern: string,
    approvalConfig: Parameters<typeof configureEventApproval>[1],
): void {
    // Register the pattern with wildcard support
    const patternBehavior = approvalConfig.preset
        ? { ...ApprovalPresets[approvalConfig.preset] }
        : {
            mode: EventMode.APPROVAL,
            interceptable: true,
            barrierConfig: approvalConfig.customBarrierConfig || {
                quorum: 1,
                timeoutMs: 30000,
                timeoutAction: "block" as const,
            },
        };

    registerEventBehavior(pattern, {
        ...patternBehavior,
        defaultPriority: "high",
    });
}

/**
 * Get events that require approval
 */
export function getApprovalRequiredEvents(): string[] {
    return Object.entries(EventRegistry)
        .filter(([_, behavior]) =>
            behavior.mode === EventMode.APPROVAL ||
            behavior.mode === EventMode.CONSENSUS,
        )
        .map(([eventType]) => eventType);
}

/**
 * Check if an event type requires approval
 */
export function requiresApproval(eventType: string): boolean {
    const behavior = getEventBehavior(eventType);
    return behavior.mode === EventMode.APPROVAL ||
        behavior.mode === EventMode.CONSENSUS;
}

/**
 * Validate if an event type is registered
 */
export function isEventTypeRegistered(eventType: string): boolean {
    // Empty event type is not valid
    if (!eventType) {
        return false;
    }
    // Any non-empty event type is considered "registered" because getEventBehavior 
    // always returns a behavior (either from registry or default)
    return true;
}

/**
 * Validate the event registry against socketEvents.ts
 * Ensures all defined events are covered by the registry
 */
export function validateEventRegistry(): { valid: boolean; errors: string[]; coverage: EventCoverage } {
    const errors: string[] = [];

    // Get all events from EventTypes
    const allSocketEvents = Object.values(EventTypes).flatMap(category => Object.values(category));

    const coverage: EventCoverage = {
        totalEvents: allSocketEvents.length,
        coveredEvents: 0,
        uncoveredEvents: [],
        patternMatches: new Map(),
    };

    for (const eventType of allSocketEvents) {
        const behavior = getEventBehavior(eventType);

        if (behavior.tags?.includes("unknown")) {
            coverage.uncoveredEvents.push(eventType);
            errors.push(`Event "${eventType}" not covered by any registry pattern`);
        } else {
            coverage.coveredEvents++;

            // Find which pattern matched
            const matchingPattern = findMatchingPattern(eventType);
            if (matchingPattern) {
                const current = coverage.patternMatches.get(matchingPattern) || [];
                coverage.patternMatches.set(matchingPattern, [...current, eventType]);
            }
        }
    }

    // Check for unused patterns
    const usedPatterns = new Set(coverage.patternMatches.keys());
    const allPatterns = Object.keys(EventRegistry).filter(key => key.includes("*"));
    const unusedPatterns = allPatterns.filter(pattern => !usedPatterns.has(pattern));

    if (unusedPatterns.length > 0) {
        errors.push(`Unused patterns: ${unusedPatterns.join(", ")}`);
    }

    return {
        valid: errors.length === 0,
        errors,
        coverage,
    };
}

/**
 * Find which pattern matches an event type
 */
function findMatchingPattern(eventType: string): string | null {
    // Check exact match first
    if (EventRegistry[eventType]) {
        return eventType;
    }

    // Find the most specific pattern match
    const matchingPatterns = Object.keys(EventRegistry)
        .filter(pattern => pattern.includes("*"))
        .map(pattern => ({
            pattern,
            specificity: calculatePatternSpecificity(pattern, eventType),
        }))
        .filter(item => item.specificity > 0)
        .sort((a, b) => b.specificity - a.specificity);

    return matchingPatterns.length > 0 ? matchingPatterns[0].pattern : null;
}

/**
 * Coverage information for registry validation
 */
export interface EventCoverage {
    totalEvents: number;
    coveredEvents: number;
    uncoveredEvents: string[];
    patternMatches: Map<string, string[]>;
}

/**
 * Print a detailed coverage report
 */
export function printCoverageReport(coverage: EventCoverage): string {
    const lines: string[] = [];

    lines.push("=== Event Registry Coverage Report ===");
    lines.push(`Total Events: ${coverage.totalEvents}`);
    lines.push(`Covered Events: ${coverage.coveredEvents}`);
    lines.push(`Coverage: ${((coverage.coveredEvents / coverage.totalEvents) * 100).toFixed(1)}%`);

    if (coverage.uncoveredEvents.length > 0) {
        lines.push("\nUncovered Events:");
        coverage.uncoveredEvents.forEach(event => lines.push(`  - ${event}`));
    }

    lines.push("\nPattern Usage:");
    Array.from(coverage.patternMatches.entries()).forEach(([pattern, events]) => {
        lines.push(`  ${pattern} (${events.length} events):`);
        events.forEach(event => lines.push(`    - ${event}`));
    });

    return lines.join("\n");
}

/**
 * Runtime validation for event usage patterns
 * Tracks whether events are properly checked for progression
 * 
 * IMPORTANT: All events should check progression because security policies
 * can dynamically override any event's default behavior
 */
export class EventUsageValidator {
    private static pendingEvents = new Map<string, { eventType: string; timestamp: number; wasBlocking: boolean }>();
    private static checkedEvents = new Set<string>();
    private static violations: string[] = [];
    private static warningsEnabled = process.env.EVENT_VALIDATION_WARNINGS !== "false";

    /**
     * Track when any event is emitted
     */
    static trackEmission(eventType: string, eventId: string, wasBlocking = false): void {
        // In the new pattern, ALL events should be checked, not just blocking ones
        this.pendingEvents.set(eventId, {
            eventType,
            timestamp: Date.now(),
            wasBlocking,
        });
    }

    /**
     * Mark that progression was checked for an event
     */
    static markProgressionChecked(eventId: string): void {
        if (this.pendingEvents.has(eventId)) {
            this.checkedEvents.add(eventId);
            this.pendingEvents.delete(eventId);
        }
    }

    /**
     * Check for violations (events without progression checks)
     * Should be called periodically or at checkpoint
     */
    static checkViolations(maxAgeMs = SECONDS_5_MS): string[] {
        const now = Date.now();
        const violations: string[] = [];

        for (const [eventId, info] of this.pendingEvents.entries()) {
            if (now - info.timestamp >= maxAgeMs) {
                const behavior = getEventBehavior(info.eventType);

                // Different severity based on event mode
                if (info.wasBlocking || behavior.mode === EventMode.APPROVAL || behavior.mode === EventMode.CONSENSUS) {
                    // Critical violation - blocking event not checked
                    const violation = `CRITICAL: Event ${info.eventType} (ID: ${eventId}) was emitted in ${behavior.mode} mode ` +
                        `but progression was NOT checked within ${maxAgeMs}ms. This could bypass security policies!`;
                    violations.push(violation);
                    this.violations.push(violation);
                } else if (this.warningsEnabled) {
                    // Warning - non-blocking event not checked (could be dynamically overridden)
                    const violation = `WARNING: Event ${info.eventType} (ID: ${eventId}) was emitted without checking progression. ` +
                        "Security policies could dynamically block this event!";
                    violations.push(violation);
                    this.violations.push(violation);
                }

                this.pendingEvents.delete(eventId);
            }
        }

        return violations;
    }

    /**
     * Get all recorded violations
     */
    static getViolations(): string[] {
        return [...this.violations];
    }

    /**
     * Clear all tracking data
     */
    static reset(): void {
        this.pendingEvents.clear();
        this.checkedEvents.clear();
        this.violations = [];
    }

    /**
     * Get current stats
     */
    static getStats(): {
        pendingEvents: number;
        checkedEvents: number;
        violations: number;
        criticalViolations: number;
        warnings: number;
    } {
        const criticalViolations = this.violations.filter(v => v.startsWith("CRITICAL:")).length;
        const warnings = this.violations.filter(v => v.startsWith("WARNING:")).length;

        return {
            pendingEvents: this.pendingEvents.size,
            checkedEvents: this.checkedEvents.size,
            violations: this.violations.length,
            criticalViolations,
            warnings,
        };
    }

    /**
     * Enable or disable warnings for non-blocking events
     */
    static setWarningsEnabled(enabled: boolean): void {
        this.warningsEnabled = enabled;
    }
}

/**
 * Decorator to validate event usage in development
 * Can be applied to methods that emit events
 */
export function validateEventUsage(target: any, propertyKey: string, descriptor: PropertyDescriptor): void {
    if (process.env.NODE_ENV === "production") {
        return; // Don't add overhead in production
    }

    const originalMethod = descriptor.value;

    descriptor.value = async function decoratedEventUsageValidator(...args: any[]) {
        // Use shorter timeout for development/testing to catch violations quickly
        const violationTimeout = process.env.NODE_ENV === "test" ? 0 : SECONDS_5_MS;
        
        // Check for violations before method execution
        const preViolations = EventUsageValidator.checkViolations(violationTimeout);
        if (preViolations.length > 0) {
            logger.warn("Event usage violations detected before method execution", {
                method: `${target.constructor.name}.${propertyKey}`,
                violations: preViolations,
            });
        }

        // Execute original method
        const result = await originalMethod.apply(this, args);

        // Check for violations after method execution
        const postViolations = EventUsageValidator.checkViolations(violationTimeout);
        if (postViolations.length > 0) {
            logger.error("Event usage violations detected after method execution", {
                method: `${target.constructor.name}.${propertyKey}`,
                violations: postViolations,
            });
        }

        return result;
    };
}
