import { type CodeVersionConfigObject, type JsonSchema, type CodeVersionInputDefinition, type ContractDetails } from "../../../shape/configs/code.js";
import { LATEST_CONFIG_VERSION } from "../../../shape/configs/utils.js";
import { type ConfigTestFixtures, mergeWithBaseDefaults } from "./baseConfigFixtures.js";

/**
 * Code configuration fixtures for testing code execution and validation
 */
export const codeConfigFixtures: ConfigTestFixtures<CodeVersionConfigObject> = {
    minimal: {
        __version: LATEST_CONFIG_VERSION,
        content: "function add(a, b) { return a + b; }"
    },
    
    complete: {
        __version: LATEST_CONFIG_VERSION,
        content: `// Comprehensive example function
function processData(input) {
    if (!input || typeof input !== 'object') {
        return { error: 'Invalid input' };
    }
    
    const { numbers = [], operation = 'sum' } = input;
    
    switch (operation) {
        case 'sum':
            return numbers.reduce((a, b) => a + b, 0);
        case 'multiply':
            return numbers.reduce((a, b) => a * b, 1);
        case 'average':
            return numbers.length > 0 ? numbers.reduce((a, b) => a + b, 0) / numbers.length : 0;
        default:
            return { error: 'Unknown operation' };
    }
}`,
        inputConfig: {
            inputSchema: {
                type: "object",
                properties: {
                    numbers: {
                        type: "array",
                        items: { type: "number" },
                        minItems: 0
                    },
                    operation: {
                        type: "string"
                    }
                },
                required: ["numbers"]
            },
            shouldSpread: false
        },
        outputConfig: [
            { type: "number" },
            {
                type: "object",
                properties: {
                    error: { type: "string" }
                },
                required: ["error"]
            }
        ],
        testCases: [
            {
                description: "Sum operation",
                input: { numbers: [1, 2, 3], operation: "sum" },
                expectedOutput: 6
            },
            {
                description: "Multiply operation",
                input: { numbers: [2, 3, 4], operation: "multiply" },
                expectedOutput: 24
            },
            {
                description: "Invalid operation",
                input: { numbers: [1, 2], operation: "invalid" },
                expectedOutput: { error: "Unknown operation" }
            }
        ],
        resources: [{
            link: "https://example.com/code-docs",
            usedFor: "OfficialWebsite",
            translations: [{
                language: "en",
                name: "Code Documentation",
                description: "Documentation for this code function"
            }]
        }]
    },
    
    withDefaults: {
        __version: LATEST_CONFIG_VERSION,
        content: "// Default empty function\nfunction main() { return null; }",
        inputConfig: {
            inputSchema: { type: "object" },
            shouldSpread: false
        },
        outputConfig: []
    },
    
    invalid: {
        missingVersion: {
            // Missing __version
            content: "function test() { return 42; }",
            inputConfig: {
                inputSchema: { type: "null" },
                shouldSpread: false
            }
        },
        invalidVersion: {
            __version: "0.5", // Invalid version
            content: "function test() { return 42; }"
        },
        malformedStructure: {
            __version: LATEST_CONFIG_VERSION,
            content: 123, // Wrong type - should be string
            inputConfig: "not an object" // Wrong type
        },
        invalidTypes: {
            __version: LATEST_CONFIG_VERSION,
            content: "function test() {}",
            testCases: "not an array", // Should be array
            outputConfig: {
                invalid: "schema" // Missing type property
            }
        }
    },
    
    variants: {
        simpleFunction: {
            __version: LATEST_CONFIG_VERSION,
            content: "function double(x) { return x * 2; }",
            inputConfig: {
                inputSchema: { type: "number" },
                shouldSpread: false
            },
            outputConfig: { type: "number" },
            testCases: [
                {
                    description: "Double positive number",
                    input: 5,
                    expectedOutput: 10
                },
                {
                    description: "Double negative number",
                    input: -3,
                    expectedOutput: -6
                }
            ]
        },
        
        spreadFunction: {
            __version: LATEST_CONFIG_VERSION,
            content: "function sum(...args) { return args.reduce((a, b) => a + b, 0); }",
            inputConfig: {
                inputSchema: {
                    type: "array",
                    items: { type: "number" }
                },
                shouldSpread: true
            },
            outputConfig: { type: "number" },
            testCases: [
                {
                    description: "Sum multiple arguments",
                    input: [1, 2, 3, 4, 5],
                    expectedOutput: 15
                },
                {
                    description: "Sum empty array",
                    input: [],
                    expectedOutput: 0
                }
            ]
        },
        
        smartContract: {
            __version: LATEST_CONFIG_VERSION,
            content: `// Solidity smart contract example
pragma solidity ^0.8.0;

contract SimpleStorage {
    uint256 private storedValue;
    
    function set(uint256 value) public {
        storedValue = value;
    }
    
    function get() public view returns (uint256) {
        return storedValue;
    }
}`,
            contractDetails: {
                blockchain: "ethereum",
                contractType: "SimpleStorage",
                address: "0x1234567890123456789012345678901234567890",
                hash: "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
                isAddressVerified: true,
                isHashVerified: true
            },
            inputConfig: {
                inputSchema: {
                    type: "object",
                    properties: {
                        method: { type: "string" },
                        value: { type: "number" }
                    },
                    required: ["method"]
                },
                shouldSpread: false
            },
            outputConfig: [
                { type: "number" },
                { type: "null" }
            ]
        },
        
        asyncFunction: {
            __version: LATEST_CONFIG_VERSION,
            content: `async function fetchData(url) {
    try {
        const response = await fetch(url);
        const data = await response.json();
        return { success: true, data };
    } catch (error) {
        return { success: false, error: error.message };
    }
}`,
            inputConfig: {
                inputSchema: { type: "string" },
                shouldSpread: false
            },
            outputConfig: {
                type: "object",
                properties: {
                    success: { type: "boolean" },
                    data: { type: "object" },
                    error: { type: "string" }
                },
                required: ["success"]
            }
        },
        
        arrayProcessor: {
            __version: LATEST_CONFIG_VERSION,
            content: `function processArray(arr, options = {}) {
    const { filter, map, reduce } = options;
    
    let result = arr;
    
    if (filter) {
        result = result.filter(eval(filter));
    }
    
    if (map) {
        result = result.map(eval(map));
    }
    
    if (reduce) {
        const [fn, initial] = reduce;
        result = result.reduce(eval(fn), initial);
    }
    
    return result;
}`,
            inputConfig: {
                inputSchema: {
                    type: "array",
                    items: [
                        {
                            type: "array",
                            items: { type: "number" }
                        },
                        {
                            type: "object",
                            properties: {
                                filter: { type: "string" },
                                map: { type: "string" },
                                reduce: {
                                    type: "array",
                                    items: [
                                        { type: "string" },
                                        { type: "number" }
                                    ],
                                    minItems: 2,
                                    maxItems: 2
                                }
                            }
                        }
                    ],
                    minItems: 1,
                    maxItems: 2
                },
                shouldSpread: true
            },
            outputConfig: [
                {
                    type: "array",
                    items: { type: "number" }
                },
                { type: "number" }
            ]
        },
        
        validationFunction: {
            __version: LATEST_CONFIG_VERSION,
            content: `function validateEmail(email) {
    const emailRegex = /^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$/;
    
    if (typeof email !== 'string') {
        return { valid: false, error: 'Email must be a string' };
    }
    
    if (email.length === 0) {
        return { valid: false, error: 'Email cannot be empty' };
    }
    
    if (!emailRegex.test(email)) {
        return { valid: false, error: 'Invalid email format' };
    }
    
    return { valid: true, email: email.toLowerCase() };
}`,
            inputConfig: {
                inputSchema: { type: "string" },
                shouldSpread: false
            },
            outputConfig: {
                type: "object",
                properties: {
                    valid: { type: "boolean" },
                    email: { type: "string" },
                    error: { type: "string" }
                },
                required: ["valid"]
            },
            testCases: [
                {
                    description: "Valid email",
                    input: "test@example.com",
                    expectedOutput: { valid: true, email: "test@example.com" }
                },
                {
                    description: "Invalid email format",
                    input: "not-an-email",
                    expectedOutput: { valid: false, error: "Invalid email format" }
                },
                {
                    description: "Empty email",
                    input: "",
                    expectedOutput: { valid: false, error: "Email cannot be empty" }
                }
            ]
        }
    }
};

