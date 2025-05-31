import { type ResourceVersion } from "../../api/types.js";
import { type PassableLogger } from "../../consts/commonTypes.js";
import { type CodeLanguage } from "../../consts/ui.js";
import { BaseConfig, type BaseConfigObject } from "./base.js";

const LATEST_CONFIG_VERSION = "1.0";

/**
 * Represents the blockchain contract details associated with a code version.
 */
export interface ContractDetails {
    address?: string;           // e.g., Ethereum address
    blockchain: string;        // e.g., "ethereum", "polygon", "solana"
    contractType?: string;      // e.g., "ERC721", "Custom"
    hash?: string;              // Deployment or transaction hash
    isAddressVerified?: boolean;
    isHashVerified?: boolean;
}

/**
 * Represents the possible types in a JSON Schema.
 */
type SchemaType = "null" | "string" | "number" | "boolean" | "array" | "object";

/**
 * Base schema interface with common properties.
 */
interface BaseSchema {
    type: SchemaType;
}

/**
 * Schema for null values.
 */
interface NullSchema extends BaseSchema {
    type: "null";
}

/**
 * Schema for strings.
 */
interface StringSchema extends BaseSchema {
    type: "string";
    // Add string-specific properties if needed, e.g., pattern, minLength
}

/**
 * Schema for numbers.
 */
interface NumberSchema extends BaseSchema {
    type: "number";
    // Add number-specific properties if needed, e.g., minimum, maximum
}

/**
 * Schema for booleans.
 */
interface BooleanSchema extends BaseSchema {
    type: "boolean";
}

/**
 * Schema for arrays, supporting both uniform arrays and tuples.
 */
interface ArraySchema extends BaseSchema {
    type: "array";
    items: JsonSchema | JsonSchema[];  // Single schema for uniform arrays, array for tuples
    minItems?: number;
    maxItems?: number;
}

/**
 * Schema for objects.
 */
interface ObjectSchema extends BaseSchema {
    type: "object";
    properties?: { [key: string]: JsonSchema };
    required?: string[];
}

/**
 * Union type for all possible JSON Schemas.
 */
export type JsonSchema = NullSchema | StringSchema | NumberSchema | BooleanSchema | ArraySchema | ObjectSchema;

/**
 * Code input definition for spread input (expects an array to spread into arguments).
 * 
 * Use this when the input looks like:
 * 
 * 1. function(a, b, c)
 * 2. function(...args)
 * 3. function(a, ...args)
 */
interface SpreadInputDefinition {
    inputSchema: ArraySchema;
    shouldSpread: true;
}

/**
 * Code input definition for direct input (passes input as a single argument).
 * 
 * Use this when the input looks like:
 * 
 * 1. function (a)
 */
interface DirectInputDefinition {
    inputSchema: JsonSchema;   // Can be any schema
    shouldSpread: false;
}

/**
 * Union type for input definitions.
 */
export type CodeVersionInputDefinition = SpreadInputDefinition | DirectInputDefinition;

/**
 * Represents a test case for validating the code object.
 */
type CodeVersionTestCase = {
    /** Optional description of what the test case is checking */
    description?: string;
    /** Input data to pass to the code. Must conform to inputSchema */
    input: unknown;
    /** Expected output from the code. Must conform to one of the schemas in outputConfig */
    expectedOutput: unknown;
};

/**
 * Represents a test result for a code version.
 */
type CodeVersionTestCaseResult = {
    /** The description of the test case */
    description?: string;
    /** Whether the test case passed */
    passed: boolean;
    /** The error message if the test case failed */
    error?: string;
    /** The actual output from the code */
    actualOutput?: unknown;
}

/**
 * Represents all data that can be stored in a code's stringified config.
 * 
 * This is basically any data that doesn't need to be queried while searching 
 * (e.g. how to pass input to the code). Things like the code's content, language, 
 * etc. are commonly used to find codes, so it makes sense to store them in 
 * their own db columns (i.e. not here).
 */
export interface CodeVersionConfigObject extends BaseConfigObject {
    /** How to pass input to the code */
    inputConfig?: CodeVersionInputDefinition;
    /** 
     * How to parse output from the code.
     * 
     * Supports multiple schemas for different output types, 
     * such as a function that sometimes returns an object and sometimes return null.
     */
    outputConfig?: JsonSchema | JsonSchema[];
    /** Test cases to validate the code */
    testCases?: CodeVersionTestCase[];
    /** The code itself */
    content: string;
    /** Optional blockchain contract details */
    contractDetails?: ContractDetails;
}

/**
 * Top-level Code config that encapsulates all code-related data.
 */
export class CodeVersionConfig extends BaseConfig<CodeVersionConfigObject> {
    inputConfig?: CodeVersionConfigObject["inputConfig"];
    outputConfig?: CodeVersionConfigObject["outputConfig"];
    testCases?: CodeVersionConfigObject["testCases"];
    content: CodeVersionConfigObject["content"];
    contractDetails?: CodeVersionConfigObject["contractDetails"];

    codeLanguage: ResourceVersion["codeLanguage"];

    constructor({ config, codeLanguage }: { config: CodeVersionConfigObject, codeLanguage: ResourceVersion["codeLanguage"] }) {
        super(config);
        this.__version = config.__version ?? LATEST_CONFIG_VERSION;
        this.inputConfig = config.inputConfig ?? CodeVersionConfig.defaultInputConfig();
        this.outputConfig = config.outputConfig ?? CodeVersionConfig.defaultOutputConfig();
        this.testCases = config.testCases ?? CodeVersionConfig.defaultTestCases();
        this.content = config.content;
        this.contractDetails = config.contractDetails;
        this.codeLanguage = codeLanguage;
    }

