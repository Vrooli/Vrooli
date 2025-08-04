import { type Command } from "commander";
import { BaseCommand } from "./BaseCommand.js";
import { type ApiClient } from "../utils/client.js";
import { type ConfigManager } from "../utils/config.js";
import { logger } from "../utils/logger.js";
import { exec } from "child_process";
import { promisify } from "util";
import fs from "fs/promises";
import path from "path";
import chalk from "chalk";

// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-08-04

const execAsync = promisify(exec);

// UI formatting constants
const REPO_INFO_SEPARATOR_LENGTH = 50;
const MIRRORS_SEPARATOR_LENGTH = 30;
const HOOKS_SEPARATOR_LENGTH = 20;

interface RepositoryConfig {
    type: string;
    url: string;
    sshUrl?: string;
    directory?: string;
    branch?: string;
    private?: boolean;
    mirrors?: string[];
    submodules?: boolean | {
        enabled: boolean;
        recursive: boolean;
        shallow: boolean;
    };
    hooks?: {
        postClone?: string;
        preBuild?: string;
        postUpdate?: string;
    };
}

export class RepoCommands extends BaseCommand {
    constructor(
        program: Command,
        client: ApiClient,
        config: ConfigManager,
    ) {
        super(program, client, config);
    }

    protected registerCommands(): void {
        const repoCmd = this.program
            .command("repo")
            .description("Repository management commands");

        repoCmd
            .command("info")
            .description("Display repository information")
            .option("--json", "Output in JSON format")
            .action(async (options) => {
                await this.showInfo(options);
            });

        repoCmd
            .command("open")
            .description("Open repository in browser")
            .action(async () => {
                await this.openRepository();
            });

        repoCmd
            .command("clone")
            .description("Clone repository with proper configuration")
            .action(async () => {
                await this.cloneRepository();
            });

        repoCmd
            .command("mirrors")
            .description("List available mirror repositories")
            .option("--json", "Output in JSON format")
            .action(async (options) => {
                await this.showMirrors(options);
            });

        repoCmd
            .command("hooks")
            .description("List and run repository hooks")
            .option("--json", "Output in JSON format")
            .action(async (options) => {
                await this.showHooks(options);
            });
    }

    private async getServiceConfig(): Promise<{ repository?: RepositoryConfig } | null> {
        try {
            // Look for service.json in common locations
            const possiblePaths = [
                path.join(process.cwd(), ".vrooli", "service.json"),
                path.join(process.env.HOME || "", ".vrooli", "service.json"),
            ];

            for (const configPath of possiblePaths) {
                try {
                    const content = await fs.readFile(configPath, "utf-8");
                    const config = JSON.parse(content);
                    if (config.service?.repository) {
                        return config.service;
                    }
                } catch {
                    // Continue to next path
                }
            }

            return null;
        } catch (error) {
            logger.error("Failed to read service configuration", error);
            return null;
        }
    }

    private formatRepositoryInfo(repository: RepositoryConfig, format: string): void {
        if (format === "json") {
            console.log(JSON.stringify(repository, null, 2));
            return;
        }

        // Table format
        console.log("Repository Information:");
        console.log("─".repeat(REPO_INFO_SEPARATOR_LENGTH));
        console.log(`Type:      ${repository.type}`);
        console.log(`URL:       ${repository.url}`);
        
        if (repository.sshUrl) {
            console.log(`SSH URL:   ${repository.sshUrl}`);
        }
        
        if (repository.directory) {
            console.log(`Directory: ${repository.directory}`);
        }
        
        if (repository.branch) {
            console.log(`Branch:    ${repository.branch}`);
        }
        
        console.log(`Private:   ${repository.private ? "Yes" : "No"}`);
        
        if (repository.mirrors && repository.mirrors.length > 0) {
            console.log("\nMirrors:");
            repository.mirrors.forEach((mirror, index) => {
                console.log(`  ${index + 1}. ${mirror}`);
            });
        }
        
        if (repository.hooks) {
            console.log("\nHooks:");
            if (repository.hooks.postClone) {
                console.log(`  Post-Clone:  ${repository.hooks.postClone}`);
            }
            if (repository.hooks.preBuild) {
                console.log(`  Pre-Build:   ${repository.hooks.preBuild}`);
            }
            if (repository.hooks.postUpdate) {
                console.log(`  Post-Update: ${repository.hooks.postUpdate}`);
            }
        }

        if (repository.submodules) {
            console.log(`\nSubmodules: ${typeof repository.submodules === "boolean" ? 
                (repository.submodules ? "Enabled" : "Disabled") : 
                `Enabled (recursive: ${repository.submodules.recursive}, shallow: ${repository.submodules.shallow})`}`);
        }
    }

