/**
 * Security Infrastructure Index
 * Exports comprehensive security components for emergent security intelligence
 * 
 * This module provides the foundation for AI swarms to learn security patterns
 * through systematic validation, threat detection, context management, and
 * comprehensive audit logging for pattern detection and learning.
 */

export { SecurityValidator } from "./securityValidator.js";
export { AISecurityValidator } from "./aiSecurityValidator.js";
export { SecurityContextManager } from "./securityContextManager.js";
export { SecurityAuditLogger } from "./securityAuditLogger.js";

/**
 * Re-export key types for convenience
 */
export type {
    EnhancedSecurityContext,
    SecurityValidation,
    AISecurityValidation,
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
    ValidationIssue,
    ViolationContext,
    SecurityAudit,
    EnhancedSecurityAudit,
    SecurityIncident,
    ThreatDetection,
    SecurityRisk,
    AuditRecommendation,
} from "@vrooli/shared";

/**
 * Security Infrastructure Factory
 * Creates coordinated security components with proper dependency injection
 */
export class SecurityInfrastructure {
    private securityValidator: SecurityValidator;
    private aiSecurityValidator: AISecurityValidator;
    private contextManager: SecurityContextManager;
    private auditLogger: SecurityAuditLogger;

    constructor(
        telemetryShim: any, // TelemetryShim type
        eventBus: any, // RedisEventBus type
        options: {
            enableAIValidation?: boolean;
            enableContextCaching?: boolean;
            enableRealTimeAlerts?: boolean;
            auditRetentionDays?: number;
        } = {},
    ) {
        // Initialize core components
        this.securityValidator = new SecurityValidator(telemetryShim, eventBus);
        this.aiSecurityValidator = new AISecurityValidator(telemetryShim, eventBus);
        this.contextManager = new SecurityContextManager(telemetryShim, eventBus, {
            cacheContexts: options.enableContextCaching !== false,
        });
        this.auditLogger = new SecurityAuditLogger(telemetryShim, eventBus, {
            enableRealTimeAlerts: options.enableRealTimeAlerts !== false,
            retentionPeriodDays: options.auditRetentionDays || 90,
        });
    }

    /**
     * Get security validator instance
     */
    getSecurityValidator(): SecurityValidator {
        return this.securityValidator;
    }

    /**
     * Get AI security validator instance
     */
    getAISecurityValidator(): AISecurityValidator {
        return this.aiSecurityValidator;
    }

    /**
     * Get security context manager instance
     */
    getContextManager(): SecurityContextManager {
        return this.contextManager;
    }

    /**
     * Get security audit logger instance
     */
    getAuditLogger(): SecurityAuditLogger {
        return this.auditLogger;
    }

    /**
     * Comprehensive security validation workflow
     * Provides a high-level interface for complete security validation
     */
    async validateSecureOperation(
        operation: {
            action: string;
            resource: string;
            data?: unknown;
            metadata?: Record<string, unknown>;
        },
        context: {
            tier: 1 | 2 | 3;
            requestId: string;
            origin: any; // SecurityOrigin type
            sessionId?: string;
            parentContext?: any; // EnhancedSecurityContext type
        },
        options: {
            enableAIValidation?: boolean;
            requireElevatedPermissions?: boolean;
            auditLevel?: "basic" | "detailed" | "comprehensive";
        } = {},
    ): Promise<{
        valid: boolean;
        securityContext: any; // EnhancedSecurityContext type
        validationResults: any; // SecurityValidation type
        aiValidation?: any; // AISecurityValidation type
        auditId: string;
        riskScore: number;
        recommendations: string[];
    }> {
        try {
            // Step 1: Create or validate security context
            const securityContext = await this.contextManager.createSecurityContext(
                context.tier,
                context.requestId,
                context.origin,
                {
                    sessionId: context.sessionId,
                    parentContext: context.parentContext,
                },
            );

            // Step 2: Perform core security validation
            const validationResults = await this.securityValidator.validateOperation(
                securityContext,
                operation,
            );

            // Step 3: Perform AI-specific validation if enabled and applicable
            let aiValidation;
            if (options.enableAIValidation && this.isAIOperation(operation)) {
                const input = this.extractInput(operation);
                const output = this.extractOutput(operation);
                
                if (input && output) {
                    aiValidation = await this.aiSecurityValidator.validateAIInteraction(
                        input,
                        output,
                        {
                            requestId: context.requestId,
                            tier: context.tier,
                            component: "SecurityInfrastructure",
                            operation: operation.action,
                        },
                    );
                }
            }

            // Step 4: Log security audit
            const audit = await this.auditLogger.logSecurityAudit(
                securityContext.origin.identifier,
                operation.action,
                operation.resource,
                validationResults.valid ? "success" : "failure",
                {
                    tier: context.tier,
                    requestId: context.requestId,
                    sessionId: context.sessionId,
                    securityContext,
                    metadata: {
                        validationResults,
                        aiValidation,
                        ...operation.metadata,
                    },
                },
            );

            // Step 5: Log any violations
            for (const violation of validationResults.violations) {
                await this.auditLogger.logSecurityViolation(violation, securityContext);
            }

            // Step 6: Calculate overall risk score
            const riskScore = this.calculateOverallRiskScore(
                securityContext.riskScore,
                validationResults,
                aiValidation,
            );

            // Step 7: Generate recommendations
            const recommendations = this.generateSecurityRecommendations(
                validationResults,
                aiValidation,
                riskScore,
            );

            return {
                valid: validationResults.valid && (!aiValidation || aiValidation.overallRisk < 0.7),
                securityContext,
                validationResults,
                aiValidation,
                auditId: audit.id,
                riskScore,
                recommendations,
            };
        } catch (error) {
            // Log security error
            await this.auditLogger.logSecurityAudit(
                context.origin.identifier,
                "security_validation_error",
                operation.resource,
                "failure",
                {
                    tier: context.tier,
                    requestId: context.requestId,
                    metadata: {
                        error: error instanceof Error ? error.message : String(error),
                        operation,
                    },
                },
            );

            throw error;
        }
    }

