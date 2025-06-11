import { type Logger } from "winston";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";

// Local type definitions
interface Location {
    id: string;
    routineId: string;
    nodeId: string;
    branchId?: string;
    index?: number;
}

interface StepInfo {
    id: string;
    name: string;
    type: string;
    description?: string;
    inputs?: Record<string, unknown>;
    outputs?: Record<string, unknown>;
    config?: Record<string, unknown>;
}

interface ContextScope {
    id: string;
    name: string;
    parentId?: string;
    variables: Record<string, unknown>;
}

interface RunContext {
    variables: Record<string, unknown>;
    blackboard: Record<string, unknown>;
    scopes: ContextScope[];
}

interface ExecutionContext {
    readonly executionId: string;
    readonly parentExecutionId?: string;
    readonly swarmId: string;
    readonly userId: string;
    readonly timestamp: Date;
    readonly correlationId: string;
    readonly stepId?: string;
    readonly routineId?: string;
    readonly stepType?: string;
    readonly inputs?: Record<string, unknown>;
    readonly config?: Record<string, unknown>;
    readonly resources?: AvailableResources;
    readonly history?: ExecutionHistory;
    readonly constraints?: ExecutionConstraints;
}

interface AvailableResources {
    models: ModelResource[];
    tools: ToolResource[];
    apis: ApiResource[];
    credits: number;
    timeLimit?: number;
}

interface ModelResource {
    provider: string;
    model: string;
    capabilities: string[];
    cost: number;
    available: boolean;
}

interface ToolResource {
    name: string;
    type: string;
    description: string;
    parameters: Record<string, unknown>;
}

interface ApiResource {
    name: string;
    endpoint: string;
    authentication?: Record<string, unknown>;
}

interface ExecutionHistory {
    recentSteps: StepExecution[];
    totalExecutions: number;
    successRate: number;
}

interface StepExecution {
    stepId: string;
    strategy: string;
    result: "success" | "failure" | "partial";
    duration: number;
}

interface ExecutionConstraints {
    maxDuration?: number;
    maxCredits?: number;
    maxMemory?: number;
    allowedModels?: string[];
    blockedTools?: string[];
    requireApproval?: boolean;
}

interface StrategyExecutionResult {
    success: boolean;
    outputs?: Record<string, unknown>;
    error?: string;
    strategy?: string;
    resourcesUsed?: {
        credits: number;
        tokens?: number;
        duration: number;
    };
}

/**
 * Step execution parameters
 */
export interface StepExecutionParams {
    runId: string;
    stepId: string;
    stepInfo: StepInfo;
    context: RunContext;
    location: Location;
}

/**
 * Step execution result
 */
export interface StepExecutionResult {
    success: boolean;
    outputs?: Record<string, unknown>;
    error?: string;
    duration: number;
    resourcesUsed?: {
        credits: number;
        time: number;
        tokens?: number;
    };
}

/**
 * StepExecutor - Bridges Tier 2 process orchestration with Tier 3 execution
 * 
 * This component is the critical interface between Tier 2's process intelligence
 * and Tier 3's execution intelligence. It handles:
 * 
 * - Step preparation and context packaging
 * - Communication with Tier 3 execution layer
 * - Result processing and context updates
 * - Resource tracking and reporting
 * - Error handling and recovery
 * 
 * The StepExecutor ensures that each step has the proper context and resources
 * for execution while maintaining the abstraction between tiers.
 */
export class StepExecutor {
    private readonly eventBus: EventBus;
    private readonly logger: Logger;
    private readonly executionTimeout: number = 300000; // 5 minutes default

    constructor(eventBus: EventBus, logger: Logger) {
        this.eventBus = eventBus;
        this.logger = logger;
    }

    /**
     * Executes a single step through Tier 3
     */
    async executeStep(params: StepExecutionParams): Promise<StepExecutionResult> {
        const startTime = Date.now();
        const { runId, stepId, stepInfo, context, location } = params;

        this.logger.info("[StepExecutor] Executing step", {
            runId,
            stepId,
            stepType: stepInfo.type,
            stepName: stepInfo.name,
        });

        try {
            // Prepare execution context for Tier 3
            const executionContext = await this.prepareExecutionContext(
                params,
            );

            // Emit step execution request to Tier 3
            const requestId = await this.requestExecution(executionContext);

            // Wait for execution result
            const result = await this.waitForResult(requestId, this.executionTimeout);

            // Process and return result
            const duration = Date.now() - startTime;
            
            if (result.success) {
                this.logger.info("[StepExecutor] Step completed successfully", {
                    runId,
                    stepId,
                    duration,
                    outputKeys: Object.keys(result.outputs || {}),
                });

                return {
                    success: true,
                    outputs: result.outputs,
                    duration,
                    resourcesUsed: result.resourcesUsed,
                };
            } else {
                this.logger.error("[StepExecutor] Step execution failed", {
                    runId,
                    stepId,
                    duration,
                    error: result.error,
                });

                return {
                    success: false,
                    error: result.error || "Execution failed",
                    duration,
                };
            }

        } catch (error) {
            const duration = Date.now() - startTime;
            
            this.logger.error("[StepExecutor] Step execution error", {
                runId,
                stepId,
                duration,
                error: error instanceof Error ? error.message : String(error),
            });

            return {
                success: false,
                error: error instanceof Error ? error.message : String(error),
                duration,
            };
        }
    }

