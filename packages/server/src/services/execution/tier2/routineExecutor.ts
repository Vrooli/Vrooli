import { EventTypes, type ExecutionResult, type RoutineExecutionInput, type RunContext, type TierExecutionRequest, type RunState, type StepExecutionInput, type ExecutionStrategy, type CoreResourceAllocation, type SessionUser } from "@vrooli/shared";
import { logger } from "../../../events/logger.js";
import { EventPublisher } from "../../events/publisher.js";
import type { ISwarmContextManager } from "../shared/SwarmContextManager.js";
import type { StepExecutor } from "../tier3/stepExecutor.js";
import { getNavigator } from "./navigators/navigatorFactory.js";
import { RoutineStateMachine } from "./routineStateMachine.js";
import type { IRunContextManager } from "./runContextManager.js";
import type { INavigator, Location, StepInfo } from "./types.js";

/**
 * RoutineExecutor - Lean execution using focused components
 * 
 * This class demonstrates the clean lean architecture for routine execution:
 * - RoutineStateMachine for state management (persisted via RunContextManager)
 * - Direct event emission for step-level events
 * - Simple navigation factory for navigation strategies
 * - Simple permission checking for security validation
 * - RunContextManager for detailed execution state persistence
 */
export class RoutineExecutor {
    private readonly stateMachine: RoutineStateMachine;
    private readonly stepExecutor: StepExecutor;
    private readonly runContextManager?: IRunContextManager;

    constructor(
        contextManager: ISwarmContextManager,
        stepExecutor: StepExecutor,
        contextId: string,
        runContextManager?: IRunContextManager,
        userId?: string,
        parentSwarmId?: string,
    ) {
        this.stepExecutor = stepExecutor;
        this.runContextManager = runContextManager;

        // Create lean components with RunContextManager integration
        this.stateMachine = new RoutineStateMachine(
            contextId,
            contextManager,
            runContextManager,
            userId,
            parentSwarmId,
        );
    }

    /**
     * Get the state machine instance (for registry access)
     */
    public getStateMachine(): RoutineStateMachine {
        return this.stateMachine;
    }

