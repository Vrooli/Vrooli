import { nanoid } from "@vrooli/shared";
import { type Logger } from "winston";
import * as yup from "yup";
import { EventTypes, EventUtils, type IEventBus } from "../../../events/index.js";
import { getUnifiedEventSystem } from "../../../events/initialization/eventSystemService.js";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";
import { ErrorHandler, type ComponentErrorHandler } from "../../shared/ErrorHandler.js";

/**
 * Validation result from schema validation
 */
export interface ValidationResult {
    valid: boolean;
    data: Record<string, unknown>;
    errors: string[];
}

/**
 * Security scan result
 */
interface SecurityScanResult {
    safe: boolean;
    issues: string[];
    sanitized?: Record<string, unknown>;
}

/**
 * Reasoning validation types (consolidated from strategy validation)
 */
interface ReasoningValidationResult {
    type: string;
    passed: boolean;
    message: string;
    details?: Record<string, unknown>;
}

interface _ReasoningResult {
    conclusion: unknown;
    reasoning: string[];
    evidence: _Evidence[];
    confidence: number;
    assumptions: string[];
}

interface _Evidence {
    type: "fact" | "inference" | "assumption";
    content: string;
    source?: string;
    confidence: number;
}

interface _ReasoningFramework {
    type: "logical" | "analytical" | "decision_tree" | "evidence_based";
    steps: Array<{
        id: string;
        type: string;
        description: string;
        input?: unknown;
        output?: unknown;
        reasoning?: string;
    }>;
}

/**
 * Validation Engine - Comprehensive output validation with emergent agent integration
 * 
 * This component provides validation infrastructure and emits events for emergent agents:
 * - Schema validation using Yup
 * - Event emission for security agents (XSS, PII, threats)
 * - Event emission for quality agents (bias, accuracy, consistency)
 * - Event emission for monitoring agents (performance, patterns)
 * 
 * EMERGENT CAPABILITIES:
 * Security, quality, and optimization intelligence emerges from specialized agents
 * that subscribe to validation events, not from hardcoded logic.
 */
export class ValidationEngine {
    private readonly logger: Logger;
    private readonly errorHandler: ComponentErrorHandler;
    private readonly unifiedEventBus: IEventBus | null;

    constructor(logger: Logger, eventBus?: EventBus) {
        this.logger = logger;

        // Get unified event system for modern event publishing
        this.unifiedEventBus = getUnifiedEventSystem();

        if (eventBus) {
            this.errorHandler = new ErrorHandler(logger).createComponentHandler("ValidationEngine");
        } else {
            this.errorHandler = {
                execute: async <T>(fn: () => Promise<T>) => fn(),
                handleError: (error: Error) => {
                    this.logger.error("[ValidationEngine] Error occurred", { error: error.message });
                },
            } as unknown as ComponentErrorHandler;
        }
    }

    /**
     * Helper method for publishing events using unified event system
     */
    private async publishUnifiedEvent(
        eventType: string,
        data: any,
        options?: {
            deliveryGuarantee?: "fire-and-forget" | "reliable" | "barrier-sync";
            priority?: "low" | "medium" | "high" | "critical";
            tags?: string[];
        },
    ): Promise<void> {
        if (!this.unifiedEventBus) {
            this.logger.debug("[ValidationEngine] Unified event bus not available, skipping event publication");
            return;
        }

        try {
            const event = EventUtils.createBaseEvent(
                eventType,
                data,
                EventUtils.createEventSource(3, "ValidationEngine", nanoid()),
                EventUtils.createEventMetadata(
                    options?.deliveryGuarantee || "fire-and-forget",
                    options?.priority || "medium",
                    {
                        tags: options?.tags || ["validation", "tier3"],
                    },
                ),
            );

            await this.unifiedEventBus.publish(event);

            this.logger.debug("[ValidationEngine] Published unified event", {
                eventType,
                deliveryGuarantee: options?.deliveryGuarantee,
                priority: options?.priority,
            });

        } catch (eventError) {
            this.logger.error("[ValidationEngine] Failed to publish unified event", {
                eventType,
                error: eventError instanceof Error ? eventError.message : String(eventError),
            });
        }
    }

