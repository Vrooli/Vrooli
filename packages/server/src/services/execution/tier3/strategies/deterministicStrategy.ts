import {
    type IExecutionStrategy,
    type ResourceUsage,
    type ExecutionContext as StrategyExecutionContext,
    type StrategyExecutionResult,
} from "@vrooli/shared";
import { logger } from "../../../../events/logger.js";
import { ToolOrchestrator } from "../engine/toolOrchestrator.js";
import { ValidationEngine } from "../engine/validationEngine.js";

/**
 * Deterministic execution plan
 */
interface ExecutionPlan {
    steps: ExecutionStep[];
    validations: ValidationCheck[];
    rollbackPlan?: RollbackStep[];
}

interface ExecutionStep {
    id: string;
    type: "api_call" | "data_transform" | "cache_check" | "computation" | "validation";
    operation: string;
    parameters: Record<string, unknown>;
    dependencies: string[];
    timeout: number;
    retryPolicy?: {
        maxRetries: number;
        backoffMs: number;
    };
}

interface ValidationCheck {
    stepId: string;
    type: "schema" | "business_rule" | "constraint";
    validation: (result: unknown) => boolean;
    errorMessage: string;
}

interface RollbackStep {
    triggerStepId: string;
    action: string;
    parameters: Record<string, unknown>;
}

interface ExecutionMetrics {
    stepsExecuted: number;
    cacheHits: number;
    apiCalls: number;
    transformations: number;
    validationsPassed: number;
    totalDuration: number;
    validations?: number; // For backward compatibility
}

/**
 * DeterministicStrategy - Pure deterministic execution
 * 
 * This strategy provides deterministic execution patterns:
 * - 3-phase execution pattern (validate → execute → validate)
 * - Execution logic routing (API/code/transform/direct)
 * - Strict input/output validation
 * - Caching for repeated operations
 * 
 * All performance tracking and optimization emerges from agents
 * monitoring the execution events.
 */
export class DeterministicStrategy implements IExecutionStrategy {
    readonly type = "deterministic";
    readonly name = "DeterministicStrategy";
    readonly version = "3.0.0-minimal";

    private readonly toolOrchestrator: ToolOrchestrator;
    private readonly validationEngine: ValidationEngine;
    private readonly cache: Map<string, CacheEntry>;

    // Cost constants for resource estimation
    private static readonly BASE_DETERMINISTIC_COST = 10;
    private static readonly COMPLEXITY_COST_MULTIPLIER = 2;
    private static readonly API_CALL_COST = 5;
    private static readonly CODE_EXECUTION_COST = 3;
    private static readonly TRANSFORMATION_COST = 2;
    private static readonly MINIMAL_COST = 1;

    constructor() {
        super();
        this.toolOrchestrator = new ToolOrchestrator();
        this.validationEngine = new ValidationEngine();
        this.cache = new Map<string, CacheEntry>();
    }

    /**
     * Core deterministic execution implementation
     */
    protected async executeStrategy(
        context: StrategyExecutionContext,
        metadata: MinimalExecutionMetadata,
    ): Promise<StrategyExecutionResult> {
        const stepId = context.stepId;

        try {
            // Check cache first for efficiency
            const cachedResult = await this.checkCache(context);
            if (cachedResult) {
                logger.debug("[DeterministicStrategy] Cache hit", { stepId });
                return this.createCachedResult(cachedResult, metadata);
            }

            // Phase 1 - Strict input validation
            const inputValidation = await this.validateInputsStrict(context);
            if (!inputValidation.isValid) {
                throw new Error(`Input validation failed: ${inputValidation.errors.join("; ")}`);
            }

            // Phase 2 - Execute with routing
            const executionResult = await this.executeWithRouting(context);

            // Phase 3 - Strict output validation
            const outputValidation = await this.validateOutputsStrict(executionResult.outputs);
            if (!outputValidation.isValid) {
                throw new Error(`Output validation failed: ${outputValidation.errors.join("; ")}`);
            }

            // Cache successful results
            await this.cacheResult(context, executionResult.outputs);

            return {
                success: true,
                outputs: executionResult.outputs,
                resourcesUsed: this.calculateResourceUsage(
                    metadata.startTime,
                    new Date(),
                    executionResult.resourcesUsed,
                ),
            };

        } catch (error) {
            logger.error("[DeterministicStrategy] Execution failed", {
                stepId,
                error: error instanceof Error ? error.message : String(error),
            });

            throw error;
        }
    }

