import {
    type ContextScope,
    type RunContext,
} from "@vrooli/shared";
import { logger } from "../../../events/logger.js";
import { type ExecutionRunContext } from "../tier3/context/runContext.js";

/**
 * Context validation rules and error types
 */
export enum ContextValidationError {
    INVALID_STRUCTURE = "invalid_structure",
    MISSING_REQUIRED_FIELD = "missing_required_field",
    INVALID_DATA_TYPE = "invalid_data_type",
    SENSITIVE_DATA_EXPOSED = "sensitive_data_exposed",
    VARIABLE_SIZE_EXCEEDED = "variable_size_exceeded",
    SCOPE_CONFLICT = "scope_conflict",
    CIRCULAR_REFERENCE = "circular_reference",
    INVALID_PERMISSIONS = "invalid_permissions",
    RESOURCE_LIMIT_EXCEEDED = "resource_limit_exceeded",
}

export interface ValidationResult {
    valid: boolean;
    errors: ValidationError[];
    warnings: ValidationWarning[];
    metadata: {
        validatedAt: Date;
        contextType: string;
        itemsValidated: number;
    };
}

export interface ValidationError {
    type: ContextValidationError;
    field: string;
    message: string;
    severity: "low" | "medium" | "high" | "critical";
    suggestion?: string;
}

export interface ValidationWarning {
    field: string;
    message: string;
    recommendation: string;
}

/**
 * Context Validator - Validates different context types for consistency and safety
 */
// Constants for validation limits
const BYTES_PER_MB = 1024 * 1024;
const MAX_VARIABLE_SIZE_BYTES = BYTES_PER_MB; // 1MB
const MAX_SCOPE_DEPTH = 10;
const LARGE_OBJECT_WARNING_SIZE = 100000; // 100KB

export class ContextValidator {
    private readonly maxVariableSize: number = MAX_VARIABLE_SIZE_BYTES;
    private readonly maxScopeDepth: number = MAX_SCOPE_DEPTH;
    private readonly sensitivePatterns: RegExp[] = [
        /password/i,
        /secret/i,
        /token/i,
        /credential/i,
        /api[_-]?key/i,
        /private[_-]?key/i,
    ];

    /**
     * Validates ExecutionRunContext (Tier 3)
     */
    validateExecutionRunContext(context: ExecutionRunContext): ValidationResult {
        const errors: ValidationError[] = [];
        const warnings: ValidationWarning[] = [];
        let itemsValidated = 0;

        // Validate required structure
        if (!context || typeof context !== "object") {
            errors.push({
                type: ContextValidationError.INVALID_STRUCTURE,
                field: "root",
                message: "Context must be a valid object",
                severity: "critical",
            });
            return this.createResult(false, errors, warnings, "ExecutionRunContext", itemsValidated);
        }

        // Validate required fields
        const requiredFields = ["runId", "routineId", "routineName", "userData"];
        for (const field of requiredFields) {
            if (!(field in context) || !context[field as keyof ExecutionRunContext]) {
                errors.push({
                    type: ContextValidationError.MISSING_REQUIRED_FIELD,
                    field,
                    message: `Required field '${field}' is missing or empty`,
                    severity: "high",
                });
            }
            itemsValidated++;
        }

        // Validate userData
        if (context.userData) {
            const userValidation = this.validateUserData(context.userData);
            errors.push(...userValidation.errors);
            warnings.push(...userValidation.warnings);
            itemsValidated++;
        }

        // Validate environment variables
        if (context.environment) {
            const envValidation = this.validateEnvironment(context.environment);
            errors.push(...envValidation.errors);
            warnings.push(...envValidation.warnings);
            itemsValidated += Object.keys(context.environment).length;
        }

        // Validate metadata
        if (context.getAllMetadata) {
            const metadata = context.getAllMetadata();
            const metadataValidation = this.validateVariables(metadata, "metadata");
            errors.push(...metadataValidation.errors);
            warnings.push(...metadataValidation.warnings);
            itemsValidated += Object.keys(metadata).length;
        }

        const isValid = errors.filter(e => e.severity === "critical" || e.severity === "high").length === 0;

        logger.debug("[ContextValidator] Validated ExecutionRunContext", {
            valid: isValid,
            errorCount: errors.length,
            warningCount: warnings.length,
            itemsValidated,
        });

        return this.createResult(isValid, errors, warnings, "ExecutionRunContext", itemsValidated);
    }

