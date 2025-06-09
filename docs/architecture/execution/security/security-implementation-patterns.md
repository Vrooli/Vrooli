# Security Implementation Patterns

This document provides concrete implementation patterns, code examples, and construction guidance for building Vrooli's security architecture across the three-tier execution system.

**Prerequisites**:
- Read [Security Overview](README.md) for architectural context
- Review [Security Boundaries](security-boundaries.md) for trust models
- Study [Types System](../types/core-types.ts) for security interface definitions

## Core Security Components Implementation

### **SecurityManager - Central Coordinator**

The SecurityManager orchestrates all security operations and provides the main integration point:

```typescript
/**
 * Central security coordinator that orchestrates all security operations
 * across the three-tier execution architecture.
 */
export class SecurityManager {
    private authService: AuthenticationService;
    private authzEngine: AuthorizationEngine;
    private auditLogger: AuditLogger;
    private contextManager: SecurityContextManager;
    private tierGuards: Map<Tier, TierSecurityGuard>;
    private aiSecurityManager: AISecurityManager;

    constructor(config: SecurityConfig) {
        this.authService = new AuthenticationService(config.auth);
        this.authzEngine = new AuthorizationEngine(config.authorization);
        this.auditLogger = new AuditLogger(config.audit);
        this.contextManager = new SecurityContextManager(config.context);
        this.aiSecurityManager = new AISecurityManager(config.aiSecurity);
        
        // Initialize tier-specific security guards
        this.tierGuards = new Map([
            [Tier.TIER_1, new Tier1SecurityGuard(this.authzEngine, this.auditLogger)],
            [Tier.TIER_2, new Tier2SecurityGuard(this.authzEngine, this.auditLogger)],
            [Tier.TIER_3, new Tier3SecurityGuard(this.authzEngine, this.auditLogger)]
        ]);
    }

    /**
     * Main validation entry point for cross-tier operations
     */
    async validateTierTransition(
        request: TierTransitionRequest
    ): Promise<SecurityValidationResult> {
        try {
            // 1. Authentication
            const authResult = await this.authService.authenticate(request.credentials);
            if (!authResult.success) {
                return this.createFailureResult("Authentication failed", request);
            }

            // 2. Create security context
            const securityContext = await this.contextManager.createOrValidate({
                requesterTier: request.sourceTier,
                targetTier: request.targetTier,
                operation: request.operation,
                user: authResult.user,
                permissions: authResult.permissions,
                sessionId: request.sessionId
            });

            // 3. AI-specific validation
            if (request.aiInput) {
                const aiValidation = await this.aiSecurityManager.validateInput(
                    request.aiInput, 
                    securityContext
                );
                if (!aiValidation.success) {
                    return {
                        success: false,
                        error: "AI input validation failed",
                        threats: aiValidation.threats,
                        auditEntry: aiValidation.auditEntry
                    };
                }
            }

            // 4. Tier-specific validation
            const targetGuard = this.tierGuards.get(request.targetTier);
            if (!targetGuard) {
                throw new Error(`No security guard found for tier: ${request.targetTier}`);
            }

            const validationResult = await targetGuard.validateRequest(request, securityContext);

            // 5. Audit logging
            await this.auditLogger.logValidation(request, securityContext, validationResult);

            return validationResult;

        } catch (error) {
            return this.createErrorResult(error as Error, request);
        }
    }

    private async createFailureResult(
        reason: string, 
        request: TierTransitionRequest
    ): Promise<SecurityValidationResult> {
        const auditEntry = await this.auditLogger.logFailure("security_validation", request, reason);
        return { 
            success: false, 
            error: reason,
            auditEntry
        };
    }

    private async createErrorResult(
        error: Error, 
        request: TierTransitionRequest
    ): Promise<SecurityValidationResult> {
        const auditEntry = await this.auditLogger.logError("security_validation_error", request, error);
        return {
            success: false,
            error: `Security validation error: ${error.message}`,
            auditEntry
        };
    }
}
```

### **Tier-Specific Security Guards**