    /**
     * MAIN VALIDATION METHOD: Validates outputs with event emission for emergent agents
     * 
     * Used by DeterministicStrategy and other execution components.
     * Emits events for security, quality, and monitoring agents to analyze.
     */
    async validateOutputs(
        outputs: Record<string, unknown>,
        schema?: Record<string, unknown> | yup.Schema,
        executionContext?: {
            executionId?: string;
            stepId?: string;
            routineId?: string;
            tier?: 1 | 2 | 3;
            strategy?: string;
        },
    ): Promise<ValidationResult> {
        const startTime = Date.now();

        return this.errorHandler.execute(
            async () => {
                const context = {
                    executionId: executionContext?.executionId || "unknown",
                    stepId: executionContext?.stepId || "unknown",
                    routineId: executionContext?.routineId || "unknown",
                    tier: executionContext?.tier || 3,
                    strategy: executionContext?.strategy || "unknown",
                };

                this.logger.debug("[ValidationEngine] Validating outputs", {
                    outputKeys: Object.keys(outputs),
                    hasSchema: !!schema,
                    context,
                });

                // 1. Schema validation first
                let validatedData = outputs;
                let schemaErrors: string[] = [];

                if (schema) {
                    const schemaResult = await this.validateSchema(outputs, schema);
                    if (!schemaResult.valid) {
                        schemaErrors = schemaResult.errors;
                    } else {
                        validatedData = schemaResult.data;
                    }
                }

                // 2. Emit raw outputs for emergent agent analysis using unified event system
                await this.publishUnifiedEvent(EventTypes.SAFETY_PRE_ACTION, {
                    executionId: context.executionId,
                    stepId: context.stepId,
                    routineId: context.routineId,
                    tier: context.tier,
                    strategy: context.strategy,
                    outputs: validatedData,
                    schema: schema ? "provided" : "none",
                    schemaValid: schemaErrors.length === 0,
                    schemaErrors,
                    timestamp: new Date(),
                }, {
                    deliveryGuarantee: "reliable",
                    priority: "high",
                    tags: ["validation", "pre-action", "security"],
                });

                // 3. Emit specific events for emergent agents
                await this.emitAgentEvents(validatedData, context);

                // 4. Basic structural validation (emergent agents will provide sophisticated analysis)
                const structuralErrors = this.performBasicStructuralValidation(validatedData);

                // 5. Emit validation completion event
                const allErrors = [...schemaErrors, ...structuralErrors];
                const isValid = allErrors.length === 0;
                const validationDuration = Date.now() - startTime;

                await this.publishUnifiedEvent(EventTypes.SAFETY_POST_ACTION, {
                    executionId: context.executionId,
                    stepId: context.stepId,
                    routineId: context.routineId,
                    tier: context.tier,
                    strategy: context.strategy,
                    valid: isValid,
                    outputs: validatedData,
                    errors: allErrors,
                    validationDuration,
                    timestamp: new Date(),
                }, {
                    deliveryGuarantee: "reliable",
                    priority: isValid ? "medium" : "high",
                    tags: ["validation", "post-action", isValid ? "success" : "failure"],
                });

                return {
                    valid: isValid,
                    data: validatedData,
                    errors: allErrors,
                };
            },
            "validateOutputs",
            executionContext,
        );
    }

    /**
     * Legacy validation method for backward compatibility
     */
    async validate(
        output: unknown,
        schema?: Record<string, unknown> | yup.Schema,
    ): Promise<ValidationResult> {
        // Convert to object and delegate to validateOutputs
        const outputs = this.normalizeToObject(output);
        return this.validateOutputs(outputs, schema);
    }

    /**
     * Validates specific value types
     */
    async validateValue(
        value: unknown,
        type: string,
        constraints?: Record<string, unknown>,
    ): Promise<boolean> {
        try {
            let schema: yup.Schema;

            switch (type) {
                case "string":
                    schema = yup.string().strict();
                    if (constraints?.minLength && typeof constraints.minLength === "number") {
                        schema = (schema as yup.StringSchema).min(constraints.minLength);
                    }
                    if (constraints?.maxLength && typeof constraints.maxLength === "number") {
                        schema = (schema as yup.StringSchema).max(constraints.maxLength);
                    }
                    if (constraints?.pattern && typeof constraints.pattern === "string") {
                        schema = (schema as yup.StringSchema).matches(new RegExp(constraints.pattern));
                    }
                    break;

                case "number":
                    schema = yup.number().strict();
                    if (constraints?.min !== undefined && typeof constraints.min === "number") {
                        schema = (schema as yup.NumberSchema).min(constraints.min);
                    }
                    if (constraints?.max !== undefined && typeof constraints.max === "number") {
                        schema = (schema as yup.NumberSchema).max(constraints.max);
                    }
                    break;

                case "boolean":
                    schema = yup.boolean().strict();
                    break;

                case "array":
                    schema = yup.array().strict();
                    if (constraints?.minItems && typeof constraints.minItems === "number") {
                        schema = (schema as yup.ArraySchema<unknown, yup.AnyObject>).min(constraints.minItems);
                    }
                    if (constraints?.maxItems && typeof constraints.maxItems === "number") {
                        schema = (schema as yup.ArraySchema<unknown, yup.AnyObject>).max(constraints.maxItems);
                    }
                    break;

                case "object":
                    schema = yup.object().strict();
                    break;

                default:
                    return true; // Unknown type, allow it
            }

            if (constraints?.required) {
                schema = schema.required();
            }

            await schema.validate(value);
            return true;

        } catch (error) {
            return false;
        }
    }

