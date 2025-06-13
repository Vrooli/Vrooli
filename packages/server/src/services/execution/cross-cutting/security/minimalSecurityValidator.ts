/**
 * Minimal Security Validator - Basic permission checks only
 * 
 * This component provides ONLY essential security infrastructure:
 * - Simple permission checking
 * - Authentication verification
 * - Event emission for security agents
 * 
 * All complex security intelligence (threat detection, risk scoring,
 * policy enforcement, etc.) emerges from security agents analyzing
 * the events emitted by this validator.
 * 
 * IMPORTANT: Do NOT add logic here. If you're tempted to add
 * threat detection, risk calculation, or policy rules, create
 * a security agent routine instead.
 */

import { type Logger } from "winston";
import { ExecutionEventEmitter } from "../monitoring/ExecutionEventEmitter.js";

export interface SecurityCheckEvent {
    type: "permission_check" | "authentication_check" | "resource_access";
    userId?: string;
    teamId?: string;
    resource: string;
    action: string;
    permissions: string[];
    result: boolean;
    metadata?: Record<string, unknown>;
    timestamp: Date;
}

/**
 * Minimal Security Validator
 * 
 * Provides only the most basic security checks and emits events
 * for security agents to analyze. This is the ENTIRE security
 * infrastructure - everything else emerges from agents.
 */
export class MinimalSecurityValidator {
    private readonly eventEmitter: ExecutionEventEmitter;
    
    constructor(
        private readonly logger: Logger,
        eventEmitter: ExecutionEventEmitter
    ) {
        this.eventEmitter = eventEmitter;
    }
    
    /**
     * Check if user has required permission
     * Just a simple string match - no complex logic
     */
    async checkPermission(
        userId: string | undefined,
        resource: string,
        action: string,
        permissions: string[]
    ): Promise<boolean> {
        const requiredPermission = `${resource}:${action}`;
        const hasPermission = permissions.includes(requiredPermission);
        
        // Emit event for security agents to analyze
        await this.emitSecurityEvent({
            type: "permission_check",
            userId,
            resource,
            action,
            permissions,
            result: hasPermission,
            timestamp: new Date()
        });
        
        this.logger.debug("[MinimalSecurityValidator] Permission check", {
            userId,
            resource,
            action,
            required: requiredPermission,
            hasPermission
        });
        
        return hasPermission;
    }
    
    /**
     * Check if user is authenticated
     * Just checks if userId exists - no complex logic
     */
    async checkAuthentication(userId: string | undefined): Promise<boolean> {
        const isAuthenticated = !!userId;
        
        // Emit event for security agents
        await this.emitSecurityEvent({
            type: "authentication_check",
            userId,
            resource: "system",
            action: "authenticate",
            permissions: [],
            result: isAuthenticated,
            timestamp: new Date()
        });
        
        return isAuthenticated;
    }
    
    /**
     * Check resource access (combines authentication and permission)
     * Still just basic checks - no risk assessment or threat detection
     */
    async checkResourceAccess(
        userId: string | undefined,
        teamId: string | undefined,
        resource: string,
        action: string,
        permissions: string[]
    ): Promise<boolean> {
        // Must be authenticated
        const isAuthenticated = await this.checkAuthentication(userId);
        if (!isAuthenticated) {
            return false;
        }
        
        // Must have permission
        const hasPermission = await this.checkPermission(
            userId,
            resource,
            action,
            permissions
        );
        
        // Emit combined access event
        await this.emitSecurityEvent({
            type: "resource_access",
            userId,
            teamId,
            resource,
            action,
            permissions,
            result: hasPermission,
            metadata: {
                authenticated: isAuthenticated,
                permitted: hasPermission
            },
            timestamp: new Date()
        });
        
        return hasPermission;
    }
    
    /**
     * Emit security event for agents to analyze
     */
    private async emitSecurityEvent(event: SecurityCheckEvent): Promise<void> {
        await this.eventEmitter.emitExecutionEvent({
            executionId: `security-${Date.now()}`,
            event: "completed",
            tier: "cross-cutting" as any,
            component: "MinimalSecurityValidator",
            data: {
                securityEvent: event
            }
        });
        
        // Also emit to security-specific channel for focused agents
        await this.eventEmitter.emitMetric({
            tier: "cross-cutting",
            component: "MinimalSecurityValidator", 
            metricType: "safety",
            name: `security.${event.type}`,
            value: event.result ? 1 : 0,
            tags: {
                resource: event.resource,
                action: event.action,
                userId: event.userId || "anonymous"
            }
        });
    }
}

/**
 * Example of what security agents would do with these events:
 * 
 * 1. Threat Detection Agent:
 *    - Subscribe to failed permission_check events
 *    - Analyze patterns of failed attempts
 *    - Emit threat alerts when patterns detected
 * 
 * 2. Compliance Agent:
 *    - Subscribe to resource_access events
 *    - Check against compliance rules
 *    - Generate audit reports
 * 
 * 3. Anomaly Detection Agent:
 *    - Subscribe to all security events
 *    - Build normal behavior profiles
 *    - Alert on deviations
 * 
 * ALL of this intelligence emerges from agents, NOT from this validator!
 */