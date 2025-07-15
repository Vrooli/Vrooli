import {
    CommentFor,
    commentFormConfig,
    commentFormFixtures,
    createTransformFunction,
    type CommentCreateInput,
    type CommentShape,
    type CommentUpdateInput,
    type Session,
} from "@vrooli/shared";
import { createUIFormTestFactory, type UIFormTestConfig } from "./UIFormTestFactory.js";

/**
 * Configuration for Comment form testing with data-driven test scenarios
 */
const commentFormTestConfig: UIFormTestConfig<CommentShape, CommentShape, CommentCreateInput, CommentUpdateInput, CommentShape> = {
    // Reference to the shared form configuration
    formConfig: commentFormConfig,
    
    // Test fixtures from shared package
    formFixtures: commentFormFixtures,
    
    // Form metadata
    objectType: "Comment",

    // Validation schemas from shared package
    validation: commentFormConfig.validation.schema,
    translationValidation: commentFormConfig.validation.translationSchema,

    // API endpoints from shared package
    endpoints: commentFormConfig.endpoints,

    // Transform functions - form already uses CommentShape, so no transformation needed
    formToShape: (formData: CommentShape) => formData,

    transformFunction: createTransformFunction(commentFormConfig),

    initialValuesFunction: (session?: Session, existing?: Partial<CommentShape>): CommentShape => {
        return commentFormConfig.transformations.getInitialValues(session, existing);
    },

    // DATA-DRIVEN TEST SCENARIOS - replaces all custom wrapper methods
    testScenarios: {
        targetValidation: {
            description: "Test commenting on different object types",
            testCases: [
                {
                    name: "Issue comment",
                    data: { commentedOn: { __typename: CommentFor.Issue, id: "issue_123" } },
                    shouldPass: true,
                },
                {
                    name: "PullRequest comment",
                    data: { commentedOn: { __typename: CommentFor.PullRequest, id: "pullrequest_456" } },
                    shouldPass: true,
                },
                {
                    name: "ResourceVersion comment",
                    data: { commentedOn: { __typename: CommentFor.ResourceVersion, id: "resourceversion_789" } },
                    shouldPass: true,
                },
            ],
        },

        contentValidation: {
            description: "Test content validation and length limits",
            testCases: [
                {
                    name: "Valid text",
                    field: "translations.0.text",
                    value: "Valid comment text",
                    shouldPass: true,
                },
                {
                    name: "Empty text",
                    field: "translations.0.text",
                    value: "",
                    shouldPass: false,
                },
                {
                    name: "At character limit",
                    field: "translations.0.text",
                    value: "A".repeat(32768),
                    shouldPass: true,
                },
                {
                    name: "Over character limit",
                    field: "translations.0.text",
                    value: "A".repeat(32769),
                    shouldPass: false,
                },
            ],
        },

        markdownSupport: {
            description: "Test markdown and formatting support",
            testCases: [
                {
                    name: "Heading with bold",
                    field: "translations.0.text",
                    value: "# Heading\\n\\nParagraph with **bold**",
                    shouldPass: true,
                },
                {
                    name: "Code block",
                    field: "translations.0.text",
                    value: "\`\`\`javascript\\nconst x = 1;\\n\`\`\`",
                    shouldPass: true,
                },
                {
                    name: "Links",
                    field: "translations.0.text",
                    value: "Link: [Example](https://example.com)",
                    shouldPass: true,
                },
            ],
        },
    },
};

/**
 * SIMPLIFIED: Direct factory export - no wrapper function needed!
 */
export const commentFormTestFactory = createUIFormTestFactory(commentFormTestConfig);

/**
 * Type exports for use in other test files
 */
export { commentFormTestConfig };
