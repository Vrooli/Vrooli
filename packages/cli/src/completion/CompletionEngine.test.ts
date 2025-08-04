import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ApiClient } from "../utils/client.js";
import type { ConfigManager } from "../utils/config.js";
import { CompletionEngine } from "./CompletionEngine.js";
import type { CompletionContext, CompletionResult } from "./types.js";

// Mock the providers and cache
const mockStaticProvider = {
    canHandle: vi.fn(),
    getCompletions: vi.fn(),
};

const mockDynamicProvider = {
    canHandle: vi.fn(),
    getCompletions: vi.fn(),
};

const mockFileProvider = {
    canHandle: vi.fn(),
    getCompletions: vi.fn(),
};

const mockCache = {
    get: vi.fn(),
    set: vi.fn(),
};

vi.mock("./providers/StaticProvider.js", () => ({
    StaticProvider: vi.fn(() => mockStaticProvider),
}));

vi.mock("./providers/DynamicProvider.js", () => ({
    DynamicProvider: vi.fn(() => mockDynamicProvider),
}));

vi.mock("./providers/FileProvider.js", () => ({
    FileProvider: vi.fn(() => mockFileProvider),
}));

vi.mock("./cache/CompletionCache.js", () => ({
    CompletionCache: vi.fn(() => mockCache),
}));