    static parse(
        version: Pick<ResourceVersion, "config" | "codeLanguage">,
        logger: PassableLogger,
        opts?: { useFallbacks?: boolean },
    ): CodeVersionConfig {
        let parsedConfigObject: CodeVersionConfigObject | null | undefined;
        if (typeof version.config === "string") {
            try {
                parsedConfigObject = JSON.parse(version.config) as CodeVersionConfigObject;
                if (parsedConfigObject && typeof parsedConfigObject.content !== "string") {
                    logger.error("Parsed CodeVersionConfig string is missing or has invalid 'content'. Using empty string for content.", { data: version.config });
                    parsedConfigObject.content = "";
                }
            } catch (e) {
                logger.error("Failed to parse CodeVersionConfig string. Initializing with default content.", { error: e, data: version.config });
                parsedConfigObject = null;
            }
        } else {
            parsedConfigObject = version.config as CodeVersionConfigObject | null | undefined;
            if (parsedConfigObject && typeof parsedConfigObject.content !== "string") {
                logger.error("CodeVersionConfig object is missing or has invalid 'content'. Using empty string for content.", { data: version.config });
                parsedConfigObject.content = "";
            }
        }

        return super.parseBase<CodeVersionConfigObject, CodeVersionConfig>(
            parsedConfigObject,
            logger,
            (cfg) => {
                const finalConfig: CodeVersionConfigObject = {
                    ...cfg,
                    inputConfig: (opts?.useFallbacks ?? true)
                        ? (cfg.inputConfig ?? CodeVersionConfig.defaultInputConfig())
                        : cfg.inputConfig,
                    outputConfig: (opts?.useFallbacks ?? true)
                        ? (cfg.outputConfig ?? CodeVersionConfig.defaultOutputConfig())
                        : cfg.outputConfig,
                    testCases: (opts?.useFallbacks ?? true)
                        ? (cfg.testCases ?? CodeVersionConfig.defaultTestCases())
                        : cfg.testCases,
                };

                if (typeof finalConfig.content !== "string") {
                    logger.error("Content was not available in parsed config; defaulting to empty string in final factory stage.");
                    finalConfig.content = "";
                }

                return new CodeVersionConfig({ config: finalConfig, codeLanguage: version.codeLanguage });
            },
        );
    }

    static default({ codeLanguage, initialContent }: { codeLanguage: ResourceVersion["codeLanguage"], initialContent: string }): CodeVersionConfig {
        return new CodeVersionConfig({
            config: {
                __version: LATEST_CONFIG_VERSION,
                resources: [],
                inputConfig: CodeVersionConfig.defaultInputConfig(),
                outputConfig: CodeVersionConfig.defaultOutputConfig(),
                testCases: CodeVersionConfig.defaultTestCases(),
                content: initialContent,
                contractDetails: undefined,
            },
            codeLanguage,
        });
    }

    override export(): CodeVersionConfigObject {
        return {
            ...super.export(),
            inputConfig: this.inputConfig,
            outputConfig: this.outputConfig,
            testCases: this.testCases,
            content: this.content,
            contractDetails: this.contractDetails,
        };
    }

    static defaultInputConfig(): CodeVersionConfigObject["inputConfig"] {
        return {
            inputSchema: { type: "object" },
            shouldSpread: false,
        };
    }

    static defaultOutputConfig(): CodeVersionConfigObject["outputConfig"] {
        return [];
    }

    static defaultTestCases(): CodeVersionConfigObject["testCases"] {
        return [];
    }

    /**
     * Runs the test cases against the code.
     * 
     * @param runSandbox Function that runs the code in a sandbox.
     * @param logger Logger to use for logging.
     * @returns A list of test results.
     */
    async runTestCases(
        runSandbox: (input: {
            code: string;
            codeLanguage: CodeLanguage;
            input?: unknown;
            shouldSpreadInput?: boolean;
        }) => Promise<{
            error?: string;
            output?: unknown;
        }>,
    ): Promise<CodeVersionTestCaseResult[]> {
        const results: CodeVersionTestCaseResult[] = [];

        // Iterate over each test case
        for (const testCase of this.testCases ?? []) {
            const { input, expectedOutput, description } = testCase;
            const shouldSpreadInput = this.inputConfig?.shouldSpread ?? false;

            try {
                const result = await runSandbox({
                    code: String.raw`${this.content}`,
                    codeLanguage: this.codeLanguage as CodeLanguage,
                    input,
                    shouldSpreadInput,
                });

                if (result.error) {
                    results.push({
                        description,
                        passed: false,
                        error: result.error,
                    });
                } else {
                    const actualOutput = result.output;
                    const passed = JSON.stringify(actualOutput) === JSON.stringify(expectedOutput);
                    if (passed) {
                        results.push({
                            description,
                            passed: true,
                        });
                    } else {
                        results.push({
                            description,
                            passed: false,
                            actualOutput,
                        });
                    }
                }
            } catch (error) {
                // Handle execution errors (e.g., runtime errors, unsupported language)
                results.push({
                    description,
                    passed: false,
                    error: error instanceof Error ? error.message : String(error),
                });
            }
        }

        return results;
    }
}
