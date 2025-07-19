import { Command } from "commander";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

describe("CLI Integration", () => {
    let consoleLogSpy: any;
    let consoleErrorSpy: any;

    beforeEach(() => {
        consoleLogSpy = vi.spyOn(console, "log").mockImplementation(() => undefined);
        consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => undefined);
    });

    afterEach(() => {
        consoleLogSpy.mockRestore();
        consoleErrorSpy.mockRestore();
        vi.clearAllMocks();
    });

    it("should create a valid Commander program", () => {
        const program = new Command();
        program
            .name("vrooli")
            .description("Vrooli CLI - Manage your Vrooli instance from the command line")
            .version("1.0.0");

        expect(program.name()).toBe("vrooli");
        expect(program.description()).toBe("Vrooli CLI - Manage your Vrooli instance from the command line");
        expect(program.version()).toBe("1.0.0");
    });

    it("should register global options", () => {
        const program = new Command();
        program
            .option("-p, --profile <profile>", "Use a specific profile", "default")
            .option("-d, --debug", "Enable debug output")
            .option("--json", "Output in JSON format");

        const options = program.opts();
        expect(options.profile).toBe("default");
        expect(options.debug).toBeUndefined();
        expect(options.json).toBeUndefined();
    });

    it("should handle profile management commands", () => {
        const program = new Command();
        const profileCmd = program
            .command("profile")
            .description("Manage CLI profiles");

        profileCmd
            .command("list")
            .description("List all profiles");

        profileCmd
            .command("use <profile>")
            .description("Switch to a different profile");

        profileCmd
            .command("create <profile>")
            .description("Create a new profile");

        const commands = program.commands.map(cmd => cmd.name());
        expect(commands).toContain("profile");

        const profileSubcommands = profileCmd.commands.map(cmd => cmd.name());
        expect(profileSubcommands).toContain("list");
        expect(profileSubcommands).toContain("use");
        expect(profileSubcommands).toContain("create");
    });

    it("should register command modules", () => {
        const program = new Command();

        // Simulate registering commands
        program.command("auth").description("Authentication commands");
        program.command("routine").description("Manage routines");
        program.command("chat").description("Chat commands for interacting with bots");

        const commands = program.commands.map(cmd => cmd.name());
        expect(commands).toContain("auth");
        expect(commands).toContain("routine");
        expect(commands).toContain("chat");
    });

    it("should handle command errors gracefully", async () => {
        const program = new Command();
        program.exitOverride(); // Prevent process.exit

        try {
            await program.parseAsync(["node", "test", "invalid-command"]);
        } catch (error: any) {
            expect(error.code).toBe("commander.unknownCommand");
        }
    });

    it("should support help command", () => {
        const program = new Command();
        program
            .name("vrooli")
            .description("Test CLI");

        const helpInfo = program.helpInformation();
        expect(helpInfo).toContain("vrooli");
        expect(helpInfo).toContain("Test CLI");
    });
});
