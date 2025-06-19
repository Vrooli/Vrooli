import { type Comment, type CommentThread, type CommentTranslation, type CommentYou, type Issue, type PullRequest, type ResourceVersion } from "@vrooli/shared";
import { minimalUserResponse, completeUserResponse, botUserResponse } from "./userResponses.js";
import { minimalTeamResponse } from "./teamResponses.js";

/**
 * API response fixtures for comments
 * These represent what components receive from API calls
 */

/**
 * Mock objects that comments can be attached to
 */
const mockIssue: Issue = {
    __typename: "Issue",
    id: "issue_123456789012345",
    bookmarkedBy: [],
    bookmarks: 5,
    closedAt: null,
    closedBy: null,
    comments: [], // Will be populated with comment fixtures
    commentsCount: 3,
    createdAt: "2024-01-01T00:00:00Z",
    createdBy: minimalUserResponse,
    issueFor: "Resource",
    referencedVersionId: null,
    reports: [],
    reportsCount: 0,
    score: 12,
    status: "Open",
    to: {
        __typename: "Resource",
        id: "resource_123456789",
        handle: "example-resource",
        createdAt: "2024-01-01T00:00:00Z",
        updatedAt: "2024-01-01T00:00:00Z",
    } as any, // Simplified for comment testing
    translations: [
        {
            __typename: "IssueTranslation",
            id: "issuetrans_123456789",
            language: "en",
            name: "Bug in data processing",
            description: "There seems to be an issue with how the data is being processed in the routine.",
        },
    ],
    translationsCount: 1,
    updatedAt: "2024-01-01T00:00:00Z",
    views: 45,
    you: {
        __typename: "IssueYou",
        canBookmark: true,
        canComment: true,
        canDelete: false,
        canReact: true,
        canReport: true,
        canUpdate: false,
        isBookmarked: false,
        reaction: null,
    },
};

const mockPullRequest: PullRequest = {
    __typename: "PullRequest",
    id: "pr_123456789012345",
    comments: [], // Will be populated with comment fixtures
    commentsCount: 2,
    createdBy: completeUserResponse,
    createdAt: "2024-01-15T00:00:00Z",
    from: {
        __typename: "ResourceVersion",
        id: "resver_from_123456789",
    } as any, // Simplified for comment testing
    closedAt: null,
    publicId: "PR-001",
    status: "Open",
    to: {
        __typename: "Resource",
        id: "resource_to_123456789",
    } as any, // Simplified for comment testing
    translations: [
        {
            __typename: "CommentTranslation",
            id: "prtrans_123456789",
            language: "en",
            text: "Add new feature for enhanced data processing capabilities",
        },
    ],
    translationsCount: 1,
    updatedAt: "2024-01-15T00:00:00Z",
    you: {
        __typename: "PullRequestYou",
        canComment: true,
        canDelete: true,
        canReport: false,
        canUpdate: true,
    },
};

const mockResourceVersion: ResourceVersion = {
    __typename: "ResourceVersion",
    id: "resver_123456789012345",
    codeLanguage: "typescript",
    comments: [], // Will be populated with comment fixtures
    commentsCount: 8,
    completedAt: "2024-01-01T00:00:00Z",
    complexity: 5,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-20T00:00:00Z",
    isAutomatable: true,
    isComplete: true,
    isLatest: true,
    isPrivate: false,
    publicId: "RV-001",
    pullRequest: null,
    relatedVersions: [],
    reports: [],
    reportsCount: 0,
    root: {
        __typename: "Resource",
        id: "resource_root_123456",
    } as any, // Simplified for comment testing
    timesCompleted: 150,
    timesStarted: 200,
    translations: [
        {
            __typename: "ResourceVersionTranslation",
            id: "resvtrans_123456789",
            language: "en",
            name: "Data Processing Pipeline v1.0",
            description: "A comprehensive data processing pipeline for analyzing user data",
            instructions: "1. Input your data\n2. Configure processing parameters\n3. Run the pipeline",
            details: "This version includes enhanced error handling and performance optimizations.",
        },
    ],
    translationsCount: 1,
    versionIndex: 0,
    versionLabel: "1.0.0",
    versionNotes: "Initial stable release",
    you: {
        __typename: "ResourceVersionYou",
        canBookmark: true,
        canComment: true,
        canCopy: true,
        canDelete: false,
        canReact: true,
        canReport: true,
        canUpdate: false,
        canUse: true,
        canRead: true,
        isBookmarked: false,
        reaction: null,
    },
};

/**
 * Minimal comment API response
 */
