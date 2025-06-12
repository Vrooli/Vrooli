/**
 * Security Context Manager
 * Manages SecurityContext creation, validation, tier-specific propagation,
 * permission inheritance, and constraint application across the execution stack
 */

import type {
    EnhancedSecurityContext,
    SecurityContext,
    SecurityLevel,
    Permission,
    SecurityOrigin,
    TrustLevel,
    OriginType,
    ThreatLevel,
    AccessPolicy,
    GuardRail,
    PermissionScope,
    TimeWindow,
} from "@vrooli/shared";
import { logger } from "../../../../events/logger.js";
import { TelemetryShimAdapter as TelemetryShim } from "../../monitoring/adapters/TelemetryShimAdapter.js";
import { RedisEventBus } from "../events/eventBus.js";
import { generatePK } from "@vrooli/shared";

/**
 * Security context management configuration
 */
export interface SecurityContextConfig {
    enablePermissionInheritance: boolean;
    enableDynamicElevation: boolean;
    enableContextPropagation: boolean;
    enableAuditTrail: boolean;
    defaultTrustLevel: TrustLevel;
    maxContextDepth: number;
    contextTimeoutMs: number;
    cacheContexts: boolean;
    maxCacheSize: number;
}

/**
 * Default security context configuration
 */
const DEFAULT_CONTEXT_CONFIG: SecurityContextConfig = {
    enablePermissionInheritance: true,
    enableDynamicElevation: false,
    enableContextPropagation: true,
    enableAuditTrail: true,
    defaultTrustLevel: TrustLevel.LOW,
    maxContextDepth: 5,
    contextTimeoutMs: 300000, // 5 minutes
    cacheContexts: true,
    maxCacheSize: 1000,
};

/**
 * Context cache entry
 */
interface ContextCacheEntry {
    context: EnhancedSecurityContext;
    createdAt: Date;
    accessCount: number;
    lastAccessed: Date;
}

/**
 * Permission template for common operation patterns
 */
interface PermissionTemplate {
    id: string;
    name: string;
    description: string;
    permissions: Permission[];
    applicableTiers: number[];
    securityLevelRequired: SecurityLevel;
}

/**
 * Security Context Manager
 * 
 * Manages the lifecycle and propagation of security contexts across tiers,
 * handles permission inheritance, and ensures security constraints are
 * properly applied at each level of the execution stack.
 */
export class SecurityContextManager {
    private readonly telemetry: TelemetryShim;
    private readonly eventBus: RedisEventBus;
    private readonly config: SecurityContextConfig;
    
    // Context cache and templates
    private readonly contextCache = new Map<string, ContextCacheEntry>();
    private readonly permissionTemplates = new Map<string, PermissionTemplate>();
    private readonly tierConstraints = new Map<number, GuardRail[]>();
    
    // Statistics
    private contextCreationCount = 0;
    private contextPropagationCount = 0;
    private permissionElevationCount = 0;
    private constraintViolationCount = 0;

    constructor(
        telemetry: TelemetryShim,
        eventBus: RedisEventBus,
        config: Partial<SecurityContextConfig> = {},
    ) {
        this.telemetry = telemetry;
        this.eventBus = eventBus;
        this.config = { ...DEFAULT_CONTEXT_CONFIG, ...config };
        
        this.initializePermissionTemplates();
        this.initializeTierConstraints();
        
        // Start cache cleanup timer
        if (this.config.cacheContexts) {
            this.startCacheCleanup();
        }
    }

