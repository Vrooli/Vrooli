import { type Resource, type ResourceVersion, type ResourceVersionTranslation, type ResourceVersionYou, type ResourceYou, ResourceType, type Tag, type Label } from "@vrooli/shared";
import { minimalUserResponse, completeUserResponse } from "./userResponses.js";
import { minimalTeamResponse } from "./teamResponses.js";

/**
 * API response fixtures for notes
 * These represent what components receive from API calls
 */

/**
 * Mock tag data for notes
 */
const documentationTag: Tag = {
    __typename: "Tag",
    id: "tag_docs_123456789",
    tag: "documentation",
    created_at: "2024-01-01T00:00:00Z",
    bookmarks: 0,
    translations: [
        {
            __typename: "TagTranslation",
            id: "tagtrans_docs_123456",
            language: "en",
            description: "Documentation and reference materials",
        },
    ],
    you: {
        __typename: "TagYou",
        isBookmarked: false,
    },
};

const personalTag: Tag = {
    __typename: "Tag",
    id: "tag_personal_123456",
    tag: "personal",
    created_at: "2024-01-01T00:00:00Z",
    bookmarks: 0,
    translations: [
        {
            __typename: "TagTranslation",
            id: "tagtrans_personal_123",
            language: "en",
            description: "Personal notes and thoughts",
        },
    ],
    you: {
        __typename: "TagYou",
        isBookmarked: false,
    },
};

/**
 * Mock label data for notes
 */
const draftLabel: Label = {
    __typename: "Label",
    id: "label_draft_123456789",
    label: "Draft",
    color: "#ffa500",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    you: {
        __typename: "LabelYou",
        canDelete: true,
        canUpdate: true,
    },
};

const importantLabel: Label = {
    __typename: "Label",
    id: "label_important_123456",
    label: "Important",
    color: "#ff0000",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    you: {
        __typename: "LabelYou",
        canDelete: true,
        canUpdate: true,
    },
};

/**
 * Minimal note version
 */
const minimalNoteVersion: ResourceVersion = {
    __typename: "ResourceVersion",
    id: "resver_note_123456789",
    publicId: "note_pub_123456",
    versionIndex: 0,
    versionLabel: "1.0.0",
    versionNotes: null,
    isLatest: true,
    isPrivate: false,
    isComplete: true,
    isDeleted: false,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    completedAt: "2024-01-01T00:00:00Z",
    complexity: 1,
    codeLanguage: null,
    config: null,
    isAutomatable: false,
    forks: [],
    forksCount: 0,
    pullRequest: null,
    relatedVersions: [],
    reports: [],
    reportsCount: 0,
    resourceSubType: null,
    timesCompleted: 0,
    timesStarted: 0,
    translations: [
        {
            __typename: "ResourceVersionTranslation",
            id: "resvertrans_note_123456",
            language: "en",
            name: "Quick Note",
            description: "A simple note for testing",
            details: "This is a minimal note with basic content.",
            instructions: null,
        },
    ],
    translationsCount: 1,
    comments: [],
    commentsCount: 0,
    root: {} as Resource, // Will be populated by parent
    you: {
        __typename: "ResourceVersionYou",
        canComment: true,
        canDelete: true,
        canReport: false,
        canUpdate: true,
        canRead: true,
        canBookmark: true,
        canCopy: true,
        canReact: true,
        canRun: false,
    },
};

/**
 * Complete note version with markdown content
 */
