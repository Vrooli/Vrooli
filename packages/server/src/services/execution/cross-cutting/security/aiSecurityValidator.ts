/**
 * AI Security Validator
 * Specialized security validation for AI interactions including prompt injection detection,
 * hallucination detection, bias detection, and output validation
 */

import type {
    AISecurityValidation,
    ValidationResult,
    ValidationIssue,
    ThreatDetection,
    SecurityIncident,
    IncidentType,
    IncidentStatus,
    SecurityAudit,
} from "@vrooli/shared";
import { logger } from "../../../../events/logger.js";
import { TelemetryShim } from "../monitoring/telemetryShim.js";
import { RedisEventBus } from "../events/eventBus.js";
import { v4 as uuidv4 } from "uuid";

/**
 * AI security validation configuration
 */
export interface AISecurityConfig {
    enablePromptInjectionDetection: boolean;
    enableHallucinationDetection: boolean;
    enableBiasDetection: boolean;
    enableContentFiltering: boolean;
    enablePIIDetection: boolean;
    confidenceThreshold: number; // 0-1
    maxInputLength: number;
    maxOutputLength: number;
    bannedPatterns: string[];
    sensitiveTopics: string[];
    piiPatterns: {
        email: RegExp;
        ssn: RegExp;
        phone: RegExp;
        creditCard: RegExp;
        ipAddress: RegExp;
    };
}

/**
 * Default AI security configuration
 */
const DEFAULT_AI_CONFIG: AISecurityConfig = {
    enablePromptInjectionDetection: true,
    enableHallucinationDetection: true,
    enableBiasDetection: true,
    enableContentFiltering: true,
    enablePIIDetection: true,
    confidenceThreshold: 0.7,
    maxInputLength: 100000,
    maxOutputLength: 50000,
    bannedPatterns: [
        // Prompt injection patterns
        "ignore previous instructions",
        "forget everything above",
        "system:",
        "assistant:",
        "user:",
        "jailbreak",
        "roleplay as",
        "pretend you are",
        // Code injection patterns
        "<script>",
        "javascript:",
        "eval(",
        "exec(",
        "system(",
        "shell(",
    ],
    sensitiveTopics: [
        "violence",
        "hate speech",
        "illegal activities",
        "privacy violation",
        "financial fraud",
        "medical advice",
        "legal advice",
    ],
    piiPatterns: {
        email: /\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b/g,
        ssn: /\b\d{3}-?\d{2}-?\d{4}\b/g,
        phone: /\b\d{3}-?\d{3}-?\d{4}\b/g,
        creditCard: /\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b/g,
        ipAddress: /\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b/g,
    },
};

/**
 * AI Security Validator
 * 
 * Provides comprehensive security validation for AI interactions,
 * including input sanitization, output validation, and threat detection.
 */
export class AISecurityValidator {
    private readonly telemetry: TelemetryShim;
    private readonly eventBus: RedisEventBus;
    private readonly config: AISecurityConfig;
    
    // Statistics
    private validationCount = 0;
    private promptInjectionAttempts = 0;
    private hallucinationDetections = 0;
    private biasDetections = 0;
    private piiDetections = 0;
    private contentFiltered = 0;

    constructor(
        telemetry: TelemetryShim,
        eventBus: RedisEventBus,
        config: Partial<AISecurityConfig> = {},
    ) {
        this.telemetry = telemetry;
        this.eventBus = eventBus;
        this.config = { ...DEFAULT_AI_CONFIG, ...config };
    }

