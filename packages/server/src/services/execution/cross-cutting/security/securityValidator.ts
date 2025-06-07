/**
 * Security Validation Engine
 * Implements tier-by-tier permission validation with SecurityContext propagation
 * Enables emergent security intelligence through event emission and pattern learning
 */

import type {
    EnhancedSecurityContext,
    SecurityValidation,
    GuardRail,
    GuardRailViolation,
    AccessPolicy,
    Permission,
    SecurityLevel,
    ThreatLevel,
    SecurityOrigin,
    TrustLevel,
    OriginType,
    ValidationResult,
    ViolationContext,
} from "@vrooli/shared";
import { logger } from "../../../../events/logger.js";
import { TelemetryShim } from "../monitoring/telemetryShim.js";
import { RedisEventBus } from "../events/eventBus.js";
import { v4 as uuidv4 } from "uuid";

/**
 * Security validation configuration
 */
export interface SecurityValidationConfig {
    enableTierValidation: boolean;
    enablePermissionInheritance: boolean;
    enableThreatDetection: boolean;
    enableAuditLogging: boolean;
    maxViolationsBeforeBlock: number;
    riskThreshold: number; // 0-1
    trustLevelThresholds: {
        [TrustLevel.UNTRUSTED]: number;
        [TrustLevel.LOW]: number;
        [TrustLevel.MEDIUM]: number;
        [TrustLevel.HIGH]: number;
        [TrustLevel.SYSTEM]: number;
    };
}

/**
 * Default security validation configuration
 */
const DEFAULT_CONFIG: SecurityValidationConfig = {
    enableTierValidation: true,
    enablePermissionInheritance: true,
    enableThreatDetection: true,
    enableAuditLogging: true,
    maxViolationsBeforeBlock: 5,
    riskThreshold: 0.7,
    trustLevelThresholds: {
        [TrustLevel.UNTRUSTED]: 0.1,
        [TrustLevel.LOW]: 0.3,
        [TrustLevel.MEDIUM]: 0.5,
        [TrustLevel.HIGH]: 0.8,
        [TrustLevel.SYSTEM]: 1.0,
    },
};

/**
 * Security Validation Engine
 * 
 * Core security infrastructure that validates operations across all tiers,
 * enforces permission models, and emits security events for learning.
 */
export class SecurityValidator {
    private readonly telemetry: TelemetryShim;
    private readonly eventBus: RedisEventBus;
    private readonly config: SecurityValidationConfig;
    
    // Caches for performance
    private readonly policyCache = new Map<string, AccessPolicy>();
    private readonly guardRailCache = new Map<string, GuardRail>();
    private readonly violationHistory = new Map<string, GuardRailViolation[]>();
    
    // Statistics
    private validationCount = 0;
    private violationCount = 0;
    private blockedOperations = 0;
    
    constructor(
        telemetry: TelemetryShim,
        eventBus: RedisEventBus,
        config: Partial<SecurityValidationConfig> = {},
    ) {
        this.telemetry = telemetry;
        this.eventBus = eventBus;
        this.config = { ...DEFAULT_CONFIG, ...config };
    }

