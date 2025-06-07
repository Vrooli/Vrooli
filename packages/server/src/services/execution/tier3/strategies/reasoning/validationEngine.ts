import { type Logger } from "winston";

/**
 * Validation result interface
 */
interface ValidationResult {
    type: string;
    passed: boolean;
    message: string;
    details?: Record<string, unknown>;
}

/**
 * Reasoning result interface for validation
 */
interface ReasoningResult {
    conclusion: unknown;
    reasoning: string[];
    evidence: Evidence[];
    confidence: number;
    assumptions: string[];
}

/**
 * Evidence interface
 */
interface Evidence {
    type: "fact" | "inference" | "assumption";
    content: string;
    source?: string;
    confidence: number;
}

/**
 * Reasoning framework interface for validation
 */
interface ReasoningFramework {
    type: "logical" | "analytical" | "decision_tree" | "evidence_based";
    steps: Array<{
        id: string;
        type: string;
        description: string;
        inputs: string[];
        outputs: string[];
    }>;
}

/**
 * Legacy validation and refinement patterns extracted from legacy ReasoningStrategy
 * 
 * This module implements comprehensive validation patterns including:
 * - Logical consistency checking
 * - Bias detection
 * - Completeness validation
 * - Result refinement and enhancement
 */
export class ValidationEngine {
    private readonly logger: Logger;

    // Validation configuration constants
    private readonly CONFIDENCE_THRESHOLD = 0.7;
    private readonly MAX_ASSUMPTIONS_THRESHOLD = 3;
    private readonly MIN_EVIDENCE_THRESHOLD = 1;

    constructor(logger: Logger) {
        this.logger = logger;
    }

    /**
     * EXTRACTED FROM LEGACY: Comprehensive validation of reasoning results
     * Validates reasoning results using multiple validation patterns
     */
    async validateReasoning(
        result: ReasoningResult,
        framework: ReasoningFramework
    ): Promise<ValidationResult[]> {
        this.logger.debug("[ValidationEngine] Starting comprehensive validation", {
            frameworkType: framework.type,
            conclusionType: typeof result.conclusion,
            reasoningStepsCount: result.reasoning.length,
            evidenceCount: result.evidence.length,
        });

        const validations: ValidationResult[] = [];

        // LEGACY PATTERN: Logic validation
        validations.push({
            type: "logic",
            passed: this.validateLogicalConsistency(result),
            message: "Logical consistency check",
            details: {
                reasoningSteps: result.reasoning.length,
                contradictionsFound: this.findContradictions(result),
            },
        });

        // LEGACY PATTERN: Completeness validation
        validations.push({
            type: "completeness",
            passed: this.validateCompleteness(result, framework),
            message: "Reasoning completeness check",
            details: {
                expectedSteps: framework.steps.length,
                actualSteps: result.reasoning.length,
                hasConclusion: result.conclusion !== undefined,
            },
        });

        // LEGACY PATTERN: Bias detection
        const biasType = this.detectBias(result);
        validations.push({
            type: "bias",
            passed: biasType === null,
            message: biasType ? `Bias detected: ${biasType}` : "No bias detected",
            details: {
                biasType,
                confidenceScore: result.confidence,
            },
        });

        // ENHANCED: Confidence validation
        validations.push({
            type: "confidence",
            passed: result.confidence >= this.CONFIDENCE_THRESHOLD,
            message: `Confidence threshold check (${this.CONFIDENCE_THRESHOLD})`,
            details: {
                actualConfidence: result.confidence,
                threshold: this.CONFIDENCE_THRESHOLD,
            },
        });

        // ENHANCED: Evidence validation
        validations.push({
            type: "evidence",
            passed: result.evidence.length >= this.MIN_EVIDENCE_THRESHOLD,
            message: "Evidence sufficiency check",
            details: {
                evidenceCount: result.evidence.length,
                minimumRequired: this.MIN_EVIDENCE_THRESHOLD,
                evidenceTypes: result.evidence.map(e => e.type),
            },
        });

        // ENHANCED: Assumptions validation
        validations.push({
            type: "assumptions",
            passed: result.assumptions.length <= this.MAX_ASSUMPTIONS_THRESHOLD,
            message: "Assumptions count check",
            details: {
                assumptionsCount: result.assumptions.length,
                maximumAllowed: this.MAX_ASSUMPTIONS_THRESHOLD,
                assumptions: result.assumptions,
            },
        });

        const passedCount = validations.filter(v => v.passed).length;
        const totalCount = validations.length;
        
        this.logger.info("[ValidationEngine] Validation completed", {
            frameworkType: framework.type,
            passedValidations: passedCount,
            totalValidations: totalCount,
            overallPassed: passedCount === totalCount,
        });

        return validations;
    }