    /**
     * Validates reasoning result structure
     */
    async validateReasoning(result: unknown): Promise<ReasoningValidationResult> {
        try {
            const reasoningSchema = yup.object({
                conclusion: yup.mixed().required(),
                reasoning: yup.array().of(yup.string()).required(),
                evidence: yup.array().of(
                    yup.object({
                        type: yup.string().oneOf(["fact", "inference", "assumption"]).required(),
                        content: yup.string().required(),
                        source: yup.string(),
                        confidence: yup.number().min(0).max(1).required(),
                    }),
                ).required(),
                confidence: yup.number().min(0).max(1).required(),
                assumptions: yup.array().of(yup.string()).required(),
            });

            const validated = await reasoningSchema.validate(result);

            return {
                type: "structure",
                passed: true,
                message: "Valid reasoning structure",
                details: { validated },
            };

        } catch (error) {
            return {
                type: "structure",
                passed: false,
                message: error instanceof Error ? error.message : "Invalid reasoning structure",
            };
        }
    }

    private normalizeToObject(data: unknown): Record<string, unknown> {
        if (this.isValidRecord(data)) {
            return data;
        }

        // Wrap non-objects
        return { value: data };
    }

    private async validateSchema(
        data: Record<string, unknown>,
        schema: Record<string, unknown> | yup.Schema,
    ): Promise<ValidationResult> {
        try {
            // If schema is a yup schema, use it directly
            let yupSchema: yup.Schema;

            if (this.isYupSchema(schema)) {
                // It's already a yup schema
                yupSchema = schema as yup.Schema;
            } else {
                // Convert JSON schema-like object to yup schema
                yupSchema = this.convertToYupSchema(schema as Record<string, unknown>);
            }

            // Validate data
            const validatedData = await yupSchema.validate(data, {
                abortEarly: false,
                stripUnknown: true,
            });

            return {
                valid: true,
                data: validatedData as Record<string, unknown>,
                errors: [],
            };

        } catch (error) {
            if (error instanceof yup.ValidationError) {
                return {
                    valid: false,
                    data,
                    errors: error.errors,
                };
            }

            return {
                valid: false,
                data,
                errors: [`Schema validation error: ${error}`],
            };
        }
    }

    /**
     * Converts a JSON schema-like object to a yup schema
     */
    private convertToYupSchema(schema: Record<string, unknown>): yup.Schema {
        const shape: Record<string, yup.Schema> = {};

        // Simple conversion - can be expanded based on needs
        for (const [key, value] of Object.entries(schema)) {
            if (typeof value === "object" && value !== null) {
                const fieldSchema = value as Record<string, unknown>;

                if (fieldSchema.type === "string") {
                    let stringSchema = yup.string();
                    if (fieldSchema.required) stringSchema = stringSchema.required();
                    if (fieldSchema.minLength) stringSchema = stringSchema.min(fieldSchema.minLength as number);
                    if (fieldSchema.maxLength) stringSchema = stringSchema.max(fieldSchema.maxLength as number);
                    if (fieldSchema.pattern) stringSchema = stringSchema.matches(new RegExp(fieldSchema.pattern as string));
                    shape[key] = stringSchema;
                } else if (fieldSchema.type === "number") {
                    let numberSchema = yup.number();
                    if (fieldSchema.required) numberSchema = numberSchema.required();
                    if (fieldSchema.min !== undefined) numberSchema = numberSchema.min(fieldSchema.min as number);
                    if (fieldSchema.max !== undefined) numberSchema = numberSchema.max(fieldSchema.max as number);
                    shape[key] = numberSchema;
                } else if (fieldSchema.type === "boolean") {
                    let booleanSchema = yup.boolean();
                    if (fieldSchema.required) booleanSchema = booleanSchema.required();
                    shape[key] = booleanSchema;
                } else if (fieldSchema.type === "array") {
                    let arraySchema = yup.array();
                    if (fieldSchema.required) arraySchema = arraySchema.required();
                    shape[key] = arraySchema;
                } else if (fieldSchema.type === "object") {
                    let objectSchema = yup.object();
                    if (fieldSchema.required) objectSchema = objectSchema.required();
                    shape[key] = objectSchema;
                }
            }
        }

        return yup.object(shape);
    }

