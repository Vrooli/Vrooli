import { type INavigator } from "../types.js";
import { SequentialNavigator } from "./SequentialNavigator.js";
import { BpmnNavigator } from "./BpmnNavigator.js";

/**
 * Simple navigator factory function
 * 
 * Returns the appropriate navigator for a given routine.
 * Much simpler than the NavigatorRegistry class.
 */
export function getNavigator(routine: unknown): INavigator | null {
    const sequentialNav = new SequentialNavigator();
    if (sequentialNav.canNavigate(routine)) {
        return sequentialNav;
    }
    
    const bpmnNav = new BpmnNavigator();
    if (bpmnNav.canNavigate(routine)) {
        return bpmnNav;
    }
    
    return null;
}

/**
 * Get navigator by type string
 */
export function getNavigatorByType(type: string): INavigator | null {
    switch (type) {
        case "sequential":
            return new SequentialNavigator();
        case "bpmn":
            return new BpmnNavigator();
        default:
            return null;
    }
}

/**
 * Get supported navigator types
 */
export function getSupportedTypes(): string[] {
    return ["sequential", "bpmn"];
}
