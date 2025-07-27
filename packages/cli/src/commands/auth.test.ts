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
vi.mock("chalk", () => ({
    default: {
        red: vi.fn((str: string) => str),
        green: vi.fn((str: string) => str),
        yellow: vi.fn((str: string) => str),
        gray: vi.fn((str: string) => str),
        bold: vi.fn((str: string) => str),
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
    let consoleLogSpy: any;
    let consoleErrorSpy: any;
    let processExitSpy: any;

    beforeEach(() => {
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

        // Setup console spies
        consoleLogSpy = vi.spyOn(console, "log").mockImplementation(() => {
            // Mock implementation
        });
        consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {
            // Mock implementation
        });
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
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("✓ Authentication successful"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("User: Test User"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Handle: testuser"));
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

            expect(consoleLogSpy).toHaveBeenCalledWith(JSON.stringify({
                success: true,
                user: {
                    id: "user-123",
                    handle: "testuser",
                    name: "Test User",
                },
            }));
        });

        it("should handle login errors", async () => {
            vi.mocked(client.post).mockRejectedValue(new Error("Invalid credentials"));

            await expect(program.parseAsync(["node", "cli", "auth", "login", "-e", "test@example.com", "-p", "wrong"]))
                .rejects.toThrow("Process exited with code 1");

            expect(consoleErrorSpy).toHaveBeenCalledWith(expect.stringContaining("✗ Login failed: Invalid credentials"));
        });

        it("should handle missing user data in response", async () => {
            const badSession = { ...mockSession, users: [] };
            vi.mocked(client.post).mockResolvedValue(badSession);

            await expect(program.parseAsync(["node", "cli", "auth", "login", "-e", "test@example.com", "-p", "password123"]))
                .rejects.toThrow("Process exited with code 1");

            expect(consoleErrorSpy).toHaveBeenCalledWith(expect.stringContaining("No user data in login response"));
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
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("✓ Credentials cleared"));
        });

        it("should clear auth even if logout API fails", async () => {
            vi.mocked(config.getAuthToken).mockReturnValue("auth-token");
            vi.mocked(client.post).mockRejectedValue(new Error("Network error"));

            await program.parseAsync(["node", "cli", "auth", "logout"]);

            expect(vi.mocked(config.clearAuth)).toHaveBeenCalled();
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("✓ Credentials cleared"));
        });

        it("should logout successfully when not authenticated", async () => {
            vi.mocked(config.getAuthToken).mockReturnValue(null);

            await program.parseAsync(["node", "cli", "auth", "logout"]);

            expect(vi.mocked(client.post)).not.toHaveBeenCalled();
            expect(vi.mocked(config.clearAuth)).toHaveBeenCalled();
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("✓ Credentials cleared"));
        });

        it("should output JSON when JSON mode is enabled", async () => {
            vi.mocked(config.isJsonOutput).mockReturnValue(true);
            vi.mocked(config.getAuthToken).mockReturnValue(null);

            await program.parseAsync(["node", "cli", "auth", "logout"]);

            expect(consoleLogSpy).toHaveBeenCalledWith(JSON.stringify({ success: true }));
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
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("✓ Authenticated"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("User: Test User"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Handle: testuser"));
        });

        it("should show not authenticated when no token", async () => {
            vi.mocked(config.getAuthToken).mockReturnValue(null);

            await program.parseAsync(["node", "cli", "auth", "status"]);

            expect(vi.mocked(client.get)).not.toHaveBeenCalled();
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("⚠ Not authenticated"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Profile: default"));
        });

        it("should handle invalid token and clear auth", async () => {
            vi.mocked(config.getAuthToken).mockReturnValue("invalid-token");
            vi.mocked(client.get).mockRejectedValue(new Error("401 Unauthorized"));

            await program.parseAsync(["node", "cli", "auth", "status"]);

            expect(vi.mocked(config.clearAuth)).toHaveBeenCalled();
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("✗ Authentication invalid"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Run 'vrooli auth login' to authenticate"));
        });

        it("should output JSON for authenticated status", async () => {
            vi.mocked(config.isJsonOutput).mockReturnValue(true);
            vi.mocked(config.getAuthToken).mockReturnValue("valid-token");
            vi.mocked(client.get).mockResolvedValue(mockSession);

            await program.parseAsync(["node", "cli", "auth", "status"]);

            expect(consoleLogSpy).toHaveBeenCalledWith(JSON.stringify({
                authenticated: true,
                profile: "default",
                user: {
                    id: "user-123",
                    handle: "testuser",
                    name: "Test User",
                },
            }));
        });

        it("should output JSON for not authenticated status", async () => {
            vi.mocked(config.isJsonOutput).mockReturnValue(true);
            vi.mocked(config.getAuthToken).mockReturnValue(null);

            await program.parseAsync(["node", "cli", "auth", "status"]);

            expect(consoleLogSpy).toHaveBeenCalledWith(JSON.stringify({
                authenticated: false,
                profile: "default",
            }));
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
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("User Information:"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("ID: user-123"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Handle: testuser"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Name: Test User"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Languages: en, es"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Credits: 100"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Premium: Yes"));
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

            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Handle: (not set)"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Name: (not set)"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Languages: none"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Credits: 0"));
            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("Premium: No"));
        });

        it("should output JSON when JSON mode is enabled", async () => {
            vi.mocked(config.isJsonOutput).mockReturnValue(true);
            vi.mocked(client.get).mockResolvedValue(mockSession);

            await program.parseAsync(["node", "cli", "auth", "whoami"]);

            const expectedUser = mockSession.users![0];
            expect(consoleLogSpy).toHaveBeenCalledWith(JSON.stringify(expectedUser));
        });

        it("should handle authentication errors", async () => {
            vi.mocked(client.get).mockRejectedValue(new Error("401 Unauthorized"));

            await expect(program.parseAsync(["node", "cli", "auth", "whoami"]))
                .rejects.toThrow("Process exited with code 1");

            expect(consoleErrorSpy).toHaveBeenCalledWith(expect.stringContaining("✗ Not authenticated. Run 'vrooli auth login' first."));
        });

        it("should handle other errors", async () => {
            vi.mocked(client.get).mockRejectedValue(new Error("Network error"));

            await expect(program.parseAsync(["node", "cli", "auth", "whoami"]))
                .rejects.toThrow("Process exited with code 1");

            expect(consoleErrorSpy).toHaveBeenCalledWith(expect.stringContaining("✗ Failed to get user info: Network error"));
        });

        it("should handle missing user data", async () => {
            const emptySession = { ...mockSession, users: [] };
            vi.mocked(client.get).mockResolvedValue(emptySession);

            await program.parseAsync(["node", "cli", "auth", "whoami"]);

            expect(consoleLogSpy).toHaveBeenCalledWith(expect.stringContaining("No user data found"));
        });
    });
});
