/**
 * BpmnModel - Proper BPMN XML parsing and model navigation
 * 
 * Replaces regex-based XML parsing with structured DOM navigation using bpmn-moddle.
 * Provides high-level BPMN element access and relationship traversal.
 */

import BpmnModdle from "bpmn-moddle";

/**
 * Utility function for case-insensitive BPMN type comparison
 * 
 * BPMN-moddle may use different case conventions than XML elements.
 * This ensures robust type checking regardless of case variations.
 */
function isElementType(element: any, expectedType: string): boolean {
    if (!element?.$type) return false;
    return element.$type.toLowerCase() === expectedType.toLowerCase();
}

/**
 * Check if element type matches any of the provided types (case-insensitive)
 */
function isAnyElementType(element: any, expectedTypes: string[]): boolean {
    if (!element?.$type) return false;
    const elementType = element.$type.toLowerCase();
    return expectedTypes.some(type => elementType === type.toLowerCase());
}

// Note: bpmn-moddle types are limited, using any for Phase 1 compatibility
type ModdleElement = any;
type Process = any;
type FlowElement = any;
type SequenceFlow = any;
type StartEvent = any;
type EndEvent = any;
type Task = any;
type Gateway = any;
type BpmnBoundaryEvent = any;
type BpmnIntermediateEvent = any;
type SubProcess = any;
type Activity = any;
import type { AbstractLocation, LocationType, EnhancedExecutionContext } from "../../types.js";

/**
 * Structured BPMN model with proper element relationship management
 */
export class BpmnModel {
    private moddle: any;
    private definitions: ModdleElement | null = null;
    private processes: Process[] = [];
    private elements: Map<string, FlowElement> = new Map();
    private flows: Map<string, SequenceFlow> = new Map();
    private boundaryEvents: Map<string, BpmnBoundaryEvent[]> = new Map(); // Activity ID -> Boundary Events

    constructor() {
        this.moddle = new BpmnModdle();
    }

    /**
     * Load BPMN XML (alias for parseXml for backward compatibility)
     */
    async loadXml(bpmnXml: string): Promise<void> {
        return this.parseXml(bpmnXml);
    }

    /**
     * Parse BPMN XML and build structured model (async)
     * Uses proper callback-based pattern like the working reference implementation
     */
    async parseXml(bpmnXml: string): Promise<void> {
        try {
            // bpmn-moddle v9+ uses Promise-based API
            const result = await this.moddle.fromXML(bpmnXml);
            
            if (!result || !result.rootElement) {
                throw new Error("No root element found in BPMN XML");
            }

            this.definitions = result.rootElement;
            this.buildElementMaps();
        } catch (error) {
            throw new Error(`Failed to parse BPMN XML: ${error instanceof Error ? error.message : String(error)}`);
        }
    }

    /**
     * Parse BPMN XML synchronously (Phase 1 compatibility)
     * 
     * Uses proper bpmn-moddle parsing in a synchronous wrapper.
     */
    parseXmlSync(bpmnXml: string): void {
        // bpmn-moddle v9+ only supports Promise-based API
        // For sync compatibility, we would need to use a different approach
        throw new Error("Synchronous XML parsing is not supported with bpmn-moddle v9+. Please use parseXml() or loadXml() instead.");
    }

    /**
     * Validate that the BPMN model is properly loaded
     */
    isValid(): boolean {
        return this.definitions !== null && this.processes.length > 0;
    }

    /**
     * Get the main process (first process found)
     */
    getMainProcess(): Process | null {
        return this.processes[0] || null;
    }

    /**
     * Get all processes in the BPMN model
     */
    getAllProcesses(): Process[] {
        return [...this.processes];
    }

    /**
     * Get element by ID with proper typing
     */
    getElementById(id: string): FlowElement | null {
        return this.elements.get(id) || null;
    }

