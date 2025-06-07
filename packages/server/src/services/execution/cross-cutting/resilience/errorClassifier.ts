/**
 * Error Classification Engine
 * Implements systematic decision tree for error classification and recovery strategy selection
 * Enables emergent resilience intelligence through pattern learning
 */

import type {
    ErrorClassification,
    ErrorContext,
    ErrorSeverity,
    ErrorCategory,
    ErrorRecoverability,
    RecoveryStrategy,
    ResilienceEvent,
    ResilienceEventType,
    ErrorPattern,
    PatternCondition,
    ConditionOperator,
} from "@vrooli/shared";
import {
    ErrorSeverity as Severity,
    ErrorCategory as Category,
    ErrorRecoverability as Recoverability,
    RecoveryStrategy as Strategy,
    ResilienceEventType as EventType,
} from "@vrooli/shared";
import { logger } from "../../../../events/logger.js";

/**
 * Classification confidence thresholds
 */
const CONFIDENCE_THRESHOLDS = {
    HIGH: 0.8,
    MEDIUM: 0.6,
    LOW: 0.4,
} as const;

/**
 * Error classification engine implementing systematic decision tree
 */
export class ErrorClassifier {
    private patterns: ErrorPattern[] = [];
    private classificationHistory: Map<string, ErrorClassification[]> = new Map();
    private readonly maxHistorySize = 1000;

    /**
     * Classify error using systematic decision tree
     * Decision path: System functional? → Impact scope? → Data risk? → Recovery possible?
     */
    async classify(
        error: Error,
        context: ErrorContext,
    ): Promise<ErrorClassification> {
        const startTime = performance.now();

        try {
            // Extract error features for classification
            const features = this.extractErrorFeatures(error, context);
            
            // Apply systematic decision tree
            const classification = this.applyDecisionTree(features, context);
            
            // Enhance with pattern matching
            const enhancedClassification = await this.enhanceWithPatterns(
                classification,
                features,
                context,
            );
            
            // Store for learning
            this.storeClassification(features.errorSignature, enhancedClassification);
            
            const duration = performance.now() - startTime;
            logger.debug("Error classified", {
                errorType: error.constructor.name,
                classification: enhancedClassification,
                duration: `${duration.toFixed(2)}ms`,
                tier: context.tier,
                component: context.component,
            });

            return enhancedClassification;
        } catch (classificationError) {
            logger.error("Error during classification", classificationError);
            
            // Fallback classification
            return this.createFallbackClassification(error, context);
        }
    }

    /**
     * Apply systematic decision tree for error classification
     */
    private applyDecisionTree(
        features: ErrorFeatures,
        context: ErrorContext,
    ): ErrorClassification {
        let severity: ErrorSeverity;
        let category: ErrorCategory;
        let recoverability: ErrorRecoverability;
        let systemFunctional: boolean;
        let multipleComponentsAffected: boolean;
        let dataRisk: boolean;
        let securityRisk: boolean;
        let confidenceScore = 0.9; // High confidence for rule-based decisions

        // Step 1: Assess system functionality
        systemFunctional = this.assessSystemFunctionality(features, context);
        
        if (!systemFunctional) {
            severity = Severity.FATAL;
            category = Category.SYSTEM;
            recoverability = Recoverability.MANUAL;
            multipleComponentsAffected = true;
        } else {
            // Step 2: Assess impact scope
            multipleComponentsAffected = this.assessImpactScope(features, context);
            
            if (multipleComponentsAffected) {
                severity = Severity.CRITICAL;
            } else {
                // Step 3: Assess data and security risks
                dataRisk = this.assessDataRisk(features, context);
                securityRisk = this.assessSecurityRisk(features, context);
                
                if (dataRisk || securityRisk) {
                    severity = Severity.CRITICAL;
                    category = securityRisk ? Category.SECURITY : Category.DATA;
                } else {
                    // Step 4: Determine error category and severity
                    category = this.determineCategory(features, context);
                    severity = this.determineSeverity(features, context, category);
                }
            }
            
            // Step 5: Assess recoverability
            recoverability = this.assessRecoverability(features, context, severity);
        }

        // Final adjustments based on context
        if (context.attemptCount > 3) {
            severity = this.escalateSeverity(severity);
            recoverability = this.degradeRecoverability(recoverability);
            confidenceScore *= 0.9; // Slightly lower confidence after multiple attempts
        }

        return {
            severity,
            category,
            recoverability,
            systemFunctional,
            multipleComponentsAffected,
            dataRisk,
            securityRisk,
            confidenceScore,
            timestamp: new Date(),
            metadata: {
                features,
                decisionPath: this.buildDecisionPath(
                    systemFunctional,
                    multipleComponentsAffected,
                    dataRisk,
                    securityRisk,
                ),
                contextualFactors: {
                    tier: context.tier,
                    attemptCount: context.attemptCount,
                    previousStrategies: context.previousStrategies,
                },
            },
        };
    }