    /**
     * Security context propagation for tier transitions
     */
    async propagateSecurityContext(
        sourceContext: any, // EnhancedSecurityContext type
        targetTier: 1 | 2 | 3,
        newRequestId?: string,
        options: {
            elevatePermissions?: boolean;
            restrictAccess?: boolean;
            auditTransition?: boolean;
        } = {},
    ): Promise<{
        context: any; // EnhancedSecurityContext type
        auditId?: string;
    }> {
        // Propagate context
        const propagatedContext = await this.contextManager.propagateContext(
            sourceContext,
            targetTier,
            newRequestId,
            {
                elevateLevel: options.elevatePermissions,
                restrictPermissions: options.restrictAccess,
            },
        );

        // Audit transition if requested
        let auditId;
        if (options.auditTransition !== false) {
            const audit = await this.auditLogger.logSecurityAudit(
                sourceContext.origin.identifier,
                "context_propagation",
                `tier_${sourceContext.tier}_to_tier_${targetTier}`,
                "success",
                {
                    tier: targetTier,
                    requestId: propagatedContext.requestId,
                    securityContext: propagatedContext,
                    metadata: {
                        sourceTier: sourceContext.tier,
                        targetTier,
                        riskScoreChange: propagatedContext.riskScore - sourceContext.riskScore,
                    },
                },
            );
            auditId = audit.id;
        }

        return {
            context: propagatedContext,
            auditId,
        };
    }

    /**
     * Security incident response workflow
     */
    async handleSecurityIncident(
        type: string, // IncidentType
        severity: "low" | "medium" | "high" | "critical",
        title: string,
        description: string,
        context: {
            tier?: number;
            requestId?: string;
            securityContext?: any; // EnhancedSecurityContext type
            evidence?: Record<string, unknown>;
        },
        response?: {
            containmentActions?: string[];
            notificationChannels?: string[];
            escalationRequired?: boolean;
        },
    ): Promise<{
        incidentId: string;
        auditId: string;
        recommendedActions: string[];
    }> {
        // Log the incident
        const incident = await this.auditLogger.logSecurityIncident(
            type as any,
            severity,
            title,
            description,
            context,
        );

        // Create audit entry for incident response
        const audit = await this.auditLogger.logSecurityAudit(
            "system",
            "incident_response",
            "security_incident",
            "success",
            {
                tier: context.tier,
                requestId: context.requestId,
                securityContext: context.securityContext,
                metadata: {
                    incidentId: incident.id,
                    severity,
                    containmentActions: response?.containmentActions,
                    escalationRequired: response?.escalationRequired,
                },
            },
        );

        // Generate recommended actions
        const recommendedActions = this.generateIncidentRecommendations(
            incident,
            severity,
            response,
        );

        return {
            incidentId: incident.id,
            auditId: audit.id,
            recommendedActions,
        };
    }

