import { type Logger } from "winston";
import Ajv, { type JSONSchemaType } from "ajv";
import addFormats from "ajv-formats";

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
 * ValidationEngine - Output validation and security enforcement
 * 
 * This component provides:
 * - JSON Schema-based output validation
 * - Type coercion and normalization
 * - Security scanning for malicious content
 * - Data sanitization
 * - Quality assurance checks
 * 
 * Key features:
 * - Strict schema enforcement
 * - XSS and injection prevention
 * - PII detection and masking
 * - Data quality metrics
 */
export class ValidationEngine {
    private readonly logger: Logger;
    private readonly ajv: Ajv;
    
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
        
        // Initialize AJV with common formats
        this.ajv = new Ajv({
            allErrors: true,
            coerceTypes: true,
            useDefaults: true,
            removeAdditional: "all",
        });
        
        addFormats(this.ajv);
        
        // Add custom formats
        this.addCustomFormats();
    }

    /**
     * Validates output against provided schema
     * 
     * This method:
     * 1. Validates structure against JSON Schema
     * 2. Performs security scanning
     * 3. Checks data quality
     * 4. Returns validated and sanitized data
     */
    async validate(
        output: unknown,
        schema?: Record<string, unknown>,
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
    ): Promise<{ valid: boolean; value?: unknown; error?: string }> {
        try {
            switch (type) {
                case "string":
                    return this.validateString(value, constraints);
                case "number":
                    return this.validateNumber(value, constraints);
                case "boolean":
                    return this.validateBoolean(value);
                case "array":
                    return this.validateArray(value, constraints);
                case "object":
                    return this.validateObject(value, constraints);
                case "date":
                    return this.validateDate(value, constraints);
                case "email":
                    return this.validateEmail(value);
                case "url":
                    return this.validateUrl(value);
                default:
                    return { valid: true, value };
            }
        } catch (error) {
            return {
                valid: false,
                error: error instanceof Error ? error.message : "Validation error",
            };
        }
    }

    /**
     * Private helper methods
     */
    private addCustomFormats(): void {
        // Add custom format validators
        this.ajv.addFormat("iso-date", {
            validate: (data: string) => {
                const date = new Date(data);
                return !isNaN(date.getTime());
            },
        });

        this.ajv.addFormat("positive-number", {
            type: "number",
            validate: (data: number) => data > 0,
        });

        this.ajv.addFormat("non-empty-string", {
            type: "string",
            validate: (data: string) => data.trim().length > 0,
        });
    }

    private normalizeToObject(output: unknown): Record<string, unknown> {
        if (typeof output === "object" && output !== null && !Array.isArray(output)) {
            return output as Record<string, unknown>;
        }

        // Wrap primitive values
        return { value: output };
    }

    private async validateSchema(
        data: Record<string, unknown>,
        schema: Record<string, unknown>,
    ): Promise<ValidationResult> {
        try {
            // Compile schema
            const validate = this.ajv.compile(schema as JSONSchemaType<unknown>);
            
            // Validate data
            const valid = validate(data);

            if (!valid) {
                const errors = validate.errors?.map(err => 
                    `${err.instancePath} ${err.message}`,
                ) || ["Schema validation failed"];

                return {
                    valid: false,
                    data,
                    errors,
                };
            }

            return {
                valid: true,
                data: data as Record<string, unknown>,
                errors: [],
            };

        } catch (error) {
            return {
                valid: false,
                data,
                errors: [`Schema compilation error: ${error}`],
            };
        }
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
        const qualityIssues: string[] = [];

        // Check for empty or null values in required fields
        this.checkForEmptyValues(data, "", qualityIssues);

        // Check for suspiciously large data
        this.checkDataSize(data, "", qualityIssues);

        // Check for data consistency
        this.checkDataConsistency(data, qualityIssues);

        return {
            valid: qualityIssues.length === 0,
            data,
            errors: qualityIssues,
        };
    }

    private checkForEmptyValues(
        obj: any,
        path: string,
        issues: string[],
    ): void {
        if (obj === null || obj === undefined) {
            issues.push(`Empty value at ${path}`);
        } else if (typeof obj === "string" && obj.trim() === "") {
            issues.push(`Empty string at ${path}`);
        } else if (Array.isArray(obj)) {
            if (obj.length === 0) {
                issues.push(`Empty array at ${path}`);
            }
            obj.forEach((item, index) => {
                this.checkForEmptyValues(item, `${path}[${index}]`, issues);
            });
        } else if (typeof obj === "object" && obj !== null) {
            const keys = Object.keys(obj);
            if (keys.length === 0) {
                issues.push(`Empty object at ${path}`);
            }
            for (const [key, value] of Object.entries(obj)) {
                this.checkForEmptyValues(value, `${path}.${key}`, issues);
            }
        }
    }

    private checkDataSize(
        obj: any,
        path: string,
        issues: string[],
        maxStringLength = 10000,
        maxArrayLength = 1000,
    ): void {
        if (typeof obj === "string" && obj.length > maxStringLength) {
            issues.push(`String too long at ${path} (${obj.length} chars)`);
        } else if (Array.isArray(obj) && obj.length > maxArrayLength) {
            issues.push(`Array too large at ${path} (${obj.length} items)`);
        } else if (typeof obj === "object" && obj !== null) {
            for (const [key, value] of Object.entries(obj)) {
                this.checkDataSize(value, `${path}.${key}`, issues, maxStringLength, maxArrayLength);
            }
        }
    }

    private checkDataConsistency(
        data: Record<string, unknown>,
        issues: string[],
    ): void {
        // Check for common consistency issues
        
        // Date consistency
        if (data.startDate && data.endDate) {
            const start = new Date(data.startDate as string);
            const end = new Date(data.endDate as string);
            if (start > end) {
                issues.push("Start date is after end date");
            }
        }

        // Numeric consistency
        if (data.min !== undefined && data.max !== undefined) {
            if ((data.min as number) > (data.max as number)) {
                issues.push("Minimum value is greater than maximum value");
            }
        }
    }

    private deepClone<T>(obj: T): T {
        return JSON.parse(JSON.stringify(obj));
    }

    /**
     * Type-specific validators
     */
    private validateString(
        value: unknown,
        constraints?: Record<string, unknown>,
    ): { valid: boolean; value?: unknown; error?: string } {
        const str = String(value);
        
        if (constraints?.minLength && str.length < (constraints.minLength as number)) {
            return { valid: false, error: `String too short (min: ${constraints.minLength})` };
        }
        
        if (constraints?.maxLength && str.length > (constraints.maxLength as number)) {
            return { valid: false, error: `String too long (max: ${constraints.maxLength})` };
        }
        
        if (constraints?.pattern) {
            const regex = new RegExp(constraints.pattern as string);
            if (!regex.test(str)) {
                return { valid: false, error: "String does not match pattern" };
            }
        }
        
        return { valid: true, value: str };
    }

    private validateNumber(
        value: unknown,
        constraints?: Record<string, unknown>,
    ): { valid: boolean; value?: unknown; error?: string } {
        const num = Number(value);
        
        if (isNaN(num)) {
            return { valid: false, error: "Not a valid number" };
        }
        
        if (constraints?.min !== undefined && num < (constraints.min as number)) {
            return { valid: false, error: `Number too small (min: ${constraints.min})` };
        }
        
        if (constraints?.max !== undefined && num > (constraints.max as number)) {
            return { valid: false, error: `Number too large (max: ${constraints.max})` };
        }
        
        return { valid: true, value: num };
    }

    private validateBoolean(value: unknown): { valid: boolean; value?: unknown; error?: string } {
        if (typeof value === "boolean") {
            return { valid: true, value };
        }
        
        if (typeof value === "string") {
            const lower = value.toLowerCase();
            if (["true", "false", "yes", "no", "1", "0"].includes(lower)) {
                return { valid: true, value: ["true", "yes", "1"].includes(lower) };
            }
        }
        
        return { valid: false, error: "Not a valid boolean" };
    }

    private validateArray(
        value: unknown,
        constraints?: Record<string, unknown>,
    ): { valid: boolean; value?: unknown; error?: string } {
        if (!Array.isArray(value)) {
            return { valid: false, error: "Not an array" };
        }
        
        if (constraints?.minItems && value.length < (constraints.minItems as number)) {
            return { valid: false, error: `Array too small (min: ${constraints.minItems})` };
        }
        
        if (constraints?.maxItems && value.length > (constraints.maxItems as number)) {
            return { valid: false, error: `Array too large (max: ${constraints.maxItems})` };
        }
        
        return { valid: true, value };
    }

    private validateObject(
        value: unknown,
        constraints?: Record<string, unknown>,
    ): { valid: boolean; value?: unknown; error?: string } {
        if (typeof value !== "object" || value === null || Array.isArray(value)) {
            return { valid: false, error: "Not an object" };
        }
        
        const obj = value as Record<string, unknown>;
        
        if (constraints?.required) {
            const required = constraints.required as string[];
            for (const field of required) {
                if (!(field in obj)) {
                    return { valid: false, error: `Missing required field: ${field}` };
                }
            }
        }
        
        return { valid: true, value };
    }

    private validateDate(
        value: unknown,
        constraints?: Record<string, unknown>,
    ): { valid: boolean; value?: unknown; error?: string } {
        const date = new Date(value as string);
        
        if (isNaN(date.getTime())) {
            return { valid: false, error: "Not a valid date" };
        }
        
        if (constraints?.min) {
            const minDate = new Date(constraints.min as string);
            if (date < minDate) {
                return { valid: false, error: `Date too early (min: ${constraints.min})` };
            }
        }
        
        if (constraints?.max) {
            const maxDate = new Date(constraints.max as string);
            if (date > maxDate) {
                return { valid: false, error: `Date too late (max: ${constraints.max})` };
            }
        }
        
        return { valid: true, value: date.toISOString() };
    }

    private validateEmail(value: unknown): { valid: boolean; value?: unknown; error?: string } {
        const email = String(value);
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        
        if (!emailRegex.test(email)) {
            return { valid: false, error: "Not a valid email address" };
        }
        
        return { valid: true, value: email.toLowerCase() };
    }

    private validateUrl(value: unknown): { valid: boolean; value?: unknown; error?: string } {
        try {
            const url = new URL(String(value));
            return { valid: true, value: url.toString() };
        } catch {
            return { valid: false, error: "Not a valid URL" };
        }
    }
}
