import BpmnModdle from "bpmn-moddle";
import { DbObject } from "../api/types.js";
import { PassableLogger } from "../consts/commonTypes.js";
import { uuid } from "../id/uuid.js";
import { LRUCache } from "../utils/lruCache.js";
import { RoutineGraphType } from "../utils/routineGraph.js";
import { GraphBpmnConfig, GraphConfig } from "./configs/routine.js";
import { BPMM_INSTANCES_CACHE_MAX_SIZE_BYTES, BPMN_DEFINITIONS_CACHE_LIMIT, BPMN_ELEMENT_CACHE_LIMIT, BPMN_ELEMENT_CACHE_MAX_SIZE_BYTES } from "./consts.js";
import { DecisionStrategy } from "./strategy.js";
import { DecisionOption, DeferredDecisionData, Id, Location, RunConfig, SubroutineContext } from "./types.js";

/**
 * Represents how to proceed after a node has been executed
 * TODO does not yet handle repeating nodes/boundary events
 */
type NavigationDecison = {
    /**
     * Any deferred decisions that need to be made.
     * 
     * This occurs when we have to choose between multiple next steps, 
     * and the `DecisionStrategy` decides requires a long or unknown amount of time 
     * to wait for a decision.
     * 
     * If any deferred decisions exist, we should not continue with the normal flow. 
     * Instead, we should store the deferred decisions in the run progress and put 
     * the branch into a waiting state.
     */
    deferredDecisions: DeferredDecisionData[];
    /**
     * True if we are waiting for an event to occur before continuing 
     * with the normal flow.
     * 
     * NOTE 1: Boundary events can still be triggered while in a waiting state,
     * so do not assume that if this is true, nextLocations will be empty.
     * 
     * NOTE 2: When this is false, every branch at this node with the same process ID 
     * should be marked as completed. This should close branches when passing through 
     * a join gateway.
     */
    isNodeStillActive: boolean;
    /**
     * The next locations to spawn branches for. All should be traversed in parallel.
     */
    nextLocations: Location[];
    /**
     * Locations which should no longer be available for spawning branches. 
     * This is useful for making sure non-repeating boundary events are not triggered
     * multiple times.
     */
    closedLocations: Location[];
    /**
     * True if we should mark the branch as failed.
     */
    triggerBranchFailure: boolean;
};

type GetAvailableStartLocationsParams<Config extends GraphConfig> = {
    /** Configuration for the graph being started - meaning the SUBROUTINE's config */
    config: Config;
    /** Unique key to identify the decision point */
    decisionKey: string;
    /** The decision strategy handler, which may be needed to choose which start nodes to run */
    decisionStrategy: DecisionStrategy;
    /** Logger to log errors */
    logger: PassableLogger;
    /** Additional subroutine information required to generate the locations */
    subroutine: DbObject<"RoutineVersion">;
    /** The current context (e.g. variables, state) for the subroutine, which may be needed to evaluate conditions */
    subcontext: SubroutineContext;
}

type GetTriggeredBoundaryEventsParams<Config extends GraphConfig> = {
    /** Configuration for the current graph - NOT the subroutine's config */
    config: Config;
    /** The current location (last item in the location stack) in the graph */
    location: Location;
    /** Logger to log errors */
    logger: PassableLogger;
    /** The current context (e.g. variables, state) for the subroutine, which may be needed to evaluate conditions */
    subcontext: SubroutineContext;
}

type GetAvailableNextLocationsParams<Config extends GraphConfig> = {
    /** Configuration for the current graph - NOT the subroutine's config */
    config: Config;
    /** Unique key to identify the decision point */
    decisionKey: string;
    /** The decision strategy handler, which may be needed to choose which start nodes to run */
    decisionStrategy: DecisionStrategy;
    /** The current location (last item in the location stack) in the graph */
    location: Location;
    /** Logger to log errors */
    logger: PassableLogger;
    /** The run config. Used to determine how strict to be with certain decisions */
    runConfig: RunConfig;
    /** The current context (e.g. variables, state) for the subroutine, which may be needed to evaluate conditions */
    subcontext: SubroutineContext;
}

type GetIONamesPassedIntoNodeParams<Config extends GraphConfig> = {
    /** The BPMN config to get the definitions for. */
    config: Config;
    /** Logger to log errors */
    logger: PassableLogger;
    /** The ID of the node to get the inputs and outputs for. */
    nodeId: string;
}

/**
 * Map of node input names to the name of the input/output linked to it.
 * 
 * Example:
 * <bpmn:callActivity id="callActivityA" name="Call A" calledElement="A">
 * //...
 * <bpmn:extensionElements>
 *     <vrooli:ioMapping>
 *         <vrooli:input name="inputA" fromContext="callActivityB.inputC" />
 *         <vrooli:input name="inputB" />
 *         <vrooli:input name="inputC" fromContext="root.inputD" />
 *         <vrooli:output name="outputX" />
 *         <vrooli:output name="outputY" toRootContext="outputZ" />
 *     </vrooli:ioMapping>
 * </bpmn:extensionElements>
 * 
 * Would result in:
 * {
 *     inputA: "callActivityB.inputC",
 *     // inputB is not linked to anything, so it will be ignored
 *     inputC: "root.inputD",
 *     // outputs are ignored
 * }
 */
type GetIONamesPassedIntoNodeResult = {
    [inputName: string]: string;
}

/**
 * Interface for navigating a routine graph step-by-step. Note that we don't need methods 
 * 
 * NOTE: We don't need methods for finding the previous step, as the state machine 
 * handles popping the location stack until a new next step is found.
 * 
 * Should be implemented by each supported workflow type (e.g. BPMN, DMN)
 */
