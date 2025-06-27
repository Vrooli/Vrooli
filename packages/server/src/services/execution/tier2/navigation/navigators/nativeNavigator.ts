import { type Logger } from "winston";
import {
    type Navigator,
    type Location,
    type StepInfo,
    type NavigationTrigger,
    type NavigationTimeout,
    type NavigationEvent,
} from "@vrooli/shared";
import { type RoutineVersionConfigObject } from "@vrooli/shared";
import { type BaseNavigator } from "./baseNavigator.js";
import { BpmnNavigator } from "./bpmnNavigator.js";
import { SingleStepNavigator } from "./singleStepNavigator.js";
import { SequentialNavigator } from "./sequentialNavigator.js";

/**
 * NativeNavigator - Factory/Adapter Navigator for Unified Routine Execution
 * 
 * ## IMPORTANT: This is NOT a separate navigation format!
 * 
 * The NativeNavigator is a **factory pattern implementation** that serves as the single entry point
 * for all routine navigation in Vrooli's execution architecture. It automatically detects the routine
 * type and delegates to the appropriate specialized navigator.
 * 
 * ## Architecture Role:
 * ```
 * Tier 2 Execution Engine
 *      ↓
 * NativeNavigator (Factory/Adapter) ← Registered as "native" in NavigatorRegistry
 *      ├── BpmnNavigator (BPMN-2.0 workflows)
 *      ├── SequentialNavigator (Vrooli's native sequential format)  
 *      └── SingleStepNavigator (direct API/AI calls)
 * ```
 * 
 * ## Why This Design?
 * 1. **Unified Interface**: Execution layers only interact with one navigator type
 * 2. **Emergent Capabilities**: All navigation types are data-driven, not code-driven
 * 3. **Future-Proof**: New formats (Langchain, Temporal) can be added without changing execution architecture
 * 4. **Separation of Concerns**: Each navigator specializes in one format
 * 
 * ## Supported Routine Types:
 * - **BPMN-2.0**: Complex business process workflows (`graph.__type === "BPMN-2.0"`)
 * - **Sequential**: Vrooli's native linear workflow format (`graph.__type === "Sequential"`)
 * - **Single-Step**: Direct action routines (no graph, just callData* configs)
 * 
 * ## Navigation Flow:
 * 1. Execution engine calls NativeNavigator methods
 * 2. NativeNavigator inspects routine config to determine type  
 * 3. Delegates to appropriate specialized navigator
 * 4. Returns unified result to execution engine
 * 
 * This design maintains Vrooli's core principle that all capabilities emerge from data configuration
 * rather than requiring code changes for new workflow types.
 */
export class NativeNavigator implements Navigator {
    readonly type = "native";
    readonly version = "1.0.0";
    
    private readonly logger: Logger;
    private readonly bpmnNavigator: BpmnNavigator;
    private readonly singleStepNavigator: SingleStepNavigator;
    private readonly sequentialNavigator: SequentialNavigator;

    constructor(logger: Logger) {
        this.logger = logger;
        this.bpmnNavigator = new BpmnNavigator(logger);
        this.singleStepNavigator = new SingleStepNavigator(logger);
        this.sequentialNavigator = new SequentialNavigator(logger);
    }

    /**
     * Checks if this navigator can handle the given routine
     * 
     * Since this is a factory navigator, it can handle any routine that
     * one of its specialized navigators can handle.
     */
    canNavigate(routine: unknown): boolean {
        try {
            const config = routine as RoutineVersionConfigObject;
            
            // Must have a version
            if (!config.__version) {
                return false;
            }

            // Try each sub-navigator
            if (this.bpmnNavigator.canNavigate(routine)) {
                return true;
            }

            if (this.sequentialNavigator.canNavigate(routine)) {
                return true;
            }

            if (this.singleStepNavigator.canNavigate(routine)) {
                return true;
            }

            return false;

        } catch (error) {
            this.logger.debug("[NativeNavigator] Cannot navigate routine", {
                error: error instanceof Error ? error.message : String(error),
            });
            return false;
        }
    }

    /**
     * Gets the starting location in the routine
     * 
     * Delegates to the appropriate specialized navigator based on routine type.
     */
    getStartLocation(routine: unknown): Location {
        const navigator = this.getAppropriateNavigator(routine);
        return navigator.getStartLocation(routine);
    }

    /**
     * Gets all possible starting locations in the routine
     */
    getAllStartLocations(routine: unknown): Location[] {
        const navigator = this.getAppropriateNavigator(routine);
        return navigator.getAllStartLocations(routine);
    }

    /**
     * Gets the next possible locations from current location
     */
    async getNextLocations(current: Location, context: Record<string, unknown>): Promise<Location[]> {
        const navigator = await this.getNavigatorForLocation(current);
        return navigator.getNextLocations(current, context);
    }

