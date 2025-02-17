import i18next from "i18next";
import { RoutineType, type RoutineVersion } from "../../api/types.js";
import { HttpMethod } from "../../consts/api.js";
import { type PassableLogger } from "../../consts/commonTypes.js";
import { InputType } from "../../consts/model.js";
import { FormInputBase, FormSchema } from "../../forms/types.js";
import { BotStyle, Id } from "../types.js";
import { LlmModel } from "./bot.js";
import { parseObject, stringifyObject, type StringifyMode } from "./utils.js";

const LATEST_VERSION = "1.0";
const DEFAULT_STRINGIFY_MODE: StringifyMode = "json";

/**
 * Represents how to call a REST (or similar) API.
 * NOTE: Untested
 */
export interface ConfigCallDataApi {
    /**
     * The URL or endpoint of the API.
     * e.g. "https://api.example.com/v1/create"
     */
    endpoint: string;
    /**
     * The HTTP method to use when calling the API.
     */
    method: HttpMethod;
    /**
     * Optional HTTP headers (e.g. authorization token, content type, etc.).
     */
    headers?: Record<string, string>;
    /**
     * The payload or request body.
     * For a GET request, this is typically not used, but you could store query params here.
     */
    body?: any;
    /**
     * Timeout or other request options as desired.
     */
    timeoutMs?: number;
    /**
     * Any other relevant metadata about the API call.
     * (e.g., a flag indicating if you should retry on failure, number of retries, etc.)
     */
    meta?: Record<string, any>;
}

/**
 * Represents how to call an LLM.
 * 
 * NOTE: This can be provided to non-generate routines, and will allow 
 * automated runs to customize how inputs are generated.
 */
export type ConfigCallDataGenerate = {
    /**
     * Determines which bot personas to use for a routine.
     */
    botStyle?: BotStyle;
    /**
     * The maximum number of tokens to generate.
     */
    maxTokens?: number | null;
    /**
     * The model to use for the LLM.
     */
    model?: LlmModel | null;
    /**
     * The prompt to use for the LLM.
     */
    prompt?: string | null;
    /**
     * The bot ID to use for the LLM.
     */
    respondingBot?: Id | null;
};

/**
 * Represents how to call a Smart Contract on a blockchain.
 * NOTE: Untested
 */
export interface ConfigCallDataSmartContract {
    /**
     * Address of the contract on the blockchain.
     */
    contractAddress: string;
    /**
     * The name or identifier of the blockchain/chain ID/network.
     * e.g. "ethereum", "polygon", "rinkeby", etc.
     */
    chain: string;
    /**
     * Name of the method/function you want to invoke in the smart contract.
     */
    methodName: string;
    /**
     * Arguments/params to pass to the contract’s method.
     */
    args?: any[];
    /**
     * Optional overrides for transaction details such as gas limit, gas price, etc.
     */
    txOptions?: {
        gasLimit?: number;
        gasPrice?: string;
        value?: string; // if the contract function requires sending ETH
        nonce?: number;
        [key: string]: any; // for any additional chain-specific fields
    };
    /**
     * Any additional metadata about how to execute the contract call.
     * Could include references to a wallet, signers, or other instructions.
     */
    meta?: Record<string, any>;
}

type CallDataApiConfigObject = {
    __version: string;
    schema: ConfigCallDataApi;
};
type CallDataGenerateConfigObject = {
    __version: string;
    schema: ConfigCallDataGenerate;
};
type CallDataSmartContractConfigObject = {
    __version: string;
    schema: ConfigCallDataSmartContract;
};

type FormInputConfigObject = {
    __version: string;
    schema: FormSchema;
};
type FormOutputConfigObject = {
    __version: string;
    schema: FormSchema;
};