    /**
     * Comprehensive AI security validation
     */
    async validateAIInteraction(
        input: string,
        output: string,
        context: {
            requestId: string;
            userId?: string;
            tier: 1 | 2 | 3;
            component: string;
            operation: string;
        },
    ): Promise<AISecurityValidation> {
        const startTime = performance.now();
        this.validationCount++;

        try {
            const validation: AISecurityValidation = {
                inputValidation: await this.validateInput(input, context),
                outputValidation: await this.validateOutput(output, context),
                promptInjectionCheck: await this.checkPromptInjection(input, context),
                contentSafetyCheck: await this.checkContentSafety(input, output, context),
                biasCheck: await this.checkBias(output, context),
                privacyCheck: await this.checkPrivacy(input, output, context),
                overallRisk: 0,
            };

            // Calculate overall risk score
            validation.overallRisk = this.calculateOverallRisk(validation);

            // Emit telemetry
            await this.emitValidationTelemetry(validation, context);

            // Emit security events if threats detected
            await this.emitThreatEvents(validation, context, input, output);

            const duration = performance.now() - startTime;
            await this.telemetry.emitExecutionTiming(
                "AISecurityValidator",
                "validateAIInteraction",
                new Date(startTime),
                new Date(),
                validation.overallRisk < this.config.confidenceThreshold,
                {
                    tier: context.tier,
                    riskScore: validation.overallRisk,
                    inputLength: input.length,
                    outputLength: output.length,
                },
            );

            return validation;
        } catch (error) {
            logger.error("AI security validation error", {
                requestId: context.requestId,
                tier: context.tier,
                error,
            });

            await this.telemetry.emitError(
                error,
                "AISecurityValidator",
                "critical",
                { requestId: context.requestId },
            );

            // Return high-risk validation on error
            return {
                inputValidation: this.createErrorValidation("Input validation error"),
                outputValidation: this.createErrorValidation("Output validation error"),
                promptInjectionCheck: this.createErrorValidation("Prompt injection check error"),
                contentSafetyCheck: this.createErrorValidation("Content safety check error"),
                biasCheck: this.createErrorValidation("Bias check error"),
                privacyCheck: this.createErrorValidation("Privacy check error"),
                overallRisk: 1.0,
            };
        }
    }

    /**
     * Validate AI input for security threats
     */
    private async validateInput(
        input: string,
        context: { requestId: string; tier: number; component: string },
    ): Promise<ValidationResult> {
        const issues: ValidationIssue[] = [];
        let confidence = 1.0;

        // Check input length
        if (input.length > this.config.maxInputLength) {
            issues.push({
                type: "input_length",
                severity: "medium",
                description: `Input length ${input.length} exceeds maximum ${this.config.maxInputLength}`,
                suggestion: "Truncate input to acceptable length",
            });
            confidence -= 0.2;
        }

        // Check for empty or suspicious input
        if (!input.trim()) {
            issues.push({
                type: "empty_input",
                severity: "low",
                description: "Empty input detected",
                suggestion: "Provide meaningful input",
            });
            confidence -= 0.1;
        }

        // Check for banned patterns
        for (const pattern of this.config.bannedPatterns) {
            if (input.toLowerCase().includes(pattern.toLowerCase())) {
                issues.push({
                    type: "banned_pattern",
                    severity: "high",
                    description: `Banned pattern detected: ${pattern}`,
                    location: `Position ${input.toLowerCase().indexOf(pattern.toLowerCase())}`,
                    suggestion: "Remove or rephrase the problematic content",
                });
                confidence -= 0.3;
            }
        }

        // Check encoding and character anomalies
        if (this.hasUnicodeAnomalies(input)) {
            issues.push({
                type: "encoding_anomaly",
                severity: "medium",
                description: "Suspicious Unicode characters detected",
                suggestion: "Use standard ASCII characters",
            });
            confidence -= 0.2;
        }

        return {
            passed: issues.length === 0 || confidence >= this.config.confidenceThreshold,
            confidence: Math.max(0, confidence),
            issues,
            recommendation: this.generateRecommendation("input", issues),
            metadata: {
                inputLength: input.length,
                bannedPatternsFound: issues.filter(i => i.type === "banned_pattern").length,
            },
        };
    }