describe("CompletionEngine", () => {
    let completionEngine: CompletionEngine;
    let mockConfig: ConfigManager;
    let mockClient: ApiClient;

    beforeEach(() => {
        vi.clearAllMocks();

        mockConfig = {} as ConfigManager;
        mockClient = {} as ApiClient;

        completionEngine = new CompletionEngine(mockConfig, mockClient);
    });

    afterEach(() => {
        vi.clearAllMocks();
    });


    describe("getCompletions", () => {
        it("should return empty array when no provider can handle the context", async () => {
            vi.mocked(mockStaticProvider.canHandle).mockReturnValue(false);
            vi.mocked(mockFileProvider.canHandle).mockReturnValue(false);
            vi.mocked(mockDynamicProvider.canHandle).mockReturnValue(false);

            const result = await completionEngine.getCompletions(["vrooli", "routine"]);

            expect(result).toEqual([]);
        });

        it("should use static provider when it can handle the context", async () => {
            const mockResults: CompletionResult[] = [{ value: "list", description: "List routines", type: "subcommand" }];
            vi.mocked(mockStaticProvider.canHandle).mockReturnValue(true);
            vi.mocked(mockStaticProvider.getCompletions).mockResolvedValue(mockResults);

            const result = await completionEngine.getCompletions(["vrooli", "routine"]);

            expect(mockStaticProvider.canHandle).toHaveBeenCalled();
            expect(mockStaticProvider.getCompletions).toHaveBeenCalled();
            expect(result).toEqual(mockResults);
        });

        it("should use file provider when it can handle the context", async () => {
            const mockResults: CompletionResult[] = [{ value: "file.json", description: "JSON file", type: "file" }];
            vi.mocked(mockStaticProvider.canHandle).mockReturnValue(false);
            vi.mocked(mockFileProvider.canHandle).mockReturnValue(true);
            vi.mocked(mockFileProvider.getCompletions).mockResolvedValue(mockResults);

            const result = await completionEngine.getCompletions(["vrooli", "routine", "import"]);

            expect(mockFileProvider.canHandle).toHaveBeenCalled();
            expect(mockFileProvider.getCompletions).toHaveBeenCalled();
            expect(result).toEqual(mockResults);
        });

        it("should use cached results for dynamic provider", async () => {
            const mockResults: CompletionResult[] = [{ value: "cached-routine", description: "Cached routine", type: "resource" }];
            vi.mocked(mockStaticProvider.canHandle).mockReturnValue(false);
            vi.mocked(mockFileProvider.canHandle).mockReturnValue(false);
            vi.mocked(mockDynamicProvider.canHandle).mockReturnValue(true);
            vi.mocked(mockCache.get).mockResolvedValue(mockResults);

            const result = await completionEngine.getCompletions(["vrooli", "routine", "get", "partial"]);

            expect(mockDynamicProvider.canHandle).toHaveBeenCalled();
            expect(mockCache.get).toHaveBeenCalled();
            expect(mockDynamicProvider.getCompletions).not.toHaveBeenCalled();
            expect(result).toEqual(mockResults);
        });

        it("should fetch and cache results for dynamic provider when not cached", async () => {
            const mockResults: CompletionResult[] = [{ value: "new-routine", description: "New routine", type: "resource" }];
            vi.mocked(mockStaticProvider.canHandle).mockReturnValue(false);
            vi.mocked(mockFileProvider.canHandle).mockReturnValue(false);
            vi.mocked(mockDynamicProvider.canHandle).mockReturnValue(true);
            vi.mocked(mockCache.get).mockResolvedValue(null);
            vi.mocked(mockDynamicProvider.getCompletions).mockResolvedValue(mockResults);

            const result = await completionEngine.getCompletions(["vrooli", "routine", "get", "partial"]);

            expect(mockDynamicProvider.canHandle).toHaveBeenCalled();
            expect(mockCache.get).toHaveBeenCalled();
            expect(mockDynamicProvider.getCompletions).toHaveBeenCalled();
            expect(mockCache.set).toHaveBeenCalledWith(
                expect.stringContaining("completion:"),
                mockResults,
                300,
            );
            expect(result).toEqual(mockResults);
        });
    });

    describe("parseContext", () => {
        it("should parse command context correctly", async () => {
            vi.mocked(mockStaticProvider.canHandle).mockImplementation((context: CompletionContext) => {
                expect(context.type).toBe("command");
                expect(context.partial).toBe("rout");
                expect(context.args).toEqual(["rout"]);
                return true;
            });
            vi.mocked(mockStaticProvider.getCompletions).mockResolvedValue([]);

            await completionEngine.getCompletions(["vrooli", "rout"]);
        });

        it("should parse subcommand context correctly", async () => {
            vi.mocked(mockStaticProvider.canHandle).mockImplementation((context: CompletionContext) => {
                expect(context.type).toBe("subcommand");
                expect(context.command).toBe("routine");
                expect(context.partial).toBe("lis");
                return true;
            });
            vi.mocked(mockStaticProvider.getCompletions).mockResolvedValue([]);

            await completionEngine.getCompletions(["vrooli", "routine", "lis"]);
        });

        it("should parse option context correctly", async () => {
            vi.mocked(mockStaticProvider.canHandle).mockImplementation((context: CompletionContext) => {
                expect(context.type).toBe("option");
                expect(context.command).toBe("routine");
                expect(context.subcommand).toBe("list");
                expect(context.partial).toBe("--form");
                return true;
            });
            vi.mocked(mockStaticProvider.getCompletions).mockResolvedValue([]);

            await completionEngine.getCompletions(["vrooli", "routine", "list", "--form"]);
        });

        it("should parse file context correctly", async () => {
            vi.mocked(mockStaticProvider.canHandle).mockReturnValue(false);
            vi.mocked(mockFileProvider.canHandle).mockImplementation((context: CompletionContext) => {
                expect(context.type).toBe("file");
                expect(context.command).toBe("routine");
                expect(context.subcommand).toBe("import");
                expect(context.partial).toBe("file.j");
                return true;
            });
            vi.mocked(mockFileProvider.getCompletions).mockResolvedValue([]);
            vi.mocked(mockDynamicProvider.canHandle).mockReturnValue(false);

            await completionEngine.getCompletions(["vrooli", "routine", "import", "file.j"]);
        });

        it("should parse resource context correctly", async () => {
            vi.mocked(mockStaticProvider.canHandle).mockReturnValue(false);
            vi.mocked(mockFileProvider.canHandle).mockReturnValue(false);
            vi.mocked(mockDynamicProvider.canHandle).mockImplementation((context: CompletionContext) => {
                expect(context.type).toBe("resource");
                expect(context.command).toBe("routine");
                expect(context.subcommand).toBe("get");
                expect(context.resourceType).toBe("routine");
                expect(context.partial).toBe("123");
                return true;
            });
            vi.mocked(mockCache.get).mockResolvedValue([]);

            await completionEngine.getCompletions(["vrooli", "routine", "get", "123"]);
        });

        it("should handle global options correctly", async () => {
            vi.mocked(mockStaticProvider.canHandle).mockImplementation((context: CompletionContext) => {
                expect(context.args).not.toContain("-p");
                expect(context.args).not.toContain("production");
                expect(context.args).not.toContain("--debug");
                return true;
            });
            vi.mocked(mockStaticProvider.getCompletions).mockResolvedValue([]);

            await completionEngine.getCompletions(["vrooli", "-p", "production", "--debug", "routine", "list"]);
        });

        it("should parse options from args correctly", async () => {
            vi.mocked(mockStaticProvider.canHandle).mockImplementation((context: CompletionContext) => {
                expect(context.options).toEqual({
                    format: "json",
                    verbose: true,
                });
                return true;
            });
            vi.mocked(mockStaticProvider.getCompletions).mockResolvedValue([]);

            await completionEngine.getCompletions(["vrooli", "routine", "list", "--format", "json", "--verbose", "test"]);
        });
    });

    describe("resource type detection", () => {
        const testCases = [
            { command: "routine", subcommand: "export", expectedType: "routine" },
            { command: "agent", subcommand: "get", expectedType: "agent" },
            { command: "team", subcommand: "monitor", expectedType: "team" },
            { command: "chat", subcommand: "create", expectedType: "bot" },
            { command: "history", subcommand: "replay", expectedType: "history" },
        ];

        testCases.forEach(({ command, subcommand, expectedType }) => {
            it(`should detect ${expectedType} resource type for ${command} ${subcommand}`, async () => {
                vi.mocked(mockStaticProvider.canHandle).mockReturnValue(false);
                vi.mocked(mockFileProvider.canHandle).mockReturnValue(false);
                vi.mocked(mockDynamicProvider.canHandle).mockImplementation((context: CompletionContext) => {
                    expect(context.resourceType).toBe(expectedType);
                    return true;
                });
                vi.mocked(mockCache.get).mockResolvedValue([]);

                await completionEngine.getCompletions(["vrooli", command, subcommand, "partial"]);
            });
        });
    });

    describe("file argument detection", () => {
        const testCases = [
            { command: "routine", subcommand: "import", position: 0, expected: true },
            { command: "routine", subcommand: "export", position: 1, expected: true },
            { command: "agent", subcommand: "validate", position: 0, expected: true },
            { command: "chat", subcommand: "interactive", position: 1, expected: true },
            { command: "routine", subcommand: "list", position: 0, expected: false },
        ];

        testCases.forEach(({ command, subcommand, position, expected }) => {
            it(`should ${expected ? "" : "not "}detect file argument for ${command} ${subcommand} at position ${position}`, async () => {
                const args = ["vrooli", command, subcommand];
                for (let i = 0; i <= position; i++) {
                    args.push(`arg${i}`);
                }

                const expectedContextType = expected ? "file" : "option";

                if (expected) {
                    vi.mocked(mockStaticProvider.canHandle).mockReturnValue(false);
                    vi.mocked(mockFileProvider.canHandle).mockImplementation((context: CompletionContext) => {
                        expect(context.type).toBe(expectedContextType);
                        return true;
                    });
                    vi.mocked(mockFileProvider.getCompletions).mockResolvedValue([]);
                    vi.mocked(mockDynamicProvider.canHandle).mockReturnValue(false);
                } else {
                    vi.mocked(mockStaticProvider.canHandle).mockImplementation((context: CompletionContext) => {
                        expect(context.type).toBe(expectedContextType);
                        return true;
                    });
                    vi.mocked(mockStaticProvider.getCompletions).mockResolvedValue([]);
                    vi.mocked(mockFileProvider.canHandle).mockReturnValue(false);
                    vi.mocked(mockDynamicProvider.canHandle).mockReturnValue(false);
                }

                await completionEngine.getCompletions(args);
            });
        });
    });
});
