import { type Logger } from "winston";
import {
    type Location,
    type StepInfo,
    type NavigationTrigger,
    type NavigationTimeout,
    type NavigationEvent,
} from "@vrooli/shared";
import { type RoutineVersionConfigObject, type GraphSequentialConfig, type SequentialStep } from "@vrooli/shared";
import { BaseNavigator } from "./baseNavigator.js";

/**
 * SequentialNavigator - Navigator for sequential (array-based) workflows
 * 
 * This navigator handles simple sequential workflows where steps are executed
 * one after another in order. It provides a simpler alternative to BPMN for
 * straightforward linear processes.
 * 
 * Key features:
 * - Sequential step execution (by default)
 * - Parallel execution mode for independent steps
 * - Skip conditions for conditional step execution
 * - Retry policies for resilient execution
 * - Simple step indexing (nodeId = step index)
 */
export class SequentialNavigator extends BaseNavigator {
    readonly type = "sequential";
    readonly version = "1.0.0";

    constructor(logger: Logger) {
        super(logger);
    }

    /**
     * Checks if this navigator can handle the given routine config
     */
    canNavigate(routine: unknown): boolean {
        try {
            const config = routine as RoutineVersionConfigObject;
            
            // Must have a version
            if (!config.__version) {
                return false;
            }

            // Must have a graph config with Sequential type
            if (!config.graph || config.graph.__type !== "Sequential") {
                return false;
            }

            // Must have sequential schema with steps
            const seqConfig = config.graph as GraphSequentialConfig;
            if (!seqConfig.schema || !Array.isArray(seqConfig.schema.steps)) {
                return false;
            }

            // Must have at least one step
            if (seqConfig.schema.steps.length === 0) {
                return false;
            }

            return true;

        } catch (error) {
            this.logger.debug("[SequentialNavigator] Cannot navigate routine", {
                error: error instanceof Error ? error.message : String(error),
            });
            return false;
        }
    }

    /**
     * Gets the starting location in the sequential workflow
     */
    getStartLocation(routine: unknown): Location {
        const routineConfig = this.validateAndCache(routine);
        
        // For sequential routines, we always start at step 0
        return this.createLocation("0", routineConfig);
    }

    /**
     * Gets all possible starting locations in the sequential workflow
     */
    getAllStartLocations(routine: unknown): Location[] {
        // Sequential routines have only one start location
        return [this.getStartLocation(routine)];
    }

    /**
     * Gets the next possible locations from current location
     */
    async getNextLocations(current: Location, context: Record<string, unknown>): Promise<Location[]> {
        const routineConfig = await this.getCachedConfig(current.routineId);
        const seqConfig = this.getSequentialConfig(routineConfig);
        
        // Parse current step index
        const currentIndex = parseInt(current.nodeId);
        if (isNaN(currentIndex)) {
            this.logger.error("[SequentialNavigator] Invalid nodeId", { nodeId: current.nodeId });
            return [];
        }

        // Check if we're in parallel mode
        if (seqConfig.schema.executionMode === "parallel") {
            // This is a design decision: in parallel mode, we use the RunStateMachine's
            // branch coordination to handle parallel execution. The navigator just needs
            // to report that these can be parallel branches.
            // For now, we'll keep it simple and let getParallelBranches handle this.
            return [];
        }

        // Sequential mode: move to next step
        const nextIndex = currentIndex + 1;
        
        // Check if we've reached the end
        if (nextIndex >= seqConfig.schema.steps.length) {
            return [];
        }

        // Find the next non-skipped step
        let currentIdx = nextIndex;
        while (currentIdx < seqConfig.schema.steps.length) {
            const step = seqConfig.schema.steps[currentIdx];
            if (!await this.shouldSkipStep(step, context)) {
                return [this.createLocation(currentIdx.toString(), routineConfig)];
            }
            currentIdx++;
        }
        
        // All remaining steps are skipped, we've reached the end
        return [];
    }