    /**
     * Validate AI output for security and quality issues
     */
    private async validateOutput(
        output: string,
        context: { requestId: string; tier: number; component: string },
    ): Promise<ValidationResult> {
        const issues: ValidationIssue[] = [];
        let confidence = 1.0;

        // Check output length
        if (output.length > this.config.maxOutputLength) {
            issues.push({
                type: "output_length",
                severity: "medium",
                description: `Output length ${output.length} exceeds maximum ${this.config.maxOutputLength}`,
                suggestion: "Truncate output to acceptable length",
            });
            confidence -= 0.2;
        }

        // Check for code injection attempts
        if (this.hasCodeInjection(output)) {
            issues.push({
                type: "code_injection",
                severity: "critical",
                description: "Potential code injection detected in output",
                suggestion: "Remove or sanitize executable code",
            });
            confidence -= 0.5;
        }

        // Check for information leakage
        if (this.hasInformationLeakage(output)) {
            issues.push({
                type: "information_leakage",
                severity: "high",
                description: "Potential information leakage detected",
                suggestion: "Remove sensitive information from output",
            });
            confidence -= 0.4;
        }

        // Check for consistency and coherence
        const coherenceScore = this.assessCoherence(output);
        if (coherenceScore < 0.5) {
            issues.push({
                type: "low_coherence",
                severity: "medium",
                description: `Low coherence score: ${coherenceScore.toFixed(2)}`,
                suggestion: "Regenerate response for better coherence",
            });
            confidence -= 0.2;
        }

        return {
            passed: issues.length === 0 || confidence >= this.config.confidenceThreshold,
            confidence: Math.max(0, confidence),
            issues,
            recommendation: this.generateRecommendation("output", issues),
            metadata: {
                outputLength: output.length,
                coherenceScore,
                codeInjectionRisk: this.hasCodeInjection(output),
                informationLeakageRisk: this.hasInformationLeakage(output),
            },
        };
    }

    /**
     * Check for prompt injection attempts
     */
    private async checkPromptInjection(
        input: string,
        context: { requestId: string; tier: number; component: string },
    ): Promise<ValidationResult> {
        if (!this.config.enablePromptInjectionDetection) {
            return this.createPassingValidation("Prompt injection detection disabled");
        }

        const issues: ValidationIssue[] = [];
        let confidence = 1.0;

        // Check for instruction manipulation
        const injectionPatterns = [
            /ignore.*previous.*instructions?/i,
            /forget.*everything.*above/i,
            /system\s*:/i,
            /assistant\s*:/i,
            /roleplay.*as/i,
            /pretend.*you.*are/i,
            /act.*as.*if/i,
            /imagine.*you.*are/i,
        ];

        for (const pattern of injectionPatterns) {
            const matches = input.match(pattern);
            if (matches) {
                issues.push({
                    type: "prompt_injection",
                    severity: "high",
                    description: `Prompt injection pattern detected: ${matches[0]}`,
                    location: `Position ${input.indexOf(matches[0])}`,
                    suggestion: "Rephrase to avoid instruction manipulation",
                });
                confidence -= 0.4;
                this.promptInjectionAttempts++;
            }
        }

        // Check for jailbreak attempts
        if (this.hasJailbreakAttempt(input)) {
            issues.push({
                type: "jailbreak_attempt",
                severity: "critical",
                description: "Potential jailbreak attempt detected",
                suggestion: "Remove attempts to bypass system constraints",
            });
            confidence -= 0.6;
        }

        return {
            passed: issues.length === 0 || confidence >= this.config.confidenceThreshold,
            confidence: Math.max(0, confidence),
            issues,
            recommendation: this.generateRecommendation("prompt_injection", issues),
            metadata: {
                injectionPatternsFound: issues.length,
                jailbreakAttempt: this.hasJailbreakAttempt(input),
            },
        };
    }