/**
 * Create a code config with specific input/output schemas
 */
export function createCodeConfigWithSchema(
    content: string,
    inputSchema: JsonSchema,
    outputSchema: JsonSchema | JsonSchema[],
    options: {
        shouldSpread?: boolean;
        testCases?: CodeVersionConfigObject["testCases"];
    } = {}
): CodeVersionConfigObject {
    return mergeWithBaseDefaults<CodeVersionConfigObject>({
        content,
        inputConfig: {
            inputSchema: inputSchema as any,
            shouldSpread: options.shouldSpread ?? false
        },
        outputConfig: outputSchema,
        testCases: options.testCases
    });
}

/**
 * Create a smart contract code config
 */
export function createSmartContractConfig(
    content: string,
    contractDetails: ContractDetails
): CodeVersionConfigObject {
    return {
        __version: LATEST_CONFIG_VERSION,
        content,
        contractDetails,
        inputConfig: {
            inputSchema: { type: "object" },
            shouldSpread: false
        },
        outputConfig: [
            { type: "object" },
            { type: "null" }
        ]
    };
}

/**
 * Create a code config with test cases
 */
export function createCodeConfigWithTests(
    content: string,
    testCases: CodeVersionConfigObject["testCases"]
): CodeVersionConfigObject {
    return mergeWithBaseDefaults<CodeVersionConfigObject>({
        content,
        testCases,
        inputConfig: {
            inputSchema: { type: "object" },
            shouldSpread: false
        },
        outputConfig: { type: "object" }
    });
}