    /**
     * Create a new security context for an operation
     */
    async createSecurityContext(
        tier: 1 | 2 | 3,
        requestId: string,
        origin: SecurityOrigin,
        options: {
            sessionId?: string;
            parentContext?: EnhancedSecurityContext;
            requiredLevel?: SecurityLevel;
            customPermissions?: Permission[];
            templateId?: string;
        } = {},
    ): Promise<EnhancedSecurityContext> {
        const startTime = performance.now();
        this.contextCreationCount++;

        try {
            // Check cache first
            const cacheKey = this.generateCacheKey(tier, requestId, origin, options);
            if (this.config.cacheContexts && this.contextCache.has(cacheKey)) {
                const cached = this.contextCache.get(cacheKey)!;
                cached.accessCount++;
                cached.lastAccessed = new Date();
                
                await this.emitContextEvent("context_cache_hit", {
                    requestId,
                    tier,
                    cacheKey,
                });
                
                return { ...cached.context, requestId }; // Update request ID
            }

            // Create base security context
            const context: EnhancedSecurityContext = {
                level: options.requiredLevel || this.determineSecurityLevel(origin, tier),
                policies: [],
                guardRails: [],
                barriers: [],
                compliance: [],
                audit: this.config.enableAuditTrail,
                tier,
                requestId,
                sessionId: options.sessionId,
                origin,
                permissions: options.customPermissions || [],
                riskScore: this.calculateInitialRiskScore(origin, tier),
                threatLevel: ThreatLevel.NONE,
                validationResults: [],
            };

            // Apply permission template if specified
            if (options.templateId) {
                await this.applyPermissionTemplate(context, options.templateId);
            }

            // Inherit permissions from parent context
            if (options.parentContext && this.config.enablePermissionInheritance) {
                await this.inheritPermissions(context, options.parentContext);
            }

            // Apply tier-specific constraints
            await this.applyTierConstraints(context);

            // Set initial threat level
            context.threatLevel = this.determineThreatLevel(context.riskScore);

            // Cache the context
            if (this.config.cacheContexts) {
                this.cacheContext(cacheKey, context);
            }

            // Emit context creation event
            await this.emitContextEvent("context_created", {
                requestId,
                tier,
                securityLevel: context.level,
                trustLevel: context.origin.trustLevel,
                permissionCount: context.permissions.length,
                riskScore: context.riskScore,
            });

            const duration = performance.now() - startTime;
            await this.telemetry.emitExecutionTiming(
                "SecurityContextManager",
                "createSecurityContext",
                new Date(startTime),
                new Date(),
                true,
                {
                    tier,
                    securityLevel: context.level,
                    permissionCount: context.permissions.length,
                },
            );

            return context;
        } catch (error) {
            logger.error("Security context creation error", {
                requestId,
                tier,
                error,
            });

            await this.telemetry.emitError(
                error,
                "SecurityContextManager",
                "critical",
                { requestId, tier },
            );

            throw error;
        }
    }

    /**
     * Propagate security context to a different tier
     */
    async propagateContext(
        sourceContext: EnhancedSecurityContext,
        targetTier: 1 | 2 | 3,
        newRequestId?: string,
        options: {
            restrictPermissions?: boolean;
            elevateLevel?: boolean;
            inheritValidations?: boolean;
        } = {},
    ): Promise<EnhancedSecurityContext> {
        const startTime = performance.now();
        this.contextPropagationCount++;

        try {
            if (!this.config.enableContextPropagation) {
                throw new Error("Context propagation is disabled");
            }

            // Create propagated context
            const propagatedContext: EnhancedSecurityContext = {
                ...sourceContext,
                tier: targetTier,
                requestId: newRequestId || sourceContext.requestId,
                validationResults: options.inheritValidations ? sourceContext.validationResults : [],
            };

            // Apply tier-specific permission filtering
            if (options.restrictPermissions !== false) {
                propagatedContext.permissions = this.filterPermissionsForTier(
                    sourceContext.permissions,
                    targetTier,
                );
            }

            // Apply security level elevation if needed
            if (options.elevateLevel && this.config.enableDynamicElevation) {
                propagatedContext.level = this.elevateSecurityLevel(
                    sourceContext.level,
                    targetTier,
                );
            }

            // Adjust risk score for target tier
            propagatedContext.riskScore = this.calculatePropagatedRiskScore(
                sourceContext.riskScore,
                sourceContext.tier,
                targetTier,
            );

            // Update threat level
            propagatedContext.threatLevel = this.determineThreatLevel(propagatedContext.riskScore);

            // Apply target tier constraints
            await this.applyTierConstraints(propagatedContext);

            // Emit propagation event
            await this.emitContextEvent("context_propagated", {
                sourceRequestId: sourceContext.requestId,
                targetRequestId: propagatedContext.requestId,
                sourceTier: sourceContext.tier,
                targetTier,
                securityLevel: propagatedContext.level,
                riskScoreChange: propagatedContext.riskScore - sourceContext.riskScore,
            });

            const duration = performance.now() - startTime;
            await this.telemetry.emitExecutionTiming(
                "SecurityContextManager",
                "propagateContext",
                new Date(startTime),
                new Date(),
                true,
                {
                    sourceTier: sourceContext.tier,
                    targetTier,
                    permissionChange: propagatedContext.permissions.length - sourceContext.permissions.length,
                },
            );

            return propagatedContext;
        } catch (error) {
            logger.error("Security context propagation error", {
                sourceRequestId: sourceContext.requestId,
                sourceTier: sourceContext.tier,
                targetTier,
                error,
            });

            await this.telemetry.emitError(
                error,
                "SecurityContextManager",
                "high",
                { sourceRequestId: sourceContext.requestId },
            );

            throw error;
        }
    }

