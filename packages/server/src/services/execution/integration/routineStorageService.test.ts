import { beforeEach, describe, expect, it, vi } from "vitest";
import type { PrismaClient } from "@prisma/client";
import { type Logger } from "winston";
import { InputType, McpToolName, ResourceSubType, generatePK } from "@vrooli/shared";
import { RoutineStorageService } from "./routineStorageService.js";

/**
 * Test fixtures for routine configurations
 * These demonstrate the data-driven nature of routines where intelligence emerges from configuration
 */
const createActionRoutineConfig = () => ({
    __version: "1.0",
    callDataAction: {
        toolName: McpToolName.createProject,
        inputTemplate: {
            id: "{{generatePrimaryKey()}}",
            name: "{{input.projectName}}",
            description: "{{input.projectDescription}}",
            isPrivate: "{{input.isPrivate}}",
        },
    },
    formInput: {
        __version: "1.0",
        schema: {
            containers: [],
            elements: [
                {
                    fieldName: "projectName",
                    id: "project_name",
                    label: "Project Name",
                    type: InputType.Text,
                    isRequired: true,
                },
                {
                    fieldName: "projectDescription",
                    id: "project_desc",
                    label: "Description",
                    type: InputType.Text,
                    isRequired: false,
                },
                {
                    fieldName: "isPrivate",
                    id: "is_private",
                    label: "Private",
                    type: InputType.Switch,
                    defaultValue: false,
                },
            ],
        },
    },
});

const createApiRoutineConfig = () => ({
    __version: "1.0",
    callDataApi: {
        method: "POST",
        url: "https://api.example.com/users",
        headers: {
            "Content-Type": "application/json",
            "Authorization": "Bearer {{input.apiKey}}",
        },
        body: {
            email: "{{input.email}}",
            name: "{{input.name}}",
        },
    },
    formInput: {
        __version: "1.0",
        schema: {
            containers: [],
            elements: [
                {
                    fieldName: "email",
                    id: "email",
                    label: "Email",
                    type: InputType.Text,
                    isRequired: true,
                },
                {
                    fieldName: "name",
                    id: "name",
                    label: "Name",
                    type: InputType.Text,
                    isRequired: true,
                },
                {
                    fieldName: "apiKey",
                    id: "api_key",
                    label: "API Key",
                    type: InputType.Password,
                    isRequired: true,
                },
            ],
        },
    },
});

const createCodeRoutineConfig = () => ({
    __version: "1.0",
    callDataCode: {
        __version: "1.0",
        schema: {
            inputTemplate: {
                numbers: "{{input.numbers}}",
            },
            outputMappings: [
                {
                    schemaIndex: 0,
                    mapping: {
                        sum: "sum",
                        average: "average",
                        min: "min",
                        max: "max",
                    },
                },
            ],
        },
    },
    formInput: {
        __version: "1.0",
        schema: {
            containers: [],
            elements: [
                {
                    fieldName: "numbers",
                    id: "numbers",
                    label: "Numbers",
                    type: InputType.JSON,
                    isRequired: true,
                    props: {
                        placeholder: "[\"1\", \"2\", \"3\"]",
                    },
                },
            ],
        },
    },
});

const createGraphRoutineConfig = () => ({
    __version: "1.0",
    graph: {
        type: "sequential",
        nodes: [
            {
                id: "start",
                type: "start",
                name: "Start",
            },
            {
                id: "step1",
                type: "action",
                name: "Step 1",
                config: {
                    toolName: McpToolName.createProject,
                },
            },
            {
                id: "end",
                type: "end",
                name: "End",
            },
        ],
        edges: [
            { from: "start", to: "step1" },
            { from: "step1", to: "end" },
        ],
    },
});

