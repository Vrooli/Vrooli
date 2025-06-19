import { type Api, type ApiVersion, type ApiVersionYou, type ApiYou, type ResourceSubTypeApi } from "@vrooli/shared";
import { minimalUserResponse, completeUserResponse } from "./userResponses.js";
import { minimalTeamResponse, completeTeamResponse } from "./teamResponses.js";

/**
 * API response fixtures for APIs
 * These represent what components receive from API calls
 */

/**
 * Minimal API version response
 */
const minimalApiVersionResponse: ApiVersion = {
    __typename: "ApiVersion",
    id: "apiver_123456789012345678",
    versionLabel: "1.0.0",
    isComplete: true,
    isDeleted: false,
    isLatest: true,
    isPrivate: false,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    completedAt: "2024-01-01T00:00:00Z",
    translations: [],
    resourceVersionTo: [],
    resourceVersionFrom: [],
    you: {
        __typename: "ApiVersionYou",
        canComment: true,
        canDelete: false,
        canReport: true,
        canUpdate: false,
        canRead: true,
        canBookmark: true,
        canReact: true,
        isBookmarked: false,
        reaction: null,
    },
};

/**
 * Complete API version response with all fields
 */
const completeApiVersionResponse: ApiVersion = {
    __typename: "ApiVersion",
    id: "apiver_987654321098765432",
    versionLabel: "2.1.0",
    versionNotes: "Added support for batch operations",
    isComplete: true,
    isDeleted: false,
    isLatest: true,
    isPrivate: false,
    createdAt: "2023-06-15T10:30:00Z",
    updatedAt: "2024-01-15T14:45:00Z",
    completedAt: "2023-07-01T09:00:00Z",
    callLink: "https://api.example.com/v2",
    documentationLink: "https://docs.example.com/api/v2",
    schemaText: JSON.stringify({
        openapi: "3.0.0",
        info: {
            title: "Example API",
            version: "2.1.0",
        },
        paths: {
            "/users": {
                get: {
                    summary: "List users",
                    responses: {
                        "200": { description: "Success" },
                    },
                },
            },
        },
    }),
    translations: [
        {
            __typename: "ApiVersionTranslation",
            id: "apitrans_123456789012345",
            language: "en",
            details: "RESTful API for user management",
            name: "User Management API",
            summary: "Manage users, teams, and permissions",
        },
        {
            __typename: "ApiVersionTranslation",
            id: "apitrans_987654321098765",
            language: "es",
            details: "API RESTful para gestión de usuarios",
            name: "API de Gestión de Usuarios",
            summary: "Gestiona usuarios, equipos y permisos",
        },
    ],
    resourceVersionTo: [],
    resourceVersionFrom: [],
    pullRequest: null,
    commentsCount: 12,
    forksCount: 3,
    reportsCount: 0,
    directoriesCount: 5,
    filesCount: 20,
    subVersionsCount: 2,
    you: {
        __typename: "ApiVersionYou",
        canComment: true,
        canDelete: false,
        canReport: true,
        canUpdate: true,
        canRead: true,
        canBookmark: true,
        canReact: true,
        isBookmarked: true,
        reaction: "like",
    },
};

/**
 * Minimal API response
 */
export const minimalApiResponse: Api = {
    __typename: "Api",
    id: "api_123456789012345678",
    publicId: "api-123456",
    isPrivate: false,
    permissions: JSON.stringify(["Read"]),
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    hasCompleteVersion: true,
    score: 0,
    bookmarksCount: 0,
    versionsCount: 1,
    viewsCount: 0,
    tags: [],
    transfersCount: 0,
    labels: [],
    versions: [minimalApiVersionResponse],
    you: {
        __typename: "ApiYou",
        canComment: true,
        canDelete: false,
        canReport: true,
        canUpdate: false,
        canRead: true,
        canBookmark: true,
        canReact: true,
        canTransfer: false,
        isBookmarked: false,
        reaction: null,
    },
};

/**
 * Complete API response with all fields
 */
export const completeApiResponse: Api = {
    __typename: "Api",
    id: "api_987654321098765432",
    publicId: "user-api-v2",
    isPrivate: false,
    permissions: JSON.stringify(["Read", "Write"]),
    createdAt: "2023-06-15T10:30:00Z",
    updatedAt: "2024-01-15T14:45:00Z",
    hasCompleteVersion: true,
    score: 42,
    bookmarksCount: 25,
    versionsCount: 3,
    viewsCount: 1250,
    owner: completeUserResponse,
    parent: null,
    tags: [
        {
            __typename: "Tag",
            id: "tag_123456789012345",
            tag: "rest-api",
            translations: [],
        },
        {
            __typename: "Tag",
            id: "tag_987654321098765",
            tag: "user-management",
            translations: [],
        },
    ],
    transfersCount: 0,
    labels: [
        {
            __typename: "Label",
            id: "label_123456789012345",
            label: "Production Ready",
            color: "#00ff00",
        },
    ],
    versions: [completeApiVersionResponse],
    you: {
        __typename: "ApiYou",
        canComment: true,
        canDelete: false,
        canReport: true,
        canUpdate: true,
        canRead: true,
        canBookmark: true,
        canReact: true,
        canTransfer: false,
        isBookmarked: true,
        reaction: "like",
    },
};

