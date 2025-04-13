import BpmnModdle from "bpmn-moddle";
import { APP_URL, BUSINESS_NAME } from "../consts/businessData.js";
import { BpmnSchema, GraphBpmnConfig, GraphBpmnConfigObject } from "../run/configs/routine.js";

// TODO need to support multiple processes for multi-user support, message flows for sending data between processes, interrupting and non-interrupting boundary events (related to messaging), intermediate catch events (also related to messaging), tasks for single-step subroutines, call activities for multi-step subroutines, etc.
// Should already have low-level functions to generate the bpmn elements for most of these, but need to add higher-level functions to that can be used in the UI.

const BPMN_VERSION_LABEL = "BPMN-2.0";
const BPMN_MANAGER_VERSION = "1.0";
const TAG_DELIMITER = ":";
const ID_DELIMITER = "_";
const DEFAULT_USE_NAMESPACE = true;
const DEFAULT_ACTIVITY_MAP: BpmnSchema["activityMap"] = {};
const DEFAULT_ROOT_CONTEXT: BpmnSchema["rootContext"] = { inputMap: {}, outputMap: {} };

type Namespace = {
    uri: string;
    prefix: string;
}

type BpmnManagerConfig = {
    namespace: Namespace;
    useNamespace: boolean;
}

/**
 * The BPMN namespace for process logic. 
 * This is technically the only required namespace for BPMN, but others may be used 
 * (e.g. for visualizing the diagram).
 */
const NAMESPACE_PROCESS: Namespace = {
    uri: "http://bpmn.io/schema/bpmn",
    prefix: "bpmn",
};
/**
 * One of the BPMN namespaces for visualizing the diagram.
 */
const NAMESPACE_DIAGRAM_INTERCHANGE: Namespace = {
    uri: "http://www.omg.org/spec/BPMN/20100524/DI",
    prefix: "bpmndi",
};
/**
 * Another BPMN namespace for visualizing the diagram.
 */
const NAMESPACE_DIAGRAM_INTERCHANGE_BASE: Namespace = {
    uri: "http://www.omg.org/spec/DD/20100524/DI",
    prefix: "di",
};
/**
 * The common BPMN namespace for visualizing the diagram.
 */
const NAMESPACE_DIAGRAM_COMMON: Namespace = {
    uri: "http://www.omg.org/spec/DD/20100524/DC",
    prefix: "dc",
};
/**
 * Custom namespace for our own extension.
 */
const NAMESPACE_VROOLI_EXTENSION: Namespace = {
    uri: `${APP_URL}/bpmn`,
    prefix: "vrooli",
};

/** Supported graph types */
export type RoutineGraphType = "BPMN-2.0"; // Add more as needed

enum BpmnElementName {
    Definitions = "Definitions",
    Process = "Process",
    Parallel = "ParallelGateway",
    Exclusive = "ExclusiveGateway",
    Inclusive = "InclusiveGateway",
    EventBased = "EventBasedGateway",
    SequenceFlow = "SequenceFlow",
    Expression = "FormalExpression",
    StartEvent = "StartEvent",
    EndEvent = "EndEvent",
    IntermediateCatchEvent = "IntermediateCatchEvent",
    IntermediateThrowEvent = "IntermediateThrowEvent",
    BoundaryEvent = "BoundaryEvent",
    Task = "Task",
    UserTask = "UserTask",
    ServiceTask = "ServiceTask",
    CallActivity = "CallActivity",
}

