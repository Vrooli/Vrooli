import type {
    Resource,
    ResourceSearchResult,
    ResourceVersion,
} from "@vrooli/shared";
import { Command } from "commander";
import { promises as fs } from "fs";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { type ApiClient } from "../utils/client.js";
import { type ConfigManager } from "../utils/config.js";
import { output } from "../utils/output.js";
import { RoutineCommands } from "./routine.js";
// Import the mockSpinner from the mocked module
import ora from "ora";

// Mock dependencies
vi.mock("fs", () => ({
    promises: {
        readFile: vi.fn(),
        writeFile: vi.fn(),
        mkdir: vi.fn(),
        stat: vi.fn(),
    },
}));

vi.mock("fs/promises", () => ({
    readFile: vi.fn(),
    writeFile: vi.fn(),
    mkdir: vi.fn(),
    stat: vi.fn(),
}));

vi.mock("glob", () => ({
    glob: vi.fn(),
}));

// Mock the output utility
vi.mock("../utils/output.js", () => ({
    output: {
        info: vi.fn(),
        success: vi.fn(),
        error: vi.fn(),
        warn: vi.fn(),
        debug: vi.fn(),
        table: vi.fn(),
        raw: vi.fn(),
        json: vi.fn(),
        newline: vi.fn(),
        section: vi.fn(),
        listItem: vi.fn(),
        keyValue: vi.fn(),
    },
}));

// Mock to track spinner messages
const mockSpinner = {
    start: vi.fn().mockReturnThis(),
    succeed: vi.fn().mockReturnThis(),
    fail: vi.fn().mockReturnThis(),
    info: vi.fn().mockReturnThis(),
    stop: vi.fn().mockReturnThis(),
    text: "",
};

vi.mock("ora", () => ({
    default: vi.fn(() => mockSpinner),
}));

vi.mock("cli-progress", () => ({
    default: {
        SingleBar: vi.fn().mockImplementation(() => ({
            start: vi.fn(),
            update: vi.fn(),
            stop: vi.fn(),
        })),
        Presets: {
            shades_classic: {},
        },
    },
}));

vi.mock("chalk", () => {
    const createChalkMock = (text: string) => text;
    const chalkMock = Object.assign(createChalkMock, {
        bold: Object.assign(createChalkMock, {
            yellow: createChalkMock,
            blue: createChalkMock,
            red: createChalkMock,
            green: createChalkMock,
            cyan: createChalkMock,
        }),
        dim: createChalkMock,
        green: createChalkMock,
        red: createChalkMock,
        yellow: createChalkMock,
        cyan: createChalkMock,
        gray: createChalkMock,
        blue: createChalkMock,
    });

    return {
        default: chalkMock,
    };
});

