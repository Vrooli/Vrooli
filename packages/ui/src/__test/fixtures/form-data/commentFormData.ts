import { CommentFor, DUMMY_ID, type CommentShape, type CommentTranslationShape } from "@vrooli/shared";

/**
 * Form data fixtures for comment creation and editing
 * These represent data as it appears in form state before submission
 */

/**
 * Minimal comment form input - just text in one language
 */
export const minimalCommentFormInput: Partial<CommentShape> = {
    translations: [{
        __typename: "CommentTranslation",
        id: DUMMY_ID,
        language: "en",
        text: "This is a simple comment",
    }],
};

/**
 * Complete comment form input with all fields
 */
export const completeCommentFormInput: Partial<CommentShape> = {
    threadId: null, // Root comment (not a reply)
    translations: [{
        __typename: "CommentTranslation",
        id: DUMMY_ID,
        language: "en",
        text: "This is a comprehensive comment with detailed thoughts and analysis about the subject matter.",
    }],
};

/**
 * Reply comment form input (part of a thread)
 */
export const replyCommentFormInput: Partial<CommentShape> = {
    threadId: "123456789012345678", // Parent thread ID (18 digits)
    translations: [{
        __typename: "CommentTranslation",
        id: DUMMY_ID,
        language: "en",
        text: "I agree with your point. Here's my additional perspective...",
    }],
};

/**
 * Multi-language comment form input
 */
export const multiLanguageCommentFormInput: Partial<CommentShape> = {
    translations: [
        {
            __typename: "CommentTranslation",
            id: DUMMY_ID,
            language: "en",
            text: "This is a comment in English",
        },
        {
            __typename: "CommentTranslation",
            id: DUMMY_ID,
            language: "es",
            text: "Este es un comentario en español",
        },
        {
            __typename: "CommentTranslation",
            id: DUMMY_ID,
            language: "fr",
            text: "Ceci est un commentaire en français",
        },
    ],
};

/**
 * Comment with markdown formatting
 */
export const markdownCommentFormInput: Partial<CommentShape> = {
    translations: [{
        __typename: "CommentTranslation",
        id: DUMMY_ID,
        language: "en",
        text: `# My Comment

This comment has **bold** and *italic* text.

## Key Points:
- First point
- Second point
- Third point

\`\`\`javascript
const example = "code block";
\`\`\`

> This is a quote

[Link to resource](https://example.com)`,
    }],
};

/**
 * Long comment form input (testing text limits)
 */
export const longCommentFormInput: Partial<CommentShape> = {
    translations: [{
        __typename: "CommentTranslation",
        id: DUMMY_ID,
        language: "en",
        text: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. ".repeat(100).trim(), // ~5600 characters
    }],
};

/**
 * Comment form input for different target types
 */
export const commentForIssueFormInput: Partial<CommentShape> = {
    commentedOn: {
        __typename: CommentFor.Issue,
        id: "123456789012345678",
    },
    translations: [{
        __typename: "CommentTranslation",
        id: DUMMY_ID,
        language: "en",
        text: "This issue needs more clarification on the requirements.",
    }],
};

export const commentForPullRequestFormInput: Partial<CommentShape> = {
    commentedOn: {
        __typename: CommentFor.PullRequest,
        id: "234567890123456789",
    },
    translations: [{
        __typename: "CommentTranslation",
        id: DUMMY_ID,
        language: "en",
        text: "Great implementation! Just a few minor suggestions...",
    }],
};

export const commentForResourceVersionFormInput: Partial<CommentShape> = {
    commentedOn: {
        __typename: CommentFor.ResourceVersion,
        id: "345678901234567890",
    },
    translations: [{
        __typename: "CommentTranslation",
        id: DUMMY_ID,
        language: "en",
        text: "This resource was very helpful. Thanks for sharing!",
    }],
};

/**
 * Edited comment form input
 */
export const editedCommentFormInput: Partial<CommentShape> = {
    id: "456789012345678901",
    translations: [{
        __typename: "CommentTranslation",
        id: "567890123456789012",
        language: "en",
        text: "EDIT: I've updated my comment to clarify my point.",
    }],
};

/**
 * Invalid form inputs for testing validation
 */
