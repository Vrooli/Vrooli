/**
 * Minimal Validation Engine - Schema validation only
 * 
 * This component provides ONLY structural validation using Yup schemas.
 * All security scanning, PII detection, and quality checks are removed.
 * 
 * These capabilities should emerge from security and quality agents that
 * subscribe to execution output events.
 * 
 * IMPORTANT: This component does NOT:
 * - Scan for XSS, SQL injection, or other security issues
 * - Detect PII (SSN, credit cards, emails)
 * - Check data quality beyond schema compliance
 * - Make any decisions about data safety
 * 
 * It ONLY validates structure against schemas.
 */

import { type Logger } from "winston";
import * as yup from "yup";
import { EventPublisher } from "../../shared/EventPublisher.js";
import { ErrorHandler, ComponentErrorHandler } from "../../shared/ErrorHandler.js";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";

/**
 * Validation result - simplified to schema validation only
 */
export interface ValidationResult {
    valid: boolean;
    data: Record<string, unknown>;
    errors: string[];
}

/**
 * Minimal Validation Engine
 * 
 * Provides basic schema validation for execution outputs.
 * Emits raw output events for security and quality agents to analyze.
 */
export class MinimalValidationEngine {
    private readonly eventPublisher: EventPublisher;
    private readonly errorHandler: ComponentErrorHandler;
    
    constructor(
        private readonly logger: Logger,
        eventBus: EventBus
    ) {
        this.eventPublisher = new EventPublisher(eventBus, logger, "MinimalValidationEngine");
        this.errorHandler = new ErrorHandler(logger, this.eventPublisher).createComponentHandler("MinimalValidationEngine");
    }
    
    /**
     * Validates output against schema and emits for agent analysis
     * 
     * This method:
     * 1. Validates structure against Yup schema (if provided)
     * 2. Emits raw output for security/quality agents
     * 3. Returns validation result
     */
    async validateAndEmit(
        output: unknown,
        executionContext: {
            executionId: string;
            stepId: string;
            routineId: string;
            tier: 1 | 2 | 3;
        },
        schema?: Record<string, unknown> | yup.Schema
    ): Promise<ValidationResult> {
        return this.errorHandler.execute(
            async () => {
                // 1. Normalize to object
                const data = this.normalizeToObject(output);
                
                // 2. Schema validation (if provided)
                let validatedData = data;
                let schemaErrors: string[] = [];
                
                if (schema) {
                    const schemaResult = await this.validateSchema(data, schema);
                    if (!schemaResult.valid) {
                        schemaErrors = schemaResult.errors;
                    } else {
                        validatedData = schemaResult.data;
                    }
                }
                
                // 3. Emit raw output for agent analysis
                // Security agents can scan for threats
                // Quality agents can check for bias, accuracy, etc.
                await this.eventPublisher.publish("execution.output.raw", {
                    executionId: executionContext.executionId,
                    stepId: executionContext.stepId,
                    routineId: executionContext.routineId,
                    tier: executionContext.tier,
                    output: validatedData,
                    schemaValid: schemaErrors.length === 0,
                    schemaErrors,
                    timestamp: new Date(),
                });
                
                // 4. Return validation result
                return {
                    valid: schemaErrors.length === 0,
                    data: validatedData,
                    errors: schemaErrors,
                };
            },
            "validateAndEmit",
            executionContext
        );
    }
    
    /**
     * Pure schema validation using Yup
     */
    async validateSchema(
        data: Record<string, unknown>,
        schema: Record<string, unknown> | yup.Schema
    ): Promise<ValidationResult> {
        try {
            // Convert plain object schema to Yup if needed
            const yupSchema = this.isYupSchema(schema) 
                ? schema 
                : this.buildYupSchema(schema);
            
            const validated = await yupSchema.validate(data, {
                abortEarly: false,
                stripUnknown: true,
            });
            
            return {
                valid: true,
                data: validated as Record<string, unknown>,
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
                errors: [error instanceof Error ? error.message : "Unknown validation error"],
            };
        }
    }
    
    /**
     * Validates a single value against type constraints
     */
    async validateValue(
        value: unknown,
        type: string,
        constraints?: Record<string, unknown>
    ): Promise<boolean> {
        try {
            const schema = this.buildValueSchema(type, constraints);
            await schema.validate(value);
            return true;
        } catch {
            return false;
        }
    }
    
    /**
     * Helper to normalize output to object
     */
    private normalizeToObject(output: unknown): Record<string, unknown> {
        if (typeof output === "object" && output !== null && !Array.isArray(output)) {
            return output as Record<string, unknown>;
        }
        
        return { value: output };
    }
    
    /**
     * Helper to check if schema is already Yup
     */
    private isYupSchema(schema: unknown): schema is yup.Schema {
        return schema instanceof yup.Schema;
    }
    
    /**
     * Helper to build Yup schema from object definition
     */
    private buildYupSchema(definition: Record<string, unknown>): yup.Schema {
        // This is a simplified version - real implementation would be more complex
        return yup.object().shape(
            Object.entries(definition).reduce((shape, [key, value]) => {
                if (typeof value === "string") {
                    shape[key] = this.buildValueSchema(value);
                } else if (typeof value === "object" && value !== null) {
                    shape[key] = this.buildValueSchema(
                        (value as any).type || "mixed",
                        value as Record<string, unknown>
                    );
                }
                return shape;
            }, {} as Record<string, yup.Schema>)
        );
    }
    
    /**
     * Helper to build schema for a single value
     */
    private buildValueSchema(type: string, constraints?: Record<string, unknown>): yup.Schema {
        let schema: yup.Schema;
        
        switch (type) {
            case "string":
                schema = yup.string();
                if (constraints?.minLength) {
                    schema = (schema as yup.StringSchema).min(constraints.minLength as number);
                }
                if (constraints?.maxLength) {
                    schema = (schema as yup.StringSchema).max(constraints.maxLength as number);
                }
                if (constraints?.pattern) {
                    schema = (schema as yup.StringSchema).matches(new RegExp(constraints.pattern as string));
                }
                break;
                
            case "number":
                schema = yup.number();
                if (constraints?.min !== undefined) {
                    schema = (schema as yup.NumberSchema).min(constraints.min as number);
                }
                if (constraints?.max !== undefined) {
                    schema = (schema as yup.NumberSchema).max(constraints.max as number);
                }
                break;
                
            case "boolean":
                schema = yup.boolean();
                break;
                
            case "array":
                schema = yup.array();
                if (constraints?.minItems) {
                    schema = (schema as yup.ArraySchema<any>).min(constraints.minItems as number);
                }
                if (constraints?.maxItems) {
                    schema = (schema as yup.ArraySchema<any>).max(constraints.maxItems as number);
                }
                break;
                
            case "object":
                schema = yup.object();
                break;
                
            default:
                schema = yup.mixed();
        }
        
        if (constraints?.required) {
            schema = schema.required();
        }
        
        return schema;
    }
}

/**
 * Example security agent that would subscribe to output events:
 * 
 * ```typescript
 * // This would be a routine deployed by teams, NOT hardcoded
 * class SecurityScanningAgent {
 *     async scanOutput(event: OutputEvent) {
 *         const output = event.output;
 *         
 *         // Check for XSS patterns
 *         if (containsXSS(output)) {
 *             await emitSecurityAlert("XSS_DETECTED", event);
 *         }
 *         
 *         // Check for PII
 *         if (containsPII(output)) {
 *             await emitSecurityAlert("PII_EXPOSED", event);
 *         }
 *         
 *         // All security intelligence emerges here
 *     }
 * }
 * ```
 */