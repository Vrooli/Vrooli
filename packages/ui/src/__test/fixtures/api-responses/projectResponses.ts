import { type Project, type ProjectVersion, type ProjectVersionYou, type Label, type Tag } from "@vrooli/shared";
import { minimalUserResponse, completeUserResponse } from "./userResponses.js";
import { minimalTeamResponse } from "./teamResponses.js";

/**
 * API response fixtures for projects
 * These represent what components receive from API calls
 */

/**
 * Mock tag data
 */
const aiTag: Tag = {
    __typename: "Tag",
    id: "tag_ai_123456789",
    tag: "artificial-intelligence",
    created_at: "2024-01-01T00:00:00Z",
    bookmarks: 0,
    translations: [
        {
            __typename: "TagTranslation",
            id: "tagtrans_123456789",
            language: "en",
            description: "Related to artificial intelligence and machine learning",
        },
    ],
    you: {
        __typename: "TagYou",
        isBookmarked: false,
    },
};

const webDevTag: Tag = {
    __typename: "Tag",
    id: "tag_webdev_123456789",
    tag: "web-development",
    created_at: "2024-01-01T00:00:00Z",
    bookmarks: 0,
    translations: [],
    you: {
        __typename: "TagYou",
        isBookmarked: false,
    },
};

/**
 * Mock label data
 */
const todoLabel: Label = {
    __typename: "Label",
    id: "label_todo_123456789",
    label: "TODO",
    color: "#ff0000",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    you: {
        __typename: "LabelYou",
        canDelete: false,
        canUpdate: false,
    },
};

const inProgressLabel: Label = {
    __typename: "Label",
    id: "label_progress_123456",
    label: "In Progress",
    color: "#ffaa00",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    you: {
        __typename: "LabelYou",
        canDelete: false,
        canUpdate: false,
    },
};

/**
 * Minimal project version
 */
const minimalProjectVersion: ProjectVersion = {
    __typename: "ProjectVersion",
    id: "projver_123456789012345",
    versionLabel: "1.0.0",
    versionNotes: null,
    isLatest: true,
    isPrivate: false,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    completedAt: null,
    translations: [
        {
            __typename: "ProjectVersionTranslation",
            id: "projvertrans_123456789",
            language: "en",
            name: "Test Project",
            description: "A minimal test project",
        },
    ],
    resourceList: null,
    you: {
        __typename: "ProjectVersionYou",
        canComment: true,
        canDelete: false,
        canReport: true,
        canUpdate: false,
        canUse: true,
        canRead: true,
    },
};

/**
 * Complete project version
 */
const completeProjectVersion: ProjectVersion = {
    __typename: "ProjectVersion",
    id: "projver_987654321098765",
    versionLabel: "2.1.0",
    versionNotes: "Added new features and bug fixes",
    isLatest: true,
    isPrivate: false,
    createdAt: "2023-06-01T00:00:00Z",
    updatedAt: "2024-01-15T14:45:00Z",
    completedAt: "2024-01-10T00:00:00Z",
    translations: [
        {
            __typename: "ProjectVersionTranslation",
            id: "projvertrans_987654321",
            language: "en",
            name: "AI Assistant Platform",
            description: "A comprehensive platform for building AI-powered assistants with advanced capabilities",
        },
        {
            __typename: "ProjectVersionTranslation",
            id: "projvertrans_876543210",
            language: "es",
            name: "Plataforma de Asistente IA",
            description: "Una plataforma integral para construir asistentes impulsados por IA con capacidades avanzadas",
        },
    ],
    resourceList: {
        __typename: "ResourceList",
        id: "reslist_123456789",
        usedFor: "Display",
        resources: [],
    },
    you: {
        __typename: "ProjectVersionYou",
        canComment: true,
        canDelete: true,
        canReport: false,
        canUpdate: true,
        canUse: true,
        canRead: true,
    },
};

/**
 * Minimal project API response
 */
