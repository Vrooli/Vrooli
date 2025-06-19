import { DUMMY_ID, type TagShape } from "@vrooli/shared";

/**
 * Form data fixtures for tag-related forms
 * These represent data as it appears in form state before submission
 */

/**
 * Minimal tag creation form data - just the tag name
 */
export const minimalTagCreateFormInput = {
    tag: "javascript",
};

/**
 * Complete tag creation form data with translations
 */
export const completeTagCreateFormInput = {
    tag: "machine-learning",
    anonymous: false,
    translations: [
        {
            language: "en",
            description: "Machine learning and AI algorithms",
        },
    ],
};

/**
 * Tag update form data - minimal
 */
export const minimalTagUpdateFormInput = {
    tag: "typescript",
};

/**
 * Complete tag update form data with translations
 */
export const completeTagUpdateFormInput = {
    tag: "artificial-intelligence",
    anonymous: false,
    translations: [
        {
            language: "en", 
            description: "Artificial intelligence and neural networks",
        },
        {
            language: "es",
            description: "Inteligencia artificial y redes neuronales",
        },
    ],
};

/**
 * Form data representing TagShape objects for UI components
 */
export const minimalTagShapeFormInput: TagShape = {
    __typename: "Tag",
    id: DUMMY_ID,
    tag: "react",
};

export const completeTagShapeFormInput: TagShape = {
    __typename: "Tag", 
    id: DUMMY_ID,
    tag: "web-development",
    anonymous: false,
    translations: [
        {
            __typename: "TagTranslation",
            id: DUMMY_ID,
            language: "en",
            description: "Building websites and web applications",
        },
    ],
};

/**
 * Common tag form data variations for different scenarios
 */
export const tagFormVariants = {
    programming: {
        tag: "programming",
        anonymous: false,
        translations: [
            {
                language: "en",
                description: "Software development and programming languages",
            },
        ],
    },
    design: {
        tag: "ui-design", 
        anonymous: false,
        translations: [
            {
                language: "en",
                description: "User interface and user experience design",
            },
        ],
    },
    framework: {
        tag: "framework",
        anonymous: false,
        translations: [
            {
                language: "en",
                description: "Software frameworks and libraries",
            },
        ],
    },
    database: {
        tag: "database",
        anonymous: false,
        translations: [
            {
                language: "en",
                description: "Database management and storage systems",
            },
        ],
    },
};

/**
 * Popular tag suggestions for autocomplete testing
 */
export const popularTagSuggestions = [
    "javascript",
    "python", 
    "react",
    "node-js",
    "typescript",
    "machine-learning",
    "web-development",
    "api-development",
    "database",
    "ui-design",
];

/**
 * Form validation test cases
 */
export const invalidTagFormInputs = {
    emptyTag: {
        tag: "", // Invalid: empty tag
        anonymous: false,
    },
    whitespaceOnlyTag: {
        tag: "   ", // Invalid: whitespace only
        anonymous: false,
    },
    tooLongTag: {
        tag: "a".repeat(150), // Invalid: exceeds max length
        anonymous: false,
    },
    invalidCharacters: {
        tag: "invalid tag!", // Invalid: contains spaces and special chars
        anonymous: false,
    },
    missingTag: {
        // @ts-expect-error - Testing invalid input
        anonymous: false, // Invalid: missing required tag field
    },
};

/**
 * Tag form validation states
 */
export const tagFormValidationStates = {
    pristine: {
        values: {},
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            tag: "invalid tag!", // Contains spaces/special chars
        },
        errors: {
            tag: "Tag can only contain letters, numbers, and hyphens",
        },
        touched: {
            tag: true,
        },
        isValid: false,
        isSubmitting: false,
    },
    valid: {
        values: minimalTagCreateFormInput,
        errors: {},
        touched: {
            tag: true,
        },
        isValid: true,
        isSubmitting: false,
    },
};

/**
 * Helper function to create tag form initial values
 */
export const createTagFormInitialValues = (tagData?: Partial<any>) => ({
    tag: tagData?.tag || "",
    anonymous: tagData?.anonymous || false,
    translations: tagData?.translations || [],
    ...tagData,
});

/**
 * Helper function to validate tag name
 */
export const validateTagName = (tag: string): string | null => {
    if (!tag) return "Tag is required";
    if (tag.trim() !== tag) return "Tag cannot have leading or trailing whitespace";
    if (tag.length < 1) return "Tag must be at least 1 character";
    if (tag.length > 128) return "Tag must be less than 128 characters";
    if (!/^[a-zA-Z0-9-_+#]+$/.test(tag)) {
        return "Tag can only contain letters, numbers, hyphens, underscores, plus signs, and hash symbols";
    }
    return null;
};

/**
 * Helper function to transform form data to TagShape format
 */
export const transformTagFormToShape = (formData: any): TagShape => ({
    __typename: "Tag",
    id: formData.id || DUMMY_ID,
    tag: formData.tag,
    anonymous: formData.anonymous || false,
    translations: formData.translations?.map((t: any) => ({
        __typename: "TagTranslation",
        id: t.id || DUMMY_ID,
        language: t.language,
        description: t.description,
    })) || null,
});

/**
 * Mock tag search results for testing autocomplete
 */
export const mockTagSearchResults = [
    { id: "123456789012345678", tag: "javascript", description: "JavaScript programming language" },
    { id: "123456789012345679", tag: "python", description: "Python programming language" },
    { id: "123456789012345680", tag: "react", description: "React JavaScript library" },
    { id: "123456789012345681", tag: "nodejs", description: "Node.js runtime environment" },
    { id: "123456789012345682", tag: "typescript", description: "TypeScript programming language" },
];

/**
 * Array of tag shapes for testing lists
 */
export const tagShapeArray: TagShape[] = [
    {
        __typename: "Tag",
        id: "123456789012345678",
        tag: "frontend",
        anonymous: false,
        translations: [
            {
                __typename: "TagTranslation",
                id: DUMMY_ID,
                language: "en",
                description: "Frontend web development",
            },
        ],
    },
    {
        __typename: "Tag", 
        id: "123456789012345679",
        tag: "backend",
        anonymous: false,
        translations: [
            {
                __typename: "TagTranslation",
                id: DUMMY_ID,
                language: "en",
                description: "Backend server development",
            },
        ],
    },
    {
        __typename: "Tag",
        id: "123456789012345680", 
        tag: "fullstack",
        anonymous: false,
    },
];