export interface IRoutineStepNavigator {
    /**
     * Whether this graph type supports parallel execution (branches).
     */
    supportsParallelExecution: boolean;
    /**
     * Return the available start location for a subroutine.
     * 
     * By available locations, we mean locations for nodes that can be run in parallel. 
     * In BPMN, this typically means one start node per process ID.
     * 
     * NOTE: This should be the first function called when starting a new subroutine.
     * 
     * @returns Navigation decisions for starting the subroutine
     */
    getAvailableStartLocations<Config extends GraphConfig>(params: GetAvailableStartLocationsParams<Config>): Promise<NavigationDecison>;
    /**
     * Returns locations for boundary events that have been triggered by the current node, 
     * such as timer events, message events, etc.
     * 
     * NOTE: This should be called before the node is executed, as well as during execution 
     * at a regular interval (for timers) and after receiving an event (for message events).
     * 
     * @returns The boundary events for the node, if any
     */
    getTriggeredBoundaryEvents<Config extends GraphConfig>(params: GetTriggeredBoundaryEventsParams<Config>): Promise<NavigationDecison>;
    /**
      * Return the locations for the next available nodes, given the current location.
      *
      * "Available" nodes are those reachable via outgoing sequence flows whose conditions (if any)
      * evaluate to true. In BPMN terms, this means:
      *
      * 1. **Condition Expressions**:
      *    - Outgoing sequence flows with no conditionExpression are treated as unconditional
      *      and are always eligible.
      *    - Outgoing sequence flows with a conditionExpression must evaluate to true.
      *
      * 2. **Boundary Events**:
      *    - If an interrupting boundary event on the current activity is triggered,
      *      it interrupts the normal flow. Therefore, outgoing flows from the underlying
      *      activity are no longer eligible.
      *    - Non-interrupting boundary events do not preclude the normal flow; they may create
      *      parallel paths if triggered, but the original activity flow can continue.
      *    - If a boundary event exists on the next node, we don't worry about it here. It 
      *      will be handled before and while the next node is executed.
      *
      * 3. **Gateways**:
      *    - **Exclusive (XOR) Gateway**: Select exactly one outgoing flow that satisfies
      *      its condition. If more than one condition is true, the first (in the defined order)
      *      is typically chosen. If none are true, but a default flow is defined, use the
      *      default flow.
      *    - **Parallel (AND) Gateway**: All outgoing flows are taken simultaneously (i.e.,
      *      they are all considered "available"), assuming the gateway's entry criteria have
      *      been met.
      *    - **Inclusive (OR) Gateway**: Any outgoing flow whose condition evaluates to true
      *      is taken. If no condition is met, but a default flow exists, use the default flow.
      *    - **Event-Based Gateway**: Waits for one of several possible events to occur.
      *      Only the flow associated with the event that actually happens is taken.
      *
      * NOTE: This does not traverse up the stack to find the next location. 
      * It simply checks if there are any next nodes in the current graph.
      * 
      * @returns The next location(s), or none if at the end of graph
      */
    getAvailableNextLocations<Config extends GraphConfig>(params: GetAvailableNextLocationsParams<Config>): Promise<NavigationDecison>;
    /**
     * Get the I/O names being passed into a node in the graph.
     * 
     * NOTE: This returns names as they are defined in the graph - not the names of the subroutine inputs and outputs. 
     * This should match how inputs and outputs are stored in the SubroutineContext.
     * 
     * @returns A list of input and output names being passed into the node, as they are defined in the graph.
     */
    getIONamesPassedIntoNode<Config extends GraphConfig>(params: GetIONamesPassedIntoNodeParams<Config>): Promise<GetIONamesPassedIntoNodeResult>;
}

export type NavigatorRegistry = {
    [graphType in RoutineGraphType]: IRoutineStepNavigator;
}

/**
 * A simple factory that returns the right navigator given a routine type.
 */
export class NavigatorFactory {
    constructor(private readonly registry: NavigatorRegistry) { }

    public getNavigator(graphType: RoutineGraphType): IRoutineStepNavigator {
        const nav = this.registry[graphType];
        if (!nav) throw new Error(`No navigator found for ${graphType}`);
        return nav;
    }
}

// Define the interface for a single input element.
export interface VrooliIOInput {
    $type: "vrooli:input";
    name: string;
    fromContext?: string;
}

// Define the interface for a single output element.
export interface VrooliIOOutput {
    $type: "vrooli:output";
    name: string;
    toRootContext?: string;
}

// Define the interface for the ioMapping element.
export interface VrooliIOMapping {
    $type: "vrooli:ioMapping";
    inputs?: VrooliIOInput[];
    outputs?: VrooliIOOutput[];
}

enum BpmnElementName {
    Definitions = "Definitions",
    Process = "Process",
    SubProcess = "SubProcess",
    GatewayComplex = "ComplexGateway",
    GatewayEventBased = "EventBasedGateway",
    GatewayExclusive = "ExclusiveGateway",
    GatewayInclusive = "InclusiveGateway",
    GatewayParallel = "ParallelGateway",
    SequenceFlow = "SequenceFlow",
    Expression = "FormalExpression",
    StartEvent = "StartEvent",
    EndEvent = "EndEvent",
    IntermediateCatchEvent = "IntermediateCatchEvent",
    IntermediateThrowEvent = "IntermediateThrowEvent",
    BoundaryEvent = "BoundaryEvent",
    Event = "Event",
    Task = "Task",
    UserTask = "UserTask",
    ServiceTask = "ServiceTask",
    CallActivity = "CallActivity",
}

type BpmnElement = {
    [BpmnElementName.Definitions]: BpmnModdle.Definitions;
    [BpmnElementName.Process]: BpmnModdle.FlowElementsContainer;
    [BpmnElementName.SubProcess]: BpmnModdle.SubProcess;
    [BpmnElementName.GatewayComplex]: BpmnModdle.ComplexGateway;
    [BpmnElementName.GatewayEventBased]: BpmnModdle.EventBasedGateway;
    [BpmnElementName.GatewayExclusive]: BpmnModdle.ExclusiveGateway;
    [BpmnElementName.GatewayInclusive]: BpmnModdle.InclusiveGateway;
    [BpmnElementName.GatewayParallel]: BpmnModdle.ParallelGateway;
    [BpmnElementName.SequenceFlow]: BpmnModdle.SequenceFlow;
    [BpmnElementName.Expression]: BpmnModdle.FormalExpression;
    [BpmnElementName.StartEvent]: BpmnModdle.StartEvent;
    [BpmnElementName.EndEvent]: BpmnModdle.EndEvent;
    [BpmnElementName.IntermediateCatchEvent]: BpmnModdle.IntermediateCatchEvent;
    [BpmnElementName.IntermediateThrowEvent]: BpmnModdle.IntermediateThrowEvent;
    [BpmnElementName.BoundaryEvent]: BpmnModdle.BoundaryEvent;
    [BpmnElementName.Event]: BpmnModdle.Event;
    [BpmnElementName.Task]: BpmnModdle.Task;
    [BpmnElementName.UserTask]: BpmnModdle.UserTask;
    [BpmnElementName.ServiceTask]: BpmnModdle.ServiceTask;
    [BpmnElementName.CallActivity]: BpmnModdle.CallActivity;
}