    /**
     * Validates shared RunContext (simple variables container)
     */
    validateRunContext(context: RunContext): ValidationResult {
        const errors: ValidationError[] = [];
        const warnings: ValidationWarning[] = [];
        let itemsValidated = 0;

        // Validate required structure
        if (!context || typeof context !== "object") {
            errors.push({
                type: ContextValidationError.INVALID_STRUCTURE,
                field: "root",
                message: "Context must be a valid object",
                severity: "critical",
            });
            return this.createResult(false, errors, warnings, "RunContext", itemsValidated);
        }

        // Validate required fields - RunContext only has variables, blackboard, scopes
        const requiredFields = ["variables", "blackboard", "scopes"] as const;
        for (const field of requiredFields) {
            if (!(field in context)) {
                errors.push({
                    type: ContextValidationError.MISSING_REQUIRED_FIELD,
                    field,
                    message: `Required field '${field}' is missing`,
                    severity: "high",
                });
            }
            itemsValidated++;
        }

        // Validate variables Record
        if (context.variables) {
            if (!this.isValidRecord(context.variables)) {
                errors.push({
                    type: ContextValidationError.INVALID_DATA_TYPE,
                    field: "variables",
                    message: "Variables must be a Record<string, unknown>",
                    severity: "high",
                });
            } else {
                const variableValidation = this.validateVariables(context.variables, "variables");
                errors.push(...variableValidation.errors);
                warnings.push(...variableValidation.warnings);
                itemsValidated += Object.keys(context.variables).length;
            }
        }

        // Validate blackboard Record
        if (context.blackboard) {
            if (!this.isValidRecord(context.blackboard)) {
                errors.push({
                    type: ContextValidationError.INVALID_DATA_TYPE,
                    field: "blackboard",
                    message: "Blackboard must be a Record<string, unknown>",
                    severity: "high",
                });
            } else {
                const blackboardValidation = this.validateVariables(context.blackboard, "blackboard");
                errors.push(...blackboardValidation.errors);
                warnings.push(...blackboardValidation.warnings);
                itemsValidated += Object.keys(context.blackboard).length;
            }
        }

        // Validate scopes array
        if (context.scopes) {
            if (!Array.isArray(context.scopes)) {
                errors.push({
                    type: ContextValidationError.INVALID_DATA_TYPE,
                    field: "scopes",
                    message: "Scopes must be an array",
                    severity: "high",
                });
            } else {
                const scopeValidation = this.validateScopes(context.scopes);
                errors.push(...scopeValidation.errors);
                warnings.push(...scopeValidation.warnings);
                itemsValidated += context.scopes.length;
            }
        }

        const isValid = errors.filter(e => e.severity === "critical" || e.severity === "high").length === 0;

        logger.debug("[ContextValidator] Validated RunContext", {
            valid: isValid,
            errorCount: errors.length,
            warningCount: warnings.length,
            itemsValidated,
        });

        return this.createResult(isValid, errors, warnings, "RunContext", itemsValidated);
    }

