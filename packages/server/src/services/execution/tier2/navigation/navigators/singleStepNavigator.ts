import { type Logger } from "winston";
import {
    type Location,
    type StepInfo,
    type NavigationTrigger,
    type NavigationTimeout,
    type NavigationEvent,
} from "@vrooli/shared";
import { type RoutineVersionConfigObject } from "@vrooli/shared";
import { BaseNavigator } from "./baseNavigator.js";

/**
 * SingleStepNavigator - Navigator for single-step routines
 * 
 * This navigator handles routines that don't have a graph configuration
 * and execute a single action (API call, code execution, AI generation, etc.).
 * These are the simplest type of routines with linear execution.
 * 
 * Key features:
 * - Handles all single-step routine types
 * - No graph navigation needed
 * - Direct execution flow (start -> single step -> end)
 * - Config-based step type detection
 */
export class SingleStepNavigator extends BaseNavigator {
    readonly type = "single-step";
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

            // Should NOT have a graph config (that's for multi-step routines)
            if (config.graph) {
                return false;
            }

            // Must have at least one call data configuration
            const hasCallData = !!(
                config.callDataAction ||
                config.callDataApi ||
                config.callDataCode ||
                config.callDataGenerate ||
                config.callDataSmartContract ||
                config.callDataWeb
            );

            if (!hasCallData) {
                return false;
            }

            return true;

        } catch (error) {
            this.logger.debug("[SingleStepNavigator] Cannot navigate routine", {
                error: error instanceof Error ? error.message : String(error),
            });
            return false;
        }
    }

    /**
     * Gets the starting location - always the single step
     */
    getStartLocation(routine: unknown): Location {
        const routineConfig = this.validateAndCache(routine);
        return this.createLocation("single_step", routineConfig);
    }

    /**
     * Gets all possible starting locations - single-step routines have only one
     */
    getAllStartLocations(routine: unknown): Location[] {
        // For single-step routines, there's only one start location
        return [this.getStartLocation(routine)];
    }

    /**
     * Gets the next possible locations - none for single-step routines
     */
    async getNextLocations(_current: Location, _context: Record<string, unknown>): Promise<Location[]> {
        // Single-step routines have no next locations
        return [];
    }

    /**
     * Checks if location is an end location - single step is always an end
     */
    async isEndLocation(location: Location): Promise<boolean> {
        // For single-step routines, the single step is always the end
        return location.nodeId === "single_step";
    }

    /**
     * Gets information about the single step
     */
    async getStepInfo(location: Location): Promise<StepInfo> {
        const routineConfig = await this.getCachedConfig(location.routineId);
        
        if (location.nodeId !== "single_step") {
            throw new Error(`Invalid node ID for single-step routine: ${location.nodeId}`);
        }

        return this.createSingleStepInfo(routineConfig);
    }

    /**
     * Gets dependencies - none for single-step routines
     */
    async getDependencies(_location: Location): Promise<string[]> {
        // Single-step routines have no internal dependencies
        return [];
    }

    /**
     * Gets parallel branches - none for single-step routines
     */
    async getParallelBranches(_location: Location): Promise<Location[][]> {
        // Single-step routines have no parallel branches
        return [];
    }

    /**
     * Gets triggers for a location - none for single-step routines
     */
    async getLocationTriggers(_location: Location): Promise<NavigationTrigger[]> {
        // Single-step routines typically don't have triggers
        // The step executes immediately when the routine starts
        return [];
    }

    /**
     * Gets timeouts for a location - uses routine-level timeout if configured
     */
    async getLocationTimeouts(location: Location): Promise<NavigationTimeout[]> {
        try {
            const routineConfig = await this.getCachedConfig(location.routineId);
            
            // Check if the routine has timeout configuration
            const timeouts: NavigationTimeout[] = [];
            
            // Look for timeout in various call data configurations
            const callDataConfigs = [
                routineConfig.callDataAction,
                routineConfig.callDataApi,
                routineConfig.callDataCode,
                routineConfig.callDataGenerate,
                routineConfig.callDataSmartContract,
                routineConfig.callDataWeb,
            ].filter(Boolean);

            for (const callData of callDataConfigs) {
                if (callData && typeof callData === "object" && "timeout" in callData) {
                    const timeout = (callData as any).timeout;
                    if (typeof timeout === "number" && timeout > 0) {
                        timeouts.push({
                            id: `timeout_${location.nodeId}`,
                            duration: timeout,
                            onTimeout: "fail", // Default to fail on timeout
                        });
                    }
                }
            }

            return timeouts;
        } catch (error) {
            this.logger.debug("[SingleStepNavigator] Error getting location timeouts", {
                locationId: location.id,
                error: error instanceof Error ? error.message : String(error),
            });
            return [];
        }
    }

    /**
     * Checks if an event can trigger at a location - single-step routines don't support events
     */
    async canTriggerEvent(_location: Location, _event: NavigationEvent): Promise<boolean> {
        // Single-step routines don't have event-driven navigation
        // They execute immediately and complete
        return false;
    }
}