    /**
     * Main validation entry point
     * Validates security context for an operation across all applicable tiers
     */
    async validateOperation(
        context: EnhancedSecurityContext,
        operation: {
            action: string;
            resource: string;
            data?: unknown;
            metadata?: Record<string, unknown>;
        },
    ): Promise<SecurityValidation> {
        const startTime = performance.now();
        this.validationCount++;

        try {
            const validation: SecurityValidation = {
                valid: true,
                violations: [],
                warnings: [],
                metadata: {
                    tier: context.tier,
                    requestId: context.requestId,
                    validationStartTime: new Date(),
                },
            };

            // Step 1: Validate security level requirements
            await this.validateSecurityLevel(context, operation, validation);

            // Step 2: Validate tier-specific permissions
            if (this.config.enableTierValidation) {
                await this.validateTierPermissions(context, operation, validation);
            }

            // Step 3: Apply guard rails
            await this.applyGuardRails(context, operation, validation);

            // Step 4: Check access policies
            await this.validateAccessPolicies(context, operation, validation);

            // Step 5: Assess threat level
            if (this.config.enableThreatDetection) {
                await this.assessThreatLevel(context, operation, validation);
            }

            // Step 6: Calculate final risk score
            this.calculateRiskScore(context, validation);

            // Step 7: Make final validation decision
            this.makeFinalDecision(context, validation);

            // Step 8: Emit security events for learning
            await this.emitSecurityEvents(context, operation, validation);

            // Step 9: Update statistics
            this.updateStatistics(validation);

            const duration = performance.now() - startTime;
            await this.telemetry.emitExecutionTiming(
                "SecurityValidator",
                "validateOperation",
                new Date(startTime),
                new Date(),
                validation.valid,
                {
                    tier: context.tier,
                    violations: validation.violations.length,
                    warnings: validation.warnings.length,
                },
            );

            return validation;
        } catch (error) {
            logger.error("Security validation error", {
                requestId: context.requestId,
                tier: context.tier,
                error,
            });

            await this.telemetry.emitError(
                error,
                "SecurityValidator",
                "critical",
                { requestId: context.requestId },
            );

            // Return blocking validation on error
            return {
                valid: false,
                violations: [{
                    id: uuidv4(),
                    timestamp: new Date(),
                    guardRailId: "system-error",
                    ruleId: "validation-error",
                    severity: "critical",
                    message: "Security validation system error",
                    context: this.createViolationContext(context, operation),
                    action: "block",
                    resolved: false,
                }],
                warnings: ["Security validation system error occurred"],
                metadata: { error: true },
            };
        }
    }

    /**
     * Create enhanced security context
     */
    async createSecurityContext(
        tier: 1 | 2 | 3,
        requestId: string,
        origin: SecurityOrigin,
        baseContext?: Partial<EnhancedSecurityContext>,
    ): Promise<EnhancedSecurityContext> {
        const context: EnhancedSecurityContext = {
            level: SecurityLevel.PRIVATE,
            policies: [],
            guardRails: [],
            barriers: [],
            compliance: [],
            audit: true,
            tier,
            requestId,
            origin,
            permissions: [],
            riskScore: 0,
            threatLevel: ThreatLevel.NONE,
            validationResults: [],
            ...baseContext,
        };

        // Inherit permissions from higher tiers if enabled
        if (this.config.enablePermissionInheritance && tier > 1) {
            await this.inheritPermissions(context);
        }

        // Calculate initial risk score based on origin
        context.riskScore = this.calculateOriginRisk(origin);

        // Set initial threat level
        context.threatLevel = this.determineThreatLevel(context.riskScore);

        return context;
    }

    /**
     * Propagate security context between tiers
     */
    async propagateContext(
        sourceContext: EnhancedSecurityContext,
        targetTier: 1 | 2 | 3,
        newRequestId?: string,
    ): Promise<EnhancedSecurityContext> {
        const propagatedContext: EnhancedSecurityContext = {
            ...sourceContext,
            tier: targetTier,
            requestId: newRequestId || sourceContext.requestId,
            validationResults: [], // Reset validation results for new tier
        };

        // Apply tier-specific security constraints
        await this.applyTierConstraints(propagatedContext);

        // Adjust permissions for target tier
        propagatedContext.permissions = this.filterPermissionsForTier(
            sourceContext.permissions,
            targetTier,
        );

        // Recalculate risk score for new tier
        propagatedContext.riskScore = Math.min(
            sourceContext.riskScore * this.getTierRiskMultiplier(targetTier),
            1.0,
        );

        // Update threat level
        propagatedContext.threatLevel = this.determineThreatLevel(propagatedContext.riskScore);

        return propagatedContext;
    }

    /**
     * Validate security level requirements
     */
    private async validateSecurityLevel(
        context: EnhancedSecurityContext,
        operation: { action: string; resource: string; data?: unknown },
        validation: SecurityValidation,
    ): Promise<void> {
        // Check if operation requires higher security level
        const requiredLevel = this.getRequiredSecurityLevel(operation);
        
        if (this.compareSecurityLevels(context.level, requiredLevel) < 0) {
            const violation: GuardRailViolation = {
                id: uuidv4(),
                timestamp: new Date(),
                guardRailId: "security-level",
                ruleId: "insufficient-security-level",
                severity: "error",
                message: `Operation requires ${requiredLevel} but context has ${context.level}`,
                context: this.createViolationContext(context, operation),
                action: "block",
                resolved: false,
            };

            validation.violations.push(violation);
            validation.valid = false;
        }
    }