Each tier implements specialized security enforcement:

```typescript
/**
 * Tier 1 Security Guard - Swarm and coordination security
 */
export class Tier1SecurityGuard implements TierSecurityGuard {
    constructor(
        private authzEngine: AuthorizationEngine,
        private auditLogger: AuditLogger
    ) {}

    async validateRequest(
        request: TierTransitionRequest,
        context: SecurityContext
    ): Promise<SecurityValidationResult> {
        const validations = [
            this.validateSwarmAccess(request, context),
            this.validateTeamPermissions(request, context),
            this.validateResourceAuthorization(request, context),
            this.validateGoalModification(request, context)
        ];

        return this.aggregateValidationResults(validations, "tier1_validation", request, context);
    }

    private async validateSwarmAccess(
        request: TierTransitionRequest,
        context: SecurityContext
    ): Promise<ValidationResult> {
        const permission: Permission = {
            resource: `swarm:${request.swarmId}`,
            action: request.operation,
            effect: PermissionEffect.ALLOW,
            scope: "team",
            priority: 1
        };

        const hasPermission = await this.authzEngine.checkPermission({
            user: context.userId!,
            permission,
            context
        });

        return {
            success: hasPermission,
            error: hasPermission ? undefined : "Insufficient swarm access permissions"
        };
    }

    private async validateTeamPermissions(
        request: TierTransitionRequest,
        context: SecurityContext
    ): Promise<ValidationResult> {
        if (this.isTeamManagementOperation(request.operation)) {
            const permission: Permission = {
                resource: `team:${request.teamId}`,
                action: "manage",
                effect: PermissionEffect.ALLOW,
                scope: "organization",
                priority: 1
            };

            const hasPermission = await this.authzEngine.checkPermission({
                user: context.userId!,
                permission,
                context
            });

            return {
                success: hasPermission,
                error: hasPermission ? undefined : "Insufficient team management permissions"
            };
        }

        return { success: true };
    }

    private async validateResourceAuthorization(
        request: TierTransitionRequest,
        context: SecurityContext
    ): Promise<ValidationResult> {
        if (request.resourceRequirements) {
            const permission: Permission = {
                resource: "resources:allocation",
                action: "allocate",
                effect: PermissionEffect.ALLOW,
                scope: "team",
                priority: 1,
                conditions: [{
                    type: "resource_limit",
                    operator: "<=",
                    value: {
                        maxCredits: request.resourceRequirements.maxCredits,
                        maxDuration: request.resourceRequirements.maxDurationMs
                    },
                    description: "Resource allocation limits"
                }]
            };

            const hasPermission = await this.authzEngine.checkPermission({
                user: context.userId!,
                permission,
                context
            });

            return {
                success: hasPermission,
                error: hasPermission ? undefined : "Insufficient resource allocation permissions"
            };
        }

        return { success: true };
    }

    private async validateGoalModification(
        request: TierTransitionRequest,
        context: SecurityContext
    ): Promise<ValidationResult> {
        if (this.isGoalModificationOperation(request.operation)) {
            const permission: Permission = {
                resource: `goal:${request.goalId}`,
                action: "modify",
                effect: PermissionEffect.ALLOW,
                scope: "swarm",
                priority: 1
            };

            const hasPermission = await this.authzEngine.checkPermission({
                user: context.userId!,
                permission,
                context
            });

            return {
                success: hasPermission,
                error: hasPermission ? undefined : "Insufficient goal modification permissions"
            };
        }

        return { success: true };
    }

    private isTeamManagementOperation(operation: string): boolean {
        return ["modify_team_structure", "assign_roles", "manage_team"].includes(operation);
    }

    private isGoalModificationOperation(operation: string): boolean {
        return ["modify_goal", "create_subtasks", "decompose_goal"].includes(operation);
    }

    private async aggregateValidationResults(
        validations: Promise<ValidationResult>[],
        operationType: string,
        request: TierTransitionRequest,
        context: SecurityContext
    ): Promise<SecurityValidationResult> {
        const results = await Promise.all(validations);
        const failed = results.filter(r => !r.success);

        if (failed.length > 0) {
            const auditEntry = await this.auditLogger.logValidationFailure(
                operationType, request, context, failed
            );
            
            return {
                success: false,
                errors: failed.map(f => f.error),
                auditEntry
            };
        }

        const auditEntry = await this.auditLogger.logValidationSuccess(
            operationType, request, context
        );

        return { success: true, auditEntry };
    }
}

/**
 * Tier 2 Security Guard - Routine and process security
 */
export class Tier2SecurityGuard implements TierSecurityGuard {
    constructor(
        private authzEngine: AuthorizationEngine,
        private auditLogger: AuditLogger
    ) {}

    async validateRequest(
        request: TierTransitionRequest,
        context: SecurityContext
    ): Promise<SecurityValidationResult> {
        const validations = [
            this.validateRoutineAccess(request, context),
            this.validateContextSecurity(request, context),
            this.validateSubroutineExecution(request, context),
            this.validateDataSensitivity(request, context)
        ];

        return this.aggregateValidationResults(validations, "tier2_validation", request, context);
    }

    private async validateRoutineAccess(
        request: TierTransitionRequest,
        context: SecurityContext
    ): Promise<ValidationResult> {
        const permission: Permission = {
            resource: `routine:${request.routineId}`,
            action: "execute",
            effect: PermissionEffect.ALLOW,
            scope: "team",
            priority: 1
        };

        const hasPermission = await this.authzEngine.checkPermission({
            user: context.userId!,
            permission,
            context
        });

        return {
            success: hasPermission,
            error: hasPermission ? undefined : "Insufficient routine execution permissions"
        };
    }

    private async validateContextSecurity(
        request: TierTransitionRequest,
        context: SecurityContext
    ): Promise<ValidationResult> {
        if (request.contextData) {
            for (const [key, value] of Object.entries(request.contextData)) {
                const dataSensitivity = this.determineSensitivity(value);
                const permission: Permission = {
                    resource: `data:${dataSensitivity}`,
                    action: "access",
                    effect: PermissionEffect.ALLOW,
                    scope: "user",
                    priority: 1
                };

                const hasPermission = await this.authzEngine.checkPermission({
                    user: context.userId!,
                    permission,
                    context
                });

                if (!hasPermission) {
                    return {
                        success: false,
                        error: `Insufficient permissions for ${dataSensitivity} data access`
                    };
                }
            }
        }

        return { success: true };
    }

    private determineSensitivity(data: unknown): DataSensitivity {
        // Basic sensitivity detection - in practice this would be more sophisticated
        if (typeof data === 'string') {
            if (data.includes('password') || data.includes('secret')) {
                return DataSensitivity.SECRET;
            }
            if (data.includes('@') || /\d{3}-\d{2}-\d{4}/.test(data)) {
                return DataSensitivity.PII;
            }
        }
        return DataSensitivity.INTERNAL;
    }

    // ... other validation methods similar to Tier1SecurityGuard
}

/**
 * Tier 3 Security Guard - Execution and tool security
 */
export class Tier3SecurityGuard implements TierSecurityGuard {
    constructor(
        private authzEngine: AuthorizationEngine,
        private auditLogger: AuditLogger,
        private sandboxManager?: SandboxManager
    ) {}

    async validateRequest(
        request: TierTransitionRequest,
        context: SecurityContext
    ): Promise<SecurityValidationResult> {
        const validations = [
            this.validateToolExecution(request, context),
            this.validateSandboxSecurity(request, context),
            this.validateOutputValidation(request, context),
            this.validateResourceAccess(request, context)
        ];

        return this.aggregateValidationResults(validations, "tier3_validation", request, context);
    }

    private async validateToolExecution(
        request: TierTransitionRequest,
        context: SecurityContext
    ): Promise<ValidationResult> {
        if (request.toolName) {
            const permission: Permission = {
                resource: `tool:${request.toolName}`,
                action: "execute",
                effect: PermissionEffect.ALLOW,
                scope: "routine",
                priority: 1,
                conditions: request.toolParameters ? [{
                    type: "tool_parameters",
                    operator: "validated",
                    value: request.toolParameters,
                    description: "Tool parameters validation"
                }] : undefined
            };

            const hasPermission = await this.authzEngine.checkPermission({
                user: context.userId!,
                permission,
                context
            });

            return {
                success: hasPermission,
                error: hasPermission ? undefined : `Insufficient permissions to execute tool: ${request.toolName}`
            };
        }

        return { success: true };
    }

    private async validateSandboxSecurity(
        request: TierTransitionRequest,
        context: SecurityContext
    ): Promise<ValidationResult> {
        if (request.requiresSandbox && this.sandboxManager) {
            const sandboxConfig = await this.sandboxManager.validateConfiguration({
                user: context.userId!,
                executionContext: request.executionContext,
                securityLevel: context.clearanceLevel
            });

            return {
                success: sandboxConfig.isValid,
                error: sandboxConfig.isValid ? undefined : sandboxConfig.error
            };
        }

        return { success: true };
    }

    // ... other validation methods
}
```

