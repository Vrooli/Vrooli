import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { AxiosError } from "axios";
import { 
    CommandError, 
    handleCommandError, 
    withErrorHandler,
    createCommandAction,
} from "./errorHandler.js";
import { output } from "./output.js";
import type { Ora } from "ora";

// Mock output module
vi.mock("./output.js", () => ({
    output: {
        error: vi.fn(),
        debug: vi.fn(),
        info: vi.fn(),
        success: vi.fn(),
    },
}));

describe("errorHandler", () => {
    let processExitSpy: any;
    let originalEnv: NodeJS.ProcessEnv;

    beforeEach(() => {
        vi.clearAllMocks();
        processExitSpy = vi.spyOn(process, "exit").mockImplementation(() => {
            throw new Error("process.exit called");
        });
        originalEnv = { ...process.env };
    });

    afterEach(() => {
        processExitSpy.mockRestore();
        process.env = originalEnv;
    });

    describe("CommandError", () => {
        it("should create error with message and default exit code", () => {
            const error = new CommandError("Test error");
            
            expect(error.message).toBe("Test error");
            expect(error.exitCode).toBe(1);
            expect(error.details).toBeUndefined();
            expect(error.name).toBe("CommandError");
        });

        it("should create error with custom exit code", () => {
            const error = new CommandError("Test error", 2);
            
            expect(error.exitCode).toBe(2);
        });

        it("should create error with details", () => {
            const details = { foo: "bar" };
            const error = new CommandError("Test error", 1, details);
            
            expect(error.details).toEqual(details);
        });
    });

    describe("handleCommandError", () => {
        it("should handle CommandError", () => {
            const error = new CommandError("Command failed", 3);
            
            expect(() => handleCommandError(error)).toThrow("process.exit called");
            
            expect(output.error).toHaveBeenCalledWith("Command failed");
            expect(processExitSpy).toHaveBeenCalledWith(3);
        });

        it("should handle CommandError with details in debug mode", () => {
            process.env.DEBUG = "true";
            const details = { detail: "info" };
            const error = new CommandError("Command failed", 1, details);
            
            expect(() => handleCommandError(error)).toThrow("process.exit called");
            
            expect(output.error).toHaveBeenCalledWith("Command failed");
            expect(output.debug).toHaveBeenCalledWith("Error details:", error);
            expect(processExitSpy).toHaveBeenCalledWith(1);
        });

        it("should fail spinner if provided", () => {
            const mockSpinner = { fail: vi.fn() } as unknown as Ora;
            const error = new Error("Test error");
            
            expect(() => handleCommandError(error, mockSpinner)).toThrow("process.exit called");
            
            expect(mockSpinner.fail).toHaveBeenCalled();
        });

        it("should add context to error message", () => {
            const error = new Error("Test error");
            
            expect(() => handleCommandError(error, undefined, "During operation")).toThrow("process.exit called");
            
            expect(output.error).toHaveBeenCalledWith("During operation: Test error");
        });

        it("should handle AxiosError with 401 status", () => {
            const error = new AxiosError("Unauthorized");
            error.response = {
                status: 401,
                data: {},
                statusText: "Unauthorized",
                headers: {},
                config: {} as any,
            };
            
            expect(() => handleCommandError(error)).toThrow("process.exit called");
            
            expect(output.error).toHaveBeenCalledWith(
                "Authentication failed. Please login again with 'vrooli auth login'",
            );
        });

        it("should handle AxiosError with 403 status", () => {
            const error = new AxiosError("Forbidden");
            error.response = {
                status: 403,
                data: {},
                statusText: "Forbidden",
                headers: {},
                config: {} as any,
            };
            
            expect(() => handleCommandError(error)).toThrow("process.exit called");
            
            expect(output.error).toHaveBeenCalledWith(
                "Permission denied. You don't have access to this resource.",
            );
        });

        it("should handle AxiosError with 404 status", () => {
            const error = new AxiosError("Not Found");
            error.response = {
                status: 404,
                data: {},
                statusText: "Not Found",
                headers: {},
                config: {} as any,
            };
            
            expect(() => handleCommandError(error)).toThrow("process.exit called");
            
            expect(output.error).toHaveBeenCalledWith("Resource not found.");
        });

        it("should handle AxiosError with 500+ status", () => {
            const error = new AxiosError("Server Error");
            error.response = {
                status: 500,
                data: {},
                statusText: "Internal Server Error",
                headers: {},
                config: {} as any,
            };
            
            expect(() => handleCommandError(error)).toThrow("process.exit called");
            
            expect(output.error).toHaveBeenCalledWith(
                "Server error. Please try again later.",
            );
        });

        it("should handle AxiosError with custom message in data", () => {
            const error = new AxiosError("Bad Request");
            error.response = {
                status: 400,
                data: { message: "Invalid input provided" },
                statusText: "Bad Request",
                headers: {},
                config: {} as any,
            };
            
            expect(() => handleCommandError(error)).toThrow("process.exit called");
            
            expect(output.error).toHaveBeenCalledWith("Invalid input provided");
        });

        it("should handle AxiosError with error details in debug mode", () => {
            process.env.DEBUG = "true";
            const error = new AxiosError("Bad Request");
            error.response = {
                status: 400,
                data: { error: "Detailed error info" },
                statusText: "Bad Request",
                headers: {},
                config: {} as any,
            };
            
            expect(() => handleCommandError(error)).toThrow("process.exit called");
            
            expect(output.debug).toHaveBeenCalledWith("Error details:", error);
        });

        it("should handle AxiosError with network error", () => {
            const error = new AxiosError("Network Error");
            error.request = {}; // Request was made but no response
            
            expect(() => handleCommandError(error)).toThrow("process.exit called");
            
            expect(output.error).toHaveBeenCalledWith(
                "Network error. Please check your connection and try again.",
            );
        });

        it("should handle AxiosError with message but no response", () => {
            const error = new AxiosError("Connection timeout");
            
            expect(() => handleCommandError(error)).toThrow("process.exit called");
            
            expect(output.error).toHaveBeenCalledWith("Connection timeout");
        });

        it("should handle generic Error", () => {
            const error = new Error("Generic error");
            
            expect(() => handleCommandError(error)).toThrow("process.exit called");
            
            expect(output.error).toHaveBeenCalledWith("Generic error");
            expect(processExitSpy).toHaveBeenCalledWith(1);
        });

        it("should handle string error", () => {
            const error = "String error message";
            
            expect(() => handleCommandError(error)).toThrow("process.exit called");
            
            expect(output.error).toHaveBeenCalledWith("String error message");
        });

        it("should handle unknown error type", () => {
            const error = { some: "object" };
            process.env.DEBUG = "true";
            
            expect(() => handleCommandError(error)).toThrow("process.exit called");
            
            expect(output.error).toHaveBeenCalledWith("An unexpected error occurred");
            expect(output.debug).toHaveBeenCalledWith("Error details:", error);
        });

        it("should handle null error", () => {
            const error = null;
            
            expect(() => handleCommandError(error)).toThrow("process.exit called");
            
            expect(output.error).toHaveBeenCalledWith("An unexpected error occurred");
        });

        it("should handle undefined error", () => {
            const error = undefined;
            
            expect(() => handleCommandError(error)).toThrow("process.exit called");
            
            expect(output.error).toHaveBeenCalledWith("An unexpected error occurred");
        });
    });

    describe("withErrorHandler", () => {
        it("should wrap async function and return result on success", async () => {
            const mockFn = vi.fn().mockResolvedValue("success");
            const wrapped = withErrorHandler(mockFn);
            
            const result = await wrapped("arg1", "arg2");
            
            expect(result).toBe("success");
            expect(mockFn).toHaveBeenCalledWith("arg1", "arg2");
        });

        it("should handle errors in wrapped function", async () => {
            const mockFn = vi.fn().mockRejectedValue(new Error("Async error"));
            const wrapped = withErrorHandler(mockFn);
            
            await expect(wrapped()).rejects.toThrow("process.exit called");
            
            expect(output.error).toHaveBeenCalledWith("Async error");
            expect(processExitSpy).toHaveBeenCalledWith(1);
        });

        it("should add context to errors in wrapped function", async () => {
            const mockFn = vi.fn().mockRejectedValue(new Error("Async error"));
            const wrapped = withErrorHandler(mockFn, "During async operation");
            
            await expect(wrapped()).rejects.toThrow("process.exit called");
            
            expect(output.error).toHaveBeenCalledWith("During async operation: Async error");
        });

        it("should preserve function parameters and types", async () => {
            const mockFn = vi.fn<[string, number, boolean], Promise<string>>()
                .mockResolvedValue("result");
            const wrapped = withErrorHandler(mockFn);
            
            await wrapped("test", 123, true);
            
            expect(mockFn).toHaveBeenCalledWith("test", 123, true);
        });
    });

    describe("createCommandAction", () => {
        it("should create wrapped command action", async () => {
            const mockAction = vi.fn().mockResolvedValue(undefined);
            const wrapped = createCommandAction(mockAction);
            
            await wrapped("arg1", "arg2");
            
            expect(mockAction).toHaveBeenCalledWith("arg1", "arg2");
        });

        it("should handle errors in command action", async () => {
            const mockAction = vi.fn().mockRejectedValue(new Error("Action error"));
            const wrapped = createCommandAction(mockAction);
            
            await expect(wrapped()).rejects.toThrow("process.exit called");
            
            expect(output.error).toHaveBeenCalledWith("Action error");
            expect(processExitSpy).toHaveBeenCalledWith(1);
        });

        it("should add context to command action errors", async () => {
            const mockAction = vi.fn().mockRejectedValue(new Error("Action error"));
            const wrapped = createCommandAction(mockAction, "Command execution");
            
            await expect(wrapped()).rejects.toThrow("process.exit called");
            
            expect(output.error).toHaveBeenCalledWith("Command execution: Action error");
        });

        it("should handle CommandError with custom exit code", async () => {
            const mockAction = vi.fn().mockRejectedValue(
                new CommandError("Custom error", 5),
            );
            const wrapped = createCommandAction(mockAction);
            
            await expect(wrapped()).rejects.toThrow("process.exit called");
            
            expect(output.error).toHaveBeenCalledWith("Custom error");
            expect(processExitSpy).toHaveBeenCalledWith(5);
        });

        it("should preserve action parameters", async () => {
            const mockAction = vi.fn<[string, number], Promise<void>>()
                .mockResolvedValue(undefined);
            const wrapped = createCommandAction(mockAction);
            
            await wrapped("test", 42);
            
            expect(mockAction).toHaveBeenCalledWith("test", 42);
        });
    });
});