type BpmnElement = {
    [BpmnElementName.Definitions]: BpmnModdle.Definitions;
    [BpmnElementName.Process]: BpmnModdle.Process;
    [BpmnElementName.Parallel]: BpmnModdle.ParallelGateway;
    [BpmnElementName.Exclusive]: BpmnModdle.ExclusiveGateway;
    [BpmnElementName.Inclusive]: BpmnModdle.InclusiveGateway;
    [BpmnElementName.EventBased]: BpmnModdle.EventBasedGateway;
    [BpmnElementName.SequenceFlow]: BpmnModdle.SequenceFlow;
    [BpmnElementName.Expression]: BpmnModdle.FormalExpression;
    [BpmnElementName.StartEvent]: BpmnModdle.StartEvent;
    [BpmnElementName.EndEvent]: BpmnModdle.EndEvent;
    [BpmnElementName.IntermediateCatchEvent]: BpmnModdle.IntermediateCatchEvent;
    [BpmnElementName.IntermediateThrowEvent]: BpmnModdle.IntermediateThrowEvent;
    [BpmnElementName.BoundaryEvent]: BpmnModdle.BoundaryEvent;
    [BpmnElementName.Task]: BpmnModdle.Task;
    [BpmnElementName.UserTask]: BpmnModdle.UserTask;
    [BpmnElementName.ServiceTask]: BpmnModdle.ServiceTask;
    [BpmnElementName.CallActivity]: BpmnModdle.CallActivity;
}

type CreateBaseElement = {
    id: string;
}

type CreateBpmnElement = CreateBaseElement & {
    name?: string;
}

type CreateGateway = CreateBpmnElement & {
    gatewayDirection: BpmnModdle.GatewayDirection;
}

type CreateExclusiveGateway = CreateGateway;
type CreateParallelGateway = CreateGateway;
type CreateInclusiveGateway = CreateGateway;
type CreateEventBasedGateway = CreateGateway;

type CreateSequenceFlow = CreateBaseElement & {
    sourceRef: string;  // the id of the source element
    targetRef: string;  // the id of the target element
    conditionExpression?: string; // the condition for the flow (e.g. "x > 5")
};

type CreateExpression = CreateBaseElement & {
    body: string;
}

type CreateEvent = CreateBaseElement & {
    // For boundary/intermediate events, you might add isInterrupting, attachedToRef, etc.
    isInterrupting?: boolean;
    attachedToRef?: string;
};

type CreateStartEvent = CreateEvent;
type CreateEndEvent = CreateEvent;
type CreateIntermediateCatchEvent = CreateEvent;
type CreateIntermediateThrowEvent = CreateEvent;
type CreateBoundaryEvent = CreateEvent;

type CreateTask = CreateBpmnElement;
type CreateServiceTask = CreateTask;
type CreateUserTask = CreateTask;
type CreateCallActivity = CreateTask;

/**
 * Prefixes a string, if it doesn't already have it.
 * 
 * @param prefix The prefix to add
 * @param str The string to prefix
 * @param delimiter The delimiter to use between the prefix and the string
 * @returns The prefixed string
 */
function withPrefix(prefix: string, str: string, delimiter: string): string {
    return str.startsWith(prefix + delimiter)
        ? str
        : prefix + delimiter + str;
}

/**
 * Removes the prefix from a string, if it has it.
 * 
 * @param prefix The prefix to remove
 * @param str The string to remove the prefix from
 * @param delimiter The delimiter to use between the prefix and the string
 * @returns The string without the prefix
 */
function withoutPrefix(prefix: string, str: string, delimiter: string): string {
    return str.startsWith(prefix + delimiter)
        ? str.slice(prefix.length + 1)
        : str;
}

/**
 * @returns The correct tag for an element
 */
function getElementTag(tag: keyof BpmnElement, namespace: Pick<Namespace, "prefix">, useNamespace: boolean): string {
    return useNamespace
        ? withPrefix(namespace.prefix, tag, TAG_DELIMITER)
        : withoutPrefix(namespace.prefix, tag, TAG_DELIMITER);
}

/**
 * @returns The correct Id for an element
 */
function getElementId(id: string, namespace: Pick<Namespace, "prefix">, useNamespace: boolean): string {
    return useNamespace
        ? withPrefix(namespace.prefix, id, ID_DELIMITER)
        : withoutPrefix(namespace.prefix, id, ID_DELIMITER);
}

class GatewayManager {
    private namespace = NAMESPACE_PROCESS;

    constructor(private bpmnModdle: BpmnModdle.BPMNModdle) { }