## AI Security Implementation

### **AI Security Manager**

```typescript
/**
 * AI Security Manager - Comprehensive AI threat detection and mitigation
 */
export class AISecurityManager {
    private promptInjectionDetector: PromptInjectionDetector;
    private hallucinationValidator: HallucinationValidator;
    private biasDetector: BiasDetector;
    private outputSanitizer: OutputSanitizer;

    constructor(config: AISecurityConfig) {
        this.promptInjectionDetector = new PromptInjectionDetector(config.promptInjection);
        this.hallucinationValidator = new HallucinationValidator(config.hallucination);
        this.biasDetector = new BiasDetector(config.bias);
        this.outputSanitizer = new OutputSanitizer(config.outputSanitization);
    }

    async validateInput(
        input: string,
        context: SecurityContext
    ): Promise<AISecurityValidationResult> {
        try {
            const validations = await Promise.all([
                this.promptInjectionDetector.detect(input, context),
                this.biasDetector.detectInputBias(input, context),
                this.detectSensitiveData(input, context)
            ]);

            const threats = validations.filter(v => v.threatDetected);
            
            if (threats.length > 0) {
                const sanitizedInput = await this.sanitizeInput(input, threats);
                
                return {
                    success: false,
                    threats: threats.map(t => t.threat!),
                    sanitizedInput,
                    auditEntry: this.createAuditEntry(
                        context, 
                        "ai_input_validation", 
                        "threats_detected",
                        { threats: threats.map(t => t.threat!.type) }
                    )
                };
            }

            return {
                success: true,
                sanitizedInput: input,
                auditEntry: this.createAuditEntry(context, "ai_input_validation", "clean")
            };

        } catch (error) {
            return {
                success: false,
                error: `AI input validation error: ${(error as Error).message}`,
                sanitizedInput: "",
                auditEntry: this.createAuditEntry(
                    context, 
                    "ai_input_validation", 
                    "validation_error",
                    { error: (error as Error).message }
                )
            };
        }
    }

    async validateOutput(
        output: string,
        context: SecurityContext,
        originalInput: string
    ): Promise<AISecurityValidationResult> {
        try {
            const validations = await Promise.all([
                this.hallucinationValidator.validate(output, originalInput, context),
                this.biasDetector.detectOutputBias(output, context),
                this.outputSanitizer.checkForSensitiveData(output, context)
            ]);

            const issues = validations.filter(v => v.issueDetected);
            
            if (issues.length > 0) {
                const sanitizedOutput = await this.sanitizeOutput(output, issues);
                
                return {
                    success: false,
                    issues: issues.map(i => i.issue!),
                    sanitizedOutput,
                    auditEntry: this.createAuditEntry(
                        context, 
                        "ai_output_validation", 
                        "issues_detected",
                        { issues: issues.map(i => i.issue!.type) }
                    )
                };
            }

            return {
                success: true,
                sanitizedOutput: output,
                auditEntry: this.createAuditEntry(context, "ai_output_validation", "clean")
            };

        } catch (error) {
            return {
                success: false,
                error: `AI output validation error: ${(error as Error).message}`,
                sanitizedOutput: "[Output validation failed - content blocked]",
                auditEntry: this.createAuditEntry(
                    context, 
                    "ai_output_validation", 
                    "validation_error",
                    { error: (error as Error).message }
                )
            };
        }
    }

    private async detectSensitiveData(
        input: string, 
        context: SecurityContext
    ): Promise<AIThreatDetectionResult> {
        // Note: For production sensitive data handling, integrate with
        // the Secrets Management Service for encryption/decryption
        // See: ../../core-services/secrets-management.md
        const sensitivePatterns = [
            { pattern: /\b\d{3}-\d{2}-\d{4}\b/, type: "ssn" },
            { pattern: /\b4[0-9]{12}(?:[0-9]{3})?\b/, type: "credit_card" },
            { pattern: /\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b/, type: "email" },
            { pattern: /(?:password|pwd|pass)\s*[:=]\s*\S+/i, type: "password" }
        ];

        const matches = sensitivePatterns.filter(p => p.pattern.test(input));
        
        return {
            threatDetected: matches.length > 0,
            threat: matches.length > 0 ? {
                type: "sensitive_data_exposure",
                confidence: 0.9,
                evidence: { detectedTypes: matches.map(m => m.type) }
            } : undefined
        };
    }

    private async sanitizeInput(input: string, threats: AIThreatDetectionResult[]): Promise<string> {
        let sanitized = input;
        
        for (const threat of threats) {
            if (threat.threat?.type === "prompt_injection") {
                sanitized = sanitized.replace(/[<>{}[\]]/g, '');
                sanitized = sanitized.replace(/\b(ignore|forget|disregard)\s+(previous|above|all)\b/gi, '[FILTERED]');
            } else if (threat.threat?.type === "sensitive_data_exposure") {
                sanitized = sanitized.replace(/\b\d{3}-\d{2}-\d{4}\b/g, 'XXX-XX-XXXX');
                sanitized = sanitized.replace(/\b4[0-9]{12}(?:[0-9]{3})?\b/g, 'XXXX-XXXX-XXXX-XXXX');
                sanitized = sanitized.replace(/(?:password|pwd|pass)\s*[:=]\s*\S+/gi, 'password=[REDACTED]');
            }
        }
        
        return sanitized;
    }

    private async sanitizeOutput(output: string, issues: AIIssueDetectionResult[]): Promise<string> {
        let sanitized = output;
        
        for (const issue of issues) {
            if (issue.issue?.type === "bias_detected") {
                sanitized = `[Content reviewed for potential bias] ${sanitized}`;
            } else if (issue.issue?.type === "hallucination_detected") {
                sanitized = `[Information requires verification] ${sanitized}`;
            } else if (issue.issue?.type === "sensitive_data_detected") {
                // Apply aggressive sanitization for sensitive data in output
                sanitized = sanitized.replace(/\b\d{3}-\d{2}-\d{4}\b/g, '[SSN-REDACTED]');
                sanitized = sanitized.replace(/\b4[0-9]{12}(?:[0-9]{3})?\b/g, '[CARD-REDACTED]');
            }
        }
        
        return sanitized;
    }

    private createAuditEntry(
        context: SecurityContext, 
        operation: string, 
        result: string, 
        details?: Record<string, unknown>
    ): AuditEntry {
        return {
            timestamp: new Date(),
            tier: context.targetTier,
            operation,
            resource: "ai_processing",
            result,
            details,
            userId: context.userId
        };
    }
}
```

