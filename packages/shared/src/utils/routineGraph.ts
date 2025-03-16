import BPMNModdle from "bpmn-moddle";

/** Supported graph types */
export type RoutineGraphType = "BPMN-2.0"; // Add more as needed

export class BpmnManager {
    /**
     * The Moddle instance used to create / parse / manipulate BPMN objects.
     */
    private moddle: BPMNModdle.BPMNModdle;

    /**
     * Our definitions object, which is the root container for BPMN processes, messages, signals, etc.
     */
    private definitions: BPMNModdle.Definitions | undefined;

    /**
     * We may store the raw XML if you wish to keep track of it.
     */
    private sourceXml: string | null = null;

    constructor(xml?: string) {
        // Create a new BPMNModdle instance (optionally with your own packages if needed)
        this.moddle = new BPMNModdle();

        // Optionally load BPMN XML if passed in
        if (xml) {
            // You could do this.loadXml(xml) in an async context. For synchronous constructor, just store the string:
            this.sourceXml = xml;
        }
    }

    /**
     * Load and parse a BPMN XML string into the definitions object.
     */
    public async loadXml(xml: string): Promise<void> {
        this.sourceXml = xml;

        // fromXML returns a Definitions object (or throws if parsing fails)
        const definitions = await this.moddle.fromXML(xml);
        this.definitions = definitions; // store it
    }

    /**
     * Initialize an empty BPMN if none is loaded.
     */
    public initializeEmpty(): void {
        // Create a new Definitions. 
        // Casting is often helpful because TS might not automatically infer from the create() overload:
        const newDefs = this.moddle.create("bpmn:Definitions", {
            // typical defaults
            id: "Definitions_" + Date.now(),
            targetNamespace: "http://bpmn.io/schema/bpmn",
            // rootElements must be an array
            rootElements: [],
        }) as BPMNModdle.Definitions;

        this.definitions = newDefs;
    }

    /**
     * Export our in-memory BPMN Definitions to an XML string.
     */
    public async toXml(pretty = true): Promise<string> {
        if (!this.definitions) {
            throw new Error("No BPMN definitions are loaded/initialized.");
        }

        // bpmn-moddle's `toXML` typically isn't typed strongly in older versions, so we might cast to `any`.
        // The returned object often has the shape { xml: string }.
        const { xml } = await (this.moddle as any).toXML(this.definitions, { format: pretty });
        return xml;
    }

    /**
     * Create a BPMN Process inside our definitions. 
     */
    public createProcess(attributes: Partial<BPMNModdle.Process> = {}): BPMNModdle.Process {
        if (!this.definitions) {
            throw new Error("Call initializeEmpty() or loadXml() before creating processes.");
        }

        // Create a bpmn:Process
        const process = this.moddle.create("bpmn:Process", {
            id: attributes.id || "Process_" + Date.now(),
            name: attributes.name || "New Process",
            isExecutable: attributes.isExecutable ?? true,
            flowElements: [],  // Important to init if you plan to push tasks, events
            ...attributes,     // merge in anything else
        }) as BPMNModdle.Process;

        // Insert into definitions.rootElements
        this.definitions.rootElements.push(process);

        return process;
    }

    /**
     * Create a FlowElement (e.g., StartEvent, UserTask, etc.) inside a given process.
     *
     * @param container The process (or sub-process) to contain this flow element.
     * @param typeName  The BPMN element type (e.g. "bpmn:StartEvent", "bpmn:UserTask")
     * @param props     Additional props (id, name, etc.)
     */
    public createFlowElement<T extends BPMNModdle.FlowElement = BPMNModdle.FlowElement>(
        container: BPMNModdle.FlowElementsContainer,
        typeName: BPMNModdle.ElementType,
        props: Partial<T> = {},
    ): T {
        // create the element
        const flowEl = this.moddle.create(typeName, {
            id: props.id || `${typeName.split(":")[1]}_${Date.now()}`,
            ...props,
        }) as T;

        // ensure container.flowElements is an array
        if (!container.flowElements) {
            container.flowElements = [];
        }
        container.flowElements.push(flowEl);

        return flowEl;
    }