type BpmnGraphData = {
    config: GraphBpmnConfig;
    location: Location;
    logger: PassableLogger;
    subcontext: SubroutineContext;
}

export class BpmnNavigator implements IRoutineStepNavigator {
    public supportsParallelExecution = true; // Running sequences in parallel is common in BPMN
    private supportedBpmnVersions = new Set(["2.0"]);
    private namespace = "bpmn"; // The namespace for processing logic

    private bpmnModdle = new BpmnModdle();

    // Store parsed BPMN definitions to avoid re-parsing the same XML
    private definitionsCache = new LRUCache<string, BpmnModdle.Definitions>(BPMN_DEFINITIONS_CACHE_LIMIT, BPMM_INSTANCES_CACHE_MAX_SIZE_BYTES);
    // Store mappings of definitionId to { [elementId: string]: BpmnElement }
    private elementCache = new LRUCache<string, Map<string, BpmnModdle.BaseElement>>(BPMN_ELEMENT_CACHE_LIMIT, BPMN_ELEMENT_CACHE_MAX_SIZE_BYTES);

    private END_STATE: NavigationDecison = {
        deferredDecisions: [],
        isNodeStillActive: false,
        nextLocations: [],
        closedLocations: [],
        triggerBranchFailure: false,
    };
    private NOOP_STATE: NavigationDecison = {
        deferredDecisions: [],
        isNodeStillActive: true,
        nextLocations: [],
        closedLocations: [],
        triggerBranchFailure: false,
    };

    /**
     * Checks if the provided Graph config is supported by this navigator.
     * 
     * @param config The Graph config to check
     * @param logger Logger to log errors
     * @returns True if the config is supported, false otherwise
     */
    private isSupportedBpmnVersion(config: GraphBpmnConfig, logger: PassableLogger): boolean {
        if (!this.supportedBpmnVersions.has(config.__type)) {
            logger.error(`Unsupported BPMN version: ${config.__type}`);
            return false;
        }
        return true;
    }

    /**
     * Helper method to retrieve BPMN definitions, using the cache if available.
     * If not cached, it will parse the XML and store the result.
     * 
     * @param config The BPMN config to get the definitions for.
     * @returns The BPMN definitions.
     */
    private async getDefinitions(config: GraphBpmnConfig): Promise<BpmnModdle.Definitions> {
        const xmlData = config.schema.data;

        // Check cache
        const definitions = this.definitionsCache.get(xmlData);
        if (definitions) {
            return definitions;
        }

        // Parse asynchronously
        const [error, defs] = await new Promise<[Error, BpmnModdle.Definitions]>((resolve) => {
            this.bpmnModdle.fromXML(xmlData, (err: Error, parsedDefs: BpmnModdle.Definitions) => {
                resolve([err, parsedDefs]);
            });
        });

        if (error) {
            throw new Error("Error parsing BPMN XML: " + error.message);
        }

        // Assign a unique ID if missing
        defs.id = defs.id || `definitions_${uuid()}`;

        // Cache it
        this.definitionsCache.set(xmlData, defs);
        this.storeElements(defs);

        return defs;
    }

    /**
     * Helper method to store all BPMN elements in the cache for quick lookup.
     * 
     * @param definitions The BPMN definitions to store
     */
    private storeElements(definitions: BpmnModdle.Definitions): void {
        // If definitions have no ID, nothing to store or key with
        if (!definitions.id) {
            return;
        }

        // This map will store elementId -> element
        const elementMap = new Map<string, BpmnModdle.BaseElement>();

        // Use a Set to avoid re-visiting the same object (in case of cycles/references)
        const visited = new Set<BpmnModdle.BaseElement>();

        /**
         * Recursively explore an element and its children. 
         * Any property that looks like a BPMN element or array of elements is visited.
         */
        function traverse(element: BpmnModdle.BaseElement) {
            if (!element || visited.has(element)) {
                return;
            }
            visited.add(element);

            // If the element has an ID, store it
            if (element.id) {
                elementMap.set(element.id, element);
            }

            // Go over each property of the element
            for (const prop in element) {
                if (!Object.prototype.hasOwnProperty.call(element, prop)) {
                    continue;
                }

                const value = (element as any)[prop];

                // If it's an array, check each item
                if (Array.isArray(value)) {
                    value.forEach((child) => {
                        // If it's an object with a $type, we consider it a BPMN element
                        if (child && typeof child === "object" && child.$type) {
                            traverse(child);
                        }
                    });
                }
                // If it's a nested object with a $type, traverse it
                else if (value && typeof value === "object" && value.$type) {
                    traverse(value);
                }
            }
        }

        // Start recursion from the top-level definitions object
        traverse(definitions);

        // Store the completed map in the LRU cache
        this.elementCache.set(definitions.id, elementMap);
    }