### **Prompt Injection Detection**

```typescript
/**
 * Prompt Injection Detector using multiple detection strategies
 */
export class PromptInjectionDetector {
    private patterns: RegExp[];
    private keywordList: string[];
    private mlModel?: MLModel;

    constructor(config: PromptInjectionConfig) {
        this.patterns = config.patterns.map(p => new RegExp(p, 'i'));
        this.keywordList = config.suspiciousKeywords || [];
        this.mlModel = config.modelPath ? new MLModel(config.modelPath) : undefined;
    }

    async detect(
        input: string,
        context: SecurityContext
    ): Promise<AIThreatDetectionResult> {
        const detectionMethods = [
            this.patternBasedDetection(input),
            this.keywordBasedDetection(input),
            this.contextualAnalysis(input, context)
        ];

        if (this.mlModel) {
            detectionMethods.push(this.mlBasedDetection(input));
        }

        const results = await Promise.all(detectionMethods);
        const highestThreat = results.reduce((max, current) => 
            (current.confidence || 0) > (max.confidence || 0) ? current : max
        );

        const threatDetected = (highestThreat.confidence || 0) > 0.7;

        return {
            threatDetected,
            threat: threatDetected ? {
                type: "prompt_injection",
                confidence: highestThreat.confidence!,
                evidence: {
                    detectionMethods: results.filter(r => r.detected).map(r => r.method),
                    suspiciousPatterns: highestThreat.patterns || []
                }
            } : undefined
        };
    }

    private async patternBasedDetection(input: string): Promise<DetectionResult> {
        const matches = this.patterns.filter(pattern => pattern.test(input));
        return {
            detected: matches.length > 0,
            confidence: matches.length > 0 ? 0.8 : 0.1,
            method: "pattern_matching",
            patterns: matches.map(m => m.source)
        };
    }

    private async keywordBasedDetection(input: string): Promise<DetectionResult> {
        const foundKeywords = this.keywordList.filter(keyword => 
            input.toLowerCase().includes(keyword.toLowerCase())
        );
        
        return {
            detected: foundKeywords.length > 2, // Require multiple keywords
            confidence: Math.min(foundKeywords.length * 0.2, 0.9),
            method: "keyword_detection",
            patterns: foundKeywords
        };
    }

    private async contextualAnalysis(input: string, context: SecurityContext): Promise<DetectionResult> {
        // Analyze input in context of user permissions and clearance level
        const riskFactors = [];
        
        if (input.length > 1000) riskFactors.push("excessive_length");
        if (/[<>{}[\]]/g.test(input)) riskFactors.push("special_characters");
        if (context.clearanceLevel === SecurityClearance.PUBLIC && input.includes("admin")) {
            riskFactors.push("privilege_escalation_attempt");
        }

        return {
            detected: riskFactors.length > 1,
            confidence: Math.min(riskFactors.length * 0.3, 0.8),
            method: "contextual_analysis",
            patterns: riskFactors
        };
    }

    private async mlBasedDetection(input: string): Promise<DetectionResult> {
        if (!this.mlModel) {
            return { detected: false, confidence: 0, method: "ml_detection" };
        }

        const score = await this.mlModel.predict(input);
        return {
            detected: score > 0.7,
            confidence: score,
            method: "ml_detection"
        };
    }
}
```