    /**
     * Get all start events in the process
     */
    getStartEvents(processId?: string): StartEvent[] {
        const process = processId ? this.getProcessById(processId) : this.getMainProcess();
        if (!process?.flowElements) return [];

        return process.flowElements.filter((element): element is StartEvent => 
            isElementType(element, "bpmn:StartEvent"),
        );
    }

    /**
     * Get all end events in the process
     */
    getEndEvents(processId?: string): EndEvent[] {
        const process = processId ? this.getProcessById(processId) : this.getMainProcess();
        if (!process?.flowElements) return [];

        return process.flowElements.filter((element): element is EndEvent => 
            isElementType(element, "bpmn:EndEvent"),
        );
    }

    /**
     * Get outgoing sequence flows from an element
     */
    getOutgoingFlows(elementId: string): SequenceFlow[] {
        const element = this.getElementById(elementId);
        if (!element || !element.outgoing) return [];

        return element.outgoing
            .map((ref: any) => this.flows.get(ref.id))
            .filter((flow: any): flow is SequenceFlow => flow !== undefined);
    }

    /**
     * Get incoming sequence flows to an element
     */
    getIncomingFlows(elementId: string): SequenceFlow[] {
        const element = this.getElementById(elementId);
        if (!element || !element.incoming) return [];

        return element.incoming
            .map((ref: any) => this.flows.get(ref.id))
            .filter((flow: any): flow is SequenceFlow => flow !== undefined);
    }

    /**
     * Get boundary events attached to an activity
     */
    getBoundaryEvents(activityId: string): BpmnBoundaryEvent[] {
        return this.boundaryEvents.get(activityId) || [];
    }

    /**
     * Check if element has any outgoing flows
     */
    hasOutgoingFlows(elementId: string): boolean {
        return this.getOutgoingFlows(elementId).length > 0;
    }

    /**
     * Check if element has any incoming flows
     */
    hasIncomingFlows(elementId: string): boolean {
        return this.getIncomingFlows(elementId).length > 0;
    }

    /**
     * Get element type information
     */
    getElementType(elementId: string): string | null {
        const element = this.getElementById(elementId);
        return element?.$type || null;
    }

    /**
     * Get element name or ID if no name
     */
    getElementName(elementId: string): string {
        const element = this.getElementById(elementId);
        return element?.name || elementId;
    }

    /**
     * Check if element is a gateway
     */
    isGateway(elementId: string): boolean {
        const type = this.getElementType(elementId);
        return type ? type.includes("Gateway") : false;
    }

    /**
     * Check if element is a task/activity
     */
    isActivity(elementId: string): boolean {
        const element = this.getElementById(elementId);
        if (!element?.$type) return false;
        
        const elementType = element.$type.toLowerCase();
        return elementType.includes("task") || elementType === "bpmn:callactivity";
    }

    /**
     * Check if element is an event
     */
    isEvent(elementId: string): boolean {
        const type = this.getElementType(elementId);
        return type ? type.includes("Event") : false;
    }

    /**
     * Get subprocess elements (for nested navigation)
     */
    getSubprocessElements(subprocessId: string): FlowElement[] {
        const subprocess = this.getElementById(subprocessId) as SubProcess;
        if (!subprocess || !isElementType(subprocess, "bpmn:SubProcess")) return [];

        return subprocess.flowElements || [];
    }

    /**
     * Evaluate sequence flow conditions
     */
    evaluateFlowCondition(flow: SequenceFlow, context: Record<string, unknown>): boolean {
        if (!flow.conditionExpression) return true;

        const condition = flow.conditionExpression.body?.trim();
        if (!condition) return true;

        return this.evaluateConditionExpression(condition, context);
    }

