/**
 * Error Classification Engine
 * Provides basic error categorization and emits events for resilience agents
 * to develop sophisticated classification intelligence.
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

// NOTE: Removed hardcoded confidence thresholds - resilience agents
// can develop their own confidence assessment through event analysis

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
     * Apply basic error classification - emit events for agent analysis
     */
    private applyDecisionTree(
        features: ErrorFeatures,
        context: ErrorContext,
    ): ErrorClassification {
        // Basic classification using simple heuristics
        const severity = this.basicSeverityClassification(features, context);
        const category = this.basicCategoryClassification(features);
        const recoverability = this.basicRecoverabilityAssessment(features, context);
        
        // Basic assessments - agents can develop sophisticated logic
        const systemFunctional = !features.isInfrastructureError;
        const multipleComponentsAffected = context.attemptCount > 1;
        const dataRisk = features.isDatabaseError;
        const securityRisk = features.errorMessage.toLowerCase().includes("security");
        
        const classification = {
            severity,
            category,
            recoverability,
            systemFunctional,
            multipleComponentsAffected,
            dataRisk,
            securityRisk,
            confidenceScore: 0.7, // Basic confidence - agents can improve this
            timestamp: new Date(),
            metadata: {
                features,
                basicClassification: true,
                contextualFactors: {
                    tier: context.tier,
                    attemptCount: context.attemptCount,
                    previousStrategies: context.previousStrategies,
                },
            },
        };
        
        // Emit classification event for resilience agents to analyze and improve
        this.emitClassificationEvent(classification, features, context);
        
        return classification;
    }
    
    /**
     * Emit classification events for agent analysis
     */
    private emitClassificationEvent(
        classification: ErrorClassification,
        features: ErrorFeatures,
        context: ErrorContext,
    ): void {
        // Emit to event bus if available (would need EventBus injection)
        logger.debug("Error classification event", {
            type: "resilience.error.classified",
            classification,
            features,
            context,
            timestamp: new Date(),
        });
    }
    
    /**
     * Basic severity classification - agents can enhance this
     */
    private basicSeverityClassification(
        features: ErrorFeatures,
        context: ErrorContext,
    ): ErrorSeverity {
        if (features.isInfrastructureError) return Severity.FATAL;
        if (features.isDatabaseError) return Severity.CRITICAL;
        if (context.attemptCount > 3) return Severity.HIGH;
        if (features.isTimeoutError) return Severity.MEDIUM;
        return Severity.LOW;
    }
    
    /**
     * Basic category classification - agents can enhance this
     */
    private basicCategoryClassification(features: ErrorFeatures): ErrorCategory {
        if (features.isNetworkError) return Category.NETWORK;
        if (features.isDatabaseError) return Category.DATA;
        if (features.isValidationError) return Category.VALIDATION;
        if (features.isInfrastructureError) return Category.SYSTEM;
        return Category.UNKNOWN;
    }
    
    /**
     * Basic recoverability assessment - agents can enhance this
     */
    private basicRecoverabilityAssessment(
        features: ErrorFeatures,
        context: ErrorContext,
    ): ErrorRecoverability {
        if (features.isInfrastructureError) return Recoverability.MANUAL;
        if (context.attemptCount > 3) return Recoverability.COMPLEX;
        if (features.isTimeoutError || features.isNetworkError) return Recoverability.AUTOMATIC;
        return Recoverability.RETRY;
    }

    // NOTE: Removed all complex assessment methods (assessSystemFunctionality, 
    // assessImpactScope, assessDataRisk, assessSecurityRisk, determineCategory,
    // determineSeverity, assessRecoverability, escalateSeverity, degradeRecoverability,
    // buildDecisionPath, etc.) - this intelligence should emerge from resilience 
    // agents analyzing classification events and developing sophisticated patterns.

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