## Security Context Management

### **Security Context Manager Implementation**

```typescript
/**
 * Security Context Manager - Creates, validates, and propagates security contexts
 */
export class SecurityContextManager {
    private contextCache: Map<string, SecurityContext>;
    private config: SecurityContextConfig;

    constructor(config: SecurityContextConfig) {
        this.config = config;
        this.contextCache = new Map();
    }

    async createOrValidate(request: SecurityContextCreationRequest): Promise<SecurityContext> {
        // Check if valid context already exists
        const existingContext = this.contextCache.get(request.sessionId);
        if (existingContext && await this.isContextValid(existingContext)) {
            return this.updateContextForOperation(existingContext, request);
        }

        // Create new context
        return this.create(request);
    }

    async create(request: SecurityContextCreationRequest): Promise<SecurityContext> {
        const context: SecurityContext = {
            requesterTier: request.requesterTier,
            targetTier: request.targetTier,
            operation: request.operation,
            clearanceLevel: await this.determineClearanceLevel(request.user),
            permissions: await this.gatherPermissions(request.user, request.operation),
            auditTrail: [{
                timestamp: new Date(),
                tier: request.requesterTier,
                operation: "context_creation",
                resource: `context:${request.operation}`,
                result: "success",
                userId: request.user.id
            }],
            encryptionRequired: this.requiresEncryption(request),
            signatureRequired: this.requiresSignature(request),
            sessionId: request.sessionId,
            timestamp: new Date(),
            userId: request.user.id
        };

        // Cache the context
        this.contextCache.set(request.sessionId, context);

        return context;
    }

    async propagate(
        parentContext: SecurityContext,
        targetTier: Tier,
        operation: string
    ): Promise<SecurityContext> {
        // Create child context with constrained permissions
        const childContext: SecurityContext = {
            ...parentContext,
            requesterTier: parentContext.targetTier,
            targetTier: targetTier,
            operation: operation,
            permissions: this.constrainPermissions(parentContext.permissions, targetTier),
            auditTrail: [
                ...parentContext.auditTrail,
                {
                    timestamp: new Date(),
                    tier: targetTier,
                    operation: "context_propagation",
                    resource: `context:${operation}`,
                    result: "success",
                    details: { 
                        parentSessionId: parentContext.sessionId,
                        constrainedPermissions: this.constrainPermissions(parentContext.permissions, targetTier).length
                    },
                    userId: parentContext.userId
                }
            ],
            timestamp: new Date()
        };

        return childContext;
    }

    private async isContextValid(context: SecurityContext): Promise<boolean> {
        const age = Date.now() - context.timestamp.getTime();
        const maxAge = this.config.maxContextAge || 300000; // 5 minutes default
        
        return age < maxAge && 
               context.permissions.length > 0 &&
               context.auditTrail.length > 0;
    }

    private updateContextForOperation(
        context: SecurityContext, 
        request: SecurityContextCreationRequest
    ): SecurityContext {
        return {
            ...context,
            operation: request.operation,
            targetTier: request.targetTier,
            timestamp: new Date()
        };
    }

    private async determineClearanceLevel(user: User): Promise<SecurityClearance> {
        return user.clearanceLevel || SecurityClearance.INTERNAL;
    }

    private async gatherPermissions(user: User, operation: string): Promise<Permission[]> {
        return user.permissions.filter(p => 
            this.isRelevantPermission(p, operation)
        );
    }

    private requiresEncryption(request: SecurityContextCreationRequest): boolean {
        return request.dataSensitivity === DataSensitivity.SECRET ||
               request.dataSensitivity === DataSensitivity.PII ||
               request.operation.includes("sensitive");
    }

    private requiresSignature(request: SecurityContextCreationRequest): boolean {
        return ["modify", "delete", "create", "deploy", "terminate"].some(action => 
            request.operation.includes(action)
        );
    }

    private constrainPermissions(permissions: Permission[], targetTier: Tier): Permission[] {
        // Apply principle of least privilege when propagating to child tiers
        return permissions.filter(p => this.isPermissionValidForTier(p, targetTier))
                         .map(p => ({ ...p, scope: this.narrowScope(p.scope, targetTier) }));
    }

    private isRelevantPermission(permission: Permission, operation: string): boolean {
        return permission.resource.includes(operation) || 
               permission.action === "all" ||
               operation.includes(permission.action);
    }

    private isPermissionValidForTier(permission: Permission, tier: Tier): boolean {
        // Define tier-specific permission constraints
        const tierPermissions = {
            [Tier.TIER_1]: ["swarm", "team", "goal", "resources"],
            [Tier.TIER_2]: ["routine", "context", "subroutine", "data"],
            [Tier.TIER_3]: ["tool", "step", "sandbox", "output"]
        };

        const allowedResources = tierPermissions[tier] || [];
        return allowedResources.some(resource => permission.resource.includes(resource));
    }

    private narrowScope(scope: string, tier: Tier): string {
        // Narrow scope when moving to lower tiers
        const scopeHierarchy = ["organization", "team", "swarm", "routine", "step"];
        const tierMinScope = {
            [Tier.TIER_1]: "team",
            [Tier.TIER_2]: "routine", 
            [Tier.TIER_3]: "step"
        };

        const minScope = tierMinScope[tier];
        const currentIndex = scopeHierarchy.indexOf(scope);
        const minIndex = scopeHierarchy.indexOf(minScope);

        return currentIndex < minIndex ? minScope : scope;
    }
}
```