    private async openRepositoryInBrowser(url: string): Promise<void> {
        const platform = process.platform;
        let command: string;

        switch (platform) {
            case "darwin":
                command = `open "${url}"`;
                break;
            case "win32":
                command = `start "${url}"`;
                break;
            default:
                command = `xdg-open "${url}"`;
                break;
        }

        try {
            await execAsync(command);
            console.log(`Opened repository in browser: ${url}`);
        } catch (error) {
            console.error(`Failed to open repository in browser: ${error}`);
            console.log(`Please open manually: ${url}`);
        }
    }

    private async cloneRepositoryWithConfig(repository: RepositoryConfig): Promise<void> {
        const cloneUrl = repository.sshUrl || repository.url;
        const branch = repository.branch || "main";
        
        console.log(`Cloning repository from ${cloneUrl}...`);
        
        try {
            let command = "git clone";
            
            if (branch !== "main" && branch !== "master") {
                command += ` --branch ${branch}`;
            }
            
            if (repository.submodules) {
                if (typeof repository.submodules === "boolean" && repository.submodules) {
                    command += " --recurse-submodules";
                } else if (typeof repository.submodules === "object" && repository.submodules.enabled) {
                    command += " --recurse-submodules";
                    if (repository.submodules.shallow) {
                        command += " --shallow-submodules";
                    }
                }
            }
            
            command += ` "${cloneUrl}"`;
            
            console.log(`Running: ${command}`);
            const { stdout, stderr } = await execAsync(command);
            
            if (stdout) console.log(stdout);
            if (stderr) console.error(stderr);
            
            // Execute post-clone hook if configured
            if (repository.hooks?.postClone) {
                console.log("\nExecuting post-clone hook...");
                const { stdout: hookStdout, stderr: hookStderr } = await execAsync(repository.hooks.postClone);
                if (hookStdout) console.log(hookStdout);
                if (hookStderr) console.error(hookStderr);
            }
            
            console.log("Repository cloned successfully!");
        } catch (error) {
            console.error(`Failed to clone repository: ${error}`);
        }
    }

    private async showInfo(options: { json?: boolean }): Promise<void> {
        const serviceConfig = await this.getServiceConfig();
        
        if (!serviceConfig?.repository) {
            console.error(chalk.red("No repository configuration found in service.json"));
            process.exit(1);
        }

        this.formatRepositoryInfo(serviceConfig.repository, options.json ? "json" : "table");
    }

    private async openRepository(): Promise<void> {
        const serviceConfig = await this.getServiceConfig();
        
        if (!serviceConfig?.repository) {
            console.error(chalk.red("No repository configuration found in service.json"));
            process.exit(1);
        }

        await this.openRepositoryInBrowser(serviceConfig.repository.url);
    }

    private async cloneRepository(): Promise<void> {
        const serviceConfig = await this.getServiceConfig();
        
        if (!serviceConfig?.repository) {
            console.error(chalk.red("No repository configuration found in service.json"));
            process.exit(1);
        }

        await this.cloneRepositoryWithConfig(serviceConfig.repository);
    }

    private async showMirrors(options: { json?: boolean }): Promise<void> {
        const serviceConfig = await this.getServiceConfig();
        
        if (!serviceConfig?.repository) {
            console.error(chalk.red("No repository configuration found in service.json"));
            process.exit(1);
        }

        const repository = serviceConfig.repository;
        
        if (repository.mirrors && repository.mirrors.length > 0) {
            if (options.json) {
                console.log(JSON.stringify(repository.mirrors, null, 2));
            } else {
                console.log(chalk.bold("Mirror Repositories:"));
                console.log("─".repeat(MIRRORS_SEPARATOR_LENGTH));
                repository.mirrors.forEach((mirror, index) => {
                    console.log(`${index + 1}. ${mirror}`);
                });
            }
        } else {
            console.log(chalk.yellow("No mirror repositories configured"));
        }
    }

    private async showHooks(options: { json?: boolean }): Promise<void> {
        const serviceConfig = await this.getServiceConfig();
        
        if (!serviceConfig?.repository) {
            console.error(chalk.red("No repository configuration found in service.json"));
            process.exit(1);
        }

        const repository = serviceConfig.repository;
        
        if (repository.hooks) {
            if (options.json) {
                console.log(JSON.stringify(repository.hooks, null, 2));
            } else {
                console.log(chalk.bold("Repository Hooks:"));
                console.log("─".repeat(HOOKS_SEPARATOR_LENGTH));
                if (repository.hooks.postClone) {
                    console.log(`Post-Clone:  ${repository.hooks.postClone}`);
                }
                if (repository.hooks.preBuild) {
                    console.log(`Pre-Build:   ${repository.hooks.preBuild}`);
                }
                if (repository.hooks.postUpdate) {
                    console.log(`Post-Update: ${repository.hooks.postUpdate}`);
                }
                
                if (!repository.hooks.postClone && !repository.hooks.preBuild && !repository.hooks.postUpdate) {
                    console.log(chalk.yellow("No hooks configured"));
                }
            }
        } else {
            console.log(chalk.yellow("No repository hooks configured"));
        }
    }
}