type GraphType = "BPMN-2.0"; // Add more as needed
type GraphBaseConfigObject = {
    __version: string;
    __type: GraphType;
};
export type BpmnSchema = {
    /** How the BPMN diagram data is stored */
    __format: "xml";
    /** The raw data of the BPMN diagram in __format */
    data: string;
    // TODO add example like below (and update type if needed) to support multiInstanceLoopCharacteristics (i.e. looping a node for each item in a list). Needed for things like the tutorial routine that loops through each chapter in a book.
    //     <bpmn:callActivity id="callActivityA" name="Call A" calledElement="A">
    //     <bpmn:multiInstanceLoopCharacteristics isSequential="true">
    //       <bpmn:extensionElements>
    //         <vrooli:ioMapping>
    //           <vrooli:input name="inputA" fromContext="callActivityB.inputC" />
    //           <vrooli:input name="inputB" />
    //           <vrooli:input name="inputC" fromContext="root.inputD" />
    //           <vrooli:output name="outputX" />
    //           <vrooli:output name="outputY" toRootContext="outputZ" />
    //         </vrooli:ioMapping>
    //       </bpmn:extensionElements>
    //       <bpmn:loopDataInputRef>items</bpmn:loopDataInputRef>
    //       <bpmn:loopDataOutputRef>results</bpmn:loopDataOutputRef>
    //       <bpmn:inputDataItem>item</bpmn:inputDataItem>
    //       <bpmn:outputDataItem>result</bpmn:outputDataItem>
    //       <bpmn:completionCondition xsi:type="bpmn:tFormalExpression">
    //         result.isSuccessful
    //       </bpmn:completionCondition>
    //     </bpmn:multiInstanceLoopCharacteristics>
    //   </bpmn:callActivity> 
    /**
     * Map of BPMN call activity ID to subroutine information.
     * 
     * Specifying subroutine information here instead of in the BPMN diagram allows us to:
     * - Keep the BPMN diagram clean and focused on the process
     * - Import and export the BPMN diagram to other platforms more easily
     * - Dynamically change the subroutine based on the context
     * 
     * To connect the BPMN diagram to the subroutine, simply use a call activity with 
     * `calledElement` set to a key in this map, and the corresponding value will be the subroutine ID and 
     * input/output mappings.
     * 
     * Full example:
     * <bpmn:callActivity id="callActivityA" name="Call A" calledElement="A">
     *     <bpmn:extensionElements>
     *       <vrooli:ioMapping>
     *         // The optional `fromContext` field allows for connecting an input to another input or output
     *         <vrooli:input name="inputA" fromContext="callActivityB.inputC" />
     *         <vrooli:iput name="inputB" />
     *         // `fromContext` with the `root` keyword allows for connecting to data passed into the routine, instead of another subroutine
     *         <vrooli:iput name="inputC"  fromContext="root.inputD" />
     *         <vrooli:output name="outputX" />
     *         // The optional `toRootContext` field allows for exporting the output to the parent context when the routine is done, rather than the 
     *         // default behavior of keeping the output in the scope of the current routine
     *         <vrooli:output name="outputY" toRootContext="outputZ" />
     *       </vrooli:ioMapping>
     *     </bpmn:extensionElements>
     * </bpmn:callActivity>
     * 
     * {
     *   callActivityA: {
     *       subroutineId: "123-456-789",
     *       inputMap: {
     *           inputA: "subroutineInputA",
     *           inputB: "subroutineInputB",
     *       },
     *       outputMap: {
     *           outputB: "subroutineOutputB",
     *       },
     * }
     */
    activityMap: Record<string, {
        /** The ID of the subroutine */
        subroutineId: string;
        /** A map of call activity input name to subroutine input name */
        inputMap: Record<string, string>;
        /** A map of call activity output name to subroutine output name */
        outputMap: Record<string, string>;
    }>;
    /**
     * Map of root context name to routine (not subroutine) input/output name
     */
    rootContext: {
        inputMap: Record<string, string>;
        outputMap: Record<string, string>;
    }
};
export type GraphBpmnConfigObject = GraphBaseConfigObject & {
    __type: "BPMN-2.0";
    schema: BpmnSchema;
};
type GraphConfigObject = GraphBpmnConfigObject;

/**
 * Represents the configuration for a routine version. 
 * Includes configs for calling another service, I/O, and more.
 */
type RoutineVersionConfigObject = {
    /** Store the version number for future compatibility */
    __version: string;
    /** Config for calling an API */
    callDataApi?: CallDataApiConfigObject;
    /** Config for calling an LLM */
    callDataGenerate?: CallDataGenerateConfigObject;
    /** Config for calling a smart contract */
    callDataSmartContract?: CallDataSmartContractConfigObject;
    /** Config for entering information to complete the routine */
    formInput?: FormInputConfigObject;
    /** Config for information generated by the routine */
    formOutput?: FormOutputConfigObject;
    /** Config for running multi-step routines */
    graph?: GraphConfigObject;
};

