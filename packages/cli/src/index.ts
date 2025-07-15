#!/usr/bin/env node
// AI_CHECK: TEST_COVERAGE=77 | LAST: 2025-07-13
import { Command } from "commander";
import { ConfigManager } from "./utils/config.js";
import { ApiClient } from "./utils/client.js";
import { AuthCommands } from "./commands/auth.js";
import { RoutineCommands } from "./commands/routine.js";
import { ChatCommands } from "./commands/chat.js";
import { AgentCommands } from "./commands/agent.js";
import { TeamCommands } from "./commands/team.js";
import { HistoryCommands } from "./commands/history.js";
import { CompletionEngine, CompletionInstaller } from "./completion/index.js";
import { HistoryManager } from "./history/HistoryManager.js";
import chalk from "chalk";

// Simple logger for CLI
const logger = {
    error: (message: string, error?: unknown) => {
        console.error(chalk.red(`[ERROR] ${message}`), error || "");
    },
    setLevel: (_level: string) => {
        // CLI logger is simple, no level setting needed
    },
};

async function main() {
    try {
        const program = new Command();
        const config = new ConfigManager();
        const client = new ApiClient(config);
        const historyManager = new HistoryManager(config);

        program
            .name("vrooli")
            .description("Vrooli CLI - Manage your Vrooli instance from the command line")
            .version("1.0.0")
            .option("-p, --profile <profile>", "Use a specific profile", "default")
            .option("-d, --debug", "Enable debug output")
            .option("--json", "Output in JSON format")
            .option("--generate-completions [args...]", "Generate completions (hidden)")
            .hook("preAction", async (thisCommand) => {
                const opts = thisCommand.opts();
                
                // Handle completion generation first
                if (opts.generateCompletions) {
                    try {
                        const completionEngine = new CompletionEngine(config, client);
                        const completions = await completionEngine.getCompletions(opts.generateCompletions);
                        
                        // Output completions for shell to consume
                        completions.forEach(c => {
                            if (c.description) {
                                console.log(`${c.value}:${c.description}`);
                            } else {
                                console.log(c.value);
                            }
                        });
                        process.exit(0);
                    } catch (error) {
                        // Fail silently for completions
                        process.exit(0);
                    }
                }
                
                if (opts.debug) {
                    config.setDebug(true);
                    logger.setLevel("debug");
                }
                if (opts.profile) {
                    config.setActiveProfile(opts.profile);
                }
                if (opts.json) {
                    config.setJsonOutput(true);
                }
                
                // Start history tracking
                try {
                    const command = thisCommand.name();
                    const args = thisCommand.args;
                    await historyManager.startCommand(command, args, opts);
                } catch (error) {
                    // Don't fail the command if history tracking fails
                    console.error(chalk.yellow(`Warning: History tracking failed: ${error instanceof Error ? error.message : String(error)}`));
                }
            })
            .hook("postAction", async (_thisCommand) => {
                // End history tracking
                try {
                    await historyManager.endCommand(0); // Success
                } catch (error) {
                    // Don't fail the command if history tracking fails
                    console.error(chalk.yellow(`Warning: History tracking failed: ${error instanceof Error ? error.message : String(error)}`));
                }
            });

        // Register command modules
        new AuthCommands(program, client, config);
        new RoutineCommands(program, client, config);
        new ChatCommands(program, client, config);
        new AgentCommands(program, client, config);
        new TeamCommands(program, client, config);
        new HistoryCommands(program, client, config);

        // Add profile management commands
        program
            .command("profile")
            .description("Manage CLI profiles")
            .addCommand(
                new Command("list")
                    .description("List all profiles")
                    .action(() => {
                        const profiles = config.listProfiles();
                        const active = config.getActiveProfileName();
                        
                        if (config.isJsonOutput()) {
                            console.log(JSON.stringify({ profiles, active }));
                        } else {
                            console.log(chalk.bold("\nProfiles:"));
                            profiles.forEach(profile => {
                                const marker = profile === active ? chalk.green("*") : " ";
                                console.log(`  ${marker} ${profile}`);
                            });
                        }
                    }),
            )
            .addCommand(
                new Command("use <profile>")
                    .description("Switch to a different profile")
                    .action((profile) => {
                        try {
                            config.setActiveProfile(profile);
                            console.log(chalk.green(`✓ Switched to profile: ${profile}`));
                        } catch (error) {
                            const errorMessage = error instanceof Error ? error.message : String(error);
                            console.error(chalk.red(`✗ ${errorMessage}`));
                            process.exit(1);
                        }
                    }),
            )
            .addCommand(
                new Command("create <profile>")
                    .description("Create a new profile")
                    .option("-u, --url <url>", "Server URL", "http://localhost:5329")
                    .action((profile, options) => {
                        try {
                            config.createProfile(profile, { url: options.url });
                            console.log(chalk.green(`✓ Created profile: ${profile}`));
                        } catch (error) {
                            const errorMessage = error instanceof Error ? error.message : String(error);
                            console.error(chalk.red(`✗ ${errorMessage}`));
                            process.exit(1);
                        }
                    }),
            );

        // Add completion management commands
        program
            .command("completion")
            .description("Manage shell completions")
            .addCommand(
                new Command("install")
                    .description("Install shell completions")
                    .option("--shell <shell>", "Shell type (bash, zsh, fish)", "auto")
                    .action(async (options) => {
                        try {
                            const installer = new CompletionInstaller();
                            await installer.install(options.shell);
                        } catch (error) {
                            const errorMessage = error instanceof Error ? error.message : String(error);
                            console.error(chalk.red(`✗ ${errorMessage}`));
                            process.exit(1);
                        }
                    }),
            )
            .addCommand(
                new Command("uninstall")
                    .description("Uninstall shell completions")
                    .action(async () => {
                        try {
                            const installer = new CompletionInstaller();
                            await installer.uninstall();
                        } catch (error) {
                            const errorMessage = error instanceof Error ? error.message : String(error);
                            console.error(chalk.red(`✗ ${errorMessage}`));
                            process.exit(1);
                        }
                    }),
            )
            .addCommand(
                new Command("status")
                    .description("Show completion installation status")
                    .action(async () => {
                        try {
                            const installer = new CompletionInstaller();
                            await installer.status();
                        } catch (error) {
                            const errorMessage = error instanceof Error ? error.message : String(error);
                            console.error(chalk.red(`✗ ${errorMessage}`));
                            process.exit(1);
                        }
                    }),
            );

        // Add process exit handlers for history tracking
        process.on("exit", (code) => {
            try {
                // Synchronous call for exit handler
                historyManager.endCommandSync(code);
            } catch (error) {
                // Ignore errors during exit
            }
        });

        const SIGINT_EXIT_CODE = 130;
        const SIGTERM_EXIT_CODE = 143;

        process.on("SIGINT", () => {
            try {
                historyManager.endCommandSync(SIGINT_EXIT_CODE); // SIGINT exit code
            } catch (error) {
                // Ignore errors during exit
            }
            process.exit(SIGINT_EXIT_CODE);
        });

        process.on("SIGTERM", () => {
            try {
                historyManager.endCommandSync(SIGTERM_EXIT_CODE); // SIGTERM exit code
            } catch (error) {
                // Ignore errors during exit
            }
            process.exit(SIGTERM_EXIT_CODE);
        });

        await program.parseAsync();
    } catch (error) {
        // Track failed command in history
        try {
            const config = new ConfigManager();
            const historyManager = new HistoryManager(config);
            await historyManager.endCommand(1, error instanceof Error ? error : new Error(String(error)));
        } catch (historyError) {
            // Ignore history tracking errors
        }
        
        logger.error("CLI error", error);
        const errorMessage = error instanceof Error ? error.message : String(error);
        console.error(chalk.red(`\n✗ Error: ${errorMessage}`));
        if (error instanceof Error && error.stack && process.env.DEBUG) {
            console.error(error.stack);
        }
        process.exit(1);
    }
}

// Run the CLI
main().catch((error) => {
    console.error(chalk.red("Fatal error:"), error);
    process.exit(1);
});