    /**
     * Validate tier-specific permissions
     */
    private async validateTierPermissions(
        context: EnhancedSecurityContext,
        operation: { action: string; resource: string; data?: unknown },
        validation: SecurityValidation,
    ): Promise<void> {
        // Check if user has permission for this action on this resource in this tier
        const hasPermission = context.permissions.some(permission => 
            this.matchesPermission(permission, operation, context.tier),
        );

        if (!hasPermission) {
            const violation: GuardRailViolation = {
                id: uuidv4(),
                timestamp: new Date(),
                guardRailId: "tier-permissions",
                ruleId: "insufficient-permissions",
                severity: "error",
                message: `No permission for ${operation.action} on ${operation.resource} in tier ${context.tier}`,
                context: this.createViolationContext(context, operation),
                action: "block",
                resolved: false,
            };

            validation.violations.push(violation);
            validation.valid = false;
        }
    }

    /**
     * Apply guard rails
     */
    private async applyGuardRails(
        context: EnhancedSecurityContext,
        operation: { action: string; resource: string; data?: unknown },
        validation: SecurityValidation,
    ): Promise<void> {
        for (const guardRailId of context.guardRails) {
            const guardRail = await this.getGuardRail(guardRailId);
            if (!guardRail || !guardRail.enabled) {
                continue;
            }

            // Evaluate guard rail rules
            for (const rule of guardRail.config.rules) {
                const violated = await this.evaluateGuardRailRule(rule, context, operation);
                
                if (violated) {
                    const violation: GuardRailViolation = {
                        id: uuidv4(),
                        timestamp: new Date(),
                        guardRailId: guardRail.id,
                        ruleId: rule.id,
                        severity: guardRail.config.severity,
                        message: rule.message,
                        context: this.createViolationContext(context, operation),
                        action: this.determineAction(guardRail.config.mode, guardRail.config.severity),
                        resolved: false,
                    };

                    validation.violations.push(violation);

                    if (guardRail.config.mode === "block") {
                        validation.valid = false;
                    }
                }
            }
        }
    }

    /**
     * Validate access policies
     */
    private async validateAccessPolicies(
        context: EnhancedSecurityContext,
        operation: { action: string; resource: string; data?: unknown },
        validation: SecurityValidation,
    ): Promise<void> {
        for (const policyId of context.policies) {
            const policy = await this.getAccessPolicy(policyId);
            if (!policy) {
                continue;
            }

            const decision = await this.evaluateAccessPolicy(policy, context, operation);
            
            if (decision.effect === "deny") {
                const violation: GuardRailViolation = {
                    id: uuidv4(),
                    timestamp: new Date(),
                    guardRailId: "access-policy",
                    ruleId: policy.id,
                    severity: "error",
                    message: `Access denied by policy: ${policy.name}`,
                    context: this.createViolationContext(context, operation),
                    action: "block",
                    resolved: false,
                };

                validation.violations.push(violation);
                validation.valid = false;
            }
        }
    }

    /**
     * Assess threat level based on context and operation
     */
    private async assessThreatLevel(
        context: EnhancedSecurityContext,
        operation: { action: string; resource: string; data?: unknown },
        validation: SecurityValidation,
    ): Promise<void> {
        // Check violation history
        const history = this.violationHistory.get(context.requestId) || [];
        
        if (history.length >= this.config.maxViolationsBeforeBlock) {
            validation.warnings.push(
                `High violation count: ${history.length} violations detected`,
            );
            
            // Increase threat level
            context.threatLevel = ThreatLevel.HIGH;
            context.riskScore = Math.min(context.riskScore + 0.2, 1.0);
        }

        // Check for suspicious patterns
        if (this.detectSuspiciousPattern(context, operation)) {
            validation.warnings.push("Suspicious access pattern detected");
            context.threatLevel = ThreatLevel.MEDIUM;
            context.riskScore = Math.min(context.riskScore + 0.1, 1.0);
        }
    }

    /**
     * Calculate overall risk score
     */
    private calculateRiskScore(
        context: EnhancedSecurityContext,
        validation: SecurityValidation,
    ): void {
        let riskScore = context.riskScore;

        // Add risk based on violations
        riskScore += validation.violations.length * 0.1;

        // Add risk based on trust level
        const trustMultiplier = this.config.trustLevelThresholds[context.origin.trustLevel];
        riskScore *= (1 - trustMultiplier);

        // Clamp to [0, 1]
        context.riskScore = Math.max(0, Math.min(riskScore, 1));
    }