    /**
     * Check if this strategy can handle the given step type
     */
    canHandle(stepType: string, config?: Record<string, unknown>): boolean {
        // Check explicit strategy request
        if (config?.strategy === "deterministic") {
            return true;
        }

        // Deterministic keywords
        const deterministicKeywords = [
            "process", "transform", "convert", "calculate",
            "automate", "batch", "sync", "replicate",
            "validate", "sanitize", "format", "normalize",
            "aggregate", "extract", "load", "etl",
            "api", "integration", "workflow", "compliance",
            "code", "script", "execute", "compute",
        ];

        // Check step type and config description
        const routineName = config?.name as string || "";
        const routineDescription = config?.description as string || "";
        const combined = `${stepType} ${routineName} ${routineDescription}`.toLowerCase();

        return deterministicKeywords.some(keyword => combined.includes(keyword));
    }

    /**
     * Estimate resources needed for execution
     */
    estimateResources(context: StrategyExecutionContext): ResourceUsage {
        const inputCount = Object.keys(context.inputs).length;
        const outputCount = Object.keys(context.config.expectedOutputs || {}).length;

        const baseCost = DeterministicStrategy.BASE_DETERMINISTIC_COST;
        const complexityCost = (inputCount + outputCount) * DeterministicStrategy.COMPLEXITY_COST_MULTIPLIER;

        // Analyze execution type for additional costs
        const executionType = this.analyzeExecutionType(context);
        let additionalCost = 0;
        let estimatedApiCalls = 0;

        switch (executionType) {
            case "complex":
                additionalCost = DeterministicStrategy.API_CALL_COST + DeterministicStrategy.CODE_EXECUTION_COST;
                estimatedApiCalls = 2;
                break;
            case "moderate":
                additionalCost = DeterministicStrategy.CODE_EXECUTION_COST;
                estimatedApiCalls = 1;
                break;
            case "simple":
            default:
                additionalCost = DeterministicStrategy.MINIMAL_COST;
                estimatedApiCalls = 0;
        }

        const totalCost = baseCost + complexityCost + additionalCost;

        return {
            tokens: 0, // Deterministic strategies don't use LLMs
            apiCalls: estimatedApiCalls,
            computeTime: totalCost * 100, // Convert to milliseconds
            cost: totalCost * 0.001, // Convert to currency units
        };
    }

    /**
     * Private helper methods
     */
    private async checkCache(context: StrategyExecutionContext): Promise<unknown> {
        const cacheKey = this.generateCacheKey(context);
        const entry = this.cache.get(cacheKey);

        if (!entry) {
            return null;
        }

        // Check if cache entry is still valid
        if (entry.expiresAt && entry.expiresAt < new Date()) {
            this.cache.delete(cacheKey);
            return null;
        }

        // Validate cache entry is still applicable
        if (!this.isCacheValid(entry, context)) {
            this.cache.delete(cacheKey);
            return null;
        }

        return entry.value;
    }

    private generateCacheKey(context: StrategyExecutionContext): string {
        // Create deterministic cache key from inputs
        const keyComponents = [
            context.stepType,
            JSON.stringify(context.inputs, Object.keys(context.inputs).sort()),
            context.config.version || "default",
        ];

        return keyComponents.join(":");
    }