export const minimalProjectResponse: Project = {
    __typename: "Project",
    id: "project_123456789012345",
    handle: "test-project",
    isPrivate: false,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    score: 0,
    views: 0,
    owner: {
        __typename: "User",
        ...minimalUserResponse,
    },
    hasCompleteVersion: false,
    labels: [],
    tags: [],
    permissions: JSON.stringify(["Read"]),
    versions: [minimalProjectVersion],
    versionsCount: 1,
    you: {
        __typename: "ProjectYou",
        canComment: true,
        canDelete: false,
        canBookmark: true,
        canReact: true,
        canRead: true,
        canReport: true,
        canUpdate: false,
        canTransfer: false,
        isBookmarked: false,
        reaction: null,
        isReported: false,
        isViewed: false,
    },
};

/**
 * Complete project API response with all fields
 */
export const completeProjectResponse: Project = {
    __typename: "Project",
    id: "project_987654321098765",
    handle: "ai-assistant-platform",
    isPrivate: false,
    createdAt: "2023-06-01T00:00:00Z",
    updatedAt: "2024-01-15T14:45:00Z",
    score: 156,
    views: 2345,
    owner: {
        __typename: "Team",
        ...minimalTeamResponse,
    },
    hasCompleteVersion: true,
    labels: [todoLabel, inProgressLabel],
    tags: [aiTag, webDevTag],
    permissions: JSON.stringify(["Read", "Write", "Delete"]),
    versions: [completeProjectVersion],
    versionsCount: 5,
    transfersCount: 0,
    you: {
        __typename: "ProjectYou",
        canComment: true,
        canDelete: true,
        canBookmark: true,
        canReact: true,
        canRead: true,
        canReport: false, // Own project
        canUpdate: true,
        canTransfer: true,
        isBookmarked: true,
        reaction: "like",
        isReported: false,
        isViewed: true,
    },
};

/**
 * Private project response
 */
export const privateProjectResponse: Project = {
    __typename: "Project",
    id: "project_private_123456",
    handle: null, // Private projects might not have handles
    isPrivate: true,
    createdAt: "2023-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    score: 0,
    views: 0,
    owner: {
        __typename: "User",
        ...completeUserResponse,
    },
    hasCompleteVersion: false,
    labels: [],
    tags: [],
    permissions: JSON.stringify([]),
    versions: [{
        ...minimalProjectVersion,
        isPrivate: true,
    }],
    versionsCount: 1,
    you: {
        __typename: "ProjectYou",
        canComment: false, // No access
        canDelete: false,
        canBookmark: false,
        canReact: false,
        canRead: false,
        canReport: false,
        canUpdate: false,
        canTransfer: false,
        isBookmarked: false,
        reaction: null,
        isReported: false,
        isViewed: false,
    },
};

/**
 * Project variant states for testing
 */
export const projectResponseVariants = {
    minimal: minimalProjectResponse,
    complete: completeProjectResponse,
    private: privateProjectResponse,
    multiVersion: {
        ...completeProjectResponse,
        id: "project_multiver_123456",
        versions: [
            completeProjectVersion,
            {
                ...minimalProjectVersion,
                id: "projver_older_123456",
                versionLabel: "1.5.0",
                isLatest: false,
            },
        ],
        versionsCount: 10,
    },
    inDevelopment: {
        ...minimalProjectResponse,
        id: "project_dev_123456789",
        handle: "work-in-progress",
        hasCompleteVersion: false,
        labels: [todoLabel],
    },
    popular: {
        ...completeProjectResponse,
        id: "project_popular_123456",
        handle: "trending-project",
        score: 9999,
        views: 50000,
    },
} as const;

/**
 * Project search response
 */
export const projectSearchResponse = {
    __typename: "ProjectSearchResult",
    edges: [
        {
            __typename: "ProjectEdge",
            cursor: "cursor_1",
            node: projectResponseVariants.complete,
        },
        {
            __typename: "ProjectEdge",
            cursor: "cursor_2",
            node: projectResponseVariants.minimal,
        },
        {
            __typename: "ProjectEdge",
            cursor: "cursor_3",
            node: projectResponseVariants.popular,
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
export const projectUIStates = {
    loading: null,
    error: {
        code: "PROJECT_NOT_FOUND",
        message: "The requested project could not be found",
    },
    versionError: {
        code: "PROJECT_VERSION_NOT_FOUND",
        message: "The requested project version could not be found",
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