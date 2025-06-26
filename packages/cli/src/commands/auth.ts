import { Command } from "commander";
import { ApiClient } from "../utils/client.js";
import { ConfigManager } from "../utils/config.js";
import chalk from "chalk";
import inquirer from "inquirer";
import ora from "ora";

export class AuthCommands {
    constructor(
        private program: Command,
        private client: ApiClient,
        private config: ConfigManager,
    ) {
        this.registerCommands();
    }

    private registerCommands(): void {
        const authCmd = this.program
            .command("auth")
            .description("Authentication commands");

        authCmd
            .command("login")
            .description("Login to your Vrooli account")
            .option("-e, --email <email>", "Email address")
            .option("-p, --password <password>", "Password (not recommended - use prompt)")
            .option("--no-save", "Don't save credentials")
            .action(async (options) => {
                await this.login(options);
            });

        authCmd
            .command("logout")
            .description("Logout and clear stored credentials")
            .action(async () => {
                await this.logout();
            });

        authCmd
            .command("status")
            .description("Check authentication status")
            .action(async () => {
                await this.status();
            });

        authCmd
            .command("whoami")
            .description("Display current user information")
            .action(async () => {
                await this.whoami();
            });
    }

    private async login(options: { email?: string; password?: string; save?: boolean }): Promise<void> {
        try {
            // Prompt for credentials if not provided
            const credentials = await this.promptCredentials(options);
            
            const spinner = ora("Logging in...").start();

            try {
                // Call login endpoint
                const response = await this.client.post("/auth/login", {
                    email: credentials.email,
                    password: credentials.password,
                });

                const { accessToken, refreshToken, user, expiresIn } = response;

                // Save tokens if requested
                if (options.save !== false) {
                    this.config.setAuth(accessToken, refreshToken, expiresIn);
                }

                spinner.succeed("Logged in successfully");

                if (this.config.isJsonOutput()) {
                    console.log(JSON.stringify({
                        success: true,
                        user: {
                            id: user.id,
                            email: user.email,
                            name: user.name,
                        },
                    }));
                } else {
                    console.log(chalk.green("\n✓ Authentication successful"));
                    console.log(`  User: ${user.name || user.email}`);
                    console.log(`  Email: ${user.email}`);
                    if (options.save !== false) {
                        console.log(`  Profile: ${this.config.getActiveProfileName()}`);
                    }
                }
            } catch (error) {
                spinner.fail("Login failed");
                throw error;
            }
        } catch (error) {
            if (this.config.isJsonOutput()) {
                console.log(JSON.stringify({
                    success: false,
                    error: error.message,
                }));
            } else {
                console.error(chalk.red(`✗ Login failed: ${error.message}`));
            }
            process.exit(1);
        }
    }

    private async logout(): Promise<void> {
        try {
            const spinner = ora("Logging out...").start();

            try {
                // Call logout endpoint if we have a token
                const token = this.config.getAuthToken();
                if (token) {
                    await this.client.post("/auth/logout");
                }
            } catch (error) {
                // Ignore logout errors - we'll clear local auth anyway
                if (this.config.isDebug()) {
                    console.error("Logout API error (ignored):", error);
                }
            }

            // Clear local auth
            this.config.clearAuth();
            spinner.succeed("Logged out successfully");

            if (this.config.isJsonOutput()) {
                console.log(JSON.stringify({ success: true }));
            } else {
                console.log(chalk.green("✓ Credentials cleared"));
            }
        } catch (error) {
            console.error(chalk.red(`✗ Logout failed: ${error.message}`));
            process.exit(1);
        }
    }

    private async status(): Promise<void> {
        try {
            const token = this.config.getAuthToken();
            
            if (!token) {
                if (this.config.isJsonOutput()) {
                    console.log(JSON.stringify({
                        authenticated: false,
                        profile: this.config.getActiveProfileName(),
                    }));
                } else {
                    console.log(chalk.yellow("⚠ Not authenticated"));
                    console.log(`  Profile: ${this.config.getActiveProfileName()}`);
                    console.log(`  Server: ${this.config.getServerUrl()}`);
                }
                return;
            }

            // Verify token with server
            const spinner = ora("Checking authentication...").start();
            
            try {
                const user = await this.client.get("/auth/me");
                spinner.succeed("Authenticated");

                if (this.config.isJsonOutput()) {
                    console.log(JSON.stringify({
                        authenticated: true,
                        profile: this.config.getActiveProfileName(),
                        user: {
                            id: user.id,
                            email: user.email,
                            name: user.name,
                        },
                    }));
                } else {
                    console.log(chalk.green("\n✓ Authenticated"));
                    console.log(`  User: ${user.name || user.email}`);
                    console.log(`  Email: ${user.email}`);
                    console.log(`  Profile: ${this.config.getActiveProfileName()}`);
                    console.log(`  Server: ${this.config.getServerUrl()}`);
                }
            } catch (error) {
                spinner.fail("Not authenticated");
                this.config.clearAuth();
                
                if (this.config.isJsonOutput()) {
                    console.log(JSON.stringify({
                        authenticated: false,
                        profile: this.config.getActiveProfileName(),
                        error: error.message,
                    }));
                } else {
                    console.log(chalk.red("\n✗ Authentication invalid"));
                    console.log(`  Profile: ${this.config.getActiveProfileName()}`);
                    console.log(`  Server: ${this.config.getServerUrl()}`);
                    console.log(chalk.gray("\nRun 'vrooli auth login' to authenticate"));
                }
            }
        } catch (error) {
            console.error(chalk.red(`✗ Status check failed: ${error.message}`));
            process.exit(1);
        }
    }

    private async whoami(): Promise<void> {
        try {
            const spinner = ora("Fetching user information...").start();

            const user = await this.client.get("/auth/me");
            spinner.succeed("User information retrieved");

            if (this.config.isJsonOutput()) {
                console.log(JSON.stringify(user));
            } else {
                console.log(chalk.bold("\nUser Information:"));
                console.log(`  ID: ${user.id}`);
                console.log(`  Email: ${user.email}`);
                console.log(`  Name: ${user.name || "(not set)"}`);
                console.log(`  Roles: ${user.roles?.join(", ") || "user"}`);
                console.log(`  Created: ${new Date(user.created_at).toLocaleString()}`);
                
                if (user.organizations?.length > 0) {
                    console.log(chalk.bold("\nOrganizations:"));
                    user.organizations.forEach((org: any) => {
                        console.log(`  - ${org.name} (${org.role})`);
                    });
                }
            }
        } catch (error) {
            if (error.message.includes("401")) {
                console.error(chalk.red("✗ Not authenticated. Run 'vrooli auth login' first."));
            } else {
                console.error(chalk.red(`✗ Failed to get user info: ${error.message}`));
            }
            process.exit(1);
        }
    }

    private async promptCredentials(options: {
        email?: string;
        password?: string;
    }): Promise<{ email: string; password: string }> {
        const questions = [];

        if (!options.email) {
            questions.push({
                type: "input",
                name: "email",
                message: "Email:",
                validate: (input: string) => {
                    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
                    return emailRegex.test(input) || "Please enter a valid email address";
                },
            });
        }

        if (!options.password) {
            questions.push({
                type: "password",
                name: "password",
                message: "Password:",
                mask: "*",
                validate: (input: string) => {
                    return input.length > 0 || "Password is required";
                },
            });
        }

        const answers = questions.length > 0 ? await inquirer.prompt(questions) : {};

        return {
            email: options.email || answers.email,
            password: options.password || answers.password,
        };
    }
}