import { type RoutineVersionConfigObject } from "@vrooli/shared";
import { type INavigator, type Location, type StepInfo } from "../types.js";

/**
 * BaseNavigator - Minimal shared functionality
 * 
 * Provides only common utilities needed by all navigators.
 * NO state management, caching, or complex logic.
 */
export abstract class BaseNavigator implements INavigator {
    abstract readonly type: string;
    abstract readonly version: string;
    
    /**
     * Creates a location identifier
     */
    protected createLocation(nodeId: string, routineId: string): Location {
        return {
            id: `${routineId}_${nodeId}`,
            routineId,
            nodeId,
        };
    }
    
    /**
     * Validates routine has required structure
     */
    protected validateRoutine(routine: unknown): RoutineVersionConfigObject {
        if (!routine || typeof routine !== "object") {
            throw new Error(`Invalid routine configuration for ${this.type} navigator`);
        }
        
        const config = routine as RoutineVersionConfigObject;
        if (!config.__version) {
            throw new Error(`Routine missing version for ${this.type} navigator`);
        }
        
        return config;
    }
    
    // Abstract methods - minimal navigation interface
    abstract canNavigate(routine: unknown): boolean;
    abstract getStartLocation(routine: unknown): Location;
    abstract getNextLocations(routine: unknown, current: Location, context?: Record<string, unknown>): Location[];
    abstract isEndLocation(routine: unknown, location: Location): boolean;
    abstract getStepInfo(routine: unknown, location: Location): StepInfo;
}
