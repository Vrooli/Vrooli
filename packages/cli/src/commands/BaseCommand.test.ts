import { Command } from "commander";
import { afterEach, beforeEach, describe, expect, it, vi, type Mock } from "vitest";
import { BaseCommand, type CommandOptions } from "./BaseCommand.js";
import { type ApiClient } from "../utils/client.js";
import { type ConfigManager } from "../utils/config.js";
import { output } from "../utils/output.js";

// Mock dependencies
const mockSpinner = {
    start: vi.fn().mockReturnThis(),
    succeed: vi.fn().mockReturnThis(),
    fail: vi.fn().mockReturnThis(),
};

vi.mock("ora", () => ({
    default: vi.fn(() => mockSpinner),
}));

vi.mock("../utils/output.js", () => ({
    output: {
        json: vi.fn(),
        error: vi.fn(),
        info: vi.fn(),
        table: vi.fn(),
    },
}));

vi.mock("fs/promises", () => ({
    readFile: vi.fn(),
    writeFile: vi.fn(),
    mkdir: vi.fn(),
}));

vi.mock("path", () => ({
    dirname: vi.fn(() => "/some/dir"),
}));

// Concrete implementation for testing the abstract class
class TestCommand extends BaseCommand {
    constructor(program: Command, client: ApiClient, config: ConfigManager) {
        super(program, client, config);
    }

    protected registerCommands(): void {
        // Test implementation
        this.program.command("test").action(() => {
            // Test action
        });
    }

    // Expose protected methods for testing
    public testExecuteWithSpinner<T>(
        message: string,
        operation: () => Promise<T>,
        successMessage?: string,
    ): Promise<T> {
        return this.executeWithSpinner(message, operation, successMessage);
    }

    public testCreateSpinner(message: string) {
        return this.createSpinner(message);
    }

    public testHandleError(error: unknown, context?: string): never {
        return this.handleError(error, context);
    }

    public testOutput(data: unknown, defaultFormatter?: () => void): void {
        return this.output(data, defaultFormatter);
    }

    public testDisplayTable(data: Array<Record<string, unknown>>): void {
        return this.displayTable(data);
    }

    public testRequireAuth(): void {
        return this.requireAuth();
    }

    public testParseJsonFile<T>(filePath: string): Promise<T> {
        return this.parseJsonFile<T>(filePath);
    }

    public testWriteJsonFile(filePath: string, data: unknown): Promise<void> {
        return this.writeJsonFile(filePath, data);
    }

    public testValidateEmail(email: string): boolean {
        return this.validateEmail(email);
    }

    public testValidateUuid(uuid: string): boolean {
        return this.validateUuid(uuid);
    }

    public testDebug(message: string, data?: unknown): void {
        return this.debug(message, data);
    }
}

