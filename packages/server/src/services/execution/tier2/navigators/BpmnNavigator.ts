import { type RoutineVersionConfigObject, type GraphBpmnConfig } from "@vrooli/shared";
import { BaseNavigator } from "./BaseNavigator.js";
import { type Location, type StepInfo } from "../types.js";

/**
 * BpmnNavigator - Pure BPMN navigation
 * 
 * Handles BPMN workflow navigation by parsing XML structure.
 * No event handling, timeouts, or complex trigger logic.
 */
export class BpmnNavigator extends BaseNavigator {
    readonly type = "bpmn";
    readonly version = "1.0.0";
    
    canNavigate(routine: unknown): boolean {
        try {
            const config = routine as RoutineVersionConfigObject;
            return !!(
                config.__version &&
                config.graph?.__type === "BPMN-2.0" &&
                config.graph.schema?.data &&
                typeof config.graph.schema.data === "string" &&
                config.graph.schema.data.includes("xmlns:bpmn")
            );
        } catch {
            return false;
        }
    }
    
    getStartLocation(routine: unknown): Location {
        const config = this.validateRoutine(routine);
        const bpmnData = this.getBpmnData(config);
        const routineId = this.generateRoutineId(config);
        
        // Find start event
        const startEventMatch = bpmnData.match(/<bpmn:startEvent[^>]*id="([^"]*)"[^>]*>/);
        if (startEventMatch && startEventMatch[1]) {
            return this.createLocation(startEventMatch[1], routineId);
        }
        
        // Find first task without incoming flows
        const taskRegex = /<bpmn:(task|callActivity|userTask|serviceTask)[^>]*id="([^"]*)"[^>]*>/g;
        let taskMatch;
        while ((taskMatch = taskRegex.exec(bpmnData)) !== null) {
            const taskId = taskMatch[2];
            if (taskId && !this.hasIncomingFlow(bpmnData, taskId)) {
                return this.createLocation(taskId, routineId);
            }
        }
        
        throw new Error("No start location found in BPMN");
    }
    
    getNextLocations(routine: unknown, current: Location, context?: Record<string, unknown>): Location[] {
        const config = this.validateRoutine(routine);
        const bpmnData = this.getBpmnData(config);
        const nextLocations: Location[] = [];
        
        // Find outgoing sequence flows
        const flowRegex = new RegExp(
            `<bpmn:sequenceFlow[^>]*sourceRef="${current.nodeId}"[^>]*targetRef="([^"]*)"[^>]*(?:>.*?<bpmn:conditionExpression[^>]*>([^<]*)</bpmn:conditionExpression>.*?</bpmn:sequenceFlow>|/>)`,
            "gs",
        );
        
        let match;
        while ((match = flowRegex.exec(bpmnData)) !== null) {
            const targetRef = match[1];
            const condition = match[2];
            
            if (targetRef) {
                // Simple condition evaluation
                if (condition && context && !this.evaluateSimpleCondition(condition.trim(), context)) {
                    continue;
                }
                
                nextLocations.push(this.createLocation(targetRef, current.routineId));
            }
        }
        
        return nextLocations;
    }
    
    isEndLocation(routine: unknown, location: Location): boolean {
        const config = this.validateRoutine(routine);
        const bpmnData = this.getBpmnData(config);
        
        // Check if it's an end event
        const endEventMatch = bpmnData.match(new RegExp(`<bpmn:endEvent[^>]*id="${location.nodeId}"`));
        if (endEventMatch) {
            return true;
        }
        
        // Check if it has no outgoing flows
        const hasOutgoing = bpmnData.match(new RegExp(`<bpmn:sequenceFlow[^>]*sourceRef="${location.nodeId}"`));
        return !hasOutgoing;
    }
    
    getStepInfo(routine: unknown, location: Location): StepInfo {
        const config = this.validateRoutine(routine);
        const bpmnData = this.getBpmnData(config);
        
        // Find element by ID
        const elementMatch = bpmnData.match(new RegExp(`<bpmn:(\\w+)[^>]*id="${location.nodeId}"[^>]*(?:name="([^"]*)"[^>]*)?>`));
        
        if (elementMatch) {
            const elementType = elementMatch[1] || "task";
            const elementName = elementMatch[2] || location.nodeId;
            
            return {
                id: location.nodeId,
                name: elementName,
                type: this.mapBpmnType(elementType),
                description: `BPMN ${elementType}`,
                config: this.getActivityConfig(config, location.nodeId),
            };
        }
        
        return {
            id: location.nodeId,
            name: location.nodeId,
            type: "task",
            description: "BPMN element",
        };
    }
    
    // Private helpers
    private getBpmnData(config: RoutineVersionConfigObject): string {
        const graph = config.graph as GraphBpmnConfig;
        return graph.schema.data;
    }
    
    private hasIncomingFlow(bpmnData: string, nodeId: string): boolean {
        return !!bpmnData.match(new RegExp(`<bpmn:sequenceFlow[^>]*targetRef="${nodeId}"`));
    }
    
    private evaluateSimpleCondition(condition: string, context: Record<string, unknown>): boolean {
        // Very simple condition evaluation - only basic variable checks
        try {
            const varMatch = condition.match(/^([a-zA-Z_$][a-zA-Z0-9_$]*)$/);
            if (varMatch && varMatch[1]) {
                return !!context[varMatch[1]];
            }
            
            const equalityMatch = condition.match(/^([a-zA-Z_$][a-zA-Z0-9_$]*)\s*==\s*(.+)$/);
            if (equalityMatch) {
                const varName = equalityMatch[1];
                const expectedValue = equalityMatch[2].replace(/['"]/g, ""); // Remove quotes
                return context[varName] === expectedValue;
            }
            
            return true; // If we can't evaluate, proceed
        } catch {
            return true;
        }
    }
    
    private mapBpmnType(bpmnType: string): string {
        const typeMap: Record<string, string> = {
            "startEvent": "start",
            "endEvent": "end",
            "task": "task",
            "userTask": "user",
            "serviceTask": "service",
            "callActivity": "subroutine",
            "exclusiveGateway": "decision",
            "parallelGateway": "parallel",
        };
        return typeMap[bpmnType] || "task";
    }
    
    private getActivityConfig(config: RoutineVersionConfigObject, nodeId: string): Record<string, unknown> | undefined {
        const graph = config.graph as GraphBpmnConfig;
        return graph.schema.activityMap?.[nodeId];
    }
    
    private generateRoutineId(config: RoutineVersionConfigObject): string {
        const bpmnData = this.getBpmnData(config);
        const processMatch = bpmnData.match(/<bpmn:process[^>]*id="([^"]*)"[^>]*>/);
        const processId = processMatch?.[1] || "process";
        return `bpmn_${config.__version}_${processId}`;
    }
}