    /**
     * Checks if location is an end location
     */
    async isEndLocation(location: Location): Promise<boolean> {
        const routineConfig = await this.getCachedConfig(location.routineId);
        const seqConfig = this.getSequentialConfig(routineConfig);
        
        const stepIndex = parseInt(location.nodeId);
        if (isNaN(stepIndex)) {
            return false;
        }
        
        // In sequential mode, last step is end location
        if (seqConfig.schema.executionMode !== "parallel") {
            return stepIndex === seqConfig.schema.steps.length - 1;
        }
        
        // In parallel mode, there's no single end location
        // We consider it "end" when all steps have been executed
        // This is handled by the execution engine
        return false;
    }

    /**
     * Gets information about a step at a location
     */
    async getStepInfo(location: Location): Promise<StepInfo> {
        const routineConfig = await this.getCachedConfig(location.routineId);
        const seqConfig = this.getSequentialConfig(routineConfig);
        
        const stepIndex = parseInt(location.nodeId);
        if (isNaN(stepIndex) || stepIndex < 0 || stepIndex >= seqConfig.schema.steps.length) {
            throw new Error(`Invalid step index: ${location.nodeId}`);
        }
        
        const step = seqConfig.schema.steps[stepIndex];
        
        if (!step) {
            throw new Error(`Step not found at index ${stepIndex}`);
        }

        return {
            id: step.id,
            name: step.name,
            type: "subroutine",
            description: step.description,
            config: {
                subroutineId: step.subroutineId,
                inputMap: step.inputMap,
                outputMap: step.outputMap,
                retryPolicy: step.retryPolicy,
            },
        };
    }

    /**
     * Gets dependencies for a step
     */
    async getDependencies(location: Location): Promise<string[]> {
        const routineConfig = await this.getCachedConfig(location.routineId);
        const seqConfig = this.getSequentialConfig(routineConfig);
        
        const stepIndex = parseInt(location.nodeId);
        if (isNaN(stepIndex)) {
            return [];
        }
        
        // In sequential mode, each step depends on the previous one
        if (seqConfig.schema.executionMode !== "parallel") {
            if (stepIndex > 0) {
                return [(stepIndex - 1).toString()];
            }
        }
        
        // First step or parallel mode has no dependencies
        return [];
    }

    /**
     * Gets parallel branches from a location
     */
    async getParallelBranches(location: Location): Promise<Location[][]> {
        const routineConfig = await this.getCachedConfig(location.routineId);
        const seqConfig = this.getSequentialConfig(routineConfig);
        
        // Only return parallel branches if we're in parallel mode
        if (seqConfig.schema.executionMode === "parallel") {
            // Check if this is called from the start of the routine
            // In our implementation, the RunStateMachine will call this
            // when it needs to determine parallel execution paths
            const stepIndex = parseInt(location.nodeId);
            
            // If we're at the beginning, return all steps as separate branches
            if (stepIndex === 0 || isNaN(stepIndex)) {
                const branches: Location[][] = [];
                const context = {}; // Context not needed for parallel branch detection in sequential mode
                
                for (let i = 0; i < seqConfig.schema.steps.length; i++) {
                    const step = seqConfig.schema.steps[i];
                    // Skip steps that should be skipped
                    if (!await this.shouldSkipStep(step, context)) {
                        branches.push([this.createLocation(i.toString(), routineConfig)]);
                    }
                }
                return branches;
            }
        }
        
        // No parallel branches in sequential mode or if not at start
        return [];
    }

