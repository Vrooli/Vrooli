import { beforeEach, describe, expect, it } from "vitest";
import type { CompletionContext } from "../types.js";
import { StaticProvider } from "./StaticProvider.js";

describe("StaticProvider", () => {
    let provider: StaticProvider;

    beforeEach(() => {
        provider = new StaticProvider();
    });

    describe("canHandle", () => {
        it("should handle command type", () => {
            const context: CompletionContext = {
                type: "command",
                partial: "auth",
                args: ["auth"],
                options: {},
            };

            expect(provider.canHandle(context)).toBe(true);
        });

        it("should handle subcommand type", () => {
            const context: CompletionContext = {
                type: "subcommand",
                command: "auth",
                partial: "log",
                args: ["auth", "log"],
                options: {},
            };

            expect(provider.canHandle(context)).toBe(true);
        });

        it("should handle option type", () => {
            const context: CompletionContext = {
                type: "option",
                command: "auth",
                subcommand: "login",
                partial: "--em",
                args: ["auth", "login", "--em"],
                options: {},
            };

            expect(provider.canHandle(context)).toBe(true);
        });

        it("should not handle file type", () => {
            const context: CompletionContext = {
                type: "file",
                command: "routine",
                subcommand: "import",
                partial: "file.j",
                args: ["routine", "import", "file.j"],
                options: {},
            };

            expect(provider.canHandle(context)).toBe(false);
        });

        it("should not handle resource type", () => {
            const context: CompletionContext = {
                type: "resource",
                command: "routine",
                subcommand: "get",
                partial: "123",
                args: ["routine", "get", "123"],
                options: {},
                resourceType: "routine",
            };

            expect(provider.canHandle(context)).toBe(false);
        });
    });

    describe("getCompletions", () => {
        describe("command completions", () => {
            it("should return all commands when partial is empty", async () => {
                const context: CompletionContext = {
                    type: "command",
                    partial: "",
                    args: [""],
                    options: {},
                };

                const results = await provider.getCompletions(context);

                expect(results).toHaveLength(8); // auth, routine, agent, team, chat, profile, completion, history
                expect(results.map(r => r.value)).toContain("auth");
                expect(results.map(r => r.value)).toContain("routine");
                expect(results.map(r => r.value)).toContain("agent");
                expect(results.map(r => r.value)).toContain("team");
                expect(results.map(r => r.value)).toContain("chat");
                expect(results.every(r => r.type === "command")).toBe(true);
            });

            it("should filter commands by partial match", async () => {
                const context: CompletionContext = {
                    type: "command",
                    partial: "a",
                    args: ["a"],
                    options: {},
                };

                const results = await provider.getCompletions(context);

                expect(results).toHaveLength(2); // auth, agent
                expect(results.map(r => r.value)).toEqual(["auth", "agent"]);
                expect(results[0].description).toBe("Authentication commands");
            });

            it("should return empty array for no matches", async () => {
                const context: CompletionContext = {
                    type: "command",
                    partial: "xyz",
                    args: ["xyz"],
                    options: {},
                };

                const results = await provider.getCompletions(context);

                expect(results).toHaveLength(0);
            });
        });

        describe("subcommand completions", () => {
            it("should return auth subcommands", async () => {
                const context: CompletionContext = {
                    type: "subcommand",
                    command: "auth",
                    partial: "",
                    args: ["auth", ""],
                    options: {},
                };

                const results = await provider.getCompletions(context);

                expect(results).toHaveLength(4); // login, logout, status, whoami
                expect(results.map(r => r.value)).toEqual(["login", "logout", "status", "whoami"]);
                expect(results[0].description).toBe("Login to your Vrooli account");
                expect(results.every(r => r.type === "subcommand")).toBe(true);
            });

            it("should filter routine subcommands by partial", async () => {
                const context: CompletionContext = {
                    type: "subcommand",
                    command: "routine",
                    partial: "ex",
                    args: ["routine", "ex"],
                    options: {},
                };

                const results = await provider.getCompletions(context);

                expect(results).toHaveLength(1);
                expect(results[0].value).toBe("export");
                expect(results[0].description).toBe("Export a routine to a JSON file");
            });

            it("should return empty for unknown command", async () => {
                const context: CompletionContext = {
                    type: "subcommand",
                    command: "unknown",
                    partial: "",
                    args: ["unknown", ""],
                    options: {},
                };

                const results = await provider.getCompletions(context);

                expect(results).toHaveLength(0);
            });

            it("should handle team subcommands", async () => {
                const context: CompletionContext = {
                    type: "subcommand",
                    command: "team",
                    partial: "s",
                    args: ["team", "s"],
                    options: {},
                };

                const results = await provider.getCompletions(context);

                expect(results.map(r => r.value)).toContain("spawn");
                expect(results.every(r => r.value.startsWith("s"))).toBe(true);
            });
        });

        describe("option completions", () => {
            it("should return subcommand-specific options", async () => {
                const context: CompletionContext = {
                    type: "option",
                    command: "auth",
                    subcommand: "login",
                    partial: "--",
                    args: ["auth", "login", "--"],
                    options: {},
                };

                const results = await provider.getCompletions(context);

                const optionValues = results.map(r => r.value);
                expect(optionValues).toContain("--email");
                expect(optionValues).toContain("--password");
                expect(optionValues).toContain("--no-save");
                expect(results.every(r => r.type === "option")).toBe(true);
            });

            it("should include global CLI options", async () => {
                const context: CompletionContext = {
                    type: "option",
                    command: "routine",
                    subcommand: "list",
                    partial: "--",
                    args: ["routine", "list", "--"],
                    options: {},
                };

                const results = await provider.getCompletions(context);

                const optionValues = results.map(r => r.value);
                expect(optionValues).toContain("--help");
                expect(optionValues).toContain("--version");
                expect(optionValues).toContain("--debug");
                expect(optionValues).toContain("--json");
                expect(optionValues).toContain("--profile");
            });

            it("should filter options by partial match", async () => {
                const context: CompletionContext = {
                    type: "option",
                    command: "routine",
                    subcommand: "list",
                    partial: "--l",
                    args: ["routine", "list", "--l"],
                    options: {},
                };

                const results = await provider.getCompletions(context);

                expect(results.map(r => r.value)).toContain("--limit");
                expect(results.every(r => r.value.startsWith("--l"))).toBe(true);
            });

            it("should include short options", async () => {
                const context: CompletionContext = {
                    type: "option",
                    command: "routine",
                    subcommand: "list",
                    partial: "-",
                    args: ["routine", "list", "-"],
                    options: {},
                };

                const results = await provider.getCompletions(context);

                const optionValues = results.map(r => r.value);
                expect(optionValues).toContain("-l");
                expect(optionValues).toContain("-s");
                expect(optionValues).toContain("-f");
                expect(optionValues).toContain("-p");
                expect(optionValues).toContain("-d");
                expect(optionValues).toContain("-h");
                expect(optionValues).toContain("-V");
            });

            it("should return empty for unknown command", async () => {
                const context: CompletionContext = {
                    type: "option",
                    command: "unknown",
                    subcommand: "test",
                    partial: "--",
                    args: ["unknown", "test", "--"],
                    options: {},
                };

                const results = await provider.getCompletions(context);

                // Should still return global options
                const optionValues = results.map(r => r.value);
                expect(optionValues).toContain("--help");
                expect(optionValues).toContain("--version");
            });

            it("should handle team options with complex configurations", async () => {
                const context: CompletionContext = {
                    type: "option",
                    command: "team",
                    subcommand: "create",
                    partial: "--ta",
                    args: ["team", "create", "--ta"],
                    options: {},
                };

                const results = await provider.getCompletions(context);

                expect(results.map(r => r.value)).toContain("--target-profit");
                expect(results.every(r => r.value.startsWith("--ta"))).toBe(true);
            });

            it("should handle chat interactive options", async () => {
                const context: CompletionContext = {
                    type: "option",
                    command: "chat",
                    subcommand: "interactive",
                    partial: "--show",
                    args: ["chat", "interactive", "--show"],
                    options: {},
                };

                const results = await provider.getCompletions(context);

                expect(results.map(r => r.value)).toContain("--show-tools");
            });

            it("should handle history export options", async () => {
                const context: CompletionContext = {
                    type: "option",
                    command: "history",
                    subcommand: "export",
                    partial: "--",
                    args: ["history", "export", "--"],
                    options: {},
                };

                const results = await provider.getCompletions(context);

                const optionValues = results.map(r => r.value);
                expect(optionValues).toContain("--output");
                expect(optionValues).toContain("--format");
                expect(optionValues).toContain("--since");
            });
        });

        describe("unsupported context types", () => {
            it("should return empty array for file context", async () => {
                const context: CompletionContext = {
                    type: "file",
                    command: "routine",
                    subcommand: "import",
                    partial: "file.j",
                    args: ["routine", "import", "file.j"],
                    options: {},
                };

                const results = await provider.getCompletions(context);

                expect(results).toHaveLength(0);
            });

            it("should return empty array for resource context", async () => {
                const context: CompletionContext = {
                    type: "resource",
                    command: "routine",
                    subcommand: "get",
                    partial: "123",
                    args: ["routine", "get", "123"],
                    options: {},
                    resourceType: "routine",
                };

                const results = await provider.getCompletions(context);

                expect(results).toHaveLength(0);
            });
        });

        describe("comprehensive command coverage", () => {
            const commands = ["auth", "routine", "agent", "team", "chat", "profile", "completion", "history"];

            commands.forEach(command => {
                it(`should provide completions for ${command} command`, async () => {
                    const context: CompletionContext = {
                        type: "subcommand",
                        command,
                        partial: "",
                        args: [command, ""],
                        options: {},
                    };

                    const results = await provider.getCompletions(context);

                    expect(results.length).toBeGreaterThan(0);
                    expect(results.every(r => r.type === "subcommand")).toBe(true);
                    expect(results.every(r => r.description.length > 0)).toBe(true);
                });
            });
        });
    });
});