    /**
     * Emits specific events for emergent agents to analyze
     */
    private async emitAgentEvents(data: Record<string, unknown>, context: {
        executionId: string;
        stepId: string;
        routineId: string;
        tier: number;
        strategy: string;
    }): Promise<void> {
        try {
            // Emit security scan request for security agents using unified event system
            await this.publishUnifiedEvent(EventTypes.THREAT_DETECTED, {
                data,
                executionId: context.executionId,
                stepId: context.stepId,
                routineId: context.routineId,
                tier: context.tier,
                strategy: context.strategy,
                threatType: "security_scan_required",
                timestamp: new Date(),
            }, {
                deliveryGuarantee: "reliable",
                priority: "high",
                tags: ["security", "threat-detection", "scan-request"],
            });

            // Emit quality check request for quality agents using unified event system
            await this.publishUnifiedEvent(EventTypes.EXECUTION_ERROR_OCCURRED, {
                data,
                executionId: context.executionId,
                stepId: context.stepId,
                routineId: context.routineId,
                tier: context.tier,
                strategy: context.strategy,
                errorType: "quality_check_required",
                timestamp: new Date(),
            }, {
                deliveryGuarantee: "fire-and-forget",
                priority: "medium",
                tags: ["quality", "validation", "check-request"],
            });
        } catch (error) {
            // Don't fail validation if event emission fails
            this.logger.warn("[ValidationEngine] Failed to emit agent events", {
                error: error instanceof Error ? error.message : String(error),
                context,
            });
        }
    }

    /**
     * Basic structural validation - emergent agents provide sophisticated analysis
     */
    private performBasicStructuralValidation(data: Record<string, unknown>): string[] {
        const errors: string[] = [];

        // Basic checks that don't require intelligence
        if (Object.keys(data).length === 0) {
            errors.push("Output data is empty");
        }

        // Check for obvious malformed data
        for (const [key, value] of Object.entries(data)) {
            if (key.length === 0) {
                errors.push("Output contains empty key");
            }

            // Very basic null/undefined check
            if (value === null) {
                errors.push(`Output field '${key}' is null`);
            }
        }

        return errors;
    }

    /**
     * Validates output data types and constraints
     */
    async validateDataTypes(
        outputs: Record<string, unknown>,
        expectedTypes?: Record<string, string>,
    ): Promise<ValidationResult> {
        return this.errorHandler.execute(
            async () => {
                const errors: string[] = [];

                if (expectedTypes) {
                    for (const [key, expectedType] of Object.entries(expectedTypes)) {
                        const value = outputs[key];

                        if (value === undefined && expectedType !== "optional") {
                            errors.push(`Required output '${key}' is missing`);
                            continue;
                        }

                        if (value !== undefined) {
                            const actualType = Array.isArray(value) ? "array" : typeof value;
                            const normalizedExpected = expectedType.replace("optional:", "");

                            if (actualType !== normalizedExpected) {
                                errors.push(`Output '${key}' type mismatch: expected ${normalizedExpected}, got ${actualType}`);
                            }
                        }
                    }
                }

                return {
                    valid: errors.length === 0,
                    data: outputs,
                    errors,
                };
            },
            "validateDataTypes",
            { outputs, expectedTypes },
        );
    }