    /**
     * Assess if system is still functional
     */
    private assessSystemFunctionality(
        features: ErrorFeatures,
        context: ErrorContext,
    ): boolean {
        // Critical system errors
        if (features.errorType.includes("OutOfMemory") ||
            features.errorType.includes("StackOverflow") ||
            features.isInfrastructureError) {
            return false;
        }

        // Database connection failures
        if (features.isDatabaseError && features.errorMessage.includes("connection")) {
            return false;
        }

        // Multiple tier failures
        if (context.systemState.tierFailures && 
            (context.systemState.tierFailures as number) > 1) {
            return false;
        }

        // Service unavailable patterns
        if (features.httpStatusCode === 503 || 
            features.httpStatusCode === 502) {
            return false;
        }

        return true;
    }

    /**
     * Assess if multiple components are affected
     */
    private assessImpactScope(
        features: ErrorFeatures,
        context: ErrorContext,
    ): boolean {
        // Network errors often affect multiple components
        if (features.isNetworkError) {
            return true;
        }

        // Authentication/authorization errors affect multiple components
        if (features.isAuthError) {
            return true;
        }

        // Rate limiting affects multiple operations
        if (features.isRateLimitError) {
            return true;
        }

        // Check system state for cascading failures
        if (context.systemState.componentFailures &&
            (context.systemState.componentFailures as number) > 2) {
            return true;
        }

        return false;
    }

    /**
     * Assess data corruption or loss risk
     */
    private assessDataRisk(
        features: ErrorFeatures,
        context: ErrorContext,
    ): boolean {
        // Database write operations
        if (features.isDatabaseError && 
            (context.operation.includes("write") || 
             context.operation.includes("update") ||
             context.operation.includes("delete"))) {
            return true;
        }

        // File system errors during writes
        if (features.isFileSystemError && 
            context.operation.includes("write")) {
            return true;
        }

        // Transaction rollback scenarios
        if (features.errorMessage.includes("transaction") &&
            features.errorMessage.includes("rollback")) {
            return true;
        }

        return false;
    }

    /**
     * Assess security risk
     */
    private assessSecurityRisk(
        features: ErrorFeatures,
        context: ErrorContext,
    ): boolean {
        // Authentication failures
        if (features.isAuthError) {
            return true;
        }

        // Authorization violations
        if (features.httpStatusCode === 403) {
            return true;
        }

        // Injection attack patterns
        if (features.errorMessage.includes("injection") ||
            features.errorMessage.includes("XSS") ||
            features.errorMessage.includes("CSRF")) {
            return true;
        }

        // Suspicious request patterns
        if (context.userContext && 
            context.systemState.suspiciousActivity) {
            return true;
        }

        return false;
    }

    /**
     * Determine error category
     */
    private determineCategory(
        features: ErrorFeatures,
        context: ErrorContext,
    ): ErrorCategory {
        if (features.isNetworkError || features.isTimeoutError) {
            return Category.TRANSIENT;
        }
        
        if (features.isResourceError || features.isRateLimitError) {
            return Category.RESOURCE;
        }
        
        if (features.isDatabaseError || features.isFileSystemError) {
            return Category.SYSTEM;
        }
        
        if (features.isValidationError) {
            return Category.LOGIC;
        }
        
        if (features.isConfigurationError) {
            return Category.CONFIGURATION;
        }
        
        if (features.isAuthError) {
            return Category.SECURITY;
        }
        
        return Category.UNKNOWN;
    }