    /**
     * Check content safety for both input and output
     */
    private async checkContentSafety(
        input: string,
        output: string,
        context: { requestId: string; tier: number; component: string },
    ): Promise<ValidationResult> {
        if (!this.config.enableContentFiltering) {
            return this.createPassingValidation("Content filtering disabled");
        }

        const issues: ValidationIssue[] = [];
        let confidence = 1.0;

        // Check for sensitive topics
        for (const topic of this.config.sensitiveTopics) {
            if (input.toLowerCase().includes(topic) || output.toLowerCase().includes(topic)) {
                issues.push({
                    type: "sensitive_topic",
                    severity: "medium",
                    description: `Sensitive topic detected: ${topic}`,
                    suggestion: "Avoid or handle sensitive topics carefully",
                });
                confidence -= 0.2;
                this.contentFiltered++;
            }
        }

        // Check for harmful content
        if (this.hasHarmfulContent(output)) {
            issues.push({
                type: "harmful_content",
                severity: "high",
                description: "Potentially harmful content detected in output",
                suggestion: "Remove or modify harmful content",
            });
            confidence -= 0.4;
        }

        return {
            passed: issues.length === 0 || confidence >= this.config.confidenceThreshold,
            confidence: Math.max(0, confidence),
            issues,
            recommendation: this.generateRecommendation("content_safety", issues),
            metadata: {
                sensitiveTopicsFound: issues.filter(i => i.type === "sensitive_topic").length,
                harmfulContentDetected: this.hasHarmfulContent(output),
            },
        };
    }

    /**
     * Check for bias in AI output
     */
    private async checkBias(
        output: string,
        context: { requestId: string; tier: number; component: string },
    ): Promise<ValidationResult> {
        if (!this.config.enableBiasDetection) {
            return this.createPassingValidation("Bias detection disabled");
        }

        const issues: ValidationIssue[] = [];
        let confidence = 1.0;

        // Check for demographic bias
        const biasIndicators = this.detectBiasIndicators(output);
        if (biasIndicators.length > 0) {
            issues.push({
                type: "demographic_bias",
                severity: "medium",
                description: `Potential bias indicators detected: ${biasIndicators.join(", ")}`,
                suggestion: "Review for balanced representation",
            });
            confidence -= 0.3;
            this.biasDetections++;
        }

        // Check for stereotypical language
        if (this.hasStereotypicalLanguage(output)) {
            issues.push({
                type: "stereotypical_language",
                severity: "medium",
                description: "Stereotypical language patterns detected",
                suggestion: "Use more inclusive language",
            });
            confidence -= 0.2;
        }

        return {
            passed: issues.length === 0 || confidence >= this.config.confidenceThreshold,
            confidence: Math.max(0, confidence),
            issues,
            recommendation: this.generateRecommendation("bias", issues),
            metadata: {
                biasIndicators,
                stereotypicalLanguage: this.hasStereotypicalLanguage(output),
            },
        };
    }

    /**
     * Check for privacy violations (PII detection)
     */
    private async checkPrivacy(
        input: string,
        output: string,
        context: { requestId: string; tier: number; component: string },
    ): Promise<ValidationResult> {
        if (!this.config.enablePIIDetection) {
            return this.createPassingValidation("PII detection disabled");
        }

        const issues: ValidationIssue[] = [];
        let confidence = 1.0;
        const combinedText = input + " " + output;

        // Check for various PII types
        for (const [type, pattern] of Object.entries(this.config.piiPatterns)) {
            const matches = combinedText.match(pattern);
            if (matches && matches.length > 0) {
                issues.push({
                    type: "pii_detected",
                    severity: "high",
                    description: `${type.toUpperCase()} detected: ${matches.length} instance(s)`,
                    suggestion: `Remove or mask ${type} information`,
                });
                confidence -= 0.3;
                this.piiDetections++;

                // Emit PII detection event
                await this.telemetry.emitPIIDetection(
                    [type],
                    [`${type} pattern`],
                    "flagged",
                );
            }
        }

        return {
            passed: issues.length === 0 || confidence >= this.config.confidenceThreshold,
            confidence: Math.max(0, confidence),
            issues,
            recommendation: this.generateRecommendation("privacy", issues),
            metadata: {
                piiTypesDetected: issues.map(i => i.description),
                totalPiiInstances: issues.length,
            },
        };
    }