    /**
     * Gets triggers for a location in the sequential workflow.
     * 
     * Sequential navigation is inherently deterministic and step-by-step, where each step
     * executes only after the previous step completes successfully. This design eliminates
     * the need for event-driven triggers that are common in other workflow types.
     * 
     * **Why Sequential Navigation Doesn't Need Triggers:**
     * - **Deterministic Flow**: Steps execute in strict order (0 → 1 → 2 → ...)
     * - **Synchronous Progression**: Each step waits for the previous step to complete
     * - **Simple Conditional Logic**: Skip conditions handle conditional execution
     * - **Clear Dependencies**: Step N always depends on step N-1 in sequential mode
     * 
     * **Contrast with Other Navigator Types:**
     * - **BPMN Navigator**: Uses triggers for events, timers, and message flows
     * - **State Machine Navigator**: Uses triggers for state transitions and external events
     * - **Graph Navigator**: May use triggers for complex branching conditions
     * 
     * **Emergent Architecture Alignment:**
     * Sequential navigation provides the foundation for workflow evolution. As teams use
     * sequential routines, agents can analyze patterns and suggest evolution to more
     * sophisticated workflow types (BPMN, state machines) when event-driven behavior
     * would provide value.
     * 
     * @param location - The current location in the sequential workflow
     * @returns Empty array - sequential navigation does not use location triggers
     * 
     * @example
     * ```typescript
     * const triggers = await navigator.getLocationTriggers(location);
     * console.log(triggers); // Always []
     * ```
     */
    async getLocationTriggers(location: Location): Promise<NavigationTrigger[]> {
        // Sequential navigators don't support event-driven triggers
        // This is by design - sequential workflows execute steps in order
        return [];
    }

    /**
     * Gets timeouts for a location in the sequential workflow.
     * 
     * Sequential navigation operates on a step-completion model where each step runs
     * to completion before proceeding to the next step. Navigation-level timeouts
     * are not applicable because the navigator doesn't control step execution timing.
     * 
     * **Why Sequential Navigation Doesn't Define Timeouts:**
     * - **Execution-Level Responsibility**: Individual step timeouts are handled by the execution engine
     * - **Step Autonomy**: Each sequential step (subroutine) manages its own execution time
     * - **No Navigation Delays**: There are no delays or waiting periods in the navigation logic
     * - **Simple Flow Control**: Navigation decisions are immediate based on completion status
     * 
     * **Where Timeouts Are Actually Handled:**
     * - **Tier 3 Execution**: Tool timeouts, LLM response timeouts, subroutine execution timeouts
     * - **Individual Steps**: Each step's retryPolicy can define timeout behavior
     * - **Resource Managers**: Connection timeouts, API call timeouts, database query timeouts
     * - **Circuit Breakers**: Resilience patterns at the service integration level
     * 
     * **Contrast with Other Navigator Types:**
     * - **BPMN Navigator**: May define timer events that trigger navigation changes
     * - **State Machine Navigator**: May have timeout-based state transitions
     * - **Event-Driven Navigators**: May timeout waiting for external events
     * 
     * **Emergent Architecture Benefits:**
     * By keeping navigation timeout-free, sequential workflows remain simple and predictable.
     * This enables agents to focus on optimizing individual step execution rather than
     * complex timing coordination, leading to clearer performance analysis and optimization.
     * 
     * @param location - The current location in the sequential workflow
     * @returns Empty array - timeouts are handled at execution level, not navigation level
     * 
     * @example
     * ```typescript
     * const timeouts = await navigator.getLocationTimeouts(location);
     * console.log(timeouts); // Always []
     * 
     * // Timeouts are configured at the step level instead:
     * const stepInfo = await navigator.getStepInfo(location);
     * console.log(stepInfo.config.retryPolicy?.timeout); // Step-specific timeout
     * ```
     */
    async getLocationTimeouts(location: Location): Promise<NavigationTimeout[]> {
        // Sequential navigators don't have built-in timeouts
        // Timeouts are handled at the execution level, not navigation level
        // This separation of concerns keeps navigation logic simple and predictable
        return [];
    }

