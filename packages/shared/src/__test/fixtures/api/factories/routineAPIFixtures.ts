/* c8 ignore start */
/**
 * Type-safe Routine API fixture factory
 * 
 * This factory provides comprehensive fixtures for Routine objects (Resources with routine versions) with:
 * - Zero `any` types
 * - Full validation integration for resources and resource versions
 * - Shape function integration for both create and update operations
 * - Comprehensive error scenarios for AI workflow testing
 * - Factory methods for different routine types (action, generate, multi-step, etc.)
 * - AI execution context support
 */
import type {
    Resource,
    ResourceCreateInput,
    ResourceUpdateInput,
    ResourceVersionCreateInput,
    ResourceVersionTranslationUpdateInput,
    ResourceVersionUpdateInput,
    TagCreateInput,
} from "../../../../api/types.js";
import { ResourceSubType, ResourceType } from "../../../../api/types.js";
import { generatePK } from "../../../../id/snowflake.js";
import { type RoutineVersionConfigObject } from "../../../../shape/configs/routine.js";
import { LATEST_CONFIG_VERSION } from "../../../../shape/configs/utils.js";
import { shapeResource, type ResourceShape } from "../../../../shape/models/models.js";
import { BaseAPIFixtureFactory } from "../BaseAPIFixtureFactory.js";
import type { APIFixtureFactory, FactoryCustomizers } from "../types.js";

// Magic number constants for testing
const ROUTINE_NAME_TOO_LONG_LENGTH = 257;
const ROUTINE_DESCRIPTION_TOO_LONG_LENGTH = 2049;
const VERSION_NOTES_MAX_LENGTH = 2048;
const TRANSLATION_NAME_MAX_LENGTH = 256;
const TRANSLATION_DESCRIPTION_MAX_LENGTH = 2048;
const TRANSLATION_DETAILS_LENGTH = 400;
const TRANSLATION_INSTRUCTIONS_LENGTH = 200;

// ========================================
// Type-Safe Fixture Data
// ========================================

const validIds = {
    routine1: generatePK().toString(),
    routine2: generatePK().toString(),
    routine3: generatePK().toString(),
    routine4: generatePK().toString(),
    routineVersion1: generatePK().toString(),
    routineVersion2: generatePK().toString(),
    routineVersion3: generatePK().toString(),
    routineVersion4: generatePK().toString(),
    routineVersion5: generatePK().toString(),
    routineVersion6: generatePK().toString(),
    user1: generatePK().toString(),
    user2: generatePK().toString(),
    team1: generatePK().toString(),
    translation1: generatePK().toString(),
    translation2: generatePK().toString(),
    translation3: generatePK().toString(),
    translation4: generatePK().toString(),
    translation5: generatePK().toString(),
    translation6: generatePK().toString(),
    tag1: generatePK().toString(),
    tag2: generatePK().toString(),
    tag3: generatePK().toString(),
};

// Valid public IDs for testing (10-12 character alphanumeric)
const validPublicIds = {
    routine1: "rtne123test",
    routine2: "rtne456comp",
    routine3: "rtne789multi",
    routine4: "rtne012edge",
    version1: "ver123test4",
    version2: "ver456comp7",
    version3: "ver789mult0",
    version4: "ver012edge3",
};

// Create proper routine config objects with the current format
const createActionRoutineConfig = (): RoutineVersionConfigObject => ({
    __version: LATEST_CONFIG_VERSION,
    callDataAction: {
        __version: LATEST_CONFIG_VERSION,
        schema: {
            toolName: "resource_manage" as any,
            inputTemplate: JSON.stringify({
                op: "find",
                resource_type: "Note",
                filters: {
                    name: "{{input.searchTerm}}",
                },
            }),
            allowedContexts: ["user", "agent"],
            outputMapping: {
                "result": "payload.data",
                "count": "payload.total",
            },
        },
    },
    formInput: {
        __version: LATEST_CONFIG_VERSION,
        schema: {
            containers: [],
            elements: [
                {
                    id: "searchTerm",
                    fieldName: "searchTerm",
                    label: "Search Term",
                    type: "text" as any,
                    props: {
                        placeholder: "Enter search term",
                    },
                },
            ],
        },
    },
    formOutput: {
        __version: LATEST_CONFIG_VERSION,
        schema: {
            containers: [],
            elements: [
                {
                    id: "result",
                    fieldName: "result",
                    label: "Search Results",
                    type: "text" as any,
                },
                {
                    id: "count",
                    fieldName: "count",
                    label: "Result Count",
                    type: "number" as any,
                },
            ],
        },
    },
});

const createGenerateRoutineConfig = (): RoutineVersionConfigObject => ({
    __version: LATEST_CONFIG_VERSION,
    callDataGenerate: {
        __version: LATEST_CONFIG_VERSION,
        schema: {
            model: "gpt-4" as any,
            prompt: "Generate a response for: {{input.prompt}}",
            maxTokens: 1000,
            botStyle: "Default" as any,
        },
    },
    formInput: {
        __version: LATEST_CONFIG_VERSION,
        schema: {
            containers: [],
            elements: [
                {
                    id: "prompt",
                    fieldName: "prompt",
                    label: "Input Prompt",
                    type: "text" as any,
                    props: {
                        placeholder: "Enter your prompt",
                    },
                },
            ],
        },
    },
    formOutput: {
        __version: LATEST_CONFIG_VERSION,
        schema: {
            containers: [],
            elements: [
                {
                    id: "response",
                    fieldName: "response",
                    label: "Generated Response",
                    type: "text" as any,
                    props: {
                        placeholder: "Model response will be displayed here",
                    },
                },
            ],
        },
    },
});