    public async getAvailableNextLocations<Config extends GraphConfig>(params: GetAvailableNextLocationsParams<Config>): Promise<NavigationDecison> {
        const { config, decisionKey, decisionStrategy, location, logger, runConfig, subcontext } = params as unknown as GetAvailableNextLocationsParams<GraphBpmnConfig>;

        // Validate the BPMN version
        if (!this.isSupportedBpmnVersion(config, logger)) {
            return this.END_STATE;
        }

        // Load the BPMN definitions
        const definitions = await this.getDefinitions(config);

        // Find the current BPMN element by ID
        const element = this.findElementById(definitions, location.locationId, logger);
        if (!element) {
            return this.END_STATE;
        }

        // If the element is an end event, there are no next locations
        if (this.isElementType(element, BpmnElementName.EndEvent)) {
            return this.END_STATE;
        }

        // Initialize next locations
        const nextLocations: Location[] = [];
        // Group the data needed for processing
        const data: BpmnGraphData = { config, location, logger, subcontext };

        // Check for boundary events on the current node
        const triggeredBoundaryEvents = this.getBoundaryEvents(element, definitions, data).triggered;
        // For every boundary event that is triggered, add it to the next locations
        let hasInterruptingBoundaryEventTriggered = false;
        for (const boundaryEvent of triggeredBoundaryEvents) {
            // Add the boundary event itself. On the next iteration, we'll move to its outgoing nodes
            nextLocations.push(this.nodeToLocation(boundaryEvent, data));
            // If boundary event is interrupting, make sure we don't continue with the normal flow
            if (boundaryEvent.cancelActivity) {
                hasInterruptingBoundaryEventTriggered = true;
                break;
            }
        }

        // If an interrupting boundary event was triggered, we don't continue with the normal flow
        if (hasInterruptingBoundaryEventTriggered) {
            return {
                deferredDecisions: [],
                isNodeStillActive: false, // Stop the normal flow
                nextLocations, // Include all triggered boundary events
                closedLocations: [], // Node is no longer active, so this doesn't matter
                triggerBranchFailure: false,
            };
        }

        // If the current node is an event, check if the event condition is met
        // NOTE: Do not include boundary events here. If you're on a boundary event, 
        // that means the event which triggered it has already occurred.
        const isNonBoundaryEvent = this.isEventNode(element) && !this.isBoundaryEvent(element);
        if (isNonBoundaryEvent) {
            // Check if the event condition is met
            const eventHasTriggered = this.isEventTriggered(element, data);
            // If not, we'll have to wait
            if (!eventHasTriggered) {
                return {
                    deferredDecisions: [],
                    isNodeStillActive: true, // Continue waiting for the event
                    nextLocations, // Include triggered boundary events in the next locations
                    closedLocations: nextLocations, // Make sure triggered boundary events are not triggered again
                    triggerBranchFailure: false,
                };
            }
        }

        // Collect all outgoing nodes, regardless of conditions
        const outgoingNodes = this.getAllOutgoingNodes(element);

        // If there are no outgoing nodes at all, treat it like an end event. 
        // This should only happen with malformed BPMN diagrams.
        if (outgoingNodes.length === 0) {
            logger.error(`No outgoing nodes found for element ${element.id}. Treating as end event.`);
            return this.END_STATE;
        }

        // Filter out nodes where the incoming sequence flow conditions are not met
        let availableOutgoingNodes = outgoingNodes.filter((node) => this.isSequenceFlowConditionMet(node, element, data));

        // If the current node is a gateway, further filter which nodes to take. 
        // Some gateways take all outgoing nodes, while others take a subset
        if (this.isGatewayNode(element)) {
            // Check if the gateway condition is met (e.g. many gateways require all incoming flows to reach the gateway before proceeding)
            //TODO
            const outgoingNodesResult = await this.pickOutgoingNodesForGateway(element, availableOutgoingNodes, decisionKey, decisionStrategy, data);
            availableOutgoingNodes = outgoingNodesResult.availableOutgoingNodes;
            // If there are deferred decisions, add them to the navigation decision
            if (outgoingNodesResult.deferredDecisions.length > 0) {
                return {
                    ...this.END_STATE,
                    deferredDecisions: outgoingNodesResult.deferredDecisions,
                };
            }
            // If there are no available outgoing nodes, use configuration to determine what to do
            if (availableOutgoingNodes.length === 0) {
                logger.error(`No outgoing nodes found for gateway ${element.id}.`);
                // Continue condition
                if (runConfig.onGatewayForkFailure === "Continue") {
                    return this.END_STATE;
                }
                // Waiting condition
                if (runConfig.onGatewayForkFailure === "Wait") {
                    return {
                        deferredDecisions: [],
                        isNodeStillActive: true, // Continue waiting for the event
                        nextLocations, // Include triggered boundary events in the next locations
                        closedLocations: nextLocations, // Make sure triggered boundary events are not triggered again
                        triggerBranchFailure: false,
                    };
                }
                // Fail condition 
                else {
                    return {
                        ...this.END_STATE,
                        triggerBranchFailure: true,
                    };
                }
            }
        }

        // If there are no available outgoing nodes, use configuration to determine what to do
        if (availableOutgoingNodes.length === 0) {
            logger.error(`No outgoing nodes found for element ${element.id}.`);
            // Continue condition
            if (runConfig.onNormalNodeFailure === "Continue") {
                return this.END_STATE;
            }
            // Waiting condition
            if (runConfig.onNormalNodeFailure === "Wait") {
                return {
                    deferredDecisions: [],
                    isNodeStillActive: true, // Continue waiting for the event
                    nextLocations, // Include triggered boundary events in the next locations
                    closedLocations: nextLocations, // Make sure triggered boundary events are not triggered again
                    triggerBranchFailure: false,
                };
            }
            // Fail condition 
            else {
                return {
                    ...this.END_STATE,
                    triggerBranchFailure: true,
                };
            }
        }

        // Convert the available outgoing nodes to locations
        for (const node of availableOutgoingNodes) {
            nextLocations.push(this.nodeToLocation(node, data));
        }

        // Return the available outgoing nodes
        return {
            deferredDecisions: [],
            isNodeStillActive: false, // Normal flow continues
            nextLocations,
            closedLocations: [], // Node is no longer active, so this doesn't matter
            triggerBranchFailure: false,
        };
    }

    public async getAvailableStartLocations<Config extends GraphConfig>(params: GetAvailableStartLocationsParams<Config>): Promise<NavigationDecison> {
        const { config, decisionKey, decisionStrategy, logger, subroutine, subcontext } = params as unknown as GetAvailableStartLocationsParams<GraphBpmnConfig>;

        // Validate the BPMN version
        if (!this.isSupportedBpmnVersion(config, logger)) {
            return this.END_STATE;
        }

        // Load the BPMN definitions
        const definitions = await this.getDefinitions(config);

        // Find all start events, organized by process ID
        const startEventsByProcess = this.findStartEventsByProcess(definitions);

        // Choose which start events to trigger
        const startEventsToTriggerResult = await this.getStartEventsToTrigger(startEventsByProcess, decisionKey, decisionStrategy, subcontext);
        const startEvents = startEventsToTriggerResult.startEvents;
        const deferredDecisions = startEventsToTriggerResult.deferredDecisions;

        // Convert start events to locations
        const data: BpmnGraphData = {
            config,
            location: {
                __typename: subroutine.__typename,
                objectId: subroutine.id,
                locationId: subroutine.id, // Can be anything, as it will be replaced by the start event ID
                subroutineId: null,
            },
            logger,
            subcontext,
        };
        const nextLocations = startEvents.map((startEvent) => {
            return this.nodeToLocation(startEvent, data);
        });

        return {
            deferredDecisions,
            isNodeStillActive: false, // Start events are not waiting for anything
            nextLocations,
            closedLocations: [], // No waiting nodes, so this doesn't matter
            triggerBranchFailure: false,
        };
    }

