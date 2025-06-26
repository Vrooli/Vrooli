#!/usr/bin/env node
import { Command } from "commander";
import { ConfigManager } from "./utils/config.js";
import { ApiClient } from "./utils/client.js";
import { AuthCommands } from "./commands/auth.js";
import { RoutineCommands } from "./commands/routine.js";
import chalk from "chalk";

// Simple logger for CLI
const logger = {
    error: (message: string, error?: any) => {
        console.error(chalk.red(`[ERROR] ${message}`), error || '');
    },
    setLevel: (level: string) => {
        // CLI logger is simple, no level setting needed
    }
};

async function main() {
    try {
        const program = new Command();
        const config = new ConfigManager();
        const client = new ApiClient(config);

        program
            .name("vrooli")
            .description("Vrooli CLI - Manage your Vrooli instance from the command line")
            .version("1.0.0")
            .option("-p, --profile <profile>", "Use a specific profile", "default")
            .option("-d, --debug", "Enable debug output")
            .option("--json", "Output in JSON format")
            .hook("preAction", (thisCommand) => {
                const opts = thisCommand.opts();
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
            });

        // Register command modules
        new AuthCommands(program, client, config);
        new RoutineCommands(program, client, config);

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
                    })
            )
            .addCommand(
                new Command("use <profile>")
                    .description("Switch to a different profile")
                    .action((profile) => {
                        try {
                            config.setActiveProfile(profile);
                            console.log(chalk.green(`✓ Switched to profile: ${profile}`));
                        } catch (error) {
                            console.error(chalk.red(`✗ ${error.message}`));
                            process.exit(1);
                        }
                    })
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
                            console.error(chalk.red(`✗ ${error.message}`));
                            process.exit(1);
                        }
                    })
            );

        await program.parseAsync();
    } catch (error) {
        logger.error("CLI error", error);
        console.error(chalk.red(`\n✗ Error: ${error.message}`));
        if (error.stack && process.env.DEBUG) {
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