    /**
     * Validate security context for an operation
     */
    async validateContext(
        context: EnhancedSecurityContext,
        operation: {
            action: string;
            resource: string;
            requiredLevel?: SecurityLevel;
            requiredPermissions?: string[];
        },
    ): Promise<{
        valid: boolean;
        violations: string[];
        warnings: string[];
        adjustedContext?: EnhancedSecurityContext;
    }> {
        const violations: string[] = [];
        const warnings: string[] = [];

        // Check security level requirements
        if (operation.requiredLevel) {
            if (this.compareSecurityLevels(context.level, operation.requiredLevel) < 0) {
                violations.push(
                    `Operation requires ${operation.requiredLevel} but context has ${context.level}`,
                );
            }
        }

        // Check required permissions
        if (operation.requiredPermissions) {
            const missingPermissions = operation.requiredPermissions.filter(
                required => !this.hasPermission(context, required, operation.resource),
            );
            
            if (missingPermissions.length > 0) {
                violations.push(
                    `Missing required permissions: ${missingPermissions.join(", ")}`,
                );
            }
        }

        // Check tier-specific constraints
        const tierViolations = await this.validateTierConstraints(context, operation);
        violations.push(...tierViolations);

        // Check threat level
        if (context.threatLevel === ThreatLevel.CRITICAL) {
            violations.push("Critical threat level - operation blocked");
        } else if (context.threatLevel === ThreatLevel.HIGH) {
            warnings.push("High threat level detected");
        }

        // Check context expiration
        if (this.isContextExpired(context)) {
            violations.push("Security context has expired");
        }

        return {
            valid: violations.length === 0,
            violations,
            warnings,
        };
    }

    /**
     * Elevate permissions for a specific operation
     */
    async elevatePermissions(
        context: EnhancedSecurityContext,
        elevationRequest: {
            permissions: Permission[];
            justification: string;
            duration?: number;
            approver?: string;
        },
    ): Promise<EnhancedSecurityContext> {
        if (!this.config.enableDynamicElevation) {
            throw new Error("Dynamic permission elevation is disabled");
        }

        this.permissionElevationCount++;

        // Create elevated context
        const elevatedContext: EnhancedSecurityContext = {
            ...context,
            permissions: [
                ...context.permissions,
                ...elevationRequest.permissions.map(permission => ({
                    ...permission,
                    temporary: true,
                    expiresAt: elevationRequest.duration
                        ? new Date(Date.now() + elevationRequest.duration)
                        : new Date(Date.now() + 3600000), // 1 hour default
                })),
            ],
        };

        // Emit elevation event
        await this.emitContextEvent("permissions_elevated", {
            requestId: context.requestId,
            tier: context.tier,
            elevatedPermissions: elevationRequest.permissions.length,
            justification: elevationRequest.justification,
            approver: elevationRequest.approver,
        });

        return elevatedContext;
    }