    public async getTriggeredBoundaryEvents<Config extends GraphConfig>(params: GetTriggeredBoundaryEventsParams<Config>): Promise<NavigationDecison> {
        const { config, location, logger, subcontext } = params as unknown as GetTriggeredBoundaryEventsParams<GraphBpmnConfig>;

        // Validate the BPMN version
        if (!this.isSupportedBpmnVersion(config, logger)) {
            return this.NOOP_STATE;
        }

        // Load the BPMN definitions
        const definitions = await this.getDefinitions(config);

        // Find the current BPMN element by ID
        const element = this.findElementById(definitions, location.locationId, logger);
        if (!element) {
            return this.NOOP_STATE;
        }

        const data: BpmnGraphData = { config, location, logger, subcontext };

        // Find triggered boundary events
        const triggeredBoundaryEvents = this.getBoundaryEvents(
            element,
            definitions,
            data,
        ).triggered;

        // If none are triggered, do nothing
        if (triggeredBoundaryEvents.length === 0) {
            return this.NOOP_STATE;
        }

        // Convert triggered boundary events into next locations
        const nextLocations = triggeredBoundaryEvents.map((boundaryEvent) =>
            this.nodeToLocation(boundaryEvent, data),
        );

        // If any boundary event is interrupting (cancelActivity = true), stop the node
        const hasInterrupting = triggeredBoundaryEvents.some(be => be.cancelActivity === true);

        // Treat all triggered boundary events as non-repeating so they won't be triggered again. Hence, we mark them as completed.
        // TODO revisit in the future. Make sure getAvailableNextLocations implements this too
        const closedLocations = nextLocations;

        // Build and return the navigation decision
        return {
            deferredDecisions: [],
            isNodeStillActive: !hasInterrupting, // False if an interrupting event fired, true otherwise
            nextLocations,
            closedLocations,
            triggerBranchFailure: false,
        };
    }

    /**
     * Finds all StartEvent elements across the BPMN diagram’s processes.
     * @param definitions The BPMN definitions to search
     * @returns StartEvent elements, grouped by process ID
     */
    private findStartEventsByProcess(definitions: BpmnModdle.Definitions): Map<string, BpmnModdle.StartEvent[]> {
        const result = new Map<string, BpmnModdle.StartEvent[]>();

        for (const root of definitions.rootElements || []) {
            // Looking for processes
            if (this.isElementType(root, BpmnElementName.Process)) {
                const process = root as BpmnModdle.FlowElementsContainer;
                const processId = process.id;

                // Initialize the process ID in the result map if not already present
                if (!result.has(processId)) {
                    result.set(processId, []);
                }

                // If there are flowElements, look for StartEvents
                const flowElements = process.flowElements || [];
                for (const fe of flowElements) {
                    if (this.isElementType(fe, BpmnElementName.StartEvent)) {
                        result.get(processId)?.push(fe as BpmnModdle.StartEvent);
                    }
                }
            }
        }
        return result;
    }

    /**
     * Determines which start events should be triggered based on the current context.
     * 
     * Rules:
     * 1. Only one start event should be triggered from each process. This is because a process 
     * typically represents a single participant (e.g. a user or bot with a role)
     * 2. If a process has multiple start events, the decision strategy class should decide which one to trigger.
     * 
     * @param startEvents StartEvent elements, grouped by process ID
     * @param decisionKey The unique key to identify the decision point
     * @param decisionStrategy The decision strategy to use
     * @param subcontext The current context for the subroutine
     * @returns An object containing a list of start events to trigger and any deferred decisions
     */
    private async getStartEventsToTrigger(
        startEvents: Map<string, BpmnModdle.StartEvent[]>,
        decisionKey: string,
        decisionStrategy: DecisionStrategy,
        subcontext: SubroutineContext,
    ): Promise<{
        startEvents: BpmnModdle.StartEvent[];
        deferredDecisions: DeferredDecisionData[];
    }> {
        for (const [, processStartEvents] of startEvents) {
            // Skip processes with no start events
            if (processStartEvents.length === 0) {
                continue;
            }
            // If there is only one start event, add it to the result
            if (processStartEvents.length === 1) {
                // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
                return { startEvents: [processStartEvents[0]!], deferredDecisions: [] };
            }
            // If there are multiple start events, use the decision strategy to pick one
            const options: DecisionOption[] = processStartEvents.map((startEvent) => {
                return {
                    nodeId: startEvent.id,
                    nodeLabel: startEvent.name || startEvent.id,
                };
            });
            const choice = await decisionStrategy.chooseOne(options, decisionKey, subcontext);
            if (choice.__type === "Waiting") {
                return { startEvents: [], deferredDecisions: [choice] };
            }
            const chosenStartEvent = processStartEvents.find((se) => se.id === choice.result);
            if (chosenStartEvent) {
                return { startEvents: [chosenStartEvent], deferredDecisions: [] };
            }
        }

        return { startEvents: [], deferredDecisions: [] };
    }

    /**
     * Find an element by ID
     */
    private findElementById(
        definitions: BpmnModdle.Definitions,
        elementId: string,
        logger: PassableLogger,
    ): BpmnModdle.BaseElement | null {
        const elementMap = this.elementCache.get(definitions.id);
        if (!elementMap) {
            logger.error(`Could not find BPMN element cache for definitions ID ${definitions.id}`);
            return null;
        }
        const element = elementMap.get(elementId);
        if (!element) {
            logger.error(`Could not find BPMN element with ID ${elementId}`);
            return null;
        }
        return element;
    }

    /**
     * Prefixes a string with the BPMN namespace, if it doesn't already have it.
     * 
     * @param name The name to prefix
     * @returns The prefixed name
     */
    private withNamespace(name: string): string {
        return name.startsWith(this.namespace + ":") ? name : this.namespace + ":" + name;
    }