    /**
     * Prepares execution context for Tier 3
     */
    private async prepareExecutionContext(
        params: StepExecutionParams,
    ): Promise<ExecutionContext> {
        const { runId, stepId, stepInfo, context, location } = params;

        // Extract variables from context
        const variables = this.extractVariables(context);

        // Build execution context
        const executionContext: ExecutionContext = {
            stepId,
            stepType: stepInfo.type,
            prompt: stepInfo.description || `Execute ${stepInfo.name}`,
            inputs: {
                ...variables,
                ...stepInfo.inputs,
            },
            availableTools: this.getAvailableTools(stepInfo),
            constraints: {
                maxTokens: 4000,
                maxExecutionTime: this.executionTimeout,
                requiredOutputs: Object.keys(stepInfo.outputs || {}),
            },
            metadata: {
                runId,
                location,
                stepName: stepInfo.name,
                config: stepInfo.config,
            },
        };

        this.logger.debug("[StepExecutor] Prepared execution context", {
            runId,
            stepId,
            inputCount: Object.keys(executionContext.inputs).length,
            toolCount: executionContext.availableTools.length,
        });

        return executionContext;
    }

    /**
     * Extracts variables from run context
     */
    private extractVariables(context: RunContext): Record<string, unknown> {
        const variables: Record<string, unknown> = {};

        // Add global variables
        Object.assign(variables, context.variables);

        // Add variables from all scopes (in order)
        for (const scope of context.scopes) {
            Object.assign(variables, scope.variables);
        }

        return variables;
    }

    /**
     * Gets available tools for step
     */
    private getAvailableTools(stepInfo: StepInfo): string[] {
        const tools: string[] = [];

        // Add step-specific tools
        if (stepInfo.config?.tools) {
            tools.push(...(stepInfo.config.tools as string[]));
        }

        // Add default tools based on step type
        switch (stepInfo.type) {
            case "action":
                tools.push("execute_code", "call_api", "transform_data");
                break;
            case "decision":
                tools.push("evaluate_condition", "compare_values");
                break;
            case "loop":
                tools.push("iterate_collection", "check_condition");
                break;
            case "parallel":
                tools.push("split_data", "merge_results");
                break;
            case "subroutine":
                tools.push("call_routine", "pass_context");
                break;
        }

        // Remove duplicates
        return [...new Set(tools)];
    }

    /**
     * Requests execution from Tier 3
     */
    private async requestExecution(
        context: ExecutionContext,
    ): Promise<string> {
        const requestId = `exec-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;

        // Publish execution request to Tier 3
        await this.eventBus.publish("execution.request", {
            requestId,
            context,
            timestamp: new Date(),
        });

        this.logger.debug("[StepExecutor] Requested execution from Tier 3", {
            requestId,
            stepId: context.stepId,
        });

        return requestId;
    }

    /**
     * Waits for execution result from Tier 3
     */
    private async waitForResult(
        requestId: string,
        timeout: number,
    ): Promise<StrategyExecutionResult> {
        return new Promise((resolve, reject) => {
            let timeoutHandle: NodeJS.Timeout;
            let unsubscribe: (() => void) | null = null;

            // Set timeout
            timeoutHandle = setTimeout(() => {
                if (unsubscribe) unsubscribe();
                reject(new Error(`Execution timeout after ${timeout}ms`));
            }, timeout);

            // Subscribe to results
            unsubscribe = this.eventBus.subscribe("execution.result", async (event) => {
                if (event.requestId === requestId) {
                    clearTimeout(timeoutHandle);
                    if (unsubscribe) unsubscribe();
                    resolve(event.result);
                }
            });
        });
    }

    /**
     * Handles step-specific execution logic
     */
    async executeStepType(
        stepInfo: StepInfo,
        context: RunContext,
    ): Promise<Record<string, unknown>> {
        switch (stepInfo.type) {
            case "action":
                return this.executeAction(stepInfo, context);
            case "decision":
                return this.executeDecision(stepInfo, context);
            case "loop":
                return this.executeLoop(stepInfo, context);
            case "parallel":
                return this.executeParallel(stepInfo, context);
            case "subroutine":
                return this.executeSubroutine(stepInfo, context);
            default:
                throw new Error(`Unknown step type: ${stepInfo.type}`);
        }
    }

    /**
     * Executes an action step
     */
    private async executeAction(
        stepInfo: StepInfo,
        context: RunContext,
    ): Promise<Record<string, unknown>> {
        // Action steps perform operations and produce outputs
        // This is handled by Tier 3 execution
        return {};
    }

    /**
     * Executes a decision step
     */
    private async executeDecision(
        stepInfo: StepInfo,
        context: RunContext,
    ): Promise<Record<string, unknown>> {
        // Decision steps evaluate conditions
        // The result determines the next path
        return { decision: true };
    }

    /**
     * Executes a loop step
     */
    private async executeLoop(
        stepInfo: StepInfo,
        context: RunContext,
    ): Promise<Record<string, unknown>> {
        // Loop steps manage iteration state
        return { continueLoop: false };
    }

    /**
     * Executes a parallel step
     */
    private async executeParallel(
        stepInfo: StepInfo,
        context: RunContext,
    ): Promise<Record<string, unknown>> {
        // Parallel steps coordinate branch execution
        // This is handled by BranchCoordinator
        return {};
    }

    /**
     * Executes a subroutine step
     */
    private async executeSubroutine(
        stepInfo: StepInfo,
        context: RunContext,
    ): Promise<Record<string, unknown>> {
        // Subroutine steps invoke other routines
        // This creates a nested run
        return {};
    }

    /**
     * Validates step outputs
     */
    async validateOutputs(
        stepInfo: StepInfo,
        outputs: Record<string, unknown>,
    ): Promise<boolean> {
        if (!stepInfo.outputs) {
            return true;
        }

        // Check required outputs are present
        for (const outputName of Object.keys(stepInfo.outputs)) {
            if (!(outputName in outputs)) {
                this.logger.warn("[StepExecutor] Missing required output", {
                    stepId: stepInfo.id,
                    outputName,
                });
                return false;
            }
        }

        return true;
    }
}