    /**
     * Get comprehensive security statistics
     */
    getSecurityStatistics(): {
        validation: {
            totalValidations: number;
            blockRate: number;
            averageViolations: number;
        };
        aiSecurity: {
            totalValidations: number;
            promptInjectionAttempts: number;
            piiDetections: number;
            averageRiskScore: number;
        };
        contextManagement: {
            contextsCreated: number;
            contextsPropagated: number;
            permissionElevations: number;
            cacheHitRate: number;
        };
        auditing: {
            totalAudits: number;
            incidentCount: number;
            threatDetections: number;
            alertsSent: number;
        };
        patterns: {
            activePatterns: number;
            recentTriggers: number;
            topThreats: string[];
        };
    } {
        const validationStats = this.securityValidator.getStatistics();
        const aiStats = this.aiSecurityValidator.getStatistics();
        const contextStats = this.contextManager.getStatistics();
        const auditStats = this.auditLogger.getStatistics();

        return {
            validation: {
                totalValidations: validationStats.validationCount,
                blockRate: validationStats.blockRate,
                averageViolations: validationStats.averageViolationsPerValidation,
            },
            aiSecurity: {
                totalValidations: aiStats.validationCount,
                promptInjectionAttempts: aiStats.promptInjectionAttempts,
                piiDetections: aiStats.piiDetections,
                averageRiskScore: aiStats.averageRiskScore,
            },
            contextManagement: {
                contextsCreated: contextStats.contextCreationCount,
                contextsPropagated: contextStats.contextPropagationCount,
                permissionElevations: contextStats.permissionElevationCount,
                cacheHitRate: contextStats.cacheHitRate,
            },
            auditing: {
                totalAudits: auditStats.totalAudits,
                incidentCount: auditStats.totalIncidents,
                threatDetections: auditStats.totalThreatDetections,
                alertsSent: auditStats.alertsSent,
            },
            patterns: {
                activePatterns: auditStats.activePatterns,
                recentTriggers: 0, // Would be calculated from recent pattern activity
                topThreats: [], // Would be calculated from threat detection data
            },
        };
    }

    /**
     * Helper methods
     */
    private isAIOperation(operation: { action: string; resource: string; data?: unknown }): boolean {
        const aiActions = ["generate", "analyze", "process", "chat", "completion"];
        const aiResources = ["llm", "ai", "model", "assistant"];
        
        return aiActions.some(action => operation.action.includes(action)) ||
               aiResources.some(resource => operation.resource.includes(resource));
    }

    private extractInput(operation: { action: string; resource: string; data?: unknown }): string | null {
        if (typeof operation.data === "object" && operation.data !== null) {
            const data = operation.data as Record<string, unknown>;
            return String(data.input || data.prompt || data.message || "");
        }
        return typeof operation.data === "string" ? operation.data : null;
    }

    private extractOutput(operation: { action: string; resource: string; data?: unknown }): string | null {
        if (typeof operation.data === "object" && operation.data !== null) {
            const data = operation.data as Record<string, unknown>;
            return String(data.output || data.response || data.result || "");
        }
        return null;
    }

    private calculateOverallRiskScore(
        contextRisk: number,
        validationResults: any, // SecurityValidation type
        aiValidation?: any, // AISecurityValidation type
    ): number {
        let totalRisk = contextRisk;

        // Add risk from validation violations
        totalRisk += validationResults.violations.length * 0.1;

        // Add risk from AI validation if present
        if (aiValidation) {
            totalRisk += aiValidation.overallRisk * 0.5;
        }

        return Math.min(totalRisk, 1.0);
    }

    private generateSecurityRecommendations(
        validationResults: any, // SecurityValidation type
        aiValidation?: any, // AISecurityValidation type
        riskScore?: number,
    ): string[] {
        const recommendations: string[] = [];

        if (validationResults.violations.length > 0) {
            recommendations.push("Review and address security violations");
        }

        if (validationResults.warnings.length > 0) {
            recommendations.push("Investigate security warnings");
        }

        if (aiValidation && aiValidation.overallRisk > 0.5) {
            recommendations.push("Review AI interaction for security concerns");
        }

        if (riskScore && riskScore > 0.7) {
            recommendations.push("Consider additional security measures");
        }

        if (recommendations.length === 0) {
            recommendations.push("Security validation passed - monitor for anomalies");
        }

        return recommendations;
    }