function defaultConfigCallDataApi(): CallDataApiConfigObject {
    return {
        __version: LATEST_VERSION,
        schema: {
            endpoint: "",
            method: "GET",
        },
    };
}
function defaultConfigCallDataGenerate(): CallDataGenerateConfigObject {
    return {
        __version: LATEST_VERSION,
        schema: {
            botStyle: BotStyle.Default,
            maxTokens: null,
            model: null,
            prompt: null,
            respondingBot: null,
        },
    };
}
function defaultConfigCallDataSmartContract(): CallDataSmartContractConfigObject {
    return {
        __version: LATEST_VERSION,
        schema: {
            contractAddress: "",
            chain: "ethereum",
            methodName: "",
        },
    };
}

function defaultSchemaInput(): FormInputConfigObject {
    return {
        __version: LATEST_VERSION,
        schema: {
            containers: [],
            elements: [],
        },
    };
}

function defaultSchemaOutput(): FormOutputConfigObject {
    return {
        __version: LATEST_VERSION,
        schema: {
            containers: [],
            elements: [],
        },
    };
}

/**
 * @return The default output form object for a Generate routine, 
 * which always returns text (for now, since we only call LLMs)
 */
function defaultSchemaOutputGenerate(): FormOutputConfigObject {
    return {
        __version: LATEST_VERSION,
        schema: {
            containers: [],
            elements: [
                {
                    fieldName: "response",
                    id: "response",
                    label: i18next.t("Response", { count: 1 }),
                    props: {
                        placeholder: "Model response will be displayed here",
                    },
                    type: InputType.Text,
                },
            ],
        },
    };
}

const defaultConfigFormInputMap = {
    [RoutineType.Action]: () => defaultSchemaInput(),
    [RoutineType.Api]: () => defaultSchemaInput(),
    [RoutineType.Code]: () => defaultSchemaInput(),
    [RoutineType.Data]: () => defaultSchemaInput(),
    [RoutineType.Generate]: () => defaultSchemaInput(),
    [RoutineType.Informational]: () => defaultSchemaInput(),
    [RoutineType.MultiStep]: () => defaultSchemaInput(),
    [RoutineType.SmartContract]: () => defaultSchemaInput(),
};

const defaultConfigFormOutputMap = {
    [RoutineType.Action]: () => defaultSchemaOutput(),
    [RoutineType.Api]: () => defaultSchemaOutput(),
    [RoutineType.Code]: () => defaultSchemaOutput(),
    [RoutineType.Data]: () => defaultSchemaOutput(),
    [RoutineType.Generate]: () => defaultSchemaOutputGenerate(),
    [RoutineType.Informational]: () => defaultSchemaOutput(),
    [RoutineType.MultiStep]: () => defaultSchemaOutput(),
    [RoutineType.SmartContract]: () => defaultSchemaOutput(),
};

function isValidFormSchema(schema: FormSchema): boolean {
    return (
        Object.prototype.hasOwnProperty.call(schema, "containers") &&
        Array.isArray(schema.containers) &&
        Object.prototype.hasOwnProperty.call(schema, "elements") &&
        Array.isArray(schema.elements)
    );
}

/**
 * Top-level routine config that encapsulates all sub-config sections.
 */
export class RoutineVersionConfig {
    __version: string;
    callDataApi?: CallDataApiConfig;
    callDataGenerate?: CallDataGenerateConfig;
    callDataSmartContract?: CallDataSmartContractConfig;
    formInput?: FormInputConfig;
    formOutput?: FormOutputConfig;
    graph?: GraphConfig;

    constructor(data: RoutineVersionConfigObject) {
        this.__version = data.__version ?? LATEST_VERSION;
        this.callDataApi = data.callDataApi ? new CallDataApiConfig(data.callDataApi) : undefined;
        this.callDataGenerate = data.callDataGenerate ? new CallDataGenerateConfig(data.callDataGenerate) : undefined;
        this.callDataSmartContract = data.callDataSmartContract ? new CallDataSmartContractConfig(data.callDataSmartContract) : undefined;
        this.formInput = data.formInput ? new FormInputConfig(data.formInput) : undefined;
        this.formOutput = data.formOutput ? new FormOutputConfig(data.formOutput) : undefined;
        this.graph = data.graph ? GraphConfig.create(data.graph) : undefined;
    }