    /**
     * EXTRACTED FROM LEGACY: validateLogicalConsistency
     * Checks for logical contradictions in reasoning
     */
    private validateLogicalConsistency(result: ReasoningResult): boolean {
        // Check for contradictions in reasoning
        const reasoningText = result.reasoning.join(" ");
        
        // LEGACY PATTERN: Simple contradiction patterns
        const contradictionPatterns = [
            /both true and false/i,
            /contradicts earlier/i,
            /inconsistent with/i,
            /cannot be both/i,
            /impossible and possible/i,
            /always and never/i,
        ];

        const hasContradictions = contradictionPatterns.some(pattern => pattern.test(reasoningText));
        
        // Additional check: conclusion consistency with reasoning
        const conclusionText = String(result.conclusion).toLowerCase();
        const hasReasoningConclusionMismatch = this.checkConclusionConsistency(
            reasoningText.toLowerCase(),
            conclusionText
        );

        return !hasContradictions && !hasReasoningConclusionMismatch;
    }

    /**
     * Checks for consistency between reasoning and conclusion
     */
    private checkConclusionConsistency(reasoningText: string, conclusionText: string): boolean {
        // Check for obvious mismatches
        const positivePatterns = [/success/i, /true/i, /positive/i, /yes/i, /approve/i];
        const negativePatterns = [/fail/i, /false/i, /negative/i, /no/i, /reject/i];
        
        const reasoningIsPositive = positivePatterns.some(p => p.test(reasoningText));
        const reasoningIsNegative = negativePatterns.some(p => p.test(reasoningText));
        const conclusionIsPositive = positivePatterns.some(p => p.test(conclusionText));
        const conclusionIsNegative = negativePatterns.some(p => p.test(conclusionText));
        
        // Check for clear mismatches
        return (reasoningIsPositive && conclusionIsNegative) || (reasoningIsNegative && conclusionIsPositive);
    }

    /**
     * Finds specific contradictions in reasoning
     */
    private findContradictions(result: ReasoningResult): string[] {
        const contradictions: string[] = [];
        const reasoningText = result.reasoning.join(" ");
        
        const contradictionChecks = [
            { pattern: /both true and false/i, description: "Boolean contradiction" },
            { pattern: /contradicts earlier/i, description: "Self-contradiction" },
            { pattern: /inconsistent with/i, description: "Inconsistency" },
            { pattern: /cannot be both/i, description: "Mutual exclusion violation" },
        ];
        
        for (const check of contradictionChecks) {
            if (check.pattern.test(reasoningText)) {
                contradictions.push(check.description);
            }
        }
        
        return contradictions;
    }

    /**
     * EXTRACTED FROM LEGACY: validateCompleteness
     * Checks if reasoning covers all expected framework steps
     */
    private validateCompleteness(result: ReasoningResult, framework: ReasoningFramework): boolean {
        // Check if all framework steps produced outputs
        const expectedOutputs = framework.steps.flatMap(s => s.outputs);
        const hasConclusion = result.conclusion !== undefined;
        const hasReasoning = result.reasoning.length >= framework.steps.length;
        
        // Enhanced completeness check
        const hasMinimumEvidence = result.evidence.length > 0;
        const reasoningCoversSteps = this.checkReasoningCoverage(result.reasoning, framework.steps);
        
        return hasConclusion && hasReasoning && hasMinimumEvidence && reasoningCoversSteps;
    }