    /**
     * Type guard to check if a given BaseElement is one of the specified types.
     * This dynamically checks for type-specific properties.
     * 
     * @param element - The BaseElement to check.
     * @param type - The type name to validate against.
     * @returns True if the element matches the specified type, false otherwise.
     */
    private isElementType<T extends keyof BpmnElement>(
        element: BpmnModdle.BaseElement,
        type: T,
    ): element is BpmnElement[T] {
        const targetType = this.withNamespace(type);
        const elementName = this.withNamespace(element.$type);
        return elementName === targetType;
    }

    /**
     * Type guard to check if a given BaseElement is a FlowNode.
     * This verifies that the element has incoming and outgoing arrays.
     * 
     * @param element The BaseElement to check.
     * @returns True if the element has incoming and outgoing arrays, false otherwise.
     */
    private isFlowNode(element: BpmnModdle.BaseElement): element is BpmnModdle.FlowNode {
        return "incoming" in element && Array.isArray((element as BpmnModdle.FlowNode).incoming)
            && "outgoing" in element && Array.isArray((element as BpmnModdle.FlowNode).outgoing);
    }

    /**
     * Type guard to check if a given BaseElement is a Gateway.
     * This verifies that the element has a gatewayDirection property.
     * 
     * @param element The BaseElement to check.
     * @returns True if the element has a gatewayDirection property, false otherwise.
     */
    private isGatewayNode(element: BpmnModdle.BaseElement): element is BpmnModdle.Gateway {
        return this.isFlowNode(element) && "gatewayDirection" in element;
    }

    /**
     * Type guard to check if a given BaseElement is an Event
     * This verifies that the element is a FlowNode and has the `properties` property.
     * 
     * @param element The BaseElement to check.
     * @returns True if the element is an event node, false otherwise.
     */
    private isEventNode(element: BpmnModdle.BaseElement): element is BpmnModdle.Event {
        return this.isFlowNode(element) && "eventDefinitions" in element;
    }

    /**
     * Type guard to check if a given BaseElement is a BoundaryEvent.
     * This verifies that the element is an Event and has the `cancelActivity` property.
     */
    private isBoundaryEvent(element: BpmnModdle.BaseElement): element is BpmnModdle.BoundaryEvent {
        return this.isEventNode(element) && "cancelActivity" in element;
    }

    /**
     * Type guard to check if a given BaseElement extensionElement value is a VrooliIOMapping.
     * 
     * @param ext The BaseElement to check.
     * @returns True if the element is a VrooliIOMapping, false otherwise.
     */
    private isVrooliIOMapping(ext: unknown): ext is VrooliIOMapping {
        return (
            typeof ext === "object" &&
            ext !== null &&
            "$type" in ext &&
            (ext.$type as string) === "vrooli:ioMapping" &&
            "inputs" in ext && Array.isArray(ext.inputs) &&
            "outputs" in ext && Array.isArray(ext.outputs)
        );
    }

    /**
     * Converts a BPMN element to a location.
     * 
     * @param element The BPMN elment to convert.
     * @param data The BPMN graph data.
     * @returns The location object.
     */
    private nodeToLocation(
        node: BpmnModdle.BaseElement,
        data: BpmnGraphData,
    ): Location {
        const { config, location: originalLocation } = data;
        // If the target is a call activity, check if it references a subroutine
        let subroutineId: Id | null = null;
        if (this.isElementType(node, BpmnElementName.CallActivity)) {
            subroutineId = config.schema.activityMap[node.id]?.subroutineId || null;
        }
        // Return the new location
        return {
            // __typename and objectId stay the same, as they reference the current routine
            __typename: originalLocation.__typename,
            objectId: originalLocation.objectId,
            // locationId changes, as it references the node we're going to next
            locationId: node.id,
            // subroutineId is set based on if the next node is tied to a subroutine
            subroutineId,
        };
    }

    /**
     * Gathers all outgoing nodes from a given element, regardless of 
     * if they are available or not.
     * 
     * @param element The BPMN element to get outgoing nodes from.
     * @returns All outgoing nodes from the element, regardless of availability.
     */
    private getAllOutgoingNodes(
        element: BpmnModdle.BaseElement,
    ): BpmnModdle.FlowNode[] {
        // Initialize result
        const nextNodes: BpmnModdle.FlowNode[] = [];
        // Ignore nodes that don't support outgoing flows
        if (!this.isFlowNode(element)) {
            return nextNodes;
        }
        // Loop through all outgoing flows
        for (const flow of element.outgoing) {
            // Make sure flow has a valid target reference
            const targetNode = flow.targetRef;
            if (!targetNode || !targetNode.id) {
                continue;
            }
            // Add the target node to the result
            nextNodes.push(targetNode);
        }
        return nextNodes;
    }

    /**
     * Gathers boundary events for the current element, distinguishing
     * which ones are triggered vs. non-triggered.
     * 
     * A boundary event is "attached" to an activity (task, sub-process, etc.) 
     * via `boundaryEvent.attachedToRef`. If that `attachedToRef` matches
     * the current element, the boundary event applies here.
     * 
     * For each boundary event, we determine whether it has been triggered 
     * (e.g., a timer elapsed, a message arrived, a signal was sent).
     * 
     * @param element The current BPMN element (usually an activity) whose boundary events we're checking.
     * @param definitions The BPMN definitions to search.
     * @param data The BPMN graph data containing the config, subcontext, etc.
     * @returns An object with `triggered` and `unTriggered` arrays of boundary events.
     */
    private getBoundaryEvents(
        element: BpmnModdle.BaseElement,
        definitions: BpmnModdle.Definitions,
        data: BpmnGraphData,
    ): { triggered: BpmnModdle.BoundaryEvent[]; unTriggered: BpmnModdle.BoundaryEvent[] } {
        const { subcontext } = data;

        // Collect all boundary events attached to the current element
        const elementBoundaryEvents = this.getAllBoundaryEvents(element, definitions);

        // Initialize arrays for triggered and untriggered boundary events
        const triggered: BpmnModdle.BoundaryEvent[] = [];
        const unTriggered: BpmnModdle.BoundaryEvent[] = [];

        // Loop through each boundary event and place it in the appropriate array
        for (const boundaryEvent of elementBoundaryEvents) {
            if (this.isBoundaryEventTriggered(boundaryEvent, subcontext)) {
                triggered.push(boundaryEvent);
            } else {
                unTriggered.push(boundaryEvent);
            }
        }

        // Return the result
        return { triggered, unTriggered };
    }