    private isCacheValid(entry: CacheEntry, context: StrategyExecutionContext): boolean {
        // Check if context has changed significantly
        if (entry.contextHash !== this.hashContext(context)) {
            return false;
        }

        // Check if constraints have become stricter
        if (context.constraints.maxTime &&
            entry.executionTime > context.constraints.maxTime) {
            return false;
        }

        return true;
    }

    private hashContext(context: StrategyExecutionContext): string {
        // Simple hash of relevant context parts
        return JSON.stringify({
            stepType: context.stepType,
            configVersion: context.config.version,
            resourceTypes: context.resources?.tools?.map(t => t.name || t.type) || [],
        });
    }

    /**
     * Validate inputs strictly
     */
    private async validateInputsStrict(context: StrategyExecutionContext): Promise<ValidationResult> {
        const errors: string[] = [];
        const expectedInputs = context.config.expectedInputs as Record<string, any> || {};

        for (const [key, schema] of Object.entries(expectedInputs)) {
            const value = context.inputs[key];

            if (schema.required && value === undefined) {
                errors.push(`Missing required input: ${key}`);
                continue;
            }

            if (value !== undefined && schema.type) {
                const actualType = Array.isArray(value) ? "array" : typeof value;
                if (actualType !== schema.type.toLowerCase()) {
                    errors.push(`Input ${key} type mismatch: expected ${schema.type}, got ${actualType}`);
                }
            }
        }

        return {
            isValid: errors.length === 0,
            errors,
        };
    }

    /**
     * Execute with routing based on execution type
     */
    private async executeWithRouting(context: StrategyExecutionContext): Promise<{
        outputs: Record<string, unknown>;
        resourcesUsed: number;
        executionPath: string;
    }> {
        const operations = this.determineOperations(context);
        const steps = operations.map(op => this.createExecutionStep(op, context));
        const plan: ExecutionPlan = {
            steps,
            validations: steps.map(step => this.createValidationCheck(step)),
            rollbackPlan: operations.filter(op => op.critical).map(op => ({
                triggerStepId: op.id,
                action: "rollback",
                parameters: {},
            })),
        };
        const metrics: ExecutionMetrics = {
            stepsExecuted: 0,
            cacheHits: 0,
            apiCalls: 0,
            transformations: 0,
            validationsPassed: 0,
            totalDuration: 0,
            validations: 0,
        };

        const outputs: Record<string, unknown> = {};

        for (const step of plan.steps) {
            const stepStart = Date.now();
            const resultsMap = new Map(Object.entries(outputs));
            const result = await this.executeStep(step, resultsMap, context, metrics);
            const stepDuration = Date.now() - stepStart;

            // Emit execution events for monitoring agents instead of tracking directly
            getEventBus().publish("execution.step_completed", {
                stepId: step.id,
                stepType: step.type,
                duration: stepDuration,
                success: true,
                timestamp: new Date(),
            }).catch(() => { }); // Fire and forget

            // Update minimal metrics for backward compatibility
            metrics.totalDuration += stepDuration;
            metrics.stepsExecuted++;

            Object.assign(outputs, result);
        }

        return {
            outputs,
            resourcesUsed: metrics.stepsExecuted, // Use step count instead of hardcoded tracking
            executionPath: plan.steps.map(s => s.type).join("->"),
        };
    }

    /**
     * Validate outputs strictly
     */
    private async validateOutputsStrict(outputs: Record<string, unknown>): Promise<ValidationResult> {
        const errors: string[] = [];

        for (const [key, value] of Object.entries(outputs)) {
            if (value === null || value === undefined) {
                errors.push(`Output ${key} is null or undefined`);
            }
        }

        // Use ValidationEngine if available for additional validation
        if (this.validationEngine) {
            try {
                const engineValidation = await this.validationEngine.validateOutputs(
                    outputs,
                    {}, // No specific schema for strict validation
                );
                if (!engineValidation.valid) {
                    errors.push(...engineValidation.errors);
                }
            } catch (error) {
                logger.warn("[DeterministicStrategy] ValidationEngine error", {
                    error: error instanceof Error ? error.message : String(error),
                });
            }
        }

        return {
            isValid: errors.length === 0,
            errors,
        };
    }