    createExclusiveGateway(data: CreateExclusiveGateway, useNamespace: boolean): BpmnModdle.ExclusiveGateway {
        const tag = getElementTag(BpmnElementName.Exclusive, this.namespace, useNamespace);
        const id = getElementId(data.id, this.namespace, useNamespace);
        return this.bpmnModdle.create(tag, { ...data, id }) as BpmnModdle.ExclusiveGateway;
    }

    createParallelGateway(data: CreateParallelGateway, useNamespace: boolean): BpmnModdle.ParallelGateway {
        const tag = getElementTag(BpmnElementName.Parallel, this.namespace, useNamespace);
        const id = getElementId(data.id, this.namespace, useNamespace);
        return this.bpmnModdle.create(tag, { ...data, id }) as BpmnModdle.ParallelGateway;
    }

    createInclusiveGateway(data: CreateInclusiveGateway, useNamespace: boolean): BpmnModdle.InclusiveGateway {
        const tag = getElementTag(BpmnElementName.Inclusive, this.namespace, useNamespace);
        const id = getElementId(data.id, this.namespace, useNamespace);
        return this.bpmnModdle.create(tag, { ...data, id }) as BpmnModdle.InclusiveGateway;
    }

    createEventBasedGateway(data: CreateEventBasedGateway, useNamespace: boolean): BpmnModdle.EventBasedGateway {
        const tag = getElementTag(BpmnElementName.EventBased, this.namespace, useNamespace);
        const id = getElementId(data.id, this.namespace, useNamespace);
        return this.bpmnModdle.create(tag, { ...data, id }) as BpmnModdle.EventBasedGateway;
    }
}

class SequenceFlowManager {
    private namespace = NAMESPACE_PROCESS;

    constructor(private bpmnModdle: BpmnModdle.BPMNModdle) { }

    public createExpression(data: CreateExpression, useNamespace: boolean): BpmnModdle.FormalExpression {
        const tag = getElementTag(BpmnElementName.Expression, this.namespace, useNamespace);
        const id = getElementId(data.id, this.namespace, useNamespace);
        return this.bpmnModdle.create(tag, { ...data, id }) as BpmnModdle.FormalExpression;
    }

    public createSequenceFlow(data: CreateSequenceFlow, useNamespace: boolean): BpmnModdle.SequenceFlow {
        const tag = getElementTag(BpmnElementName.SequenceFlow, this.namespace, useNamespace);
        const id = getElementId(data.id, this.namespace, useNamespace);

        const { conditionExpression, ...restData } = data;
        const sequenceFlow = this.bpmnModdle.create(tag, { ...restData, id }) as BpmnModdle.SequenceFlow;
        if (conditionExpression) {
            const expression = this.createExpression({ id: `${id}_expression`, body: conditionExpression }, useNamespace);
            sequenceFlow.conditionExpression = expression;
        }
        return sequenceFlow;
    }
}

export class EventManager {
    private namespace = NAMESPACE_PROCESS;

    constructor(private bpmnModdle: BpmnModdle.BPMNModdle) { }

    public createStartEvent(data: CreateStartEvent, useNamespace: boolean): BpmnModdle.StartEvent {
        const tag = getElementTag(BpmnElementName.StartEvent, this.namespace, useNamespace);
        const id = getElementId(data.id, this.namespace, useNamespace);
        return this.bpmnModdle.create(tag, { ...data, id }) as BpmnModdle.StartEvent;
    }

    public createEndEvent(data: CreateEndEvent, useNamespace: boolean): BpmnModdle.EndEvent {
        const tag = getElementTag(BpmnElementName.EndEvent, this.namespace, useNamespace);
        const id = getElementId(data.id, this.namespace, useNamespace);
        return this.bpmnModdle.create(tag, { ...data, id }) as BpmnModdle.EndEvent;
    }