    /**
     * Private validation methods
     */
    private validateVariables(
        variables: Record<string, unknown>,
        fieldPrefix: string,
    ): { errors: ValidationError[], warnings: ValidationWarning[] } {
        const errors: ValidationError[] = [];
        const warnings: ValidationWarning[] = [];

        for (const [key, value] of Object.entries(variables)) {
            const fieldPath = `${fieldPrefix}.${key}`;

            // Check for sensitive data exposure
            if (this.containsSensitiveData(key)) {
                errors.push({
                    type: ContextValidationError.SENSITIVE_DATA_EXPOSED,
                    field: fieldPath,
                    message: `Variable key '${key}' appears to contain sensitive information`,
                    severity: "high",
                    suggestion: "Use secure storage for sensitive data",
                });
            }

            // Check variable size
            if (!this.validateVariableSize(value)) {
                errors.push({
                    type: ContextValidationError.VARIABLE_SIZE_EXCEEDED,
                    field: fieldPath,
                    message: `Variable '${key}' exceeds maximum size limit`,
                    severity: "medium",
                    suggestion: `Keep variables under ${this.maxVariableSize / BYTES_PER_MB}MB`,
                });
            }

            // Check for circular references
            if (this.hasCircularReference(value)) {
                errors.push({
                    type: ContextValidationError.CIRCULAR_REFERENCE,
                    field: fieldPath,
                    message: `Variable '${key}' contains circular references`,
                    severity: "high",
                    suggestion: "Remove circular references before storing",
                });
            }

            // Performance warning for large objects
            if (typeof value === "object" && value !== null) {
                const size = this.estimateObjectSize(value);
                if (size > LARGE_OBJECT_WARNING_SIZE) {
                    warnings.push({
                        field: fieldPath,
                        message: `Large object detected (${Math.round(size / 1024)}KB)`,
                        recommendation: "Consider optimizing large objects for better performance",
                    });
                }
            }
        }

        return { errors, warnings };
    }

    private validateScopes(scopes: ContextScope[]): { errors: ValidationError[], warnings: ValidationWarning[] } {
        const errors: ValidationError[] = [];
        const warnings: ValidationWarning[] = [];
        const scopeIds = new Set<string>();

        // Check for global scope
        const hasGlobalScope = scopes.some(scope => scope.id === "global");
        if (!hasGlobalScope) {
            warnings.push({
                field: "scopes",
                message: "No global scope found",
                recommendation: "Consider adding a global scope for shared variables",
            });
        }

        for (const [index, scope] of scopes.entries()) {
            const fieldPath = `scopes[${index}]`;

            // Check required fields
            if (!scope.id) {
                errors.push({
                    type: ContextValidationError.MISSING_REQUIRED_FIELD,
                    field: `${fieldPath}.id`,
                    message: "Scope ID is required",
                    severity: "high",
                });
            }

            if (!scope.name) {
                errors.push({
                    type: ContextValidationError.MISSING_REQUIRED_FIELD,
                    field: `${fieldPath}.name`,
                    message: "Scope name is required",
                    severity: "medium",
                });
            }

            // Check for duplicate scope IDs
            if (scope.id && scopeIds.has(scope.id)) {
                errors.push({
                    type: ContextValidationError.SCOPE_CONFLICT,
                    field: `${fieldPath}.id`,
                    message: `Duplicate scope ID: ${scope.id}`,
                    severity: "high",
                    suggestion: "Ensure all scope IDs are unique",
                });
            } else if (scope.id) {
                scopeIds.add(scope.id);
            }

            // Validate scope variables
            if (scope.variables) {
                const varValidation = this.validateVariables(scope.variables, `${fieldPath}.variables`);
                errors.push(...varValidation.errors);
                warnings.push(...varValidation.warnings);
            }
        }

        // Check scope depth
        if (scopes.length > this.maxScopeDepth) {
            warnings.push({
                field: "scopes",
                message: `Scope depth (${scopes.length}) exceeds recommended maximum (${this.maxScopeDepth})`,
                recommendation: "Consider flattening deeply nested scopes for better performance",
            });
        }

        return { errors, warnings };
    }

    private validateUserData(userData: Record<string, unknown>): { errors: ValidationError[], warnings: ValidationWarning[] } {
        const errors: ValidationError[] = [];
        const warnings: ValidationWarning[] = [];

        if (!userData.id) {
            errors.push({
                type: ContextValidationError.MISSING_REQUIRED_FIELD,
                field: "userData.id",
                message: "User ID is required",
                severity: "high",
            });
        }

        // Validate language preferences
        if (userData.languages && !Array.isArray(userData.languages)) {
            errors.push({
                type: ContextValidationError.INVALID_DATA_TYPE,
                field: "userData.languages",
                message: "Languages must be an array",
                severity: "medium",
            });
        }

        return { errors, warnings };
    }

