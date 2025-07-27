import { type Command } from "commander";
import { type ApiClient } from "../utils/client.js";
import { type ConfigManager } from "../utils/config.js";
import chalk from "chalk";
import inquirer from "inquirer";
import ora from "ora";
import type { Session } from "@vrooli/shared";

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
                const session = await this.client.post<Session>("/auth/email/login", {
                    email: credentials.email,
                    password: credentials.password,
                });

                const user = session.users?.[0];
                if (!user) {
                    throw new Error("No user data in login response");
                }

                // Tokens are handled via HTTP-only cookies automatically
                // Save session info if requested
                if (options.save !== false) {
                    this.config.setSession(session);
                }

                spinner.succeed("Logged in successfully");

                if (this.config.isJsonOutput()) {
                    console.log(JSON.stringify({
                        success: true,
                        user: {
                            id: user.id,
                            handle: user.handle,
                            name: user.name,
                        },
                    }));
                } else {
                    console.log(chalk.green("\n✓ Authentication successful"));
                    console.log(`  User: ${user.name || user.handle}`);
                    console.log(`  Handle: ${user.handle}`);
                    if (options.save !== false) {
                        console.log(`  Profile: ${this.config.getActiveProfileName()}`);
                    }
                }
            } catch (error) {
                spinner.fail("Login failed");
                throw error;
            }
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            if (this.config.isJsonOutput()) {
                console.log(JSON.stringify({
                    success: false,
                    error: errorMessage,
                }));
            } else {
                console.error(chalk.red(`✗ Login failed: ${errorMessage}`));
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
            const errorMessage = error instanceof Error ? error.message : String(error);
            console.error(chalk.red(`✗ Logout failed: ${errorMessage}`));
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
                const session = await this.client.get<Session>("/profile");
                spinner.succeed("Authenticated");

                const user = session.users?.[0];
                if (!user) {
                    spinner.fail("No user data found in session");
                    return;
                }

                if (this.config.isJsonOutput()) {
                    console.log(JSON.stringify({
                        authenticated: true,
                        profile: this.config.getActiveProfileName(),
                        user: {
                            id: user.id,
                            handle: user.handle,
                            name: user.name,
                        },
                    }));
                } else {
                    console.log(chalk.green("\n✓ Authenticated"));
                    console.log(`  User: ${user.name || user.handle}`);
                    console.log(`  Handle: ${user.handle}`);
                    console.log(`  Profile: ${this.config.getActiveProfileName()}`);
                    console.log(`  Server: ${this.config.getServerUrl()}`);
                }
            } catch (error) {
                spinner.fail("Not authenticated");
                this.config.clearAuth();
                
                const errorMessage = error instanceof Error ? error.message : String(error);
                if (this.config.isJsonOutput()) {
                    console.log(JSON.stringify({
                        authenticated: false,
                        profile: this.config.getActiveProfileName(),
                        error: errorMessage,
                    }));
                } else {
                    console.log(chalk.red("\n✗ Authentication invalid"));
                    console.log(`  Profile: ${this.config.getActiveProfileName()}`);
                    console.log(`  Server: ${this.config.getServerUrl()}`);
                    console.log(chalk.gray("\nRun 'vrooli auth login' to authenticate"));
                }
            }
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            console.error(chalk.red(`✗ Status check failed: ${errorMessage}`));
            process.exit(1);
        }
    }

    private async whoami(): Promise<void> {
        try {
            const spinner = ora("Fetching user information...").start();

            const session = await this.client.get<Session>("/profile");
            spinner.succeed("User information retrieved");

            const user = session.users?.[0];
            if (!user) {
                console.log(chalk.yellow("No user data found"));
                return;
            }

            if (this.config.isJsonOutput()) {
                console.log(JSON.stringify(user));
            } else {
                console.log(chalk.bold("\nUser Information:"));
                console.log(`  ID: ${user.id}`);
                console.log(`  Handle: ${user.handle || "(not set)"}`);
                console.log(`  Name: ${user.name || "(not set)"}`);
                console.log(`  Languages: ${user.languages?.join(", ") || "none"}`);
                console.log(`  Credits: ${user.credits || "0"}`);
                console.log(`  Premium: ${user.hasPremium ? "Yes" : "No"}`);
                console.log(`  Last Updated: ${new Date(user.updatedAt).toLocaleString()}`);
            }
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            if (errorMessage.includes("401")) {
                console.error(chalk.red("✗ Not authenticated. Run 'vrooli auth login' first."));
            } else {
                console.error(chalk.red(`✗ Failed to get user info: ${errorMessage}`));
            }
            process.exit(1);
        }
    }

    private async promptCredentials(options: {
        email?: string;
        password?: string;
    }): Promise<{ email: string; password: string }> {
        const answers: { email?: string; password?: string } = {};

        if (!options.email) {
            const emailAnswer = await inquirer.prompt([{
                type: "input",
                name: "email",
                message: "Email:",
                validate: (input: string) => {
                    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
                    return emailRegex.test(input) || "Please enter a valid email address";
                },
            }]);
            answers.email = emailAnswer.email;
        }

        if (!options.password) {
            const passwordAnswer = await inquirer.prompt([{
                type: "password",
                name: "password",
                message: "Password:",
                mask: "*",
                validate: (input: string) => {
                    return input.length > 0 || "Password is required";
                },
            }]);
            answers.password = passwordAnswer.password;
        }

        return {
            email: options.email || answers.email || "",
            password: options.password || answers.password || "",
        };
    }
}