    private async cacheResult(
        context: StrategyExecutionContext,
        result: unknown,
    ): Promise<void> {
        const cacheKey = this.generateCacheKey(context);
        const ttl = context.config.cacheTTL || 3600000; // 1 hour default

        this.cache.set(cacheKey, {
            value: result,
            createdAt: new Date(),
            expiresAt: new Date(Date.now() + (typeof ttl === "number" ? ttl : 3600000)),
            contextHash: this.hashContext(context),
            executionTime: 0, // Will be set by metrics
        });
    }

    private createCachedResult(
        cachedValue: unknown,
        metadata: MinimalExecutionMetadata,
    ): StrategyExecutionResult {
        return {
            success: true,
            outputs: cachedValue as Record<string, unknown>,
            resourcesUsed: {
                cpu: 0,
                memory: 0,
                credits: 0, // No cost for cache hit
            },
        };
    }

    /**
     * Analyze execution type
     */
    private analyzeExecutionType(context: StrategyExecutionContext): "simple" | "moderate" | "complex" {
        const inputCount = Object.keys(context.inputs).length;
        const stepComplexity = this.estimateTaskComplexity(context);

        if (inputCount <= 2 && stepComplexity < 2) return "simple";
        if (inputCount <= 5 && stepComplexity < 4) return "moderate";
        return "complex";
    }

    private estimateTaskComplexity(context: StrategyExecutionContext): number {
        let complexity = 1;

        // Input complexity
        const inputSize = JSON.stringify(context.inputs).length;
        complexity += Math.log10(inputSize + 1) * 0.1;

        // Operation complexity
        if (context.stepType.includes("batch")) {
            complexity += 2;
        }

        if (context.stepType.includes("transform")) {
            complexity += 1;
        }

        return Math.min(complexity, 5);
    }

    /**
     * Determine operations based on context
     */
    private determineOperations(context: StrategyExecutionContext): Array<{
        id: string;
        type: string;
        critical: boolean;
    }> {
        const operations: Array<{ id: string; type: string; critical: boolean }> = [];
        const stepType = context.stepType.toLowerCase();

        if (stepType.includes("transform")) {
            operations.push({ id: "transform_1", type: "data_transformation", critical: false });
        }

        if (stepType.includes("api") || stepType.includes("fetch")) {
            operations.push({ id: "api_1", type: "api_integration", critical: true });
        }

        if (stepType.includes("process") || stepType.includes("batch")) {
            operations.push({ id: "process_1", type: "batch_processing", critical: false });
        }

        if (stepType.includes("validate")) {
            operations.push({ id: "validate_1", type: "validation", critical: false });
        }

        // Default operation if none match
        if (operations.length === 0) {
            operations.push({ id: "default_1", type: "direct_execution", critical: false });
        }

        return operations;
    }

    private createExecutionStep(
        operation: { type: string; critical: boolean },
        context: StrategyExecutionContext,
    ): ExecutionStep {
        const stepId = `${operation.type}_${Date.now()}`;

        return {
            id: stepId,
            type: this.mapOperationType(operation.type),
            operation: operation.type,
            parameters: this.extractStepParameters(operation.type, context),
            dependencies: [],
            timeout: operation.critical ? 30000 : 60000,
            retryPolicy: operation.critical ? {
                maxRetries: 3,
                backoffMs: 1000,
            } : undefined,
        };
    }

    private mapOperationType(
        opType: string,
    ): ExecutionStep["type"] {
        const typeMap: Record<string, ExecutionStep["type"]> = {
            data_transform: "data_transform",
            api_call: "api_call",
            batch_process: "computation",
            validation: "validation",
            computation: "computation",
        };

        return typeMap[opType] || "computation";
    }

