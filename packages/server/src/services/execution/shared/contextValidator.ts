import { type Logger } from "winston";
import { 
    type RunContext,
    type ExecutionContext,
    type ProcessContext,
    type CoordinationContext,
    type ContextScope,
    DataSensitivity,
} from "@vrooli/shared";
import { type ProcessRunContext } from "../tier2/context/contextManager.js";
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
export class ContextValidator {
    private readonly logger: Logger;
    private readonly maxVariableSize: number = 1024 * 1024; // 1MB
    private readonly maxScopeDepth: number = 10;
    private readonly sensitivePatterns: RegExp[] = [
        /password/i,
        /secret/i,
        /token/i,
        /credential/i,
        /api[_-]?key/i,
        /private[_-]?key/i,
    ];

    constructor(logger: Logger) {
        this.logger = logger;
    }

    /**
     * Validates ProcessRunContext (Tier 2)
     */
    validateProcessRunContext(context: ProcessRunContext): ValidationResult {
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
            return this.createResult(false, errors, warnings, "ProcessRunContext", itemsValidated);
        }

        // Validate variables
        if (context.variables) {
            const variableValidation = this.validateVariables(context.variables, "variables");
            errors.push(...variableValidation.errors);
            warnings.push(...variableValidation.warnings);
            itemsValidated += Object.keys(context.variables).length;
        }

        // Validate blackboard
        if (context.blackboard) {
            const blackboardValidation = this.validateVariables(context.blackboard, "blackboard");
            errors.push(...blackboardValidation.errors);
            warnings.push(...blackboardValidation.warnings);
            itemsValidated += Object.keys(context.blackboard).length;
        }

        // Validate scopes
        if (context.scopes) {
            const scopeValidation = this.validateScopes(context.scopes);
            errors.push(...scopeValidation.errors);
            warnings.push(...scopeValidation.warnings);
            itemsValidated += context.scopes.length;
        } else {
            errors.push({
                type: ContextValidationError.MISSING_REQUIRED_FIELD,
                field: "scopes",
                message: "Scopes array is required",
                severity: "high",
                suggestion: "Initialize with at least a global scope",
            });
        }

        const isValid = errors.filter(e => e.severity === "critical" || e.severity === "high").length === 0;

        this.logger.debug("[ContextValidator] Validated ProcessRunContext", {
            valid: isValid,
            errorCount: errors.length,
            warningCount: warnings.length,
            itemsValidated,
        });

        return this.createResult(isValid, errors, warnings, "ProcessRunContext", itemsValidated);
    }

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

        this.logger.debug("[ContextValidator] Validated ExecutionRunContext", {
            valid: isValid,
            errorCount: errors.length,
            warningCount: warnings.length,
            itemsValidated,
        });

        return this.createResult(isValid, errors, warnings, "ExecutionRunContext", itemsValidated);
    }

    /**
     * Validates shared RunContext
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

        // Validate required fields
        const requiredFields = ["runId", "routineManifest", "variables"];
        for (const field of requiredFields) {
            if (!(field in context) || !context[field as keyof RunContext]) {
                errors.push({
                    type: ContextValidationError.MISSING_REQUIRED_FIELD,
                    field,
                    message: `Required field '${field}' is missing or empty`,
                    severity: "high",
                });
            }
            itemsValidated++;
        }

        // Validate variables Map
        if (context.variables && context.variables instanceof Map) {
            for (const [key, variable] of context.variables) {
                const varValidation = this.validateContextVariable(key, variable);
                errors.push(...varValidation.errors);
                warnings.push(...varValidation.warnings);
                itemsValidated++;
            }
        } else if (context.variables) {
            errors.push({
                type: ContextValidationError.INVALID_DATA_TYPE,
                field: "variables",
                message: "Variables must be a Map instance",
                severity: "high",
            });
        }

        // Validate resource limits
        if (context.resourceLimits) {
            const limitsValidation = this.validateResourceLimits(context.resourceLimits);
            errors.push(...limitsValidation.errors);
            warnings.push(...limitsValidation.warnings);
            itemsValidated++;
        }

        // Validate permissions
        if (context.permissions) {
            const permissionsValidation = this.validatePermissions(context.permissions);
            errors.push(...permissionsValidation.errors);
            warnings.push(...permissionsValidation.warnings);
            itemsValidated += context.permissions.length;
        }

        const isValid = errors.filter(e => e.severity === "critical" || e.severity === "high").length === 0;

        this.logger.debug("[ContextValidator] Validated RunContext", {
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
                    suggestion: `Keep variables under ${this.maxVariableSize} bytes`,
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
                if (size > 100000) { // 100KB
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

    private validateUserData(userData: any): { errors: ValidationError[], warnings: ValidationWarning[] } {
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

    private validateContextVariable(key: string, variable: any): { errors: ValidationError[], warnings: ValidationWarning[] } {
        const errors: ValidationError[] = [];
        const warnings: ValidationWarning[] = [];

        // Validate variable structure
        const requiredFields = ["key", "value", "source", "timestamp", "sensitivity", "mutable"];
        for (const field of requiredFields) {
            if (!(field in variable)) {
                errors.push({
                    type: ContextValidationError.MISSING_REQUIRED_FIELD,
                    field: `variables.${key}.${field}`,
                    message: `Variable field '${field}' is missing`,
                    severity: "medium",
                });
            }
        }

        // Validate sensitivity level
        if (variable.sensitivity && !Object.values(DataSensitivity).includes(variable.sensitivity)) {
            errors.push({
                type: ContextValidationError.INVALID_DATA_TYPE,
                field: `variables.${key}.sensitivity`,
                message: "Invalid sensitivity level",
                severity: "medium",
            });
        }

        return { errors, warnings };
    }

    private validateResourceLimits(limits: any): { errors: ValidationError[], warnings: ValidationWarning[] } {
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

    private validatePermissions(permissions: any[]): { errors: ValidationError[], warnings: ValidationWarning[] } {
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

/**
 * Factory for creating context validators
 */
export class ContextValidatorFactory {
    static create(logger: Logger): ContextValidator {
        return new ContextValidator(logger);
    }
}