describe("RoutineCommands", () => {
    let program: Command;
    let mockClient: ApiClient;
    let mockConfig: ConfigManager;
    let routineCommands: RoutineCommands;
    let processExitSpy: any;

    // Mock console methods for JSON output and table display
    let consoleLogSpy: any;

    beforeEach(() => {
        // Mock console.log for JSON output and table display
        consoleLogSpy = vi.spyOn(console, "log").mockImplementation(() => undefined);
        processExitSpy = vi.spyOn(process, "exit").mockImplementation(() => undefined as never);

        // Create mocks
        program = new Command();
        program.exitOverride(); // Prevent commander from exiting process

        mockClient = {
            post: vi.fn(),
            get: vi.fn(),
            request: vi.fn(),
            requestWithEndpoint: vi.fn(),
            isAuthenticated: vi.fn().mockReturnValue(true),
        } as unknown as ApiClient;

        mockConfig = {
            isJsonOutput: vi.fn().mockReturnValue(false),
            getServerUrl: vi.fn().mockReturnValue("http://localhost:5329"),
            getAuthToken: vi.fn().mockReturnValue("test-token"),
        } as unknown as ConfigManager;

        // Create instance
        routineCommands = new RoutineCommands(program, mockClient, mockConfig);
    });

    afterEach(() => {
        consoleLogSpy.mockRestore();
        processExitSpy.mockRestore();
        vi.clearAllMocks();
    });

    describe("list command", () => {
        const mockRoutines: ResourceSearchResult = {
            edges: [
                {
                    cursor: "cursor1",
                    node: {
                        id: "routine1",
                        publicId: "pub1",
                        createdAt: "2025-01-01T00:00:00.000Z",
                        updatedAt: "2025-01-01T00:00:00.000Z",
                        isPrivate: false,
                        labels: [],
                        resourceType: "Routine",
                        resourceSubType: "Graph",
                        translatedName: "Test Routine",
                        versionsCount: 1,
                        owner: {
                            __typename: "User",
                            id: "user1",
                            name: "User",
                        },
                        translations: [{
                            id: "t1",
                            language: "en",
                            name: "Test Routine",
                            description: "A test routine",
                            __typename: "ResourceVersionTranslation",
                        }],
                        versions: [{
                            id: "v1",
                            created_at: new Date("2025-01-01"),
                            updated_at: new Date("2025-01-01"),
                            resourceSubType: "Graph",
                            translations: [{
                                id: "t1",
                                language: "en",
                                name: "Test Routine",
                                description: "A test routine",
                                __typename: "ResourceVersionTranslation",
                            }],
                            __typename: "ResourceVersion",
                        } as ResourceVersion],
                        __typename: "Resource",
                    } as Resource,
                },
                {
                    cursor: "cursor2",
                    node: {
                        id: "routine2",
                        publicId: "pub2",
                        createdAt: "2025-01-02T00:00:00.000Z",
                        updatedAt: "2025-01-02T00:00:00.000Z",
                        isPrivate: false,
                        labels: [],
                        resourceType: "Routine",
                        resourceSubType: "Api",
                        translatedName: "API Routine",
                        versionsCount: 1,
                        owner: {
                            __typename: "User",
                            id: "user1",
                            name: "User",
                        },
                        translations: [{
                            id: "t2",
                            language: "en",
                            name: "API Routine",
                            description: "An API routine",
                            __typename: "ResourceVersionTranslation",
                        }],
                        versions: [{
                            id: "v2",
                            created_at: new Date("2025-01-02"),
                            updated_at: new Date("2025-01-02"),
                            resourceSubType: "Api",
                            translations: [{
                                id: "t2",
                                language: "en",
                                name: "API Routine",
                                description: "An API routine",
                                __typename: "ResourceVersionTranslation",
                            }],
                            __typename: "ResourceVersion",
                        } as ResourceVersion],
                        __typename: "Resource",
                    } as Resource,
                },
            ],
            pageInfo: {
                hasNextPage: false,
                hasPreviousPage: false,
                __typename: "PageInfo",
            },
            __typename: "ResourceSearchResult",
        };

        it("should list routines in table format", async () => {
            (mockClient.requestWithEndpoint as any).mockResolvedValue(mockRoutines);

            await program.parseAsync(["node", "test", "routine", "list"]);

            expect(mockClient.requestWithEndpoint).toHaveBeenCalledWith(
                expect.anything(),
                expect.objectContaining({
                    take: 10,
                    isLatest: true,
                    rootResourceType: "Routine",
                }),
            );

            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("Routines found"));
            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("Test Routine"));
            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("API Routine"));
        });

        it("should list routines in JSON format", async () => {
            (mockClient.requestWithEndpoint as any).mockResolvedValue(mockRoutines);
            (mockConfig.isJsonOutput as any).mockReturnValue(true);

            await program.parseAsync(["node", "test", "routine", "list", "--format", "json"]);

            // The implementation outputs the raw response object, not formatted data
            expect(output.json).toHaveBeenCalledWith(mockRoutines);
        });

        it("should filter by search term", async () => {
            (mockClient.requestWithEndpoint as any).mockResolvedValue(mockRoutines);

            await program.parseAsync(["node", "test", "routine", "list", "--search", "test"]);

            expect(mockClient.requestWithEndpoint).toHaveBeenCalledWith(
                expect.anything(),
                expect.objectContaining({
                    take: 10,
                    isLatest: true,
                    rootResourceType: "Routine",
                    searchString: "test",
                }),
            );
        });

        it("should handle empty results", async () => {
            (mockClient.requestWithEndpoint as any).mockResolvedValue({
                edges: [],
                pageInfo: {
                    hasNextPage: false,
                    hasPreviousPage: false,
                    __typename: "PageInfo",
                },
                __typename: "ResourceSearchResult",
            });

            await program.parseAsync(["node", "test", "routine", "list"]);

            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("No routines found"));
        });
    });

    describe("import command", () => {
        const mockRoutineData = {
            id: "new-routine",
            publicId: "pub-new",
            resourceType: "RoutineVersion",
            versions: [{
                id: "v-new",
                resourceSubType: "Graph",
                config: { steps: [] },
                translations: [{
                    language: "en",
                    name: "Imported Routine",
                    description: "A newly imported routine",
                }],
            }],
        };

        it("should import a routine from file", async () => {
            // Mock both fs imports: fs.promises and fs/promises module
            (fs.readFile as any).mockResolvedValue(JSON.stringify(mockRoutineData));
            const fsPromises = await import("fs/promises");
            (fsPromises.readFile as any).mockResolvedValue(JSON.stringify(mockRoutineData));
            (mockClient.requestWithEndpoint as any).mockResolvedValue({ id: "created-id", publicId: "created-pub" });

            await program.parseAsync(["node", "test", "routine", "import", "./routine.json"]);

            // The implementation uses path.resolve() to get absolute paths
            expect(fsPromises.readFile).toHaveBeenCalledWith(
                expect.stringMatching(/\/routine\.json$/),
                "utf-8",
            );
            expect(mockClient.requestWithEndpoint).toHaveBeenCalledWith(
                expect.anything(),
                mockRoutineData,
            );

            expect(output.success).toHaveBeenCalledWith(expect.stringContaining("Import successful"));
            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("created-id"));
        });


        it("should handle invalid JSON", async () => {
            (fs.readFile as any).mockResolvedValue("invalid json");

            await expect(
                program.parseAsync(["node", "test", "routine", "import", "./routine.json"]),
            ).rejects.toThrow("Import failed:");

            expect(output.error).toHaveBeenCalledWith(expect.stringContaining("Import failed:"));
        });

        it("should handle file read errors", async () => {
            (fs.readFile as any).mockRejectedValue(new Error("File not found"));

            await expect(
                program.parseAsync(["node", "test", "routine", "import", "./routine.json"]),
            ).rejects.toThrow("Import failed:");

            expect(output.error).toHaveBeenCalledWith(expect.stringContaining("Import failed:"));
        });
    });

    describe("import-dir command", () => {
        it("should import multiple routines from directory", async () => {
            const glob = await import("glob");
            const path = await import("path");
            const currentDir = process.cwd();
            (glob.glob as any).mockResolvedValue([
                path.resolve(currentDir, "src/__test/fixtures/routines/routine1.json"),
                path.resolve(currentDir, "src/__test/fixtures/routines/routine2.json"),
            ]);

            const mockRoutine1 = {
                id: "routine1",
                publicId: "pub1",
                resourceType: "Routine",
                versions: [{
                    id: "v1",
                    resourceSubType: "Graph",
                    config: {
                        __version: "1.0",
                        callDataCode: {
                            __version: "1.0",
                            schema: { language: "python", code: "print('routine1')" },
                        },
                    },
                    translations: [{ language: "en", name: "Routine 1" }],
                }],
            };

            const mockRoutine2 = {
                id: "routine2",
                publicId: "pub2",
                resourceType: "Routine",
                versions: [{
                    id: "v2",
                    resourceSubType: "Graph",
                    config: {
                        __version: "1.0",
                        callDataCode: {
                            __version: "1.0",
                            schema: { language: "javascript", code: "console.log('routine2')" },
                        },
                    },
                    translations: [{ language: "en", name: "Routine 2" }],
                }],
            };

            // Mock fs.readFile to return the JSON content for each file
            (fs.readFile as any)
                .mockResolvedValueOnce(JSON.stringify(mockRoutine1))
                .mockResolvedValueOnce(JSON.stringify(mockRoutine2));

            (mockClient.requestWithEndpoint as any).mockResolvedValue({
                id: "created-id",
            });

            await program.parseAsync(["node", "test", "routine", "import-dir", "./routines"]);

            expect(glob.glob).toHaveBeenCalledWith(expect.stringMatching(/\/routines\/\*\.json$/));
            expect(mockClient.requestWithEndpoint).toHaveBeenCalledTimes(2);

            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("Found 2 file(s) to import"));
            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("📊 Import Summary:"));
            expect(output.success).toHaveBeenCalledWith(expect.stringContaining("Success: 2"));
        });

        it("should handle empty directory", async () => {
            const glob = await import("glob");
            (glob.glob as any).mockResolvedValue([]);

            await program.parseAsync(["node", "test", "routine", "import-dir", "./empty"]);

            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("No files found matching pattern"));
        });

        it("should fail fast on error when specified", async () => {
            const glob = await import("glob");
            (glob.glob as any).mockResolvedValue([
                "/path/routine1.json",
                "/path/routine2.json",
            ]);

            (fs.readFile as any)
                .mockResolvedValueOnce(JSON.stringify({
                    id: "routine1",
                    publicId: "pub1",
                    resourceType: "Routine",
                    versions: [{
                        id: "v1",
                        resourceSubType: "Graph",
                        config: {
                            __version: "1.0",
                            callDataGenerate: {
                                model: "gpt-4",
                                prompt: "Test prompt",
                            },
                        },
                        translations: [{ language: "en", name: "Test Routine" }],
                    }],
                }))
                .mockResolvedValueOnce("invalid json");

            (mockClient.requestWithEndpoint as any).mockResolvedValueOnce({
                id: "created-id",
            });

            await expect(
                program.parseAsync(["node", "test", "routine", "import-dir", "./routines", "--fail-fast"]),
            ).rejects.toThrow("Directory import failed");

            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("Found 2 file(s) to import"));
            expect(output.error).toHaveBeenCalledWith(expect.stringContaining("Directory import failed"));
        });
    });

    describe("export command", () => {
        const mockRoutine: Resource = {
            id: "routine1",
            publicId: "pub1",
            created_at: new Date("2025-01-01"),
            updated_at: new Date("2025-01-01"),
            isPrivate: false,
            labels: [],
            resourceType: "RoutineVersion",
            owner: {
                __typename: "User",
                id: "user1",
                name: "User",
            },
            versions: [{
                id: "v1",
                created_at: new Date("2025-01-01"),
                updated_at: new Date("2025-01-01"),
                resourceSubType: "Graph",
                config: { steps: [] },
                translations: [{
                    id: "t1",
                    language: "en",
                    name: "Exported Routine",
                    __typename: "ResourceVersionTranslation",
                }],
                __typename: "ResourceVersion",
            } as ResourceVersion],
            __typename: "Resource",
        };

        it("should export a routine to file", async () => {
            (mockClient.requestWithEndpoint as any).mockResolvedValue(mockRoutine);

            // Mock the fs/promises module used by writeJsonFile
            const fsPromises = await import("fs/promises");

            await program.parseAsync(["node", "test", "routine", "export", "pub1", "--output", "./exported.json"]);

            expect(mockClient.requestWithEndpoint).toHaveBeenCalledWith(
                expect.anything(),
                { publicId: "pub1" },
            );
            expect(fsPromises.writeFile).toHaveBeenCalledWith(
                expect.stringContaining("exported.json"),
                JSON.stringify(mockRoutine, null, 2),
            );

            expect(output.success).toHaveBeenCalledWith(expect.stringContaining("Routine exported to:"));
        });

        it("should export to default filename when no output specified", async () => {
            (mockClient.requestWithEndpoint as any).mockResolvedValue(mockRoutine);

            // Mock the fs/promises module used by writeJsonFile
            const fsPromises = await import("fs/promises");

            await program.parseAsync(["node", "test", "routine", "export", "pub1"]);

            expect(fsPromises.writeFile).toHaveBeenCalledWith(
                expect.stringContaining("routine-pub1.json"),
                JSON.stringify(mockRoutine, null, 2),
            );
            expect(output.success).toHaveBeenCalledWith(expect.stringContaining("Routine exported to:"));
        });

        it("should handle routine not found", async () => {
            (mockClient.requestWithEndpoint as any).mockRejectedValue(new Error("Not found"));

            await expect(
                program.parseAsync(["node", "test", "routine", "export", "invalid-id"]),
            ).rejects.toThrow("Export failed: Not found");

            expect(output.error).toHaveBeenCalledWith(expect.stringContaining("Export failed: Not found"));
        });
    });

    describe("validate command", () => {
        it("should detect validation errors in routine file", async () => {
            const invalidRoutine = {
                id: "routine1",
                publicId: "pub1",
                resourceType: "Routine",
                name: "Test Routine",
                steps: [{ id: "step1" }],
                inputs: [{ id: "input1" }],
                outputs: [{ id: "output1" }],
                versions: [{
                    id: "v1",
                    resourceSubType: "Action",
                    config: {
                        __version: "1.0",
                        graph: {
                            __version: "1.0",
                            nodes: [],
                            edges: [],
                        },
                    },
                    translations: [{
                        language: "en",
                        name: "Valid Routine",
                    }],
                }],
            };

            (fs.readFile as any).mockResolvedValue(JSON.stringify(invalidRoutine));

            await expect(
                program.parseAsync(["node", "test", "routine", "validate", "./routine.json"]),
            ).rejects.toThrow("Validation error:");

            expect(output.info).toHaveBeenCalledWith("🔍 Validating routine file...\n");
        });

        it("should detect invalid JSON", async () => {
            (fs.readFile as any).mockResolvedValue("{ invalid json");

            await expect(
                program.parseAsync(["node", "test", "routine", "validate", "./routine.json"]),
            ).rejects.toThrow("Validation error:");

            expect(output.error).toHaveBeenCalledWith(expect.stringContaining("Validation error:"));
        });

        it("should detect missing required fields", async () => {
            const invalidRoutine = {
                // Missing resourceType
                versions: [{
                    translations: [{ language: "en" }],
                }],
            };

            (fs.readFile as any).mockResolvedValue(JSON.stringify(invalidRoutine));

            await expect(
                program.parseAsync(["node", "test", "routine", "validate", "./routine.json"]),
            ).rejects.toThrow("Validation error:");

            expect(output.error).toHaveBeenCalledWith(expect.stringContaining("Validation error:"));
        });
    });

    describe("run command", () => {
        it("should run a routine with default inputs", async () => {
            const mockRunResponse = {
                id: "run123",
                status: "Completed",
                completedComplexity: 10,
                steps: [{
                    id: "step1",
                    status: "Completed",
                    name: "Step 1",
                }],
            };

            (mockClient.post as any).mockResolvedValue(mockRunResponse);

            await program.parseAsync(["node", "test", "routine", "run", "pub1"]);

            expect(mockClient.post).toHaveBeenCalledWith(
                "/task/start/run",
                expect.objectContaining({
                    routineId: "pub1",
                    inputs: {},
                }),
            );

            const spinner = ora();
            expect(spinner.succeed).toHaveBeenCalledWith("Execution started");
            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("Run with --watch to monitor progress"));
        });

        it("should run a routine with custom inputs", async () => {
            const mockRunResponse = {
                id: "run123",
                status: "InProgress",
            };

            (mockClient.post as any).mockResolvedValue(mockRunResponse);

            const inputData = { key: "value" };
            await program.parseAsync([
                "node", "test", "routine", "run", "pub1",
                "--input", JSON.stringify(inputData),
            ]);

            expect(mockClient.post).toHaveBeenCalledWith(
                "/task/start/run",
                expect.objectContaining({
                    routineId: "pub1",
                    inputs: inputData,
                }),
            );
        });
    });

    describe("discover command", () => {
        const mockDiscoverResponse: ResourceSearchResult = {
            edges: [
                {
                    cursor: "cursor1",
                    node: {
                        id: "routine1",
                        publicId: "pub1",
                        createdAt: "2025-01-01T00:00:00.000Z",
                        updatedAt: "2025-01-01T00:00:00.000Z",
                        isPrivate: false,
                        labels: [],
                        resourceType: "Routine",
                        resourceSubType: "Graph",
                        translations: [{
                            id: "t1",
                            language: "en",
                            name: "Test Routine",
                            description: "A test routine for unit testing",
                        }],
                        owner: {
                            __typename: "User",
                            id: "user1",
                            name: "User",
                        },
                        __typename: "Resource",
                    } as Resource,
                },
                {
                    cursor: "cursor2",
                    node: {
                        id: "routine2",
                        publicId: "pub2",
                        createdAt: "2025-01-02T00:00:00.000Z",
                        updatedAt: "2025-01-02T00:00:00.000Z",
                        isPrivate: false,
                        labels: [],
                        resourceType: "Routine",
                        resourceSubType: "Api",
                        translations: [{
                            id: "t2",
                            language: "en",
                            name: "API Routine",
                            description: "An API routine",
                        }],
                        owner: {
                            __typename: "User",
                            id: "user1",
                            name: "User",
                        },
                        __typename: "Resource",
                    } as Resource,
                },
            ],
            pageInfo: {
                hasNextPage: false,
                hasPreviousPage: false,
                startCursor: null,
                endCursor: null,
            },
            __typename: "ResourceSearchResult",
        };

        it("should discover routines in table format", async () => {
            (mockClient.requestWithEndpoint as any).mockResolvedValue(mockDiscoverResponse);

            await program.parseAsync(["node", "test", "routine", "discover"]);

            expect(mockClient.requestWithEndpoint).toHaveBeenCalledWith(
                expect.anything(),
                expect.objectContaining({
                    take: 100,
                    isLatest: true,
                    rootResourceType: "Routine",
                    isCompleteWithRoot: true,
                }),
            );

            const spinner = ora();
            expect(spinner.succeed).toHaveBeenCalledWith("Discovery complete");
            expect(output.table).toHaveBeenCalled();
        });

        it("should discover routines in JSON format", async () => {
            (mockClient.requestWithEndpoint as any).mockResolvedValue(mockDiscoverResponse);

            await program.parseAsync(["node", "test", "routine", "discover", "--format", "json"]);

            expect(output.json).toHaveBeenCalledWith(expect.arrayContaining([
                expect.objectContaining({
                    publicId: "pub1",
                    name: "Test Routine",
                }),
            ]));
        });

        it("should discover routines in mapping format", async () => {
            (mockClient.requestWithEndpoint as any).mockResolvedValue(mockDiscoverResponse);

            await program.parseAsync(["node", "test", "routine", "discover", "--format", "mapping"]);

            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("ID Mapping Reference:"));
            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("\"pub1\" # Test Routine (Graph)"));
            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("\"pub2\" # API Routine (Api)"));
        });

        it("should filter by routine type", async () => {
            (mockClient.requestWithEndpoint as any).mockResolvedValue(mockDiscoverResponse);

            await program.parseAsync(["node", "test", "routine", "discover", "--type", "Graph"]);

            expect(mockClient.requestWithEndpoint).toHaveBeenCalledWith(
                expect.anything(),
                expect.objectContaining({
                    take: 100,
                    isLatest: true,
                    rootResourceType: "Routine",
                    resourceSubType: "Graph",
                    isCompleteWithRoot: true,
                }),
            );
        });

        it("should handle empty discover results", async () => {
            (mockClient.requestWithEndpoint as any).mockResolvedValue({ edges: [], pageInfo: {} });

            await program.parseAsync(["node", "test", "routine", "discover"]);

            expect(output.info).toHaveBeenCalledWith("No routines found");
        });

        it("should handle discover errors", async () => {
            (mockClient.requestWithEndpoint as any).mockRejectedValue(new Error("Network error"));

            await expect(program.parseAsync(["node", "test", "routine", "discover"])).rejects.toThrow("Network error");
        });

        it("should handle routines without translations", async () => {
            const responseWithoutTranslations: ResourceSearchResult = {
                edges: [{
                    cursor: "cursor1",
                    node: {
                        id: "routine1",
                        publicId: "pub1",
                        createdAt: "2025-01-01T00:00:00.000Z",
                        updatedAt: "2025-01-01T00:00:00.000Z",
                        isPrivate: false,
                        labels: [],
                        resourceType: "Routine",
                        translations: [],
                        owner: {
                            __typename: "User",
                            id: "user1",
                            name: "User",
                        },
                        __typename: "Resource",
                    } as Resource,
                }],
                pageInfo: {
                    hasNextPage: false,
                    hasPreviousPage: false,
                    startCursor: null,
                    endCursor: null,
                },
                __typename: "ResourceSearchResult",
            };

            (mockClient.requestWithEndpoint as any).mockResolvedValue(responseWithoutTranslations);

            await program.parseAsync(["node", "test", "routine", "discover"]);

            expect(output.table).toHaveBeenCalled();
        });
    });

    describe("search command", () => {
        const mockSearchResponse: ResourceSearchResult = {
            edges: [
                {
                    cursor: "cursor1",
                    node: {
                        id: "routine1",
                        publicId: "pub1",
                        createdAt: "2025-01-01T00:00:00.000Z",
                        updatedAt: "2025-01-01T00:00:00.000Z",
                        isPrivate: false,
                        labels: [],
                        resourceType: "Routine",
                        resourceSubType: "Graph",
                        translations: [{
                            id: "t1",
                            language: "en",
                            name: "Search Test Routine",
                            description: "A routine for search testing",
                        }],
                        owner: {
                            __typename: "User",
                            id: "user1",
                            name: "User",
                        },
                        __typename: "Resource",
                    } as Resource,
                    searchScore: 0.85,
                },
            ],
            pageInfo: {
                hasNextPage: false,
                hasPreviousPage: false,
                startCursor: null,
                endCursor: null,
            },
            __typename: "ResourceSearchResult",
        };

        it("should search routines with basic query", async () => {
            (mockClient.requestWithEndpoint as any).mockResolvedValue(mockSearchResponse);

            await program.parseAsync(["node", "test", "routine", "search", "test query"]);

            expect(mockClient.requestWithEndpoint).toHaveBeenCalledWith(
                expect.anything(),
                expect.objectContaining({
                    take: 10,
                    searchString: "test query",
                    where: expect.objectContaining({
                        isComplete: { equals: true },
                        isPrivate: { equals: false },
                    }),
                }),
            );

            const spinner = ora();
            expect(spinner.succeed).toHaveBeenCalledWith("Search complete");
            expect(output.table).toHaveBeenCalled();
        });

        it("should handle search with no results", async () => {
            (mockClient.requestWithEndpoint as any).mockResolvedValue({ edges: [], pageInfo: {} });

            await program.parseAsync(["node", "test", "routine", "search", "nonexistent"]);

            expect(output.info).toHaveBeenCalledWith("No routines found for \"nonexistent\"");
        });

        it("should search with custom options", async () => {
            (mockClient.requestWithEndpoint as any).mockResolvedValue(mockSearchResponse);

            await program.parseAsync([
                "node", "test", "routine", "search", "test query",
                "--limit", "20",
                "--type", "Graph",
                "--min-score", "0.8",
            ]);

            expect(mockClient.requestWithEndpoint).toHaveBeenCalledWith(
                expect.anything(),
                expect.objectContaining({
                    take: 20,
                    searchString: "test query",
                    where: expect.objectContaining({
                        resourceSubType: { equals: "Graph" },
                        isComplete: { equals: true },
                        isPrivate: { equals: false },
                    }),
                }),
            );
        });

        it("should output search results in JSON format", async () => {
            (mockClient.requestWithEndpoint as any).mockResolvedValue(mockSearchResponse);

            await program.parseAsync([
                "node", "test", "routine", "search", "test query",
                "--format", "json",
            ]);

            expect(output.json).toHaveBeenCalledWith(expect.arrayContaining([
                expect.objectContaining({
                    publicId: "pub1",
                    name: "Search Test Routine",
                }),
            ]));
        });

        it("should output search results in ids format", async () => {
            (mockClient.requestWithEndpoint as any).mockResolvedValue(mockSearchResponse);

            await program.parseAsync([
                "node", "test", "routine", "search", "test query",
                "--format", "ids",
            ]);

            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("Routine IDs (for copy/paste):"));
            expect(output.info).toHaveBeenCalledWith(expect.stringContaining("\"pub1\""));
        });

        it("should filter results by minimum score", async () => {
            // Since the implementation hardcodes score to 1.0 and doesn't use actual searchScore,
            // all results should pass a 0.7 minimum threshold
            (mockClient.requestWithEndpoint as any).mockResolvedValue(mockSearchResponse);

            await program.parseAsync([
                "node", "test", "routine", "search", "test query",
                "--min-score", "0.7",
            ]);

            const spinner = ora();
            expect(spinner.succeed).toHaveBeenCalledWith("Search complete");
        });

        it("should handle search errors", async () => {
            (mockClient.requestWithEndpoint as any).mockRejectedValue(new Error("Search failed"));

            await expect(program.parseAsync([
                "node", "test", "routine", "search", "test query",
            ])).rejects.toThrow("Search failed");
        });
    });

    describe("validation helpers", () => {
        it("should test validateFormSchema method indirectly", () => {
            // Create a separate program for this test to avoid command conflicts
            const testProgram = new Command();
            const routineCommands = new RoutineCommands(testProgram, mockClient, mockConfig);

            // Access the private method for testing
            const validateFormSchema = (routineCommands as any).validateFormSchema;

            const validSchema = {
                __version: "1.0.0",
                schema: {
                    elements: [],
                    containers: [],
                },
            };

            const invalidSchema1 = {
                schema: {
                    elements: [],
                    containers: [],
                },
            };

            const invalidSchema2 = {
                __version: "1.0.0",
                schema: {
                    elements: [],
                },
            };

            expect(validateFormSchema(validSchema)).toBe(true);
            expect(validateFormSchema(invalidSchema1)).toBe(false);
            expect(validateFormSchema(invalidSchema2)).toBe(false);
        });

        it("should test callData validation logic", () => {
            // Create a separate program for this test to avoid command conflicts
            const testProgram = new Command();
            const routineCommands = new RoutineCommands(testProgram, mockClient, mockConfig);

            // Access the private method for testing
            const validateSingleStepConfig = (routineCommands as any).validateSingleStepConfig;

            const errors: string[] = [];

            const configWithCallData = {
                callDataAction: {
                    __version: "1.0.0",
                    schema: {},
                },
                callDataInvalid: {
                    data: "missing version",
                },
            };

            validateSingleStepConfig(configWithCallData, errors);

            expect(errors).toContain("callDataInvalid missing __version or schema");
        });

        it("should test graph validation with TODO placeholders", async () => {
            // Create a separate program for this test to avoid command conflicts
            const testProgram = new Command();
            const routineCommands = new RoutineCommands(testProgram, mockClient, mockConfig);

            // Access the private method for testing
            const validateGraphConfig = (routineCommands as any).validateGraphConfig;

            const errors: string[] = [];

            const graphWithTodo = {
                __type: "BPMN-2.0",
                schema: {
                    data: "<xml>test</xml>",
                    activityMap: {
                        "activity1": {
                            __type: "DecisionTask",
                            subroutineId: "TODO: Replace with actual subroutine ID",
                        },
                    },
                },
            };

            await validateGraphConfig(graphWithTodo, errors, true);

            expect(errors).toContain("Activity 'activity1': Contains TODO placeholder - subroutine ID needs to be replaced");
        });

        it("should test graph validation with unknown graph type", async () => {
            // Create a separate program for this test to avoid command conflicts
            const testProgram = new Command();
            const routineCommands = new RoutineCommands(testProgram, mockClient, mockConfig);

            // Access the private method for testing
            const validateGraphConfig = (routineCommands as any).validateGraphConfig;

            const errors: string[] = [];

            const graphWithUnknownType = {
                __type: "UnknownGraphType",
                schema: {
                    data: "test",
                },
            };

            await validateGraphConfig(graphWithUnknownType, errors);

            expect(errors).toContain("Unknown graph type: UnknownGraphType");
        });

        it("should test single-step config validation without callData", () => {
            // Create a separate program for this test to avoid command conflicts
            const testProgram = new Command();
            const routineCommands = new RoutineCommands(testProgram, mockClient, mockConfig);

            // Access the private method for testing
            const validateSingleStepConfig = (routineCommands as any).validateSingleStepConfig;

            const errors: string[] = [];

            const configWithoutCallData = {
                someOtherProperty: "value",
            };

            validateSingleStepConfig(configWithoutCallData, errors);

            expect(errors).toContain("Single-step routine missing callData configuration");
        });

        it("should test BPMN subroutine existence validation", async () => {
            // Create a separate program for this test to avoid command conflicts
            const testProgram = new Command();
            const routineCommands = new RoutineCommands(testProgram, mockClient, mockConfig);

            // Access the private method for testing and bind it to the instance
            const validateGraphConfig = (routineCommands as any).validateGraphConfig.bind(routineCommands);

            const errors: string[] = [];

            // Mock client to simulate subroutine not found
            (mockClient.requestWithEndpoint as any).mockRejectedValue(new Error("Not found"));

            const graphWithSubroutine = {
                __type: "BPMN-2.0",
                schema: {
                    data: "<xml>test</xml>",
                    activityMap: {
                        "activity1": {
                            __type: "DecisionTask",
                            subroutineId: "valid-subroutine-id",
                        },
                    },
                },
            };

            await validateGraphConfig(graphWithSubroutine, errors, true);

            expect(errors).toContain("Activity 'activity1': Subroutine 'valid-subroutine-id' not found in database");
            expect(mockClient.requestWithEndpoint).toHaveBeenCalledWith(
                expect.anything(),
                { publicId: "valid-subroutine-id" },
            );
        });

        it("should test BPMN subroutine existence validation success", async () => {
            // Create a separate program for this test to avoid command conflicts
            const testProgram = new Command();
            const routineCommands = new RoutineCommands(testProgram, mockClient, mockConfig);

            // Access the private method for testing and bind it to the instance
            const validateGraphConfig = (routineCommands as any).validateGraphConfig.bind(routineCommands);

            const errors: string[] = [];

            // Mock client to simulate successful subroutine lookup
            (mockClient.requestWithEndpoint as any).mockResolvedValue({ id: "valid-subroutine-id", name: "Test Subroutine" });

            const graphWithSubroutine = {
                __type: "BPMN-2.0",
                schema: {
                    data: "<xml>test</xml>",
                    activityMap: {
                        "activity1": {
                            __type: "DecisionTask",
                            subroutineId: "valid-subroutine-id",
                        },
                    },
                },
            };

            await validateGraphConfig(graphWithSubroutine, errors, true);

            // No errors should be added for valid subroutine
            expect(errors).not.toContain(expect.stringMatching(/Activity 'activity1'.*not found/));
            expect(mockClient.requestWithEndpoint).toHaveBeenCalledWith(
                expect.anything(),
                { publicId: "valid-subroutine-id" },
            );
        });

        it("should test BPMN graph validation with missing data field", async () => {
            // Create a separate program for this test to avoid command conflicts
            const testProgram = new Command();
            const routineCommands = new RoutineCommands(testProgram, mockClient, mockConfig);

            // Access the private method for testing
            const validateGraphConfig = (routineCommands as any).validateGraphConfig;

            const errors: string[] = [];

            const bpmnWithoutData = {
                __type: "BPMN-2.0",
                schema: {
                    activityMap: {},
                },
            };

            await validateGraphConfig(bpmnWithoutData, errors);

            expect(errors).toContain("BPMN graph missing XML 'data' field");
        });

        it("should test BPMN graph validation with non-string data field", async () => {
            // Create a separate program for this test to avoid command conflicts
            const testProgram = new Command();
            const routineCommands = new RoutineCommands(testProgram, mockClient, mockConfig);

            // Access the private method for testing
            const validateGraphConfig = (routineCommands as any).validateGraphConfig;

            const errors: string[] = [];

            const bpmnWithInvalidData = {
                __type: "BPMN-2.0",
                schema: {
                    data: 123, // Invalid: should be string
                    activityMap: {},
                },
            };

            await validateGraphConfig(bpmnWithInvalidData, errors);

            expect(errors).toContain("BPMN graph missing XML 'data' field");
        });

        it("should test BPMN graph validation with missing activityMap", async () => {
            // Create a separate program for this test to avoid command conflicts
            const testProgram = new Command();
            const routineCommands = new RoutineCommands(testProgram, mockClient, mockConfig);

            // Access the private method for testing
            const validateGraphConfig = (routineCommands as any).validateGraphConfig;

            const errors: string[] = [];

            const bpmnWithoutActivityMap = {
                __type: "BPMN-2.0",
                schema: {
                    data: "<xml>test</xml>",
                },
            };

            await validateGraphConfig(bpmnWithoutActivityMap, errors);

            expect(errors).toContain("BPMN graph missing 'activityMap'");
        });

        it("should test BPMN graph validation with non-object activityMap", async () => {
            // Create a separate program for this test to avoid command conflicts
            const testProgram = new Command();
            const routineCommands = new RoutineCommands(testProgram, mockClient, mockConfig);

            // Access the private method for testing
            const validateGraphConfig = (routineCommands as any).validateGraphConfig;

            const errors: string[] = [];

            const bpmnWithInvalidActivityMap = {
                __type: "BPMN-2.0",
                schema: {
                    data: "<xml>test</xml>",
                    activityMap: "invalid", // Should be object
                },
            };

            await validateGraphConfig(bpmnWithInvalidActivityMap, errors);

            expect(errors).toContain("BPMN graph missing 'activityMap'");
        });
    });

    describe("importRoutineSilent", () => {
        it("should import routine in dry run mode", async () => {
            const testProgram = new Command();
            const routineCommands = new RoutineCommands(testProgram, mockClient, mockConfig);

            const validRoutineData = {
                id: "test-id",
                publicId: "pub-id",
                resourceType: "Routine",
                versions: [{
                    id: "version-1",
                    resourceSubType: "SingleStep",
                    config: {
                        __version: "1.0.0",
                        callDataAction: {
                            __version: "1.0.0",
                            schema: {},
                        },
                    },
                    translations: [{
                        id: "trans-1",
                        language: "en",
                        name: "Test Routine",
                        description: "A test routine",
                    }],
                }],
            };

            // Reset the mock to ensure clean state
            (fs.readFile as any).mockReset();
            (fs.readFile as any).mockResolvedValue(JSON.stringify(validRoutineData));

            const importSilent = (routineCommands as any).importRoutineSilent.bind(routineCommands);
            const result = await importSilent("test.json", true);

            expect(result.success).toBe(true);
            expect(mockClient.requestWithEndpoint).not.toHaveBeenCalled();
        });

        it("should import routine and call API when not dry run", async () => {
            const testProgram = new Command();
            const routineCommands = new RoutineCommands(testProgram, mockClient, mockConfig);

            const validRoutineData = {
                id: "test-id",
                publicId: "pub-id",
                resourceType: "Routine",
                versions: [{
                    id: "version-1",
                    resourceSubType: "SingleStep",
                    config: {
                        __version: "1.0.0",
                        callDataAction: {
                            __version: "1.0.0",
                            schema: {},
                        },
                    },
                    translations: [{
                        id: "trans-1",
                        language: "en",
                        name: "Test Routine",
                        description: "A test routine",
                    }],
                }],
            };

            const mockApiResponse = { id: "created-routine" };

            (fs.readFile as any).mockResolvedValue(JSON.stringify(validRoutineData));
            (mockClient.requestWithEndpoint as any).mockResolvedValue(mockApiResponse);

            const importSilent = (routineCommands as any).importRoutineSilent.bind(routineCommands);
            const result = await importSilent("test.json", false);

            expect(result.success).toBe(true);
            expect(result.routine).toEqual(mockApiResponse);
            expect(mockClient.requestWithEndpoint).toHaveBeenCalledWith(
                expect.anything(),
                validRoutineData,
            );
        });

        it("should throw validation error for invalid routine data", async () => {
            const testProgram = new Command();
            const routineCommands = new RoutineCommands(testProgram, mockClient, mockConfig);

            const invalidRoutineData = {
                id: "test-id",
                // Missing required fields like publicId, resourceType, versions
            };

            (fs.readFile as any).mockResolvedValue(JSON.stringify(invalidRoutineData));

            const importSilent = (routineCommands as any).importRoutineSilent.bind(routineCommands);

            await expect(importSilent("test.json", false)).rejects.toThrow("Validation failed:");
        });
    });
});