export const minimalCommentResponse: Comment = {
    __typename: "Comment",
    id: "comment_123456789012345",
    bookmarkedBy: [],
    bookmarks: 0,
    commentedOn: mockIssue,
    createdAt: "2024-01-20T10:00:00Z",
    owner: minimalUserResponse,
    reports: [],
    reportsCount: 0,
    score: 0,
    translations: [
        {
            __typename: "CommentTranslation",
            id: "commenttrans_123456789",
            language: "en",
            text: "I can confirm this issue. I'm experiencing the same problem.",
        },
    ],
    translationsCount: 1,
    updatedAt: "2024-01-20T10:00:00Z",
    you: {
        __typename: "CommentYou",
        canBookmark: true,
        canDelete: false,
        canReact: true,
        canReply: true,
        canReport: true,
        canUpdate: false,
        isBookmarked: false,
        reaction: null,
    },
};

/**
 * Complete comment API response with all fields
 */
export const completeCommentResponse: Comment = {
    __typename: "Comment",
    id: "comment_987654321098765",
    bookmarkedBy: [completeUserResponse],
    bookmarks: 8,
    commentedOn: mockResourceVersion,
    createdAt: "2024-01-18T14:30:00Z",
    owner: completeUserResponse,
    reports: [],
    reportsCount: 0,
    score: 15,
    translations: [
        {
            __typename: "CommentTranslation",
            id: "commenttrans_987654321",
            language: "en",
            text: "Excellent implementation! The error handling is particularly well done. I've tested this with various edge cases and it handles them gracefully.\n\nOne suggestion: consider adding more detailed logging for debugging purposes. Other than that, this looks ready for production.",
        },
        {
            __typename: "CommentTranslation",
            id: "commenttrans_876543210",
            language: "es",
            text: "¡Excelente implementación! El manejo de errores está particularmente bien hecho.",
        },
    ],
    translationsCount: 2,
    updatedAt: "2024-01-18T15:00:00Z",
    you: {
        __typename: "CommentYou",
        canBookmark: true,
        canDelete: true, // Own comment
        canReact: true,
        canReply: true,
        canReport: false, // Can't report own comment
        canUpdate: true, // Own comment
        isBookmarked: true,
        reaction: "like",
    },
};

/**
 * Bot comment response
 */
export const botCommentResponse: Comment = {
    __typename: "Comment",
    id: "comment_bot_123456789",
    bookmarkedBy: [],
    bookmarks: 3,
    commentedOn: mockIssue,
    createdAt: "2024-01-20T11:00:00Z",
    owner: botUserResponse,
    reports: [],
    reportsCount: 0,
    score: 12,
    translations: [
        {
            __typename: "CommentTranslation",
            id: "commenttrans_bot_123",
            language: "en",
            text: "I've analyzed the code and identified the root cause of this issue. The problem occurs in the data validation step when handling null values.\n\nHere's a suggested fix:\n\n```typescript\nif (data && data.value !== null) {\n  // Process data\n}\n```\n\nThis should resolve the issue you're experiencing.",
        },
    ],
    translationsCount: 1,
    updatedAt: "2024-01-20T11:00:00Z",
    you: {
        __typename: "CommentYou",
        canBookmark: true,
        canDelete: false,
        canReact: true,
        canReply: true,
        canReport: true,
        canUpdate: false,
        isBookmarked: false,
        reaction: "helpful",
    },
};

/**
 * Reply comment (child comment)
 */
export const replyCommentResponse: Comment = {
    __typename: "Comment",
    id: "comment_reply_123456789",
    bookmarkedBy: [],
    bookmarks: 1,
    commentedOn: mockIssue,
    createdAt: "2024-01-20T12:00:00Z",
    owner: minimalUserResponse,
    reports: [],
    reportsCount: 0,
    score: 5,
    translations: [
        {
            __typename: "CommentTranslation",
            id: "commenttrans_reply_123",
            language: "en",
            text: "Thanks for the solution! That fixed it perfectly. The validation is working as expected now.",
        },
    ],
    translationsCount: 1,
    updatedAt: "2024-01-20T12:00:00Z",
    you: {
        __typename: "CommentYou",
        canBookmark: true,
        canDelete: false,
        canReact: true,
        canReply: true,
        canReport: true,
        canUpdate: false,
        isBookmarked: false,
        reaction: null,
    },
};

/**
 * Deleted comment response
 */
