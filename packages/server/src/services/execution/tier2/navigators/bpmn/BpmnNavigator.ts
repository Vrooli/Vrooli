import { type RoutineVersionConfigObject, type GraphBpmnConfig } from "@vrooli/shared";
import { BaseNavigator } from "../BaseNavigator.js";
import { type Location, type StepInfo, type EnhancedExecutionContext, type AbstractLocation, type LocationType } from "../../types.js";
import { type BpmnModel, createBpmnModel } from "./BpmnModel.js";
import { contextTransformer, ContextUtils } from "../contextTransformer.js";
import { BpmnEventHandler } from "./BpmnEventHandler.js";
import { BpmnParallelHandler } from "./BpmnParallelHandler.js";
import { BpmnIntermediateHandler } from "./BpmnIntermediateHandler.js";
import { BpmnSubprocessHandler } from "./BpmnSubprocessHandler.js";
import { BpmnInclusiveHandler } from "./BpmnInclusiveHandler.js";
import { BpmnEventSubprocessHandler } from "./BpmnEventSubprocessHandler.js";
import { BpmnAdvancedActivityHandler } from "./BpmnAdvancedActivityHandler.js";
import BpmnModdle from "bpmn-moddle";

/**
 * Utility function for case-insensitive BPMN type comparison
 * Ensures consistent type checking across BPMN element comparisons
 */
function isElementType(element: any, expectedType: string): boolean {
    if (!element?.$type) return false;
    return element.$type.toLowerCase() === expectedType.toLowerCase();
}