    /**
     * Helper methods
     */
    private initializePermissionTemplates(): void {
        // Basic read-only template
        this.permissionTemplates.set("readonly", {
            id: "readonly",
            name: "Read-Only Access",
            description: "Basic read-only permissions for all tiers",
            permissions: [
                {
                    action: "read",
                    resource: "*",
                    scope: { tier: 1 },
                },
                {
                    action: "read",
                    resource: "*",
                    scope: { tier: 2 },
                },
                {
                    action: "read",
                    resource: "*",
                    scope: { tier: 3 },
                },
            ],
            applicableTiers: [1, 2, 3],
            securityLevelRequired: SecurityLevel.PRIVATE,
        });

        // Standard execution template
        this.permissionTemplates.set("execution", {
            id: "execution",
            name: "Standard Execution",
            description: "Permissions for standard routine execution",
            permissions: [
                {
                    action: "read",
                    resource: "*",
                    scope: { tier: 2 },
                },
                {
                    action: "execute",
                    resource: "routine.*",
                    scope: { tier: 2 },
                },
                {
                    action: "write",
                    resource: "state.*",
                    scope: { tier: 2, component: "state-manager" },
                },
                {
                    action: "*",
                    resource: "tools.*",
                    scope: { tier: 3 },
                },
            ],
            applicableTiers: [2, 3],
            securityLevelRequired: SecurityLevel.PRIVATE,
        });

        // Administrative template
        this.permissionTemplates.set("admin", {
            id: "admin",
            name: "Administrative Access",
            description: "Full administrative permissions",
            permissions: [
                {
                    action: "*",
                    resource: "*",
                    scope: { tier: 1 },
                },
                {
                    action: "*",
                    resource: "*",
                    scope: { tier: 2 },
                },
                {
                    action: "*",
                    resource: "*",
                    scope: { tier: 3 },
                },
            ],
            applicableTiers: [1, 2, 3],
            securityLevelRequired: SecurityLevel.CONFIDENTIAL,
        });
    }

    private initializeTierConstraints(): void {
        // Tier 1 constraints (Coordination)
        this.tierConstraints.set(1, [
            {
                id: "tier1-resource-limit",
                type: "RESOURCE_LIMIT" as any,
                name: "Tier 1 Resource Limits",
                description: "Resource consumption limits for coordination tier",
                enabled: true,
                config: {
                    severity: "error" as const,
                    mode: "block" as const,
                    rules: [
                        {
                            id: "memory-limit",
                            condition: "memory > 1GB",
                            message: "Memory usage exceeds tier 1 limits",
                        },
                        {
                            id: "time-limit",
                            condition: "duration > 30min",
                            message: "Execution time exceeds tier 1 limits",
                        },
                    ],
                },
                actions: [{ type: "block", config: {} }],
            },
        ]);

        // Tier 2 constraints (Process)
        this.tierConstraints.set(2, [
            {
                id: "tier2-access-control",
                type: "ACCESS_CONTROL" as any,
                name: "Tier 2 Access Control",
                description: "Access control for process tier",
                enabled: true,
                config: {
                    severity: "warning" as const,
                    mode: "monitor" as const,
                    rules: [
                        {
                            id: "routine-access",
                            condition: "action = 'execute' AND resource LIKE 'routine.*'",
                            message: "Routine execution monitored",
                        },
                    ],
                },
                actions: [{ type: "log", config: {} }],
            },
        ]);

        // Tier 3 constraints (Execution)
        this.tierConstraints.set(3, [
            {
                id: "tier3-tool-safety",
                type: "CONTENT_FILTER" as any,
                name: "Tier 3 Tool Safety",
                description: "Safety constraints for tool execution",
                enabled: true,
                config: {
                    severity: "critical" as const,
                    mode: "block" as const,
                    rules: [
                        {
                            id: "dangerous-tool",
                            condition: "tool IN ('shell', 'file-delete', 'network-access')",
                            message: "Dangerous tool usage requires elevated permissions",
                        },
                    ],
                },
                actions: [{ type: "block", config: {} }],
            },
        ]);
    }

    private determineSecurityLevel(origin: SecurityOrigin, tier: number): SecurityLevel {
        // Determine security level based on origin trust and tier
        const baseLevels = {
            [TrustLevel.UNTRUSTED]: SecurityLevel.PUBLIC,
            [TrustLevel.LOW]: SecurityLevel.PRIVATE,
            [TrustLevel.MEDIUM]: SecurityLevel.PRIVATE,
            [TrustLevel.HIGH]: SecurityLevel.RESTRICTED,
            [TrustLevel.SYSTEM]: SecurityLevel.CONFIDENTIAL,
        };

        let level = baseLevels[origin.trustLevel];

        // Elevate for higher tiers
        if (tier === 1 && level === SecurityLevel.PRIVATE) {
            level = SecurityLevel.RESTRICTED;
        }

        return level;
    }

