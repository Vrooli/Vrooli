import { InputGenerationStrategy, ResourceSubType, SubroutineExecutionStrategy, type ExecutionContext, type ExecutionResult, type PassableLogger, type ResourceVersion } from "@vrooli/shared";
import { type ExecutionStrategy } from "./interfaces.js";

// Temporary type placeholders - will be replaced when components are created
interface IOProcessor {
    buildIOMapping(routine: ResourceVersion, providedInputs: Record<string, unknown>, logger: PassableLogger): Promise<any>;
    findMissingRequiredInputs(ioMapping: any): string[];
}

interface TestExecutor {
    executeTestMode(ioMapping: any): Promise<ExecutionResult>;
}

interface ContextManager {
    // Placeholder for context management methods
    createExecutionContext?(...args: any[]): any;
}

/**
 * Core routine execution logic extracted from legacy SubroutineExecutor.
 * Handles the main execution flow for single-step routines with all the
 * business logic for input validation, manual execution, and test mode.
 */
export class RoutineExecutor {
    constructor(
        private ioProcessor: IOProcessor,
        private testExecutor: TestExecutor,
        private contextManager: ContextManager,
        private logger: PassableLogger,
    ) { }

    /**
     * Migrated from SubroutineExecutor.run()
     * Main execution flow for single-step routines
     */
    async executeRoutine(
        context: ExecutionContext,
        strategy: ExecutionStrategy,
    ): Promise<ExecutionResult> {
        const { routine, config } = context;

        // Don't run for multi-step routines
        if (this.isMultiStepRoutine(routine)) {
            throw new Error("Multi-step routines should not be run directly");
        }

        // Build I/O mapping (migrated from buildIOMapping)
        const ioMapping = await this.ioProcessor.buildIOMapping(
            routine,
            this.extractProvidedInputs(context),
            this.logger,
        );

        // Check if there are missing required inputs
        const missingRequiredInputNames = this.ioProcessor.findMissingRequiredInputs(ioMapping);

        // Don't continue if we need to manually fill in missing inputs
        if (missingRequiredInputNames.length > 0 && config.decisionConfig.inputGeneration === InputGenerationStrategy.Manual) {
            return this.createWaitingResult("Missing required inputs - manual generation required", {
                missingInputs: missingRequiredInputNames,
                waitingFor: "manual_input_generation",
            });
        }

        // Don't continue if we need to manually confirm execution
        if (config.decisionConfig.subroutineExecution === SubroutineExecutionStrategy.Manual) {
            // But if we can automatically generate inputs, do that first
            if (missingRequiredInputNames.length > 0 && config.decisionConfig.inputGeneration !== InputGenerationStrategy.Manual) {
                const missingInputsResult = await this.handleMissingInputs(context, ioMapping, missingRequiredInputNames, strategy);
                if (!missingInputsResult.success) {
                    return missingInputsResult;
                }
                // Update ioMapping with generated inputs
                Object.assign(ioMapping.inputs, missingInputsResult.ioMapping.inputs);
            }

            return this.createWaitingResult("Manual execution confirmation required", {
                waitingFor: "manual_execution_confirmation",
                ioMapping,
            });
        }

        // Generate missing inputs automatically if needed
        if (missingRequiredInputNames.length > 0) {
            const missingInputsResult = await this.handleMissingInputs(context, ioMapping, missingRequiredInputNames, strategy);
            if (!missingInputsResult.success) {
                return missingInputsResult;
            }
            // Update ioMapping with generated inputs
            Object.assign(ioMapping.inputs, missingInputsResult.ioMapping.inputs);
        }

        // Perform the real run or a dummy run
        const inTestMode = config.testMode === true;
        if (inTestMode) {
            return await this.testExecutor.executeTestMode(ioMapping);
        } else {
            // Execute the actual routine using the strategy
            const updatedContext = { ...context, ioMapping };
            return await strategy.executeRoutine(updatedContext, ioMapping);
        }
    }

    /**
     * Handles missing required inputs by generating them using the strategy
     */
    private async handleMissingInputs(
        context: ExecutionContext,
        ioMapping: any, // SubroutineIOMapping
        missingInputNames: string[],
        strategy: ExecutionStrategy,
    ): Promise<ExecutionResult> {
        try {
            // Use the strategy to generate missing inputs
            return await strategy.generateMissingInputs(context, ioMapping, missingInputNames);
        } catch (error) {
            this.logger.error("Failed to generate missing inputs", { error, missingInputNames });
            return {
                success: false,
                ioMapping,
                creditsUsed: BigInt(0),
                timeElapsed: 0,
                toolCallsCount: 0,
                error: {
                    code: "INPUT_GENERATION_FAILED",
                    message: error instanceof Error ? error.message : "Failed to generate missing inputs",
                    details: error,
                },
                metadata: {
                    strategy: strategy.name,
                    missingInputNames,
                } as any, // Allow custom metadata properties
            };
        }
    }

    /**
     * Creates a result indicating the execution is waiting for user action
     */
    private createWaitingResult(message: string, metadata: Record<string, unknown>): ExecutionResult {
        return {
            success: false,
            ioMapping: {} as any, // Will be populated by caller
            creditsUsed: BigInt(0),
            timeElapsed: 0,
            toolCallsCount: 0,
            error: {
                code: "EXECUTION_WAITING",
                message,
                details: metadata,
            },
            metadata: {
                strategy: "waiting",
                ...metadata,
            } as any, // Allow custom metadata properties
        };
    }

    /**
     * Extract provided inputs from ExecutionContext
     */
    private extractProvidedInputs(context: ExecutionContext): Record<string, unknown> {
        const inputs: Record<string, unknown> = {};

        // Extract from ioMapping inputs that have values
        for (const [key, inputInfo] of Object.entries(context.ioMapping.inputs)) {
            if (typeof inputInfo === "object" && inputInfo !== null && "value" in inputInfo) {
                const typedInput = inputInfo as { value?: unknown };
                if (typedInput.value !== undefined) {
                    inputs[key] = typedInput.value;
                }
            }
        }

        return inputs;
    }

    /**
     * Migrated from SubroutineExecutor.isSingleStepRoutine()
     */
    public isSingleStepRoutine(resource: ResourceVersion): boolean {
        return Boolean(resource && resource.resourceSubType !== ResourceSubType.RoutineMultiStep
            && resource.resourceSubType && resource.resourceSubType.startsWith("Routine"));
    }

    /**
     * Migrated from SubroutineExecutor.isMultiStepRoutine()
     */
    public isMultiStepRoutine(resource: ResourceVersion): boolean {
        return resource.resourceSubType === ResourceSubType.RoutineMultiStep;
    }
} 
