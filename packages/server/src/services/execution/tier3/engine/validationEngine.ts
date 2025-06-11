import { type Logger } from "winston";
import * as yup from "yup";

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

interface ReasoningResult {
    conclusion: unknown;
    reasoning: string[];
    evidence: Evidence[];
    confidence: number;
    assumptions: string[];
}

interface Evidence {
    type: "fact" | "inference" | "assumption";
    content: string;
    source?: string;
    confidence: number;
}

interface ReasoningFramework {
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
 * Validation Engine - Validates outputs and ensures data integrity
 * 
 * This component provides comprehensive validation for execution outputs,
 * including schema validation, security scanning, and quality checks.
 * 
 * Key features:
 * - Strict schema enforcement using Yup
 * - XSS and injection prevention
 * - PII detection and masking
 * - Data quality metrics
 */
export class ValidationEngine {
    private readonly logger: Logger;
    
    // Common patterns for security scanning
    private readonly dangerousPatterns = {
        script: /<script[^>]*>[\s\S]*?<\/script>/gi,
        sqlInjection: /(\b(SELECT|INSERT|UPDATE|DELETE|DROP|UNION|CREATE|ALTER)\b.*\b(FROM|WHERE|INTO|VALUES)\b)/gi,
        commandInjection: /(;|\||&|`|\$\(|<|>)/g,
        pathTraversal: /(\.\.|\/\/|\\\\)/g,
    };

    // PII patterns for detection
    private readonly piiPatterns = {
        ssn: /\b\d{3}-\d{2}-\d{4}\b/g,
        creditCard: /\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b/g,
        email: /\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b/g,
        phone: /\b(\+?\d{1,3}[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}\b/g,
    };

    constructor(logger: Logger) {
        this.logger = logger;
    }

    /**
     * Validates output against provided schema
     * 
     * This method:
     * 1. Validates structure against Yup Schema
     * 2. Performs security scanning
     * 3. Checks data quality
     * 4. Returns validated and sanitized data
     */
    async validate(
        output: unknown,
        schema?: Record<string, unknown> | yup.Schema,
    ): Promise<ValidationResult> {
        this.logger.debug("[ValidationEngine] Starting validation");

        try {
            // 1. Ensure output is an object
            const data = this.normalizeToObject(output);

            // 2. Apply schema validation if provided
            let validatedData = data;
            if (schema) {
                const schemaResult = await this.validateSchema(data, schema);
                if (!schemaResult.valid) {
                    return schemaResult;
                }
                validatedData = schemaResult.data;
            }

            // 3. Perform security scanning
            const securityResult = await this.performSecurityScan(validatedData);
            if (!securityResult.safe) {
                return {
                    valid: false,
                    data: validatedData,
                    errors: securityResult.issues,
                };
            }

            // 4. Apply data quality checks
            const qualityResult = await this.checkDataQuality(validatedData);
            if (!qualityResult.valid) {
                return qualityResult;
            }

            // 5. Return validated result
            return {
                valid: true,
                data: securityResult.sanitized || validatedData,
                errors: [],
            };

        } catch (error) {
            this.logger.error("[ValidationEngine] Validation failed", {
                error: error instanceof Error ? error.message : String(error),
            });

            return {
                valid: false,
                data: {},
                errors: [error instanceof Error ? error.message : "Unknown validation error"],
            };
        }
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
                    })
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
        if (typeof data === "object" && data !== null && !Array.isArray(data)) {
            return data as Record<string, unknown>;
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
            
            if (schema && typeof schema === 'object' && 'validate' in schema) {
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
        const shape: Record<string, any> = {};
        
        // Simple conversion - can be expanded based on needs
        for (const [key, value] of Object.entries(schema)) {
            if (typeof value === 'object' && value !== null) {
                const fieldSchema = value as any;
                
                if (fieldSchema.type === 'string') {
                    shape[key] = yup.string();
                    if (fieldSchema.required) shape[key] = shape[key].required();
                    if (fieldSchema.minLength) shape[key] = shape[key].min(fieldSchema.minLength);
                    if (fieldSchema.maxLength) shape[key] = shape[key].max(fieldSchema.maxLength);
                    if (fieldSchema.pattern) shape[key] = shape[key].matches(new RegExp(fieldSchema.pattern));
                } else if (fieldSchema.type === 'number') {
                    shape[key] = yup.number();
                    if (fieldSchema.required) shape[key] = shape[key].required();
                    if (fieldSchema.min !== undefined) shape[key] = shape[key].min(fieldSchema.min);
                    if (fieldSchema.max !== undefined) shape[key] = shape[key].max(fieldSchema.max);
                } else if (fieldSchema.type === 'boolean') {
                    shape[key] = yup.boolean();
                    if (fieldSchema.required) shape[key] = shape[key].required();
                } else if (fieldSchema.type === 'array') {
                    shape[key] = yup.array();
                    if (fieldSchema.required) shape[key] = shape[key].required();
                } else if (fieldSchema.type === 'object') {
                    shape[key] = yup.object();
                    if (fieldSchema.required) shape[key] = shape[key].required();
                }
            }
        }
        
        return yup.object(shape);
    }

    private async performSecurityScan(
        data: Record<string, unknown>,
    ): Promise<SecurityScanResult> {
        const issues: string[] = [];
        const sanitized = this.deepClone(data);

        // Scan for dangerous patterns
        this.scanForDangerousContent(sanitized, "", issues);

        // Scan for PII
        this.scanForPII(sanitized, "", issues);

        return {
            safe: issues.length === 0,
            issues,
            sanitized,
        };
    }

    private scanForDangerousContent(
        obj: any,
        path: string,
        issues: string[],
    ): void {
        if (typeof obj === "string") {
            for (const [type, pattern] of Object.entries(this.dangerousPatterns)) {
                if (pattern.test(obj)) {
                    issues.push(`Potential ${type} detected at ${path}`);
                    // Sanitize by removing dangerous content
                    obj = obj.replace(pattern, "[SANITIZED]");
                }
            }
        } else if (Array.isArray(obj)) {
            obj.forEach((item, index) => {
                this.scanForDangerousContent(item, `${path}[${index}]`, issues);
            });
        } else if (typeof obj === "object" && obj !== null) {
            for (const [key, value] of Object.entries(obj)) {
                this.scanForDangerousContent(value, `${path}.${key}`, issues);
            }
        }
    }

    private scanForPII(
        obj: any,
        path: string,
        issues: string[],
    ): void {
        if (typeof obj === "string") {
            for (const [type, pattern] of Object.entries(this.piiPatterns)) {
                if (pattern.test(obj)) {
                    issues.push(`Potential PII (${type}) detected at ${path}`);
                    // Mask PII
                    obj = obj.replace(pattern, `[${type.toUpperCase()}_MASKED]`);
                }
            }
        } else if (Array.isArray(obj)) {
            obj.forEach((item, index) => {
                this.scanForPII(item, `${path}[${index}]`, issues);
            });
        } else if (typeof obj === "object" && obj !== null) {
            for (const [key, value] of Object.entries(obj)) {
                this.scanForPII(value, `${path}.${key}`, issues);
            }
        }
    }

    private async checkDataQuality(
        data: Record<string, unknown>,
    ): Promise<ValidationResult> {
        const issues: string[] = [];

        // Check for empty or minimal data
        const dataKeys = Object.keys(data);
        if (dataKeys.length === 0) {
            issues.push("Output contains no data");
        }

        // Check for null/undefined values in critical fields
        const criticalFields = ["result", "output", "value", "data"];
        for (const field of criticalFields) {
            if (field in data && (data[field] === null || data[field] === undefined)) {
                issues.push(`Critical field '${field}' is null or undefined`);
            }
        }

        // Check for suspiciously large data
        const jsonSize = JSON.stringify(data).length;
        if (jsonSize > 1000000) { // 1MB
            issues.push("Output data exceeds size limit");
        }

        return {
            valid: issues.length === 0,
            data,
            errors: issues,
        };
    }

    private deepClone<T>(obj: T): T {
        return JSON.parse(JSON.stringify(obj));
    }
}