    /**
     * Example method: Create a SequenceFlow from source to target within the same process.
     */
    public createSequenceFlow(
        process: BPMNModdle.Process,
        sourceRef: BPMNModdle.FlowNode,
        targetRef: BPMNModdle.FlowNode,
        props?: Partial<BPMNModdle.SequenceFlow>,
    ): BPMNModdle.SequenceFlow {
        if (!process.flowElements) process.flowElements = [];

        // Create sequence flow
        const seqFlow = this.moddle.create("bpmn:SequenceFlow", {
            id: props?.id || "SequenceFlow_" + Date.now(),
            sourceRef,
            targetRef,
            ...props,
        }) as BPMNModdle.SequenceFlow;

        // Add to process flowElements
        process.flowElements.push(seqFlow);

        // Also update the incoming/outgoing arrays on the nodes:
        if (!sourceRef.outgoing) sourceRef.outgoing = [];
        sourceRef.outgoing.push(seqFlow);

        if (!targetRef.incoming) targetRef.incoming = [];
        targetRef.incoming.push(seqFlow);

        return seqFlow;
    }

    /**
     * Find an element by ID among all root processes (and optionally sub-processes if you handle recursion).
     */
    public findElementById(elementId: string): BPMNModdle.FlowElement | undefined {
        if (!this.definitions) return undefined;

        for (const rootEl of this.definitions.rootElements || []) {
            // We only care about rootEl if it's a Process or FlowElementsContainer
            if (!("flowElements" in rootEl)) continue; // skip e.g. bpmn:Message, etc.

            const process = rootEl as BPMNModdle.Process;
            if (!process.flowElements) continue;

            const found = process.flowElements.find(fe => fe.id === elementId);
            if (found) return found;
        }

        return undefined;
    }

    /**
     * Update some of the properties of an element (like name).
     */
    public updateElement<T extends BPMNModdle.BaseElement>(
        element: T,
        updates: Partial<T>,
    ): void {
        Object.entries(updates).forEach(([key, val]) => {
            // Typescript might complain if val is not exactly correct type
            // but let's assume it matches for brevity:
            (element as any)[key] = val;
        });
    }

    /**
     * Remove an element from the definitions. 
     * (In BPMN, you also typically remove any SequenceFlows referencing it, etc.)
     */
    public removeElement(elementId: string): void {
        if (!this.definitions) return;

        for (const rootEl of this.definitions.rootElements || []) {
            if (!("flowElements" in rootEl)) continue;

            const process = rootEl as BPMNModdle.Process;
            if (!process.flowElements) continue;

            const idx = process.flowElements.findIndex(fe => fe.id === elementId);
            if (idx !== -1) {
                // Remove it
                process.flowElements.splice(idx, 1);
                // optionally also remove references from other elements’ incoming/outgoing
                break;
            }
        }
    }

    /**
     * Additional utility methods could be:
     *  - createStartEvent()
     *  - createUserTask()
     *  - createEndEvent()
     *  - createLane()
     *  - queryAllUserTasks()
     *  - etc.
     */

    /**
     * Example: find all UserTasks
     */
    public findAllUserTasks(): BPMNModdle.UserTask[] {
        if (!this.definitions) return [];

        const tasks: BPMNModdle.UserTask[] = [];
        for (const rootEl of this.definitions.rootElements || []) {
            if (!("flowElements" in rootEl)) continue;
            const process = rootEl as BPMNModdle.Process;

            (process.flowElements || []).forEach(el => {
                if (el.$type === "bpmn:UserTask") {
                    tasks.push(el as BPMNModdle.UserTask);
                }
            });
        }
        return tasks;
    }
}