/**
 * BpmnNavigator - Pure BPMN navigation
 * 
 * Handles BPMN workflow navigation by parsing XML structure.
 * No event handling, timeouts, or complex trigger logic.
 * 
 * ⚠️  MAJOR TODO: COMPREHENSIVE BPMN 2.0 SUPPORT NEEDED ⚠️
 * 
 * Current implementation only supports basic BPMN elements using regex-based XML parsing.
 * This approach is fundamentally limited and missing critical BPMN 2.0 features.
 * 
 * ## CRITICAL GAPS - HIGH PRIORITY:
 * 
 * ### 1. EVENT HANDLING (BLOCKING ISSUE)
 * - ❌ Boundary Events: timer, error, message, signal, compensation
 * - ❌ Intermediate Events: throw/catch events  
 * - ❌ Event Gateways: event-based decision points
 * - 💡 Impact: Cannot handle timeouts, error recovery, or event-driven workflows
 * - 🔧 Fix: Implement proper event detection and handling in navigation logic
 * 
 * ### 2. SUBPROCESS SUPPORT (CRITICAL FOR COMPLEX WORKFLOWS)
 * - ❌ Embedded Subprocesses: nested workflow structures
 * - ❌ Ad-hoc Subprocesses: dynamic task creation
 * - ❌ Event Subprocesses: interrupting/non-interrupting event handling
 * - ❌ Transaction Subprocesses: compensation and rollback handling
 * - 💡 Impact: Cannot handle hierarchical or complex business processes
 * - 🔧 Fix: Add subprocess navigation and scope management
 * 
 * ### 3. ADVANCED GATEWAY TYPES
 * - ❌ Inclusive Gateways: OR semantics (multiple path activation)
 * - ❌ Complex Gateways: custom condition logic
 * - ❌ Event-based Gateways: wait for specific events
 * - 💡 Impact: Limited decision-making capabilities in workflows
 * - 🔧 Fix: Extend gateway handling beyond exclusive/parallel
 * 
 * ## IMPORTANT GAPS - MEDIUM PRIORITY:
 * 
 * ### 4. ACTIVITY CHARACTERISTICS
 * - ❌ Multi-instance Activities: parallel/sequential loops over collections
 * - ❌ Loop Characteristics: repeat activities based on conditions
 * - ❌ Compensation Handlers: undo/rollback logic for activities
 * - 💡 Impact: Cannot handle batch processing or error recovery patterns
 * - 🔧 Fix: Parse and respect activity characteristics from XML
 * 
 * ### 5. DATA FLOW MODELING
 * - ❌ Data Objects: workflow data modeling and associations
 * - ❌ Data Stores: persistent data references
 * - ❌ Message Flows: inter-process communication
 * - 💡 Impact: No data modeling or cross-process integration
 * - 🔧 Fix: Add data object parsing and association tracking
 * 
 * ## SPECIALIZED FEATURES - LOW PRIORITY:
 * 
 * ### 6. COLLABORATION FEATURES
 * - ❌ Pools and Lanes: organizational structure modeling
 * - ❌ Conversation Diagrams: high-level process interaction
 * - ❌ Choreography: service orchestration patterns
 * - 💡 Impact: Limited to single-process workflows
 * - 🔧 Fix: Add collaboration diagram support
 * 
 * ## ROOT CAUSE: REGEX-BASED XML PARSING
 * 
 * The fundamental issue is using regex patterns to parse XML (lines 36, 42, 60-63, etc.).
 * This approach:
 * - Cannot handle XML structure and hierarchy
 * - Ignores parent-child relationships (e.g., boundary events attached to activities)
 * - Cannot navigate complex nested elements (e.g., subprocess contents)
 * - Doesn't validate against BPMN 2.0 specification
 * 
 * ## RECOMMENDED SOLUTION APPROACH:
 * 
 * ### Phase 1: XML Parser Foundation (2-3 days)
 * Replace regex parsing with proper XML DOM parsing:
 * ```typescript
 * private parseXML(bpmnData: string): Document {
 *     const parser = new DOMParser();
 *     return parser.parseFromString(bpmnData, "text/xml");
 * }
 * 
 * private findBoundaryEvents(activityElement: Element): BoundaryEvent[] {
 *     return Array.from(activityElement.parentElement?.querySelectorAll(
 *         `bpmn\\:boundaryEvent[attachedToRef="${activityElement.id}"]`
 *     ) || []);
 * }
 * ```
 * 
 * ### Phase 2: BPMN Object Model (1-2 weeks)
 * Implement proper BPMN domain model:
 * ```typescript
 * interface BpmnProcess {
 *     elements: Map<string, BpmnElement>;
 *     flows: SequenceFlow[];
 *     boundaryEvents: BoundaryEvent[];
 *     subprocesses: Subprocess[];
 * }
 * 
 * interface BpmnElement {
 *     id: string;
 *     type: BpmnElementType;
 *     boundaryEvents: BoundaryEvent[];
 *     loopCharacteristics?: LoopCharacteristics;
 *     multiInstanceType?: MultiInstanceType;
 * }
 * ```
 * 
 * ### Phase 3: Event System Integration (1 week)
 * Integrate with existing SwarmContextManager for event handling:
 * ```typescript
 * interface BpmnEventHandler {
 *     handleTimerEvent(event: TimerBoundaryEvent): Promise<Location[]>;
 *     handleErrorEvent(event: ErrorBoundaryEvent): Promise<Location[]>;
 *     handleMessageEvent(event: MessageEvent): Promise<Location[]>;
 * }
 * ```
 * 
 * ## TESTING STATUS:
 * Comprehensive tests for missing features have been added to BpmnNavigator.test.ts.
 * All advanced BPMN feature tests currently FAIL as expected, demonstrating the gaps.
 * 
 * ## EFFORT ESTIMATE: 3-4 weeks for complete BPMN 2.0 support
 * 
 * ## NEXT STEPS:
 * 1. Implement XML DOM parser foundation
 * 2. Add boundary event detection and handling  
 * 3. Implement subprocess navigation
 * 4. Add support for inclusive/complex gateways
 * 5. Update all tests to validate new capabilities
 * 
 * 📍 CURRENT STATUS: Basic sequential flow navigation only
 * 🎯 TARGET STATUS: Full BPMN 2.0 specification compliance
 */
export class BpmnNavigator extends BaseNavigator {
    readonly type = "bpmn";
    readonly version = "1.0.0";
    
    // Use proper bpmn-moddle like the working reference implementation
    private bpmnModdle = new BpmnModdle();
    
    // Cache for parsed BPMN definitions (like the working example)
    private definitionsCache = new Map<string, any>();
    
    // Phase 2: BPMN Feature Handlers
    private eventHandler = new BpmnEventHandler();
    private parallelHandler = new BpmnParallelHandler();
    private intermediateHandler = new BpmnIntermediateHandler();
    private subprocessHandler = new BpmnSubprocessHandler();
    private inclusiveHandler = new BpmnInclusiveHandler();
    private eventSubprocessHandler = new BpmnEventSubprocessHandler();
    private advancedActivityHandler = new BpmnAdvancedActivityHandler();
    