    /**
     * Determine error severity
     */
    private determineSeverity(
        features: ErrorFeatures,
        context: ErrorContext,
        category: ErrorCategory,
    ): ErrorSeverity {
        // User-facing operations are more severe
        const isUserFacing = context.userContext !== undefined;
        
        // Critical errors
        if (features.errorType.includes("Critical") ||
            features.httpStatusCode >= 500) {
            return isUserFacing ? Severity.CRITICAL : Severity.ERROR;
        }
        
        // Client errors
        if (features.httpStatusCode >= 400 && features.httpStatusCode < 500) {
            return Severity.WARNING;
        }
        
        // Transient errors are less severe unless repeated
        if (category === Category.TRANSIENT) {
            return context.attemptCount > 2 ? Severity.ERROR : Severity.WARNING;
        }
        
        // Resource errors escalate with attempts
        if (category === Category.RESOURCE) {
            return context.attemptCount > 1 ? Severity.ERROR : Severity.WARNING;
        }
        
        return Severity.ERROR;
    }

    /**
     * Assess error recoverability
     */
    private assessRecoverability(
        features: ErrorFeatures,
        context: ErrorContext,
        severity: ErrorSeverity,
    ): ErrorRecoverability {
        if (severity === Severity.FATAL) {
            return Recoverability.NONE;
        }
        
        if (severity === Severity.CRITICAL) {
            return Recoverability.MANUAL;
        }
        
        // Transient errors are usually automatically recoverable
        if (features.isNetworkError || features.isTimeoutError) {
            return context.attemptCount < 3 ? 
                Recoverability.AUTOMATIC : 
                Recoverability.PARTIAL;
        }
        
        // Resource errors may be recoverable with backoff
        if (features.isResourceError || features.isRateLimitError) {
            return Recoverability.AUTOMATIC;
        }
        
        // Configuration errors need manual intervention
        if (features.isConfigurationError) {
            return Recoverability.MANUAL;
        }
        
        // Database errors depend on type
        if (features.isDatabaseError) {
            return features.errorMessage.includes("connection") ?
                Recoverability.AUTOMATIC :
                Recoverability.PARTIAL;
        }
        
        return Recoverability.PARTIAL;
    }

    /**
     * Enhance classification with pattern matching
     */
    private async enhanceWithPatterns(
        classification: ErrorClassification,
        features: ErrorFeatures,
        context: ErrorContext,
    ): Promise<ErrorClassification> {
        const matchingPatterns = this.findMatchingPatterns(features, context);
        
        if (matchingPatterns.length === 0) {
            return classification;
        }
        
        // Use the best matching pattern to refine classification
        const bestPattern = matchingPatterns.reduce((best, current) =>
            best.confidence > current.confidence ? best : current,
        );
        
        // Adjust confidence based on pattern match
        const patternConfidence = bestPattern.confidence;
        const adjustedConfidence = (classification.confidenceScore + patternConfidence) / 2;
        
        return {
            ...classification,
            confidenceScore: adjustedConfidence,
            metadata: {
                ...classification.metadata,
                matchingPatterns: matchingPatterns.map(p => ({
                    id: p.id,
                    name: p.name,
                    confidence: p.confidence,
                })),
                bestPattern: {
                    id: bestPattern.id,
                    name: bestPattern.name,
                    confidence: bestPattern.confidence,
                },
            },
        };
    }

    /**
     * Find matching error patterns
     */
    private findMatchingPatterns(
        features: ErrorFeatures,
        context: ErrorContext,
    ): ErrorPattern[] {
        return this.patterns
            .map(pattern => ({
                ...pattern,
                confidence: this.calculatePatternMatch(pattern, features, context),
            }))
            .filter(pattern => pattern.confidence > CONFIDENCE_THRESHOLDS.LOW)
            .sort((a, b) => b.confidence - a.confidence);
    }