    /**
     * Make final validation decision
     */
    private makeFinalDecision(
        context: EnhancedSecurityContext,
        validation: SecurityValidation,
    ): void {
        // Block if risk score exceeds threshold
        if (context.riskScore > this.config.riskThreshold) {
            validation.valid = false;
            validation.warnings.push(
                `Risk score ${context.riskScore.toFixed(2)} exceeds threshold ${this.config.riskThreshold}`,
            );
        }

        // Block if threat level is critical
        if (context.threatLevel === ThreatLevel.CRITICAL) {
            validation.valid = false;
            validation.warnings.push("Critical threat level detected");
        }
    }

    /**
     * Emit security events for AI learning
     */
    private async emitSecurityEvents(
        context: EnhancedSecurityContext,
        operation: { action: string; resource: string; data?: unknown },
        validation: SecurityValidation,
    ): Promise<void> {
        // Emit validation event
        await this.eventBus.publish({
            id: uuidv4(),
            type: "security.validation.completed",
            timestamp: new Date(),
            source: {
                tier: context.tier,
                component: "SecurityValidator",
                service: "security",
            },
            correlationId: context.requestId,
            payload: {
                valid: validation.valid,
                riskScore: context.riskScore,
                threatLevel: context.threatLevel,
                violationCount: validation.violations.length,
                warningCount: validation.warnings.length,
                operation: {
                    action: operation.action,
                    resource: operation.resource,
                },
                context: {
                    tier: context.tier,
                    securityLevel: context.level,
                    trustLevel: context.origin.trustLevel,
                    originType: context.origin.type,
                },
            },
            metadata: {
                category: "security",
                priority: validation.valid ? "normal" : "high",
            },
        });

        // Emit violation events
        for (const violation of validation.violations) {
            await this.eventBus.publish({
                id: uuidv4(),
                type: "security.violation.detected",
                timestamp: new Date(),
                source: {
                    tier: context.tier,
                    component: "SecurityValidator",
                    service: "security",
                },
                correlationId: context.requestId,
                payload: violation,
                metadata: {
                    category: "security",
                    priority: violation.severity === "critical" ? "always" : "high",
                },
            });
        }
    }

    /**
     * Helper methods
     */
    private async getGuardRail(id: string): Promise<GuardRail | null> {
        if (this.guardRailCache.has(id)) {
            return this.guardRailCache.get(id)!;
        }

        // In a real implementation, this would fetch from database
        // For now, return null
        return null;
    }

    private async getAccessPolicy(id: string): Promise<AccessPolicy | null> {
        if (this.policyCache.has(id)) {
            return this.policyCache.get(id)!;
        }

        // In a real implementation, this would fetch from database
        // For now, return null
        return null;
    }

