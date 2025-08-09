import { type RoutineVersionConfigObject, type GraphSequentialConfig } from "@vrooli/shared";
import { BaseNavigator } from "./BaseNavigator.js";
import { type Location, type StepInfo } from "../types.js";

/**
 * SequentialNavigator - Pure sequential navigation
 * 
 * Handles simple step-by-step navigation (0 → 1 → 2 → ...).
 * No event handling, caching, or complex state management.
 */
export class SequentialNavigator extends BaseNavigator {
    readonly type = "sequential";
    readonly version = "1.0.0";
    
    canNavigate(routine: unknown): boolean {
        try {
            const config = routine as RoutineVersionConfigObject;
            return !!(
                config.__version &&
                config.graph?.__type === "Sequential" &&
                Array.isArray(config.graph.schema?.steps) &&
                config.graph.schema.steps.length > 0
            );
        } catch {
            return false;
        }
    }
    
    getStartLocation(routine: unknown): Location {
        const config = this.validateRoutine(routine);
        const routineId = this.generateRoutineId(config);
        return this.createLocation("0", routineId);
    }
    
    getNextLocations(routine: unknown, current: Location, context?: Record<string, unknown>): Location[] {
        const config = this.validateRoutine(routine);
        const steps = this.getSteps(config);
        const currentIndex = parseInt(current.nodeId);
        
        if (isNaN(currentIndex) || currentIndex < 0) {
            return [];
        }
        
        const nextIndex = currentIndex + 1;
        if (nextIndex >= steps.length) {
            return []; // End of sequence
        }
        
        // Simple skip condition check (optional)
        const nextStep = steps[nextIndex];
        if (context && this.shouldSkip(nextStep, context)) {
            // Recursively check next step
            const nextLocation = this.createLocation(nextIndex.toString(), current.routineId);
            return this.getNextLocations(routine, nextLocation, context);
        }
        
        return [this.createLocation(nextIndex.toString(), current.routineId)];
    }
    
    isEndLocation(routine: unknown, location: Location): boolean {
        const config = this.validateRoutine(routine);
        const steps = this.getSteps(config);
        const currentIndex = parseInt(location.nodeId);
        
        return currentIndex === steps.length - 1;
    }
    
    getStepInfo(routine: unknown, location: Location): StepInfo {
        const config = this.validateRoutine(routine);
        const steps = this.getSteps(config);
        const stepIndex = parseInt(location.nodeId);
        
        if (stepIndex < 0 || stepIndex >= steps.length) {
            throw new Error(`Invalid step index: ${location.nodeId}`);
        }
        
        const step = steps[stepIndex];
        return {
            id: step.id,
            name: step.name,
            type: "subroutine",
            description: step.description,
            config: {
                subroutineId: step.subroutineId,
                inputMap: step.inputMap,
                outputMap: step.outputMap,
            },
        };
    }
    
    // Private helpers
    private getSteps(config: RoutineVersionConfigObject) {
        const graph = config.graph as GraphSequentialConfig;
        return graph.schema.steps;
    }
    
    private shouldSkip(step: any, context: Record<string, unknown>): boolean {
        // Simple skip condition - only basic variable checks
        if (!step.skipCondition) return false;
        
        const condition = step.skipCondition.trim();
        const match = condition.match(/^context\.([a-zA-Z_$][a-zA-Z0-9_$]*)$/);
        if (match && match[1]) {
            return context[match[1]] === true;
        }
        
        return false;
    }
    
    private generateRoutineId(config: RoutineVersionConfigObject): string {
        // Simple deterministic ID
        return `seq_${config.__version}_${this.hashSteps(config)}`;
    }
    
    private hashSteps(config: RoutineVersionConfigObject): string {
        const steps = this.getSteps(config);
        const stepIds = steps.map(s => s.id).join("_");
        return Math.abs(this.simpleHash(stepIds)).toString(16);
    }
    
    private simpleHash(str: string): number {
        let hash = 0;
        for (let i = 0; i < str.length; i++) {
            const char = str.charCodeAt(i);
            hash = ((hash << 5) - hash) + char;
            hash = hash & hash; // Convert to 32-bit integer
        }
        return hash;
    }
}