    private generateIncidentRecommendations(
        incident: any, // SecurityIncident type
        severity: string,
        response?: any,
    ): string[] {
        const recommendations: string[] = [];

        if (severity === "critical") {
            recommendations.push("Immediate containment required");
            recommendations.push("Escalate to security team");
            recommendations.push("Preserve evidence");
        } else if (severity === "high") {
            recommendations.push("Investigate within 1 hour");
            recommendations.push("Monitor for escalation");
        }

        recommendations.push("Document incident details");
        recommendations.push("Review security policies");
        recommendations.push("Update threat intelligence");

        return recommendations;
    }

    /**
     * Shutdown all components gracefully
     */
    async shutdown(): Promise<void> {
        // Security components don't currently have shutdown methods
        // In a real implementation, they would clean up resources, flush logs, etc.
    }
}

/**
 * Default instance factory
 * Creates a pre-configured security infrastructure instance
 */
export function createSecurityInfrastructure(
    telemetryShim: any,
    eventBus: any,
    options?: {
        enableAIValidation?: boolean;
        enableContextCaching?: boolean;
        enableRealTimeAlerts?: boolean;
        auditRetentionDays?: number;
        maxOverheadMs?: number;
    },
): SecurityInfrastructure {
    return new SecurityInfrastructure(telemetryShim, eventBus, options);
}

/**
 * Usage Examples and Integration Patterns
 * 
 * Basic Security Validation:
 * ```typescript
 * const security = createSecurityInfrastructure(telemetryShim, eventBus);
 * 
 * // Validate a secure operation
 * const result = await security.validateSecureOperation(
 *   {
 *     action: "execute",
 *     resource: "routine.sensitive-data-processing",
 *     data: { input: "user data" }
 *   },
 *   {
 *     tier: 2,
 *     requestId: "req-123",
 *     origin: {
 *       type: OriginType.USER,
 *       identifier: "user-456",
 *       verified: true,
 *       trustLevel: TrustLevel.MEDIUM,
 *       metadata: {}
 *     }
 *   },
 *   {
 *     enableAIValidation: true,
 *     auditLevel: "comprehensive"
 *   }
 * );
 * 
 * if (!result.valid) {
 *   console.error("Security validation failed:", result.recommendations);
 * }
 * ```
 * 
 * Context Propagation:
 * ```typescript
 * // Propagate security context from tier 1 to tier 2
 * const { context } = await security.propagateSecurityContext(
 *   coordinationContext,
 *   2, // target tier
 *   "new-request-id",
 *   {
 *     elevatePermissions: false,
 *     restrictAccess: true,
 *     auditTransition: true
 *   }
 * );
 * ```
 * 
 * Incident Response:
 * ```typescript
 * // Handle a security incident
 * const incident = await security.handleSecurityIncident(
 *   IncidentType.INJECTION_ATTACK,
 *   "high",
 *   "SQL Injection Attempt",
 *   "Detected SQL injection in user input",
 *   {
 *     tier: 3,
 *     requestId: "req-789",
 *     evidence: { maliciousInput: "'; DROP TABLE users; --" }
 *   },
 *   {
 *     containmentActions: ["block_user", "sanitize_input"],
 *     escalationRequired: true
 *   }
 * );
 * ```
 * 
 * Agent Learning Integration:
 * ```typescript
 * // Agents can subscribe to security events for learning
 * eventBus.subscribe({
 *   id: "security-learning-agent",
 *   filters: [{ field: "type", value: "security.*" }],
 *   handler: async (event) => {
 *     // Extract security patterns and update agent knowledge
 *     await agent.learnFromSecurityEvent(event);
 *   }
 * });
 * ```
 * 
 * AI Security Validation:
 * ```typescript
 * // Specialized AI security validation
 * const aiValidator = security.getAISecurityValidator();
 * 
 * const aiValidation = await aiValidator.validateAIInteraction(
 *   "User prompt input",
 *   "AI generated response",
 *   {
 *     requestId: "ai-req-123",
 *     tier: 3,
 *     component: "LLMService",
 *     operation: "generate_response"
 *   }
 * );
 * 
 * if (aiValidation.overallRisk > 0.7) {
 *   console.warn("High risk AI interaction detected");
 * }
 * ```
 * 
 * Security Analytics:
 * ```typescript
 * // Get comprehensive security statistics
 * const stats = security.getSecurityStatistics();
 * console.log("Security Overview:", {
 *   blockRate: stats.validation.blockRate,
 *   threatDetections: stats.auditing.threatDetections,
 *   averageRisk: stats.aiSecurity.averageRiskScore
 * });
 * ```
 */