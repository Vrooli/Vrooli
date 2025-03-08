import i18next from "i18next";
import { LlmTask, RoutineType, type RoutineVersion } from "../../api/types.js";
import { HttpMethod } from "../../consts/api.js";
import { type PassableLogger } from "../../consts/commonTypes.js";
import { InputType } from "../../consts/model.js";
import { DEFAULT_LANGUAGE } from "../../consts/ui.js";
import { type FormInputBase, type FormSchema } from "../../forms/types.js";
import { uuid } from "../../id/uuid.js";
import { getDotNotationValue } from "../../utils/objects.js";
import { BotStyle, Id, type SubroutineIOMapping } from "../types.js";
import { type LlmModel } from "./bot.js";
import { CodeVersionConfigObject, JsonSchema } from "./code.js";
import { parseObject, stringifyObject, type StringifyMode } from "./utils.js";

export const LATEST_ROUTINE_CONFIG_VERSION = "1.0";
const DEFAULT_STRINGIFY_MODE: StringifyMode = "json";

/**
 * Represents how to perform an action within Vrooli.
 */
export interface ConfigCallDataAction {
    /**
     * The task to perform.
     */
    task: LlmTask;
    /**
     * JSON template to use for generating the task input. 
     * 
     * This should match the shape used to perform the task through the API (e.g. ApiCreateInput, NoteSearchInput, etc.)
     * 
     * Use double curly braces to reference routine inputs and special functions/values.
     * Available data:
     * - Routine inputs: `{{input.routineInputName}}`
     * - UUID generator: `{{uuid()}}`
     *     NOTE: To reference the same UUID for multiple fields, pass in a "seed" value (e.g. `{{uuid(123)}}`). 
     *     This isn't actually a seed in the sense that the UUID will be the same each time, but it will be the same
     *     within this specific template transformation.
     * - Main user language: `{{userLanguage}}`
     * - All user languages: `{{userLanguages}}`
     * - Current date/time: `{{now()}}`
     * - Random number generator: `{{random()}}`
     *     NOTE: To control the range of the random number, pass in a min and max value (e.g. `{{random(1, 10)}}`)
     * 
     * Full example:
     * {
     *     "id": "{{uuid()}}",
     *     "isPrivate": "{{input.isPrivate}}",
     *     "tagsCreate": [
     *         {
     *             "id": "{{uuid()}}",
     *             "translationsCreate": [
     *                 {
     *                     "id": "{{uuid()}}",
     *                     "language": "{{userLanguage}}",
     *                     "name": "{{input.name}}"
     *                 }
     *             ]
     *         }
     *     ],
     *     "versionsCreate": [
     *         {
     *             "id": "{{uuid()}}",
     *             "isPrivate": "{{input.isPrivate}}",
     *             "translationsCreate": [
     *                 {
     *                     "id": "{{uuid()}}",
     *                     "language": "{{userLanguage}}",
     *                     "name": "{{input.name}}"
     *                 }
     *             ]
     *         }
     *     ]
     * }
     */
    inputTemplate: string;
    /**
     * Mapping from routine output names to task output fields.
     * 
     * For example, { "routineOutputX": "id", "routineOutputY": "child[0].name" }
     * This means the 'id' field from the task output will be assigned to "routineOutputX",
     * and the 'name' field from the first child of the task output will be assigned to "routineOutputY".
     * 
     * NOTE: Supports dot notation.
     */
    outputMapping: Record<string, string>;
}

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
 * Represents how to run sandboxed code.
 * 
 * Can be applied to any attached code, whether it's for data transformation, 
 * smart contract execution, or other purposes.
 */