describe("BaseCommand", () => {
    let mockProgram: Command;
    let mockClient: ApiClient;
    let mockConfig: ConfigManager;
    let testCommand: TestCommand;
    let consoleDebugSpy: Mock;
    let processExitSpy: Mock;

    beforeEach(() => {
        // Reset spinner mocks
        vi.clearAllMocks();

        // Mock program
        mockProgram = {
            command: vi.fn().mockReturnThis(),
            action: vi.fn().mockReturnThis(),
        } as any;

        // Mock client
        mockClient = {} as ApiClient;

        // Mock config
        mockConfig = {
            isJsonOutput: vi.fn().mockReturnValue(false),
            isDebug: vi.fn().mockReturnValue(false),
            getAuthToken: vi.fn().mockReturnValue("test-token"),
        } as any;

        consoleDebugSpy = vi.spyOn(console, "debug").mockImplementation(() => {});
        processExitSpy = vi.spyOn(process, "exit").mockImplementation(() => undefined as never);

        testCommand = new TestCommand(mockProgram, mockClient, mockConfig);
    });

    afterEach(() => {
        consoleDebugSpy.mockRestore();
        processExitSpy.mockRestore();
    });

    describe("constructor and registerCommands", () => {
        it("should call registerCommands during construction", () => {
            expect(mockProgram.command).toHaveBeenCalledWith("test");
        });
    });

    describe("executeWithSpinner", () => {
        it("should execute operation with success", async () => {
            const operation = vi.fn().mockResolvedValue("result");

            const result = await testCommand.testExecuteWithSpinner("Testing", operation, "Success");

            expect(result).toBe("result");
            expect(mockSpinner.start).toHaveBeenCalled();
            expect(mockSpinner.succeed).toHaveBeenCalledWith("Success");
            expect(operation).toHaveBeenCalled();
        });

        it("should handle operation failure", async () => {
            const error = new Error("Operation failed");
            const operation = vi.fn().mockRejectedValue(error);

            await expect(testCommand.testExecuteWithSpinner("Testing", operation)).rejects.toThrow("Operation failed");

            expect(mockSpinner.start).toHaveBeenCalled();
            expect(mockSpinner.fail).toHaveBeenCalledWith("Failed: Testing");
        });

        it("should use default success message if not provided", async () => {
            const operation = vi.fn().mockResolvedValue("result");

            await testCommand.testExecuteWithSpinner("Testing", operation);

            expect(mockSpinner.succeed).toHaveBeenCalledWith("Testing");
        });
    });

    describe("createSpinner", () => {
        it("should create a spinner with the given message", () => {
            const spinner = testCommand.testCreateSpinner("Test message");
            expect(spinner).toBeDefined();
        });
    });

    describe("handleError", () => {
        beforeEach(() => {
            // Set NODE_ENV to test to prevent process.exit
            process.env.NODE_ENV = "test";
        });

        afterEach(() => {
            delete process.env.NODE_ENV;
        });

        it("should handle Error objects in JSON mode", () => {
            mockConfig.isJsonOutput = vi.fn().mockReturnValue(true);
            const error = new Error("Test error");

            expect(() => testCommand.testHandleError(error, "Test context")).toThrow("Test context: Test error");
            expect(output.json).toHaveBeenCalledWith({
                error: true,
                message: "Test error",
                context: "Test context",
            });
        });

        it("should handle Error objects in normal mode", () => {
            mockConfig.isJsonOutput = vi.fn().mockReturnValue(false);
            const error = new Error("Test error");

            expect(() => testCommand.testHandleError(error, "Test context")).toThrow("Test context: Test error");
            expect(output.error).toHaveBeenCalledWith("Test context: Test error");
        });

        it("should handle non-Error objects", () => {
            mockConfig.isJsonOutput = vi.fn().mockReturnValue(false);

            expect(() => testCommand.testHandleError("String error")).toThrow("String error");
            expect(output.error).toHaveBeenCalledWith("String error");
        });

        it("should handle error without context", () => {
            mockConfig.isJsonOutput = vi.fn().mockReturnValue(false);
            const error = new Error("Test error");

            expect(() => testCommand.testHandleError(error)).toThrow("Test error");
            expect(output.error).toHaveBeenCalledWith("Test error");
        });

        it("should call process.exit when not in test environment", () => {
            delete process.env.NODE_ENV;
            const error = new Error("Test error");

            testCommand.testHandleError(error);

            expect(processExitSpy).toHaveBeenCalledWith(1);
        });
    });

    describe("output", () => {
        it("should output JSON when JSON mode is enabled", () => {
            mockConfig.isJsonOutput = vi.fn().mockReturnValue(true);
            const data = { test: "data" };

            testCommand.testOutput(data);

            expect(output.json).toHaveBeenCalledWith(data);
        });

        it("should use default formatter when provided and not in JSON mode", () => {
            mockConfig.isJsonOutput = vi.fn().mockReturnValue(false);
            const formatter = vi.fn();

            testCommand.testOutput("data", formatter);

            expect(formatter).toHaveBeenCalled();
            expect(output.info).not.toHaveBeenCalled();
        });

        it("should use output.info when no formatter provided and not in JSON mode", () => {
            mockConfig.isJsonOutput = vi.fn().mockReturnValue(false);

            testCommand.testOutput("test data");

            expect(output.info).toHaveBeenCalledWith("test data");
        });
    });

    describe("displayTable", () => {
        it("should output JSON when JSON mode is enabled", () => {
            mockConfig.isJsonOutput = vi.fn().mockReturnValue(true);
            const data = [{ col1: "value1", col2: "value2" }];

            testCommand.testDisplayTable(data);

            expect(output.json).toHaveBeenCalledWith(data);
        });

        it("should use output.table when not in JSON mode", () => {
            mockConfig.isJsonOutput = vi.fn().mockReturnValue(false);
            const data = [{ col1: "value1", col2: "value2" }];

            testCommand.testDisplayTable(data);

            expect(output.table).toHaveBeenCalledWith(data);
        });
    });

    describe("requireAuth", () => {
        beforeEach(() => {
            process.env.NODE_ENV = "test";
        });

        afterEach(() => {
            delete process.env.NODE_ENV;
        });

        it("should pass when user is authenticated", () => {
            mockConfig.getAuthToken = vi.fn().mockReturnValue("valid-token");

            expect(() => testCommand.testRequireAuth()).not.toThrow();
        });

        it("should throw when user is not authenticated", () => {
            mockConfig.getAuthToken = vi.fn().mockReturnValue(null);

            expect(() => testCommand.testRequireAuth()).toThrow("Authentication required: You must be logged in to use this command");
        });
    });

    describe("parseJsonFile", () => {
        let fsMock: any;

        beforeEach(async () => {
            fsMock = await import("fs/promises");
        });

        it("should parse valid JSON file", async () => {
            const testData = { test: "data" };
            fsMock.readFile.mockResolvedValue(JSON.stringify(testData));

            const result = await testCommand.testParseJsonFile<typeof testData>("test.json");

            expect(result).toEqual(testData);
            expect(fsMock.readFile).toHaveBeenCalledWith("test.json", "utf-8");
        });

        it("should handle file not found error", async () => {
            const error = new Error("File not found") as NodeJS.ErrnoException;
            error.code = "ENOENT";
            fsMock.readFile.mockRejectedValue(error);

            await expect(testCommand.testParseJsonFile("nonexistent.json")).rejects.toThrow("File not found: nonexistent.json");
        });

        it("should handle invalid JSON", async () => {
            const syntaxError = new SyntaxError("Unexpected token");
            fsMock.readFile.mockResolvedValue("invalid json");
            // Mock JSON.parse to throw SyntaxError
            const originalParse = JSON.parse;
            JSON.parse = vi.fn().mockImplementation(() => {
                throw syntaxError;
            });

            await expect(testCommand.testParseJsonFile("invalid.json")).rejects.toThrow("Invalid JSON in file: invalid.json");

            JSON.parse = originalParse;
        });

        it("should handle other file system errors", async () => {
            const error = new Error("Permission denied");
            fsMock.readFile.mockRejectedValue(error);

            await expect(testCommand.testParseJsonFile("test.json")).rejects.toThrow("Permission denied");
        });
    });

    describe("writeJsonFile", () => {
        let fsMock: any;
        let pathMock: any;

        beforeEach(async () => {
            fsMock = await import("fs/promises");
            pathMock = await import("path");
            fsMock.mkdir.mockResolvedValue(undefined);
            fsMock.writeFile.mockResolvedValue(undefined);
        });

        it("should write JSON data to file", async () => {
            const data = { test: "data" };
            pathMock.dirname.mockReturnValue("/some/dir");

            await testCommand.testWriteJsonFile("/some/dir/test.json", data);

            expect(fsMock.mkdir).toHaveBeenCalledWith("/some/dir", { recursive: true });
            expect(fsMock.writeFile).toHaveBeenCalledWith("/some/dir/test.json", JSON.stringify(data, null, 2));
        });
    });

    describe("validateEmail", () => {
        it("should validate correct email addresses", () => {
            expect(testCommand.testValidateEmail("test@example.com")).toBe(true);
            expect(testCommand.testValidateEmail("user.name@domain.co.uk")).toBe(true);
            expect(testCommand.testValidateEmail("test+tag@example.org")).toBe(true);
        });

        it("should reject invalid email addresses", () => {
            expect(testCommand.testValidateEmail("invalid")).toBe(false);
            expect(testCommand.testValidateEmail("@example.com")).toBe(false);
            expect(testCommand.testValidateEmail("test@")).toBe(false);
            expect(testCommand.testValidateEmail("test@.com")).toBe(false);
            expect(testCommand.testValidateEmail("test test@example.com")).toBe(false);
        });
    });

    describe("validateUuid", () => {
        it("should validate correct UUIDs", () => {
            expect(testCommand.testValidateUuid("123e4567-e89b-12d3-a456-426614174000")).toBe(true);
            expect(testCommand.testValidateUuid("550e8400-e29b-41d4-a716-446655440000")).toBe(true);
            expect(testCommand.testValidateUuid("6ba7b810-9dad-11d1-80b4-00c04fd430c8")).toBe(true);
        });

        it("should reject invalid UUIDs", () => {
            expect(testCommand.testValidateUuid("invalid")).toBe(false);
            expect(testCommand.testValidateUuid("123e4567-e89b-12d3-a456")).toBe(false);
            expect(testCommand.testValidateUuid("123e4567-e89b-12d3-a456-42661417400g")).toBe(false);
            expect(testCommand.testValidateUuid("123e4567e89b12d3a456426614174000")).toBe(false);
        });
    });

    describe("debug", () => {
        it("should log debug message when debug mode is enabled", () => {
            mockConfig.isDebug = vi.fn().mockReturnValue(true);

            testCommand.testDebug("Test message");

            expect(consoleDebugSpy).toHaveBeenCalledWith("[DEBUG] Test message");
        });

        it("should log debug message with data when debug mode is enabled", () => {
            mockConfig.isDebug = vi.fn().mockReturnValue(true);
            const testData = { test: "data" };

            testCommand.testDebug("Test message", testData);

            expect(consoleDebugSpy).toHaveBeenCalledWith("[DEBUG] Test message:", testData);
        });

        it("should not log when debug mode is disabled", () => {
            mockConfig.isDebug = vi.fn().mockReturnValue(false);

            testCommand.testDebug("Test message");

            expect(consoleDebugSpy).not.toHaveBeenCalled();
        });
    });
});