    /**
     * Gathers all boundary events for a given node, regardless of type and whether they are triggered.
     * 
     * NOTE: Boundary events are not directly linked to nodes. Instead, they reference them via their 
     * `attachedToRef` property. This means that we have to go through the full graph definitions to find them.
     * 
     * @param node The BPMN node to get boundary events for.
     * @param definitions The BPMN definitions to search.
     * @returns The boundary events for the node.
     */
    private getAllBoundaryEvents(
        node: BpmnModdle.BaseElement,
        definitions: BpmnModdle.Definitions,
    ): BpmnModdle.BoundaryEvent[] {
        // Initialize result
        const boundaryEvents: BpmnModdle.BoundaryEvent[] = [];

        // If there's no ID on the definitions, we can't look up anything in the cache
        if (!definitions.id) {
            return boundaryEvents;
        }

        // Get the cached map of elementId -> element
        const elementMap = this.elementCache.get(definitions.id);
        if (!elementMap) {
            return boundaryEvents;
        }

        // Filter for boundary events that match the attachedToRef
        for (const element of elementMap.values()) {
            if (
                this.isElementType(element, BpmnElementName.BoundaryEvent) &&
                element.attachedToRef?.id === node.id
            ) {
                boundaryEvents.push(element as BpmnModdle.BoundaryEvent);
            }
        }

        return boundaryEvents;
    }

    /**
     * Determines if a boundary event has been triggered, based on your run context
     * and the event definition. This is a placeholder for your actual logic:
     * 
     * - Timer boundary events might check timestamps or durations in `context`.
     * - Message boundary events might check if a certain message was received.
     * - Error boundary events might check if an error was thrown in the sub-process, etc.
     * 
     * @param boundaryEvent The boundary event to evaluate.
     * @param subcontext The current subroutine context, which might store event signals, timestamps, etc.
     * @returns True if the boundary event has been triggered, false otherwise.
     */
    private isBoundaryEventTriggered(boundaryEvent: BpmnModdle.BoundaryEvent, subcontext: SubroutineContext): boolean {
        // Replace with real checks. This is a stub.
        // For example, if boundaryEvent.eventDefinitions[0] is a TimerEventDefinition, 
        // check if the timer has elapsed in the subcontext, etc.
        //TODO
        return false;
    }

    /**
     * Determines if a regular event node (e.g. intermediate event, start event)
     * has been triggered. This typically applies to catching events like:
     * - IntermediateCatchEvent (timer, message, signal, etc.)
     * - StartEvent with some event definitions
     * - Possibly custom events.
     * 
     * @param element The BPMN event element to check (not a boundary event).
     * @param data The BPMN graph data for config, subcontext, etc.
     * @returns True if the event has triggered, false otherwise.
     */
    private isEventTriggered(element: BpmnModdle.Event, data: BpmnGraphData): boolean {
        const { subcontext } = data;

        // You might check eventDefinitions here:
        // e.g. if element.eventDefinitions[0] is "bpmn:TimerEventDefinition"
        // or "bpmn:MessageEventDefinition". Then see if subcontext indicates it's been triggered.
        // For now, this is a simple placeholder:
        //TODO
        return false;
    }

    /**
     * Checks if a sequence flow's conditionExpression (if present) evaluates to true
     * in the current subroutine context. If there's no conditionExpression, it is treated
     * as an unconditional flow.
     * 
     * Note: BPMN's condition expression might be in various forms: an expression
     * language snippet, a script, etc. You may need your own expression parser
     * or a universal expression library. This method uses a hypothetical
     * `evaluateExpression` function for demonstration.
     * 
     * @param targetElement The target node of the sequence flow (i.e. where we're going to).
     * @param sourceElement The source node of the sequence flow (i.e. where we're coming from).
     * @param data The BPMN graph data containing config, subcontext, etc.
     * @returns True if the flow is either unconditional or the condition is met, false otherwise.
     */
    private isSequenceFlowConditionMet(
        targetElement: BpmnModdle.BaseElement,
        sourceElement: BpmnModdle.BaseElement,
        data: BpmnGraphData,
    ): boolean {
        // Get the sequence flow connecting the source and target elements
        const sequenceFlow = this.getSequenceFlowBetween(sourceElement, targetElement);
        if (!sequenceFlow) {
            return false;
        }

        // If it's not a sequence flow, assume it's valid
        if (!this.isElementType(sequenceFlow, BpmnElementName.SequenceFlow)) {
            return true;
        }

        // Get the condition expression, if any
        const conditionExpression = sequenceFlow.conditionExpression;

        // If there's no condition, assume it's always valid
        if (!conditionExpression) {
            return true;
        }

        // If there's a condition, evaluate it against the subroutine context
        // In BPMN, conditionExpression.text might contain the actual expression (e.g. "${foo > 5}")
        const expressionText = conditionExpression.body || (conditionExpression as { text?: string }).text;
        if (!expressionText) {
            // If we can't find any text, treat as invalid or fallback to unconditional
            return false;
        }

        // Evaluate the expression using your engine
        return this.evaluateExpression(expressionText, data.subcontext);
    }

    /**
     * Finds the sequence flow that connects two BPMN elements: the source (sourceElement)
     * and the target (targetElement). This is useful for determining the conditions
     * on the flow between two nodes.
     * 
     * @param sourceElement The source element in the BPMN graph.
     * @param targetElement The target element in the BPMN graph.
     * @returns The sequence flow connecting the two elements, or null if no connection exists.
     */
    private getSequenceFlowBetween(
        sourceElement: BpmnModdle.BaseElement,
        targetElement: BpmnModdle.BaseElement,
    ): BpmnModdle.SequenceFlow | null {
        // If either element is not a FlowNode, there can't be a sequence flow
        if (!this.isFlowNode(sourceElement) || !this.isFlowNode(targetElement)) {
            return null;
        }

        // Outgoing flows from the current element
        const outgoingFlows = sourceElement.outgoing as BpmnModdle.SequenceFlow[];

        if (!outgoingFlows || outgoingFlows.length === 0) {
            // No outgoing flows; return null
            return null;
        }

        // Find the sequence flow whose targetRef matches the targetElement
        for (const flow of outgoingFlows) {
            if (flow.targetRef && flow.targetRef.id === targetElement.id) {
                return flow; // Return the sequence flow if it connects to the target element
            }
        }

        // If no matching sequence flow is found, return null
        return null;
    }