export const invalidCommentFormInputs = {
    emptyText: {
        translations: [{
            __typename: "CommentTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            text: "",
        }],
    },
    whitespaceOnly: {
        translations: [{
            __typename: "CommentTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            text: "   \n\t   ",
        }],
    },
    missingTranslations: {
        // @ts-expect-error - Testing missing required translations
        translations: [],
    },
    noTranslations: {
        // @ts-expect-error - Testing undefined translations
        translations: undefined,
    },
    missingText: {
        translations: [{
            __typename: "CommentTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            // @ts-expect-error - Testing missing required text
            text: undefined,
        }],
    },
    missingLanguage: {
        translations: [{
            __typename: "CommentTranslation" as const,
            id: DUMMY_ID,
            // @ts-expect-error - Testing missing required language
            language: undefined,
            text: "Valid text",
        }],
    },
    textTooLong: {
        translations: [{
            __typename: "CommentTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            text: "x".repeat(32769), // Over the 32768 character limit
        }],
    },
    invalidThreadId: {
        threadId: "invalid-id", // Not a valid snowflake ID
        translations: [{
            __typename: "CommentTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            text: "Valid text",
        }],
    },
    missingCommentedOn: {
        // @ts-expect-error - Testing missing commentedOn when creating
        commentedOn: undefined,
        translations: [{
            __typename: "CommentTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            text: "Valid text",
        }],
    },
    invalidCommentedOnType: {
        commentedOn: {
            // @ts-expect-error - Testing invalid type
            __typename: "InvalidType",
            id: "123456789012345678",
        },
        translations: [{
            __typename: "CommentTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            text: "Valid text",
        }],
    },
    invalidCommentedOnId: {
        commentedOn: {
            __typename: CommentFor.Issue,
            id: "invalid-id", // Not a valid snowflake ID
        },
        translations: [{
            __typename: "CommentTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            text: "Valid text",
        }],
    },
};

/**
 * Helper function to transform form data to API input
 */
export const transformCommentFormToApiInput = (formData: Partial<CommentShape>): CommentShape => {
    if (!formData.commentedOn) {
        throw new Error("commentedOn is required for comment creation");
    }

    return {
        __typename: "Comment",
        id: formData.id || DUMMY_ID,
        commentedOn: formData.commentedOn,
        threadId: formData.threadId || null,
        translations: formData.translations || [],
    };
};

/**
 * Helper function to validate comment content
 */
export const validateCommentContent = (translations: CommentTranslationShape[]): string | null => {
    if (!translations || translations.length === 0) {
        return "At least one translation is required";
    }

    for (const translation of translations) {
        if (!translation.text || !translation.text.trim()) {
            return `Comment text cannot be empty for ${translation.language || "unknown"} language`;
        }
        if (translation.text.length > 32768) {
            return `Comment text is too long for ${translation.language || "unknown"} language (max 32,768 characters)`;
        }
        if (!translation.language) {
            return "Language code is required for all translations";
        }
    }

    return null;
};

/**
 * Helper function to count words in comment
 */
export const countCommentWords = (text: string): number => {
    return text.trim().split(/\s+/).filter(word => word.length > 0).length;
};

/**
 * Helper function to check if comment is a reply
 */
export const isReplyComment = (comment: Partial<CommentShape>): boolean => {
    return Boolean(comment.threadId);
};

/**
 * Mock form states for testing
 */
export const commentFormStates = {
    pristine: {
        values: { translations: [{ language: "en", text: "" }] },
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    typing: {
        values: { translations: [{ language: "en", text: "This is my comm" }] },
        errors: {},
        touched: { translations: [{ text: true }] },
        isValid: true,
        isSubmitting: false,
    },
    withErrors: {
        values: { translations: [{ language: "en", text: "" }] },
        errors: { translations: [{ text: "Comment text cannot be empty" }] },
        touched: { translations: [{ text: true }] },
        isValid: false,
        isSubmitting: false,
    },
    submitting: {
        values: { translations: [{ language: "en", text: "Submitting this comment..." }] },
        errors: {},
        touched: { translations: [{ text: true }] },
        isValid: true,
        isSubmitting: true,
    },
    submitted: {
        values: { translations: [{ language: "en", text: "" }] }, // Reset after successful submit
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
};

/**
 * Comment form scenarios for different contexts
 */
export const commentFormScenarios = {
    issueDiscussion: {
        commentedOn: { __typename: CommentFor.Issue as const, id: "678901234567890123" },
        threadId: null,
        translations: [{
            __typename: "CommentTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            text: "I've encountered this issue as well. Here's what I found...",
        }],
    },
    codeReview: {
        commentedOn: { __typename: CommentFor.PullRequest as const, id: "789012345678901234" },
        threadId: "890123456789012345",
        translations: [{
            __typename: "CommentTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            text: "Consider using a more efficient algorithm here. For example...",
        }],
    },
    resourceFeedback: {
        commentedOn: { __typename: CommentFor.ResourceVersion as const, id: "901234567890123456" },
        threadId: null,
        translations: [{
            __typename: "CommentTranslation" as const,
            id: DUMMY_ID,
            language: "en",
            text: "This tutorial was exactly what I needed! One suggestion though...",
        }],
    },
};