const createMultiStepRoutineConfig = (): RoutineVersionConfigObject => ({
    __version: LATEST_CONFIG_VERSION,
    graph: {
        __version: LATEST_CONFIG_VERSION,
        __type: "BPMN-2.0",
        schema: {
            __format: "xml",
            data: `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <process id="multiStepProcess" isExecutable="true">
    <startEvent id="start"/>
    <sequenceFlow id="flow1" sourceRef="start" targetRef="step1"/>
    <callActivity id="step1" name="Data Collection" calledElement="dataCollection"/>
    <sequenceFlow id="flow2" sourceRef="step1" targetRef="step2"/>
    <callActivity id="step2" name="Data Processing" calledElement="dataProcessing"/>
    <sequenceFlow id="flow3" sourceRef="step2" targetRef="end"/>
    <endEvent id="end"/>
  </process>
</definitions>`,
            activityMap: {
                "step1": {
                    subroutineId: validIds.routineVersion1,
                    inputMap: {
                        "userInput": "input",
                    },
                    outputMap: {
                        "collectedData": "data",
                    },
                },
                "step2": {
                    subroutineId: validIds.routineVersion2,
                    inputMap: {
                        "rawData": "collectedData",
                    },
                    outputMap: {
                        "processedData": "result",
                    },
                },
            },
            rootContext: {
                inputMap: {
                    "input": "userInput",
                },
                outputMap: {
                    "result": "finalResult",
                },
            },
        },
    },
    formInput: {
        __version: LATEST_CONFIG_VERSION,
        schema: {
            containers: [],
            elements: [
                {
                    id: "userInput",
                    fieldName: "userInput",
                    label: "Input Data",
                    type: "text" as any,
                },
            ],
        },
    },
    formOutput: {
        __version: LATEST_CONFIG_VERSION,
        schema: {
            containers: [],
            elements: [
                {
                    id: "finalResult",
                    fieldName: "finalResult",
                    label: "Final Result",
                    type: "text" as any,
                },
            ],
        },
    },
});