export const deletedCommentResponse: Comment = {
    __typename: "Comment",
    id: "comment_deleted_123456",
    bookmarkedBy: [],
    bookmarks: 0,
    commentedOn: mockPullRequest,
    createdAt: "2024-01-19T00:00:00Z",
    owner: null, // Deleted comments have null owner
    reports: [],
    reportsCount: 0,
    score: 0,
    translations: [
        {
            __typename: "CommentTranslation",
            id: "commenttrans_deleted_123",
            language: "en",
            text: "[This comment has been deleted]",
        },
    ],
    translationsCount: 1,
    updatedAt: "2024-01-19T09:00:00Z",
    you: {
        __typename: "CommentYou",
        canBookmark: false,
        canDelete: false,
        canReact: false,
        canReply: false,
        canReport: false,
        canUpdate: false,
        isBookmarked: false,
        reaction: null,
    },
};

/**
 * Pull request comment
 */
export const pullRequestCommentResponse: Comment = {
    __typename: "Comment",
    id: "comment_pr_123456789",
    bookmarkedBy: [],
    bookmarks: 2,
    commentedOn: mockPullRequest,
    createdAt: "2024-01-16T00:00:00Z",
    owner: completeUserResponse,
    reports: [],
    reportsCount: 0,
    score: 8,
    translations: [
        {
            __typename: "CommentTranslation",
            id: "commenttrans_pr_123",
            language: "en",
            text: "The implementation looks good overall. I have a few minor suggestions:\n\n1. Consider adding unit tests for the new functionality\n2. The error messages could be more descriptive\n3. Documentation should be updated to reflect these changes\n\nOtherwise, this is ready to merge!",
        },
    ],
    translationsCount: 1,
    updatedAt: "2024-01-16T00:00:00Z",
    you: {
        __typename: "CommentYou",
        canBookmark: true,
        canDelete: false,
        canReact: true,
        canReply: true,
        canReport: true,
        canUpdate: false,
        isBookmarked: false,
        reaction: null,
    },
};

/**
 * Reported comment
 */
export const reportedCommentResponse: Comment = {
    __typename: "Comment",
    id: "comment_reported_123456",
    bookmarkedBy: [],
    bookmarks: 0,
    commentedOn: mockIssue,
    createdAt: "2024-01-17T00:00:00Z",
    owner: {
        ...minimalUserResponse,
        id: "user_problematic_123456",
        handle: "problemuser",
        name: "Problem User",
    },
    reports: [
        {
            __typename: "Report",
            id: "report_123456789",
            createdAt: "2024-01-17T10:00:00Z",
            updatedAt: "2024-01-17T10:00:00Z",
        } as any, // Simplified for comment testing
    ],
    reportsCount: 2,
    score: -5,
    translations: [
        {
            __typename: "CommentTranslation",
            id: "commenttrans_reported_123",
            language: "en",
            text: "This content has been flagged for review.",
        },
    ],
    translationsCount: 1,
    updatedAt: "2024-01-17T10:00:00Z",
    you: {
        __typename: "CommentYou",
        canBookmark: false,
        canDelete: false,
        canReact: false,
        canReply: false,
        canReport: true,
        canUpdate: false,
        isBookmarked: false,
        reaction: null,
    },
};

/**
 * Comment thread structures
 */
export const minimalCommentThread: CommentThread = {
    __typename: "CommentThread",
    comment: minimalCommentResponse,
    childThreads: [],
    endCursor: null,
    totalInThread: 1,
};

export const completeCommentThread: CommentThread = {
    __typename: "CommentThread",
    comment: completeCommentResponse,
    childThreads: [
        {
            __typename: "CommentThread",
            comment: replyCommentResponse,
            childThreads: [],
            endCursor: null,
            totalInThread: 1,
        },
    ],
    endCursor: "cursor_thread_end",
    totalInThread: 2,
};

export const nestedCommentThread: CommentThread = {
    __typename: "CommentThread",
    comment: botCommentResponse,
    childThreads: [
        {
            __typename: "CommentThread",
            comment: replyCommentResponse,
            childThreads: [
                {
                    __typename: "CommentThread",
                    comment: {
                        ...minimalCommentResponse,
                        id: "comment_nested_123456",
                        translations: [
                            {
                                __typename: "CommentTranslation",
                                id: "commenttrans_nested_123",
                                language: "en",
                                text: "Great! Glad it worked out.",
                            },
                        ],
                    },
                    childThreads: [],
                    endCursor: null,
                    totalInThread: 1,
                },
            ],
            endCursor: "cursor_nested_end",
            totalInThread: 2,
        },
    ],
    endCursor: "cursor_parent_end",
    totalInThread: 4,
};

/**
 * Comment variant states for testing
 */
