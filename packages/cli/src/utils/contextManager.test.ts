// AI_CHECK: TEST_COVERAGE=1 | LAST: 2025-01-12
import { readFile, stat } from "fs/promises";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { LIMITS } from "./constants.js";
import { ContextManager } from "./contextManager.js";

// Mock fs/promises
vi.mock("fs/promises", () => ({
    readFile: vi.fn(),
    stat: vi.fn(),
}));

// Mock fetch
global.fetch = vi.fn() as any;

describe("ContextManager", () => {
    let contextManager: ContextManager;
    let consoleLogSpy: any;

    beforeEach(() => {
        contextManager = new ContextManager();
        consoleLogSpy = vi.spyOn(console, "log").mockImplementation(() => undefined);
    });

    afterEach(() => {
        consoleLogSpy.mockRestore();
        vi.clearAllMocks();
    });

    describe("addFile", () => {
        it("should add a text file successfully", async () => {
            const mockStats = {
                isFile: () => true,
                size: 1024,
            };
            (stat as any).mockResolvedValue(mockStats);
            (readFile as any).mockResolvedValue("file content");

            const contextId = await contextManager.addFile("/path/to/file.txt");

            expect(contextId).toBe("file.txt");
            expect(stat).toHaveBeenCalledWith("/path/to/file.txt");
            expect(readFile).toHaveBeenCalledWith("/path/to/file.txt", "utf8");

            const contexts = contextManager.getAllContexts();
            expect(contexts.size).toBe(1);
            expect(contexts.get("file.txt")).toMatchObject({
                path: "/path/to/file.txt",
                name: "file.txt",
                content: "file content",
                type: "text",
                size: 1024,
                encoding: "utf8",
            });
        });

        it("should add a code file with correct type detection", async () => {
            const mockStats = {
                isFile: () => true,
                size: 2048,
            };
            (stat as any).mockResolvedValue(mockStats);
            (readFile as any).mockResolvedValue("const x = 1;");

            const contextId = await contextManager.addFile("/path/to/file.ts", "mycode");

            expect(contextId).toBe("mycode");
            const contexts = contextManager.getAllContexts();
            expect(contexts.get("mycode")?.type).toBe("code");
        });

        it("should add a binary file as base64", async () => {
            const mockStats = {
                isFile: () => true,
                size: 512,
            };
            const mockBuffer = Buffer.from("binary data");
            (stat as any).mockResolvedValue(mockStats);
            (readFile as any).mockResolvedValue(mockBuffer);

            const contextId = await contextManager.addFile("/path/to/image.png");

            expect(readFile).toHaveBeenCalledWith("/path/to/image.png");
            const contexts = contextManager.getAllContexts();
            expect(contexts.get("image.png")).toMatchObject({
                type: "binary",
                content: mockBuffer.toString("base64"),
                encoding: "base64",
            });
        });

        it("should reject files that are too large", async () => {
            const mockStats = {
                isFile: () => true,
                size: LIMITS.MAX_CONTEXT_SIZE_BYTES + 1,
            };
            (stat as any).mockResolvedValue(mockStats);

            await expect(contextManager.addFile("/path/to/large.bin")).rejects.toThrow("File too large");
        });

        it("should reject non-file paths", async () => {
            const mockStats = {
                isFile: () => false,
                size: 1024,
            };
            (stat as any).mockResolvedValue(mockStats);

            await expect(contextManager.addFile("/path/to/directory")).rejects.toThrow("Path is not a file");
        });

        it("should reject when context limit is reached", async () => {
            const mockStats = {
                isFile: () => true,
                size: 100,
            };
            (stat as any).mockResolvedValue(mockStats);
            (readFile as any).mockResolvedValue("content");

            // Add contexts up to the limit
            for (let i = 0; i < LIMITS.MAX_CONTEXT_MESSAGES; i++) {
                await contextManager.addFile(`/file${i}.txt`, `file${i}`);
            }

            // Try to add one more
            await expect(contextManager.addFile("/extra.txt")).rejects.toThrow("Too many context items");
        });

        it("should handle file read errors", async () => {
            const mockStats = {
                isFile: () => true,
                size: 1024,
            };
            (stat as any).mockResolvedValue(mockStats);
            (readFile as any).mockRejectedValue(new Error("Permission denied"));

            await expect(contextManager.addFile("/protected.txt")).rejects.toThrow("Failed to add file context: Permission denied");
        });
    });

    describe("addUrl", () => {
        it("should add a webpage successfully", async () => {
            const mockResponse = {
                ok: true,
                headers: {
                    get: (name: string) => name === "content-type" ? "text/html" : null,
                },
                text: async () => "<html><title>Test Page</title><body>Content here</body></html>",
            };
            (global.fetch as any).mockResolvedValue(mockResponse);

            const contextId = await contextManager.addUrl("https://example.com");

            expect(contextId).toBe("example.com");
            expect(fetch).toHaveBeenCalledWith("https://example.com", {
                headers: { "User-Agent": "Vrooli CLI Bot" },
            });

            const contexts = contextManager.getAllContexts();
            expect(contexts.size).toBe(1);
            expect(contexts.get("example.com")).toMatchObject({
                url: "https://example.com",
                name: "example.com",
                title: "Test Page",
                content: expect.stringContaining("Content here"),
                type: "webpage",
            });
        });

        it("should use alias when provided", async () => {
            const mockResponse = {
                ok: true,
                headers: {
                    get: () => null,
                },
                text: async () => "Plain text content",
            };
            (global.fetch as any).mockResolvedValue(mockResponse);

            const contextId = await contextManager.addUrl("https://api.example.com/data", "api-data");

            expect(contextId).toBe("api-data");
        });

        it("should reject non-HTTP(S) URLs", async () => {
            await expect(contextManager.addUrl("ftp://example.com")).rejects.toThrow("Only HTTP and HTTPS URLs are supported");
        });

        it("should handle fetch errors", async () => {
            (global.fetch as any).mockRejectedValue(new Error("Network error"));

            await expect(contextManager.addUrl("https://example.com")).rejects.toThrow("Failed to add URL context: Failed to fetch URL: Network error");
        });

        it("should handle HTTP errors", async () => {
            const mockResponse = {
                ok: false,
                status: 404,
                statusText: "Not Found",
            };
            (global.fetch as any).mockResolvedValue(mockResponse);

            await expect(contextManager.addUrl("https://example.com/missing")).rejects.toThrow("HTTP 404: Not Found");
        });

        it("should strip HTML tags from content", async () => {
            const mockResponse = {
                ok: true,
                headers: {
                    get: (name: string) => name === "content-type" ? "text/html" : null,
                },
                text: async () => `
                    <html>
                        <head>
                            <title>Test</title>
                            <script>alert('test');</script>
                            <style>body { color: red; }</style>
                        </head>
                        <body>
                            <p>Hello <b>world</b>!</p>
                        </body>
                    </html>
                `,
            };
            (global.fetch as any).mockResolvedValue(mockResponse);

            await contextManager.addUrl("https://example.com");

            const contexts = contextManager.getAllContexts();
            const content = contexts.get("example.com")?.content;
            expect(content).not.toContain("<script>");
            expect(content).not.toContain("<style>");
            expect(content).not.toContain("<p>");
            expect(content).toContain("Hello world!");
        });
    });

    describe("addRoutine", () => {
        it("should add a routine successfully", () => {
            const contextId = contextManager.addRoutine("routine123", "My Routine", "Test routine description");

            expect(contextId).toBe("My Routine");

            const contexts = contextManager.getAllContexts();
            expect(contexts.size).toBe(1);
            expect(contexts.get("My Routine")).toMatchObject({
                id: "routine123",
                name: "My Routine",
                description: "Test routine description",
                type: "routine",
            });
        });

        it("should use routine ID as name if name not provided", () => {
            const contextId = contextManager.addRoutine("routine456", "");

            expect(contextId).toBe("routine456");
        });

        it("should reject when context limit is reached", () => {
            // Add contexts up to the limit
            for (let i = 0; i < LIMITS.MAX_CONTEXT_MESSAGES; i++) {
                contextManager.addRoutine(`routine${i}`, `Routine ${i}`);
            }

            // Try to add one more
            expect(() => contextManager.addRoutine("extra", "Extra")).toThrow("Too many context items");
        });
    });

    describe("removeContext", () => {
        it("should remove an existing context", async () => {
            // Add a context first
            const mockStats = {
                isFile: () => true,
                size: 100,
            };
            (stat as any).mockResolvedValue(mockStats);
            (readFile as any).mockResolvedValue("content");
            await contextManager.addFile("/test.txt");

            const result = contextManager.removeContext("test.txt");

            expect(result).toBe(true);
            expect(contextManager.getAllContexts().size).toBe(0);
        });

        it("should return false when removing non-existent context", () => {
            const result = contextManager.removeContext("non-existent");

            expect(result).toBe(false);
        });
    });

    describe("clearAll", () => {
        it("should clear all contexts", async () => {
            // Add multiple contexts
            const mockStats = {
                isFile: () => true,
                size: 100,
            };
            (stat as any).mockResolvedValue(mockStats);
            (readFile as any).mockResolvedValue("content");

            await contextManager.addFile("/test1.txt");
            await contextManager.addFile("/test2.txt");
            contextManager.addRoutine("routine1", "Routine 1");

            expect(contextManager.getAllContexts().size).toBe(3);

            contextManager.clearAll();

            expect(contextManager.getAllContexts().size).toBe(0);
        });
    });

    describe("getTaskContexts", () => {
        it("should convert file contexts to TaskContextInfoInput", async () => {
            const mockStats = {
                isFile: () => true,
                size: 100,
            };
            (stat as any).mockResolvedValue(mockStats);
            (readFile as any).mockResolvedValue("file content");

            await contextManager.addFile("/test.txt", "myfile");

            const taskContexts = contextManager.getTaskContexts();

            expect(taskContexts).toHaveLength(1);
            expect(taskContexts[0]).toMatchObject({
                id: "myfile",
                label: "myfile",
                name: "myfile",
                data: {
                    type: "file",
                    path: "/test.txt",
                    content: "file content",
                    encoding: "utf8",
                    size: 100,
                },
            });
        });

        it("should convert URL contexts to TaskContextInfoInput", async () => {
            const mockResponse = {
                ok: true,
                headers: {
                    get: () => null,
                },
                text: async () => "Web content",
            };
            (global.fetch as any).mockResolvedValue(mockResponse);

            await contextManager.addUrl("https://example.com", "webpage");

            const taskContexts = contextManager.getTaskContexts();

            expect(taskContexts).toHaveLength(1);
            expect(taskContexts[0]).toMatchObject({
                id: "webpage",
                label: "webpage",
                name: "webpage",
                data: {
                    type: "url",
                    url: "https://example.com",
                    content: "Web content",
                    size: 11,
                },
            });
        });

        it("should convert routine contexts to TaskContextInfoInput", () => {
            contextManager.addRoutine("routine123", "My Routine", "Description");

            const taskContexts = contextManager.getTaskContexts();

            expect(taskContexts).toHaveLength(1);
            expect(taskContexts[0]).toMatchObject({
                id: "My Routine",
                label: "My Routine",
                name: "My Routine",
                description: "Description",
                data: {
                    type: "routine",
                    routineId: "routine123",
                    description: "Description",
                },
            });
        });
    });

    describe("displayContextSummary", () => {
        it("should display empty message when no contexts", () => {
            contextManager.displayContextSummary();

            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("No context items"));
        });

        it("should display file context summary", async () => {
            const mockStats = {
                isFile: () => true,
                size: 1024,
            };
            (stat as any).mockResolvedValue(mockStats);
            (readFile as any).mockResolvedValue("content");

            await contextManager.addFile("/test.txt");

            contextManager.displayContextSummary();

            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Context Items (1/100):"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("📄"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("test.txt"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Type: text"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Size: 1 KB"));
        });

        it("should display URL context summary", async () => {
            const mockResponse = {
                ok: true,
                headers: {
                    get: (name: string) => name === "content-type" ? "text/html" : null,
                },
                text: async () => "<title>Test Page</title>Content",
            };
            (global.fetch as any).mockResolvedValue(mockResponse);

            await contextManager.addUrl("https://example.com");

            contextManager.displayContextSummary();

            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("🌐"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("example.com"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Title: Test Page"));
        });

        it("should display routine context summary", () => {
            contextManager.addRoutine("routine123", "My Routine", "Test description");

            contextManager.displayContextSummary();

            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("⚙️"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("My Routine"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Test description"));
        });
    });

    describe("getStats", () => {
        it("should return correct statistics", async () => {
            // Add various types of contexts
            const mockStats = {
                isFile: () => true,
                size: 1024,
            };
            (stat as any).mockResolvedValue(mockStats);
            (readFile as any).mockResolvedValue("content");

            const mockResponse = {
                ok: true,
                headers: {
                    get: () => null,
                },
                text: async () => "Web content",
            };
            (global.fetch as any).mockResolvedValue(mockResponse);

            await contextManager.addFile("/test1.txt");
            await contextManager.addFile("/test2.txt");
            await contextManager.addUrl("https://example.com");
            contextManager.addRoutine("routine1", "Routine");

            const stats = contextManager.getStats();

            expect(stats).toMatchObject({
                totalContexts: 4,
                maxContexts: LIMITS.MAX_CONTEXT_MESSAGES,
                totalSize: 2048 + 11, // 2 files of 1024 bytes + URL content of 11 bytes
                byType: {
                    file: 2,
                    url: 1,
                    routine: 1,
                },
            });
        });

        it("should handle empty context manager", () => {
            const stats = contextManager.getStats();

            expect(stats).toMatchObject({
                totalContexts: 0,
                maxContexts: LIMITS.MAX_CONTEXT_MESSAGES,
                totalSize: 0,
                byType: {},
            });
        });
    });

    describe("file type detection", () => {
        const testCases = [
            { file: "test.ts", expectedType: "code" },
            { file: "test.js", expectedType: "code" },
            { file: "test.py", expectedType: "code" },
            { file: "test.json", expectedType: "json" },
            { file: "test.md", expectedType: "markdown" },
            { file: "test.txt", expectedType: "text" },
            { file: "test.log", expectedType: "text" },
            { file: "test.png", expectedType: "binary" },
            { file: "test.unknown", expectedType: "binary" },
        ];

        testCases.forEach(({ file, expectedType }) => {
            it(`should detect ${file} as ${expectedType}`, async () => {
                const mockStats = {
                    isFile: () => true,
                    size: 100,
                };
                (stat as any).mockResolvedValue(mockStats);
                (readFile as any).mockResolvedValue(expectedType === "binary" ? Buffer.from("data") : "content");

                await contextManager.addFile(`/path/${file}`);

                const contexts = contextManager.getAllContexts();
                expect(contexts.get(file)?.type).toBe(expectedType);
            });
        });
    });
});