    private validateEnvironment(environment: Record<string, string>): { errors: ValidationError[], warnings: ValidationWarning[] } {
        const errors: ValidationError[] = [];
        const warnings: ValidationWarning[] = [];

        for (const [key, value] of Object.entries(environment)) {
            if (typeof value !== "string") {
                errors.push({
                    type: ContextValidationError.INVALID_DATA_TYPE,
                    field: `environment.${key}`,
                    message: "Environment variables must be strings",
                    severity: "medium",
                });
            }

            if (this.containsSensitiveData(key)) {
                warnings.push({
                    field: `environment.${key}`,
                    message: "Environment variable may contain sensitive data",
                    recommendation: "Ensure sensitive environment variables are properly secured",
                });
            }
        }

        return { errors, warnings };
    }


    private validateResourceLimits(limits: Record<string, unknown>): { errors: ValidationError[], warnings: ValidationWarning[] } {
        const errors: ValidationError[] = [];
        const warnings: ValidationWarning[] = [];

        const requiredLimits = ["maxCredits", "maxDurationMs", "maxMemoryMB"];
        for (const limit of requiredLimits) {
            if (!(limit in limits) || typeof limits[limit] !== "number" || limits[limit] <= 0) {
                errors.push({
                    type: ContextValidationError.INVALID_DATA_TYPE,
                    field: `resourceLimits.${limit}`,
                    message: `${limit} must be a positive number`,
                    severity: "high",
                });
            }
        }

        return { errors, warnings };
    }

    private validatePermissions(permissions: Record<string, unknown>[]): { errors: ValidationError[], warnings: ValidationWarning[] } {
        const errors: ValidationError[] = [];
        const warnings: ValidationWarning[] = [];

        for (const [index, permission] of permissions.entries()) {
            const requiredFields = ["resource", "action", "effect"];
            for (const field of requiredFields) {
                if (!(field in permission)) {
                    errors.push({
                        type: ContextValidationError.INVALID_PERMISSIONS,
                        field: `permissions[${index}].${field}`,
                        message: `Permission field '${field}' is missing`,
                        severity: "high",
                    });
                }
            }
        }

        return { errors, warnings };
    }

    /**
     * Helper methods
     */
    private containsSensitiveData(key: string): boolean {
        return this.sensitivePatterns.some(pattern => pattern.test(key));
    }

    private validateVariableSize(value: unknown): boolean {
        try {
            const serialized = JSON.stringify(value);
            return serialized.length <= this.maxVariableSize;
        } catch {
            return false;
        }
    }

    private hasCircularReference(obj: unknown, seen = new WeakSet()): boolean {
        if (obj === null || typeof obj !== "object") return false;
        if (seen.has(obj as object)) return true;

        seen.add(obj as object);

        for (const value of Object.values(obj)) {
            if (this.hasCircularReference(value, seen)) return true;
        }

        seen.delete(obj as object);
        return false;
    }

    private estimateObjectSize(obj: unknown): number {
        try {
            return JSON.stringify(obj).length;
        } catch {
            return 0;
        }
    }

    /**
     * Type guard for validating Record<string, unknown> objects
     */
    private isValidRecord(obj: unknown): obj is Record<string, unknown> {
        return typeof obj === "object" && obj !== null && !Array.isArray(obj);
    }

    private createResult(
        valid: boolean,
        errors: ValidationError[],
        warnings: ValidationWarning[],
        contextType: string,
        itemsValidated: number,
    ): ValidationResult {
        return {
            valid,
            errors,
            warnings,
            metadata: {
                validatedAt: new Date(),
                contextType,
                itemsValidated,
            },
        };
    }
}