    /**
     * Checks if reasoning steps cover the framework requirements
     */
    private checkReasoningCoverage(reasoning: string[], frameworkSteps: any[]): boolean {
        if (frameworkSteps.length === 0) return true;
        
        const reasoningText = reasoning.join(" ").toLowerCase();
        let coveredSteps = 0;
        
        for (const step of frameworkSteps) {
            const stepKeywords = step.type.toLowerCase().split("_");
            const stepCovered = stepKeywords.some(keyword => reasoningText.includes(keyword));
            if (stepCovered) {
                coveredSteps++;
            }
        }
        
        // At least 70% of steps should be covered
        return (coveredSteps / frameworkSteps.length) >= 0.7;
    }

    /**
     * EXTRACTED FROM LEGACY: detectBias
     * Detects common reasoning biases
     */
    private detectBias(result: ReasoningResult): string | null {
        const reasoningText = result.reasoning.join(" ");
        
        // LEGACY PATTERN: Simple bias detection
        const biasIndicators = [
            { pattern: /always|never|all|none/i, type: "absolute thinking" },
            { pattern: /obviously|clearly|definitely/i, type: "overconfidence" },
            { pattern: /probably not|unlikely|doubt/i, type: "negative bias" },
            { pattern: /surely|certainly|without question/i, type: "certainty bias" },
            { pattern: /everyone knows|it's obvious/i, type: "appeal to common knowledge" },
            { pattern: /must be|has to be|can only be/i, type: "false dichotomy" },
        ];
        
        for (const indicator of biasIndicators) {
            if (indicator.pattern.test(reasoningText)) {
                return indicator.type;
            }
        }
        
        // Check for confidence-evidence mismatch
        if (result.confidence > 0.9 && result.evidence.length < 2) {
            return "overconfidence without evidence";
        }
        
        return null;
    }

    /**
     * EXTRACTED FROM LEGACY: Enhanced result refinement
     * Suggests improvements for reasoning results
     */
    suggestImprovements(result: ReasoningResult, validations: ValidationResult[]): string[] {
        const improvements: string[] = [];

        // Check validation failures
        const failedValidations = validations.filter(v => !v.passed);
        if (failedValidations.length > 0) {
            improvements.push(`Address validation issues: ${failedValidations.map(v => v.type).join(", ")}`);
        }

        // LEGACY PATTERN: Confidence-based improvements
        if (result.confidence < this.CONFIDENCE_THRESHOLD) {
            improvements.push("Gather more evidence to increase confidence");
            improvements.push("Consider alternative perspectives or approaches");
        }

        // LEGACY PATTERN: Assumptions-based improvements
        if (result.assumptions.length > this.MAX_ASSUMPTIONS_THRESHOLD) {
            improvements.push("Reduce number of assumptions by gathering more facts");
            improvements.push("Validate key assumptions with additional evidence");
        }

        // Evidence-based improvements
        if (result.evidence.length < this.MIN_EVIDENCE_THRESHOLD) {
            improvements.push("Collect more supporting evidence");
            improvements.push("Include diverse types of evidence (facts, inferences, external sources)");
        }

        // Reasoning structure improvements
        if (result.reasoning.length < 3) {
            improvements.push("Provide more detailed reasoning steps");
            improvements.push("Include intermediate conclusions in reasoning chain");
        }

        // Check for bias-specific improvements
        const biasType = this.detectBias(result);
        if (biasType) {
            improvements.push(`Address detected bias: ${biasType}`);
            improvements.push("Consider alternative viewpoints and perspectives");
        }

        return improvements;
    }

    /**
     * Creates an overall validation summary
     */
    createValidationSummary(validations: ValidationResult[]): {
        overallPassed: boolean;
        passedCount: number;
        totalCount: number;
        criticalFailures: string[];
        warnings: string[];
        score: number;
    } {
        const passedCount = validations.filter(v => v.passed).length;
        const totalCount = validations.length;
        const overallPassed = passedCount === totalCount;

        // Categorize failures
        const failedValidations = validations.filter(v => !v.passed);
        const criticalTypes = ["logic", "completeness"];
        const criticalFailures = failedValidations
            .filter(v => criticalTypes.includes(v.type))
            .map(v => v.message);
        
        const warnings = failedValidations
            .filter(v => !criticalTypes.includes(v.type))
            .map(v => v.message);

        // Calculate validation score
        const score = passedCount / totalCount;

        return {
            overallPassed,
            passedCount,
            totalCount,
            criticalFailures,
            warnings,
            score,
        };
    }

    /**
     * ENHANCED: Advanced consistency checking
     * Performs deep consistency analysis across reasoning steps
     */
    checkAdvancedConsistency(result: ReasoningResult): {
        consistent: boolean;
        issues: string[];
        confidence: number;
    } {
        const issues: string[] = [];
        
        // Check step-by-step consistency
        for (let i = 1; i < result.reasoning.length; i++) {
            const currentStep = result.reasoning[i];
            const previousStep = result.reasoning[i - 1];
            
            if (this.stepsContradictEachOther(currentStep, previousStep)) {
                issues.push(`Contradiction between steps ${i} and ${i + 1}`);
            }
        }
        
        // Check evidence consistency
        const evidenceConsistency = this.checkEvidenceConsistency(result.evidence);
        if (!evidenceConsistency.consistent) {
            issues.push(...evidenceConsistency.issues);
        }
        
        // Calculate consistency confidence
        const consistencyScore = Math.max(0, 1 - (issues.length * 0.2));
        
        return {
            consistent: issues.length === 0,
            issues,
            confidence: consistencyScore,
        };
    }

    /**
     * Checks if two reasoning steps contradict each other
     */
    private stepsContradictEachOther(step1: string, step2: string): boolean {
        const contradictionPairs = [
            ["true", "false"],
            ["positive", "negative"],
            ["success", "failure"],
            ["valid", "invalid"],
            ["possible", "impossible"],
        ];
        
        const step1Lower = step1.toLowerCase();
        const step2Lower = step2.toLowerCase();
        
        return contradictionPairs.some(([pos, neg]) => 
            (step1Lower.includes(pos) && step2Lower.includes(neg)) ||
            (step1Lower.includes(neg) && step2Lower.includes(pos))
        );
    }

    /**
     * Checks evidence consistency
     */
    private checkEvidenceConsistency(evidence: Evidence[]): {
        consistent: boolean;
        issues: string[];
    } {
        const issues: string[] = [];
        
        // Check for contradictory evidence
        for (let i = 0; i < evidence.length; i++) {
            for (let j = i + 1; j < evidence.length; j++) {
                if (this.evidenceContradicts(evidence[i], evidence[j])) {
                    issues.push(`Evidence ${i + 1} contradicts evidence ${j + 1}`);
                }
            }
        }
        
        // Check confidence levels
        const lowConfidenceEvidence = evidence.filter(e => e.confidence < 0.5);
        if (lowConfidenceEvidence.length > evidence.length / 2) {
            issues.push("More than half of evidence has low confidence");
        }
        
        return {
            consistent: issues.length === 0,
            issues,
        };
    }

    /**
     * Checks if two pieces of evidence contradict each other
     */
    private evidenceContradicts(evidence1: Evidence, evidence2: Evidence): boolean {
        // Simple contradiction detection for evidence content
        const content1 = evidence1.content.toLowerCase();
        const content2 = evidence2.content.toLowerCase();
        
        const contradictionPairs = [
            ["supports", "refutes"],
            ["proves", "disproves"],
            ["confirms", "denies"],
            ["validates", "invalidates"],
        ];
        
        return contradictionPairs.some(([pos, neg]) => 
            (content1.includes(pos) && content2.includes(neg)) ||
            (content1.includes(neg) && content2.includes(pos))
        );
    }
}