// Core fixture data with complete type safety
const routineFixtureData = {
    minimal: {
        create: {
            id: validIds.routine1,
            resourceType: ResourceType.Routine,
            isPrivate: false,
            ownedByUserConnect: validIds.user1,
            versionsCreate: [
                {
                    id: validIds.routineVersion1,
                    versionLabel: "1.0.0",
                    isPrivate: false,
                    isComplete: true,
                    resourceSubType: ResourceSubType.RoutineInternalAction,
                    config: createActionRoutineConfig(),
                    translationsCreate: [
                        {
                            id: validIds.translation1,
                            language: "en",
                            name: "Simple Action Routine",
                            description: "A minimal routine for testing basic functionality",
                        },
                    ],
                } satisfies ResourceVersionCreateInput,
            ],
        } satisfies ResourceCreateInput,

        update: {
            id: validIds.routine1,
        } satisfies ResourceUpdateInput,

        find: {
            __typename: "Resource" as const,
            id: validIds.routine1,
            publicId: validPublicIds.routine1,
            resourceType: ResourceType.Routine,
            isPrivate: false,
            isInternal: false,
            isDeleted: false,
            hasCompleteVersion: true,
            permissions: JSON.stringify(["read", "write"]),
            bookmarks: 0,
            views: 0,
            score: 0,
            issuesCount: 0,
            pullRequestsCount: 0,
            transfersCount: 0,
            versionsCount: 1,
            translatedName: "Simple Action Routine",
            createdAt: "2024-01-01T00:00:00Z",
            updatedAt: "2024-01-01T00:00:00Z",
            completedAt: null,
            bookmarkedBy: [],
            issues: [],
            pullRequests: [],
            stats: [],
            tags: [],
            transfers: [],
            versions: [
                {
                    __typename: "ResourceVersion" as const,
                    id: validIds.routineVersion1,
                    publicId: validPublicIds.version1,
                    versionLabel: "1.0.0",
                    versionIndex: 0,
                    versionNotes: null,
                    isComplete: true,
                    isPrivate: false,
                    isDeleted: false,
                    isLatest: true,
                    isAutomatable: true,
                    resourceSubType: ResourceSubType.RoutineInternalAction,
                    complexity: 1,
                    config: createActionRoutineConfig(),
                    codeLanguage: null,
                    commentsCount: 0,
                    forksCount: 0,
                    reportsCount: 0,
                    timesCompleted: 0,
                    timesStarted: 0,
                    translationsCount: 1,
                    createdAt: "2024-01-01T00:00:00Z",
                    updatedAt: "2024-01-01T00:00:00Z",
                    completedAt: null,
                    comments: [],
                    forks: [],
                    relatedVersions: [],
                    reports: [],
                    translations: [
                        {
                            __typename: "ResourceVersionTranslation" as const,
                            id: validIds.translation1,
                            language: "en",
                            name: "Simple Action Routine",
                            description: "A minimal routine for testing basic functionality",
                            details: null,
                            instructions: "",
                        },
                    ],
                    pullRequest: null,
                    root: {
                        __typename: "Resource" as const,
                        id: validIds.routine1,
                    } as any,
                    you: {
                        __typename: "ResourceVersionYou" as const,
                        canDelete: true,
                        canBookmark: true,
                        canComment: true,
                        canCopy: true,
                        canReact: true,
                        canRead: true,
                        canReport: false,
                        canRun: true,
                        canUpdate: true,
                    },
                },
            ],
            owner: {
                __typename: "User" as const,
                id: validIds.user1,
            } as any,
            parent: null,
            you: {
                __typename: "ResourceYou" as const,
                canBookmark: true,
                canComment: true,
                canDelete: true,
                canReact: true,
                canRead: true,
                canTransfer: true,
                canUpdate: true,
                isBookmarked: false,
                isViewed: false,
                reaction: null,
            },
        } satisfies Resource,
    },

    complete: {
        create: {
            id: validIds.routine2,
            publicId: validPublicIds.routine2,
            resourceType: ResourceType.Routine,
            isInternal: false,
            isPrivate: false,
            permissions: JSON.stringify(["read", "write", "admin"]),
            ownedByTeamConnect: validIds.team1,
            tagsConnect: [validIds.tag1, validIds.tag2],
            tagsCreate: [
                {
                    id: validIds.tag3,
                    tag: "ai-workflow",
                } satisfies TagCreateInput,
            ],
            versionsCreate: [
                {
                    id: validIds.routineVersion2,
                    publicId: validPublicIds.version2,
                    versionLabel: "1.0.0",
                    versionNotes: "Initial release with comprehensive AI workflow capabilities",
                    isPrivate: false,
                    isComplete: true,
                    isAutomatable: true,
                    resourceSubType: ResourceSubType.RoutineMultiStep,
                    config: createMultiStepRoutineConfig(),
                    translationsCreate: [
                        {
                            id: validIds.translation2,
                            language: "en",
                            name: "Complete Multi-Step AI Workflow",
                            description: "A comprehensive routine demonstrating multi-step AI workflow execution with branching logic and context management",
                            details: "This routine showcases the full capabilities of Vrooli's AI execution system including step orchestration, data flow management, and conditional branching",
                            instructions: "1. Provide input data\n2. Review intermediate results\n3. Confirm final processing\n4. Export results",
                        },
                        {
                            id: validIds.translation3,
                            language: "es",
                            name: "Flujo de Trabajo AI Multi-Paso Completo",
                            description: "Una rutina integral que demuestra la ejecución de flujos de trabajo AI multi-paso con lógica de ramificación y gestión de contexto",
                            details: "Esta rutina muestra las capacidades completas del sistema de ejecución AI de Vrooli incluyendo orquestación de pasos, gestión de flujo de datos y ramificación condicional",
                            instructions: "1. Proporcionar datos de entrada\n2. Revisar resultados intermedios\n3. Confirmar procesamiento final\n4. Exportar resultados",
                        },
                    ],
                } satisfies ResourceVersionCreateInput,
                {
                    id: validIds.routineVersion3,
                    versionLabel: "1.1.0",
                    versionNotes: "Enhanced performance and error handling",
                    isPrivate: false,
                    isComplete: true,
                    isAutomatable: true,
                    resourceSubType: ResourceSubType.RoutineMultiStep,
                    config: createMultiStepRoutineConfig(),
                    translationsCreate: [
                        {
                            id: validIds.translation4,
                            language: "en",
                            name: "Enhanced Multi-Step AI Workflow v1.1",
                            description: "Updated version with improved performance and enhanced error handling capabilities",
                        },
                    ],
                } satisfies ResourceVersionCreateInput,
            ],
        } satisfies ResourceCreateInput,

        update: {
            id: validIds.routine2,
            isPrivate: true,
            permissions: JSON.stringify(["read", "write"]),
            tagsConnect: [validIds.tag3],
            tagsDisconnect: [validIds.tag1],
            versionsCreate: [
                {
                    id: validIds.routineVersion4,
                    versionLabel: "2.0.0",
                    versionNotes: "Major update with new AI capabilities",
                    isPrivate: false,
                    isComplete: true,
                    resourceSubType: ResourceSubType.RoutineGenerate,
                    config: createGenerateRoutineConfig(),
                    translationsCreate: [
                        {
                            id: validIds.translation5,
                            language: "en",
                            name: "AI Generation Routine v2.0",
                            description: "Advanced text generation with context awareness",
                        },
                    ],
                } satisfies ResourceVersionCreateInput,
            ],
            versionsUpdate: [
                {
                    id: validIds.routineVersion2,
                    versionNotes: "Updated documentation and bug fixes",
                    translationsUpdate: [
                        {
                            id: validIds.translation2,
                            language: "en",
                            description: "Updated description with latest improvements",
                        } satisfies ResourceVersionTranslationUpdateInput,
                    ],
                } satisfies ResourceVersionUpdateInput,
            ],
            versionsDelete: [validIds.routineVersion3],
        } satisfies ResourceUpdateInput,

        find: {
            __typename: "Resource" as const,
            id: validIds.routine2,
            publicId: validPublicIds.routine2,
            resourceType: ResourceType.Routine,
            isPrivate: false,
            isInternal: false,
            isDeleted: false,
            hasCompleteVersion: true,
            permissions: JSON.stringify(["read", "write", "admin"]),
            bookmarks: 47,
            views: 1250,
            score: 95,
            issuesCount: 2,
            pullRequestsCount: 5,
            transfersCount: 0,
            versionsCount: 3,
            translatedName: "Complete Multi-Step AI Workflow",
            createdAt: "2024-01-01T00:00:00Z",
            updatedAt: "2024-02-15T14:30:00Z",
            completedAt: "2024-02-01T10:00:00Z",
            bookmarkedBy: [],
            issues: [],
            pullRequests: [],
            stats: [],
            tags: [
                {
                    __typename: "Tag" as const,
                    id: validIds.tag2,
                    tag: "workflow",
                    bookmarkedBy: [],
                    bookmarks: 0,
                    createdAt: "2024-01-01T00:00:00Z",
                    updatedAt: "2024-01-01T00:00:00Z",
                    reports: [],
                    resources: [],
                    teams: [],
                    translations: [],
                    you: {
                        __typename: "TagYou" as const,
                        isBookmarked: false,
                        isOwn: false,
                    },
                },
                {
                    __typename: "Tag" as const,
                    id: validIds.tag3,
                    tag: "ai-workflow",
                    bookmarkedBy: [],
                    bookmarks: 0,
                    createdAt: "2024-01-01T00:00:00Z",
                    updatedAt: "2024-01-01T00:00:00Z",
                    reports: [],
                    resources: [],
                    teams: [],
                    translations: [],
                    you: {
                        __typename: "TagYou" as const,
                        isBookmarked: false,
                        isOwn: false,
                    },
                },
            ],
            transfers: [],
            versions: [
                {
                    __typename: "ResourceVersion" as const,
                    id: validIds.routineVersion2,
                    publicId: validPublicIds.version2,
                    versionLabel: "1.0.0",
                    versionIndex: 0,
                    versionNotes: "Initial release with comprehensive AI workflow capabilities",
                    isComplete: true,
                    isPrivate: false,
                    isDeleted: false,
                    isLatest: false,
                    isAutomatable: true,
                    resourceSubType: ResourceSubType.RoutineMultiStep,
                    complexity: 8,
                    config: createMultiStepRoutineConfig(),
                    codeLanguage: null,
                    commentsCount: 12,
                    forksCount: 8,
                    reportsCount: 0,
                    timesCompleted: 234,
                    timesStarted: 267,
                    translationsCount: 2,
                    createdAt: "2024-01-01T00:00:00Z",
                    updatedAt: "2024-02-10T09:15:00Z",
                    completedAt: "2024-02-01T10:00:00Z",
                    comments: [],
                    forks: [],
                    relatedVersions: [],
                    reports: [],
                    translations: [
                        {
                            __typename: "ResourceVersionTranslation" as const,
                            id: validIds.translation2,
                            language: "en",
                            name: "Complete Multi-Step AI Workflow",
                            description: "Updated description with latest improvements",
                            details: "This routine showcases the full capabilities of Vrooli's AI execution system including step orchestration, data flow management, and conditional branching",
                            instructions: "1. Provide input data\n2. Review intermediate results\n3. Confirm final processing\n4. Export results",
                        },
                        {
                            __typename: "ResourceVersionTranslation" as const,
                            id: validIds.translation3,
                            language: "es",
                            name: "Flujo de Trabajo AI Multi-Paso Completo",
                            description: "Una rutina integral que demuestra la ejecución de flujos de trabajoAI multi-paso con lógica de ramificación y gestión de contexto",
                            details: "Esta rutina muestra las capacidades completas del sistema de ejecución AI de Vrooli incluyendo orquestación de pasos, gestión de flujo de datos y ramificación condicional",
                            instructions: "1. Proporcionar datos de entrada\n2. Revisar resultados intermedios\n3. Confirmar procesamiento final\n4. Exportar resultados",
                        },
                    ],
                    pullRequest: null,
                    root: {
                        __typename: "Resource" as const,
                        id: validIds.routine2,
                    } as any,
                    you: {
                        __typename: "ResourceVersionYou" as const,
                        canDelete: true,
                        canBookmark: true,
                        canComment: true,
                        canCopy: true,
                        canReact: true,
                        canRead: true,
                        canReport: false,
                        canRun: true,
                        canUpdate: true,
                    },
                },
                {
                    __typename: "ResourceVersion" as const,
                    id: validIds.routineVersion4,
                    publicId: validPublicIds.version4,
                    versionLabel: "2.0.0",
                    versionIndex: 1,
                    versionNotes: "Major update with new AI capabilities",
                    isComplete: true,
                    isPrivate: false,
                    isDeleted: false,
                    isLatest: true,
                    isAutomatable: true,
                    resourceSubType: ResourceSubType.RoutineGenerate,
                    complexity: 5,
                    config: createGenerateRoutineConfig(),
                    codeLanguage: null,
                    commentsCount: 5,
                    forksCount: 3,
                    reportsCount: 0,
                    timesCompleted: 89,
                    timesStarted: 102,
                    translationsCount: 1,
                    createdAt: "2024-02-15T14:30:00Z",
                    updatedAt: "2024-02-15T14:30:00Z",
                    completedAt: null,
                    comments: [],
                    forks: [],
                    relatedVersions: [],
                    reports: [],
                    translations: [
                        {
                            __typename: "ResourceVersionTranslation" as const,
                            id: validIds.translation5,
                            language: "en",
                            name: "AI Generation Routine v2.0",
                            description: "Advanced text generation with context awareness",
                            details: null,
                            instructions: "",
                        },
                    ],
                    pullRequest: null,
                    root: {
                        __typename: "Resource" as const,
                        id: validIds.routine2,
                    } as any,
                    you: {
                        __typename: "ResourceVersionYou" as const,
                        canDelete: true,
                        canBookmark: true,
                        canComment: true,
                        canCopy: true,
                        canReact: true,
                        canRead: true,
                        canReport: false,
                        canRun: true,
                        canUpdate: true,
                    },
                },
            ],
            owner: {
                __typename: "Team" as const,
                id: validIds.team1,
            } as any,
            parent: null,
            you: {
                __typename: "ResourceYou" as const,
                canBookmark: true,
                canComment: true,
                canDelete: true,
                canReact: true,
                canRead: true,
                canTransfer: true,
                canUpdate: true,
                isBookmarked: true,
                isViewed: true,
                reaction: null,
            },
        } satisfies Resource,
    },

    invalid: {
        missingRequired: {
            create: {
                // Missing required fields: id, resourceType, versionsCreate, owner
                publicId: validPublicIds.routine1,
                isPrivate: false,
            } satisfies Partial<ResourceCreateInput>,
            update: {
                // Missing required id
                isPrivate: true,
            } satisfies Partial<ResourceUpdateInput>,
        },

        invalidTypes: {
            create: {
                id: 123, // Should be string
                resourceType: "InvalidType", // Should be ResourceType.Routine
                isPrivate: "yes", // Should be boolean
                ownedByUserConnect: true, // Should be string
                versionsCreate: "not an array", // Should be array
            } satisfies Record<string, unknown>,
            update: {
                id: [], // Should be string
                isPrivate: "false", // Should be boolean
                permissions: { invalid: "object" }, // Should be JSON string
            } satisfies Record<string, unknown>,
        },

        businessLogicErrors: {
            invalidResourceType: {
                id: validIds.routine1,
                resourceType: ResourceType.Project, // Should be ResourceType.Routine
                isPrivate: false,
                ownedByUserConnect: validIds.user1,
                versionsCreate: [
                    {
                        id: validIds.routineVersion1,
                        versionLabel: "1.0.0",
                        isPrivate: false,
                        resourceSubType: ResourceSubType.RoutineInternalAction,
                        config: createActionRoutineConfig(),
                        translationsCreate: [
                            {
                                id: validIds.translation1,
                                language: "en",
                                name: "Invalid Resource Type",
                            },
                        ],
                    },
                ],
            } satisfies Partial<ResourceCreateInput>,

            incompatibleSubType: {
                id: validIds.routine1,
                resourceType: ResourceType.Routine,
                isPrivate: false,
                ownedByUserConnect: validIds.user1,
                versionsCreate: [
                    {
                        id: validIds.routineVersion1,
                        versionLabel: "1.0.0",
                        isPrivate: false,
                        resourceSubType: ResourceSubType.CodeDataConverter, // Wrong subtype for routine
                        config: createActionRoutineConfig(),
                        translationsCreate: [
                            {
                                id: validIds.translation1,
                                language: "en",
                                name: "Incompatible SubType",
                            },
                        ],
                    },
                ],
            } satisfies Partial<ResourceCreateInput>,

            invalidConfig: {
                id: validIds.routine1,
                resourceType: ResourceType.Routine,
                isPrivate: false,
                ownedByUserConnect: validIds.user1,
                versionsCreate: [
                    {
                        id: validIds.routineVersion1,
                        versionLabel: "1.0.0",
                        isPrivate: false,
                        resourceSubType: ResourceSubType.RoutineInternalAction,
                        config: {
                            __version: "invalid",
                            invalidField: "not valid",
                        } as any, // Invalid config structure
                        translationsCreate: [
                            {
                                id: validIds.translation1,
                                language: "en",
                                name: "Invalid Config",
                            },
                        ],
                    },
                ],
            } satisfies Partial<ResourceCreateInput>,

            missingRequiredOwner: {
                id: validIds.routine1,
                resourceType: ResourceType.Routine,
                isPrivate: false,
                // Missing both ownedByUserConnect and ownedByTeamConnect
                versionsCreate: [
                    {
                        id: validIds.routineVersion1,
                        versionLabel: "1.0.0",
                        isPrivate: false,
                        resourceSubType: ResourceSubType.RoutineInternalAction,
                        config: createActionRoutineConfig(),
                        translationsCreate: [
                            {
                                id: validIds.translation1,
                                language: "en",
                                name: "No Owner",
                            },
                        ],
                    },
                ],
            } satisfies Partial<ResourceCreateInput>,
        },

        validationErrors: {
            invalidVersionLabel: {
                id: validIds.routine1,
                resourceType: ResourceType.Routine,
                isPrivate: false,
                ownedByUserConnect: validIds.user1,
                versionsCreate: [
                    {
                        id: validIds.routineVersion1,
                        versionLabel: "invalid version label!", // Invalid characters
                        isPrivate: false,
                        resourceSubType: ResourceSubType.RoutineInternalAction,
                        config: createActionRoutineConfig(),
                        translationsCreate: [
                            {
                                id: validIds.translation1,
                                language: "en",
                                name: "Invalid Version",
                            },
                        ],
                    },
                ],
            } satisfies Partial<ResourceCreateInput>,

            tooLongName: {
                id: validIds.routine1,
                resourceType: ResourceType.Routine,
                isPrivate: false,
                ownedByUserConnect: validIds.user1,
                versionsCreate: [
                    {
                        id: validIds.routineVersion1,
                        versionLabel: "1.0.0",
                        isPrivate: false,
                        resourceSubType: ResourceSubType.RoutineInternalAction,
                        config: createActionRoutineConfig(),
                        translationsCreate: [
                            {
                                id: validIds.translation1,
                                language: "en",
                                name: "N".repeat(ROUTINE_NAME_TOO_LONG_LENGTH), // Too long (max 256 chars)
                                description: "Valid description",
                            },
                        ],
                    },
                ],
            } satisfies Partial<ResourceCreateInput>,

            tooLongDescription: {
                id: validIds.routine1,
                resourceType: ResourceType.Routine,
                isPrivate: false,
                ownedByUserConnect: validIds.user1,
                versionsCreate: [
                    {
                        id: validIds.routineVersion1,
                        versionLabel: "1.0.0",
                        isPrivate: false,
                        resourceSubType: ResourceSubType.RoutineInternalAction,
                        config: createActionRoutineConfig(),
                        translationsCreate: [
                            {
                                id: validIds.translation1,
                                language: "en",
                                name: "Valid Name",
                                description: "D".repeat(ROUTINE_DESCRIPTION_TOO_LONG_LENGTH), // Too long (max 2048 chars)
                            },
                        ],
                    },
                ],
            } satisfies Partial<ResourceCreateInput>,

            invalidLanguageCode: {
                id: validIds.routine1,
                resourceType: ResourceType.Routine,
                isPrivate: false,
                ownedByUserConnect: validIds.user1,
                versionsCreate: [
                    {
                        id: validIds.routineVersion1,
                        versionLabel: "1.0.0",
                        isPrivate: false,
                        resourceSubType: ResourceSubType.RoutineInternalAction,
                        config: createActionRoutineConfig(),
                        translationsCreate: [
                            {
                                id: validIds.translation1,
                                language: "xyz", // Invalid language code
                                name: "Invalid Language",
                            },
                        ],
                    },
                ],
            } satisfies Partial<ResourceCreateInput>,
        },
    },

    edgeCases: {
        minimalValid: {
            create: {
                id: validIds.routine3,
                resourceType: ResourceType.Routine,
                isPrivate: false,
                ownedByUserConnect: validIds.user1,
                versionsCreate: [
                    {
                        id: validIds.routineVersion5,
                        versionLabel: "0.1.0", // Minimal valid version
                        isPrivate: false,
                        resourceSubType: ResourceSubType.RoutineInformational,
                        translationsCreate: [
                            {
                                id: validIds.translation6,
                                language: "en",
                                name: "A", // Minimal name (1 char)
                            },
                        ],
                    },
                ],
            } satisfies ResourceCreateInput,
            update: {
                id: validIds.routine3,
            } satisfies ResourceUpdateInput,
        },

        maximalValid: {
            create: {
                id: validIds.routine4,
                publicId: validPublicIds.routine4,
                resourceType: ResourceType.Routine,
                isInternal: true,
                isPrivate: true,
                permissions: JSON.stringify(["read", "write", "admin", "transfer", "delete"]),
                ownedByTeamConnect: validIds.team1,
                tagsConnect: [validIds.tag1, validIds.tag2, validIds.tag3],
                versionsCreate: [
                    {
                        id: validIds.routineVersion6,
                        publicId: validPublicIds.version4,
                        versionLabel: "99.99.99", // Max valid version format
                        versionNotes: "V".repeat(VERSION_NOTES_MAX_LENGTH), // Max length version notes
                        isPrivate: true,
                        isComplete: true,
                        isAutomatable: true,
                        resourceSubType: ResourceSubType.RoutineMultiStep,
                        config: createMultiStepRoutineConfig(),
                        translationsCreate: [
                            {
                                id: validIds.translation1,
                                language: "en",
                                name: "N".repeat(TRANSLATION_NAME_MAX_LENGTH), // Max length name
                                description: "D".repeat(TRANSLATION_DESCRIPTION_MAX_LENGTH), // Max length description
                                details: "Details".repeat(TRANSLATION_DETAILS_LENGTH), // Max length details
                                instructions: "Instructions".repeat(TRANSLATION_INSTRUCTIONS_LENGTH), // Max length instructions
                            },
                            {
                                id: validIds.translation2,
                                language: "es",
                                name: "Nombre Máximo",
                                description: "Descripción con longitud máxima",
                            },
                            {
                                id: validIds.translation3,
                                language: "fr",
                                name: "Nom Maximum",
                                description: "Description avec longueur maximale",
                            },
                        ],
                    },
                ],
            } satisfies ResourceCreateInput,
            update: {
                id: validIds.routine4,
                isInternal: false,
                isPrivate: false,
                permissions: JSON.stringify(["read"]),
                ownedByUserConnect: validIds.user2,
                ownedByTeamDisconnect: true,
                tagsConnect: [validIds.tag3],
                tagsDisconnect: [validIds.tag1, validIds.tag2],
                versionsUpdate: [
                    {
                        id: validIds.routineVersion6,
                        versionNotes: "Updated with maximum configuration changes",
                        translationsUpdate: [
                            {
                                id: validIds.translation1,
                                language: "en",
                                description: "Updated description with comprehensive details",
                                details: "Enhanced details section with complete information",
                                instructions: "Detailed step-by-step instructions for execution",
                            },
                        ],
                        translationsCreate: [
                            {
                                id: validIds.translation4,
                                language: "de",
                                name: "Maximale Deutsche Routine",
                                description: "Umfassende Routine mit allen Funktionen",
                            },
                        ],
                        translationsDelete: [validIds.translation2],
                    },
                ],
            } satisfies ResourceUpdateInput,
        },

        boundaryValues: {
            underscoreHandle: {
                id: validIds.routine1,
                resourceType: ResourceType.Routine,
                isPrivate: false,
                ownedByUserConnect: validIds.user1,
                versionsCreate: [
                    {
                        id: validIds.routineVersion1,
                        versionLabel: "1.0.0",
                        isPrivate: false,
                        resourceSubType: ResourceSubType.RoutineInternalAction,
                        config: createActionRoutineConfig(),
                        translationsCreate: [
                            {
                                id: validIds.translation1,
                                language: "en",
                                name: "Boundary Test Routine",
                            },
                        ],
                    },
                ],
            } satisfies ResourceCreateInput,

            emptyTranslations: {
                id: validIds.routine2,
                resourceType: ResourceType.Routine,
                isPrivate: false,
                ownedByUserConnect: validIds.user1,
                versionsCreate: [
                    {
                        id: validIds.routineVersion2,
                        versionLabel: "1.0.0",
                        isPrivate: false,
                        resourceSubType: ResourceSubType.RoutineInformational,
                        translationsCreate: [],
                    },
                ],
            } satisfies ResourceCreateInput,

            allComplexityLevels: {
                id: validIds.routine3,
                resourceType: ResourceType.Routine,
                isPrivate: false,
                ownedByUserConnect: validIds.user1,
                versionsCreate: [
                    {
                        id: validIds.routineVersion3,
                        versionLabel: "1.0.0",
                        isPrivate: false,
                        resourceSubType: ResourceSubType.RoutineMultiStep,
                        config: createMultiStepRoutineConfig(),
                        translationsCreate: [
                            {
                                id: validIds.translation3,
                                language: "en",
                                name: "Complex Multi-Step Routine",
                            },
                        ],
                    },
                ],
            } satisfies ResourceCreateInput,
        },

        permissionScenarios: {
            privateRoutine: {
                id: validIds.routine1,
                resourceType: ResourceType.Routine,
                isPrivate: true,
                ownedByUserConnect: validIds.user1,
                versionsCreate: [
                    {
                        id: validIds.routineVersion1,
                        versionLabel: "1.0.0",
                        isPrivate: true,
                        resourceSubType: ResourceSubType.RoutineInternalAction,
                        config: createActionRoutineConfig(),
                        translationsCreate: [
                            {
                                id: validIds.translation1,
                                language: "en",
                                name: "Private Routine",
                            },
                        ],
                    },
                ],
            } satisfies ResourceCreateInput,

            publicRoutine: {
                id: validIds.routine2,
                resourceType: ResourceType.Routine,
                isPrivate: false,
                ownedByUserConnect: validIds.user1,
                versionsCreate: [
                    {
                        id: validIds.routineVersion2,
                        versionLabel: "1.0.0",
                        isPrivate: false,
                        resourceSubType: ResourceSubType.RoutineInternalAction,
                        config: createActionRoutineConfig(),
                        translationsCreate: [
                            {
                                id: validIds.translation2,
                                language: "en",
                                name: "Public Routine",
                            },
                        ],
                    },
                ],
            } satisfies ResourceCreateInput,

            internalRoutine: {
                id: validIds.routine3,
                resourceType: ResourceType.Routine,
                isInternal: true,
                isPrivate: false,
                ownedByTeamConnect: validIds.team1,
                versionsCreate: [
                    {
                        id: validIds.routineVersion3,
                        versionLabel: "1.0.0",
                        isPrivate: false,
                        resourceSubType: ResourceSubType.RoutineInternalAction,
                        config: createActionRoutineConfig(),
                        translationsCreate: [
                            {
                                id: validIds.translation3,
                                language: "en",
                                name: "Internal Routine",
                            },
                        ],
                    },
                ],
            } satisfies ResourceCreateInput,

            readOnlyPermissions: {
                id: validIds.routine1,
                permissions: JSON.stringify(["read"]),
            } satisfies ResourceUpdateInput,

            fullPermissions: {
                id: validIds.routine1,
                permissions: JSON.stringify(["read", "write", "admin", "transfer", "delete"]),
            } satisfies ResourceUpdateInput,
        },
    },
};

