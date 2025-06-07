import { type Logger } from "winston";
import { type RunContext } from "../context/runContext.js";

/**
 * IOProcessor - Handles input/output processing for step execution
 * 
 * This component manages:
 * - Input payload preparation and transformation
 * - Data type conversion and validation
 * - Context injection from parent scopes
 * - Output formatting and schema compliance
 * - Cross-step data flow management
 * 
 * Key features:
 * - Type-safe input transformation
 * - Context-aware data enrichment
 * - Schema-based validation
 * - Efficient data serialization
 */
export class IOProcessor {
    private readonly logger: Logger;

    constructor(logger: Logger) {
        this.logger = logger;
    }

    /**
     * Builds input payload for step execution
     * 
     * This method:
     * 1. Extracts raw inputs from execution context
     * 2. Applies type conversions and transformations
     * 3. Injects context data from parent scopes
     * 4. Validates against input schema if provided
     * 5. Returns prepared payload ready for execution
     */
    async buildInputPayload(
        inputs: Record<string, unknown>,
        runContext: RunContext,
    ): Promise<Record<string, unknown>> {
        this.logger.debug("[IOProcessor] Building input payload", {
            inputKeys: Object.keys(inputs),
            runId: runContext.runId,
        });

        try {
            // 1. Clone inputs to avoid mutations
            const payload = this.deepClone(inputs);

            // 2. Apply type transformations
            const transformed = await this.applyTransformations(payload, runContext);

            // 3. Inject context data
            const enriched = await this.injectContextData(transformed, runContext);

            // 4. Resolve references to previous step outputs
            const resolved = await this.resolveReferences(enriched, runContext);

            // 5. Validate completeness
            this.validateInputCompleteness(resolved, runContext);

            this.logger.debug("[IOProcessor] Input payload built successfully", {
                payloadKeys: Object.keys(resolved),
                runId: runContext.runId,
            });

            return resolved;

        } catch (error) {
            this.logger.error("[IOProcessor] Failed to build input payload", {
                error: error instanceof Error ? error.message : String(error),
                runId: runContext.runId,
            });
            throw error;
        }
    }

    /**
     * Processes output data from step execution
     * 
     * This method:
     * 1. Extracts outputs according to schema
     * 2. Applies output transformations
     * 3. Stores for future step references
     * 4. Prepares for validation
     */
    async processOutputs(
        rawOutputs: unknown,
        outputSchema?: Record<string, unknown>,
        runContext?: RunContext,
    ): Promise<Record<string, unknown>> {
        this.logger.debug("[IOProcessor] Processing outputs", {
            hasSchema: !!outputSchema,
            runId: runContext?.runId,
        });

        try {
            // 1. Normalize output structure
            const normalized = this.normalizeOutputStructure(rawOutputs);

            // 2. Apply schema mapping if provided
            const mapped = outputSchema 
                ? await this.applySchemaMapping(normalized, outputSchema)
                : normalized;

            // 3. Apply output transformations
            const transformed = await this.applyOutputTransformations(mapped, runContext);

            // 4. Store outputs for future reference
            if (runContext) {
                await this.storeOutputsForReference(transformed, runContext);
            }

            return transformed;

        } catch (error) {
            this.logger.error("[IOProcessor] Failed to process outputs", {
                error: error instanceof Error ? error.message : String(error),
                runId: runContext?.runId,
            });
            throw error;
        }
    }

    /**
     * Private helper methods
     */
    private deepClone<T>(obj: T): T {
        return JSON.parse(JSON.stringify(obj));
    }

    private async applyTransformations(
        inputs: Record<string, unknown>,
        runContext: RunContext,
    ): Promise<Record<string, unknown>> {
        const transformed: Record<string, unknown> = {};

        for (const [key, value] of Object.entries(inputs)) {
            transformed[key] = await this.transformValue(key, value, runContext);
        }

        return transformed;
    }

    private async transformValue(
        key: string,
        value: unknown,
        runContext: RunContext,
    ): Promise<unknown> {
        // Handle special transformation cases
        if (key.endsWith("_json") && typeof value === "string") {
            try {
                return JSON.parse(value);
            } catch {
                this.logger.warn(`[IOProcessor] Failed to parse JSON for ${key}`);
                return value;
            }
        }

        if (key.endsWith("_number") && typeof value === "string") {
            const num = parseFloat(value);
            return isNaN(num) ? value : num;
        }

        if (key.endsWith("_boolean")) {
            return this.parseBoolean(value);
        }

        if (key.endsWith("_array") && typeof value === "string") {
            return this.parseArray(value);
        }

        // Handle template strings
        if (typeof value === "string" && value.includes("{{")) {
            return this.resolveTemplate(value, runContext);
        }

        return value;
    }

    private parseBoolean(value: unknown): boolean {
        if (typeof value === "boolean") return value;
        if (typeof value === "string") {
            return ["true", "yes", "1", "on"].includes(value.toLowerCase());
        }
        return !!value;
    }

    private parseArray(value: unknown): unknown[] {
        if (Array.isArray(value)) return value;
        if (typeof value === "string") {
            // Try JSON parse first
            try {
                const parsed = JSON.parse(value);
                return Array.isArray(parsed) ? parsed : [value];
            } catch {
                // Fall back to comma-separated
                return value.split(",").map(v => v.trim());
            }
        }
        return [value];
    }