    /**
     * Checks if location is an end location
     */
    async isEndLocation(location: Location): Promise<boolean> {
        const navigator = await this.getNavigatorForLocation(location);
        return navigator.isEndLocation(location);
    }

    /**
     * Gets information about a step at a location
     */
    async getStepInfo(location: Location): Promise<StepInfo> {
        const navigator = await this.getNavigatorForLocation(location);
        return navigator.getStepInfo(location);
    }

    /**
     * Gets dependencies for a step
     */
    async getDependencies(location: Location): Promise<string[]> {
        const navigator = await this.getNavigatorForLocation(location);
        return navigator.getDependencies(location);
    }

    /**
     * Gets parallel branches from a location
     */
    async getParallelBranches(location: Location): Promise<Location[][]> {
        const navigator = await this.getNavigatorForLocation(location);
        return navigator.getParallelBranches(location);
    }

    /**
     * Gets triggers for a location
     */
    async getLocationTriggers(location: Location): Promise<NavigationTrigger[]> {
        const navigator = await this.getNavigatorForLocation(location);
        return navigator.getLocationTriggers(location);
    }

    /**
     * Gets timeouts for a location
     */
    async getLocationTimeouts(location: Location): Promise<NavigationTimeout[]> {
        const navigator = await this.getNavigatorForLocation(location);
        return navigator.getLocationTimeouts(location);
    }

    /**
     * Checks if an event can trigger at a location
     */
    async canTriggerEvent(location: Location, event: NavigationEvent): Promise<boolean> {
        const navigator = await this.getNavigatorForLocation(location);
        return navigator.canTriggerEvent(location, event);
    }

    /**
     * Determines which specialized navigator to use for a given routine
     * 
     * This method implements the core factory logic by testing each navigator's
     * canNavigate() method until it finds one that can handle the routine.
     * 
     * @param routine - The routine configuration object
     * @returns The appropriate specialized navigator
     * @throws Error if no navigator can handle the routine
     */
    private getAppropriateNavigator(routine: unknown): BaseNavigator {
        // Try BPMN first (most complex workflows)
        if (this.bpmnNavigator.canNavigate(routine)) {
            this.logger.debug("[NativeNavigator] Routing to BpmnNavigator");
            return this.bpmnNavigator;
        }

        // Try sequential (Vrooli's native linear workflows)
        if (this.sequentialNavigator.canNavigate(routine)) {
            this.logger.debug("[NativeNavigator] Routing to SequentialNavigator");
            return this.sequentialNavigator;
        }

        // Try single-step (direct actions without graphs)
        if (this.singleStepNavigator.canNavigate(routine)) {
            this.logger.debug("[NativeNavigator] Routing to SingleStepNavigator");
            return this.singleStepNavigator;
        }

        throw new Error("No appropriate navigator found for routine configuration");
    }

    /**
     * Determines which specialized navigator to use for a specific execution location
     * 
     * This method is used during execution when we have a Location object rather than
     * the full routine configuration. It uses multiple strategies to determine the 
     * correct navigator:
     * 1. Fast path: Check routineId prefix (set during navigation)
     * 2. Cache lookup: Try to find cached config in each navigator
     * 
     * @param location - The execution location to find a navigator for
     * @returns The appropriate specialized navigator
     * @throws Error if no navigator can handle the location
     */
    private async getNavigatorForLocation(location: Location): Promise<BaseNavigator> {
        // Fast path: Extract the navigator type from the routineId prefix
        if (location.routineId.startsWith("bpmn_config_")) {
            return this.bpmnNavigator;
        }
        
        if (location.routineId.startsWith("sequential_config_")) {
            return this.sequentialNavigator;
        }
        
        if (location.routineId.startsWith("single-step_config_")) {
            return this.singleStepNavigator;
        }

        // Fallback: Try to get the config from each navigator's cache
        try {
            // Try BPMN navigator first
            const bpmnConfig = await this.bpmnNavigator.getCachedConfig(location.routineId);
            if (bpmnConfig) {
                return this.bpmnNavigator;
            }
        } catch {
            // Ignore error, try next navigator
        }

        try {
            // Try sequential navigator
            const sequentialConfig = await this.sequentialNavigator.getCachedConfig(location.routineId);
            if (sequentialConfig) {
                return this.sequentialNavigator;
            }
        } catch {
            // Ignore error, try next navigator
        }

        try {
            // Try single-step navigator
            const singleStepConfig = await this.singleStepNavigator.getCachedConfig(location.routineId);
            if (singleStepConfig) {
                return this.singleStepNavigator;
            }
        } catch {
            // Ignore error
        }

        throw new Error(`Could not determine navigator for location: ${location.routineId}`);
    }
}