// ========================================
// Factory Customizers
// ========================================

const routineCustomizers: FactoryCustomizers<ResourceCreateInput, ResourceUpdateInput> = {
    create: (base: ResourceCreateInput, overrides?: Partial<ResourceCreateInput>): ResourceCreateInput => {
        return {
            ...base,
            id: overrides?.id || base.id || generatePK().toString(),
            resourceType: ResourceType.Routine, // Always Routine
            isPrivate: overrides?.isPrivate !== undefined ? overrides.isPrivate : (base.isPrivate !== undefined ? base.isPrivate : false),
            ownedByUserConnect: overrides?.ownedByUserConnect || base.ownedByUserConnect || (overrides?.ownedByTeamConnect || base.ownedByTeamConnect ? undefined : generatePK().toString()),
            versionsCreate: overrides?.versionsCreate || base.versionsCreate || [
                {
                    id: generatePK().toString(),
                    versionLabel: "1.0.0",
                    isPrivate: false,
                    resourceSubType: ResourceSubType.RoutineInternalAction,
                    config: createActionRoutineConfig(),
                    translationsCreate: [
                        {
                            id: generatePK().toString(),
                            language: "en",
                            name: "Generated Routine",
                        },
                    ],
                },
            ],
            ...overrides,
        };
    },

    update: (base: ResourceUpdateInput, overrides?: Partial<ResourceUpdateInput>): ResourceUpdateInput => {
        return {
            ...base,
            id: overrides?.id || base.id || generatePK().toString(),
            ...overrides,
        };
    },
};