    /**
     * Create abstract location for complex execution states
     */
    createAbstractLocation(
        nodeId: string, 
        routineId: string, 
        locationType: LocationType = "node",
        options: {
            parentNodeId?: string;
            branchId?: string;
            subprocessId?: string;
            eventId?: string;
            metadata?: Record<string, unknown>;
        } = {},
    ): AbstractLocation {
        const locationId = this.generateLocationId(nodeId, locationType, options);
        
        // Merge all options into metadata for test compatibility
        const metadata = {
            ...(options.metadata || {}),
        };
        
        // Add optional properties to metadata if they exist
        if (options.parentNodeId) metadata.parentNodeId = options.parentNodeId;
        if (options.branchId) metadata.branchId = options.branchId;
        if (options.subprocessId) metadata.subprocessId = options.subprocessId;
        if (options.eventId) metadata.eventId = options.eventId;
        if (options.metadata) metadata.metadata = options.metadata;

        return {
            id: locationId,
            routineId,
            nodeId,
            locationType,
            type: locationType, // Compatibility property for tests
            parentNodeId: options.parentNodeId,
            branchId: options.branchId,
            subprocessId: options.subprocessId,
            eventId: options.eventId,
            metadata,
        };
    }

    /**
     * Parse abstract location to extract state information
     */
    parseAbstractLocation(location: AbstractLocation): {
        baseNodeId: string;
        locationType: LocationType;
        stateInfo: Record<string, unknown>;
    } {
        return {
            baseNodeId: location.parentNodeId || location.nodeId,
            locationType: location.locationType,
            stateInfo: {
                branchId: location.branchId,
                subprocessId: location.subprocessId,
                eventId: location.eventId,
                metadata: location.metadata,
            },
        };
    }

    /**
     * Get all elements of a specific type
     */
    getElementsByType<T extends FlowElement>(type: string): T[] {
        return Array.from(this.elements.values())
            .filter((element): element is T => isElementType(element, type));
    }

    /**
     * Find elements without incoming flows (potential start points)
     */
    getElementsWithoutIncoming(): FlowElement[] {
        return Array.from(this.elements.values())
            .filter(element => !this.hasIncomingFlows(element.id));
    }

    // Private helper methods

    private buildElementMaps(): void {
        if (!this.definitions) return;

        // Extract processes
        this.processes = this.extractProcesses(this.definitions);

        // Build element and flow maps
        for (const process of this.processes) {
            if (process.flowElements) {
                for (const element of process.flowElements) {
                    this.elements.set(element.id, element);

                    // Build sequence flow map
                    if (isElementType(element, "bpmn:SequenceFlow")) {
                        this.flows.set(element.id, element);
                    }

                    // Build boundary events map
                    if (isElementType(element, "bpmn:BoundaryEvent")) {
                        const attachedToRef = element.attachedToRef?.id;
                        if (attachedToRef) {
                            if (!this.boundaryEvents.has(attachedToRef)) {
                                this.boundaryEvents.set(attachedToRef, []);
                            }
                            this.boundaryEvents.get(attachedToRef)!.push(element);
                        }
                    }
                }
            }
        }
    }

    private extractProcesses(definitions: ModdleElement): Process[] {
        const processes: Process[] = [];
        
        if (definitions.rootElements) {
            for (const rootElement of definitions.rootElements) {
                if (isElementType(rootElement, "bpmn:Process")) {
                    processes.push(rootElement as Process);
                }
            }
        }

        return processes;
    }

