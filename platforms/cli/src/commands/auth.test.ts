import type { Session } from "@vrooli/shared";
import { Command } from "commander";
import inquirer from "inquirer";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "../utils/client.js";
import { ConfigManager } from "../utils/config.js";
import { AuthCommands } from "./auth.js";

// Mock dependencies
vi.mock("../utils/client.js");
vi.mock("../utils/config.js");
vi.mock("../utils/output.js", () => ({
    output: {
        success: vi.fn(),
        error: vi.fn(),
        warn: vi.fn(),
        info: vi.fn(),
        debug: vi.fn(),
        json: vi.fn(),
    },
}));
vi.mock("chalk", () => ({
    default: {
        red: vi.fn((str: string) => str),
        green: vi.fn((str: string) => str),
        yellow: vi.fn((str: string) => str),
        gray: vi.fn((str: string) => str),
        bold: vi.fn((str: string) => str),
        blue: vi.fn((str: string) => str),
        underline: { bold: vi.fn((str: string) => str) },
    },
}));
vi.mock("ora", () => ({
    default: vi.fn(() => ({
        start: vi.fn().mockReturnThis(),
        succeed: vi.fn().mockReturnThis(),
        fail: vi.fn().mockReturnThis(),
    })),
}));
vi.mock("inquirer", () => ({
    default: {
        prompt: vi.fn(),
    },
}));