    public createIntermediateCatchEvent(data: CreateIntermediateCatchEvent, useNamespace: boolean): BpmnModdle.IntermediateCatchEvent {
        const tag = getElementTag(BpmnElementName.IntermediateCatchEvent, this.namespace, useNamespace);
        const id = getElementId(data.id, this.namespace, useNamespace);
        return this.bpmnModdle.create(tag, { ...data, id }) as BpmnModdle.IntermediateCatchEvent;
    }

    public createIntermediateThrowEvent(data: CreateIntermediateThrowEvent, useNamespace: boolean): BpmnModdle.IntermediateThrowEvent {
        const tag = getElementTag(BpmnElementName.IntermediateThrowEvent, this.namespace, useNamespace);
        const id = getElementId(data.id, this.namespace, useNamespace);
        return this.bpmnModdle.create(tag, { ...data, id }) as BpmnModdle.IntermediateThrowEvent;
    }

    public createBoundaryEvent(
        data: CreateBoundaryEvent,
        useNamespace: boolean,
    ): BpmnModdle.BoundaryEvent {
        const tag = getElementTag(BpmnElementName.BoundaryEvent, this.namespace, useNamespace);
        const id = getElementId(data.id, this.namespace, useNamespace);

        // Note: For a boundary event, you often need:
        //  - attachedToRef: ID of the activity the boundary is attached to
        //  - cancelActivity: if it’s interrupting (isInterrupting = true)
        return this.bpmnModdle.create(tag, {
            ...data,
            id,
            cancelActivity: data.isInterrupting !== false, // default to true if not specified
            attachedToRef: data.attachedToRef, // must be set if attaching to an activity
        }) as BpmnModdle.BoundaryEvent;
    }
}

export class ActivityManager {
    private namespace = NAMESPACE_PROCESS;

    constructor(private bpmnModdle: BpmnModdle.BPMNModdle) { }

    public createTask(data: CreateTask, useNamespace: boolean): BpmnModdle.Task {
        const tag = getElementTag(BpmnElementName.Task, this.namespace, useNamespace);
        const id = getElementId(data.id, this.namespace, useNamespace);
        return this.bpmnModdle.create(tag, { ...data, id }) as BpmnModdle.Task;
    }

    public createUserTask(data: CreateUserTask, useNamespace: boolean): BpmnModdle.UserTask {
        const tag = getElementTag(BpmnElementName.UserTask, this.namespace, useNamespace);
        const id = getElementId(data.id, this.namespace, useNamespace);
        return this.bpmnModdle.create(tag, { ...data, id }) as BpmnModdle.UserTask;
    }

    public createServiceTask(data: CreateServiceTask, useNamespace: boolean): BpmnModdle.ServiceTask {
        const tag = getElementTag(BpmnElementName.ServiceTask, this.namespace, useNamespace);
        const id = getElementId(data.id, this.namespace, useNamespace);
        return this.bpmnModdle.create(tag, { ...data, id }) as BpmnModdle.ServiceTask;
    }

    public createCallActivity(data: CreateCallActivity, useNamespace: boolean): BpmnModdle.CallActivity {
        const tag = getElementTag(BpmnElementName.CallActivity, this.namespace, useNamespace);
        const id = getElementId(data.id, this.namespace, useNamespace);
        return this.bpmnModdle.create(tag, { ...data, id }) as BpmnModdle.CallActivity;
    }
}

export class BpmnManager {
    private bpmnModdle = new BpmnModdle();

    public gatewayManager: GatewayManager;
    public activityManager: ActivityManager;
    public eventManager: EventManager;
    public sequenceFlowManager: SequenceFlowManager;

    /**
     * Whether to use the namespace for BPMN elements.
     */
    private useNamespace: boolean;
    /**
     * Our definitions object, which is the root container for BPMN processes, messages, signals, etc.
     */
    private definitions: BpmnModdle.Definitions | undefined;
    /**
     * Maps call activities to subroutine information.
     */
    private activityMap: BpmnSchema["activityMap"];
    /**
     * Maps routine (not subroutine!) inputs and outputs to their names in the BPMN schema.
     */
    private rootContext: BpmnSchema["rootContext"];