export interface ConfigCallDataCode {
    /** 
     * Template to map routine inputs to the sandbox input.
     * 
     * Examples:
     * - `{ "input": "{{routineInputA}}" }`
     * - `["{{routineInputA}}", "{{routineInputB}}"]`
     * - `{ "inputA": "{{routineInputA}}", "inputB": "{{routineInputB}}" }`
     */
    inputTemplate: object | string[];
    /**
     * Maps sandbox output formats (since there could be multiple) to routine outputs.
     */
    outputMappings: Array<{
        schemaIndex: number; // Index of the schema in outputConfig this mapping applies to
        mapping: { [routineOutputName: string]: string }; // Routine output names to output paths
    }>;
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

export type CallDataActionConfigObject = {
    __version: string;
    schema: ConfigCallDataAction;
};
export type CallDataApiConfigObject = {
    __version: string;
    schema: ConfigCallDataApi;
};
export type CallDataCodeConfigObject = {
    __version: string;
    schema: ConfigCallDataCode;
};
export type CallDataGenerateConfigObject = {
    __version: string;
    schema: ConfigCallDataGenerate;
};
export type CallDataSmartContractConfigObject = {
    __version: string;
    schema: ConfigCallDataSmartContract;
};

export type FormInputConfigObject = {
    __version: string;
    schema: FormSchema;
};
export type FormOutputConfigObject = {
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
export type GraphConfigObject = GraphBpmnConfigObject;

/**
 * Represents the configuration for a routine version. 
 * Includes configs for calling another service, I/O, and more.
 */
export type RoutineVersionConfigObject = {
    /** Store the version number for future compatibility */
    __version: string;
    /** Config for calling an action */
    callDataAction?: CallDataActionConfigObject;
    /** Config for calling an API */
    callDataApi?: CallDataApiConfigObject;
    /** Config for calling sandboxed code */
    callDataCode?: CallDataCodeConfigObject;
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

function defaultConfigCallDataAction(): CallDataActionConfigObject {
    return {
        __version: LATEST_ROUTINE_CONFIG_VERSION,
        schema: {
            task: LlmTask.RoutineFind,
            inputTemplate: "",
            outputMapping: {},
        },
    };
}
function defaultConfigCallDataApi(): CallDataApiConfigObject {
    return {
        __version: LATEST_ROUTINE_CONFIG_VERSION,
        schema: {
            endpoint: "",
            method: "GET",
        },
    };
}
function defaultConfigCallDataCode(): CallDataCodeConfigObject {
    return {
        __version: LATEST_ROUTINE_CONFIG_VERSION,
        schema: {
            inputTemplate: {},
            outputMappings: [],
        },
    };
}
function defaultConfigCallDataGenerate(): CallDataGenerateConfigObject {
    return {
        __version: LATEST_ROUTINE_CONFIG_VERSION,
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
        __version: LATEST_ROUTINE_CONFIG_VERSION,
        schema: {
            contractAddress: "",
            chain: "ethereum",
            methodName: "",
        },
    };
}

function defaultSchemaInput(): FormInputConfigObject {
    return {
        __version: LATEST_ROUTINE_CONFIG_VERSION,
        schema: {
            containers: [],
            elements: [],
        },
    };
}

function defaultSchemaOutput(): FormOutputConfigObject {
    return {
        __version: LATEST_ROUTINE_CONFIG_VERSION,
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
        __version: LATEST_ROUTINE_CONFIG_VERSION,
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

export const defaultConfigFormInputMap = {
    [RoutineType.Action]: () => defaultSchemaInput(),
    [RoutineType.Api]: () => defaultSchemaInput(),
    [RoutineType.Code]: () => defaultSchemaInput(),
    [RoutineType.Data]: () => defaultSchemaInput(),
    [RoutineType.Generate]: () => defaultSchemaInput(),
    [RoutineType.Informational]: () => defaultSchemaInput(),
    [RoutineType.MultiStep]: () => defaultSchemaInput(),
    [RoutineType.SmartContract]: () => defaultSchemaInput(),
};

export const defaultConfigFormOutputMap = {
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
    callDataAction?: CallDataActionConfig;
    callDataApi?: CallDataApiConfig;
    callDataCode?: CallDataCodeConfig;
    callDataGenerate?: CallDataGenerateConfig;
    callDataSmartContract?: CallDataSmartContractConfig;
    formInput?: FormInputConfig;
    formOutput?: FormOutputConfig;
    graph?: GraphConfig;

    constructor(data: RoutineVersionConfigObject) {
        this.__version = data.__version ?? LATEST_ROUTINE_CONFIG_VERSION;
        this.callDataAction = data.callDataAction ? new CallDataActionConfig(data.callDataAction) : undefined;
        this.callDataApi = data.callDataApi ? new CallDataApiConfig(data.callDataApi) : undefined;
        this.callDataCode = data.callDataCode ? new CallDataCodeConfig(data.callDataCode) : undefined;
        this.callDataGenerate = data.callDataGenerate ? new CallDataGenerateConfig(data.callDataGenerate) : undefined;
        this.callDataSmartContract = data.callDataSmartContract ? new CallDataSmartContractConfig(data.callDataSmartContract) : undefined;
        this.formInput = data.formInput ? new FormInputConfig(data.formInput) : undefined;
        this.formOutput = data.formOutput ? new FormOutputConfig(data.formOutput) : undefined;
        this.graph = data.graph ? GraphConfig.create(data.graph) : undefined;
    }

    static deserialize(
        { config, routineType }: Pick<RoutineVersion, "config" | "routineType">,
        logger: PassableLogger,
        { mode = DEFAULT_STRINGIFY_MODE, useFallbacks = true }: { mode?: StringifyMode, useFallbacks?: boolean } = {},
    ): RoutineVersionConfig {
        let obj = config ? parseObject<RoutineVersionConfigObject>(config, mode, logger) : null;
        if (!obj) {
            obj = { __version: LATEST_ROUTINE_CONFIG_VERSION };
        }
        if (useFallbacks) {
            if (!obj.callDataAction && routineType === RoutineType.Action) {
                obj.callDataAction = defaultConfigCallDataAction();
            }
            if (!obj.callDataApi && routineType === RoutineType.Api) {
                obj.callDataApi = defaultConfigCallDataApi();
            }
            if (!obj.callDataCode && routineType === RoutineType.Code) {
                obj.callDataCode = defaultConfigCallDataCode();
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
            callDataAction: this.callDataAction?.export(),
            callDataApi: this.callDataApi?.export(),
            callDataCode: this.callDataCode?.export(),
            callDataGenerate: this.callDataGenerate?.export(),
            callDataSmartContract: this.callDataSmartContract?.export(),
            formInput: this.formInput?.export(),
            formOutput: this.formOutput?.export(),
            graph: this.graph?.export(),
        };
    }
}

/**
 * Configuration that's passed between methods for template processing.
 */
type TemplateConfig = {
    inputs: SubroutineIOMapping["inputs"];
    userLanguages: string[];
    seededUUIDs: Record<string, string>;
};

export class CallDataActionConfig {
    __version: string;
    schema: ConfigCallDataAction;

    constructor(data: CallDataActionConfigObject) {
        this.__version = data.__version ?? LATEST_ROUTINE_CONFIG_VERSION;
        this.schema = data.schema;
    }

    private getPlaceholderValue(
        placeholder: string,
        { inputs, userLanguages, seededUUIDs }: TemplateConfig,
    ): unknown {
        // Handle routine inputs: input.routineInputName
        if (placeholder.startsWith("input.")) {
            const inputName = placeholder.slice("input.".length);
            const inputValue = inputs[inputName]?.value;
            if (inputValue === undefined) {
                throw new Error(`Input "${inputName}" not found in SubroutineIOMapping`);
            }
            return inputValue;
        }

        // Handle special functions
        switch (placeholder) {
            case "userLanguage":
                return userLanguages[0] || DEFAULT_LANGUAGE;
            case "userLanguages":
                return userLanguages;
            case "now()":
                return new Date().toISOString();
        }

        // Handle uuid()
        if (placeholder.startsWith("uuid")) {
            // Check if there are any arguments
            const args = placeholder.slice("uuid(".length, -")".length).trim();
            if (args) {
                const seededId = seededUUIDs[args];
                if (seededId) {
                    return seededId;
                }
                const id = uuid();
                seededUUIDs[args] = id;
                return id;
            }
            return uuid();
        }

        // Handle random()
        if (placeholder.startsWith("random")) {
            const args = placeholder.slice("random(".length, -")".length).trim();
            if (args && args.includes(",")) {
                const [min, max] = args.split(",").map(Number);
                if (min === undefined || max === undefined || isNaN(min) || isNaN(max)) {
                    throw new Error(`Invalid arguments for random(): ${args}`);
                }
                return Math.floor(Math.random() * (max - min + 1)) + min;
            }
            return Math.random();
        }

        throw new Error(`Unknown placeholder: ${placeholder}`);
    }

    /**
     * Helper function to replace placeholders in a string with actual values.
     * 
     * @param str The string containing placeholders
     * @param config Configuration for template processing
     * @returns Value with all placeholders replaced
     */
    private replacePlaceholders(
        str: string,
        config: TemplateConfig,
    ): unknown {
        console.log("start", str);

        const placeholderRegex = /\{\{([^}]+)\}\}/g;
        const matches = str.match(placeholderRegex);

        // If the entire string is a single placeholder, return the raw value
        if (matches && matches.length === 1 && str.trim() === matches[0]) {
            const placeholder = matches[0].slice("{{".length, -"}}".length).trim();
            return this.getPlaceholderValue(placeholder, config);
        }

        // Otherwise, perform string interpolation
        return str.replace(placeholderRegex, (_, placeholder) => {
            const value = this.getPlaceholderValue(placeholder.trim(), config);
            return typeof value === "string" ? value : JSON.stringify(value);
        });
    }

    /**
     * Helper function to recursively process the template and replace placeholders
     * @param template The template to process (can be string, array, or object)
     * @param config Configuration for template processing
     * @returns Processed template with all placeholders replaced
     */
    private processTemplate(
        template: unknown,
        config: TemplateConfig,
    ): unknown {
        if (typeof template === "string") {
            return this.replacePlaceholders(template, config);
        } else if (Array.isArray(template)) {
            return template.map(item => this.processTemplate(item, config));
        } else if (typeof template === "object" && template !== null) {
            const result: Record<string, unknown> = {};
            for (const key in template) {
                result[key] = this.processTemplate(template[key], config);
            }
            return result;
        }
        return template; // Return primitives (numbers, booleans, etc.) as-is
    }

    /**
     * Converts the inputTemplate string into an actual input object for the task
     * @param ioMapping The mapping of input and output names to their values and display info
     * @param userLanguages List of user's preferred languages
     * @returns The processed input object with all placeholders replaced
     * @throws Error if inputTemplate is invalid JSON or contains unknown placeholders
     */
    buildTaskInput(
        ioMapping: SubroutineIOMapping,
        userLanguages: string[],
    ): Record<string, unknown> {
        const config: TemplateConfig = {
            inputs: ioMapping.inputs,
            userLanguages,
            seededUUIDs: {},
        };

        // Parse the inputTemplate string into a JavaScript object
        let templateObj;
        try {
            templateObj = JSON.parse(this.schema.inputTemplate);
        } catch (error) {
            throw new Error(`Failed to parse inputTemplate: ${(error as Error).message}`);
        }

        // Process the template to replace all placeholders
        return this.processTemplate(templateObj, config) as Record<string, unknown>;
    }

    /**
     * Applies the output mapping to the action's result and updates the ioMappings.outputs.
     * 
     * @param ioMappings The ioMappings object containing the outputs to update.
     * @param result The result object from the action.
     */
    public parseActionResult(ioMappings: SubroutineIOMapping, result: { payload?: unknown }): void {
        // If there is no payload, return
        if (result.payload === undefined) {
            return;
        }

        // Iterate over each mapping in outputMapping
        for (const [outputName, path] of Object.entries(this.schema.outputMapping)) {
            // Prefix path with "payload." (removing leading dot if present)
            let prefixedPath = path;
            // Handle array indices
            if (path.startsWith("[")) {
                prefixedPath = "payload" + path;
            }
            // Handle object paths
            else {
                prefixedPath = "payload" + (path.startsWith(".") ? path : "." + path);
            }
            // Extract the value from the result using the dot notation path
            const value = getDotNotationValue(result, prefixedPath);
            if (value === undefined) {
                continue;
            }

            // Check if the output exists in ioMappings.outputs
            const output = ioMappings.outputs[outputName];
            if (output) {
                output.value = value;
            }
        }
    }

    export(): CallDataActionConfigObject {
        return {
            __version: this.__version,
            schema: this.schema,
        };
    }
}

class CallDataApiConfig {
    __version: string;
    schema: ConfigCallDataApi;

    constructor(data: CallDataApiConfigObject) {
        this.__version = data.__version ?? LATEST_ROUTINE_CONFIG_VERSION;
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

type SandboxInput = {
    input: unknown;
    shouldSpreadInput: boolean;
}

export class CallDataCodeConfig {
    __version: string;
    schema: ConfigCallDataCode;

    constructor(data: CallDataCodeConfigObject) {
        this.__version = data.__version ?? LATEST_ROUTINE_CONFIG_VERSION;
        this.schema = data.schema;
    }

    /**
     * Builds the input for the sandboxed code by mapping routine inputs to the expected structure.
     * 
     * @param ioMapping Mapping of routine input names to their current values.
     * @param inputConfig The code’s input configuration (e.g., shouldSpread, inputSchema).
     * @returns The constructed input, typed according to inputConfig.shouldSpread.
     * @throws Error if the routine's input template is incompatible with the code's input definition.
     */
    buildSandboxInput(
        ioMapping: Pick<SubroutineIOMapping, "inputs">,
        inputConfig: CodeVersionConfigObject["inputConfig"],
    ): SandboxInput {
        const template = this.schema.inputTemplate;
        let input: unknown;

        if (inputConfig.shouldSpread) {
            // When shouldSpread is true, template must be an array
            if (!Array.isArray(template)) {
                throw new Error("Input template must be an array when shouldSpread is true");
            }
            // Type assertion to unknown[] since we know it's an array and shouldSpread is true
            input = this.replacePlaceholders(template, ioMapping);
        } else {
            // When shouldSpread is false, template can be any type
            input = this.replacePlaceholders(template, ioMapping);
        }

        return {
            input,
            shouldSpreadInput: inputConfig.shouldSpread,
        };
    }

    /**
     * Recursively replaces placeholders in the input template with values from ioMapping.
     * 
     * @param template The input template (string, array, or object).
     * @param ioMapping Mapping of routine input names to their values.
     * @returns The template with all placeholders substituted.
     */
    private replacePlaceholders(template: object | string[] | string, ioMapping: Pick<SubroutineIOMapping, "inputs">): unknown {
        if (typeof template === "string") {
            // Check if the string is a whole placeholder (e.g., "{{routineInputC}}")
            const wholePlaceholderMatch = template.match(/^{{(.+)}}$/);
            if (wholePlaceholderMatch) {
                const inputName = wholePlaceholderMatch[1] ?? ""; // Extract the input name (e.g., "routineInputC")
                const inputValue = ioMapping.inputs[inputName]?.value;
                if (inputValue === undefined) {
                    throw new Error(`Input "${inputName}" not found in ioMapping or missing value`);
                }
                return inputValue; // Return the raw value (e.g., 42 as a number)
            } else {
                // Handle embedded placeholders (e.g., "h: {{routineInputH}}")
                return template.replace(/{{(.+?)}}/g, (match, inputName) => {
                    const inputValue = ioMapping.inputs[inputName]?.value;
                    if (inputValue === undefined) {
                        throw new Error(`Input "${inputName}" not found in ioMapping or missing value`);
                    }
                    // Custom stringification logic for embedded placeholders
                    if (inputValue instanceof Date) {
                        return inputValue.toISOString(); // ISO format for dates
                    } else if (typeof inputValue === "object" && inputValue !== null) {
                        return JSON.stringify(inputValue); // JSON string for objects
                    } else {
                        return String(inputValue); // Default string conversion for primitives
                    }
                });
            }
        } else if (Array.isArray(template)) {
            // Process each element in the array
            return template.map((item) => this.replacePlaceholders(item, ioMapping));
        } else if (typeof template === "object" && template !== null) {
            // Process each value in the object
            const result: { [key: string]: any } = {};
            for (const key in template) {
                result[key] = this.replacePlaceholders(template[key], ioMapping);
            }
            return result;
        } else {
            // Return non-string/array/object values as is (e.g., null, numbers)
            return template;
        }
    }

    /**
     * Maps the sandbox output to the routine's outputs based on outputConfig and outputMappings.
     * @param runOutput Output from the sandboxed code execution.
     * @param ioMapping Routine's IO mapping to update with output values.
     * @param outputConfig Expected output schemas from CodeVersionConfig.
     */
    parseSandboxOutput(
        runOutput: { error?: string, output?: unknown },
        ioMapping: Pick<SubroutineIOMapping, "outputs">,
        outputConfig: CodeVersionConfigObject["outputConfig"],
    ): void {
        if (runOutput.error) {
            throw new Error(`Code execution error: ${runOutput.error}`);
        }
        if (runOutput.output === undefined) {
            return;
        }

        // Normalize output config and output mappings to be arrays
        const outputConfigs: readonly JsonSchema[] = Array.isArray(outputConfig) ? outputConfig : [outputConfig];
        const outputMappings = Array.isArray(this.schema.outputMappings) ? this.schema.outputMappings : [this.schema.outputMappings];
        // If the arrays are not the same length, we're not handling every case. Throw an error.
        if (outputMappings.length !== outputConfigs.length) {
            throw new Error("Output mappings and output configs must be the same length");
        }

        // Step 1: Set all routine outputs to null
        for (const outputName in ioMapping.outputs) {
            const output = ioMapping.outputs[outputName];
            if (output) {
                output.value = null;
            }
        }

        // Step 2: Find the matching schema index
        let matchingSchemaIndex: number | null = null;
        for (let i = 0; i < outputConfigs.length; i++) {
            if (this.validateOutput(runOutput.output, outputConfigs[i])) {
                matchingSchemaIndex = i;
                break; // Assume output matches only one schema
            }
        }

        // Step 3: If no schema matches, throw an error
        if (matchingSchemaIndex === null) {
            throw new Error("Output does not match any expected schema");
        }

        // Step 4: Apply mappings only for the matching schema index
        const mappingsToApply = outputMappings.filter(mapping => mapping.schemaIndex === matchingSchemaIndex);
        for (const mapping of mappingsToApply) {
            for (const [routineOutputName, path] of Object.entries(mapping.mapping)) {
                // Prefix path with "output." (removing leading dot if present)
                let prefixedPath = path;
                // Handle array indices
                if (path.startsWith("[")) {
                    prefixedPath = "output" + path;
                }
                // Handle object paths
                else {
                    prefixedPath = "output" + (path.startsWith(".") ? path : "." + path);
                }
                const value = getDotNotationValue(runOutput, prefixedPath);
                const outputObject = ioMapping.outputs[routineOutputName];
                if (!outputObject) {
                    throw new Error(`Output "${routineOutputName}" not found in ioMapping`);
                }
                outputObject.value = value;
            }
        }
    }

    /**
     * Validates if the output matches the given schema.
     * 
     * NOTE: This is a simplified validation check. Add a more robust check if needed.
     * 
     * @param output The output to validate.
     * @param schema The schema to validate against.
     * @returns True if the output matches the schema, false otherwise.
     */
    private validateOutput(output: unknown, schema: JsonSchema | undefined): boolean {
        if (!schema) {
            return false;
        }
        switch (schema.type) {
            case "string":
                return typeof output === "string";
            case "number":
                return typeof output === "number";
            case "boolean":
                return typeof output === "boolean";
            case "null":
                return output === null;
            case "array":
                if (!Array.isArray(output)) {
                    return false;
                }
                if (schema.minItems !== undefined && output.length < schema.minItems) {
                    return false;
                }
                if (schema.maxItems !== undefined && output.length > schema.maxItems) {
                    return false;
                }
                // if (schema.items) {
                //     for (const item of output) {
                //         if (!this.validateOutput(item, schema.items)) {
                //             return false;
                //         }
                //     }
                // }
                // return true;
                return true;
            case "object":
                if (typeof output !== "object" || output === null) {
                    return false;
                }
                if (schema.properties) {
                    for (const prop in schema.properties) {
                        const propSchema = schema.properties[prop];
                        const propValue = output[prop];
                        if (!this.validateOutput(propValue, propSchema)) {
                            return false;
                        }
                    }
                }
                return true;
            default:
                return false; // Unknown schema type
        }
    }

    export(): CallDataCodeConfigObject {
        return {
            __version: this.__version,
            schema: this.schema,
        };
    }
}

export class CallDataGenerateConfig {
    __version: string;
    schema: ConfigCallDataGenerate;

    constructor(data: CallDataGenerateConfigObject) {
        this.__version = data.__version ?? LATEST_ROUTINE_CONFIG_VERSION;
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
        this.__version = data.__version ?? LATEST_ROUTINE_CONFIG_VERSION;
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
        this.__version = data.__version ?? LATEST_ROUTINE_CONFIG_VERSION;
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
        this.__version = data.__version ?? LATEST_ROUTINE_CONFIG_VERSION;
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
        this.__version = data.__version ?? LATEST_ROUTINE_CONFIG_VERSION;
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