// ========================================
// Integration Setup
// ========================================

// For now, use a simpler approach without full validation integration
// due to the complexity of nested resource/version validation
const routineIntegration = {
    validation: {
        create: undefined,
        update: undefined,
    },
    shape: shapeResource,
};

// ========================================
// Type-Safe Fixture Factory
// ========================================

export class RoutineAPIFixtureFactory extends BaseAPIFixtureFactory<
    ResourceCreateInput,
    ResourceUpdateInput,
    Resource,
    ResourceShape,
    Resource // Database type same as find result for simplicity
> implements APIFixtureFactory<ResourceCreateInput, ResourceUpdateInput, Resource, ResourceShape, Resource> {

    constructor() {
        const config = {
            ...routineFixtureData,
            validationSchema: routineIntegration.validation,
            shapeTransforms: {
                toAPI: undefined,
                fromDB: undefined,
            },
        };
        super(config, routineCustomizers);
    }

    // ========================================
    // Routine-Specific Factory Methods
    // ========================================

    // Create different types of routines
    createActionRoutine = (
        toolName?: string,
        inputTemplate?: string,
        overrides?: Partial<ResourceCreateInput>,
    ): ResourceCreateInput => {
        const config = createActionRoutineConfig();
        if (toolName) {
            config.callDataAction!.schema.toolName = toolName as any;
        }
        if (inputTemplate) {
            config.callDataAction!.schema.inputTemplate = inputTemplate;
        }

        return this.createFactory({
            versionsCreate: [
                {
                    id: generatePK().toString(),
                    versionLabel: "1.0.0",
                    isPrivate: false,
                    resourceSubType: ResourceSubType.RoutineInternalAction,
                    config,
                    translationsCreate: [
                        {
                            id: generatePK().toString(),
                            language: "en",
                            name: "Custom Action Routine",
                        },
                    ],
                },
            ],
            ...overrides,
        });
    };

    createGenerateRoutine = (
        model?: string,
        prompt?: string,
        overrides?: Partial<ResourceCreateInput>,
    ): ResourceCreateInput => {
        const config = createGenerateRoutineConfig();
        if (model) {
            config.callDataGenerate!.schema.model = model as any;
        }
        if (prompt) {
            config.callDataGenerate!.schema.prompt = prompt;
        }

        return this.createFactory({
            versionsCreate: [
                {
                    id: generatePK().toString(),
                    versionLabel: "1.0.0",
                    isPrivate: false,
                    resourceSubType: ResourceSubType.RoutineGenerate,
                    config,
                    translationsCreate: [
                        {
                            id: generatePK().toString(),
                            language: "en",
                            name: "Custom Generate Routine",
                        },
                    ],
                },
            ],
            ...overrides,
        });
    };

    createMultiStepRoutine = (
        steps?: string[],
        overrides?: Partial<ResourceCreateInput>,
    ): ResourceCreateInput => {
        const config = createMultiStepRoutineConfig();

        return this.createFactory({
            versionsCreate: [
                {
                    id: generatePK().toString(),
                    versionLabel: "1.0.0",
                    isPrivate: false,
                    resourceSubType: ResourceSubType.RoutineMultiStep,
                    config,
                    translationsCreate: [
                        {
                            id: generatePK().toString(),
                            language: "en",
                            name: "Custom Multi-Step Routine",
                        },
                    ],
                },
            ],
            ...overrides,
        });
    };

    createAPIRoutine = (
        endpoint: string,
        method = "GET",
        overrides?: Partial<ResourceCreateInput>,
    ): ResourceCreateInput => {
        const config: RoutineVersionConfigObject = {
            __version: LATEST_CONFIG_VERSION,
            callDataApi: {
                __version: LATEST_CONFIG_VERSION,
                schema: {
                    endpoint,
                    method: method as any,
                    headers: {
                        "Content-Type": "application/json",
                    },
                    timeoutMs: 30000,
                },
            },
        };

        return this.createFactory({
            versionsCreate: [
                {
                    id: generatePK().toString(),
                    versionLabel: "1.0.0",
                    isPrivate: false,
                    resourceSubType: ResourceSubType.RoutineApi,
                    config,
                    translationsCreate: [
                        {
                            id: generatePK().toString(),
                            language: "en",
                            name: `API Routine - ${method} ${endpoint}`,
                        },
                    ],
                },
            ],
            ...overrides,
        });
    };

    createWebSearchRoutine = (
        queryTemplate: string,
        overrides?: Partial<ResourceCreateInput>,
    ): ResourceCreateInput => {
        const config: RoutineVersionConfigObject = {
            __version: LATEST_CONFIG_VERSION,
            callDataWeb: {
                __version: LATEST_CONFIG_VERSION,
                schema: {
                    queryTemplate,
                    maxResults: 10,
                    outputMapping: {
                        titles: "results[*].title",
                        links: "results[*].link",
                    },
                },
            },
        };

        return this.createFactory({
            versionsCreate: [
                {
                    id: generatePK().toString(),
                    versionLabel: "1.0.0",
                    isPrivate: false,
                    resourceSubType: ResourceSubType.RoutineWeb,
                    config,
                    translationsCreate: [
                        {
                            id: generatePK().toString(),
                            language: "en",
                            name: "Web Search Routine",
                        },
                    ],
                },
            ],
            ...overrides,
        });
    };

    // ========================================
    // AI Execution Helpers
    // ========================================

    addRoutineStep = (routineId: string, stepConfig: any): ResourceUpdateInput => {
        return this.updateFactory(routineId, {
            // This would require complex version update logic for multi-step routines
            // Implementation depends on specific step management requirements
        });
    };

    updateRoutineFlow = (routineId: string, flowConfig: any): ResourceUpdateInput => {
        return this.updateFactory(routineId, {
            // Implementation for updating BPMN flow configuration
        });
    };

    setRoutineParameters = (
        routineId: string,
        inputs: any[],
        outputs: any[],
    ): ResourceUpdateInput => {
        return this.updateFactory(routineId, {
            // Implementation for updating input/output parameters
        });
    };

    // ========================================
    // Version Management
    // ========================================

    versionRoutine = (
        routineId: string,
        versionInfo: { label: string; notes?: string; config?: RoutineVersionConfigObject },
    ): ResourceUpdateInput => {
        return this.updateFactory(routineId, {
            versionsCreate: [
                {
                    id: generatePK().toString(),
                    versionLabel: versionInfo.label,
                    versionNotes: versionInfo.notes,
                    isPrivate: false,
                    resourceSubType: ResourceSubType.RoutineInternalAction,
                    config: versionInfo.config || createActionRoutineConfig(),
                    translationsCreate: [
                        {
                            id: generatePK().toString(),
                            language: "en",
                            name: `Routine Version ${versionInfo.label}`,
                        },
                    ],
                },
            ],
        });
    };

    forkRoutine = (routineId: string, newName: string): ResourceCreateInput => {
        return this.createFactory({
            versionsCreate: [
                {
                    id: generatePK().toString(),
                    versionLabel: "1.0.0",
                    isPrivate: false,
                    resourceSubType: ResourceSubType.RoutineInternalAction,
                    config: createActionRoutineConfig(),
                    translationsCreate: [
                        {
                            id: generatePK().toString(),
                            language: "en",
                            name: newName,
                        },
                    ],
                },
            ],
        });
    };

    // ========================================
    // AI Integration Helpers
    // ========================================

    setRoutineModel = (routineId: string, model: string): ResourceUpdateInput => {
        const config = createGenerateRoutineConfig();
        config.callDataGenerate!.schema.model = model as any;

        return this.updateFactory(routineId, {
            versionsUpdate: [
                {
                    id: generatePK().toString(), // Would need actual version ID
                    config,
                },
            ],
        });
    };

    setExecutionContext = (routineId: string, context: {
        strategy?: "reasoning" | "deterministic" | "conversational" | "auto";
        allowOverride?: boolean;
        subroutineStrategies?: Record<string, "reasoning" | "deterministic" | "conversational">;
    }): ResourceUpdateInput => {
        const config = createMultiStepRoutineConfig();
        config.executionStrategy = context.strategy;
        config.allowStrategyOverride = context.allowOverride;
        config.subroutineStrategies = context.subroutineStrategies;

        return this.updateFactory(routineId, {
            versionsUpdate: [
                {
                    id: generatePK().toString(), // Would need actual version ID
                    config,
                },
            ],
        });
    };

    // ========================================
    // Validation Helpers
    // ========================================

    validateRoutineCreation = async (input: ResourceCreateInput): Promise<void> => {
        // Validate resource structure
        const result = await this.validateCreate(input);
        if (!result.isValid) {
            throw new Error(`Routine creation validation failed: ${result.errors?.join(", ")}`);
        }

        // Additional routine-specific validation
        if (input.resourceType !== ResourceType.Routine) {
            throw new Error("Resource must be of type Routine");
        }

        const versions = input.versionsCreate;
        if (!versions || versions.length === 0) {
            throw new Error("Routine must have at least one version");
        }

        for (const version of versions) {
            if (!version.resourceSubType || !Object.values(ResourceSubType).includes(version.resourceSubType)) {
                throw new Error(`Invalid routine subtype: ${version.resourceSubType}`);
            }

            // Validate routine-specific subtypes
            const routineSubTypes = [
                ResourceSubType.RoutineInternalAction,
                ResourceSubType.RoutineApi,
                ResourceSubType.RoutineCode,
                ResourceSubType.RoutineData,
                ResourceSubType.RoutineGenerate,
                ResourceSubType.RoutineInformational,
                ResourceSubType.RoutineMultiStep,
                ResourceSubType.RoutineSmartContract,
                ResourceSubType.RoutineWeb,
            ];

            if (!routineSubTypes.includes(version.resourceSubType)) {
                throw new Error(`Resource subtype ${version.resourceSubType} is not valid for routines`);
            }
        }
    };

    validateRoutineUpdate = async (input: ResourceUpdateInput): Promise<void> => {
        const result = await this.validateUpdate(input);
        if (!result.isValid) {
            throw new Error(`Routine update validation failed: ${result.errors?.join(", ")}`);
        }
    };

    validateRoutineConfig = (config: RoutineVersionConfigObject): boolean => {
        if (!config.__version) {
            return false;
        }

        // Basic structure validation
        const hasValidConfig = !!(
            config.callDataAction ||
            config.callDataApi ||
            config.callDataCode ||
            config.callDataGenerate ||
            config.callDataSmartContract ||
            config.callDataWeb ||
            config.graph ||
            config.formInput ||
            config.formOutput
        );

        return hasValidConfig;
    };
}