    static deserialize(
        { config, routineType }: Pick<RoutineVersion, "config" | "routineType">,
        logger: PassableLogger,
        { mode = DEFAULT_STRINGIFY_MODE, useFallbacks = true }: { mode?: StringifyMode, useFallbacks?: boolean },
    ): RoutineVersionConfig {
        let obj = config ? parseObject<RoutineVersionConfigObject>(config, mode, logger) : null;
        if (!obj) {
            obj = { __version: LATEST_VERSION };
        }
        if (useFallbacks) {
            if (!obj.callDataApi && routineType === RoutineType.Api) {
                obj.callDataApi = defaultConfigCallDataApi();
            }
            if (!obj.callDataGenerate && routineType === RoutineType.Generate) {
                obj.callDataGenerate = defaultConfigCallDataGenerate();
            }
            if (!obj.callDataSmartContract && routineType === RoutineType.SmartContract) {
                obj.callDataSmartContract = defaultConfigCallDataSmartContract();
            }
            if ((!obj.formInput || !isValidFormSchema(obj.formInput.schema)) && routineType in defaultConfigFormInputMap) {
                obj.formInput = defaultConfigFormInputMap[routineType]();
            }
            if ((!obj.formOutput || !isValidFormSchema(obj.formOutput.schema)) && routineType in defaultConfigFormOutputMap) {
                obj.formOutput = defaultConfigFormOutputMap[routineType]();
            }
            if (!obj.graph) {
                // Add if needed
            }
        }
        return new RoutineVersionConfig(obj);
    }

    serialize(mode: StringifyMode): string {
        return stringifyObject(this.export(), mode);
    }

    export(): RoutineVersionConfigObject {
        return {
            __version: this.__version,
            callDataApi: this.callDataApi?.export(),
            callDataGenerate: this.callDataGenerate?.export(),
            callDataSmartContract: this.callDataSmartContract?.export(),
            formInput: this.formInput?.export(),
            formOutput: this.formOutput?.export(),
            graph: this.graph?.export(),
        };
    }
}

class CallDataApiConfig {
    __version: string;
    schema: ConfigCallDataApi;

    constructor(data: CallDataApiConfigObject) {
        this.__version = data.__version ?? LATEST_VERSION;
        this.schema = data.schema;
    }

    export(): CallDataApiConfigObject {
        return {
            __version: this.__version,
            schema: this.schema,
        };
    }

    // Add API-specific methods here
}

export class CallDataGenerateConfig {
    __version: string;
    schema: ConfigCallDataGenerate;

    constructor(data: CallDataGenerateConfigObject) {
        this.__version = data.__version ?? LATEST_VERSION;
        this.schema = data.schema;
    }

    export(): CallDataGenerateConfigObject {
        return {
            __version: this.__version,
            schema: this.schema,
        };
    }

    // Add Generate-specific methods here
}

class CallDataSmartContractConfig {
    __version: string;
    schema: ConfigCallDataSmartContract;

    constructor(data: CallDataSmartContractConfigObject) {
        this.__version = data.__version ?? LATEST_VERSION;
        this.schema = data.schema;
    }

    export(): CallDataSmartContractConfigObject {
        return {
            __version: this.__version,
            schema: this.schema,
        };
    }

    // Add SmartContract-specific methods here
}

/**
 * Represents the configuration for form input fields in a routine.
 */
class FormInputConfig {
    __version: string;
    schema: FormSchema;

    constructor(data: FormInputConfigObject) {
        this.__version = data.__version ?? LATEST_VERSION;
        this.schema = data.schema;
    }

    /**
     * Get the names of all required and optional inputs in the form input schema.
     * @returns The names of all required and optional inputs in the form input schema.
     */
    getInputNames(): string[] {
        return this.schema.elements
            .filter(element => !Object.prototype.hasOwnProperty.call(element, "fieldName"))
            .map(element => (element as FormInputBase).fieldName);
    }

    export(): FormInputConfigObject {
        return {
            __version: this.__version,
            schema: this.schema,
        };
    }
}

/**
 * Represents the configuration for form output fields in a routine.
 */
class FormOutputConfig {
    __version: string;
    schema: FormSchema;

