import { type Command } from "commander";
import { type ApiClient } from "../utils/client.js";
import { type ConfigManager } from "../utils/config.js";
import { output } from "../utils/output.js";
import chalk from "chalk";
import inquirer from "inquirer";
import ora from "ora";
import type { 
    Session, 
    EmailRequestPasswordChangeInput, 
    EmailResetPasswordInput, 
} from "@vrooli/shared";
import { AUTH_CONFIG } from "../utils/constants.js";
import { BaseCommand } from "./BaseCommand.js";

export class AuthCommands extends BaseCommand {
    constructor(
        program: Command,
        client: ApiClient,
        config: ConfigManager,
    ) {
        super(program, client, config);
    }

    protected registerCommands(): void {
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

        authCmd
            .command("reset-password")
            .description("Reset your password")
            .option("-e, --email <email>", "Email address")
            .action(async (options) => {
                await this.resetPassword(options);
            });

        authCmd
            .command("verify-email")
            .description("Verify your email address")
            .option("-c, --code <code>", "Verification code")
            .action(async (options) => {
                await this.verifyEmail(options);
            });
    }

    private async login(options: { email?: string; password?: string; save?: boolean }): Promise<void> {
        try {
            // Prompt for credentials if not provided
            const credentials = await this.promptCredentials(options);
            
            const session = await this.executeWithSpinner(
                "Logging in...",
                async () => {
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

                    return session;
                },
                "Logged in successfully",
            );

            const user = session.users?.[0];
            if (!user) {
                throw new Error("No user data in response");
            }
            
            this.output(
                {
                    success: true,
                    user: {
                        id: user.id,
                        handle: user.handle,
                        name: user.name,
                    },
                },
                () => {
                    output.success("\n✓ Authentication successful");
                    output.info(`  User: ${user.name || user.handle}`);
                    output.info(`  Handle: ${user.handle}`);
                    if (options.save !== false) {
                        output.info(`  Profile: ${this.config.getActiveProfileName()}`);
                    }
                },
            );
        } catch (error) {
            this.handleError(error, "Login failed");
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
                    output.debug("Logout API error (ignored):", error);
                }
            }

            // Clear local auth
            this.config.clearAuth();
            spinner.succeed("Logged out successfully");