const completeNoteVersion: ResourceVersion = {
    __typename: "ResourceVersion",
    id: "resver_note_987654321",
    publicId: "note_pub_987654321",
    versionIndex: 2,
    versionLabel: "2.1.0",
    versionNotes: "Updated with additional examples and formatting improvements",
    isLatest: true,
    isPrivate: false,
    isComplete: true,
    isDeleted: false,
    createdAt: "2023-06-01T00:00:00Z",
    updatedAt: "2024-01-15T14:45:00Z",
    completedAt: "2024-01-10T00:00:00Z",
    complexity: 3,
    codeLanguage: null,
    config: {
        __version: "1.0",
        resources: [],
    },
    isAutomatable: false,
    forks: [],
    forksCount: 2,
    pullRequest: null,
    relatedVersions: [],
    reports: [],
    reportsCount: 0,
    resourceSubType: null,
    timesCompleted: 0,
    timesStarted: 0,
    translations: [
        {
            __typename: "ResourceVersionTranslation",
            id: "resvertrans_note_987654",
            language: "en",
            name: "Project Planning Guide",
            description: "Comprehensive guide for planning and managing projects",
            details: `# Project Planning Guide

## Overview
This guide provides a comprehensive approach to project planning and management.

## Key Phases

### 1. Planning Phase
- Define project scope and objectives
- Identify stakeholders and requirements
- Create project timeline and milestones

### 2. Execution Phase
- Implement project plan
- Monitor progress and track milestones
- Manage resources and team coordination

### 3. Review Phase
- Evaluate project outcomes
- Document lessons learned
- Plan for future improvements

## Best Practices
- Regular communication with stakeholders
- Use of project management tools
- Risk assessment and mitigation planning
- Continuous monitoring and adjustment

## Resources
- Project templates and checklists
- Communication protocols
- Risk management frameworks`,
            instructions: "Use this guide as a reference for your project planning activities. Adapt the content to your specific project needs.",
        },
        {
            __typename: "ResourceVersionTranslation",
            id: "resvertrans_note_876543",
            language: "es",
            name: "Guía de Planificación de Proyectos",
            description: "Guía completa para la planificación y gestión de proyectos",
            details: `# Guía de Planificación de Proyectos

## Descripción General
Esta guía proporciona un enfoque integral para la planificación y gestión de proyectos.

## Fases Clave

### 1. Fase de Planificación
- Definir el alcance y objetivos del proyecto
- Identificar partes interesadas y requisitos
- Crear cronograma y hitos del proyecto

### 2. Fase de Ejecución
- Implementar el plan del proyecto
- Monitorear el progreso y seguir los hitos
- Gestionar recursos y coordinación del equipo

### 3. Fase de Revisión
- Evaluar los resultados del proyecto
- Documentar lecciones aprendidas
- Planificar mejoras futuras`,
            instructions: "Utilice esta guía como referencia para sus actividades de planificación de proyectos.",
        },
    ],
    translationsCount: 2,
    comments: [],
    commentsCount: 5,
    root: {} as Resource, // Will be populated by parent
    you: {
        __typename: "ResourceVersionYou",
        canComment: true,
        canDelete: true,
        canReport: false,
        canUpdate: true,
        canRead: true,
        canBookmark: true,
        canCopy: true,
        canReact: true,
        canRun: false,
    },
};

/**
 * Minimal note API response
 */
export const minimalNoteResponse: Resource = {
    __typename: "Resource",
    id: "resource_note_123456789",
    publicId: "note_pub_123456",
    resourceType: ResourceType.Note,
    isPrivate: false,
    isDeleted: false,
    isInternal: false,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    completedAt: "2024-01-01T00:00:00Z",
    hasCompleteVersion: true,
    score: 0,
    bookmarks: 0,
    owner: {
        __typename: "User",
        ...minimalUserResponse,
    },
    createdBy: {
        __typename: "User",
        ...minimalUserResponse,
    },
    parent: null,
    bookmarkedBy: [],
    issues: [],
    issuesCount: 0,
    pullRequests: [],
    pullRequestsCount: 0,
    stats: [],
    tags: [],
    labels: [],
    versions: [{ ...minimalNoteVersion, root: {} as Resource }],
    versionsCount: 1,
    permissions: JSON.stringify(["Read", "Write", "Delete"]),
    you: {
        __typename: "ResourceYou",
        canComment: true,
        canDelete: true,
        canBookmark: true,
        canReact: true,
        canRead: true,
        canTransfer: true,
        canUpdate: true,
        isBookmarked: false,
        isViewed: false,
        reaction: null,
    },
};

// Set up circular reference for minimal note
minimalNoteResponse.versions[0].root = minimalNoteResponse;

/**
 * Complete note API response with all fields
 */
export const completeNoteResponse: Resource = {
    __typename: "Resource",
    id: "resource_note_987654321",
    publicId: "note_pub_987654321",
    resourceType: ResourceType.Note,
    isPrivate: false,
    isDeleted: false,
    isInternal: false,
    createdAt: "2023-06-01T00:00:00Z",
    updatedAt: "2024-01-15T14:45:00Z",
    completedAt: "2024-01-10T00:00:00Z",
    hasCompleteVersion: true,
    score: 45,
    bookmarks: 12,
    owner: {
        __typename: "Team",
        ...minimalTeamResponse,
    },
    createdBy: {
        __typename: "User",
        ...completeUserResponse,
    },
    parent: null,
    bookmarkedBy: [],
    issues: [],
    issuesCount: 0,
    pullRequests: [],
    pullRequestsCount: 0,
    stats: [],
    tags: [documentationTag, personalTag],
    labels: [draftLabel, importantLabel],
    versions: [{ ...completeNoteVersion, root: {} as Resource }],
    versionsCount: 3,
    permissions: JSON.stringify(["Read", "Write", "Delete"]),
    you: {
        __typename: "ResourceYou",
        canComment: true,
        canDelete: true,
        canBookmark: true,
        canReact: true,
        canRead: true,
        canTransfer: true,
        canUpdate: true,
        isBookmarked: true,
        isViewed: true,
        reaction: "like",
    },
};