/**
 * Private API response
 */
export const privateApiResponse: Api = {
    __typename: "Api",
    id: "api_private_123456789",
    publicId: "private-api",
    isPrivate: true,
    permissions: JSON.stringify([]),
    createdAt: "2023-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    hasCompleteVersion: true,
    score: 0,
    bookmarksCount: 0,
    versionsCount: 1,
    viewsCount: 0,
    tags: [],
    transfersCount: 0,
    labels: [],
    versions: [
        {
            ...minimalApiVersionResponse,
            id: "apiver_private_123456789",
            isPrivate: true,
        },
    ],
    you: {
        __typename: "ApiYou",
        canComment: false,
        canDelete: false,
        canReport: false,
        canUpdate: false,
        canRead: false, // No access to private API
        canBookmark: false,
        canReact: false,
        canTransfer: false,
        isBookmarked: false,
        reaction: null,
    },
};

/**
 * Team-owned API response
 */
export const teamOwnedApiResponse: Api = {
    __typename: "Api",
    id: "api_team_123456789",
    publicId: "team-api",
    isPrivate: false,
    permissions: JSON.stringify(["Read"]),
    createdAt: "2023-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    hasCompleteVersion: true,
    score: 15,
    bookmarksCount: 8,
    versionsCount: 2,
    viewsCount: 450,
    owner: minimalTeamResponse,
    tags: [],
    transfersCount: 0,
    labels: [],
    versions: [minimalApiVersionResponse],
    you: {
        __typename: "ApiYou",
        canComment: true,
        canDelete: false,
        canReport: true,
        canUpdate: false, // Not a team member
        canRead: true,
        canBookmark: true,
        canReact: true,
        canTransfer: false,
        isBookmarked: false,
        reaction: null,
    },
};

/**
 * API with no versions (incomplete)
 */
export const incompleteApiResponse: Api = {
    __typename: "Api",
    id: "api_incomplete_123456789",
    publicId: "incomplete-api",
    isPrivate: false,
    permissions: JSON.stringify(["Read"]),
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    hasCompleteVersion: false,
    score: 0,
    bookmarksCount: 0,
    versionsCount: 0,
    viewsCount: 0,
    tags: [],
    transfersCount: 0,
    labels: [],
    versions: [],
    you: {
        __typename: "ApiYou",
        canComment: false, // Can't comment on incomplete
        canDelete: false,
        canReport: true,
        canUpdate: false,
        canRead: true,
        canBookmark: true,
        canReact: false, // Can't react to incomplete
        canTransfer: false,
        isBookmarked: false,
        reaction: null,
    },
};

/**
 * API variant states for testing
 */
export const apiResponseVariants = {
    minimal: minimalApiResponse,
    complete: completeApiResponse,
    private: privateApiResponse,
    teamOwned: teamOwnedApiResponse,
    incomplete: incompleteApiResponse,
    withManyVersions: {
        ...completeApiResponse,
        id: "api_versions_123456789",
        versionsCount: 10,
        versions: [
            completeApiVersionResponse,
            {
                ...minimalApiVersionResponse,
                id: "apiver_old_123456789",
                versionLabel: "1.5.0",
                isLatest: false,
            },
        ],
    },
    popular: {
        ...minimalApiResponse,
        id: "api_popular_123456789",
        score: 100,
        bookmarksCount: 150,
        viewsCount: 5000,
    },
} as const;

/**
 * API search response
 */
export const apiSearchResponse = {
    __typename: "ApiSearchResult",
    edges: [
        {
            __typename: "ApiEdge",
            cursor: "cursor_1",
            node: apiResponseVariants.complete,
        },
        {
            __typename: "ApiEdge",
            cursor: "cursor_2",
            node: apiResponseVariants.minimal,
        },
        {
            __typename: "ApiEdge",
            cursor: "cursor_3",
            node: apiResponseVariants.teamOwned,
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

/**
 * Loading and error states for UI testing
 */
export const apiUIStates = {
    loading: null,
    error: {
        code: "API_NOT_FOUND",
        message: "The requested API could not be found",
    },
    invalidSchema: {
        code: "API_SCHEMA_INVALID",
        message: "The API schema is not valid OpenAPI/Swagger format",
    },
    rateLimited: {
        code: "API_RATE_LIMITED",
        message: "API rate limit exceeded. Please try again later.",
    },
    empty: {
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