describe("AuthCommands", () => {
    let program: Command;
    let client: ApiClient;
    let config: ConfigManager;
    let authCommands: AuthCommands;
    let outputMock: any;
    let processExitSpy: any;

    beforeEach(async () => {
        // Clear all mocks
        vi.clearAllMocks();

        // Create fresh instances
        program = new Command();
        program.exitOverride(); // Prevent actual process exit during tests

        client = new ApiClient({} as ConfigManager);
        config = new ConfigManager();

        // Setup config mocks
        vi.mocked(config).getActiveProfileName.mockReturnValue("default");
        vi.mocked(config).getServerUrl.mockReturnValue("http://localhost:5329");
        vi.mocked(config).isJsonOutput.mockReturnValue(false);
        vi.mocked(config).isDebug.mockReturnValue(false);

        // Get output mock
        const { output } = await import("../utils/output.js");
        outputMock = vi.mocked(output);

        // Setup process.exit spy
        processExitSpy = vi.spyOn(process, "exit").mockImplementation((code?: number) => {
            throw new Error(`Process exited with code ${code}`);
        });

        // Create AuthCommands instance
        authCommands = new AuthCommands(program, client, config);
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    describe("Command Registration", () => {
        it("should register auth command with subcommands", () => {
            const authCmd = program.commands.find(cmd => cmd.name() === "auth");
            expect(authCmd).toBeDefined();
            expect(authCmd?.description()).toBe("Authentication commands");

            const subCommands = authCmd?.commands.map(cmd => cmd.name());
            expect(subCommands).toContain("login");
            expect(subCommands).toContain("logout");
            expect(subCommands).toContain("status");
            expect(subCommands).toContain("whoami");
        });
    });

    describe("login command", () => {
        const mockSession: Session = {
            id: "session-123",
            userId: "user-123",
            token: "auth-token",
            users: [{
                id: "user-123",
                handle: "testuser",
                name: "Test User",
                languages: ["en"],
                credits: "100",
                hasPremium: false,
                updatedAt: new Date().toISOString(),
            }],
        } as any;

        it("should login successfully with provided credentials", async () => {
            vi.mocked(client.post).mockResolvedValue(mockSession);

            await program.parseAsync(["node", "cli", "auth", "login", "-e", "test@example.com", "-p", "password123"]);

            expect(vi.mocked(client.post)).toHaveBeenCalledWith("/auth/email/login", {
                email: "test@example.com",
                password: "password123",
            });
            expect(vi.mocked(config.setSession)).toHaveBeenCalledWith(mockSession);
            expect(outputMock.success).toHaveBeenCalledWith(expect.stringContaining("✓ Authentication successful"));
            expect(outputMock.info).toHaveBeenCalledWith(expect.stringContaining("User: Test User"));
            expect(outputMock.info).toHaveBeenCalledWith(expect.stringContaining("Handle: testuser"));
        });

        it("should prompt for credentials when not provided", async () => {
            vi.mocked(inquirer.prompt)
                .mockResolvedValueOnce({ email: "prompt@example.com" })
                .mockResolvedValueOnce({ password: "promptpass" });
            vi.mocked(client.post).mockResolvedValue(mockSession);

            await program.parseAsync(["node", "cli", "auth", "login"]);

            expect(vi.mocked(inquirer.prompt)).toHaveBeenCalledTimes(2);
            expect(vi.mocked(client.post)).toHaveBeenCalledWith("/auth/email/login", {
                email: "prompt@example.com",
                password: "promptpass",
            });
        });

        it("should not save credentials when --no-save is used", async () => {
            vi.mocked(client.post).mockResolvedValue(mockSession);

            await program.parseAsync(["node", "cli", "auth", "login", "-e", "test@example.com", "-p", "password123", "--no-save"]);

            expect(vi.mocked(config.setSession)).not.toHaveBeenCalled();
        });

        it("should output JSON when JSON mode is enabled", async () => {
            vi.mocked(config.isJsonOutput).mockReturnValue(true);
            vi.mocked(client.post).mockResolvedValue(mockSession);

            await program.parseAsync(["node", "cli", "auth", "login", "-e", "test@example.com", "-p", "password123"]);

            expect(outputMock.json).toHaveBeenCalledWith({
                success: true,
                user: {
                    id: "user-123",
                    handle: "testuser",
                    name: "Test User",
                },
            });
        });

        it("should handle login errors", async () => {
            vi.mocked(client.post).mockRejectedValue(new Error("Invalid credentials"));

            await expect(program.parseAsync(["node", "cli", "auth", "login", "-e", "test@example.com", "-p", "wrong"]))
                .rejects.toThrow("Login failed: Invalid credentials");

            expect(outputMock.error).toHaveBeenCalledWith("Login failed: Invalid credentials");
        });

        it("should handle missing user data in response", async () => {
            const badSession = { ...mockSession, users: [] };
            vi.mocked(client.post).mockResolvedValue(badSession);

            await expect(program.parseAsync(["node", "cli", "auth", "login", "-e", "test@example.com", "-p", "password123"]))
                .rejects.toThrow("Login failed: No user data in login response");

            expect(outputMock.error).toHaveBeenCalledWith("Login failed: No user data in login response");
        });

        it("should validate email format in prompt", async () => {
            // Setup prompt mocks before triggering login
            vi.mocked(inquirer.prompt)
                .mockResolvedValueOnce({ email: "test@example.com" })
                .mockResolvedValueOnce({ password: "password" });
            vi.mocked(client.post).mockResolvedValue(mockSession);

            // Trigger a login to capture the prompt configuration
            await program.parseAsync(["node", "cli", "auth", "login"]);

            // Now check the validation
            const promptConfig = vi.mocked(inquirer.prompt).mock.calls[0][0] as any;
            const emailValidation = promptConfig[0].validate;

            expect(emailValidation("invalid-email")).toBe("Please enter a valid email address");
            expect(emailValidation("valid@email.com")).toBe(true);
        });
    });

    describe("logout command", () => {
        it("should logout successfully when authenticated", async () => {
            vi.mocked(config.getAuthToken).mockReturnValue("auth-token");
            vi.mocked(client.post).mockResolvedValue({});

            await program.parseAsync(["node", "cli", "auth", "logout"]);

            expect(vi.mocked(client.post)).toHaveBeenCalledWith("/auth/logout");
            expect(vi.mocked(config.clearAuth)).toHaveBeenCalled();
            expect(outputMock.success).toHaveBeenCalledWith("✓ Credentials cleared");
        });

        it("should clear auth even if logout API fails", async () => {
            vi.mocked(config.getAuthToken).mockReturnValue("auth-token");
            vi.mocked(client.post).mockRejectedValue(new Error("Network error"));

            await program.parseAsync(["node", "cli", "auth", "logout"]);

            expect(vi.mocked(config.clearAuth)).toHaveBeenCalled();
            expect(outputMock.success).toHaveBeenCalledWith("✓ Credentials cleared");
        });

        it("should logout successfully when not authenticated", async () => {
            vi.mocked(config.getAuthToken).mockReturnValue(null);

            await program.parseAsync(["node", "cli", "auth", "logout"]);

            expect(vi.mocked(client.post)).not.toHaveBeenCalled();
            expect(vi.mocked(config.clearAuth)).toHaveBeenCalled();
            expect(outputMock.success).toHaveBeenCalledWith("✓ Credentials cleared");
        });

        it("should output JSON when JSON mode is enabled", async () => {
            vi.mocked(config.isJsonOutput).mockReturnValue(true);
            vi.mocked(config.getAuthToken).mockReturnValue(null);

            await program.parseAsync(["node", "cli", "auth", "logout"]);

            expect(outputMock.json).toHaveBeenCalledWith({ success: true });
        });
    });

    describe("status command", () => {
        const mockSession: Session = {
            id: "session-123",
            userId: "user-123",
            users: [{
                id: "user-123",
                handle: "testuser",
                name: "Test User",
            }],
        } as any;

        it("should show authenticated status when token is valid", async () => {
            vi.mocked(config.getAuthToken).mockReturnValue("valid-token");
            vi.mocked(client.get).mockResolvedValue(mockSession);

            await program.parseAsync(["node", "cli", "auth", "status"]);

            expect(vi.mocked(client.get)).toHaveBeenCalledWith("/profile");
            expect(outputMock.success).toHaveBeenCalledWith(expect.stringContaining("✓ Authenticated"));
            expect(outputMock.info).toHaveBeenCalledWith(expect.stringContaining("User: Test User"));
            expect(outputMock.info).toHaveBeenCalledWith(expect.stringContaining("Handle: testuser"));
        });

        it("should show not authenticated when no token", async () => {
            vi.mocked(config.getAuthToken).mockReturnValue(null);

            await program.parseAsync(["node", "cli", "auth", "status"]);

            expect(vi.mocked(client.get)).not.toHaveBeenCalled();
            expect(outputMock.warn).toHaveBeenCalledWith("⚠ Not authenticated");
            expect(outputMock.info).toHaveBeenCalledWith(expect.stringContaining("Profile: default"));
        });

        it("should handle invalid token and clear auth", async () => {
            vi.mocked(config.getAuthToken).mockReturnValue("invalid-token");
            vi.mocked(client.get).mockRejectedValue(new Error("401 Unauthorized"));

            await program.parseAsync(["node", "cli", "auth", "status"]);

            expect(vi.mocked(config.clearAuth)).toHaveBeenCalled();
            expect(outputMock.error).toHaveBeenCalledWith(expect.stringContaining("✗ Authentication invalid"));
            expect(outputMock.info).toHaveBeenCalledWith(expect.stringContaining("Run 'vrooli auth login' to authenticate"));
        });

        it("should output JSON for authenticated status", async () => {
            vi.mocked(config.isJsonOutput).mockReturnValue(true);
            vi.mocked(config.getAuthToken).mockReturnValue("valid-token");
            vi.mocked(client.get).mockResolvedValue(mockSession);

            await program.parseAsync(["node", "cli", "auth", "status"]);

            expect(outputMock.json).toHaveBeenCalledWith({
                authenticated: true,
                profile: "default",
                user: {
                    id: "user-123",
                    handle: "testuser",
                    name: "Test User",
                },
            });
        });

        it("should output JSON for not authenticated status", async () => {
            vi.mocked(config.isJsonOutput).mockReturnValue(true);
            vi.mocked(config.getAuthToken).mockReturnValue(null);

            await program.parseAsync(["node", "cli", "auth", "status"]);

            expect(outputMock.json).toHaveBeenCalledWith({
                authenticated: false,
                profile: "default",
            });
        });
    });

    describe("whoami command", () => {
        const mockSession: Session = {
            id: "session-123",
            userId: "user-123",
            users: [{
                id: "user-123",
                handle: "testuser",
                name: "Test User",
                languages: ["en", "es"],
                credits: "100",
                hasPremium: true,
                updatedAt: new Date("2024-01-01").toISOString(),
            }],
        } as any;

        it("should display user information", async () => {
            vi.mocked(client.get).mockResolvedValue(mockSession);

            await program.parseAsync(["node", "cli", "auth", "whoami"]);

            expect(vi.mocked(client.get)).toHaveBeenCalledWith("/profile");
            expect(outputMock.info).toHaveBeenCalledWith(expect.stringContaining("User Information:"));
            expect(outputMock.info).toHaveBeenCalledWith(expect.stringContaining("ID: user-123"));
            expect(outputMock.info).toHaveBeenCalledWith(expect.stringContaining("Handle: testuser"));
            expect(outputMock.info).toHaveBeenCalledWith(expect.stringContaining("Name: Test User"));
            expect(outputMock.info).toHaveBeenCalledWith(expect.stringContaining("Languages: en, es"));
            expect(outputMock.info).toHaveBeenCalledWith(expect.stringContaining("Credits: 100"));
            expect(outputMock.info).toHaveBeenCalledWith(expect.stringContaining("Premium: Yes"));
        });

        it("should handle missing user fields gracefully", async () => {
            const minimalSession = {
                ...mockSession,
                users: [{
                    id: "user-123",
                    updatedAt: new Date().toISOString(),
                }],
            };
            vi.mocked(client.get).mockResolvedValue(minimalSession);

            await program.parseAsync(["node", "cli", "auth", "whoami"]);

            expect(outputMock.info).toHaveBeenCalledWith(expect.stringContaining("Handle: (not set)"));
            expect(outputMock.info).toHaveBeenCalledWith(expect.stringContaining("Name: (not set)"));
            expect(outputMock.info).toHaveBeenCalledWith(expect.stringContaining("Languages: none"));
            expect(outputMock.info).toHaveBeenCalledWith(expect.stringContaining("Credits: 0"));
            expect(outputMock.info).toHaveBeenCalledWith(expect.stringContaining("Premium: No"));
        });

        it("should output JSON when JSON mode is enabled", async () => {
            vi.mocked(config.isJsonOutput).mockReturnValue(true);
            vi.mocked(client.get).mockResolvedValue(mockSession);

            await program.parseAsync(["node", "cli", "auth", "whoami"]);

            const expectedUser = mockSession.users![0];
            expect(outputMock.json).toHaveBeenCalledWith(expectedUser);
        });

        it("should handle authentication errors", async () => {
            vi.mocked(client.get).mockRejectedValue(new Error("401 Unauthorized"));

            await expect(program.parseAsync(["node", "cli", "auth", "whoami"]))
                .rejects.toThrow("Process exited with code 1");

            expect(outputMock.error).toHaveBeenCalledWith("✗ Not authenticated. Run 'vrooli auth login' first.");
        });

        it("should handle other errors", async () => {
            vi.mocked(client.get).mockRejectedValue(new Error("Network error"));

            await expect(program.parseAsync(["node", "cli", "auth", "whoami"]))
                .rejects.toThrow("Process exited with code 1");

            expect(outputMock.error).toHaveBeenCalledWith("✗ Failed to get user info: Network error");
        });

        it("should handle missing user data", async () => {
            const emptySession = { ...mockSession, users: [] };
            vi.mocked(client.get).mockResolvedValue(emptySession);

            await program.parseAsync(["node", "cli", "auth", "whoami"]);

            expect(outputMock.warn).toHaveBeenCalledWith("No user data found");
        });
    });

    describe("reset-password command", () => {
        it("should register reset-password command", () => {
            const authCmd = program.commands.find(cmd => cmd.name() === "auth");
            const subCommands = authCmd?.commands.map(cmd => cmd.name());
            expect(subCommands).toContain("reset-password");
        });

        it("should reset password successfully with provided email", async () => {
            vi.mocked(client.post)
                .mockResolvedValueOnce({}) // requestPasswordChange response
                .mockResolvedValueOnce({}); // resetPassword response

            // Mock prompt responses for reset code and new password
            vi.mocked(inquirer.prompt)
                .mockResolvedValueOnce({
                    code: "reset123",
                    newPassword: "newPassword123",
                    confirmPassword: "newPassword123",
                });

            await program.parseAsync(["node", "cli", "auth", "reset-password", "-e", "test@example.com"]);

            expect(vi.mocked(client.post)).toHaveBeenNthCalledWith(1, "/auth/email/requestPasswordChange", {
                email: "test@example.com",
            });
            expect(vi.mocked(client.post)).toHaveBeenNthCalledWith(2, "/auth/email/resetPassword", {
                code: "reset123",
                newPassword: "newPassword123",
            });
            expect(outputMock.success).toHaveBeenCalledWith(expect.stringContaining("✓ Password reset email sent"));
            expect(outputMock.success).toHaveBeenCalledWith(expect.stringContaining("✓ Password reset successfully"));
        });

        it("should prompt for email when not provided", async () => {
            vi.mocked(inquirer.prompt)
                .mockResolvedValueOnce({ email: "prompt@example.com" })
                .mockResolvedValueOnce({
                    code: "reset123",
                    newPassword: "newPassword123",
                    confirmPassword: "newPassword123",
                });

            vi.mocked(client.post)
                .mockResolvedValueOnce({})
                .mockResolvedValueOnce({});

            await program.parseAsync(["node", "cli", "auth", "reset-password"]);

            expect(vi.mocked(inquirer.prompt)).toHaveBeenCalledWith([{
                type: "input",
                name: "email",
                message: "Email address:",
                validate: expect.any(Function),
            }]);
            expect(vi.mocked(client.post)).toHaveBeenNthCalledWith(1, "/auth/email/requestPasswordChange", {
                email: "prompt@example.com",
            });
        });

        it("should validate email format in prompt", async () => {
            vi.mocked(inquirer.prompt)
                .mockResolvedValueOnce({ email: "test@example.com" })
                .mockResolvedValueOnce({
                    code: "reset123",
                    newPassword: "newPassword123",
                    confirmPassword: "newPassword123",
                });

            vi.mocked(client.post)
                .mockResolvedValueOnce({})
                .mockResolvedValueOnce({});

            await program.parseAsync(["node", "cli", "auth", "reset-password"]);

            const emailPromptConfig = vi.mocked(inquirer.prompt).mock.calls[0][0] as any;
            const emailValidation = emailPromptConfig[0].validate;

            expect(emailValidation("invalid-email")).toBe("Please enter a valid email address");
            expect(emailValidation("valid@email.com")).toBe(true);
        });

        it("should validate password requirements and confirmation", async () => {
            vi.mocked(inquirer.prompt)
                .mockResolvedValueOnce({ email: "test@example.com" })
                .mockResolvedValueOnce({
                    code: "reset123",
                    newPassword: "newPassword123",
                    confirmPassword: "newPassword123",
                });

            vi.mocked(client.post)
                .mockResolvedValueOnce({})
                .mockResolvedValueOnce({});

            await program.parseAsync(["node", "cli", "auth", "reset-password"]);

            const resetPromptConfig = vi.mocked(inquirer.prompt).mock.calls[1][0] as any;
            
            // Test password validation
            const passwordValidation = resetPromptConfig[1].validate;
            expect(passwordValidation("short")).toContain("Password must be at least");
            expect(passwordValidation("longEnoughPassword")).toBe(true);

            // Test password confirmation validation
            const confirmValidation = resetPromptConfig[2].validate;
            expect(confirmValidation("different", { newPassword: "original" })).toBe("Passwords do not match");
            expect(confirmValidation("same", { newPassword: "same" })).toBe(true);
        });

        it("should output JSON when JSON mode is enabled", async () => {
            vi.mocked(config.isJsonOutput).mockReturnValue(true);
            vi.mocked(client.post)
                .mockResolvedValueOnce({})
                .mockResolvedValueOnce({});

            vi.mocked(inquirer.prompt)
                .mockResolvedValueOnce({
                    code: "reset123",
                    newPassword: "newPassword123",
                    confirmPassword: "newPassword123",
                });

            await program.parseAsync(["node", "cli", "auth", "reset-password", "-e", "test@example.com"]);

            expect(outputMock.json).toHaveBeenCalledWith({ success: true });
        });

        it("should handle password reset request failure", async () => {
            vi.mocked(client.post).mockRejectedValue(new Error("Email not found"));

            await expect(program.parseAsync(["node", "cli", "auth", "reset-password", "-e", "test@example.com"]))
                .rejects.toThrow("Process exited with code 1");

            expect(outputMock.error).toHaveBeenCalledWith(expect.stringContaining("✗ Password reset failed: Email not found"));
        });

        it("should handle password reset completion failure", async () => {
            vi.mocked(client.post)
                .mockResolvedValueOnce({}) // requestPasswordChange succeeds
                .mockRejectedValue(new Error("Invalid reset code")); // resetPassword fails

            vi.mocked(inquirer.prompt)
                .mockResolvedValueOnce({
                    code: "invalid123",
                    newPassword: "newPassword123",
                    confirmPassword: "newPassword123",
                });

            await expect(program.parseAsync(["node", "cli", "auth", "reset-password", "-e", "test@example.com"]))
                .rejects.toThrow("Process exited with code 1");

            expect(outputMock.error).toHaveBeenCalledWith(expect.stringContaining("✗ Password reset failed: Invalid reset code"));
        });

        it("should handle missing email error", async () => {
            vi.mocked(inquirer.prompt).mockResolvedValueOnce({ email: "" });

            await expect(program.parseAsync(["node", "cli", "auth", "reset-password"]))
                .rejects.toThrow("Process exited with code 1");

            expect(outputMock.error).toHaveBeenCalledWith(expect.stringContaining("✗ Password reset failed: Email is required"));
        });
    });

    describe("verify-email command", () => {
        it("should register verify-email command", () => {
            const authCmd = program.commands.find(cmd => cmd.name() === "auth");
            const subCommands = authCmd?.commands.map(cmd => cmd.name());
            expect(subCommands).toContain("verify-email");
        });

        it("should verify email successfully with provided code", async () => {
            vi.mocked(client.post).mockResolvedValue({});

            await program.parseAsync(["node", "cli", "auth", "verify-email", "-c", "verify123"]);

            expect(vi.mocked(client.post)).toHaveBeenCalledWith("/email/verify", {
                code: "verify123",
            });
            expect(outputMock.success).toHaveBeenCalledWith(expect.stringContaining("✓ Email verified successfully"));
            expect(outputMock.info).toHaveBeenCalledWith(expect.stringContaining("Your email is now verified"));
        });

        it("should prompt for code when not provided", async () => {
            vi.mocked(inquirer.prompt).mockResolvedValueOnce({ code: "prompt123" });
            vi.mocked(client.post).mockResolvedValue({});

            await program.parseAsync(["node", "cli", "auth", "verify-email"]);

            expect(vi.mocked(inquirer.prompt)).toHaveBeenCalledWith([{
                type: "input",
                name: "code",
                message: "Enter the verification code from your email:",
                validate: expect.any(Function),
            }]);
            expect(vi.mocked(client.post)).toHaveBeenCalledWith("/email/verify", {
                code: "prompt123",
            });
        });

        it("should validate code in prompt", async () => {
            vi.mocked(inquirer.prompt).mockResolvedValueOnce({ code: "verify123" });
            vi.mocked(client.post).mockResolvedValue({});

            await program.parseAsync(["node", "cli", "auth", "verify-email"]);

            const promptConfig = vi.mocked(inquirer.prompt).mock.calls[0][0] as any;
            const codeValidation = promptConfig[0].validate;

            expect(codeValidation("")).toBe("Verification code is required");
            expect(codeValidation("valid-code")).toBe(true);
        });

        it("should output JSON when JSON mode is enabled", async () => {
            vi.mocked(config.isJsonOutput).mockReturnValue(true);
            vi.mocked(client.post).mockResolvedValue({});

            await program.parseAsync(["node", "cli", "auth", "verify-email", "-c", "verify123"]);

            expect(outputMock.json).toHaveBeenCalledWith({ success: true });
        });

        it("should handle verification failure", async () => {
            vi.mocked(client.post).mockRejectedValue(new Error("Invalid verification code"));

            await expect(program.parseAsync(["node", "cli", "auth", "verify-email", "-c", "invalid123"]))
                .rejects.toThrow("Process exited with code 1");

            expect(outputMock.error).toHaveBeenCalledWith(expect.stringContaining("✗ Email verification failed: Invalid verification code"));
            expect(outputMock.info).toHaveBeenCalledWith(expect.stringContaining("Make sure you have the correct verification code"));
            expect(outputMock.info).toHaveBeenCalledWith(expect.stringContaining("Codes typically expire after a few minutes"));
        });

        it("should handle verification failure in JSON mode", async () => {
            vi.mocked(config.isJsonOutput).mockReturnValue(true);
            vi.mocked(client.post).mockRejectedValue(new Error("Invalid code"));

            await expect(program.parseAsync(["node", "cli", "auth", "verify-email", "-c", "invalid123"]))
                .rejects.toThrow("Process exited with code 1");

            expect(outputMock.json).toHaveBeenCalledWith({
                success: false,
                error: "Invalid code",
            });
        });
    });
});