    /**
     * Checks if an event can trigger navigation changes at a location.
     * 
     * Sequential navigation follows a strict step-by-step execution model where
     * navigation decisions are based solely on step completion status, not external events.
     * This design ensures predictable, deterministic workflow execution.
     * 
     * **Why Sequential Navigation Doesn't Support Event Triggers:**
     * - **Deterministic Execution**: Navigation follows a predictable path (step 0 → 1 → 2 → ...)
     * - **Completion-Based Progression**: Next step is determined by current step completion
     * - **No External Dependencies**: Navigation doesn't wait for or react to external events
     * - **Simplified Debugging**: Execution flow is always traceable and predictable
     * 
     * **Event Handling at Other Levels:**
     * - **Tier 1 Coordination**: Swarms can react to events and coordinate responses
     * - **Tier 3 Execution**: Individual steps can respond to tool events and API responses
     * - **Cross-Tier Events**: Event bus enables communication between tiers
     * - **Agent Events**: Specialized agents can subscribe to and process domain events
     * 
     * **When Event-Driven Navigation Is Needed:**
     * If your workflow requires event-driven behavior, consider using:
     * - **BPMN Navigator**: For complex workflows with message events, timer events, and signals
     * - **State Machine Navigator**: For state-based workflows that react to external events
     * - **Custom Navigator**: For domain-specific event-driven navigation patterns
     * 
     * **Emergent Evolution Path:**
     * Sequential workflows that repeatedly need event-driven behavior can evolve:
     * 1. **Start**: Simple sequential routine (A → B → C)
     * 2. **Detect**: Agents observe frequent manual interventions or external dependencies
     * 3. **Suggest**: Optimization agents recommend BPMN or state machine patterns
     * 4. **Evolve**: Routine transforms to event-driven pattern when benefits are clear
     * 
     * @param location - The current location in the sequential workflow
     * @param event - The navigation event to check (ignored in sequential navigation)
     * @returns Always false - sequential navigation does not support event-driven changes
     * 
     * @example
     * ```typescript
     * const canTrigger = await navigator.canTriggerEvent(location, {
     *     type: 'timer',
     *     payload: { duration: 5000 }
     * });
     * console.log(canTrigger); // Always false
     * 
     * // For event-driven workflows, use BPMN or state machine navigators instead
     * ```
     */
    async canTriggerEvent(location: Location, event: NavigationEvent): Promise<boolean> {
        // Sequential navigators don't support event-driven navigation
        // This is intentional - sequential workflows execute in strict order
        // Events are handled at the execution level (Tier 3) or coordination level (Tier 1)
        return false;
    }

    /**
     * Private helper methods
     */
    private getSequentialConfig(config: RoutineVersionConfigObject): GraphSequentialConfig {
        if (!config.graph || config.graph.__type !== "Sequential") {
            throw new Error("Invalid sequential configuration");
        }
        return config.graph as GraphSequentialConfig;
    }

    /**
     * Evaluates if a step should be skipped based on its skip condition
     */
    private async shouldSkipStep(step: SequentialStep, context: Record<string, unknown>): Promise<boolean> {
        if (!step.skipCondition) {
            return false;
        }

        try {
            // Simple expression evaluation with safety constraints
            // In a production implementation, this would use a proper sandboxed expression evaluator
            const condition = step.skipCondition.trim();
            
            // Safety check: only allow specific patterns
            const contextVarPattern = /^context\.([a-zA-Z_$][a-zA-Z0-9_$]*)$/;
            const negatedContextVarPattern = /^!context\.([a-zA-Z_$][a-zA-Z0-9_$]*)$/;
            
            // Check if it's a simple context variable reference
            const contextMatch = condition.match(contextVarPattern);
            if (contextMatch && contextMatch[1]) {
                const varName = contextMatch[1];
                const value = context[varName];
                // Explicitly check for boolean true to skip
                return value === true;
            }
            
            // Check for negation
            const negatedMatch = condition.match(negatedContextVarPattern);
            if (negatedMatch && negatedMatch[1]) {
                const varName = negatedMatch[1];
                const value = context[varName];
                // Skip if value is not true (including undefined, null, false, etc.)
                return value !== true;
            }
            
            // If the pattern doesn't match our safe patterns, log warning and don't skip
            this.logger.warn("[SequentialNavigator] Skip condition doesn't match safe patterns", {
                stepId: step.id,
                condition: step.skipCondition,
            });
            return false;
            
        } catch (error) {
            this.logger.warn("[SequentialNavigator] Error evaluating skip condition", {
                stepId: step.id,
                condition: step.skipCondition,
                error: error instanceof Error ? error.message : String(error),
            });
            return false;
        }
    }
}