// Set up circular reference for complete note
completeNoteResponse.versions[0].root = completeNoteResponse;

/**
 * Private note response
 */
export const privateNoteResponse: Resource = {
    __typename: "Resource",
    id: "resource_note_private_123",
    publicId: "note_priv_123456",
    resourceType: ResourceType.Note,
    isPrivate: true,
    isDeleted: false,
    isInternal: false,
    createdAt: "2023-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    completedAt: "2024-01-01T00:00:00Z",
    hasCompleteVersion: true,
    score: 0,
    bookmarks: 0,
    owner: {
        __typename: "User",
        ...completeUserResponse,
    },
    createdBy: {
        __typename: "User",
        ...completeUserResponse,
    },
    parent: null,
    bookmarkedBy: [],
    issues: [],
    issuesCount: 0,
    pullRequests: [],
    pullRequestsCount: 0,
    stats: [],
    tags: [personalTag],
    labels: [],
    versions: [{
        ...minimalNoteVersion,
        id: "resver_note_private_123",
        publicId: "note_priv_ver_123",
        isPrivate: true,
        translations: [
            {
                __typename: "ResourceVersionTranslation",
                id: "resvertrans_private_123",
                language: "en",
                name: "Private Thoughts",
                description: "Personal reflections and private notes",
                details: "This is a private note containing personal thoughts and reflections.",
                instructions: null,
            },
        ],
        root: {} as Resource,
    }],
    versionsCount: 1,
    permissions: JSON.stringify(["Read", "Write", "Delete"]),
    you: {
        __typename: "ResourceYou",
        canComment: false,
        canDelete: true,
        canBookmark: false,
        canReact: false,
        canRead: true,
        canTransfer: true,
        canUpdate: true,
        isBookmarked: false,
        isViewed: false,
        reaction: null,
    },
};

// Set up circular reference for private note
privateNoteResponse.versions[0].root = privateNoteResponse;

/**
 * Shared note response (team collaboration)
 */
export const sharedNoteResponse: Resource = {
    __typename: "Resource",
    id: "resource_note_shared_456",
    publicId: "note_shared_456789",
    resourceType: ResourceType.Note,
    isPrivate: false,
    isDeleted: false,
    isInternal: false,
    createdAt: "2023-12-01T00:00:00Z",
    updatedAt: "2024-01-20T15:30:00Z",
    completedAt: "2024-01-15T00:00:00Z",
    hasCompleteVersion: true,
    score: 23,
    bookmarks: 8,
    owner: {
        __typename: "Team",
        ...minimalTeamResponse,
    },
    createdBy: {
        __typename: "User",
        ...minimalUserResponse,
    },
    parent: null,
    bookmarkedBy: [],
    issues: [],
    issuesCount: 0,
    pullRequests: [],
    pullRequestsCount: 0,
    stats: [],
    tags: [documentationTag],
    labels: [importantLabel],
    versions: [{
        ...completeNoteVersion,
        id: "resver_note_shared_456",
        publicId: "note_shared_ver_456",
        translations: [
            {
                __typename: "ResourceVersionTranslation",
                id: "resvertrans_shared_456",
                language: "en",
                name: "Team Meeting Notes",
                description: "Notes from weekly team meetings and decisions",
                details: `# Team Meeting Notes - Week of Jan 15, 2024

## Attendees
- Alice Johnson (Team Lead)
- Bob Smith (Developer)
- Carol Davis (Designer)
- Dave Wilson (QA)

## Agenda Items

### 1. Sprint Review
- Completed 8 out of 10 planned items
- Two items moved to next sprint due to dependencies
- Overall sprint velocity improved by 15%

### 2. Upcoming Releases
- Version 2.1 scheduled for February 1st
- Testing phase begins next week
- Documentation updates needed

### 3. Team Updates
- New team member starting Monday
- Training schedule to be finalized
- Mentorship assignments

## Action Items
- [ ] Update project timeline (Alice)
- [ ] Prepare onboarding materials (Bob)
- [ ] Review UI mockups (Carol)
- [ ] Set up test environments (Dave)

## Next Meeting
Tuesday, January 23rd at 10:00 AM`,
                instructions: "Keep meeting notes updated and share with team members who couldn't attend.",
            },
        ],
        root: {} as Resource,
        commentsCount: 3,
        you: {
            __typename: "ResourceVersionYou",
            canComment: true,
            canDelete: false, // Team member but not owner
            canReport: false,
            canUpdate: true,
            canRead: true,
            canBookmark: true,
            canCopy: true,
            canReact: true,
            canRun: false,
        },
    }],
    versionsCount: 1,
    permissions: JSON.stringify(["Read", "Write"]),
    you: {
        __typename: "ResourceYou",
        canComment: true,
        canDelete: false,
        canBookmark: true,
        canReact: true,
        canRead: true,
        canTransfer: false,
        canUpdate: true,
        isBookmarked: true,
        isViewed: true,
        reaction: null,
    },
};

