/**
 * Comment Form Test Fixtures
 * 
 * Test data for Comment forms used in both UI and integration tests.
 */

import { CommentFor } from "../../../api/types.js";
import type { CommentShape } from "../../../shape/models/models.js";
import type { FormFixtures } from "../../../forms/configs/types.js";

/**
 * Test fixtures for Comment forms
 */
export const commentFormFixtures: FormFixtures<CommentShape> = {
    configType: "Comment",
    
    valid: {
        minimal: {
            __typename: "Comment" as const,
            id: "comment_minimal",
            commentedOn: { __typename: CommentFor.Issue, id: "issue_123" },
            translations: [{
                __typename: "CommentTranslation" as const,
                id: "trans_minimal",
                language: "en",
                text: "Great project!",
            }],
        },
        
        complete: {
            __typename: "Comment" as const,
            id: "comment_complete",
            commentedOn: { __typename: CommentFor.ResourceVersion, id: "resourceversion_456" },
            threadId: "thread_123",
            translations: [{
                __typename: "CommentTranslation" as const,
                id: "trans_complete",
                language: "en",
                text: "This is an excellent project with comprehensive documentation. It includes detailed setup instructions and API references.",
            }],
        },
        
        withMultipleTranslations: {
            __typename: "Comment" as const,
            id: "comment_multi_lang",
            commentedOn: { __typename: CommentFor.PullRequest, id: "pr_789" },
            translations: [
                {
                    __typename: "CommentTranslation" as const,
                    id: "trans_en",
                    language: "en",
                    text: "Excellent implementation!",
                },
                {
                    __typename: "CommentTranslation" as const,
                    id: "trans_es",
                    language: "es",
                    text: "¡Excelente implementación!",
                },
            ],
        },
    },
    
    invalid: {
        missingRequired: {
            __typename: "Comment" as const,
            id: "comment_missing",
            commentedOn: { __typename: CommentFor.Issue, id: "issue_123" },
            translations: [], // Invalid: no translations provided
        },
        
        invalidValues: {
            __typename: "Comment" as const,
            id: "comment_invalid",
            commentedOn: { __typename: CommentFor.Issue, id: "issue_123" },
            translations: [{
                __typename: "CommentTranslation" as const,
                id: "trans_invalid",
                language: "en",
                text: "", // Invalid: empty text
            }],
        },
        
        missingCommentedOn: {
            __typename: "Comment" as const,
            id: "comment_no_target",
            commentedOn: { __typename: CommentFor.Issue, id: "" }, // Invalid: empty ID
            translations: [{
                __typename: "CommentTranslation" as const,
                id: "trans_no_target",
                language: "en",
                text: "This comment has no target",
            }],
        },
    },
    
    edge: {
        maxValues: {
            __typename: "Comment" as const,
            id: "comment_max",
            commentedOn: { __typename: CommentFor.Issue, id: "issue_123" },
            translations: [{
                __typename: "CommentTranslation" as const,
                id: "trans_max",
                language: "en",
                text: "A".repeat(32768), // Maximum allowed text length
            }],
        },
        
        minValues: {
            __typename: "Comment" as const,
            id: "comment_min",
            commentedOn: { __typename: CommentFor.Issue, id: "issue_123" },
            translations: [{
                __typename: "CommentTranslation" as const,
                id: "trans_min",
                language: "en",
                text: "A", // Minimum valid text (1 character)
            }],
        },
        
        overMaxLength: {
            __typename: "Comment" as const,
            id: "comment_over_max",
            commentedOn: { __typename: CommentFor.Issue, id: "issue_123" },
            translations: [{
                __typename: "CommentTranslation" as const,
                id: "trans_over_max",
                language: "en",
                text: "A".repeat(32769), // Over maximum length
            }],
        },
    },
};