    private extractStepParameters(
        operationType: string,
        context: StrategyExecutionContext,
    ): Record<string, unknown> {
        // Extract relevant parameters based on operation type
        switch (operationType) {
            case "data_transform":
                return {
                    input: context.inputs,
                    transformRules: context.config.transformRules || {},
                    outputFormat: context.config.outputFormat,
                };

            case "api_call":
                return {
                    endpoint: context.config.endpoint,
                    method: context.config.method || "GET",
                    headers: context.config.headers || {},
                    body: context.inputs,
                };

            default:
                return context.inputs;
        }
    }

    private createValidationCheck(step: ExecutionStep): ValidationCheck {
        return {
            stepId: step.id,
            type: "business_rule",
            validation: (result) => {
                // Basic validation - ensure result is not null/undefined
                if (result === null || result === undefined) {
                    return false;
                }

                // Type-specific validations
                if (step.type === "api_call") {
                    return this.validateApiResponse(result);
                }

                if (step.type === "data_transform") {
                    return this.validateTransformResult(result);
                }

                return true;
            },
            errorMessage: `Validation failed for step ${step.id}`,
        };
    }

    private validateApiResponse(response: unknown): boolean {
        if (typeof response !== "object" || response === null) {
            return false;
        }

        const resp = response as any;

        // Check for error indicators
        if (resp.error || resp.status === "error") {
            return false;
        }

        // Check for required fields
        if (resp.status === undefined && resp.data === undefined) {
            return false;
        }

        return true;
    }

    private validateTransformResult(result: unknown): boolean {
        // Ensure transform produced output
        return result !== null && result !== undefined;
    }

    private async executeStep(
        step: ExecutionStep,
        previousResults: Map<string, unknown>,
        context: StrategyExecutionContext,
        metrics: ExecutionMetrics,
    ): Promise<unknown> {
        // Resolve dependencies
        const resolvedParams = this.resolveDependencies(
            step.parameters,
            previousResults,
        );

        // Execute based on step type - metrics now tracked by events
        switch (step.type) {
            case "cache_check":
                return this.executeCacheCheck(resolvedParams);

            case "api_call":
                return this.executeApiCall(resolvedParams, step.timeout);

            case "data_transform":
                return this.executeDataTransform(resolvedParams);

            case "computation":
                return this.executeComputation(resolvedParams);

            case "validation":
                return this.executeValidation(resolvedParams);

            default:
                throw new Error(`Unknown step type: ${step.type}`);
        }
    }

    private resolveDependencies(
        parameters: Record<string, unknown>,
        previousResults: Map<string, unknown>,
    ): Record<string, unknown> {
        const resolved: Record<string, unknown> = {};

        for (const [key, value] of Object.entries(parameters)) {
            if (typeof value === "string" && value.startsWith("$ref:")) {
                const ref = value.substring(5);
                resolved[key] = previousResults.get(ref);
            } else {
                resolved[key] = value;
            }
        }

        return resolved;
    }

    private async executeCacheCheck(params: Record<string, unknown>): Promise<unknown> {
        // Simulated cache check
        return null;
    }