    //TODO call sandbox to evaluate expression
    /**
     * Evaluates a BPMN condition expression against the subroutine context.
     * Depending on your process engine or expression language, 
     * you might parse a string like "${foo > 5}" or "foo == 'bar'".
     * 
     * @param expression The raw expression string extracted from the BPMN conditionExpression.
     * @param subcontext The subroutine context, containing variables and state needed for evaluation.
     * @returns True if the expression evaluates to a truthy value, false otherwise.
     */
    private evaluateExpression(expression: string, subcontext: SubroutineContext): boolean {
        // For demonstration, a naive approach:
        // 1. Strip out BPMN-style placeholders: "${...}"
        // 2. Evaluate in some JavaScript sandbox or your own expression library.
        // 
        // WARNING: Do NOT eval untrusted strings in real code!
        // Use a safe parser or expression engine.
        if (expression.startsWith("${") && expression.endsWith("}")) {
            expression = expression.slice(2, -1).trim();
        }

        // Stub logic: check if context has the variable.
        // For example: "foo > 5" => parse context['foo'] if it exists
        // This is a toy. Replace with your real logic or library.
        try {
            // If you trust your expression or have a safe parser, do something like:
            // return mySafeExpressionEvaluator(expression, context);
            // For demonstration:
            return Boolean(expression);
        } catch (err) {
            // If expression fails to evaluate, log or handle error
            return false;
        }
    }

    /**
     * Determines which outgoing nodes to pick for a given gateway based on
     * its gateway type, the run configuration, and the decision strategy.
     * 
     * This method covers common BPMN gateway behaviors (Exclusive, Parallel, Inclusive).
     * 
     * @param gatewayElement The BPMN gateway element being evaluated.
     * @param availableOutgoingNodes The outgoing flows already filtered by condition (if any).
     * @param decisionKey The unique key to identify the decision point
     * @param decisionStrategy The decision strategy to use for picking nodes.
     * @param data The BPMN graph data containing config, subcontext, etc.
     * @returns An object containing a subset (or all) of the outgoing nodes that should be taken, 
     * and any deferred decisions that need to be made.
     */
    private async pickOutgoingNodesForGateway(
        gatewayElement: BpmnModdle.Gateway,
        availableOutgoingNodes: BpmnModdle.FlowNode[],
        decisionKey: string,
        decisionStrategy: DecisionStrategy,
        data: BpmnGraphData,
    ): Promise<{ availableOutgoingNodes: BpmnModdle.FlowNode[], deferredDecisions: DeferredDecisionData[] }> {
        const { logger, subcontext } = data;

        function nodeToOption(node: BpmnModdle.FlowNode): DecisionOption {
            return {
                nodeId: node.id,
                nodeLabel: node.name || node.id,
            };
        }
        const nodesAsOptions = availableOutgoingNodes.map(nodeToOption);

        // Exclusive (XOR) Gateway => pick exactly one
        if (this.isElementType(gatewayElement, BpmnElementName.GatewayExclusive)) {
            if (availableOutgoingNodes.length <= 1) {
                return { availableOutgoingNodes, deferredDecisions: [] };
            }
            // If multiple are valid, ask the decisionStrategy which one to pick
            const choice = await decisionStrategy.chooseOne(nodesAsOptions, decisionKey, subcontext);
            if (choice.__type === "Waiting") {
                return { availableOutgoingNodes: [], deferredDecisions: [choice] };
            }
            const chosenNodes = availableOutgoingNodes.filter(node => node.id === choice.result);
            return { availableOutgoingNodes: chosenNodes, deferredDecisions: [] };
        }
        // Parallel Gateway => pick them all
        if (this.isElementType(gatewayElement, BpmnElementName.GatewayParallel)) {
            return { availableOutgoingNodes, deferredDecisions: [] };
        }
        // Inclusive Gateway => pick any or all that are valid.
        if (this.isElementType(gatewayElement, BpmnElementName.GatewayInclusive)) {
            const choices = await decisionStrategy.chooseMultiple(nodesAsOptions, decisionKey, subcontext);
            if (choices.__type === "Waiting") {
                return { availableOutgoingNodes: [], deferredDecisions: [choices] };
            }
            const chosenNodes = availableOutgoingNodes.filter(node => choices.result.includes(node.id));
            return { availableOutgoingNodes: chosenNodes, deferredDecisions: [] };
        }
        // Fall back to picking all nodes if the gateway type is unrecognized
        logger.error(`pickOutgoingNodesForGateway: Unrecognized gateway type: ${(gatewayElement as { $type: string }).$type}`);
        return { availableOutgoingNodes, deferredDecisions: [] };
    }

    public async getIONamesPassedIntoNode<Config extends GraphConfig>(data: GetIONamesPassedIntoNodeParams<Config>): Promise<GetIONamesPassedIntoNodeResult> {
        // Parse inputs
        const { config, logger, nodeId } = data as unknown as GetIONamesPassedIntoNodeParams<GraphBpmnConfig>;

        // Initialize result
        const result: GetIONamesPassedIntoNodeResult = {};

        // Validate the BPMN version
        if (!this.isSupportedBpmnVersion(config, logger)) {
            return result;
        }

        // Load the BPMN definitions
        const definitions = await this.getDefinitions(config);

        // Find the node in the BPMN definitions
        const nodeElement = this.findElementById(definitions, nodeId, logger);
        if (!nodeElement) {
            return result;
        }

        // Return early if the node has no extensionElements
        if (!nodeElement.extensionElements || !nodeElement.extensionElements.values) {
            return result;
        }

        // Loop through the extensionElements
        nodeElement.extensionElements.values.forEach((ext: unknown) => {
            // Skip if not an ioMapping or it doesn't have inputs
            if (!this.isVrooliIOMapping(ext) || !ext.inputs) {
                return;
            }
            // Loop through inputs
            ext.inputs.forEach((input) => {
                // Skip inputs that aren't linked to anything
                if (!input.fromContext) {
                    return;
                }
                // Add the name to the result
                result[input.name] = input.fromContext;
            });
        });

        // Return the result
        return result;
    }

}