// Set up circular reference for shared note
sharedNoteResponse.versions[0].root = sharedNoteResponse;

/**
 * Note variant states for testing
 */
export const noteResponseVariants = {
    minimal: minimalNoteResponse,
    complete: completeNoteResponse,
    private: privateNoteResponse,
    shared: sharedNoteResponse,
    documentation: {
        ...completeNoteResponse,
        id: "resource_note_docs_789",
        publicId: "note_docs_789012",
        tags: [documentationTag],
        labels: [],
        versions: [{
            ...completeNoteVersion,
            id: "resver_note_docs_789",
            publicId: "note_docs_ver_789",
            translations: [
                {
                    __typename: "ResourceVersionTranslation",
                    id: "resvertrans_docs_789",
                    language: "en",
                    name: "API Documentation",
                    description: "Complete API reference and usage examples",
                    details: `# API Documentation

## Authentication
All API requests require authentication using API keys.

## Endpoints

### GET /api/users
Returns a list of users.

### POST /api/users
Creates a new user.

### GET /api/users/:id
Returns a specific user by ID.`,
                    instructions: "Keep this documentation updated with any API changes.",
                },
            ],
            root: {} as Resource,
        }],
    },
    versionHistory: {
        ...minimalNoteResponse,
        id: "resource_note_versions_999",
        publicId: "note_versions_999",
        versions: [
            {
                ...minimalNoteVersion,
                id: "resver_note_v3_999",
                publicId: "note_v3_999",
                versionIndex: 2,
                versionLabel: "3.0.0",
                isLatest: true,
                translations: [
                    {
                        __typename: "ResourceVersionTranslation",
                        id: "resvertrans_v3_999",
                        language: "en",
                        name: "Evolving Note v3.0",
                        description: "Latest version with major updates",
                        details: "This is the latest version with significant improvements.",
                        instructions: null,
                    },
                ],
                root: {} as Resource,
            },
            {
                ...minimalNoteVersion,
                id: "resver_note_v2_998",
                publicId: "note_v2_998",
                versionIndex: 1,
                versionLabel: "2.0.0",
                isLatest: false,
                translations: [
                    {
                        __typename: "ResourceVersionTranslation",
                        id: "resvertrans_v2_998",
                        language: "en",
                        name: "Evolving Note v2.0",
                        description: "Previous version with some features",
                        details: "This is an older version with basic features.",
                        instructions: null,
                    },
                ],
                root: {} as Resource,
            },
        ],
        versionsCount: 3,
    },
} as const;

// Set up circular references for variant notes
noteResponseVariants.documentation.versions[0].root = noteResponseVariants.documentation;
noteResponseVariants.versionHistory.versions[0].root = noteResponseVariants.versionHistory;
noteResponseVariants.versionHistory.versions[1].root = noteResponseVariants.versionHistory;

/**
 * Note search response
 */
export const noteSearchResponse = {
    __typename: "ResourceSearchResult",
    edges: [
        {
            __typename: "ResourceEdge",
            cursor: "cursor_1",
            node: noteResponseVariants.complete,
        },
        {
            __typename: "ResourceEdge",
            cursor: "cursor_2",
            node: noteResponseVariants.shared,
        },
        {
            __typename: "ResourceEdge",
            cursor: "cursor_3",
            node: noteResponseVariants.documentation,
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
export const noteUIStates = {
    loading: null,
    error: {
        code: "NOTE_NOT_FOUND",
        message: "The requested note could not be found",
    },
    versionError: {
        code: "NOTE_VERSION_NOT_FOUND",
        message: "The requested note version could not be found",
    },
    accessDenied: {
        code: "NOTE_ACCESS_DENIED",
        message: "You don't have permission to access this note",
    },
    saveError: {
        code: "NOTE_SAVE_FAILED",
        message: "Failed to save note. Please try again.",
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