    /**
     * Execute a routine using lean architecture with resource management
     */
    async execute<TInput extends RoutineExecutionInput, TOutput>(
        request: TierExecutionRequest<TInput>,
    ): Promise<ExecutionResult<TOutput>> {
        const { context, input } = request;
        const swarmId = context.swarmId;
        const resourceVersionId = input.resourceVersionId;

        logger.info("[RoutineExecutor] Starting lean execution with RunContextManager", {
            swarmId,
            resourceVersionId,
            hasRunContextManager: !!this.runContextManager,
        });

        // Track execution start time for duration calculation
        const executionStartTime = Date.now();

        try {
            // Step 1: Initialize execution state with resource allocation
            await this.stateMachine.initializeExecution(
                swarmId,
                resourceVersionId,
                context.swarmId,
                input.runId, // Pass runId for resumption support
            );

            // Step 2: Start execution (this will emit the run started event)
            await this.stateMachine.start();

            // Step 3: Navigate through routine structure using simplified navigation
            const navigator = getNavigator(input.resourceVersionId);
            if (!navigator) {
                throw new Error("No navigator available for this routine type");
            }
            const startLocation = navigator.getStartLocation(input.resourceVersionId);

            // Step 4: Execute routine steps with resource tracking
            const result = await this.executeRoutineSteps(
                swarmId,
                input.resourceVersionId,
                startLocation,
                context,
            );

            // Step 5: Complete execution with results and resource cleanup (this will emit the run completed event)
            await this.stateMachine.complete(result);

            logger.info("[RoutineExecutor] Lean execution completed", {
                swarmId,
                resourceUsage: this.stateMachine.getResourceUsage(),
            });

            return {
                success: true,
                result: result as TOutput,
                resourcesUsed: this.stateMachine.getResourceUsage(),
                duration: Date.now() - executionStartTime,
                metadata: {
                    swarmId,
                    strategy: "lean",
                    componentCount: 2, // stateMachine, navigator
                },
            } as ExecutionResult<TOutput>;

        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);

            logger.error("[RoutineExecutor] Lean execution failed", {
                swarmId,
                error: errorMessage,
                resourceUsage: this.stateMachine.getResourceUsage(),
            });

            // Fail the state machine (this will emit the run failed event)
            await this.stateMachine.fail(errorMessage);

            return {
                success: false,
                error: {
                    code: "ROUTINE_EXECUTION_FAILED",
                    message: errorMessage,
                    tier: "tier2",
                    type: "execution_error",
                    strategy: "lean",
                    phase: "execution",
                    cause: error instanceof Error ? error : undefined,
                    context: {
                        swarmId,
                        resourceVersionId,
                        currentState: this.stateMachine.getState(),
                    },
                },
                resourcesUsed: this.stateMachine.getResourceUsage(),
                duration: Date.now() - executionStartTime,
                metadata: {
                    swarmId,
                    strategy: "lean",
                    failurePoint: "execution",
                },
            } as ExecutionResult<TOutput>;
        }
    }

    /**
     * Execute routine steps using proper navigation loop with resource tracking
     */
    private async executeRoutineSteps(
        swarmId: string,
        routine: any,
        startLocation: any,
        context: any,
    ): Promise<any> {
        logger.info("[RoutineExecutor] Starting navigation loop", {
            swarmId,
            routineId: routine.id || routine,
        });

        // Get navigator using existing factory
        const navigator = getNavigator(routine.id || routine);
        if (!navigator) {
            throw new Error(`No navigator found for routine ${routine.id || routine}`);
        }

        // Use existing INavigator interface
        let currentLocation: Location = {
            id: startLocation.id || "start",
            routineId: routine.id || routine,
            nodeId: startLocation.nodeId || "start",
        };

        const stepResults: any[] = [];
        const MAX_STEPS = 1000; // Safety limit
        let stepCount = 0;

        // Navigation loop using existing navigator methods
        while (!navigator.isEndLocation(routine, currentLocation) && stepCount < MAX_STEPS) {
            stepCount++;
            
            // Get step info using existing navigator interface
            const stepInfo: StepInfo = navigator.getStepInfo(routine, currentLocation);
            
            logger.info("[RoutineExecutor] Executing step", {
                stepId: stepInfo.id,
                stepType: stepInfo.type,
                stepCount,
            });

            // Execute single step (keeping existing logic)
            const stepResult = await this.executeSingleStep(
                swarmId,
                stepInfo,
                context,
            );

            stepResults.push(stepResult);

            // Get next locations using existing navigator
            const nextLocations = navigator.getNextLocations(
                routine, 
                currentLocation, 
                { variables: context.variables || {} },
            );

            if (nextLocations.length === 0) {
                logger.info("[RoutineExecutor] No more steps, ending navigation");
                break;
            }

            // Move to next location (take first for sequential execution)
            currentLocation = nextLocations[0];
        }

        logger.info("[RoutineExecutor] Navigation completed", {
            swarmId,
            stepsExecuted: stepCount,
            totalResults: stepResults.length,
        });

        // Return aggregated outputs
        return stepResults.reduce((outputs, result) => ({
            ...outputs,
            ...result.outputs,
        }), {});
    }

    /**
     * Execute a single step with resource tracking and event emission
     */
    private async executeSingleStep(
        swarmId: string,
        stepInfo: StepInfo,
        context: TierExecutionRequest<RoutineExecutionInput>["context"],
    ): Promise<any> {
        const stepId = stepInfo.id;
        const stepType = stepInfo.type;
        // Step 1: Allocate step-level resources if RunContextManager available
        let stepAllocation;
        if (this.runContextManager) {
            try {
                stepAllocation = await this.runContextManager.allocateForStep(swarmId, {
                    stepId,
                    stepType,
                    estimatedRequirements: {
                        credits: "50", // Estimated credits for this step
                        durationMs: 60000, // 1 minute
                        memoryMB: 128,
                    },
                });

                logger.info("[RoutineExecutor] Step resources allocated", {
                    swarmId,
                    stepId,
                    allocation: stepAllocation,
                });
            } catch (error) {
                logger.error("[RoutineExecutor] Failed to allocate step resources", {
                    swarmId,
                    stepId,
                    error: error instanceof Error ? error.message : String(error),
                });
                throw error;
            }
        }

        try {
            // Step 2: Validate permissions with simple role-based check
            const permissionCheck = await this.checkStepPermission(
                swarmId,
                stepId,
                {
                    variables: context.variables || {},
                    blackboard: {},
                },
            );

            if (!permissionCheck) {
                throw new Error("Permission denied for routine execution");
            }

            // Step 3: Emit step started event
            const { proceed: stepStartProceed, reason: stepStartReason } = await EventPublisher.emit(EventTypes.RUN.STEP_STARTED, {
                runId: swarmId,
                stepId,
                name: stepType,
                inputs: {},
                parentSwarmId: swarmId,
            });

            if (!stepStartProceed) {
                logger.warn("[RoutineExecutor] Step start event blocked", {
                    runId: swarmId,
                    stepId,
                    reason: stepStartReason,
                });
                // Continue anyway - step tracking must proceed
            }

            // Step 4: Track step execution start
            const stepStartTime = Date.now();

            // Step 5: Delegate actual execution to Tier 3
            const tier3Request = this.createTier3Request(
                stepInfo,
                context,
                stepAllocation,
            );

            const tier3Result = await this.stepExecutor.execute(tier3Request);

            // Step 6: Calculate actual step resource usage
            const actualUsage = {
                credits: "30", // Would be calculated based on actual execution
                durationMs: Date.now() - stepStartTime,
                memoryMB: 100, // Would be measured
                stepsExecuted: 1,
            };

            // Step 7: Update state machine resource tracking
            this.stateMachine.addCreditsUsed(actualUsage.credits);
            this.stateMachine.incrementStepCount();

            // Step 8: Release step resources back to run
            if (this.runContextManager && stepAllocation) {
                await this.runContextManager.releaseFromStep(
                    swarmId,
                    stepId,
                    actualUsage,
                );
            }

            // Step 9: Emit step completed event
            const { proceed: stepCompleteProceed, reason: stepCompleteReason } = await EventPublisher.emit(EventTypes.RUN.STEP_COMPLETED, {
                runId: swarmId,
                routineId: "unknown", // Will be filled in from context if available
                stepId,
                outputs: tier3Result.outputs || {},
                success: tier3Result.success,
                metrics: {
                    duration: actualUsage.durationMs,
                    creditsUsed: actualUsage.credits,
                    stepsExecuted: 1,
                    stepsFailed: tier3Result.success ? 0 : 1,
                },
                error: tier3Result.success ? undefined : (tier3Result as any).error,
                parentSwarmId: swarmId,
            });

            if (!stepCompleteProceed) {
                logger.warn("[RoutineExecutor] Step completion event blocked", {
                    runId: swarmId,
                    stepId,
                    reason: stepCompleteReason,
                });
                // Continue anyway - completion tracking must proceed
            }

            return tier3Result;

        } catch (error) {
            // If step fails, still release resources
            if (this.runContextManager && stepAllocation) {
                const partialUsage = {
                    credits: "15", // Partial usage on failure
                    durationMs: Date.now() - Date.now(), // Calculate actual time
                    memoryMB: 50,
                    stepsExecuted: 0,
                };

                await this.runContextManager.releaseFromStep(
                    swarmId,
                    stepId,
                    partialUsage,
                );
            }

            throw error;
        }
    }

    /**
     * Create a properly formatted TierExecutionRequest for Tier 3
     * 
     * This method transforms the StepInfo from the navigator and the context
     * into the standardized TierExecutionRequest format expected by Tier 3.
     */
    private createTier3Request(
        stepInfo: StepInfo,
        context: TierExecutionRequest<RoutineExecutionInput>["context"],
        allocation?: CoreResourceAllocation,
    ): TierExecutionRequest<StepExecutionInput> {
        // Extract tool name from step config or type
        const toolName = this.extractToolName(stepInfo);
        
        // Determine execution strategy based on step type
        const strategy = this.determineStrategy(stepInfo);
        
        // Extract parameters from step config
        const parameters = this.extractParameters(stepInfo);
        
        return {
            context: {
                swarmId: context.swarmId,
                userData: context.userData,
                parentSwarmId: context.parentSwarmId,
                timestamp: context.timestamp || new Date(),
            },
            input: {
                stepId: stepInfo.id,
                stepType: stepInfo.type,
                toolName,
                parameters,
                strategy,
            },
            allocation: allocation || {
                allocated: {
                    credits: "50",
                    timeoutMs: 60000,
                    memoryMB: 128,
                    concurrentExecutions: 1,
                },
                remaining: {
                    credits: "50",
                    timeoutMs: 60000,
                    memoryMB: 128,
                    concurrentExecutions: 1,
                },
                allocatedAt: new Date(),
                expiresAt: new Date(Date.now() + 60000),
            },
            options: {
                timeout: 30000,
                retryCount: 0,
                priority: "medium",
            },
        };
    }

    /**
     * Extract tool name from step information
     */
    private extractToolName(stepInfo: StepInfo): string | undefined {
        // Check if tool name is explicitly defined in config
        if (stepInfo.config?.toolName && typeof stepInfo.config.toolName === "string") {
            return stepInfo.config.toolName;
        }
        
        // Check if step type indicates a tool
        if (stepInfo.type === "tool_call" && stepInfo.config?.tool && typeof stepInfo.config.tool === "string") {
            return stepInfo.config.tool;
        }
        
        // For certain step types, derive tool name
        if (stepInfo.type === "api_call") {
            return "http_request";
        }
        
        return undefined;
    }

    /**
     * Determine execution strategy based on step type and configuration
     */
    private determineStrategy(stepInfo: StepInfo): ExecutionStrategy {
        // Check if strategy is explicitly defined
        if (stepInfo.config?.strategy && typeof stepInfo.config.strategy === "string") {
            return stepInfo.config.strategy as ExecutionStrategy;
        }
        
        // Determine strategy based on step type
        switch (stepInfo.type) {
            case "llm_call":
                return "reasoning";
            case "tool_call":
            case "api_call":
                return "deterministic";
            case "code_execution":
                return "deterministic";
            default:
                return "reasoning";
        }
    }

    /**
     * Extract parameters from step configuration
     */
    private extractParameters(stepInfo: StepInfo): Record<string, unknown> {
        const params: Record<string, unknown> = {};
        
        // Include all config as parameters, excluding special fields
        if (stepInfo.config) {
            const { toolName, strategy, ...configParams } = stepInfo.config;
            Object.assign(params, configParams);
        }
        
        // Add step metadata
        params.stepName = stepInfo.name;
        params.stepDescription = stepInfo.description;
        
        // Map common fields to expected locations for stepExecutor
        if (stepInfo.config?.messages) {
            params.messages = stepInfo.config.messages;
        }
        if (stepInfo.config?.prompt) {
            params.prompt = stepInfo.config.prompt;
        }
        if (stepInfo.config?.code) {
            params.code = stepInfo.config.code;
        }
        if (stepInfo.config?.routineConfig) {
            params.routineConfig = stepInfo.config.routineConfig;
        }
        
        return params;
    }

    /**
     * Get current execution state
     */
    getState(): RunState {
        return this.stateMachine.getState();
    }

    /**
     * Check if execution can proceed
     */
    async canProceed(): Promise<boolean> {
        return await this.stateMachine.canProceed();
    }

    /**
     * Pause execution
     */
    async pause(): Promise<void> {
        await this.stateMachine.pause();
        logger.info("[RoutineExecutor] Execution paused");
    }

    /**
     * Resume execution
     */
    async resume(): Promise<void> {
        await this.stateMachine.resume();
        logger.info("[RoutineExecutor] Execution resumed");
    }

    /**
     * Stop execution
     */
    async stop(reason?: string): Promise<void> {
        await this.stateMachine.stop(reason);
        logger.info("[RoutineExecutor] Execution stopped", { reason });
    }

    /**
     * Simple permission check for step execution
     * Replaces the over-engineered MOISEGate with a straightforward role-based check
     */
    private async checkStepPermission(runId: string, stepId: string, context: RunContext): Promise<boolean> {
        const userRole = context.variables.userRole as string || "user";
        const stepType = context.variables[`step_${stepId}_type`] as string || "action";

        // Simple role-based permissions
        const allowedRoles: Record<string, string[]> = {
            "admin": ["*"], // Admin can execute any step type
            "agent": ["action", "decision", "subroutine", "parallel", "loop"],
            "user": ["action", "decision", "loop"],
            "guest": [], // Guests cannot execute any steps
        };

        const permissions = allowedRoles[userRole] || [];
        const hasPermission = permissions.includes("*") || permissions.includes(stepType);

        if (!hasPermission) {
            logger.warn("[RoutineExecutor] Permission denied", {
                runId,
                stepId,
                userRole,
                stepType,
                allowedStepTypes: permissions,
            });
        }

        return hasPermission;
    }
}