describe("RoutineStorageService", () => {
    let service: RoutineStorageService;
    let mockPrisma: any;
    let mockLogger: Logger;

    beforeEach(() => {
        // Mock Prisma client
        mockPrisma = {
            resource_version: {
                findFirst: vi.fn(),
                findMany: vi.fn(),
            },
            resource: {
                findFirst: vi.fn(),
            },
            member: {
                findFirst: vi.fn(),
            },
        };

        // Mock logger
        mockLogger = {
            debug: vi.fn(),
            info: vi.fn(),
            warn: vi.fn(),
            error: vi.fn(),
        } as any;

        service = new RoutineStorageService(mockPrisma as PrismaClient, mockLogger);
    });

    describe("parseDefinition", () => {
        it("parses config object correctly", async () => {
            const config = createActionRoutineConfig();
            const resourceVersion = {
                id: BigInt(generatePK()),
                publicId: "test_routine_1",
                config,
                versionLabel: "1.0.0",
                resourceType: ResourceSubType.RoutineAction,
            };

            mockPrisma.resource_version.findFirst.mockResolvedValue({
                ...resourceVersion,
                root: { publicId: "routine_root_1" },
                translations: [{ language: { code: "en" }, name: "Test Routine" }],
            });

            const routine = await service.loadRoutine("test_routine_1");

            expect(routine.definition).toEqual(config);
            expect(routine.type).toBe("action");
        });

        it("parses config string correctly", async () => {
            const config = createApiRoutineConfig();
            const resourceVersion = {
                id: BigInt(generatePK()),
                publicId: "test_routine_2",
                config: JSON.stringify(config), // String format
                versionLabel: "1.0.0",
                resourceType: ResourceSubType.RoutineApi,
            };

            mockPrisma.resource_version.findFirst.mockResolvedValue({
                ...resourceVersion,
                root: { publicId: "routine_root_2" },
                translations: [{ language: { code: "en" }, name: "API Routine" }],
            });

            const routine = await service.loadRoutine("test_routine_2");

            expect(routine.definition).toEqual(config);
            expect(routine.type).toBe("api");
        });

        it("returns empty object when config is missing", async () => {
            const resourceVersion = {
                id: BigInt(generatePK()),
                publicId: "test_routine_3",
                // No config field
                versionLabel: "1.0.0",
                resourceType: ResourceSubType.RoutineGeneric,
            };

            mockPrisma.resource_version.findFirst.mockResolvedValue({
                ...resourceVersion,
                root: { publicId: "routine_root_3" },
                translations: [{ language: { code: "en" }, name: "Empty Routine" }],
            });

            const routine = await service.loadRoutine("test_routine_3");

            expect(routine.definition).toEqual({});
            expect(mockLogger.warn).toHaveBeenCalledWith(
                "[RoutineStorageService] Resource version missing config",
                expect.objectContaining({
                    resourceVersionId: resourceVersion.id,
                    publicId: resourceVersion.publicId,
                }),
            );
        });

        it("handles JSON parse errors gracefully", async () => {
            const resourceVersion = {
                id: BigInt(generatePK()),
                publicId: "test_routine_4",
                config: "{ invalid json", // Invalid JSON
                versionLabel: "1.0.0",
                resourceType: ResourceSubType.RoutineGeneric,
            };

            mockPrisma.resource_version.findFirst.mockResolvedValue({
                ...resourceVersion,
                root: { publicId: "routine_root_4" },
                translations: [{ language: { code: "en" }, name: "Invalid Routine" }],
            });

            const routine = await service.loadRoutine("test_routine_4");

            expect(routine.definition).toEqual({});
            expect(mockLogger.warn).toHaveBeenCalledWith(
                "[RoutineStorageService] Failed to parse definition",
                expect.objectContaining({
                    resourceVersionId: resourceVersion.id,
                    error: expect.any(String),
                }),
            );
        });
    });

    describe("inferRoutineType", () => {
        it("identifies action routine type", async () => {
            const config = createActionRoutineConfig();
            const resourceVersion = {
                id: BigInt(generatePK()),
                publicId: "action_routine",
                config,
                versionLabel: "1.0.0",
                resourceType: ResourceSubType.RoutineAction,
            };

            mockPrisma.resource_version.findFirst.mockResolvedValue({
                ...resourceVersion,
                root: { publicId: "routine_root" },
                translations: [{ language: { code: "en" }, name: "Action Routine" }],
            });

            const routine = await service.loadRoutine("action_routine");
            expect(routine.type).toBe("action");
        });

        it("identifies API routine type", async () => {
            const config = createApiRoutineConfig();
            const resourceVersion = {
                id: BigInt(generatePK()),
                publicId: "api_routine",
                config,
                versionLabel: "1.0.0",
                resourceType: ResourceSubType.RoutineApi,
            };

            mockPrisma.resource_version.findFirst.mockResolvedValue({
                ...resourceVersion,
                root: { publicId: "routine_root" },
                translations: [{ language: { code: "en" }, name: "API Routine" }],
            });

            const routine = await service.loadRoutine("api_routine");
            expect(routine.type).toBe("api");
        });

        it("identifies code routine type", async () => {
            const config = createCodeRoutineConfig();
            const resourceVersion = {
                id: BigInt(generatePK()),
                publicId: "code_routine",
                config,
                versionLabel: "1.0.0",
                resourceType: ResourceSubType.RoutineCode,
            };

            mockPrisma.resource_version.findFirst.mockResolvedValue({
                ...resourceVersion,
                root: { publicId: "routine_root" },
                translations: [{ language: { code: "en" }, name: "Code Routine" }],
            });

            const routine = await service.loadRoutine("code_routine");
            expect(routine.type).toBe("code");
        });

        it("identifies graph routine type", async () => {
            const config = createGraphRoutineConfig();
            const resourceVersion = {
                id: BigInt(generatePK()),
                publicId: "graph_routine",
                config,
                versionLabel: "1.0.0",
                resourceType: ResourceSubType.RoutineGraph,
            };

            mockPrisma.resource_version.findFirst.mockResolvedValue({
                ...resourceVersion,
                root: { publicId: "routine_root" },
                translations: [{ language: { code: "en" }, name: "Graph Routine" }],
            });

            const routine = await service.loadRoutine("graph_routine");
            expect(routine.type).toBe("sequential");
        });

        it("defaults to native for unknown types", async () => {
            const config = { __version: "1.0", unknownField: "test" };
            const resourceVersion = {
                id: BigInt(generatePK()),
                publicId: "unknown_routine",
                config,
                versionLabel: "1.0.0",
                resourceType: ResourceSubType.RoutineGeneric,
            };

            mockPrisma.resource_version.findFirst.mockResolvedValue({
                ...resourceVersion,
                root: { publicId: "routine_root" },
                translations: [{ language: { code: "en" }, name: "Unknown Routine" }],
            });

            const routine = await service.loadRoutine("unknown_routine");
            expect(routine.type).toBe("native");
        });
    });

    describe("loadRoutine", () => {
        it("loads routine by numeric ID", async () => {
            const config = createActionRoutineConfig();
            const resourceVersion = {
                id: BigInt("123456789"),
                publicId: "routine_123",
                config,
                versionLabel: "2.1.0",
                resourceType: ResourceSubType.RoutineAction,
                isPrivate: false,
                createdAt: new Date(),
                updatedAt: new Date(),
            };

            mockPrisma.resource_version.findFirst.mockResolvedValue({
                ...resourceVersion,
                root: {
                    publicId: "routine_root",
                    createdById: BigInt("987654321"),
                    createdAt: resourceVersion.createdAt,
                    updatedAt: resourceVersion.updatedAt,
                    createdBy: { id: BigInt("987654321"), handle: "testuser" },
                    team: null,
                },
                translations: [
                    { 
                        language: { code: "en" }, 
                        name: "Create Project Routine",
                        description: "Creates a new project with metadata",
                    },
                ],
                resourceList: null,
            });

            const routine = await service.loadRoutine("123456789");

            expect(routine).toEqual({
                id: "routine_123",
                type: "action",
                version: "2.1.0",
                name: "Create Project Routine",
                definition: config,
                metadata: {
                    author: "987654321",
                    createdAt: resourceVersion.createdAt,
                    updatedAt: resourceVersion.updatedAt,
                    tags: [],
                    complexity: "simple",
                    version: "2.1.0",
                    description: "Creates a new project with metadata",
                },
            });

            expect(mockPrisma.resource_version.findFirst).toHaveBeenCalledWith({
                where: {
                    OR: [
                        { id: BigInt("123456789") },
                        { publicId: "123456789" },
                        { root: { publicId: "123456789" } },
                    ],
                },
                include: expect.any(Object),
                orderBy: { versionIndex: "desc" },
            });
        });

        it("throws error when routine not found", async () => {
            mockPrisma.resource_version.findFirst.mockResolvedValue(null);

            await expect(service.loadRoutine("non_existent"))
                .rejects.toThrow("Routine not found: non_existent");

            expect(mockLogger.error).toHaveBeenCalledWith(
                "[RoutineStorageService] Failed to load routine",
                expect.objectContaining({
                    routineId: "non_existent",
                    error: "Routine not found: non_existent",
                }),
            );
        });
    });

    describe("searchRoutines", () => {
        it("searches routines with query", async () => {
            const routines = [
                {
                    id: BigInt(generatePK()),
                    publicId: "routine_1",
                    config: createActionRoutineConfig(),
                    versionLabel: "1.0.0",
                    resourceType: ResourceSubType.RoutineAction,
                    root: { publicId: "root_1" },
                    translations: [{ language: { code: "en" }, name: "Routine 1" }],
                },
                {
                    id: BigInt(generatePK()),
                    publicId: "routine_2",
                    config: createApiRoutineConfig(),
                    versionLabel: "1.0.0",
                    resourceType: ResourceSubType.RoutineApi,
                    root: { publicId: "root_2" },
                    translations: [{ language: { code: "en" }, name: "Routine 2" }],
                },
            ];

            mockPrisma.resource_version.findMany.mockResolvedValue(routines);

            const results = await service.searchRoutines({
                query: "test",
                limit: 10,
                offset: 0,
            });

            expect(results).toHaveLength(2);
            expect(results[0].type).toBe("action");
            expect(results[1].type).toBe("api");

            expect(mockPrisma.resource_version.findMany).toHaveBeenCalled();
            const callArgs = mockPrisma.resource_version.findMany.mock.calls[0][0];
            expect(callArgs.take).toBe(10);
            expect(callArgs.skip).toBe(0);
            // Check the AND array contains the query conditions
            const andConditions = callArgs.where.AND;
            const queryCondition = andConditions.find((cond: any) => cond.OR);
            expect(queryCondition.OR[0].translations.some.name.contains).toBe("test");
            expect(queryCondition.OR[0].translations.some.name.mode).toBe("insensitive");
        });

        it("filters by tags", async () => {
            // Create a mock result
            mockPrisma.resource_version.findMany.mockResolvedValue([]);
            
            await service.searchRoutines({
                tags: ["api", "integration"],
                limit: 5,
            });

            expect(mockPrisma.resource_version.findMany).toHaveBeenCalled();
            const callArgs = mockPrisma.resource_version.findMany.mock.calls[0][0];
            expect(callArgs.take).toBe(5);
            // Check the AND array contains the tags condition
            const andConditions = callArgs.where.AND;
            const tagsCondition = andConditions.find((cond: any) => cond.tags);
            expect(tagsCondition.tags.some.tag.tag.in).toEqual(["api", "integration"]);
        });
    });
});
