import type { RunIOCreateInput, RunIOUpdateInput } from "../../../api/types.js";
import { type ModelTestFixtures, TypedTestDataFactory, createTypedFixtures } from "../../../validation/models/__test/validationTestUtils.js";
import { runIOValidation } from "../../../validation/models/runIO.js";

// Magic number constants for testing
const DATA_TOO_LONG_LENGTH = 8193;
const NAME_TOO_LONG_LENGTH = 129;
const DATA_MAX_LENGTH = 8192;
const NAME_MAX_LENGTH = 128;

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
    id4: "123456789012345681",
    id5: "123456789012345682",
    id6: "123456789012345683",
    id7: "123456789012345684",
    id8: "123456789012345685",
};

// Shared runIO test fixtures
export const runIOFixtures: ModelTestFixtures<RunIOCreateInput, RunIOUpdateInput> = {
    minimal: {
        create: {
            id: validIds.id1,
            data: "input data",
            nodeInputName: "input1",
            nodeName: "ProcessNode",
            runConnect: validIds.id2,
        },
        update: {
            id: validIds.id1,
            data: "updated data",
        },
    },
    complete: {
        create: {
            id: validIds.id1,
            data: JSON.stringify({
                type: "object",
                properties: {
                    name: { type: "string" },
                    value: { type: "number" },
                    enabled: { type: "boolean" },
                },
                values: {
                    name: "test input",
                    value: 42,
                    enabled: true,
                },
            }),
            nodeInputName: "complexInput",
            nodeName: "DataProcessorNode",
            runConnect: validIds.id2,
            // Add some extra fields that will be stripped
            // @ts-expect-error - Testing unknown fields
            unknownField1: "should be stripped",
            // @ts-expect-error - Testing unknown fields
            unknownField2: 123,
            // @ts-expect-error - Testing unknown fields
            unknownField3: true,
        },
        update: {
            id: validIds.id1,
            data: JSON.stringify({
                type: "result",
                status: "processed",
                output: "processed successfully",
                metadata: {
                    timestamp: "2023-01-01T00:00:00Z",
                    duration: 1500,
                },
            }),
            // Add some extra fields that will be stripped
            // @ts-expect-error - Testing unknown fields
            unknownField1: "should be stripped",
            // @ts-expect-error - Testing unknown fields
            unknownField2: 456,
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing required id, data, nodeInputName, nodeName, and runConnect
                unknownField: "test",
            } as unknown as RunIOCreateInput,
            update: {
                // Missing required id and data
                unknownField: "test",
            } as unknown as RunIOUpdateInput,
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                data: 456, // Should be string
                nodeInputName: 789, // Should be string
                nodeName: 101112, // Should be string
                runConnect: 131415, // Should be string
            } as unknown as RunIOCreateInput,
            update: {
                id: validIds.id1,
                data: 123, // Should be string
            } as unknown as RunIOUpdateInput,
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
                data: "test data",
                nodeInputName: "input1",
                nodeName: "TestNode",
                runConnect: validIds.id2,
            },
            update: {
                id: "invalid-id",
                data: "test data",
            },
        },
        missingData: {
            create: {
                id: validIds.id1,
                nodeInputName: "input1",
                nodeName: "TestNode",
                runConnect: validIds.id2,
                // Missing required data
            } as RunIOCreateInput,
        },
        missingNodeInputName: {
            create: {
                id: validIds.id1,
                data: "test data",
                nodeName: "TestNode",
                runConnect: validIds.id2,
                // Missing required nodeInputName
            } as RunIOCreateInput,
        },
        missingNodeName: {
            create: {
                id: validIds.id1,
                data: "test data",
                nodeInputName: "input1",
                runConnect: validIds.id2,
                // Missing required nodeName
            } as RunIOCreateInput,
        },
        missingRunConnect: {
            create: {
                id: validIds.id1,
                data: "test data",
                nodeInputName: "input1",
                nodeName: "TestNode",
                // Missing required runConnect
            } as RunIOCreateInput,
        },
        longData: {
            create: {
                id: validIds.id1,
                data: "x".repeat(DATA_TOO_LONG_LENGTH), // Too long (exceeds 8192)
                nodeInputName: "input1",
                nodeName: "TestNode",
                runConnect: validIds.id2,
            },
        },
        longNodeInputName: {
            create: {
                id: validIds.id1,
                data: "test data",
                nodeInputName: "x".repeat(NAME_TOO_LONG_LENGTH), // Too long (exceeds 128)
                nodeName: "TestNode",
                runConnect: validIds.id2,
            },
        },
        longNodeName: {
            create: {
                id: validIds.id1,
                data: "test data",
                nodeInputName: "input1",
                nodeName: "x".repeat(NAME_TOO_LONG_LENGTH), // Too long (exceeds 128)
                runConnect: validIds.id2,
            },
        },
        emptyData: {
            create: {
                id: validIds.id1,
                data: "", // Empty string becomes undefined
                nodeInputName: "input1",
                nodeName: "TestNode",
                runConnect: validIds.id2,
            },
        },
        emptyNodeInputName: {
            create: {
                id: validIds.id1,
                data: "test data",
                nodeInputName: "", // Empty string becomes undefined
                nodeName: "TestNode",
                runConnect: validIds.id2,
            },
        },
        emptyNodeName: {
            create: {
                id: validIds.id1,
                data: "test data",
                nodeInputName: "input1",
                nodeName: "", // Empty string becomes undefined
                runConnect: validIds.id2,
            },
        },
    },
    edgeCases: {
        simpleInput: {
            create: {
                id: validIds.id1,
                data: "simple string input",
                nodeInputName: "stringInput",
                nodeName: "StringProcessorNode",
                runConnect: validIds.id2,
            },
        },
        jsonInput: {
            create: {
                id: validIds.id1,
                data: JSON.stringify({ message: "hello world", count: 5 }),
                nodeInputName: "jsonInput",
                nodeName: "JsonProcessorNode",
                runConnect: validIds.id2,
            },
        },
        xmlInput: {
            create: {
                id: validIds.id1,
                data: "<root><item>value</item></root>",
                nodeInputName: "xmlInput",
                nodeName: "XmlProcessorNode",
                runConnect: validIds.id2,
            },
        },
        binaryData: {
            create: {
                id: validIds.id1,
                data: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==",
                nodeInputName: "imageInput",
                nodeName: "ImageProcessorNode",
                runConnect: validIds.id2,
            },
        },
        maxLengthData: {
            create: {
                id: validIds.id1,
                data: "a".repeat(DATA_MAX_LENGTH), // Maximum length data
                nodeInputName: "maxInput",
                nodeName: "MaxDataNode",
                runConnect: validIds.id2,
            },
        },
        maxLengthNames: {
            create: {
                id: validIds.id1,
                data: "test data",
                nodeInputName: "a".repeat(NAME_MAX_LENGTH), // Maximum length
                nodeName: "b".repeat(NAME_MAX_LENGTH), // Maximum length
                runConnect: validIds.id2,
            },
        },
        numberAsString: {
            create: {
                id: validIds.id1,
                data: "123.456",
                nodeInputName: "numberInput",
                nodeName: "NumberProcessorNode",
                runConnect: validIds.id2,
            },
        },
        booleanAsString: {
            create: {
                id: validIds.id1,
                data: "true",
                nodeInputName: "booleanInput",
                nodeName: "BooleanProcessorNode",
                runConnect: validIds.id2,
            },
        },
        arrayAsString: {
            create: {
                id: validIds.id1,
                data: JSON.stringify([1, 2, 3, "four", true]),
                nodeInputName: "arrayInput",
                nodeName: "ArrayProcessorNode",
                runConnect: validIds.id2,
            },
        },
        multilineData: {
            create: {
                id: validIds.id1,
                data: "line 1\nline 2\nline 3\n\ttabbed content\n    spaced content",
                nodeInputName: "textInput",
                nodeName: "TextProcessorNode",
                runConnect: validIds.id2,
            },
        },
        specialCharacters: {
            create: {
                id: validIds.id1,
                data: "Special chars: !@#$%^&*()[]{}|;':\",./<>?`~",
                nodeInputName: "specialInput",
                nodeName: "SpecialProcessorNode",
                runConnect: validIds.id2,
            },
        },
        unicodeData: {
            create: {
                id: validIds.id1,
                data: "Unicode: 🚀 🌟 ⭐ 💫 ✨ 中文 العربية हिन्दी",
                nodeInputName: "unicodeInput",
                nodeName: "UnicodeProcessorNode",
                runConnect: validIds.id2,
            },
        },
        updateDataOnly: {
            update: {
                id: validIds.id1,
                data: "updated input data",
            },
        },
        updateComplexData: {
            update: {
                id: validIds.id1,
                data: JSON.stringify({
                    status: "updated",
                    result: {
                        processed: true,
                        output: "new result",
                        metadata: {
                            updatedAt: "2023-12-01T10:30:00Z",
                        },
                    },
                }),
            },
        },
        updateMaxLengthData: {
            update: {
                id: validIds.id1,
                data: "z".repeat(DATA_MAX_LENGTH), // Maximum length in update
            },
        },
        updateEmptyData: {
            update: {
                id: validIds.id1,
                data: "", // This should become undefined and fail validation since data is required
            } as RunIOUpdateInput,
        },
    },
};

// Custom factory that always generates valid IDs and required fields
const customizers = {
    create: (base: Partial<RunIOCreateInput>): RunIOCreateInput => ({
        id: validIds.id1,
        data: "default data",
        nodeInputName: "defaultInput",
        nodeName: "DefaultNode",
        runConnect: validIds.id2,
        ...base,
    }),
    update: (base: Partial<RunIOUpdateInput>): RunIOUpdateInput => ({
        id: validIds.id1,
        data: "default updated data",
        ...base,
    }),
};

// Export a factory for creating test data programmatically
export const runIOTestDataFactory = new TypedTestDataFactory(runIOFixtures, runIOValidation, customizers);
export const typedRunIOFixtures = createTypedFixtures(runIOFixtures, runIOValidation);