    canNavigate(routine: unknown): boolean {
        try {
            const config = routine as RoutineVersionConfigObject;
            return !!(
                config.__version &&
                config.graph?.__type === "BPMN-2.0" &&
                config.graph.schema?.data !== undefined &&
                typeof config.graph.schema.data === "string" &&
                (config.graph.schema.data === "" || config.graph.schema.data.includes("xmlns:bpmn"))
            );
        } catch {
            return false;
        }
    }
    
    /**
     * Get BPMN definitions using proper async parsing (like the working example)
     */
    private async getDefinitions(config: RoutineVersionConfigObject): Promise<any> {
        const bpmnData = this.getBpmnData(config);
        
        // Check cache first
        if (this.definitionsCache.has(bpmnData)) {
            return this.definitionsCache.get(bpmnData);
        }
        
        // Parse using callback-based approach like the working example
        const [error, definitions] = await new Promise<[Error | null, any]>((resolve) => {
            this.bpmnModdle.fromXML(bpmnData, (err: Error | null, result: any) => {
                if (err) {
                    resolve([err, null]);
                } else {
                    resolve([null, result.rootElement]);
                }
            });
        });
        
        if (error) {
            throw new Error(`Error parsing BPMN XML: ${error.message}`);
        }
        
        // Cache the result
        this.definitionsCache.set(bpmnData, definitions);
        return definitions;
    }
    
    /**
     * Find element by ID in definitions (like the working example)
     */
    private findElementById(definitions: any, elementId: string): any {
        if (!definitions || !definitions.rootElements) {
            return null;
        }
        
        // Simple recursive search through the BPMN structure
        function searchInElement(element: any): any {
            if (!element) return null;
            
            if (element.id === elementId) {
                return element;
            }
            
            // Search in flowElements (processes, subprocesses)
            if (element.flowElements) {
                for (const flowElement of element.flowElements) {
                    const found = searchInElement(flowElement);
                    if (found) return found;
                }
            }
            
            // Search in rootElements
            if (element.rootElements) {
                for (const rootElement of element.rootElements) {
                    const found = searchInElement(rootElement);
                    if (found) return found;
                }
            }
            
            return null;
        }
        
        return searchInElement(definitions);
    }
    
    getStartLocation(routine: unknown): Location {
        const config = this.validateRoutine(routine);
        const routineId = this.generateRoutineId(config);
        
        try {
            // Use legacy implementation directly - avoids broken model parsing
            return this.getStartLocationLegacy(config, routineId);
        } catch (error) {
            // Ensure error is thrown for malformed BPMN (for test compatibility)
            throw new Error("No start location found in BPMN");
        }
    }
    
