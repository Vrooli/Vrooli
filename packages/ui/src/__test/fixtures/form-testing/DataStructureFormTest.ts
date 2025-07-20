import {
    endpointsResource,
    resourceVersionTranslationValidation,
    resourceVersionValidation,
    type ResourceVersionCreateInput,
    type ResourceVersionShape,
    type ResourceVersionUpdateInput,
    type Session,
} from "@vrooli/shared";
import { dataStructureInitialValues, transformDataStructureVersionValues } from "../../../views/objects/dataStructure/DataStructureUpsert.js";
import { createUIFormTestFactory, type UIFormTestConfig } from "./UIFormTestFactory.js";

/**
 * Configuration for DataStructure form testing with data-driven test scenarios
 */
const dataStructureFormTestConfig: UIFormTestConfig<ResourceVersionShape, ResourceVersionShape, ResourceVersionCreateInput, ResourceVersionUpdateInput, ResourceVersionShape> = {
    // Form metadata
    objectType: "ResourceVersion",
    formFixtures: {
        minimal: {
            __typename: "ResourceVersion" as const,
            id: "datastructure_minimal",
            codeLanguage: "Json",
            config: null,
            isAutomatable: false,
            isComplete: false,
            isPrivate: false,
            resourceSubType: "DataStructure",
            versionLabel: "1.0.0",
            versionNotes: null,
            translations: [{
                __typename: "ResourceVersionTranslation" as const,
                id: "trans_minimal",
                language: "en",
                name: "Test Data Structure",
                description: "A minimal test data structure",
                instructions: null,
                details: null,
            }],
        },
        complete: {
            __typename: "ResourceVersion" as const,
            id: "datastructure_complete",
            codeLanguage: "Json",
            config: {
                schema: JSON.stringify({
                    default: {
                        name: "string",
                        email: "string",
                        age: "number",
                    },
                    yup: {
                        name: "string().required()",
                        email: "string().email().required()",
                        age: "number().positive().integer()",
                    },
                }),
                props: {
                    validation: true,
                    strictMode: true,
                    allowNulls: false,
                },
                isFile: false,
            },
            isAutomatable: true,
            isComplete: true,
            isPrivate: false,
            resourceSubType: "DataStructure",
            versionLabel: "1.2.0",
            versionNotes: "Added comprehensive validation",
            translations: [{
                __typename: "ResourceVersionTranslation" as const,
                id: "trans_complete",
                language: "en",
                name: "Complete Test Data Structure",
                description: "A comprehensive test data structure with all fields filled",
                instructions: "Validate user input according to schema",
                details: "Includes email validation and age constraints",
            }],
        },
        invalid: {
            __typename: "ResourceVersion" as const,
            id: "datastructure_invalid",
            codeLanguage: "Json",
            config: null,
            isAutomatable: false,
            isComplete: false,
            isPrivate: false,
            resourceSubType: "DataStructure",
            versionLabel: "", // Invalid: required field is empty
            versionNotes: null,
            translations: [{
                __typename: "ResourceVersionTranslation" as const,
                id: "trans_invalid",
                language: "en",
                name: "", // Invalid: required field is empty
                description: "",
                instructions: null,
                details: null,
            }],
        },
        edgeCase: {
            __typename: "ResourceVersion" as const,
            id: "datastructure_edge",
            codeLanguage: "Json",
            config: {
                schema: JSON.stringify({
                    default: {},
                    yup: {},
                    nested: {
                        deep: {
                            very: {
                                deeply: {
                                    nested: "value",
                                },
                            },
                        },
                    },
                }),
                props: {
                    complexProp: {
                        array: [1, 2, 3, { nested: true }],
                        boolean: true,
                        null: null,
                    },
                },
                isFile: true,
            },
            isAutomatable: false,
            isComplete: true,
            isPrivate: true,
            resourceSubType: "DataStructure",
            versionLabel: "999.999.999", // Edge case: high version numbers
            versionNotes: "Edge case with complex nested structures",
            translations: [{
                __typename: "ResourceVersionTranslation" as const,
                id: "trans_edge",
                language: "en",
                name: "A".repeat(200), // Edge case: very long name
                description: "Description with special characters: @#$%^&*()[]{}|\\:;\"'<>,.?/~`",
                instructions: "Handle complex nested data structures",
                details: "Support for deeply nested objects and arrays",
            }],
        },
    },

    // Validation schemas from shared package
    validation: resourceVersionValidation,
    translationValidation: resourceVersionTranslationValidation,

    // API endpoints from shared package
    endpoints: {
        create: endpointsResource.createOne,
        update: endpointsResource.updateOne,
    },

    // Transform functions - form already uses ResourceVersionShape, so no transformation needed
    formToShape: (formData: ResourceVersionShape) => formData,

    transformFunction: (shape: ResourceVersionShape, existing: ResourceVersionShape, isCreate: boolean) => {
        const result = transformDataStructureVersionValues(shape, existing, isCreate);
        if (!result) {
            throw new Error("Transform function returned undefined");
        }
        return result;
    },

    initialValuesFunction: (session?: Session, existing?: Partial<ResourceVersionShape>): ResourceVersionShape => {
        return dataStructureInitialValues(session, existing || {});
    },

    // DATA-DRIVEN TEST SCENARIOS - replaces all custom wrapper methods
    testScenarios: {
        schemaValidation: {
            description: "Test data structure schema validation",
            testCases: [
                {
                    name: "Valid JSON schema",
                    data: {
                        config: {
                            schema: JSON.stringify({ default: {}, yup: {} }),
                        },
                    },
                    shouldPass: true,
                },
                {
                    name: "Invalid JSON schema",
                    data: {
                        config: {
                            schema: "invalid json {",
                        },
                    },
                    shouldPass: false,
                },
                {
                    name: "Complex nested schema",
                    data: {
                        config: {
                            schema: JSON.stringify({
                                default: { user: { name: "", email: "" } },
                                yup: { user: "object().shape({ name: string().required(), email: string().email() })" },
                            }),
                        },
                    },
                    shouldPass: true,
                },
            ],
        },

        versionValidation: {
            description: "Test version label validation",
            testCases: [
                {
                    name: "Valid semantic version",
                    field: "versionLabel",
                    value: "1.0.0",
                    shouldPass: true,
                },
                {
                    name: "Empty version",
                    field: "versionLabel",
                    value: "",
                    shouldPass: false,
                },
                {
                    name: "Development version",
                    field: "versionLabel",
                    value: "0.1.0-dev",
                    shouldPass: true,
                },
            ],
        },

        fileHandling: {
            description: "Test file-based vs in-memory data structures",
            testCases: [
                {
                    name: "In-memory data structure",
                    data: {
                        config: { isFile: false },
                    },
                    shouldPass: true,
                },
                {
                    name: "File-based data structure",
                    data: {
                        config: { isFile: true },
                    },
                    shouldPass: true,
                },
            ],
        },

        completenessStates: {
            description: "Test different completion states",
            testCases: [
                {
                    name: "Draft structure",
                    data: {
                        isComplete: false,
                        isAutomatable: false,
                    },
                    shouldPass: true,
                },
                {
                    name: "Complete structure",
                    data: {
                        isComplete: true,
                        isAutomatable: true,
                    },
                    shouldPass: true,
                },
            ],
        },
    },
};

/**
 * SIMPLIFIED: Direct factory export - no wrapper function needed!
 */
export const dataStructureFormTestFactory = createUIFormTestFactory(dataStructureFormTestConfig);

/**
 * Generated test cases for data-driven testing
 */
export const dataStructureTestCases = dataStructureFormTestFactory.generateUITestCases();

/**
 * Type exports for use in other test files
 */
export { dataStructureFormTestConfig };