    constructor(graphConfig: GraphBpmnConfig, managerConfig: BpmnManagerConfig) {
        this.gatewayManager = new GatewayManager(this.bpmnModdle);
        this.activityManager = new ActivityManager(this.bpmnModdle);
        this.eventManager = new EventManager(this.bpmnModdle);
        this.sequenceFlowManager = new SequenceFlowManager(this.bpmnModdle);

        const xmlData = graphConfig.schema.data;

        // Parse the XML data into a BPMN definitions object
        this.bpmnModdle.fromXML(xmlData, (error: Error, defs: BpmnModdle.Definitions) => {
            if (error) {
                throw new Error("Error parsing BPMN XML: " + error.message);
            }
            this.definitions = defs;
        });
        // If no XML data is provided, initialize an empty BPMN
        if (!this.definitions) {
            this.initialize();
        }

        this.useNamespace = managerConfig.useNamespace ?? DEFAULT_USE_NAMESPACE;
        this.activityMap = graphConfig.schema.activityMap || DEFAULT_ACTIVITY_MAP;
        this.rootContext = graphConfig.schema.rootContext || DEFAULT_ROOT_CONTEXT;
    }

    /**
     * Enable or disable the use of the namespace for BPMN elements.
     * 
     * @param useNamespace Whether to use the namespace.
     */
    public setUseNamespace(useNamespace: boolean): void {
        this.useNamespace = useNamespace;
    }

    /**
     * Initialize an empty BPMN if none is loaded.
     */
    public initialize(): void {
        // Create a new Definitions (root container)
        const definitionTag = getElementTag(BpmnElementName.Definitions, NAMESPACE_PROCESS, this.useNamespace);
        const idBase = Date.now();
        const newDefs = this.bpmnModdle.create(definitionTag, {
            [`xmlns:${NAMESPACE_PROCESS.prefix}`]: NAMESPACE_PROCESS.uri,
            [`xmlns:${NAMESPACE_DIAGRAM_INTERCHANGE.prefix}`]: NAMESPACE_DIAGRAM_INTERCHANGE.uri,
            [`xmlns:${NAMESPACE_DIAGRAM_INTERCHANGE_BASE.prefix}`]: NAMESPACE_DIAGRAM_INTERCHANGE_BASE.uri,
            [`xmlns:${NAMESPACE_DIAGRAM_COMMON.prefix}`]: NAMESPACE_DIAGRAM_COMMON.uri,
            [`xmlns:${NAMESPACE_VROOLI_EXTENSION.prefix}`]: NAMESPACE_VROOLI_EXTENSION.uri,
            id: `Definitions_${idBase}`,
            exporter: `${BUSINESS_NAME} BPMN Manager`,
            exporterVersion: BPMN_MANAGER_VERSION,
            targetNamespace: `${APP_URL}/bpmn/${idBase}`,
        }) as BpmnModdle.Definitions;

        this.definitions = newDefs;

        if (!this.activityMap) {
            this.activityMap = DEFAULT_ACTIVITY_MAP;
        }
        if (!this.rootContext) {
            this.rootContext = DEFAULT_ROOT_CONTEXT;
        }
    }

    /**
     * Export our in-memory data to a GraphBpmnConfigObject.
     */
    public async export(pretty = true): Promise<GraphBpmnConfigObject> {
        if (!this.definitions) {
            throw new Error("No BPMN definitions are loaded/initialized.");
        }

        // bpmn-moddle's `toXML` typically isn't typed strongly in older versions, so we might cast to `any`.
        // The returned object often has the shape { xml: string }.
        const { xml } = await (this.bpmnModdle as any).toXML(this.definitions, { format: pretty });

        return {
            __type: BPMN_VERSION_LABEL,
            __version: BPMN_MANAGER_VERSION,
            schema: {
                __format: "xml",
                data: xml,
                activityMap: this.activityMap,
                rootContext: this.rootContext,
            },
        };
    }
}
