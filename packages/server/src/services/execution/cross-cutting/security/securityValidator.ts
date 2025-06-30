/**
 * Security Validator - Basic permission checks for execution operations
 * 
 * This component provides simple permission validation.
 * All complex security intelligence emerges from security agents
 * analyzing the events emitted by this validator.
 */

import type {
    EnhancedSecurityContext,
    Permission,
} from "@vrooli/shared";
import { logger } from "../../../../events/logger.js";
import { EventTypes } from "../../../events/index.js";
import { type IEventBus } from "../../../events/types.js";

/**
 * Security Validator
 * 
 * Performs basic permission checks and emits events for security agents.
 * Does NOT implement threat detection, risk scoring, or complex policies.
 * Those capabilities emerge from agent analysis.
 */
export class SecurityValidator {
    private readonly eventBus: IEventBus;

    constructor(eventBus: IEventBus) {
        this.eventBus = eventBus;
    }

    /**
     * Validates if a security context has required permissions
     */
    async validatePermissions(
        context: EnhancedSecurityContext,
        requiredPermissions: Permission[],
        operation: string,
    ): Promise<boolean> {
        logger.debug("[SecurityValidator] Validating permissions", {
            userId: context.userId,
            operation,
            requiredPermissions,
            contextPermissions: context.permissions,
        });

        // Simple permission check
        const hasPermissions = requiredPermissions.every(required =>
            context.permissions.includes(required),
        );

        // Emit validation event for security agents to analyze
        await this.eventBus.publish("security.events", {
            type: "PERMISSION_VALIDATION",
            timestamp: new Date(),
            metadata: {
                userId: context.userId,
                teamId: context.teamId,
                operation,
                requiredPermissions,
                contextPermissions: context.permissions,
                result: hasPermissions,
                origin: context.origin,
                tier: context.tier,
            },
        });

        return hasPermissions;
    }

    /**
     * Checks if user is authenticated
     */
    isAuthenticated(context: EnhancedSecurityContext): boolean {
        return !!context.userId && context.isAuthenticated;
    }

    /**
     * Checks if user belongs to a team
     */
    belongsToTeam(context: EnhancedSecurityContext, teamId: string): boolean {
        return context.teamId === teamId;
    }

    /**
     * Creates a security context for a sub-operation
     */
    createSubContext(
        parentContext: EnhancedSecurityContext,
        operation: string,
    ): EnhancedSecurityContext {
        // Inherit permissions from parent
        return {
            ...parentContext,
            parentContextId: parentContext.contextId,
            operation,
        };
    }

    /**
     * Records security event for audit
     */
    async recordSecurityEvent(
        context: EnhancedSecurityContext,
        eventType: string,
        details: any,
    ): Promise<void> {
        await this.eventBus.publish(EventTypes.SECURITY_AUDIT, {
            type: "SECURITY_AUDIT",
            timestamp: new Date(),
            metadata: {
                userId: context.userId,
                teamId: context.teamId,
                eventType,
                details,
                contextId: context.contextId,
            },
        });
    }
}