    /**
     * Calculate how well a pattern matches current error
     */
    private calculatePatternMatch(
        pattern: ErrorPattern,
        features: ErrorFeatures,
        context: ErrorContext,
    ): number {
        let totalWeight = 0;
        let matchedWeight = 0;
        
        for (const condition of pattern.triggerConditions) {
            totalWeight += condition.weight;
            
            if (this.evaluateCondition(condition, features, context)) {
                matchedWeight += condition.weight;
            }
        }
        
        return totalWeight > 0 ? matchedWeight / totalWeight : 0;
    }

    /**
     * Evaluate a pattern condition
     */
    private evaluateCondition(
        condition: PatternCondition,
        features: ErrorFeatures,
        context: ErrorContext,
    ): boolean {
        const value = this.getValueFromContext(condition.field, features, context);
        
        return this.compareValues(value, condition.value, condition.operator, condition.tolerance);
    }

    /**
     * Get value from features or context
     */
    private getValueFromContext(
        field: string,
        features: ErrorFeatures,
        context: ErrorContext,
    ): unknown {
        // Check features first
        if (field in features) {
            return (features as any)[field];
        }
        
        // Check context
        if (field in context) {
            return (context as any)[field];
        }
        
        // Check nested fields
        if (field.includes(".")) {
            const parts = field.split(".");
            let obj: any = { ...features, ...context };
            
            for (const part of parts) {
                if (obj && typeof obj === "object" && part in obj) {
                    obj = obj[part];
                } else {
                    return undefined;
                }
            }
            
            return obj;
        }
        
        return undefined;
    }

    /**
     * Compare values using operator
     */
    private compareValues(
        actual: unknown,
        expected: unknown,
        operator: ConditionOperator,
        tolerance?: number,
    ): boolean {
        switch (operator) {
            case "EQUALS":
                return actual === expected;
            case "NOT_EQUALS":
                return actual !== expected;
            case "GREATER_THAN":
                return typeof actual === "number" && typeof expected === "number" &&
                    actual > expected;
            case "LESS_THAN":
                return typeof actual === "number" && typeof expected === "number" &&
                    actual < expected;
            case "CONTAINS":
                return typeof actual === "string" && typeof expected === "string" &&
                    actual.includes(expected);
            case "IN":
                return Array.isArray(expected) && expected.includes(actual);
            case "NOT_IN":
                return Array.isArray(expected) && !expected.includes(actual);
            case "REGEX_MATCH":
                return typeof actual === "string" && typeof expected === "string" &&
                    new RegExp(expected).test(actual);
            default:
                return false;
        }
    }

    /**
     * Extract error features for classification
     */
    private extractErrorFeatures(error: Error, context: ErrorContext): ErrorFeatures {
        const errorMessage = error.message.toLowerCase();
        const errorType = error.constructor.name;
        const stack = error.stack || "";
        
        return {
            errorType,
            errorMessage: error.message,
            errorSignature: this.generateErrorSignature(error, context),
            
            // Network-related
            isNetworkError: this.isNetworkError(error),
            isTimeoutError: this.isTimeoutError(error),
            httpStatusCode: this.extractHttpStatusCode(error),
            
            // Database-related
            isDatabaseError: this.isDatabaseError(error),
            
            // File system
            isFileSystemError: this.isFileSystemError(error),
            
            // Authentication/Authorization
            isAuthError: this.isAuthError(error),
            
            // Resource-related
            isResourceError: this.isResourceError(error),
            isRateLimitError: this.isRateLimitError(error),
            
            // Infrastructure
            isInfrastructureError: this.isInfrastructureError(error),
            
            // Validation
            isValidationError: this.isValidationError(error),
            
            // Configuration
            isConfigurationError: this.isConfigurationError(error),
            
            // Context
            stackDepth: stack.split("\n").length,
            occurredInTier: context.tier,
            operationType: this.categorizeOperation(context.operation),
        };
    }

