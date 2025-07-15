import {
    endpointsResource,
    resourceVersionTranslationValidation,
    resourceVersionValidation,
    type ResourceVersionCreateInput,
    type ResourceVersionShape,
    type ResourceVersionUpdateInput,
    type Session,
} from "@vrooli/shared";
import { dataConverterInitialValues, transformDataConverterVersionValues } from "../../../views/objects/dataConverter/DataConverterUpsert.js";
import { createUIFormTestFactory, type UIFormTestConfig } from "./UIFormTestFactory.js";

/**
 * Configuration for DataConverter form testing with data-driven test scenarios
 */
const dataConverterFormTestConfig: UIFormTestConfig<ResourceVersionShape, ResourceVersionShape, ResourceVersionCreateInput, ResourceVersionUpdateInput, ResourceVersionShape> = {
    // Form metadata
    objectType: "ResourceVersion",
    formFixtures: {
        minimal: {
            __typename: "ResourceVersion" as const,
            id: "dataconverter_minimal",
            codeLanguage: "Haskell",
            config: null,
            isAutomatable: false,
            isComplete: false,
            isPrivate: false,
            resourceSubType: "DataConverter",
            versionLabel: "1.0.0",
            versionNotes: null,
            translations: [{
                __typename: "ResourceVersionTranslation" as const,
                id: "trans_minimal",
                language: "en",
                name: "Test Data Converter",
                description: "A minimal test data converter",
                instructions: "convert :: String -> Int\nconvert s = read s",
                details: null,
            }],
        },
        complete: {
            __typename: "ResourceVersion" as const,
            id: "dataconverter_complete",
            codeLanguage: "Python",
            config: null,
            isAutomatable: true,
            isComplete: true,
            isPrivate: false,
            resourceSubType: "DataConverter",
            versionLabel: "2.1.0",
            versionNotes: "Enhanced with error handling",
            translations: [{
                __typename: "ResourceVersionTranslation" as const,
                id: "trans_complete",
                language: "en",
                name: "Complete Data Converter",
                description: "A comprehensive data converter with detailed logic. This converter handles multiple data types and provides robust error handling.",
                instructions: "def convert_data(input_data):\n    \"\"\"Convert input data to the target format.\"\"\"\n    if isinstance(input_data, str):\n        try:\n            return int(input_data)\n        except ValueError:\n            return 0\n    elif isinstance(input_data, list):\n        return [convert_data(item) for item in input_data]\n    else:\n        return input_data",
                details: "Handles multiple data types with error recovery",
            }],
        },
        invalid: {
            __typename: "ResourceVersion" as const,
            id: "dataconverter_invalid",
            codeLanguage: "Haskell",
            config: null,
            isAutomatable: false,
            isComplete: false,
            isPrivate: false,
            resourceSubType: "DataConverter",
            versionLabel: "not-a-valid-version", // Invalid version format
            versionNotes: null,
            translations: [{
                __typename: "ResourceVersionTranslation" as const,
                id: "trans_invalid",
                language: "en",
                name: "", // Invalid: required field is empty
                description: "",
                instructions: "", // Invalid: empty code
                details: null,
            }],
        },
        edgeCase: {
            __typename: "ResourceVersion" as const,
            id: "dataconverter_edge",
            codeLanguage: "Typescript",
            config: null,
            isAutomatable: false,
            isComplete: false,
            isPrivate: false,
            resourceSubType: "DataConverter",
            versionLabel: "999.999.999",
            versionNotes: "Edge case with complex types",
            translations: [{
                __typename: "ResourceVersionTranslation" as const,
                id: "trans_edge",
                language: "en",
                name: "A".repeat(200), // Edge case: very long name
                description: "Description with special characters: @#$%^&*()[]{}|\\:;\"'<>,.?/~`\n\nMultiple\nline\nbreaks\n\nAnd emoji: ⚡ 🔄 📊",
                instructions: "// Complex converter with special characters\ninterface DataInput {\n  field: string;\n  value: number | string | boolean;\n}\n\nconst convert = (data: DataInput[]): Record<string, any> => {\n  return data.reduce((acc, item) => {\n    acc[item.field] = item.value;\n    return acc;\n  }, {} as Record<string, any>);\n};",
                details: "Complex TypeScript converter with type safety",
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
        const result = transformDataConverterVersionValues(shape, existing, isCreate);
        if (!result) {
            throw new Error("Transform function returned undefined");
        }
        return result;
    },

    initialValuesFunction: (session?: Session, existing?: Partial<ResourceVersionShape>): ResourceVersionShape => {
        return dataConverterInitialValues(session, existing || {});
    },

    // DATA-DRIVEN TEST SCENARIOS - replaces all custom wrapper methods
    testScenarios: {
        languageValidation: {
            description: "Test data converter language support",
            testCases: [
                {
                    name: "Haskell language",
                    data: { codeLanguage: "Haskell" },
                    shouldPass: true,
                },
                {
                    name: "Python language",
                    data: { codeLanguage: "Python" },
                    shouldPass: true,
                },
                {
                    name: "Javascript language",
                    data: { codeLanguage: "Javascript" },
                    shouldPass: true,
                },
                {
                    name: "Typescript language",
                    data: { codeLanguage: "Typescript" },
                    shouldPass: true,
                },
            ],
        },

        codeValidation: {
            description: "Test data converter code validation",
            testCases: [
                {
                    name: "Valid Haskell code",
                    field: "translations.0.instructions",
                    value: "convert :: String -> Int\nconvert s = read s",
                    shouldPass: true,
                },
                {
                    name: "Empty code",
                    field: "translations.0.instructions",
                    value: "",
                    shouldPass: false,
                },
                {
                    name: "Complex Python converter",
                    field: "translations.0.instructions",
                    value: "def convert(data):\n    return [int(x) for x in data if x.isdigit()]",
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
                    name: "Invalid version format",
                    field: "versionLabel",
                    value: "not-a-version",
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

        converterFeatures: {
            description: "Test different data converter features",
            testCases: [
                {
                    name: "Simple converter",
                    data: {
                        isComplete: false,
                        isAutomatable: false,
                        resourceSubType: "DataConverter",
                    },
                    shouldPass: true,
                },
                {
                    name: "Automatable converter",
                    data: {
                        isComplete: true,
                        isAutomatable: true,
                        resourceSubType: "DataConverter",
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
export const dataConverterFormTestFactory = createUIFormTestFactory(dataConverterFormTestConfig);

/**
 * Type exports for use in other test files
 */
export { dataConverterFormTestConfig };