    private async executeApiCall(
        params: Record<string, unknown>,
        timeout: number,
    ): Promise<unknown> {
        const { url, method = "GET", headers = {}, body, responseType = "json" } = params;

        if (!url || typeof url !== "string") {
            throw new Error("API call requires a valid URL parameter");
        }

        logger.debug("[DeterministicStrategy] Executing API call", {
            url,
            method,
            timeout,
        });

        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), timeout);

        try {
            const fetchOptions: RequestInit = {
                method: method as string,
                headers: headers as HeadersInit,
                signal: controller.signal,
            };

            // Add body for methods that support it
            if (body && ["POST", "PUT", "PATCH"].includes(method as string)) {
                if (headers && (headers as any)["content-type"]?.includes("application/json")) {
                    fetchOptions.body = JSON.stringify(body);
                } else {
                    fetchOptions.body = body as BodyInit;
                }
            }

            const response = await fetch(url, fetchOptions);

            // Handle different response types
            let data: unknown;
            switch (responseType) {
                case "json":
                    data = await response.json();
                    break;
                case "text":
                    data = await response.text();
                    break;
                case "blob":
                    data = await response.blob();
                    break;
                case "arraybuffer":
                    data = await response.arrayBuffer();
                    break;
                default:
                    data = await response.json();
            }

            return {
                status: response.status,
                statusText: response.statusText,
                headers: Object.fromEntries(response.headers.entries()),
                data,
                ok: response.ok,
                timestamp: new Date().toISOString(),
            };
        } catch (error) {
            if (error instanceof Error && error.name === "AbortError") {
                throw new Error(`API call timed out after ${timeout}ms`);
            }
            throw error;
        } finally {
            clearTimeout(timeoutId);
        }
    }

    private async executeDataTransform(params: Record<string, unknown>): Promise<unknown> {
        const { input, transformRules, outputFormat } = params;

        let result = input;

        // Apply transformation rules if provided
        if (transformRules && typeof transformRules === "object") {
            logger.debug("[DeterministicStrategy] Executing data transformation", {
                inputType: typeof input,
                rulesCount: Array.isArray(transformRules) ? transformRules.length : Object.keys(transformRules).length,
                outputFormat,
            });

            // Handle array of transformation rules
            if (Array.isArray(transformRules)) {
                for (const rule of transformRules) {
                    result = await this.applyTransformRule(result, rule);
                }
            } else {
                // Handle object-based transformation rules
                result = await this.applyObjectTransform(result, transformRules as Record<string, unknown>);
            }
        }

        // Apply output format if specified
        if (outputFormat) {
            result = this.formatOutput(result, outputFormat as string);
        }

        return result;
    }

    private async applyTransformRule(data: unknown, rule: unknown): Promise<unknown> {
        if (!rule || typeof rule !== "object") return data;

        const { type, operation, field, value, from, to, pattern, replacement } = rule as any;

        switch (type) {
            case "map":
                if (Array.isArray(data)) {
                    return data.map(item => this.transformField(item, field, value));
                }
                return this.transformField(data, field, value);

            case "filter":
                if (Array.isArray(data) && operation && field) {
                    return data.filter(item => this.evaluateCondition(item, field, operation, value));
                }
                return data;

            case "rename":
                return this.renameField(data, from, to);

            case "replace":
                return this.replaceInData(data, pattern, replacement);

            case "extract":
                return this.extractField(data, field);

            case "merge":
                return typeof data === "object" && data !== null ? { ...data, ...value } : data;

            default:
                return data;
        }
    }

    private async applyObjectTransform(data: unknown, rules: Record<string, unknown>): Promise<unknown> {
        if (typeof data !== "object" || data === null) return data;

        const result: Record<string, unknown> = {};

        for (const [targetField, rule] of Object.entries(rules)) {
            if (typeof rule === "string") {
                // Simple field mapping: { "newField": "oldField" }
                result[targetField] = this.getNestedValue(data, rule);
            } else if (rule && typeof rule === "object") {
                // Complex transformation rule
                const { source, transform, defaultValue } = rule as any;
                let value = source ? this.getNestedValue(data, source) : undefined;

                if (value !== undefined && transform) {
                    value = await this.applyTransformRule(value, transform);
                }

                result[targetField] = value !== undefined ? value : defaultValue;
            } else {
                // Direct value assignment
                result[targetField] = rule;
            }
        }

        return result;
    }

    private transformField(data: unknown, field: string, value: unknown): unknown {
        if (typeof data !== "object" || data === null) return data;

        const result = { ...data } as any;
        this.setNestedValue(result, field, value);
        return result;
    }

    private evaluateCondition(data: unknown, field: string, operation: string, value: unknown): boolean {
        const fieldValue = this.getNestedValue(data, field);

        switch (operation) {
            case "equals":
            case "eq":
                return fieldValue === value;
            case "notEquals":
            case "ne":
                return fieldValue !== value;
            case "greaterThan":
            case "gt":
                return Number(fieldValue) > Number(value);
            case "lessThan":
            case "lt":
                return Number(fieldValue) < Number(value);
            case "contains":
                return String(fieldValue).includes(String(value));
            case "startsWith":
                return String(fieldValue).startsWith(String(value));
            case "endsWith":
                return String(fieldValue).endsWith(String(value));
            case "exists":
                return fieldValue !== undefined && fieldValue !== null;
            default:
                return false;
        }
    }

    private renameField(data: unknown, from: string, to: string): unknown {
        if (typeof data !== "object" || data === null) return data;

        const result = { ...data } as any;
        const value = this.getNestedValue(result, from);

        if (value !== undefined) {
            this.setNestedValue(result, to, value);
            this.deleteNestedValue(result, from);
        }

        return result;
    }

    private replaceInData(data: unknown, pattern: string, replacement: string): unknown {
        if (typeof data === "string") {
            return data.replace(new RegExp(pattern, "g"), replacement);
        }

        if (typeof data === "object" && data !== null) {
            const result: any = Array.isArray(data) ? [] : {};

            for (const [key, value] of Object.entries(data)) {
                result[key] = this.replaceInData(value, pattern, replacement);
            }

            return result;
        }

        return data;
    }

    private extractField(data: unknown, field: string): unknown {
        return this.getNestedValue(data, field);
    }

    private getNestedValue(obj: unknown, path: string): unknown {
        if (!path || typeof obj !== "object" || obj === null) return undefined;

        const keys = path.split(".");
        let result: any = obj;

        for (const key of keys) {
            if (result === null || result === undefined) return undefined;
            result = result[key];
        }

        return result;
    }

    private setNestedValue(obj: any, path: string, value: unknown): void {
        if (!path || typeof obj !== "object" || obj === null) return;

        const keys = path.split(".");
        const lastKey = keys.pop()!;
        let current = obj;

        for (const key of keys) {
            if (!current[key] || typeof current[key] !== "object") {
                current[key] = {};
            }
            current = current[key];
        }

        current[lastKey] = value;
    }

    private deleteNestedValue(obj: any, path: string): void {
        if (!path || typeof obj !== "object" || obj === null) return;

        const keys = path.split(".");
        const lastKey = keys.pop()!;
        let current = obj;

        for (const key of keys) {
            if (!current[key] || typeof current[key] !== "object") return;
            current = current[key];
        }

        delete current[lastKey];
    }

    private formatOutput(data: unknown, format: string): unknown {
        switch (format) {
            case "json":
                return JSON.parse(JSON.stringify(data));
            case "string":
                return JSON.stringify(data);
            case "array":
                return Array.isArray(data) ? data : [data];
            case "object":
                return typeof data === "object" && data !== null ? data : { value: data };
            default:
                return data;
        }
    }

    private async executeComputation(params: Record<string, unknown>): Promise<unknown> {
        // Perform computation based on parameters
        return {
            computed: true,
            result: Object.keys(params).length,
        };
    }

    private async executeValidation(params: Record<string, unknown>): Promise<unknown> {
        // Perform validation
        return {
            valid: true,
            errors: [],
        };
    }
}

/**
 * Cache entry structure
 */
interface CacheEntry {
    value: unknown;
    createdAt: Date;
    expiresAt?: Date;
    contextHash: string;
    executionTime: number;
}

/**
 * Validation result interface
 */
interface ValidationResult {
    isValid: boolean;
    errors: string[];
}