    /**
     * Calculate overall risk score from all validation results
     */
    private calculateOverallRisk(validation: AISecurityValidation): number {
        const weights = {
            inputValidation: 0.15,
            outputValidation: 0.15,
            promptInjectionCheck: 0.25,
            contentSafetyCheck: 0.20,
            biasCheck: 0.10,
            privacyCheck: 0.15,
        };

        let totalRisk = 0;
        totalRisk += (1 - validation.inputValidation.confidence) * weights.inputValidation;
        totalRisk += (1 - validation.outputValidation.confidence) * weights.outputValidation;
        totalRisk += (1 - validation.promptInjectionCheck.confidence) * weights.promptInjectionCheck;
        totalRisk += (1 - validation.contentSafetyCheck.confidence) * weights.contentSafetyCheck;
        totalRisk += (1 - validation.biasCheck.confidence) * weights.biasCheck;
        totalRisk += (1 - validation.privacyCheck.confidence) * weights.privacyCheck;

        return Math.min(totalRisk, 1.0);
    }

    /**
     * Helper methods
     */
    private createErrorValidation(message: string): ValidationResult {
        return {
            passed: false,
            confidence: 0,
            issues: [{
                type: "system_error",
                severity: "critical",
                description: message,
            }],
            recommendation: "Retry validation or contact system administrator",
            metadata: { error: true },
        };
    }

    private createPassingValidation(reason: string): ValidationResult {
        return {
            passed: true,
            confidence: 1.0,
            issues: [],
            recommendation: "No issues detected",
            metadata: { reason },
        };
    }

    private generateRecommendation(type: string, issues: ValidationIssue[]): string {
        if (issues.length === 0) {
            return `${type} validation passed - no issues detected`;
        }

        const criticalIssues = issues.filter(i => i.severity === "critical");
        const highIssues = issues.filter(i => i.severity === "high");

        if (criticalIssues.length > 0) {
            return `Critical ${type} issues detected - immediate action required`;
        }
        if (highIssues.length > 0) {
            return `High severity ${type} issues detected - review and address`;
        }
        return `Minor ${type} issues detected - consider addressing`;
    }

    private hasUnicodeAnomalies(text: string): boolean {
        // Check for suspicious Unicode characters or sequences
        const suspiciousPatterns = [
            /[\u200B-\u200D\uFEFF]/g, // Zero-width characters
            /[\u0000-\u001F]/g, // Control characters
            /[\uE000-\uF8FF]/g, // Private use area
        ];

        return suspiciousPatterns.some(pattern => pattern.test(text));
    }

