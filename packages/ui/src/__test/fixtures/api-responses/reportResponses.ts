import { type Report, type ReportResponse, type ReportYou, type ReportResponseYou, type Comment, type Issue, type User, type Team, ReportFor, ReportStatus, ReportSuggestedAction } from "@vrooli/shared";
import { minimalUserResponse, completeUserResponse, botUserResponse } from "./userResponses.js";
import { minimalTeamResponse } from "./teamResponses.js";

/**
 * API response fixtures for reports
 * These represent what components receive from API calls
 */

/**
 * Mock objects that reports can be created for
 */
const mockReportedComment: Comment = {
    __typename: "Comment",
    id: "comment_reported_123456789",
    bookmarkedBy: [],
    bookmarks: 0,
    commentedOn: null as any, // Simplified for report testing
    createdAt: "2024-01-15T00:00:00Z",
    owner: {
        ...minimalUserResponse,
        id: "user_reported_123456",
        handle: "reporteduser",
        name: "Reported User",
    },
    reports: [],
    reportsCount: 1,
    score: -2,
    translations: [
        {
            __typename: "CommentTranslation",
            id: "commenttrans_reported_123",
            language: "en",
            text: "This is inappropriate content that violates community guidelines.",
        },
    ],
    translationsCount: 1,
    updatedAt: "2024-01-15T00:00:00Z",
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

const mockReportedUser: User = {
    ...minimalUserResponse,
    id: "user_spam_123456",
    handle: "spamuser",
    name: "Spam User",
    reportsReceivedCount: 5,
};

const mockReportedTeam: Team = {
    ...minimalTeamResponse,
    id: "team_inappropriate_123456",
    handle: "badteam",
    name: "Inappropriate Team",
    reportsReceivedCount: 2,
};

/**
 * Minimal report API response
 */
export const minimalReportResponse: Report = {
    __typename: "Report",
    id: "report_123456789012345678",
    createdFor: ReportFor.Comment,
    createdAt: "2024-01-20T10:00:00Z",
    details: null,
    language: "en",
    publicId: "RPT-001",
    reason: "spam",
    responses: [],
    responsesCount: 0,
    status: ReportStatus.Open,
    updatedAt: "2024-01-20T10:00:00Z",
    you: {
        __typename: "ReportYou",
        canDelete: true, // Own report
        canRespond: false,
        canUpdate: true, // Own report
        isOwn: true,
    },
};

/**
 * Complete report API response with all fields
 */
export const completeReportResponse: Report = {
    __typename: "Report",
    id: "report_987654321098765432",
    createdFor: ReportFor.User,
    createdAt: "2024-01-18T14:30:00Z",
    details: "This user has been posting spam content repeatedly across multiple threads. They seem to be promoting commercial services which violates the community guidelines.",
    language: "en",
    publicId: "RPT-025",
    reason: "spam",
    responses: [
        {
            __typename: "ReportResponse",
            id: "reportresponse_123456789",
            actionSuggested: ReportSuggestedAction.SuspendUser,
            createdAt: "2024-01-19T09:00:00Z",
            details: "Reviewed the user's activity. Confirmed spam behavior. Recommending suspension.",
            language: "en",
            report: null as any, // Circular reference simplified
            updatedAt: "2024-01-19T09:00:00Z",
            you: {
                __typename: "ReportResponseYou",
                canDelete: false,
                canUpdate: false,
            },
        },
    ],
    responsesCount: 1,
    status: ReportStatus.Open,
    updatedAt: "2024-01-19T09:00:00Z",
    you: {
        __typename: "ReportYou",
        canDelete: false,
        canRespond: false,
        canUpdate: false,
        isOwn: false,
    },
};

/**
 * Harassment report response
 */
export const harassmentReportResponse: Report = {
    __typename: "Report",
    id: "report_harassment_123456789",
    createdFor: ReportFor.Comment,
    createdAt: "2024-01-17T16:45:00Z",
    details: "This comment contains personal attacks and harassment directed at another community member.",
    language: "en",
    publicId: "RPT-015",
    reason: "harassment",
    responses: [],
    responsesCount: 0,
    status: ReportStatus.Open,
    updatedAt: "2024-01-17T16:45:00Z",
    you: {
        __typename: "ReportYou",
        canDelete: true,
        canRespond: false,
        canUpdate: true,
        isOwn: true,
    },
};

/**
 * Inappropriate content report response
 */
export const inappropriateContentReportResponse: Report = {
    __typename: "Report",
    id: "report_inappropriate_123456789",
    createdFor: ReportFor.Team,
    createdAt: "2024-01-16T12:20:00Z",
    details: "Team profile contains inappropriate imagery and offensive language in the description.",
    language: "en",
    publicId: "RPT-012",
    reason: "inappropriate_content",
    responses: [
        {
            __typename: "ReportResponse",
            id: "reportresponse_inappropriate_123",
            actionSuggested: ReportSuggestedAction.HideUntilFixed,
            createdAt: "2024-01-16T15:30:00Z",
            details: "Content has been reviewed and found to violate guidelines. Team profile should be hidden until corrected.",
            language: "en",
            report: null as any, // Circular reference simplified
            updatedAt: "2024-01-16T15:30:00Z",
            you: {
                __typename: "ReportResponseYou",
                canDelete: false,
                canUpdate: false,
            },
        },
    ],
    responsesCount: 1,
    status: ReportStatus.ClosedHidden,
    updatedAt: "2024-01-16T15:30:00Z",
    you: {
        __typename: "ReportYou",
        canDelete: false,
        canRespond: false,
        canUpdate: false,
        isOwn: false,
    },
};

/**
 * False report response (closed as false)
 */
export const falseReportResponse: Report = {
    __typename: "Report",
    id: "report_false_123456789",
    createdFor: ReportFor.Comment,
    createdAt: "2024-01-14T08:15:00Z",
    details: "This comment is offensive and should be removed.",
    language: "en",
    publicId: "RPT-008",
    reason: "inappropriate_content",
    responses: [
        {
            __typename: "ReportResponse",
            id: "reportresponse_false_123",
            actionSuggested: ReportSuggestedAction.FalseReport,
            createdAt: "2024-01-14T14:20:00Z",
            details: "After review, the reported content does not violate community guidelines. This appears to be a disagreement rather than a policy violation.",
            language: "en",
            report: null as any, // Circular reference simplified
            updatedAt: "2024-01-14T14:20:00Z",
            you: {
                __typename: "ReportResponseYou",
                canDelete: false,
                canUpdate: false,
            },
        },
    ],
    responsesCount: 1,
    status: ReportStatus.ClosedFalseReport,
    updatedAt: "2024-01-14T14:20:00Z",
    you: {
        __typename: "ReportYou",
        canDelete: false,
        canRespond: false,
        canUpdate: false,
        isOwn: false,
    },
};

/**
 * Resolved report with deletion action
 */
export const deletedContentReportResponse: Report = {
    __typename: "Report",
    id: "report_deleted_123456789",
    createdFor: ReportFor.Issue,
    createdAt: "2024-01-13T11:30:00Z",
    details: "Issue contains spam links and promotional content.",
    language: "en",
    publicId: "RPT-005",
    reason: "spam",
    responses: [
        {
            __typename: "ReportResponse",
            id: "reportresponse_deleted_123",
            actionSuggested: ReportSuggestedAction.Delete,
            createdAt: "2024-01-13T16:45:00Z",
            details: "Content confirmed as spam. Issue has been deleted and user has been warned.",
            language: "en",
            report: null as any, // Circular reference simplified
            updatedAt: "2024-01-13T16:45:00Z",
            you: {
                __typename: "ReportResponseYou",
                canDelete: false,
                canUpdate: false,
            },
        },
    ],
    responsesCount: 1,
    status: ReportStatus.ClosedDeleted,
    updatedAt: "2024-01-13T16:45:00Z",
    you: {
        __typename: "ReportYou",
        canDelete: false,
        canRespond: false,
        canUpdate: false,
        isOwn: false,
    },
};

/**
 * Report with multiple responses
 */
export const multipleResponsesReportResponse: Report = {
    __typename: "Report",
    id: "report_multiple_123456789",
    createdFor: ReportFor.User,
    createdAt: "2024-01-12T09:15:00Z",
    details: "User has been harassing multiple community members through direct messages and comments.",
    language: "en",
    publicId: "RPT-003",
    reason: "harassment",
    responses: [
        {
            __typename: "ReportResponse",
            id: "reportresponse_multi_1",
            actionSuggested: ReportSuggestedAction.NonIssue,
            createdAt: "2024-01-12T14:30:00Z",
            details: "Initial review suggests this may be a misunderstanding. Requesting more information.",
            language: "en",
            report: null as any, // Circular reference simplified
            updatedAt: "2024-01-12T14:30:00Z",
            you: {
                __typename: "ReportResponseYou",
                canDelete: false,
                canUpdate: false,
            },
        },
        {
            __typename: "ReportResponse",
            id: "reportresponse_multi_2",
            actionSuggested: ReportSuggestedAction.SuspendUser,
            createdAt: "2024-01-13T10:15:00Z",
            details: "After further investigation and additional reports, confirmed harassment pattern. Recommending suspension.",
            language: "en",
            report: null as any, // Circular reference simplified
            updatedAt: "2024-01-13T10:15:00Z",
            you: {
                __typename: "ReportResponseYou",
                canDelete: false,
                canUpdate: false,
            },
        },
    ],
    responsesCount: 2,
    status: ReportStatus.ClosedSuspended,
    updatedAt: "2024-01-13T10:15:00Z",
    you: {
        __typename: "ReportYou",
        canDelete: false,
        canRespond: false,
        canUpdate: false,
        isOwn: false,
    },
};

/**
 * Report with admin privileges (can respond)
 */
export const adminReportResponse: Report = {
    __typename: "Report",
    id: "report_admin_123456789",
    createdFor: ReportFor.Comment,
    createdAt: "2024-01-21T13:45:00Z",
    details: "Comment contains misinformation about platform policies.",
    language: "en",
    publicId: "RPT-030",
    reason: "misinformation",
    responses: [],
    responsesCount: 0,
    status: ReportStatus.Open,
    updatedAt: "2024-01-21T13:45:00Z",
    you: {
        __typename: "ReportYou",
        canDelete: true,
        canRespond: true, // Admin can respond
        canUpdate: true,
        isOwn: false,
    },
};

/**
 * Report variant states for testing
 */
export const reportResponseVariants = {
    minimal: minimalReportResponse,
    complete: completeReportResponse,
    harassment: harassmentReportResponse,
    inappropriateContent: inappropriateContentReportResponse,
    falseReport: falseReportResponse,
    deletedContent: deletedContentReportResponse,
    multipleResponses: multipleResponsesReportResponse,
    admin: adminReportResponse,
    spam: {
        ...minimalReportResponse,
        id: "report_spam_123456789",
        publicId: "RPT-040",
        reason: "spam",
        details: "User posting repetitive promotional content.",
    },
    copyright: {
        ...minimalReportResponse,
        id: "report_copyright_123456789",
        publicId: "RPT-041",
        reason: "copyright_violation",
        details: "Content appears to violate copyright restrictions.",
        createdFor: ReportFor.ResourceVersion,
    },
    misinformation: {
        ...completeReportResponse,
        id: "report_misinfo_123456789",
        publicId: "RPT-042",
        reason: "misinformation",
        details: "Post contains false information that could mislead users.",
        createdFor: ReportFor.Issue,
    },
    pending: {
        ...minimalReportResponse,
        id: "report_pending_123456789",
        publicId: "RPT-043",
        status: ReportStatus.Open,
        responses: [],
        responsesCount: 0,
    },
    resolved: {
        ...completeReportResponse,
        id: "report_resolved_123456789",
        publicId: "RPT-044",
        status: ReportStatus.ClosedNonIssue,
        responses: [
            {
                __typename: "ReportResponse",
                id: "reportresponse_resolved_123",
                actionSuggested: ReportSuggestedAction.NonIssue,
                createdAt: "2024-01-20T11:00:00Z",
                details: "Reviewed and determined not to be a policy violation.",
                language: "en",
                report: null as any,
                updatedAt: "2024-01-20T11:00:00Z",
                you: {
                    __typename: "ReportResponseYou",
                    canDelete: false,
                    canUpdate: false,
                },
            },
        ],
        responsesCount: 1,
    },
} as const;

/**
 * Report responses by type for testing different report categories
 */
export const reportsByType = {
    spam: [
        reportResponseVariants.spam,
        reportResponseVariants.deletedContent,
    ],
    harassment: [
        reportResponseVariants.harassment,
        reportResponseVariants.multipleResponses,
    ],
    inappropriateContent: [
        reportResponseVariants.inappropriateContent,
        reportResponseVariants.copyright,
    ],
    misinformation: [
        reportResponseVariants.misinformation,
    ],
} as const;

/**
 * Report responses by status for testing different states
 */
export const reportsByStatus = {
    open: [
        reportResponseVariants.minimal,
        reportResponseVariants.complete,
        reportResponseVariants.harassment,
        reportResponseVariants.admin,
        reportResponseVariants.pending,
    ],
    closed: [
        reportResponseVariants.falseReport,
        reportResponseVariants.deletedContent,
        reportResponseVariants.multipleResponses,
        reportResponseVariants.inappropriateContent,
        reportResponseVariants.resolved,
    ],
    closedDeleted: [
        reportResponseVariants.deletedContent,
    ],
    closedFalseReport: [
        reportResponseVariants.falseReport,
    ],
    closedHidden: [
        reportResponseVariants.inappropriateContent,
    ],
    closedSuspended: [
        reportResponseVariants.multipleResponses,
    ],
    closedNonIssue: [
        reportResponseVariants.resolved,
    ],
} as const;

/**
 * Report responses by permission level for testing access control
 */
export const reportsByPermission = {
    own: [
        reportResponseVariants.minimal,
        reportResponseVariants.harassment,
    ],
    viewOnly: [
        reportResponseVariants.complete,
        reportResponseVariants.falseReport,
        reportResponseVariants.deletedContent,
    ],
    admin: [
        reportResponseVariants.admin,
    ],
} as const;

/**
 * Loading and error states for UI testing
 */
export const reportUIStates = {
    loading: null,
    error: {
        code: "REPORT_NOT_FOUND",
        message: "The requested report could not be found",
    },
    createError: {
        code: "REPORT_CREATE_FAILED",
        message: "Failed to create report. Please try again.",
    },
    updateError: {
        code: "REPORT_UPDATE_FAILED",
        message: "Failed to update report. Please check your permissions.",
    },
    deleteError: {
        code: "REPORT_DELETE_FAILED",
        message: "Failed to delete report. Please try again.",
    },
    permissionError: {
        code: "REPORT_PERMISSION_DENIED",
        message: "You don't have permission to perform this action on this report",
    },
    empty: {
        __typename: "ReportSearchResult",
        edges: [],
        pageInfo: {
            __typename: "PageInfo",
            hasNextPage: false,
            hasPreviousPage: false,
            startCursor: null,
            endCursor: null,
        },
    },
};

/**
 * Array of reports for bulk testing
 */
export const reportArrayResponse = [
    reportResponseVariants.minimal,
    reportResponseVariants.complete,
    reportResponseVariants.harassment,
    reportResponseVariants.inappropriateContent,
    reportResponseVariants.falseReport,
    reportResponseVariants.deletedContent,
];

/**
 * Report search response
 */
export const reportSearchResponse = {
    __typename: "ReportSearchResult",
    edges: [
        {
            __typename: "ReportEdge",
            cursor: "cursor_1",
            node: reportResponseVariants.complete,
        },
        {
            __typename: "ReportEdge",
            cursor: "cursor_2",
            node: reportResponseVariants.harassment,
        },
        {
            __typename: "ReportEdge",
            cursor: "cursor_3",
            node: reportResponseVariants.inappropriateContent,
        },
    ],
    pageInfo: {
        __typename: "PageInfo",
        hasNextPage: true,
        hasPreviousPage: false,
        startCursor: "cursor_1",
        endCursor: "cursor_3",
    },
};