    private async resolveTemplate(
        template: string,
        runContext: RunContext,
    ): Promise<string> {
        let resolved = template;

        // Resolve context variables
        const contextVars = this.extractTemplateVariables(template);
        for (const varName of contextVars) {
            const value = await this.resolveContextVariable(varName, runContext);
            if (value !== undefined) {
                resolved = resolved.replace(`{{${varName}}}`, String(value));
            }
        }

        return resolved;
    }

    private extractTemplateVariables(template: string): string[] {
        const regex = /\{\{(\w+(?:\.\w+)*)\}\}/g;
        const variables: string[] = [];
        let match;

        while ((match = regex.exec(template)) !== null) {
            variables.push(match[1]);
        }

        return variables;
    }

    private async resolveContextVariable(
        varName: string,
        runContext: RunContext,
    ): Promise<unknown> {
        const parts = varName.split(".");
        let value: any = {
            run: {
                id: runContext.runId,
                routine: runContext.routineId,
            },
            user: runContext.userData,
            env: runContext.environment,
        };

        for (const part of parts) {
            value = value?.[part];
            if (value === undefined) break;
        }

        return value;
    }

    private async injectContextData(
        inputs: Record<string, unknown>,
        runContext: RunContext,
    ): Promise<Record<string, unknown>> {
        // Inject common context data
        return {
            ...inputs,
            _context: {
                runId: runContext.runId,
                routineId: runContext.routineId,
                userId: runContext.userData?.id,
                timestamp: new Date().toISOString(),
            },
        };
    }

    private async resolveReferences(
        inputs: Record<string, unknown>,
        runContext: RunContext,
    ): Promise<Record<string, unknown>> {
        const resolved: Record<string, unknown> = {};

        for (const [key, value] of Object.entries(inputs)) {
            if (typeof value === "string" && value.startsWith("$ref:")) {
                // Reference to previous step output
                const ref = value.substring(5);
                resolved[key] = await this.resolveStepReference(ref, runContext);
            } else if (typeof value === "object" && value !== null) {
                // Recursively resolve nested references
                resolved[key] = await this.resolveReferences(
                    value as Record<string, unknown>,
                    runContext,
                );
            } else {
                resolved[key] = value;
            }
        }

        return resolved;
    }

    private async resolveStepReference(
        ref: string,
        runContext: RunContext,
    ): Promise<unknown> {
        // Format: stepId.outputKey
        const [stepId, ...outputPath] = ref.split(".");
        
        // TODO: Implement actual step output retrieval
        this.logger.debug(`[IOProcessor] Resolving reference: ${ref}`);
        
        // For now, return placeholder
        return `[Reference to ${stepId}.${outputPath.join(".")}]`;
    }

    private validateInputCompleteness(
        inputs: Record<string, unknown>,
        runContext: RunContext,
    ): void {
        // Check for required inputs based on step configuration
        const requiredInputs = runContext.stepConfig?.requiredInputs || [];
        
        for (const required of requiredInputs) {
            if (!(required in inputs) || inputs[required] === undefined) {
                throw new Error(`Required input missing: ${required}`);
            }
        }
    }

    private normalizeOutputStructure(rawOutputs: unknown): Record<string, unknown> {
        if (typeof rawOutputs === "object" && rawOutputs !== null && !Array.isArray(rawOutputs)) {
            return rawOutputs as Record<string, unknown>;
        }

        // Wrap non-object outputs
        return {
            result: rawOutputs,
        };
    }

    private async applySchemaMapping(
        outputs: Record<string, unknown>,
        schema: Record<string, unknown>,
    ): Promise<Record<string, unknown>> {
        const mapped: Record<string, unknown> = {};

        // Map outputs according to schema
        for (const [key, schemaValue] of Object.entries(schema)) {
            if (typeof schemaValue === "object" && schemaValue !== null) {
                const schemaDef = schemaValue as any;
                const sourcePath = schemaDef.source || key;
                mapped[key] = this.getValueByPath(outputs, sourcePath);
            } else {
                mapped[key] = outputs[key];
            }
        }

        return mapped;
    }

    private getValueByPath(obj: Record<string, unknown>, path: string): unknown {
        const parts = path.split(".");
        let value: any = obj;

        for (const part of parts) {
            value = value?.[part];
            if (value === undefined) break;
        }

        return value;
    }

    private async applyOutputTransformations(
        outputs: Record<string, unknown>,
        runContext?: RunContext,
    ): Promise<Record<string, unknown>> {
        // Apply any output transformations defined in configuration
        if (!runContext?.stepConfig?.outputTransformations) {
            return outputs;
        }

        const transformed = { ...outputs };
        
        // TODO: Implement transformation rules
        
        return transformed;
    }

    private async storeOutputsForReference(
        outputs: Record<string, unknown>,
        runContext: RunContext,
    ): Promise<void> {
        // TODO: Store outputs in state store for future reference
        this.logger.debug("[IOProcessor] Storing outputs for reference", {
            runId: runContext.runId,
            stepId: runContext.currentStepId,
            outputKeys: Object.keys(outputs),
        });
    }
}
