import {
    endpointsResource,
    resourceVersionTranslationValidation,
    resourceVersionValidation,
    type ResourceVersionCreateInput,
    type ResourceVersionShape,
    type ResourceVersionUpdateInput,
    type Session,
} from "@vrooli/shared";
import { promptInitialValues, transformPromptVersionValues } from "../../../views/objects/prompt/PromptUpsert.js";
import { createUIFormTestFactory, type UIFormTestConfig } from "./UIFormTestFactory.js";

/**
 * Configuration for Prompt form testing with data-driven test scenarios
 */
const promptFormTestConfig: UIFormTestConfig<ResourceVersionShape, ResourceVersionShape, ResourceVersionCreateInput, ResourceVersionUpdateInput, ResourceVersionShape> = {
    // Form metadata
    objectType: "ResourceVersion",
    formFixtures: {
        minimal: {
            __typename: "ResourceVersion" as const,
            id: "prompt_minimal",
            codeLanguage: "Javascript",
            config: {
                schema: JSON.stringify({
                    default: {},
                    yup: {},
                }),
            },
            isAutomatable: false,
            isComplete: false,
            isPrivate: false,
            resourceSubType: "StandardPrompt",
            versionLabel: "1.0.0",
            versionNotes: null,
            translations: [{
                __typename: "ResourceVersionTranslation" as const,
                id: "trans_minimal",
                language: "en",
                name: "Test Prompt",
                description: "A minimal test prompt",
                instructions: "You are a helpful assistant. Answer the following question: {{question}}",
                details: null,
            }],
        },
        complete: {
            __typename: "ResourceVersion" as const,
            id: "prompt_complete",
            codeLanguage: "Typescript",
            config: {
                schema: JSON.stringify({
                    default: {
                        domain: "general",
                        input_type: "question",
                        user_input: "",
                        context: "",
                    },
                    yup: {
                        domain: "string().required()",
                        input_type: "string().oneOf(['question', 'problem', 'request'])",
                        user_input: "string().required().min(10)",
                        context: "string()",
                    },
                }),
            },
            isAutomatable: true,
            isComplete: true,
            isPrivate: false,
            resourceSubType: "StandardPrompt",
            versionLabel: "2.1.0",
            versionNotes: "Enhanced with schema validation",
            translations: [{
                __typename: "ResourceVersionTranslation" as const,
                id: "trans_complete",
                language: "en",
                name: "Complete AI Prompt",
                description: "A comprehensive AI prompt with detailed instructions and schema. This prompt handles multiple input types and provides structured responses.",
                instructions: `You are an expert assistant specialized in {{domain}}. 

Instructions:
1. Analyze the user's {{input_type}} carefully
2. Provide a structured response following the format below
3. Include relevant examples when appropriate

User Input: {{user_input}}
Context: {{context}}

Please provide your response in the following format:
- Analysis: [Your analysis here]
- Recommendation: [Your recommendation here]
- Examples: [Relevant examples here]`,
                details: "Advanced prompt with variable substitution and structured output",
            }],
        },
        invalid: {
            __typename: "ResourceVersion" as const,
            id: "prompt_invalid",
            codeLanguage: "Javascript",
            config: {
                schema: "invalid-json", // Invalid JSON
            },
            isAutomatable: false,
            isComplete: false,
            isPrivate: false,
            resourceSubType: "StandardPrompt",
            versionLabel: "not-a-valid-version", // Invalid version format
            versionNotes: null,
            translations: [{
                __typename: "ResourceVersionTranslation" as const,
                id: "trans_invalid",
                language: "en",
                name: "", // Invalid: required field is empty
                description: "",
                instructions: "", // Invalid: empty prompt
                details: null,
            }],
        },
        edgeCase: {
            __typename: "ResourceVersion" as const,
            id: "prompt_edge",
            codeLanguage: "Typescript",
            config: {
                schema: JSON.stringify({
                    default: {
                        system_message: "You are a helpful assistant.",
                        user_input: "",
                        name: "",
                        age: 0,
                        preferences: [],
                        metadata: {},
                        status: "success",
                        result: "",
                        confidence: 0.9,
                    },
                    yup: {
                        system_message: "string().required()",
                        user_input: "string().required().min(1).max(1000)",
                        name: "string().required().min(2).max(100)",
                        age: "number().integer().min(0).max(150)",
                        preferences: "array().of(string())",
                        metadata: "object()",
                        status: "string().oneOf(['success', 'error', 'pending'])",
                        result: "string().required()",
                        confidence: "number().min(0).max(1)",
                    },
                }),
            },
            isAutomatable: false,
            isComplete: false,
            isPrivate: false,
            resourceSubType: "StandardPrompt",
            versionLabel: "999.999.999",
            versionNotes: "Complex edge case with advanced variables",
            translations: [{
                __typename: "ResourceVersionTranslation" as const,
                id: "trans_edge",
                language: "en",
                name: "A".repeat(200), // Edge case: very long name
                description: "Description with special characters: @#$%^&*()[]{}|\\:;\"'<>,.?/~`\n\nMultiple\nline\nbreaks\n\nAnd emoji: 🤖 💬 ✨",
                instructions: `Complex prompt with special characters and variables:

{{system_message}}

User: {{user_input}}

Instructions with special chars: @#$%^&*()[]{}|\\:;\"'<>,.?/~\`

Variables to replace:
- {{name}} (string)
- {{age}} (number)
- {{preferences}} (array)
- {{metadata}} (object)

Response format:
\`\`\`json
{
  "status": "{{status}}",
  "result": "{{result}}",
  "confidence": {{confidence}}
}
\`\`\``,
                details: "Complex prompt with variable substitution and JSON output formatting",
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
        const result = transformPromptVersionValues(shape, existing, isCreate);
        if (!result) {
            throw new Error("Transform function returned undefined");
        }
        return result;
    },

    initialValuesFunction: (session?: Session, existing?: Partial<ResourceVersionShape>): ResourceVersionShape => {
        return promptInitialValues(session, existing || {});
    },

    // DATA-DRIVEN TEST SCENARIOS - replaces all custom wrapper methods
    testScenarios: {
        languageValidation: {
            description: "Test prompt code language support",
            testCases: [
                {
                    name: "Javascript language",
                    data: { codeLanguage: "Javascript" },
                    shouldPass: true,
                },
                {
                    name: "Python language",
                    data: { codeLanguage: "Python" },
                    shouldPass: true,
                },
                {
                    name: "Typescript language",
                    data: { codeLanguage: "Typescript" },
                    shouldPass: true,
                },
            ],
        },

        promptValidation: {
            description: "Test prompt content validation",
            testCases: [
                {
                    name: "Valid prompt with variables",
                    field: "translations.0.instructions",
                    value: "You are a helpful assistant. Answer: {{question}}",
                    shouldPass: true,
                },
                {
                    name: "Empty prompt",
                    field: "translations.0.instructions",
                    value: "",
                    shouldPass: false,
                },
                {
                    name: "Complex prompt with multiple variables",
                    field: "translations.0.instructions",
                    value: "Context: {{context}}\nUser: {{user_input}}\nResponse in {{format}} format.",
                    shouldPass: true,
                },
            ],
        },

        schemaValidation: {
            description: "Test prompt schema configuration",
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
                    name: "Complex schema with validation",
                    data: {
                        config: {
                            schema: JSON.stringify({
                                default: { question: "", context: "" },
                                yup: { question: "string().required()", context: "string()" },
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
                    name: "Invalid version format",
                    field: "versionLabel",
                    value: "not-a-version",
                    shouldPass: false,
                },
                {
                    name: "Pre-release version",
                    field: "versionLabel",
                    value: "1.0.0-alpha.1",
                    shouldPass: true,
                },
            ],
        },

        promptFeatures: {
            description: "Test different prompt features",
            testCases: [
                {
                    name: "Simple prompt",
                    data: {
                        isComplete: false,
                        isAutomatable: false,
                        resourceSubType: "StandardPrompt",
                    },
                    shouldPass: true,
                },
                {
                    name: "Automatable prompt",
                    data: {
                        isComplete: true,
                        isAutomatable: true,
                        resourceSubType: "StandardPrompt",
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
export const promptFormTestFactory = createUIFormTestFactory(promptFormTestConfig);

/**
 * Type exports for use in other test files
 */
export { promptFormTestConfig };
