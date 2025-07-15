// AI_CHECK: TEST_COVERAGE=1 | LAST: 2025-07-13
import { describe, it, expect } from "vitest";
import type {
    CommandOptions,
    AuthLoginOptions,
    ChatStartOptions,
    ChatListOptions,
    RoutineRunOptions,
    RoutineListOptions,
    RoutineUploadOptions,
    RoutineSearchOptions,
    CLIError,
    PaginatedResponse,
    RoutineSearchResult,
    ChatSearchResult,
    AxiosRequestConfigWithRetry,
} from "./command.js";

describe("Command Types", () => {
    describe("CommandOptions", () => {
        it("should have optional profile property", () => {
            const options: CommandOptions = { profile: "test" };
            expect(options.profile).toBe("test");
        });

        it("should allow additional properties", () => {
            const options: CommandOptions = { profile: "test", customFlag: true };
            expect(options.customFlag).toBe(true);
        });
    });

    describe("AuthLoginOptions", () => {
        it("should extend CommandOptions", () => {
            const options: AuthLoginOptions = {
                profile: "test",
                username: "user",
                password: "pass",
            };
            expect(options.profile).toBe("test");
            expect(options.username).toBe("user");
            expect(options.password).toBe("pass");
        });
    });

    describe("ChatStartOptions", () => {
        it("should have chat-specific properties", () => {
            const options: ChatStartOptions = {
                interactive: true,
                context: ["file1.txt"],
                bot: "assistant",
                timeout: "30s",
            };
            expect(options.interactive).toBe(true);
            expect(options.context).toEqual(["file1.txt"]);
            expect(options.bot).toBe("assistant");
            expect(options.timeout).toBe("30s");
        });
    });

    describe("ChatListOptions", () => {
        it("should have list-specific properties", () => {
            const options: ChatListOptions = {
                limit: "10",
                page: "1",
                sort: "name",
            };
            expect(options.limit).toBe("10");
            expect(options.page).toBe("1");
            expect(options.sort).toBe("name");
        });
    });

    describe("RoutineRunOptions", () => {
        it("should have routine run properties", () => {
            const options: RoutineRunOptions = {
                inputs: ["input1", "input2"],
                timeout: "60s",
                watch: true,
            };
            expect(options.inputs).toEqual(["input1", "input2"]);
            expect(options.timeout).toBe("60s");
            expect(options.watch).toBe(true);
        });
    });

    describe("RoutineListOptions", () => {
        it("should have routine list properties", () => {
            const options: RoutineListOptions = {
                limit: "20",
                page: "2",
                type: "workflow",
                owned: true,
            };
            expect(options.limit).toBe("20");
            expect(options.page).toBe("2");
            expect(options.type).toBe("workflow");
            expect(options.owned).toBe(true);
        });
    });

    describe("RoutineUploadOptions", () => {
        it("should have upload properties", () => {
            const options: RoutineUploadOptions = {
                name: "Test Routine",
                description: "A test routine",
                type: "workflow",
                force: true,
            };
            expect(options.name).toBe("Test Routine");
            expect(options.description).toBe("A test routine");
            expect(options.type).toBe("workflow");
            expect(options.force).toBe(true);
        });
    });

    describe("RoutineSearchOptions", () => {
        it("should have search properties", () => {
            const options: RoutineSearchOptions = {
                limit: "50",
                type: "script",
                sort: "relevance",
            };
            expect(options.limit).toBe("50");
            expect(options.type).toBe("script");
            expect(options.sort).toBe("relevance");
        });
    });

    describe("CLIError", () => {
        it("should extend Error with additional properties", () => {
            const error: CLIError = {
                name: "CLIError",
                message: "Test error",
                code: "E001",
                statusCode: 400,
                details: { extra: "info" },
            };
            expect(error.message).toBe("Test error");
            expect(error.code).toBe("E001");
            expect(error.statusCode).toBe(400);
            expect(error.details).toEqual({ extra: "info" });
        });
    });

    describe("PaginatedResponse", () => {
        it("should have correct structure", () => {
            const response: PaginatedResponse<string> = {
                edges: [{ node: "item1" }, { node: "item2" }],
                pageInfo: {
                    hasNextPage: true,
                    hasPreviousPage: false,
                    startCursor: "cursor1",
                    endCursor: "cursor2",
                    count: 2,
                },
            };
            expect(response.edges).toHaveLength(2);
            expect(response.pageInfo.count).toBe(2);
            expect(response.pageInfo.hasNextPage).toBe(true);
        });
    });

    describe("RoutineSearchResult", () => {
        it("should have search result properties", () => {
            const result: RoutineSearchResult = {
                publicId: "routine123",
                name: "Test Routine",
                description: "A test routine",
                type: "workflow",
                score: 0.95,
            };
            expect(result.publicId).toBe("routine123");
            expect(result.name).toBe("Test Routine");
            expect(result.score).toBe(0.95);
        });
    });

    describe("ChatSearchResult", () => {
        it("should have chat search properties", () => {
            const result: ChatSearchResult = {
                id: "chat123",
                publicId: "pub123",
                name: "Test Chat",
                created_at: "2025-01-01T00:00:00Z",
                messages: 5,
            };
            expect(result.id).toBe("chat123");
            expect(result.publicId).toBe("pub123");
            expect(result.name).toBe("Test Chat");
            expect(result.messages).toBe(5);
        });
    });

    describe("AxiosRequestConfigWithRetry", () => {
        it("should extend AxiosRequestConfig with retry flag", () => {
            const config: AxiosRequestConfigWithRetry = {
                url: "/api/test",
                method: "GET",
                _retry: true,
            };
            expect(config.url).toBe("/api/test");
            expect(config.method).toBe("GET");
            expect(config._retry).toBe(true);
        });
    });
});