    getNextLocations(routine: unknown, current: Location, context?: Record<string, unknown>): Location[] {
        const config = this.validateRoutine(routine);
        
        // Use legacy implementation for now - enhanced version has parsing issues
        return this.getNextLocationsLegacy(config, current, context || {});
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
    
    /**
     * Async version of getStartLocation using proper BPMN parsing
     */
    async getStartLocationAsync(routine: unknown): Promise<Location> {
        const config = this.validateRoutine(routine);
        const routineId = this.generateRoutineId(config);
        
        try {
            const definitions = await this.getDefinitions(config);
            
            // Find start events in the definitions
            if (definitions.rootElements) {
                for (const rootElement of definitions.rootElements) {
                    if (isElementType(rootElement, "bpmn:Process") && rootElement.flowElements) {
                        for (const flowElement of rootElement.flowElements) {
                            if (isElementType(flowElement, "bpmn:StartEvent")) {
                                return this.createLocation(flowElement.id, routineId);
                            }
                        }
                    }
                }
            }
            
            throw new Error("No start event found in BPMN");
        } catch (error) {
            // Fallback to legacy implementation
            return this.getStartLocationLegacy(config, routineId);
        }
    }
    
    /**
     * Async version of getNextLocations using proper BPMN parsing
     */
    async getNextLocationsAsync(routine: unknown, current: Location, context?: Record<string, unknown>): Promise<Location[]> {
        const config = this.validateRoutine(routine);
        
        try {
            const definitions = await this.getDefinitions(config);
            const currentElement = this.findElementById(definitions, current.nodeId);
            
            if (!currentElement) {
                return [];
            }
            
            // Find outgoing sequence flows
            const nextLocations: Location[] = [];
            if (currentElement.outgoing) {
                for (const outgoingRef of currentElement.outgoing) {
                    const sequenceFlow = this.findElementById(definitions, outgoingRef.id);
                    if (sequenceFlow && sequenceFlow.targetRef) {
                        const targetElement = this.findElementById(definitions, sequenceFlow.targetRef.id);
                        if (targetElement) {
                            nextLocations.push(this.createLocation(targetElement.id, current.routineId));
                        }
                    }
                }
            }
            
            return nextLocations;
        } catch (error) {
            // Fallback to legacy implementation
            return this.getNextLocationsLegacy(config, current, context || {});
        }
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
    
    // Phase 1 Foundation: New enhanced methods
    
    /**
     * Legacy method - kept for backward compatibility but not used in main flow
     */
    private getBpmnModel(config: RoutineVersionConfigObject): BpmnModel {
        // This method is deprecated - use getDefinitions() instead
        throw new Error("getBpmnModel is deprecated - use async getDefinitions instead");
    }
    
    /**
     * Enhanced context conversion
     */
    private enhanceContext(basicContext: Record<string, unknown>): EnhancedExecutionContext {
        return contextTransformer.enhance(basicContext);
    }
    
    /**
     * Check if location is abstract location
     */
    private isAbstractLocation(location: Location): boolean {
        return "locationType" in location;
    }
    
    /**
     * Enhanced navigation for complex BPMN features
     */
    private getNextLocationsEnhanced(model: BpmnModel, current: Location, context: EnhancedExecutionContext): Location[] {
        const abstractCurrent = current as AbstractLocation;
        
        // Handle abstract location navigation
        if (this.isAbstractLocation(current)) {
            return this.handleAbstractLocationNavigation(model, abstractCurrent, context);
        }
        
        // Get element from model
        const currentElement = model.getElementById(current.nodeId);
        if (!currentElement) {
            return [];
        }
        
        // Determine element type and process accordingly
        const elementType = model.getElementType(current.nodeId);
        
        try {
            if (elementType && elementType.toLowerCase() === "bpmn:parallelgateway") {
                return this.handleParallelGateway(model, current.nodeId, abstractCurrent, context);
            } else if (elementType && (elementType.toLowerCase() === "bpmn:intermediatethrowevent" || elementType.toLowerCase() === "bpmn:intermediatecatchevent")) {
                return this.handleIntermediateEvent(model, current.nodeId, abstractCurrent, context);
            } else if (elementType && (elementType.toLowerCase() === "bpmn:callactivity" || elementType.toLowerCase() === "bpmn:subprocess")) {
                return this.handleSubprocess(model, current.nodeId, abstractCurrent, context);
            } else if (model.isActivity(current.nodeId)) {
                return this.handleActivity(model, current.nodeId, abstractCurrent, context);
            } else if (model.isGateway(current.nodeId)) {
                return this.handleGateway(model, current.nodeId, abstractCurrent, context);
            } else {
                // Default flow navigation for simple elements
                return this.handleSimpleNavigation(model, current.nodeId, abstractCurrent, context);
            }
        } catch (error) {
            // If enhanced processing fails, fall back to simple navigation
            console.warn(`Enhanced BPMN processing failed for ${current.nodeId}, falling back to simple navigation:`, error);
            return this.handleSimpleNavigation(model, current.nodeId, abstractCurrent, context);
        }
    }
    
    /**
     * Legacy navigation for backward compatibility
     */
    private getNextLocationsLegacy(config: RoutineVersionConfigObject, current: Location, context: Record<string, unknown>): Location[] {
        const bpmnData = this.getBpmnData(config);
        const nextLocations: Location[] = [];
        
        // Find all sequence flows that have this node as sourceRef
        // First, find all sequenceFlow elements
        const flowMatches = Array.from(bpmnData.matchAll(/<bpmn:sequenceFlow[^>]*>/g));
        
        for (const flowMatch of flowMatches) {
            const flowElement = flowMatch[0];
            
            // Extract sourceRef attribute
            const sourceRefMatch = flowElement.match(/sourceRef="([^"]*)"/);
            if (!sourceRefMatch || sourceRefMatch[1] !== current.nodeId) {
                continue;
            }
            
            // Extract targetRef attribute
            const targetRefMatch = flowElement.match(/targetRef="([^"]*)"/);
            if (!targetRefMatch || !targetRefMatch[1]) {
                continue;
            }
            
            const targetRef = targetRefMatch[1];
            
            // Check if this flow has a condition
            let condition: string | undefined;
            
            if (!flowElement.endsWith("/>")) {
                // Not self-closing, might have condition
                // Find the position of this flow element and look for its closing tag
                const flowStartIndex = bpmnData.indexOf(flowElement);
                if (flowStartIndex !== -1) {
                    // Look for the closing tag after this position
                    const afterFlow = bpmnData.substring(flowStartIndex);
                    const closingMatch = afterFlow.match(/<\/bpmn:sequenceFlow>/);
                    
                    if (closingMatch && closingMatch.index !== undefined) {
                        // Extract the content between opening and closing tags
                        const flowContent = afterFlow.substring(flowElement.length, closingMatch.index);
                        
                        // Look for condition expression within this content
                        const conditionMatch = flowContent.match(/<bpmn:conditionExpression[^>]*>([^<]*)<\/bpmn:conditionExpression>/);
                        if (conditionMatch) {
                            condition = conditionMatch[1];
                        }
                    }
                }
            }
            
            // Simple condition evaluation - if no context or can't evaluate, proceed (return true)
            if (condition && Object.keys(context).length > 0 && !this.evaluateSimpleCondition(condition.trim(), context)) {
                continue;
            }
            
            nextLocations.push(this.createLocation(targetRef, current.routineId));
        }
        
        return nextLocations;
    }
    