    /**
     * Validates execution context outputs
     */
    async validateExecutionOutputs(
        outputs: Record<string, unknown>,
        expectedOutputs?: Record<string, unknown>,
        executionContext?: {
            executionId?: string;
            stepId?: string;
            routineId?: string;
            tier?: 1 | 2 | 3;
        },
    ): Promise<ValidationResult> {
        return this.errorHandler.execute(
            async () => {
                const context = {
                    executionId: executionContext?.executionId || "unknown",
                    stepId: executionContext?.stepId || "unknown",
                    routineId: executionContext?.routineId || "unknown",
                    tier: executionContext?.tier || 3,
                };

                // Emit for emergent agents before validation using unified event system
                await this.publishUnifiedEvent(EventTypes.SAFETY_PRE_ACTION, {
                    executionId: context.executionId,
                    stepId: context.stepId,
                    routineId: context.routineId,
                    tier: context.tier,
                    outputs,
                    expectedOutputs,
                    validationType: "execution_outputs",
                    timestamp: new Date(),
                }, {
                    deliveryGuarantee: "reliable",
                    priority: "medium",
                    tags: ["validation", "pre-action", "execution"],
                });

                const errors: string[] = [];

                // Check if expected outputs are present
                if (expectedOutputs) {
                    for (const [key, config] of Object.entries(expectedOutputs)) {
                        if (!Object.prototype.hasOwnProperty.call(outputs, key)) {
                            const outputConfig = config as Record<string, unknown>;
                            if (outputConfig?.required !== false) {
                                errors.push(`Required output '${key}' is missing`);
                            }
                        }
                    }
                }

                // Basic quality checks
                const structuralErrors = this.performBasicStructuralValidation(outputs);
                errors.push(...structuralErrors);

                const isValid = errors.length === 0;

                // Emit completion event for emergent agents using unified event system
                await this.publishUnifiedEvent(EventTypes.SAFETY_POST_ACTION, {
                    executionId: context.executionId,
                    stepId: context.stepId,
                    routineId: context.routineId,
                    tier: context.tier,
                    valid: isValid,
                    outputs,
                    errors,
                    validationType: "execution_outputs",
                    timestamp: new Date(),
                }, {
                    deliveryGuarantee: "reliable",
                    priority: isValid ? "medium" : "high",
                    tags: ["validation", "post-action", isValid ? "success" : "failure"],
                });

                return {
                    valid: isValid,
                    data: outputs,
                    errors,
                };
            },
            "validateExecutionOutputs",
            executionContext,
        );
    }

    /**
     * Simplified security scan - emergent agents handle sophisticated threats
     */
    private async performSecurityScan(
        data: Record<string, unknown>,
    ): Promise<SecurityScanResult> {
        // Emit raw data for security agents to analyze using unified event system
        await this.publishUnifiedEvent(EventTypes.THREAT_DETECTED, {
            data,
            threatType: "security_scan_required",
            scanType: "output_validation",
            timestamp: new Date(),
        }, {
            deliveryGuarantee: "reliable",
            priority: "high",
            tags: ["security", "threat-detection", "output-scan"],
        });

        // Minimal baseline check - security agents provide sophisticated scanning
        return {
            safe: true, // Security agents will determine actual safety
            issues: [], // Security agents will identify threats via events
            sanitized: this.deepClone(data),
        };
    }

    /**
     * Basic quality check - emergent agents provide sophisticated analysis
     */
    private async checkDataQuality(
        data: Record<string, unknown>,
    ): Promise<ValidationResult> {
        // Emit for quality agents to analyze using unified event system
        await this.publishUnifiedEvent(EventTypes.EXECUTION_ERROR_OCCURRED, {
            data,
            errorType: "quality_check_required",
            checkType: "data_quality",
            timestamp: new Date(),
        }, {
            deliveryGuarantee: "fire-and-forget",
            priority: "medium",
            tags: ["quality", "validation", "data-check"],
        });

        // Minimal baseline check - quality agents provide sophisticated analysis
        const dataKeys = Object.keys(data);

        return {
            valid: dataKeys.length > 0, // Basic existence check
            data,
            errors: dataKeys.length === 0 ? ["No output data"] : [],
        };
    }

    /**
     * Type guard to safely detect Yup schemas
     */
    private isYupSchema(obj: unknown): obj is yup.Schema {
        return (
            obj !== null &&
            typeof obj === "object" &&
            "validate" in obj &&
            "validateSync" in obj &&
            typeof (obj as Record<string, unknown>).validate === "function" &&
            typeof (obj as Record<string, unknown>).validateSync === "function"
        );
    }

    /**
     * Type guard for validating Record<string, unknown> objects
     */
    private isValidRecord(obj: unknown): obj is Record<string, unknown> {
        return typeof obj === "object" && obj !== null && !Array.isArray(obj);
    }

    private deepClone<T>(obj: T): T {
        try {
            // Test for circular references first
            JSON.stringify(obj);
            return JSON.parse(JSON.stringify(obj));
        } catch (error) {
            this.logger.warn("[ValidationEngine] Failed to deep clone object, returning original", {
                error: error instanceof Error ? error.message : String(error),
            });
            return obj; // Return original if cloning fails
        }
    }
}