    /**
     * Extract processes using regex (Phase 1 sync compatibility)
     */
    private extractProcessesSync(bpmnXml: string): any[] {
        const processes: any[] = [];
        
        // Find process elements using regex
        const processRegex = /<bpmn:process[^>]*id="([^"]*)"[^>]*>/g;
        let match;
        
        while ((match = processRegex.exec(bpmnXml)) !== null) {
            const processId = match[1];
            if (processId) {
                processes.push({
                    $type: "bpmn:Process",
                    id: processId,
                    flowElements: this.extractFlowElementsSync(bpmnXml, processId),
                });
            }
        }
        
        return processes;
    }

    /**
     * Extract flow elements using regex (Phase 1 sync compatibility)
     */
    private extractFlowElementsSync(bpmnXml: string, processId: string): any[] {
        const elements: any[] = [];
        
        // Extract various BPMN elements using regex (BPMN 2.0 uses lowercase element names)
        const elementPatterns = [
            { type: "bpmn:startEvent", regex: /<bpmn:startEvent[^>]*id="([^"]*)"[^>]*(?:name="([^"]*)"[^>]*)?/g },
            { type: "bpmn:endEvent", regex: /<bpmn:endEvent[^>]*id="([^"]*)"[^>]*(?:name="([^"]*)"[^>]*)?/g },
            { type: "bpmn:task", regex: /<bpmn:task[^>]*id="([^"]*)"[^>]*(?:name="([^"]*)"[^>]*)?/g },
            { type: "bpmn:userTask", regex: /<bpmn:userTask[^>]*id="([^"]*)"[^>]*(?:name="([^"]*)"[^>]*)?/g },
            { type: "bpmn:serviceTask", regex: /<bpmn:serviceTask[^>]*id="([^"]*)"[^>]*(?:name="([^"]*)"[^>]*)?/g },
            { type: "bpmn:callActivity", regex: /<bpmn:callActivity[^>]*id="([^"]*)"[^>]*(?:name="([^"]*)"[^>]*)?/g },
            { type: "bpmn:exclusiveGateway", regex: /<bpmn:exclusiveGateway[^>]*id="([^"]*)"[^>]*(?:name="([^"]*)"[^>]*)?/g },
            { type: "bpmn:parallelGateway", regex: /<bpmn:parallelGateway[^>]*id="([^"]*)"[^>]*(?:name="([^"]*)"[^>]*)?/g },
            { type: "bpmn:sequenceFlow", regex: /<bpmn:sequenceFlow[^>]*>/g }, // Handle separately due to attribute order
            { type: "bpmn:boundaryEvent", regex: /<bpmn:boundaryEvent[^>]*id="([^"]*)"[^>]*(?:name="([^"]*)"[^>]*)?[^>]*attachedToRef="([^"]*)"/g },
        ];

        for (const pattern of elementPatterns) {
            let match;
            while ((match = pattern.regex.exec(bpmnXml)) !== null) {
                if (pattern.type === "bpmn:sequenceFlow") {
                    // Handle sequence flow separately to extract attributes in any order
                    const flowElement = match[0];
                    const idMatch = flowElement.match(/id="([^"]*)"/);
                    const nameMatch = flowElement.match(/name="([^"]*)"/);
                    const sourceRefMatch = flowElement.match(/sourceRef="([^"]*)"/);
                    const targetRefMatch = flowElement.match(/targetRef="([^"]*)"/);
                    
                    if (idMatch && sourceRefMatch && targetRefMatch) {
                        elements.push({
                            $type: pattern.type,
                            id: idMatch[1],
                            name: nameMatch?.[1] || idMatch[1],
                            sourceRef: { id: sourceRefMatch[1] },
                            targetRef: { id: targetRefMatch[1] },
                        });
                    }
                } else {
                    const id = match[1];
                    const name = match[2];
                    
                    if (pattern.type === "bpmn:boundaryEvent") {
                        const attachedToRef = match[3];
                        elements.push({
                            $type: pattern.type,
                            id,
                            name: name || id,
                            attachedToRef: { id: attachedToRef },
                        });
                    } else {
                        elements.push({
                            $type: pattern.type,
                            id,
                            name: name || id,
                            incoming: this.extractIncomingFlowsSync(bpmnXml, id),
                            outgoing: this.extractOutgoingFlowsSync(bpmnXml, id),
                        });
                    }
                }
            }
        }

        return elements;
    }

    /**
     * Extract incoming flows using regex (Phase 1 sync compatibility)
     */
    private extractIncomingFlowsSync(bpmnXml: string, elementId: string): any[] {
        const flows: any[] = [];
        const flowRegex = new RegExp(`<bpmn:sequenceFlow[^>]*id="([^"]*)"[^>]*targetRef="${elementId}"`, "g");
        
        let match;
        while ((match = flowRegex.exec(bpmnXml)) !== null) {
            const flowId = match[1];
            if (flowId) {
                flows.push({ id: flowId });
            }
        }
        
        return flows;
    }

    /**
     * Extract outgoing flows using regex (Phase 1 sync compatibility)
     */
    private extractOutgoingFlowsSync(bpmnXml: string, elementId: string): any[] {
        const flows: any[] = [];
        const flowRegex = new RegExp(`<bpmn:sequenceFlow[^>]*id="([^"]*)"[^>]*sourceRef="${elementId}"`, "g");
        
        let match;
        while ((match = flowRegex.exec(bpmnXml)) !== null) {
            const flowId = match[1];
            if (flowId) {
                flows.push({ id: flowId });
            }
        }
        
        return flows;
    }

    /**
     * Build element maps synchronously (Phase 1 compatibility)
     */
    private buildElementMapsSync(bpmnXml: string): void {
        if (!this.definitions) return;

        // Extract processes from sync-parsed definitions
        this.processes = this.definitions.rootElements?.filter((el: any) => isElementType(el, "bpmn:Process")) || [];

        // Build element and flow maps
        for (const process of this.processes) {
            if (process.flowElements) {
                for (const element of process.flowElements) {
                    this.elements.set(element.id, element);

                    // Build sequence flow map
                    if (isElementType(element, "bpmn:SequenceFlow")) {
                        this.flows.set(element.id, element);
                    }

                    // Build boundary events map
                    if (isElementType(element, "bpmn:BoundaryEvent")) {
                        const attachedToRef = element.attachedToRef?.id;
                        if (attachedToRef) {
                            if (!this.boundaryEvents.has(attachedToRef)) {
                                this.boundaryEvents.set(attachedToRef, []);
                            }
                            this.boundaryEvents.get(attachedToRef)!.push(element);
                        }
                    }
                }
            }
        }
    }

    private getProcessById(processId: string): Process | null {
        return this.processes.find(p => p.id === processId) || null;
    }

    private evaluateConditionExpression(condition: string, context: Record<string, unknown>): boolean {
        // Simple condition evaluation - can be enhanced for complex expressions
        try {
            // Handle simple variable checks
            const varMatch = condition.match(/^([a-zA-Z_$][a-zA-Z0-9_$]*)$/);
            if (varMatch && varMatch[1]) {
                return !!context[varMatch[1]];
            }

            // Handle equality comparisons
            const equalityMatch = condition.match(/^([a-zA-Z_$][a-zA-Z0-9_$]*)\s*==\s*(.+)$/);
            if (equalityMatch) {
                const varName = equalityMatch[1];
                const expectedValue = equalityMatch[2].replace(/['"]/g, "");
                return context[varName] === expectedValue;
            }

            // Handle boolean expressions
            const boolMatch = condition.match(/^([a-zA-Z_$][a-zA-Z0-9_$]*)\s*(==|!=)\s*(true|false)$/);
            if (boolMatch) {
                const varName = boolMatch[1];
                const operator = boolMatch[2];
                const expectedBool = boolMatch[3] === "true";
                const actualValue = !!context[varName];
                return operator === "==" ? actualValue === expectedBool : actualValue !== expectedBool;
            }

            // If we can't evaluate, default to true to proceed
            return true;
        } catch {
            return true;
        }
    }

    private generateLocationId(
        nodeId: string, 
        locationType: LocationType, 
        options: {
            branchId?: string;
            subprocessId?: string;
            eventId?: string;
        },
    ): string {
        const parts = [nodeId, locationType];
        
        if (options.branchId) parts.push(`branch_${options.branchId}`);
        if (options.subprocessId) parts.push(`subprocess_${options.subprocessId}`);
        if (options.eventId) parts.push(`event_${options.eventId}`);
        
        return parts.join("_");
    }
}

/**
 * BPMN model factory function
 */
export async function createBpmnModel(bpmnXml: string): Promise<BpmnModel> {
    const model = new BpmnModel();
    await model.parseXml(bpmnXml);
    
    if (!model.isValid()) {
        throw new Error("Invalid BPMN model: No valid processes found");
    }
    
    return model;
}