    /**
     * Legacy start location for backward compatibility
     */
    private getStartLocationLegacy(config: RoutineVersionConfigObject, routineId: string): Location {
        const bpmnData = this.getBpmnData(config);
        
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
    
    /**
     * Handle abstract location navigation (Phase 2 implementation)
     */
    private handleAbstractLocationNavigation(model: BpmnModel, location: AbstractLocation, context: EnhancedExecutionContext): Location[] {
        try {
            switch (location.locationType) {
                case "boundary_event_monitor":
                    return this.handleBoundaryEventMonitoring(model, location, context);
                case "parallel_branch":
                    return this.handleParallelBranchNavigation(model, location, context);
                case "timer_waiting":
                case "message_waiting": 
                case "signal_waiting":
                case "event_waiting":
                    return this.handleEventWaiting(model, location, context);
                case "subprocess_context":
                    return this.handleSubprocessContext(model, location, context);
                case "gateway_evaluation":
                    return this.handleGatewayEvaluation(model, location, context);
                case "multi_instance_execution":
                    return this.handleMultiInstanceExecution(model, location, context);
                case "multi_instance_waiting":
                    return this.handleMultiInstanceWaiting(model, location, context);
                case "loop_execution":
                    return this.handleLoopExecution(model, location, context);
                default:
                    // Default to normal node navigation for the base node
                    const baseLocation: Location = {
                        id: location.parentNodeId || location.nodeId,
                        routineId: location.routineId,
                        nodeId: location.parentNodeId || location.nodeId,
                    };
                    return this.getNextLocationsEnhanced(model, baseLocation, context);
            }
        } catch (error) {
            console.warn(`Abstract location navigation failed for ${location.locationType}:`, error);
            return [];
        }
    }
    
    // Phase 2: Enhanced Handler Implementations
    
    /**
     * Handle parallel gateway processing
     */
    private handleParallelGateway(model: BpmnModel, gatewayId: string, current: AbstractLocation, context: EnhancedExecutionContext): Location[] {
        const result = this.parallelHandler.processParallelGateway(model, gatewayId, current, context);
        
        // Update context with parallel execution state
        Object.assign(context, result.updatedContext);
        
        return result.nextLocations;
    }
    
    /**
     * Handle intermediate event processing
     */
    private handleIntermediateEvent(model: BpmnModel, eventId: string, current: AbstractLocation, context: EnhancedExecutionContext): Location[] {
        const result = this.intermediateHandler.processIntermediateEvent(model, eventId, current, context);
        
        // Update context with event state
        Object.assign(context, result.updatedContext);
        
        // Handle external events (signals, messages, etc.)
        if (result.thrownEvents.length > 0) {
            // These would be published to external event bus
            console.log("Thrown events:", result.thrownEvents);
        }
        
        return result.nextLocations;
    }
    
    /**
     * Handle subprocess processing
     */
    private handleSubprocess(model: BpmnModel, subprocessId: string, current: AbstractLocation, context: EnhancedExecutionContext): Location[] {
        const result = this.subprocessHandler.processSubprocess(model, subprocessId, current, context);
        
        // Update context with subprocess state
        Object.assign(context, result.updatedContext);
        
        // Handle subprocess calls (call activities)
        if (result.subprocessCall) {
            // This would trigger external routine execution
            console.log("Subprocess call:", result.subprocessCall);
        }
        
        return result.nextLocations;
    }
    
    /**
     * Handle activity with boundary events
     */
    private handleActivity(model: BpmnModel, activityId: string, current: AbstractLocation, context: EnhancedExecutionContext): Location[] {
        const nextLocations: Location[] = [];
        
        // Check for advanced activity characteristics (multi-instance, loops, compensation)
        const advancedResult = this.advancedActivityHandler.processAdvancedActivity(model, activityId, current, context);
        
        if (advancedResult.nextLocations.length > 0) {
            // Advanced activity is handling the execution
            Object.assign(context, advancedResult.updatedContext);
            nextLocations.push(...advancedResult.nextLocations.map(loc => this.convertAbstractToLocation(loc)));
            
            if (advancedResult.activityComplete) {
                // Activity completed through advanced processing
                return nextLocations;
            }
        }
        
        // Process boundary events attached to this activity
        const eventResult = this.eventHandler.processBoundaryEvents(model, current, context);
        
        // Update context with event state
        Object.assign(context, eventResult.updatedContext);
        
        // Add boundary event monitoring locations
        nextLocations.push(...eventResult.nextLocations);
        
        // Monitor for event subprocess triggers
        const eventSubprocessResult = this.eventSubprocessHandler.monitorEventSubprocesses(model, current, context);
        
        // Update context with event subprocess state
        Object.assign(context, eventSubprocessResult.updatedContext);
        
        // Add event subprocess locations if any are triggered
        nextLocations.push(...eventSubprocessResult.nextLocations.map(loc => this.convertAbstractToLocation(loc)));
        
        // Handle subprocess activations
        if (eventSubprocessResult.subprocessActivations.length > 0) {
            console.log("Event subprocesses activated:", eventSubprocessResult.subprocessActivations);
        }
        
        // Handle external fired events
        if (eventResult.firedEvents.length > 0) {
            console.log("Fired boundary events:", eventResult.firedEvents);
        }
        
        // If no interrupting boundary event or event subprocess fired, continue with normal flow
        if (!eventResult.shouldTerminate && !eventSubprocessResult.shouldInterruptMainProcess) {
            const simpleFlow = this.handleSimpleNavigation(model, activityId, current, context);
            nextLocations.push(...simpleFlow);
        }
        
        return nextLocations;
    }
    
    /**
     * Handle general gateway processing
     */
    private handleGateway(model: BpmnModel, gatewayId: string, current: AbstractLocation, context: EnhancedExecutionContext): Location[] {
        const elementType = model.getElementType(gatewayId);
        
        switch (elementType?.toLowerCase()) {
            case "bpmn:parallelgateway":
                return this.handleParallelGateway(model, gatewayId, current, context);
            case "bpmn:exclusivegateway":
                return this.handleExclusiveGateway(model, gatewayId, current, context);
            case "bpmn:inclusivegateway":
                return this.handleInclusiveGateway(model, gatewayId, current, context);
            default:
                // Fallback to simple navigation
                return this.handleSimpleNavigation(model, gatewayId, current, context);
        }
    }
    
    /**
     * Handle exclusive gateway (basic conditional routing)
     */
    private handleExclusiveGateway(model: BpmnModel, gatewayId: string, current: AbstractLocation, context: EnhancedExecutionContext): Location[] {
        const nextLocations: Location[] = [];
        const outgoingFlows = model.getOutgoingFlows(gatewayId);
        
        // Evaluate conditions and take first matching path
        for (const flow of outgoingFlows) {
            if (flow.targetRef && model.evaluateFlowCondition(flow, context.variables)) {
                const nextLocation = model.createAbstractLocation(
                    flow.targetRef.id,
                    current.routineId,
                    "node",
                    { parentNodeId: flow.targetRef.id },
                );
                nextLocations.push(nextLocation);
                break; // Exclusive - only one path
            }
        }
        
        return nextLocations;
    }
    
    /**
     * Handle inclusive gateway (OR-semantics with multi-path activation)
     */
    private handleInclusiveGateway(model: BpmnModel, gatewayId: string, current: AbstractLocation, context: EnhancedExecutionContext): Location[] {
        const result = this.inclusiveHandler.processInclusiveGateway(model, gatewayId, current, context);
        
        // Update context with inclusive gateway state changes
        this.updateContextAfterProcessing(context, result.updatedContext);
        
        // Convert abstract locations to regular locations
        return result.nextLocations.map(loc => this.convertAbstractToLocation(loc));
    }
    
    /**
     * Handle simple navigation (basic flow following)
     */
    private handleSimpleNavigation(model: BpmnModel, nodeId: string, current: AbstractLocation, context: EnhancedExecutionContext): Location[] {
        const nextLocations: Location[] = [];
        const outgoingFlows = model.getOutgoingFlows(nodeId);
        
        for (const flow of outgoingFlows) {
            if (flow.targetRef && model.evaluateFlowCondition(flow, context.variables)) {
                const nextLocation = model.createAbstractLocation(
                    flow.targetRef.id,
                    current.routineId,
                    "node",
                    { parentNodeId: flow.targetRef.id },
                );
                nextLocations.push(nextLocation);
            }
        }
        
        return nextLocations;
    }
    
    // Abstract Location Handlers
    
    /**
     * Handle boundary event monitoring
     */
    private handleBoundaryEventMonitoring(model: BpmnModel, location: AbstractLocation, context: EnhancedExecutionContext): Location[] {
        const eventResult = this.eventHandler.processBoundaryEvents(model, location, context);
        
        // Update context
        Object.assign(context, eventResult.updatedContext);
        
        return eventResult.nextLocations;
    }
    
    /**
     * Handle parallel branch navigation
     */
    private handleParallelBranchNavigation(model: BpmnModel, location: AbstractLocation, context: EnhancedExecutionContext): Location[] {
        // Continue normal navigation within the parallel branch
        const baseNodeId = location.parentNodeId || location.nodeId;
        return this.handleSimpleNavigation(model, baseNodeId, location, context);
    }
    
    /**
     * Handle event waiting states
     */
    private handleEventWaiting(model: BpmnModel, location: AbstractLocation, context: EnhancedExecutionContext): Location[] {
        // Check if the event we're waiting for has occurred
        switch (location.locationType) {
            case "timer_waiting":
                return this.handleTimerWaiting(model, location, context);
            case "message_waiting":
                return this.handleMessageWaiting(model, location, context);
            case "signal_waiting":
                return this.handleSignalWaiting(model, location, context);
            default:
                // Continue waiting
                return [location];
        }
    }
    
    /**
     * Handle subprocess context navigation
     */
    private handleSubprocessContext(model: BpmnModel, location: AbstractLocation, context: EnhancedExecutionContext): Location[] {
        if (location.subprocessId) {
            return this.handleSubprocess(model, location.subprocessId, location, context);
        }
        return [];
    }
    
    /**
     * Handle gateway evaluation states
     */
    private handleGatewayEvaluation(model: BpmnModel, location: AbstractLocation, context: EnhancedExecutionContext): Location[] {
        const gatewayId = location.parentNodeId || location.nodeId;
        return this.handleGateway(model, gatewayId, location, context);
    }
    
    // Event Waiting Handlers
    
    private handleTimerWaiting(model: BpmnModel, location: AbstractLocation, context: EnhancedExecutionContext): Location[] {
        const eventId = location.eventId;
        if (!eventId) return [location];
        
        const result = this.intermediateHandler.processIntermediateEvent(model, eventId, location, context);
        Object.assign(context, result.updatedContext);
        
        return result.shouldWait ? [location] : result.nextLocations;
    }
    
    private handleMessageWaiting(model: BpmnModel, location: AbstractLocation, context: EnhancedExecutionContext): Location[] {
        const eventId = location.eventId;
        if (!eventId) return [location];
        
        const result = this.intermediateHandler.processIntermediateEvent(model, eventId, location, context);
        Object.assign(context, result.updatedContext);
        
        return result.shouldWait ? [location] : result.nextLocations;
    }
    
    private handleSignalWaiting(model: BpmnModel, location: AbstractLocation, context: EnhancedExecutionContext): Location[] {
        const eventId = location.eventId;
        if (!eventId) return [location];
        
        const result = this.intermediateHandler.processIntermediateEvent(model, eventId, location, context);
        Object.assign(context, result.updatedContext);
        
        return result.shouldWait ? [location] : result.nextLocations;
    }
    
    // Advanced Activity Handlers
    
    private handleMultiInstanceExecution(model: BpmnModel, location: AbstractLocation, context: EnhancedExecutionContext): Location[] {
        const activityId = location.parentNodeId || location.nodeId;
        const result = this.advancedActivityHandler.processAdvancedActivity(model, activityId, location, context);
        
        Object.assign(context, result.updatedContext);
        return result.nextLocations.map(loc => this.convertAbstractToLocation(loc));
    }
    
    private handleMultiInstanceWaiting(model: BpmnModel, location: AbstractLocation, context: EnhancedExecutionContext): Location[] {
        const activityId = location.parentNodeId || location.nodeId;
        const result = this.advancedActivityHandler.processAdvancedActivity(model, activityId, location, context);
        
        Object.assign(context, result.updatedContext);
        
        if (result.activityComplete) {
            return result.nextLocations.map(loc => this.convertAbstractToLocation(loc));
        } else {
            // Still waiting
            return [this.convertAbstractToLocation(location)];
        }
    }
    
    private handleLoopExecution(model: BpmnModel, location: AbstractLocation, context: EnhancedExecutionContext): Location[] {
        const activityId = location.parentNodeId || location.nodeId;
        const result = this.advancedActivityHandler.processAdvancedActivity(model, activityId, location, context);
        
        Object.assign(context, result.updatedContext);
        return result.nextLocations.map(loc => this.convertAbstractToLocation(loc));
    }
    
    /**
     * Generate cache key for BPMN model
     */
    private generateCacheKey(config: RoutineVersionConfigObject): string {
        const bpmnData = this.getBpmnData(config);
        // Simple hash of BPMN data for caching
        let hash = 0;
        for (let i = 0; i < bpmnData.length; i++) {
            const char = bpmnData.charCodeAt(i);
            hash = ((hash << 5) - hash) + char;
            hash = hash & hash; // Convert to 32-bit integer
        }
        return `bpmn_${config.__version}_${Math.abs(hash).toString(16)}`;
    }
    
    /**
     * Synchronous XML parsing wrapper (Phase 1 compatibility)
     * Will be updated in Phase 2 to handle async properly
     */
    private parseXmlSync(bpmnXml: string): any {
        // Phase 1: Basic synchronous parsing
        // This is a temporary solution - Phase 2 will properly handle async parsing
        try {
            // For now, just validate that it's valid XML
            if (!bpmnXml.includes("xmlns:bpmn")) {
                throw new Error("Invalid BPMN XML: Missing namespace");
            }
            return true;
        } catch (error) {
            throw new Error(`XML parsing failed: ${error instanceof Error ? error.message : String(error)}`);
        }
    }
    
    // Phase 3: Helper methods for advanced BPMN features
    
    /**
     * Update context after handler processing
     */
    private updateContextAfterProcessing(targetContext: EnhancedExecutionContext, sourceContext: EnhancedExecutionContext): void {
        // Update variables
        Object.assign(targetContext.variables, sourceContext.variables);
        
        // Update events
        targetContext.events = { ...sourceContext.events };
        
        // Update parallel execution
        targetContext.parallelExecution = { ...sourceContext.parallelExecution };
        
        // Update subprocesses
        targetContext.subprocesses = { ...sourceContext.subprocesses };
        
        // Update external events
        targetContext.external = { ...sourceContext.external };
        
        // Update gateways
        targetContext.gateways = { ...sourceContext.gateways };
    }
    
    /**
     * Convert abstract location to regular location for interface compatibility
     */
    private convertAbstractToLocation(abstractLocation: AbstractLocation): Location {
        return {
            id: abstractLocation.id,
            nodeId: abstractLocation.nodeId,
            routineId: abstractLocation.routineId,
        };
    }
}