    private createViolationContext(
        context: EnhancedSecurityContext,
        operation: { action: string; resource: string; data?: unknown },
    ): ViolationContext {
        return {
            tier: context.tier,
            component: "SecurityValidator",
            requestId: context.requestId,
            input: operation,
            metadata: {
                securityLevel: context.level,
                threatLevel: context.threatLevel,
                riskScore: context.riskScore,
                trustLevel: context.origin.trustLevel,
            },
        };
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

    private getRequiredSecurityLevel(operation: { action: string; resource: string }): SecurityLevel {
        // Determine required security level based on operation
        if (operation.resource.includes("sensitive") || operation.action === "admin") {
            return SecurityLevel.CONFIDENTIAL;
        }
        if (operation.action === "write" || operation.action === "delete") {
            return SecurityLevel.RESTRICTED;
        }
        return SecurityLevel.PRIVATE;
    }

    private matchesPermission(
        permission: Permission,
        operation: { action: string; resource: string },
        tier: number,
    ): boolean {
        // Check action match
        if (permission.action !== "*" && permission.action !== operation.action) {
            return false;
        }

        // Check resource match
        if (permission.resource !== "*" && !operation.resource.startsWith(permission.resource)) {
            return false;
        }

        // Check tier scope
        if (permission.scope?.tier && permission.scope.tier !== tier) {
            return false;
        }

        // Check expiration
        if (permission.expiresAt && permission.expiresAt < new Date()) {
            return false;
        }

        return true;
    }

    private async evaluateGuardRailRule(
        rule: { id: string; condition: string; message: string },
        context: EnhancedSecurityContext,
        operation: { action: string; resource: string; data?: unknown },
    ): Promise<boolean> {
        // In a real implementation, this would evaluate the condition expression
        // For now, return false (no violation)
        return false;
    }

    private async evaluateAccessPolicy(
        policy: AccessPolicy,
        context: EnhancedSecurityContext,
        operation: { action: string; resource: string; data?: unknown },
    ): Promise<{ effect: "allow" | "deny"; reason?: string }> {
        // In a real implementation, this would evaluate the policy
        // For now, return allow
        return { effect: "allow" };
    }

    private determineAction(mode: "monitor" | "block" | "modify", severity: string): string {
        if (mode === "block" || severity === "critical") {
            return "block";
        }
        if (mode === "modify") {
            return "modify";
        }
        return "log";
    }

    private calculateOriginRisk(origin: SecurityOrigin): number {
        const trustMultiplier = this.config.trustLevelThresholds[origin.trustLevel];
        let baseRisk = 1 - trustMultiplier;

        // Adjust based on origin type
        switch (origin.type) {
            case OriginType.SYSTEM:
                baseRisk *= 0.1;
                break;
            case OriginType.AGENT:
                baseRisk *= 0.5;
                break;
            case OriginType.USER:
                baseRisk *= 0.7;
                break;
            case OriginType.EXTERNAL_API:
                baseRisk *= 1.2;
                break;
        }

        return Math.max(0, Math.min(baseRisk, 1));
    }

    private determineThreatLevel(riskScore: number): ThreatLevel {
        if (riskScore >= 0.8) return ThreatLevel.CRITICAL;
        if (riskScore >= 0.6) return ThreatLevel.HIGH;
        if (riskScore >= 0.4) return ThreatLevel.MEDIUM;
        if (riskScore >= 0.2) return ThreatLevel.LOW;
        return ThreatLevel.NONE;
    }

    private async inheritPermissions(context: EnhancedSecurityContext): Promise<void> {
        // In a real implementation, this would inherit permissions from parent tiers
        // For now, keep existing permissions
    }

    private async applyTierConstraints(context: EnhancedSecurityContext): Promise<void> {
        // Apply tier-specific security constraints
        switch (context.tier) {
            case 1:
                // Tier 1 (Coordination) - highest privileges, most restrictions
                context.guardRails.push("coordination-guardrail");
                break;
            case 2:
                // Tier 2 (Process) - medium privileges
                context.guardRails.push("process-guardrail");
                break;
            case 3:
                // Tier 3 (Execution) - limited privileges, direct execution
                context.guardRails.push("execution-guardrail");
                break;
        }
    }

    private filterPermissionsForTier(permissions: Permission[], tier: number): Permission[] {
        return permissions.filter(permission => {
            // Filter permissions based on tier scope
            if (permission.scope?.tier && permission.scope.tier !== tier) {
                return false;
            }
            return true;
        });
    }

    private getTierRiskMultiplier(tier: number): number {
        // Higher tiers have more responsibility, so higher risk multiplier
        switch (tier) {
            case 1: return 1.2; // Coordination tier
            case 2: return 1.0; // Process tier
            case 3: return 0.8; // Execution tier
            default: return 1.0;
        }
    }

    private detectSuspiciousPattern(
        context: EnhancedSecurityContext,
        operation: { action: string; resource: string; data?: unknown },
    ): boolean {
        // In a real implementation, this would use ML or rule-based pattern detection
        // For now, return false
        return false;
    }

    private updateStatistics(validation: SecurityValidation): void {
        if (!validation.valid) {
            this.blockedOperations++;
        }
        this.violationCount += validation.violations.length;
    }

    /**
     * Get security validation statistics
     */
    getStatistics(): {
        validationCount: number;
        violationCount: number;
        blockedOperations: number;
        averageViolationsPerValidation: number;
        blockRate: number;
    } {
        return {
            validationCount: this.validationCount,
            violationCount: this.violationCount,
            blockedOperations: this.blockedOperations,
            averageViolationsPerValidation: this.validationCount > 0
                ? this.violationCount / this.validationCount
                : 0,
            blockRate: this.validationCount > 0
                ? this.blockedOperations / this.validationCount
                : 0,
        };
    }
}