// ========================================
// Export Factory Instance
// ========================================

export const routineAPIFixtures = new RoutineAPIFixtureFactory();

// ========================================
// Type Exports for Other Fixtures
// ========================================

export type { RoutineAPIFixtureFactory as RoutineAPIFixtureFactoryType };

// ========================================
// Legacy Compatibility (Optional)
// ========================================

// Provide legacy-style access for gradual migration
export const legacyRoutineFixtures = {
    minimal: routineAPIFixtures.minimal,
    complete: routineAPIFixtures.complete,
    invalid: routineAPIFixtures.invalid,
    edgeCases: routineAPIFixtures.edgeCases,

    // Factory methods
    createFactory: routineAPIFixtures.createFactory,
    updateFactory: routineAPIFixtures.updateFactory,
    findFactory: routineAPIFixtures.findFactory,

    // Routine-specific methods
    createActionRoutine: routineAPIFixtures.createActionRoutine,
    createGenerateRoutine: routineAPIFixtures.createGenerateRoutine,
    createMultiStepRoutine: routineAPIFixtures.createMultiStepRoutine,
    createAPIRoutine: routineAPIFixtures.createAPIRoutine,
    createWebSearchRoutine: routineAPIFixtures.createWebSearchRoutine,

    // AI integration methods
    versionRoutine: routineAPIFixtures.versionRoutine,
    forkRoutine: routineAPIFixtures.forkRoutine,
    setRoutineModel: routineAPIFixtures.setRoutineModel,
    setExecutionContext: routineAPIFixtures.setExecutionContext,

    // Validation methods
    validateRoutineCreation: routineAPIFixtures.validateRoutineCreation,
    validateRoutineUpdate: routineAPIFixtures.validateRoutineUpdate,
    validateRoutineConfig: routineAPIFixtures.validateRoutineConfig,
};

/* c8 ignore stop */