## Integration with Execution Pipeline

### **Security Hooks for Execution**

```typescript
/**
 * Security hooks that integrate with the execution pipeline
 */
export class ExecutionSecurityHooks {
    constructor(private securityManager: SecurityManager) {}

    async beforeExecution(
        request: ExecutionRequest,
        context: SecurityContext
    ): Promise<SecurityCheckResult> {
        const validation = await this.securityManager.validateTierTransition({
            sourceTier: request.sourceTier,
            targetTier: request.targetTier,
            operation: request.operation,
            credentials: request.credentials,
            resourceRequirements: request.resourceRequirements,
            sessionId: context.sessionId,
            swarmId: request.swarmId,
            routineId: request.routineId,
            toolName: request.toolName,
            aiInput: request.aiInput
        });

        return {
            approved: validation.success,
            modifiedRequest: validation.success ? request : undefined,
            securityContext: validation.success ? context : undefined,
            auditEntry: validation.auditEntry,
            errors: validation.success ? [] : [validation.error!]
        };
    }

    async afterExecution(
        result: ExecutionResult,
        context: SecurityContext
    ): Promise<SecurityCheckResult> {
        if (result.output && typeof result.output === 'string') {
            const outputValidation = await this.securityManager['aiSecurityManager'].validateOutput(
                result.output,
                context,
                result.originalInput || ""
            );

            return {
                approved: outputValidation.success,
                modifiedResult: outputValidation.success ? result : {
                    ...result,
                    output: outputValidation.sanitizedOutput
                },
                auditEntry: outputValidation.auditEntry,
                errors: outputValidation.success ? [] : [outputValidation.error!]
            };
        }

        return { approved: true, modifiedResult: result, errors: [] };
    }

    async detectSecurityIncident(
        execution: ExecutionContext,
        anomaly: SecurityAnomaly
    ): Promise<IncidentDetectionResult> {
        const incident: SecurityIncident = {
            id: `incident_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
            type: anomaly.type,
            severity: this.calculateSeverity(anomaly),
            description: anomaly.description,
            context: execution.securityContext,
            timestamp: new Date(),
            evidence: anomaly.evidence
        };

        const response = await this.securityManager.handleSecurityIncident(incident);

        return {
            incidentDetected: true,
            incident,
            response,
            actionRequired: response.response.some(action => action.requiresImmediate)
        };
    }

    private calculateSeverity(anomaly: SecurityAnomaly): SecuritySeverity {
        if (anomaly.type.includes("injection") || anomaly.type.includes("breach")) {
            return SecuritySeverity.HIGH;
        }
        if (anomaly.type.includes("unauthorized") || anomaly.type.includes("escalation")) {
            return SecuritySeverity.MEDIUM;
        }
        return SecuritySeverity.LOW;
    }
}
```

## Related Documentation

### **Execution Architecture**
- **[Security Overview](README.md)** - High-level security architecture
- **[Security Boundaries](security-boundaries.md)** - Trust models and permission systems
- **[Main Execution Architecture](../README.md)** - Complete architectural overview
- **[Types System](../types/core-types.ts)** - Security interface definitions
- **[Communication Patterns](../communication/communication-patterns.md)** - Security integration with communication
- **[Error Handling](../resilience/error-propagation.md)** - Security error handling and recovery

### **Core Services Integration**
- **[Secrets Management](../../core-services/secrets-management.md)** - Encryption and secure storage for sensitive run inputs/outputs
- **[Notification Service](../../core-services/notification-service.md)** - Security event notifications and alerts
- **[Event Bus System](../../core-services/event-bus-system.md)** - Security event distribution across tiers

This implementation guide provides the concrete patterns and code examples needed to build Vrooli's comprehensive security architecture. 