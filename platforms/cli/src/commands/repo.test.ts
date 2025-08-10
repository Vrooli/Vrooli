import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { Command } from "commander";
import { RepoCommands } from "./repo.js";
import { type ApiClient } from "../utils/client.js";
import { type ConfigManager } from "../utils/config.js";
import { logger } from "../utils/logger.js";
import fs from "fs/promises";
import path from "path";

// Create hoisted mocks
const mockExecAsync = vi.hoisted(() => vi.fn());

// Mock dependencies with proper hoisting
vi.mock("../utils/client.js");
vi.mock("../utils/config.js");
vi.mock("../utils/logger.js");
vi.mock("fs/promises");
vi.mock("child_process", () => ({
    exec: vi.fn(),
}));
vi.mock("util", () => ({
    promisify: vi.fn().mockReturnValue(mockExecAsync),
}));

describe("RepoCommands", () => {
    let program: Command;
    let mockClient: ApiClient;
    let mockConfig: ConfigManager;
    let repoCommands: RepoCommands;
    let consoleSpy: {
        log: ReturnType<typeof vi.spyOn>;
        error: ReturnType<typeof vi.spyOn>;
    };

    const mockRepositoryConfig = {
        type: "git",
        url: "https://github.com/example/repo.git",
        sshUrl: "git@github.com:example/repo.git",
        directory: "repo",
        branch: "main",
        private: false,
        mirrors: ["https://gitlab.com/example/repo.git"],
        submodules: {
            enabled: true,
            recursive: true,
            shallow: false,
        },
        hooks: {
            postClone: "echo 'Post clone hook'",
            preBuild: "echo 'Pre build hook'",
            postUpdate: "echo 'Post update hook'",
        },
    };

    beforeEach(() => {
        program = new Command();
        mockClient = {} as ApiClient;
        mockConfig = {} as ConfigManager;
        
        consoleSpy = {
            log: vi.spyOn(console, "log").mockImplementation(() => { /* mock implementation */ }),
            error: vi.spyOn(console, "error").mockImplementation(() => { /* mock implementation */ }),
        };

        vi.clearAllMocks();
        
        // Setup mocks
        vi.mocked(logger.error).mockImplementation(() => { /* mock implementation */ });
        vi.mocked(logger.warn).mockImplementation(() => { /* mock implementation */ });
        vi.mocked(logger.info).mockImplementation(() => { /* mock implementation */ });
        vi.mocked(logger.debug).mockImplementation(() => { /* mock implementation */ });
        vi.mocked(logger.success).mockImplementation(() => { /* mock implementation */ });
        
        // Reset process.exit mock
        vi.spyOn(process, "exit").mockImplementation(() => {
            throw new Error("process.exit called");
        });
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    describe("Constructor and Command Registration", () => {
        it("should register repo commands correctly", () => {
            repoCommands = new RepoCommands(program, mockClient, mockConfig);
            
            // Verify main repo command was added
            const repoCommand = program.commands.find(cmd => cmd.name() === "repo");
            expect(repoCommand).toBeDefined();
            expect(repoCommand?.description()).toBe("Repository management commands");
            
            // Verify subcommands were added
            const subcommands = repoCommand?.commands.map(cmd => cmd.name()) || [];
            expect(subcommands).toContain("info");
            expect(subcommands).toContain("open");
            expect(subcommands).toContain("clone");
            expect(subcommands).toContain("mirrors");
            expect(subcommands).toContain("hooks");
        });

        it("should configure info command with correct options", () => {
            repoCommands = new RepoCommands(program, mockClient, mockConfig);
            const repoCommand = program.commands.find(cmd => cmd.name() === "repo");
            const infoCommand = repoCommand?.commands.find(cmd => cmd.name() === "info");
            
            expect(infoCommand?.description()).toBe("Display repository information");
            expect(infoCommand?.options.some(opt => opt.long === "--json")).toBe(true);
        });

        it("should configure mirrors command with correct options", () => {
            repoCommands = new RepoCommands(program, mockClient, mockConfig);
            const repoCommand = program.commands.find(cmd => cmd.name() === "repo");
            const mirrorsCommand = repoCommand?.commands.find(cmd => cmd.name() === "mirrors");
            
            expect(mirrorsCommand?.description()).toBe("List available mirror repositories");
            expect(mirrorsCommand?.options.some(opt => opt.long === "--json")).toBe(true);
        });

        it("should configure hooks command with correct options", () => {
            repoCommands = new RepoCommands(program, mockClient, mockConfig);
            const repoCommand = program.commands.find(cmd => cmd.name() === "repo");
            const hooksCommand = repoCommand?.commands.find(cmd => cmd.name() === "hooks");
            
            expect(hooksCommand?.description()).toBe("List and run repository hooks");
            expect(hooksCommand?.options.some(opt => opt.long === "--json")).toBe(true);
        });
    });

    describe("Service Configuration", () => {
        it("should read service config from .vrooli/service.json in cwd", async () => {
            const mockServiceConfig = {
                service: {
                    repository: mockRepositoryConfig,
                },
            };

            vi.mocked(fs.readFile).mockResolvedValueOnce(JSON.stringify(mockServiceConfig));
            
            repoCommands = new RepoCommands(program, mockClient, mockConfig);
            
            // Access private method through any type casting
            const config = await (repoCommands as any).getServiceConfig();
            
            expect(config).toEqual(mockServiceConfig.service);
            expect(fs.readFile).toHaveBeenCalledWith(
                path.join(process.cwd(), ".vrooli", "service.json"),
                "utf-8",
            );
        });

        it("should fallback to home directory config if cwd config not found", async () => {
            const mockServiceConfig = {
                service: {
                    repository: mockRepositoryConfig,
                },
            };

            vi.mocked(fs.readFile)
                .mockRejectedValueOnce(new Error("File not found"))
                .mockResolvedValueOnce(JSON.stringify(mockServiceConfig));
            
            repoCommands = new RepoCommands(program, mockClient, mockConfig);
            const config = await (repoCommands as any).getServiceConfig();
            
            expect(config).toEqual(mockServiceConfig.service);
            expect(fs.readFile).toHaveBeenCalledWith(
                path.join(process.env.HOME || "", ".vrooli", "service.json"),
                "utf-8",
            );
        });

        it("should return null if no config found", async () => {
            vi.mocked(fs.readFile).mockRejectedValue(new Error("File not found"));
            
            repoCommands = new RepoCommands(program, mockClient, mockConfig);
            const config = await (repoCommands as any).getServiceConfig();
            
            expect(config).toBeNull();
        });

        it("should handle JSON parse errors gracefully", async () => {
            vi.mocked(fs.readFile).mockResolvedValue("invalid json");
            
            repoCommands = new RepoCommands(program, mockClient, mockConfig);
            const config = await (repoCommands as any).getServiceConfig();
            
            expect(config).toBeNull();
        });

        it("should log error and return null on exception", async () => {
            // Mock process.cwd to throw an error to trigger the outer try-catch
            vi.spyOn(process, "cwd").mockImplementation(() => {
                throw new Error("Unexpected error");
            });
            
            repoCommands = new RepoCommands(program, mockClient, mockConfig);
            const config = await (repoCommands as any).getServiceConfig();
            
            expect(config).toBeNull();
            expect(logger.error).toHaveBeenCalledWith(
                "Failed to read service configuration",
                expect.any(Error),
            );
        });
    });

    describe("Repository Information Display", () => {
        beforeEach(() => {
            repoCommands = new RepoCommands(program, mockClient, mockConfig);
        });

        it("should format repository info as JSON when requested", () => {
            (repoCommands as any).formatRepositoryInfo(mockRepositoryConfig, "json");
            
            expect(consoleSpy.log).toHaveBeenCalledWith(
                JSON.stringify(mockRepositoryConfig, null, 2),
            );
        });

        it("should format repository info as table by default", () => {
            (repoCommands as any).formatRepositoryInfo(mockRepositoryConfig, "table");
            
            expect(consoleSpy.log).toHaveBeenCalledWith("Repository Information:");
            expect(consoleSpy.log).toHaveBeenCalledWith("─".repeat(50));
            expect(consoleSpy.log).toHaveBeenCalledWith(`Type:      ${mockRepositoryConfig.type}`);
            expect(consoleSpy.log).toHaveBeenCalledWith(`URL:       ${mockRepositoryConfig.url}`);
            expect(consoleSpy.log).toHaveBeenCalledWith(`SSH URL:   ${mockRepositoryConfig.sshUrl}`);
        });

        it("should display mirrors when available", () => {
            (repoCommands as any).formatRepositoryInfo(mockRepositoryConfig, "table");
            
            expect(consoleSpy.log).toHaveBeenCalledWith("\nMirrors:");
            expect(consoleSpy.log).toHaveBeenCalledWith("  1. https://gitlab.com/example/repo.git");
        });

        it("should display hooks when available", () => {
            (repoCommands as any).formatRepositoryInfo(mockRepositoryConfig, "table");
            
            expect(consoleSpy.log).toHaveBeenCalledWith("\nHooks:");
            expect(consoleSpy.log).toHaveBeenCalledWith("  Post-Clone:  echo 'Post clone hook'");
            expect(consoleSpy.log).toHaveBeenCalledWith("  Pre-Build:   echo 'Pre build hook'");
            expect(consoleSpy.log).toHaveBeenCalledWith("  Post-Update: echo 'Post update hook'");
        });

        it("should display submodules configuration", () => {
            (repoCommands as any).formatRepositoryInfo(mockRepositoryConfig, "table");
            
            expect(consoleSpy.log).toHaveBeenCalledWith(
                "\nSubmodules: Enabled (recursive: true, shallow: false)",
            );
        });

        it("should handle boolean submodules configuration", () => {
            const configWithBooleanSubmodules = {
                ...mockRepositoryConfig,
                submodules: true,
            };
            
            (repoCommands as any).formatRepositoryInfo(configWithBooleanSubmodules, "table");
            
            expect(consoleSpy.log).toHaveBeenCalledWith("\nSubmodules: Enabled");
        });
    });

    describe("Repository Browser Opening", () => {
        beforeEach(() => {
            repoCommands = new RepoCommands(program, mockClient, mockConfig);
        });

        it("should open repository in browser on macOS", async () => {
            Object.defineProperty(process, "platform", { value: "darwin" });
            mockExecAsync.mockResolvedValue({ stdout: "", stderr: "" });
            
            await (repoCommands as any).openRepositoryInBrowser("https://github.com/test/repo");
            
            expect(mockExecAsync).toHaveBeenCalledWith("open \"https://github.com/test/repo\"");
            expect(consoleSpy.log).toHaveBeenCalledWith("Opened repository in browser: https://github.com/test/repo");
        });

        it("should open repository in browser on Windows", async () => {
            Object.defineProperty(process, "platform", { value: "win32" });
            mockExecAsync.mockResolvedValue({ stdout: "", stderr: "" });
            
            await (repoCommands as any).openRepositoryInBrowser("https://github.com/test/repo");
            
            expect(mockExecAsync).toHaveBeenCalledWith("start \"https://github.com/test/repo\"");
        });

        it("should open repository in browser on Linux", async () => {
            Object.defineProperty(process, "platform", { value: "linux" });
            mockExecAsync.mockResolvedValue({ stdout: "", stderr: "" });
            
            await (repoCommands as any).openRepositoryInBrowser("https://github.com/test/repo");
            
            expect(mockExecAsync).toHaveBeenCalledWith("xdg-open \"https://github.com/test/repo\"");
        });

        it("should handle browser open errors gracefully", async () => {
            mockExecAsync.mockRejectedValue(new Error("Command failed"));
            
            await (repoCommands as any).openRepositoryInBrowser("https://github.com/test/repo");
            
            expect(consoleSpy.error).toHaveBeenCalledWith("Failed to open repository in browser: Error: Command failed");
            expect(consoleSpy.log).toHaveBeenCalledWith("Please open manually: https://github.com/test/repo");
        });
    });

    describe("Repository Cloning", () => {
        beforeEach(() => {
            repoCommands = new RepoCommands(program, mockClient, mockConfig);
        });

        it("should clone repository with basic configuration", async () => {
            const basicConfig = {
                type: "git",
                url: "https://github.com/test/repo.git",
                branch: "main",
            };
            
            mockExecAsync.mockResolvedValue({ stdout: "Cloning...", stderr: "" });
            
            await (repoCommands as any).cloneRepositoryWithConfig(basicConfig);
            
            expect(mockExecAsync).toHaveBeenCalledWith("git clone \"https://github.com/test/repo.git\"");
            expect(consoleSpy.log).toHaveBeenCalledWith("Repository cloned successfully!");
        });

        it("should use SSH URL when available", async () => {
            mockExecAsync.mockResolvedValue({ stdout: "Cloning...", stderr: "" });
            
            await (repoCommands as any).cloneRepositoryWithConfig(mockRepositoryConfig);
            
            expect(mockExecAsync).toHaveBeenCalledWith(
                expect.stringContaining("\"git@github.com:example/repo.git\""),
            );
        });

        it("should include branch when not main/master", async () => {
            const configWithBranch = {
                ...mockRepositoryConfig,
                branch: "develop",
            };
            
            mockExecAsync.mockResolvedValue({ stdout: "Cloning...", stderr: "" });
            
            await (repoCommands as any).cloneRepositoryWithConfig(configWithBranch);
            
            expect(mockExecAsync).toHaveBeenCalledWith(
                expect.stringContaining("--branch develop"),
            );
        });

        it("should include submodules when enabled", async () => {
            mockExecAsync.mockResolvedValue({ stdout: "Cloning...", stderr: "" });
            
            await (repoCommands as any).cloneRepositoryWithConfig(mockRepositoryConfig);
            
            expect(mockExecAsync).toHaveBeenCalledWith(
                expect.stringContaining("--recurse-submodules"),
            );
        });

        it("should include shallow submodules when configured", async () => {
            const configWithShallowSubmodules = {
                ...mockRepositoryConfig,
                submodules: {
                    enabled: true,
                    recursive: true,
                    shallow: true,
                },
            };
            
            mockExecAsync.mockResolvedValue({ stdout: "Cloning...", stderr: "" });
            
            await (repoCommands as any).cloneRepositoryWithConfig(configWithShallowSubmodules);
            
            expect(mockExecAsync).toHaveBeenCalledWith(
                expect.stringContaining("--shallow-submodules"),
            );
        });

        it("should execute post-clone hook when configured", async () => {
            mockExecAsync
                .mockResolvedValueOnce({ stdout: "Cloning...", stderr: "" })
                .mockResolvedValueOnce({ stdout: "Hook executed", stderr: "" });
            
            await (repoCommands as any).cloneRepositoryWithConfig(mockRepositoryConfig);
            
            expect(mockExecAsync).toHaveBeenCalledWith("echo 'Post clone hook'");
            expect(consoleSpy.log).toHaveBeenCalledWith("\nExecuting post-clone hook...");
        });

        it("should handle clone errors gracefully", async () => {
            mockExecAsync.mockRejectedValue(new Error("Clone failed"));
            
            await (repoCommands as any).cloneRepositoryWithConfig(mockRepositoryConfig);
            
            expect(consoleSpy.error).toHaveBeenCalledWith("Failed to clone repository: Error: Clone failed");
        });
    });

    describe("Command Actions", () => {
        beforeEach(() => {
            repoCommands = new RepoCommands(program, mockClient, mockConfig);
            vi.spyOn(repoCommands as any, "getServiceConfig").mockResolvedValue({
                repository: mockRepositoryConfig,
            });
        });

        it("should handle info command with table format", async () => {
            vi.spyOn(repoCommands as any, "formatRepositoryInfo").mockImplementation(() => { /* mock implementation */ });
            
            await (repoCommands as any).showInfo({ json: false });
            
            expect((repoCommands as any).formatRepositoryInfo).toHaveBeenCalledWith(
                mockRepositoryConfig,
                "table",
            );
        });

        it("should handle info command with JSON format", async () => {
            vi.spyOn(repoCommands as any, "formatRepositoryInfo").mockImplementation(() => { /* mock implementation */ });
            
            await (repoCommands as any).showInfo({ json: true });
            
            expect((repoCommands as any).formatRepositoryInfo).toHaveBeenCalledWith(
                mockRepositoryConfig,
                "json",
            );
        });

        it("should handle missing repository config in info command", async () => {
            vi.spyOn(repoCommands as any, "getServiceConfig").mockResolvedValue(null);
            
            await expect(async () => {
                await (repoCommands as any).showInfo({});
            }).rejects.toThrow("process.exit called");
            
            expect(consoleSpy.error).toHaveBeenCalledWith(
                expect.stringContaining("No repository configuration found"),
            );
        });

        it("should handle open repository command", async () => {
            vi.spyOn(repoCommands as any, "openRepositoryInBrowser").mockResolvedValue(undefined);
            
            await (repoCommands as any).openRepository();
            
            expect((repoCommands as any).openRepositoryInBrowser).toHaveBeenCalledWith(
                mockRepositoryConfig.url,
            );
        });

        it("should handle clone repository command", async () => {
            vi.spyOn(repoCommands as any, "cloneRepositoryWithConfig").mockResolvedValue(undefined);
            
            await (repoCommands as any).cloneRepository();
            
            expect((repoCommands as any).cloneRepositoryWithConfig).toHaveBeenCalledWith(
                mockRepositoryConfig,
            );
        });

        it("should show mirrors in JSON format", async () => {
            await (repoCommands as any).showMirrors({ json: true });
            
            expect(consoleSpy.log).toHaveBeenCalledWith(
                JSON.stringify(mockRepositoryConfig.mirrors, null, 2),
            );
        });

        it("should show mirrors in table format", async () => {
            await (repoCommands as any).showMirrors({ json: false });
            
            expect(consoleSpy.log).toHaveBeenCalledWith(expect.stringContaining("Mirror Repositories:"));
            expect(consoleSpy.log).toHaveBeenCalledWith("1. https://gitlab.com/example/repo.git");
        });

        it("should handle no mirrors configured", async () => {
            vi.spyOn(repoCommands as any, "getServiceConfig").mockResolvedValue({
                repository: { ...mockRepositoryConfig, mirrors: [] },
            });
            
            await (repoCommands as any).showMirrors({ json: false });
            
            expect(consoleSpy.log).toHaveBeenCalledWith(
                expect.stringContaining("No mirror repositories configured"),
            );
        });

        it("should show hooks in JSON format", async () => {
            await (repoCommands as any).showHooks({ json: true });
            
            expect(consoleSpy.log).toHaveBeenCalledWith(
                JSON.stringify(mockRepositoryConfig.hooks, null, 2),
            );
        });

        it("should show hooks in table format", async () => {
            await (repoCommands as any).showHooks({ json: false });
            
            expect(consoleSpy.log).toHaveBeenCalledWith(expect.stringContaining("Repository Hooks:"));
            expect(consoleSpy.log).toHaveBeenCalledWith("Post-Clone:  echo 'Post clone hook'");
        });

        it("should handle no hooks configured", async () => {
            vi.spyOn(repoCommands as any, "getServiceConfig").mockResolvedValue({
                repository: { ...mockRepositoryConfig, hooks: {} },
            });
            
            await (repoCommands as any).showHooks({ json: false });
            
            expect(consoleSpy.log).toHaveBeenCalledWith(
                expect.stringContaining("No hooks configured"),
            );
        });
    });
});
