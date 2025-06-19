import { type PassableLogger } from "../../../consts/commonTypes.js";
import { LATEST_CONFIG_VERSION, type StringifyMode } from "../../../shape/configs/utils.js";
import { type ConfigTestFixtures } from "./baseConfigFixtures.js";

/**
 * Mock logger for testing
 */
export const mockLogger: PassableLogger = {
    error: jest.fn(),
    warn: jest.fn(),
    info: jest.fn(),
    debug: jest.fn(),
    trace: jest.fn(),
};

/**
 * Create a fresh mock logger with jest spies
 */
export function createMockLogger(): PassableLogger {
    return {
        error: jest.fn(),
        warn: jest.fn(),
        info: jest.fn(),
        debug: jest.fn(),
        trace: jest.fn(),
    };
}

/**
 * Test data for stringifyObject and parseObject functions
 */
export const utilsTestData = {
    simple: {
        name: "Test",
        value: 123,
        active: true,
    },
    nested: {
        user: {
            id: "user_123",
            name: "Test User",
            settings: {
                theme: "dark",
                notifications: true,
            },
        },
        metadata: {
            created: "2024-01-01T00:00:00Z",
            version: LATEST_CONFIG_VERSION,
        },
    },
    withArrays: {
        items: ["item1", "item2", "item3"],
        numbers: [1, 2, 3, 4, 5],
        mixed: [1, "two", true, null],
    },
    withNull: {
        value: null,
        optional: undefined,
    },
    empty: {},
    complex: {
        __version: LATEST_CONFIG_VERSION,
        data: {
            users: [
                { id: 1, name: "Alice", roles: ["admin", "user"] },
                { id: 2, name: "Bob", roles: ["user"] },
            ],
            settings: {
                global: {
                    theme: "light",
                    language: "en",
                },
                features: {
                    experimental: false,
                    beta: true,
                },
            },
        },
        metadata: {
            timestamp: Date.now(),
            source: "test",
        },
    },
};

/**
 * Invalid JSON strings for parsing tests
 */
export const invalidJsonStrings = {
    malformed: "{invalid json}",
    truncated: '{"name": "test"',
    extraComma: '{"name": "test",}',
    singleQuotes: "{'name': 'test'}",
    unquotedKeys: "{name: 'test'}",
    undefined: "undefined",
    null: "null",
    empty: "",
};

/**
 * Valid JSON strings for parsing tests
 */
export const validJsonStrings = {
    simple: JSON.stringify(utilsTestData.simple),
    nested: JSON.stringify(utilsTestData.nested),
    withArrays: JSON.stringify(utilsTestData.withArrays),
    empty: JSON.stringify(utilsTestData.empty),
    complex: JSON.stringify(utilsTestData.complex),
};

/**
 * StringifyMode test fixtures
 */
export const stringifyModeFixtures: Record<StringifyMode, {
    supported: boolean;
    testData?: Record<string, string>;
    expectedError?: string;
}> = {
    json: {
        supported: true,
        testData: validJsonStrings,
    },
    // Future modes can be added here as they're implemented
    // yaml: {
    //     supported: false,
    //     expectedError: "Unsupported stringify mode: yaml"
    // },
    // xml: {
    //     supported: false,
    //     expectedError: "Unsupported stringify mode: xml"
    // }
};

/**
 * Edge case test data
 */
export const edgeCases = {
    circular: (() => {
        const obj: any = { name: "circular" };
        obj.self = obj;
        return obj;
    })(),
    largeData: {
        array: Array.from({ length: 1000 }, (_, i) => ({
            id: i,
            value: `item_${i}`,
            nested: {
                data: Array.from({ length: 10 }, (_, j) => j),
            },
        })),
    },
    specialCharacters: {
        unicode: "Hello 👋 World 🌍",
        escaped: "Line 1\nLine 2\tTabbed",
        quotes: 'He said "Hello"',
        backslash: "C:\\Users\\test",
    },
    dateTypes: {
        date: new Date("2024-01-01T00:00:00Z"),
        isoString: "2024-01-01T00:00:00Z",
        timestamp: 1704067200000,
    },
};

/**
 * Performance test fixtures
 */
export const performanceFixtures = {
    small: utilsTestData.simple,
    medium: utilsTestData.complex,
    large: edgeCases.largeData,
};

/**
 * Scenario-based fixtures for common use cases
 */
export const scenarioFixtures = {
    configSerialization: {
        input: {
            __version: LATEST_CONFIG_VERSION,
            settings: {
                theme: "dark",
                language: "en",
                features: ["feature1", "feature2"],
            },
            user: {
                id: "user_123",
                preferences: {
                    notifications: true,
                    emailDigest: "weekly",
                },
            },
        },
        expectedJson: '{"__version":"1.0","settings":{"theme":"dark","language":"en","features":["feature1","feature2"]},"user":{"id":"user_123","preferences":{"notifications":true,"emailDigest":"weekly"}}}',
    },
    apiResponse: {
        input: {
            success: true,
            data: {
                items: [
                    { id: 1, name: "Item 1" },
                    { id: 2, name: "Item 2" },
                ],
                total: 2,
            },
            metadata: {
                requestId: "req_123",
                timestamp: "2024-01-01T00:00:00Z",
            },
        },
    },
    errorResponse: {
        input: {
            success: false,
            error: {
                code: "VALIDATION_ERROR",
                message: "Invalid input",
                details: {
                    field: "email",
                    reason: "Invalid format",
                },
            },
        },
    },
};

/**
 * Utils config fixtures following the standard pattern
 * Note: utils.ts doesn't define a config object type, so we create fixtures
 * for testing the utility functions themselves
 */
export const utilsConfigFixtures: ConfigTestFixtures<{ mode: StringifyMode; data: any }> = {
    minimal: {
        mode: "json" as StringifyMode,
        data: utilsTestData.simple,
    },
    complete: {
        mode: "json" as StringifyMode,
        data: utilsTestData.complex,
    },
    withDefaults: {
        mode: "json" as StringifyMode,
        data: {},
    },
    invalid: {
        missingMode: {
            // Missing mode
            data: utilsTestData.simple,
        } as any,
        unsupportedMode: {
            mode: "yaml" as any, // Unsupported mode
            data: utilsTestData.simple,
        },
        invalidData: {
            mode: "json" as StringifyMode,
            data: edgeCases.circular, // Circular reference
        },
    },
    variants: {
        jsonSimple: {
            mode: "json" as StringifyMode,
            data: utilsTestData.simple,
        },
        jsonNested: {
            mode: "json" as StringifyMode,
            data: utilsTestData.nested,
        },
        jsonArrays: {
            mode: "json" as StringifyMode,
            data: utilsTestData.withArrays,
        },
        jsonEmpty: {
            mode: "json" as StringifyMode,
            data: utilsTestData.empty,
        },
        jsonComplex: {
            mode: "json" as StringifyMode,
            data: utilsTestData.complex,
        },
        jsonSpecialChars: {
            mode: "json" as StringifyMode,
            data: edgeCases.specialCharacters,
        },
    },
};

/**
 * Helper to create test cases for stringify/parse round trips
 */
export function createRoundTripTest(data: any, mode: StringifyMode = "json") {
    return {
        original: data,
        mode,
        stringify: () => JSON.stringify(data),
        parse: (str: string) => JSON.parse(str),
    };
}

/**
 * Helper to test error scenarios
 */
export function createErrorScenario(
    description: string,
    input: string,
    mode: StringifyMode = "json",
    expectedError?: string,
) {
    return {
        description,
        input,
        mode,
        expectedError: expectedError || "Error parsing data",
    };
}