    private calculateInitialRiskScore(origin: SecurityOrigin, tier: number): number {
        let riskScore = 0;

        // Base risk from trust level
        const trustRisk = {
            [TrustLevel.UNTRUSTED]: 0.9,
            [TrustLevel.LOW]: 0.6,
            [TrustLevel.MEDIUM]: 0.4,
            [TrustLevel.HIGH]: 0.2,
            [TrustLevel.SYSTEM]: 0.1,
        };
        
        riskScore += trustRisk[origin.trustLevel];

        // Origin type risk
        const originRisk = {
            [OriginType.SYSTEM]: 0.1,
            [OriginType.AGENT]: 0.3,
            [OriginType.USER]: 0.5,
            [OriginType.EXTERNAL_API]: 0.7,
            [OriginType.TIER]: 0.2,
        };
        
        riskScore += originRisk[origin.type] * 0.5;

        // Tier multiplier
        const tierMultiplier = {
            1: 1.2, // Coordination has higher risk
            2: 1.0,
            3: 0.8, // Execution is more constrained
        };
        
        riskScore *= tierMultiplier[tier];

        return Math.max(0, Math.min(riskScore, 1));
    }

    private determineThreatLevel(riskScore: number): ThreatLevel {
        if (riskScore >= 0.8) return ThreatLevel.CRITICAL;
        if (riskScore >= 0.6) return ThreatLevel.HIGH;
        if (riskScore >= 0.4) return ThreatLevel.MEDIUM;
        if (riskScore >= 0.2) return ThreatLevel.LOW;
        return ThreatLevel.NONE;
    }

    private async applyPermissionTemplate(
        context: EnhancedSecurityContext,
        templateId: string,
    ): Promise<void> {
        const template = this.permissionTemplates.get(templateId);
        if (!template) {
            throw new Error(`Permission template not found: ${templateId}`);
        }

        // Check if template is applicable to this tier
        if (!template.applicableTiers.includes(context.tier)) {
            throw new Error(
                `Template ${templateId} not applicable to tier ${context.tier}`,
            );
        }

        // Check security level requirement
        if (this.compareSecurityLevels(context.level, template.securityLevelRequired) < 0) {
            throw new Error(
                `Template ${templateId} requires ${template.securityLevelRequired} but context has ${context.level}`,
            );
        }

        // Apply template permissions
        context.permissions.push(...template.permissions);
    }

    private async inheritPermissions(
        childContext: EnhancedSecurityContext,
        parentContext: EnhancedSecurityContext,
    ): Promise<void> {
        // Filter parent permissions for child tier
        const inheritablePermissions = parentContext.permissions.filter(permission => {
            // Don't inherit temporary permissions
            if (permission.temporary && permission.expiresAt && permission.expiresAt < new Date()) {
                return false;
            }

            // Check if permission is applicable to child tier
            if (permission.scope?.tier && permission.scope.tier !== childContext.tier) {
                return false;
            }

            return true;
        });

        childContext.permissions.push(...inheritablePermissions);
    }

    private async applyTierConstraints(context: EnhancedSecurityContext): Promise<void> {
        const constraints = this.tierConstraints.get(context.tier) || [];
        
        for (const constraint of constraints) {
            if (constraint.enabled) {
                context.guardRails.push(constraint.id);
            }
        }
    }

    private filterPermissionsForTier(permissions: Permission[], tier: number): Permission[] {
        return permissions.filter(permission => {
            // Filter based on tier scope
            if (permission.scope?.tier && permission.scope.tier !== tier) {
                return false;
            }

            // Check expiration
            if (permission.expiresAt && permission.expiresAt < new Date()) {
                return false;
            }

            return true;
        });
    }

    private elevateSecurityLevel(currentLevel: SecurityLevel, tier: number): SecurityLevel {
        const levels = [
            SecurityLevel.PUBLIC,
            SecurityLevel.PRIVATE,
            SecurityLevel.RESTRICTED,
            SecurityLevel.CONFIDENTIAL,
        ];

        const currentIndex = levels.indexOf(currentLevel);
        const maxElevation = tier === 1 ? 1 : 0; // Only tier 1 can elevate by one level
        
        return levels[Math.min(currentIndex + maxElevation, levels.length - 1)];
    }

    private calculatePropagatedRiskScore(
        sourceRisk: number,
        sourceTier: number,
        targetTier: number,
    ): number {
        // Risk changes based on tier transition
        const tierRiskMultipliers: Record<string, number> = {
            "1->2": 0.9, // Coordination to Process: slight decrease
            "1->3": 0.8, // Coordination to Execution: more decrease
            "2->1": 1.1, // Process to Coordination: slight increase
            "2->3": 0.9, // Process to Execution: slight decrease
            "3->1": 1.2, // Execution to Coordination: increase
            "3->2": 1.1, // Execution to Process: slight increase
        };

        const key = `${sourceTier}->${targetTier}`;
        const multiplier = tierRiskMultipliers[key] || 1.0;
        
        return Math.max(0, Math.min(sourceRisk * multiplier, 1));
    }