    constructor(data: FormOutputConfigObject) {
        this.__version = data.__version ?? LATEST_VERSION;
        this.schema = data.schema;
    }

    /**
     * Get the names of all required and optional outputs in the form output schema.
     * @returns The names of all required and optional outputs in the form output schema.
     */
    getOutputNames(): string[] {
        return this.schema.elements
            .filter(element => !Object.prototype.hasOwnProperty.call(element, "fieldName"))
            .map(element => (element as FormInputBase).fieldName);
    }

    export(): FormOutputConfigObject {
        return {
            __version: this.__version,
            schema: this.schema,
        };
    }
}

/**
 * Represents the configuration for a graph in a routine.
 */
export abstract class GraphConfig {
    __version: string;
    __type: GraphType;

    constructor(data: GraphBaseConfigObject) {
        this.__version = data.__version ?? LATEST_VERSION;
        this.__type = data.__type;
    }

    abstract export(): GraphConfigObject;

    static create(data: GraphConfigObject): GraphConfig {
        switch (data.__type) {
            case "BPMN-2.0":
                return new GraphBpmnConfig(data);
            default:
                throw new Error(`Unsupported __type: ${(data as { __type: string }).__type}`);
        }
    }

    /**
     * Get a map of input names for a node in the graph to the name of the input in the subroutine's formInput.
     *
     * @param nodeId The ID of the node in the graph.
     * @returns A map of input names for the node to the name of the input in the subroutine's formInput.
     */
    abstract getIONamesToSubroutineInputNames(nodeId: string): Record<string, string>;

    /**
     * Get a map of output names for a node in the graph to the name of the output in the subroutine's formOutput.
     *
     * @param nodeId The ID of the node in the graph.
     * @returns A map of output names for the node to the name of the output in the subroutine's formOutput.
     */
    abstract getIONamesToSubroutineOutputNames(nodeId: string): Record<string, string>;

    /**
     * Get a map of input names for the root of a graph to the overall routine's input names.
     * 
     * In other words, these are the inputs used when starting the routine, and not inputs 
     * passed into one of the nodes in its graph.
     * 
     * NOTE: The keys in the result should be composite keys that look like `root.${inputName}`, 
     * so we can distinguish these inputs from node inputs that may share the same name.
     * 
     * @returns A map of input names for the root of the graph to the overall routine's input names.
     */
    abstract getRootIONamesToRoutineInputNames(): Record<string, string>;

    /**
     * Get a map of output names for the root of a graph to the overall routine's output names.
     * 
     * In other words, these are the outputs returned from the completion of a routine.
     * 
     * NOTE: The keys in the result should be composite keys that look like `root.${outputName}`, 
     * so we can distinguish these outputs from node outputs that may share the same name.
     * 
     * @returns A map of output names for the root of the graph to the overall routine's output names.
     */
    abstract getRootIONamesToRoutineOutputNames(): Record<string, string>;
}

export class GraphBpmnConfig extends GraphConfig {
    schema: BpmnSchema;
    rootInputPrefix = "root";

    constructor(data: GraphBpmnConfigObject) {
        super(data);
        this.schema = data.schema;
    }

    export(): GraphBpmnConfigObject {
        return {
            __version: this.__version,
            __type: this.__type as GraphType,
            schema: this.schema,
        };
    }

    getIONamesToSubroutineInputNames(nodeId: string): Record<string, string> {
        const activity = this.schema.activityMap[nodeId];
        if (!activity) {
            return {};
        }
        return activity.inputMap;
    }

    getIONamesToSubroutineOutputNames(nodeId: string): Record<string, string> {
        const activity = this.schema.activityMap[nodeId];
        if (!activity) {
            return {};
        }
        return activity.outputMap;
    }

    getRootIONamesToRoutineInputNames(): Record<string, string> {
        const inputMap = this.schema.rootContext.inputMap;
        return Object.fromEntries(Object.entries(inputMap).map(([key, value]) => [
            `${this.rootInputPrefix}.${key}`,
            value,
        ]));
    }

    getRootIONamesToRoutineOutputNames(): Record<string, string> {
        const outputMap = this.schema.rootContext.outputMap;
        return Object.fromEntries(Object.entries(outputMap).map(([key, value]) => [
            `${this.rootInputPrefix}.${key}`,
            value,
        ]));
    }
}