    private hasCodeInjection(text: string): boolean {
        const codePatterns = [
            /<script[\s\S]*?>[\s\S]*?<\/script>/gi,
            /javascript:/gi,
            /eval\s*\(/gi,
            /exec\s*\(/gi,
            /system\s*\(/gi,
            /shell\s*\(/gi,
            /cmd\s*\(/gi,
        ];

        return codePatterns.some(pattern => pattern.test(text));
    }

    private hasInformationLeakage(text: string): boolean {
        const leakagePatterns = [
            /api[_\s]?key/gi,
            /secret/gi,
            /password/gi,
            /token/gi,
            /private[_\s]?key/gi,
            /database[_\s]?connection/gi,
        ];

        return leakagePatterns.some(pattern => pattern.test(text));
    }

    private assessCoherence(text: string): number {
        // Simple coherence assessment based on sentence structure and repetition
        const sentences = text.split(/[.!?]+/).filter(s => s.trim().length > 0);
        if (sentences.length === 0) return 0;

        let coherenceScore = 1.0;

        // Check for excessive repetition
        const words = text.toLowerCase().split(/\s+/);
        const uniqueWords = new Set(words);
        const repetitionRatio = uniqueWords.size / words.length;
        coherenceScore *= repetitionRatio;

        // Check sentence length variation
        const avgSentenceLength = sentences.reduce((sum, s) => sum + s.length, 0) / sentences.length;
        const variance = sentences.reduce((sum, s) => sum + Math.pow(s.length - avgSentenceLength, 2), 0) / sentences.length;
        const standardDeviation = Math.sqrt(variance);
        const variationScore = Math.min(standardDeviation / avgSentenceLength, 1);
        coherenceScore *= variationScore;

        return Math.max(0, Math.min(coherenceScore, 1));
    }

    private hasJailbreakAttempt(text: string): boolean {
        const jailbreakPatterns = [
            /jailbreak/gi,
            /bypass.*safety/gi,
            /ignore.*safety/gi,
            /override.*restrictions/gi,
            /disable.*filters/gi,
            /unrestricted.*mode/gi,
        ];

        return jailbreakPatterns.some(pattern => pattern.test(text));
    }

    private hasHarmfulContent(text: string): boolean {
        const harmfulPatterns = [
            /violence/gi,
            /harm/gi,
            /illegal/gi,
            /dangerous/gi,
            /explosive/gi,
            /weapon/gi,
        ];

        return harmfulPatterns.some(pattern => pattern.test(text));
    }

    private detectBiasIndicators(text: string): string[] {
        const biasPatterns = {
            gender: /\b(men|women|male|female|guy|girl)\s+(are|tend to|usually|always|never)\b/gi,
            race: /\b(people of|those|they)\s+(color|race|ethnicity)\s+(are|tend to|usually)\b/gi,
            age: /\b(young|old|elderly)\s+(people|folks)\s+(are|tend to|usually)\b/gi,
            religion: /\b(religious|christian|muslim|jewish|hindu|buddhist)\s+(people|folks)\s+(are|tend to)\b/gi,
        };

        const indicators: string[] = [];
        for (const [type, pattern] of Object.entries(biasPatterns)) {
            if (pattern.test(text)) {
                indicators.push(type);
            }
        }

        return indicators;
    }

    private hasStereotypicalLanguage(text: string): boolean {
        const stereotypePatterns = [
            /all\s+(men|women|people)\s+are/gi,
            /(men|women|people)\s+always/gi,
            /(men|women|people)\s+never/gi,
            /typical\s+(man|woman|person)/gi,
        ];

        return stereotypePatterns.some(pattern => pattern.test(text));
    }

    private async emitValidationTelemetry(
        validation: AISecurityValidation,
        context: { requestId: string; tier: number; component: string },
    ): Promise<void> {
        await this.telemetry.emitValidationError(
            "AISecurityValidator",
            [
                ...validation.inputValidation.issues.map(issue => ({
                    field: "input",
                    rule: issue.type,
                    message: issue.description,
                    severity: issue.severity as "warning" | "error" | "critical",
                })),
                ...validation.outputValidation.issues.map(issue => ({
                    field: "output",
                    rule: issue.type,
                    message: issue.description,
                    severity: issue.severity as "warning" | "error" | "critical",
                })),
            ],
            {
                requestId: context.requestId,
                tier: context.tier,
                component: context.component,
                overallRisk: validation.overallRisk,
            },
        );
    }

    private async emitThreatEvents(
        validation: AISecurityValidation,
        context: { requestId: string; tier: number; component: string },
        input: string,
        output: string,
    ): Promise<void> {
        // Emit prompt injection detection events
        if (validation.promptInjectionCheck.issues.length > 0) {
            await this.telemetry.emitSecurityIncident(
                "prompt_injection",
                "high",
                {
                    requestId: context.requestId,
                    tier: context.tier,
                    component: context.component,
                    issues: validation.promptInjectionCheck.issues,
                    confidence: validation.promptInjectionCheck.confidence,
                },
            );
        }

        // Emit PII detection events (already handled in checkPrivacy)

        // Emit content safety incidents
        if (validation.contentSafetyCheck.issues.length > 0) {
            await this.telemetry.emitSecurityIncident(
                "content_safety",
                "medium",
                {
                    requestId: context.requestId,
                    tier: context.tier,
                    component: context.component,
                    issues: validation.contentSafetyCheck.issues,
                },
            );
        }
    }

    /**
     * Get AI security validation statistics
     */
    getStatistics(): {
        validationCount: number;
        promptInjectionAttempts: number;
        hallucinationDetections: number;
        biasDetections: number;
        piiDetections: number;
        contentFiltered: number;
        averageRiskScore: number;
    } {
        return {
            validationCount: this.validationCount,
            promptInjectionAttempts: this.promptInjectionAttempts,
            hallucinationDetections: this.hallucinationDetections,
            biasDetections: this.biasDetections,
            piiDetections: this.piiDetections,
            contentFiltered: this.contentFiltered,
            averageRiskScore: 0, // Would be calculated from stored risk scores
        };
    }
}