export const commentResponseVariants = {
    minimal: minimalCommentResponse,
    complete: completeCommentResponse,
    bot: botCommentResponse,
    reply: replyCommentResponse,
    deleted: deletedCommentResponse,
    pullRequest: pullRequestCommentResponse,
    reported: reportedCommentResponse,
    onIssue: {
        ...minimalCommentResponse,
        id: "comment_issue_123456",
        commentedOn: mockIssue,
    },
    onResourceVersion: {
        ...minimalCommentResponse,
        id: "comment_resource_123456",
        commentedOn: mockResourceVersion,
    },
    withReactions: {
        ...completeCommentResponse,
        id: "comment_reactions_123456",
        score: 25,
        bookmarks: 12,
        you: {
            ...completeCommentResponse.you,
            reaction: "love",
            isBookmarked: true,
        },
    },
    longText: {
        ...minimalCommentResponse,
        id: "comment_long_123456",
        translations: [
            {
                __typename: "CommentTranslation",
                id: "commenttrans_long_123",
                language: "en",
                text: "This is a very long comment that demonstrates how the UI handles extensive text content. ".repeat(10) + 
                      "It includes multiple paragraphs, code examples, and detailed explanations that would typically be found in technical discussions.\n\n" +
                      "Here's some example code:\n\n```typescript\nconst processData = async (input: DataType) => {\n  try {\n    const result = await transformer.process(input);\n    return { success: true, data: result };\n  } catch (error) {\n    logger.error('Processing failed:', error);\n    throw new ProcessingError(error.message);\n  }\n};\n```\n\n" +
                      "The implementation above shows proper error handling and logging practices that should be followed in production code.",
            },
        ],
    },
} as const;

/**
 * Comment thread variants for testing different structures
 */
export const commentThreadVariants = {
    minimal: minimalCommentThread,
    complete: completeCommentThread,
    nested: nestedCommentThread,
    singleReply: {
        __typename: "CommentThread",
        comment: completeCommentResponse,
        childThreads: [
            {
                __typename: "CommentThread",
                comment: botCommentResponse,
                childThreads: [],
                endCursor: null,
                totalInThread: 1,
            },
        ],
        endCursor: null,
        totalInThread: 2,
    },
    multipleReplies: {
        __typename: "CommentThread",
        comment: minimalCommentResponse,
        childThreads: [
            {
                __typename: "CommentThread",
                comment: replyCommentResponse,
                childThreads: [],
                endCursor: null,
                totalInThread: 1,
            },
            {
                __typename: "CommentThread",
                comment: botCommentResponse,
                childThreads: [],
                endCursor: null,
                totalInThread: 1,
            },
            {
                __typename: "CommentThread",
                comment: pullRequestCommentResponse,
                childThreads: [],
                endCursor: null,
                totalInThread: 1,
            },
        ],
        endCursor: "cursor_multiple_end",
        totalInThread: 4,
    },
} as const;

/**
 * Comment search response
 */
export const commentSearchResponse = {
    __typename: "CommentSearchResult",
    endCursor: "cursor_search_end",
    threads: [
        commentThreadVariants.complete,
        commentThreadVariants.minimal,
        commentThreadVariants.singleReply,
    ],
    totalThreads: 25,
};

/**
 * Loading and error states for UI testing
 */
export const commentUIStates = {
    loading: null,
    error: {
        code: "COMMENT_NOT_FOUND",
        message: "The requested comment could not be found",
    },
    createError: {
        code: "COMMENT_CREATE_FAILED",
        message: "Failed to create comment. Please try again.",
    },
    updateError: {
        code: "COMMENT_UPDATE_FAILED",
        message: "Failed to update comment. Please check your permissions.",
    },
    deleteError: {
        code: "COMMENT_DELETE_FAILED",
        message: "Failed to delete comment. Please try again.",
    },
    permissionError: {
        code: "COMMENT_PERMISSION_DENIED",
        message: "You don't have permission to perform this action on this comment",
    },
    empty: {
        __typename: "CommentSearchResult",
        endCursor: null,
        threads: [],
        totalThreads: 0,
    },
};

/**
 * Array of comments for bulk testing
 */
export const commentArrayResponse = [
    commentResponseVariants.minimal,
    commentResponseVariants.complete,
    commentResponseVariants.bot,
    commentResponseVariants.reply,
    commentResponseVariants.pullRequest,
];

/**
 * Comments grouped by context (for testing different comment contexts)
 */
export const commentsByContext = {
    issueComments: [
        commentResponseVariants.minimal,
        commentResponseVariants.bot,
        commentResponseVariants.reply,
    ],
    pullRequestComments: [
        commentResponseVariants.pullRequest,
        commentResponseVariants.complete,
    ],
    resourceVersionComments: [
        commentResponseVariants.onResourceVersion,
        commentResponseVariants.withReactions,
        commentResponseVariants.longText,
    ],
} as const;