    private compareSecurityLevels(level1: SecurityLevel, level2: SecurityLevel): number {
        const levels = [
            SecurityLevel.PUBLIC,
            SecurityLevel.PRIVATE,
            SecurityLevel.RESTRICTED,
            SecurityLevel.CONFIDENTIAL,
        ];
        return levels.indexOf(level1) - levels.indexOf(level2);
    }

    private hasPermission(
        context: EnhancedSecurityContext,
        action: string,
        resource: string,
    ): boolean {
        return context.permissions.some(permission => {
            // Check action match
            if (permission.action !== "*" && permission.action !== action) {
                return false;
            }

            // Check resource match
            if (permission.resource !== "*" && !resource.startsWith(permission.resource)) {
                return false;
            }

            // Check tier scope
            if (permission.scope?.tier && permission.scope.tier !== context.tier) {
                return false;
            }

            // Check expiration
            if (permission.expiresAt && permission.expiresAt < new Date()) {
                return false;
            }

            return true;
        });
    }

    private async validateTierConstraints(
        context: EnhancedSecurityContext,
        operation: { action: string; resource: string },
    ): Promise<string[]> {
        const violations: string[] = [];
        const constraints = this.tierConstraints.get(context.tier) || [];

        for (const constraint of constraints) {
            if (!constraint.enabled) continue;

            // Simplified constraint evaluation
            for (const rule of constraint.config.rules) {
                // In a real implementation, this would evaluate the condition
                // For now, we'll do basic checks
                if (rule.id === "dangerous-tool" && operation.resource.includes("shell")) {
                    violations.push(rule.message);
                    this.constraintViolationCount++;
                }
            }
        }

        return violations;
    }

    private isContextExpired(context: EnhancedSecurityContext): boolean {
        // Check for context timeout (simplified implementation)
        return false; // In real implementation, would check creation time
    }

    private generateCacheKey(
        tier: number,
        requestId: string,
        origin: SecurityOrigin,
        options: any,
    ): string {
        return `${tier}:${origin.type}:${origin.trustLevel}:${options.templateId || 'default'}`;
    }

    private cacheContext(key: string, context: EnhancedSecurityContext): void {
        if (this.contextCache.size >= this.config.maxCacheSize) {
            // Remove oldest entry
            const oldestKey = Array.from(this.contextCache.keys())[0];
            this.contextCache.delete(oldestKey);
        }

        this.contextCache.set(key, {
            context: { ...context },
            createdAt: new Date(),
            accessCount: 1,
            lastAccessed: new Date(),
        });
    }

    private startCacheCleanup(): void {
        setInterval(() => {
            const now = Date.now();
            
            for (const [key, entry] of this.contextCache.entries()) {
                if (now - entry.lastAccessed.getTime() > this.config.contextTimeoutMs) {
                    this.contextCache.delete(key);
                }
            }
        }, 60000); // Clean up every minute
    }

    private async emitContextEvent(
        eventType: string,
        payload: Record<string, unknown>,
    ): Promise<void> {
        await this.eventBus.publish({
            id: generatePK().toString(),
            type: `security.context.${eventType}`,
            timestamp: new Date(),
            source: {
                tier: payload.tier as number || 0,
                component: "SecurityContextManager",
                service: "security",
            },
            correlationId: payload.requestId as string || generatePK().toString(),
            payload,
            metadata: {
                category: "security",
                priority: "normal",
            },
        });
    }

    /**
     * Get security context management statistics
     */
    getStatistics(): {
        contextCreationCount: number;
        contextPropagationCount: number;
        permissionElevationCount: number;
        constraintViolationCount: number;
        cacheSize: number;
        cacheHitRate: number;
    } {
        return {
            contextCreationCount: this.contextCreationCount,
            contextPropagationCount: this.contextPropagationCount,
            permissionElevationCount: this.permissionElevationCount,
            constraintViolationCount: this.constraintViolationCount,
            cacheSize: this.contextCache.size,
            cacheHitRate: 0, // Would be calculated from cache hits vs misses
        };
    }
}