            if (this.config.isJsonOutput()) {
                output.json({ success: true });
            } else {
                output.success("✓ Credentials cleared");
            }
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            output.error(`✗ Logout failed: ${errorMessage}`);
            process.exit(1);
        }
    }

    private async status(): Promise<void> {
        try {
            const token = this.config.getAuthToken();
            
            if (!token) {
                if (this.config.isJsonOutput()) {
                    output.json({
                        authenticated: false,
                        profile: this.config.getActiveProfileName(),
                    });
                } else {
                    output.warn("⚠ Not authenticated");
                    output.info(`  Profile: ${this.config.getActiveProfileName()}`);
                    output.info(`  Server: ${this.config.getServerUrl()}`);
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
                    output.json({
                        authenticated: true,
                        profile: this.config.getActiveProfileName(),
                        user: {
                            id: user.id,
                            handle: user.handle,
                            name: user.name,
                        },
                    });
                } else {
                    output.success("\n✓ Authenticated");
                    output.info(`  User: ${user.name || user.handle}`);
                    output.info(`  Handle: ${user.handle}`);
                    output.info(`  Profile: ${this.config.getActiveProfileName()}`);
                    output.info(`  Server: ${this.config.getServerUrl()}`);
                }
            } catch (error) {
                spinner.fail("Not authenticated");
                this.config.clearAuth();
                
                const errorMessage = error instanceof Error ? error.message : String(error);
                if (this.config.isJsonOutput()) {
                    output.json({
                        authenticated: false,
                        profile: this.config.getActiveProfileName(),
                        error: errorMessage,
                    });
                } else {
                    output.error("\n✗ Authentication invalid");
                    output.info(`  Profile: ${this.config.getActiveProfileName()}`);
                    output.info(`  Server: ${this.config.getServerUrl()}`);
                    output.info(chalk.gray("\nRun 'vrooli auth login' to authenticate"));
                }
            }
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            output.error(`✗ Status check failed: ${errorMessage}`);
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
                output.warn("No user data found");
                return;
            }

            if (this.config.isJsonOutput()) {
                output.json(user);
            } else {
                output.info(chalk.bold("\nUser Information:"));
                output.info(`  ID: ${user.id}`);
                output.info(`  Handle: ${user.handle || "(not set)"}`);
                output.info(`  Name: ${user.name || "(not set)"}`);
                output.info(`  Languages: ${user.languages?.join(", ") || "none"}`);
                output.info(`  Credits: ${user.credits || "0"}`);
                output.info(`  Premium: ${user.hasPremium ? "Yes" : "No"}`);
                output.info(`  Last Updated: ${new Date(user.updatedAt).toLocaleString()}`);
            }
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            if (errorMessage.includes("401")) {
                output.error("✗ Not authenticated. Run 'vrooli auth login' first.");
            } else {
                output.error(`✗ Failed to get user info: ${errorMessage}`);
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

    private async resetPassword(options: { email?: string }): Promise<void> {
        try {
            let email = options.email;

            // Prompt for email if not provided
            if (!email) {
                const emailAnswer = await inquirer.prompt([{
                    type: "input",
                    name: "email",
                    message: "Email address:",
                    validate: (input: string) => {
                        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
                        return emailRegex.test(input) || "Please enter a valid email address";
                    },
                }]);
                email = emailAnswer.email;
            }

            if (!email) {
                throw new Error("Email is required");
            }

            const spinner = ora("Requesting password reset...").start();

            try {
                // First, request password change  
                const requestInput: EmailRequestPasswordChangeInput = { email };
                await this.client.post("/auth/email/requestPasswordChange", requestInput);

                spinner.succeed("Password reset requested");

                if (this.config.isJsonOutput()) {
                    output.json({ success: true });
                } else {
                    output.success("\n✓ Password reset email sent");
                    output.info("  Check your email for reset instructions");
                    output.info(chalk.gray("  Use 'vrooli auth reset-password --help' for more options"));
                }

                // Prompt for reset code and new password
                const resetAnswers = await inquirer.prompt<{
                    code: string;
                    newPassword: string;
                    confirmPassword: string;
                }>([
                    {
                        type: "input",
                        name: "code",
                        message: "Enter the reset code from your email:",
                        validate: (input: string) => {
                            return input.length > 0 || "Reset code is required";
                        },
                    },
                    {
                        type: "password",
                        name: "newPassword",
                        message: "Enter your new password:",
                        mask: "*",
                        validate: (input: string) => {
                            return input.length >= AUTH_CONFIG.MIN_PASSWORD_LENGTH || `Password must be at least ${AUTH_CONFIG.MIN_PASSWORD_LENGTH} characters`;
                        },
                    },
                    {
                        type: "password",
                        name: "confirmPassword",
                        message: "Confirm your new password:",
                        mask: "*",
                        validate: (input: string, answers?: Record<string, unknown>) => {
                            return input === (answers as { newPassword?: string })?.newPassword || "Passwords do not match";
                        },
                    },
                ]);

                const resetSpinner = ora("Resetting password...").start();

                // Reset password with code
                const resetInput: EmailResetPasswordInput = {
                    code: resetAnswers.code,
                    newPassword: resetAnswers.newPassword,
                };
                await this.client.post("/auth/email/resetPassword", resetInput);

                resetSpinner.succeed("Password reset successfully");

                if (this.config.isJsonOutput()) {
                    output.json({ success: true });
                } else {
                    output.success("\n✓ Password reset successfully");
                    output.info("  You can now log in with your new password");
                }

            } catch (error) {
                spinner.fail("Password reset failed");
                throw error;
            }
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            if (this.config.isJsonOutput()) {
                output.json({
                    success: false,
                    error: errorMessage,
                });
            } else {
                output.error(`✗ Password reset failed: ${errorMessage}`);
            }
            process.exit(1);
        }
    }

    private async verifyEmail(options: { code?: string }): Promise<void> {
        try {
            let code = options.code;

            // Prompt for code if not provided
            if (!code) {
                const codeAnswer = await inquirer.prompt([{
                    type: "input",
                    name: "code",
                    message: "Enter the verification code from your email:",
                    validate: (input: string) => {
                        return input.length > 0 || "Verification code is required";
                    },
                }]);
                code = codeAnswer.code;
            }

            const spinner = ora("Verifying email...").start();

            try {
                // Call email verification endpoint
                await this.client.post("/email/verify", { code });

                spinner.succeed("Email verified successfully");

                if (this.config.isJsonOutput()) {
                    output.json({ success: true });
                } else {
                    output.success("\n✓ Email verified successfully");
                    output.info("  Your email is now verified and you have access to all features");
                }
            } catch (error) {
                spinner.fail("Email verification failed");
                throw error;
            }
        } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error);
            if (this.config.isJsonOutput()) {
                output.json({
                    success: false,
                    error: errorMessage,
                });
            } else {
                output.error(`✗ Email verification failed: ${errorMessage}`);
                output.info(chalk.gray("  Make sure you have the correct verification code"));
                output.info(chalk.gray("  Codes typically expire after a few minutes"));
            }
            process.exit(1);
        }
    }
}