    /**
     * Generate unique signature for error pattern matching
     */
    private generateErrorSignature(error: Error, context: ErrorContext): string {
        const components = [
            error.constructor.name,
            this.normalizeErrorMessage(error.message),
            context.tier,
            context.component,
            this.categorizeOperation(context.operation),
        ];
        
        return components.join("::");
    }

    /**
     * Normalize error message for pattern matching
     */
    private normalizeErrorMessage(message: string): string {
        return message
            .toLowerCase()
            .replace(/\d+/g, "N") // Replace numbers with N
            .replace(/['"]/g, "") // Remove quotes
            .replace(/\s+/g, " ") // Normalize whitespace
            .trim();
    }

    /**
     * Error type detection methods
     */
    private isNetworkError(error: Error): boolean {
        const message = error.message.toLowerCase();
        return message.includes("network") ||
            message.includes("connection") ||
            message.includes("socket") ||
            error.constructor.name.includes("Network");
    }

    private isTimeoutError(error: Error): boolean {
        const message = error.message.toLowerCase();
        return message.includes("timeout") ||
            message.includes("timed out") ||
            error.constructor.name.includes("Timeout");
    }

    private extractHttpStatusCode(error: Error): number | undefined {
        const match = error.message.match(/\b[45]\d{2}\b/);
        return match ? parseInt(match[0], 10) : undefined;
    }

    private isDatabaseError(error: Error): boolean {
        const message = error.message.toLowerCase();
        return message.includes("database") ||
            message.includes("sql") ||
            message.includes("prisma") ||
            message.includes("postgres");
    }

    private isFileSystemError(error: Error): boolean {
        const message = error.message.toLowerCase();
        return message.includes("file") ||
            message.includes("directory") ||
            message.includes("enoent") ||
            message.includes("eacces");
    }

    private isAuthError(error: Error): boolean {
        const message = error.message.toLowerCase();
        return message.includes("auth") ||
            message.includes("unauthorized") ||
            message.includes("forbidden") ||
            message.includes("token");
    }

    private isResourceError(error: Error): boolean {
        const message = error.message.toLowerCase();
        return message.includes("memory") ||
            message.includes("limit") ||
            message.includes("quota") ||
            message.includes("capacity");
    }

    private isRateLimitError(error: Error): boolean {
        const message = error.message.toLowerCase();
        return message.includes("rate limit") ||
            message.includes("too many requests") ||
            message.includes("429");
    }

    private isInfrastructureError(error: Error): boolean {
        const message = error.message.toLowerCase();
        return message.includes("service unavailable") ||
            message.includes("infrastructure") ||
            message.includes("503") ||
            message.includes("502");
    }

    private isValidationError(error: Error): boolean {
        const message = error.message.toLowerCase();
        return message.includes("validation") ||
            message.includes("invalid") ||
            message.includes("schema") ||
            error.constructor.name.includes("Validation");
    }

    private isConfigurationError(error: Error): boolean {
        const message = error.message.toLowerCase();
        return message.includes("config") ||
            message.includes("environment") ||
            message.includes("missing") ||
            message.includes("undefined");
    }

    private categorizeOperation(operation: string): string {
        const op = operation.toLowerCase();
        
        if (op.includes("read") || op.includes("get") || op.includes("fetch")) {
            return "read";
        }
        if (op.includes("write") || op.includes("post") || op.includes("create")) {
            return "write";
        }
        if (op.includes("update") || op.includes("put") || op.includes("patch")) {
            return "update";
        }
        if (op.includes("delete") || op.includes("remove")) {
            return "delete";
        }
        
        return "other";
    }

    /**
     * Helper methods for severity escalation
     */
    private escalateSeverity(severity: ErrorSeverity): ErrorSeverity {
        switch (severity) {
            case Severity.INFO: return Severity.WARNING;
            case Severity.WARNING: return Severity.ERROR;
            case Severity.ERROR: return Severity.CRITICAL;
            case Severity.CRITICAL: return Severity.FATAL;
            default: return severity;
        }
    }

    private degradeRecoverability(recoverability: ErrorRecoverability): ErrorRecoverability {
        switch (recoverability) {
            case Recoverability.AUTOMATIC: return Recoverability.PARTIAL;
            case Recoverability.PARTIAL: return Recoverability.MANUAL;
            default: return recoverability;
        }
    }

    /**
     * Build decision path for debugging
     */
    private buildDecisionPath(
        systemFunctional: boolean,
        multipleComponentsAffected: boolean,
        dataRisk: boolean,
        securityRisk: boolean,
    ): string[] {
        const path = [`system_functional: ${systemFunctional}`];
        
        if (systemFunctional) {
            path.push(`multiple_components_affected: ${multipleComponentsAffected}`);
            
            if (!multipleComponentsAffected) {
                path.push(`data_risk: ${dataRisk}`);
                path.push(`security_risk: ${securityRisk}`);
            }
        }
        
        return path;
    }

    /**
     * Create fallback classification for unclassifiable errors
     */
    private createFallbackClassification(
        error: Error,
        context: ErrorContext,
    ): ErrorClassification {
        return {
            severity: Severity.ERROR,
            category: Category.UNKNOWN,
            recoverability: Recoverability.PARTIAL,
            systemFunctional: true,
            multipleComponentsAffected: false,
            dataRisk: false,
            securityRisk: false,
            confidenceScore: 0.3, // Low confidence for fallback
            timestamp: new Date(),
            metadata: {
                fallback: true,
                errorType: error.constructor.name,
                errorMessage: error.message,
                tier: context.tier,
            },
        };
    }

    /**
     * Store classification for learning
     */
    private storeClassification(
        errorSignature: string,
        classification: ErrorClassification,
    ): void {
        const history = this.classificationHistory.get(errorSignature) || [];
        history.push(classification);
        
        // Keep only recent history
        if (history.length > this.maxHistorySize) {
            history.splice(0, history.length - this.maxHistorySize);
        }
        
        this.classificationHistory.set(errorSignature, history);
    }

    /**
     * Add learned pattern
     */
    addPattern(pattern: ErrorPattern): void {
        // Remove existing pattern with same ID
        this.patterns = this.patterns.filter(p => p.id !== pattern.id);
        
        // Add new pattern
        this.patterns.push(pattern);
        
        logger.debug("Pattern added to classifier", {
            patternId: pattern.id,
            name: pattern.name,
            totalPatterns: this.patterns.length,
        });
    }

    /**
     * Get classification statistics
     */
    getStatistics(): {
        totalClassifications: number;
        uniqueErrors: number;
        patterns: number;
        averageConfidence: number;
    } {
        let totalClassifications = 0;
        let totalConfidence = 0;
        
        for (const history of this.classificationHistory.values()) {
            totalClassifications += history.length;
            totalConfidence += history.reduce((sum, c) => sum + c.confidenceScore, 0);
        }
        
        return {
            totalClassifications,
            uniqueErrors: this.classificationHistory.size,
            patterns: this.patterns.length,
            averageConfidence: totalClassifications > 0 ? 
                totalConfidence / totalClassifications : 0,
        };
    }
}

/**
 * Error features extracted for classification
 */
interface ErrorFeatures {
    errorType: string;
    errorMessage: string;
    errorSignature: string;
    
    // Network-related
    isNetworkError: boolean;
    isTimeoutError: boolean;
    httpStatusCode?: number;
    
    // Database-related
    isDatabaseError: boolean;
    
    // File system
    isFileSystemError: boolean;
    
    // Authentication/Authorization
    isAuthError: boolean;
    
    // Resource-related
    isResourceError: boolean;
    isRateLimitError: boolean;
    
    // Infrastructure
    isInfrastructureError: boolean;
    
    // Validation
    isValidationError: boolean;
    
    // Configuration
    isConfigurationError: boolean;
    
    // Context
    stackDepth: number;
    occurredInTier: 1 | 2 | 3;
    operationType: string;
}