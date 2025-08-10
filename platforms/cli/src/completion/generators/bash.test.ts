import { beforeEach, describe, expect, it } from "vitest";
import { BashCompletionGenerator } from "./bash.js";

describe("BashCompletionGenerator", () => {
    let generator: BashCompletionGenerator;

    beforeEach(() => {
        generator = new BashCompletionGenerator();
    });

    describe("generate", () => {
        it("should generate bash completion script for given command name", () => {
            const commandName = "test-cli";
            const result = generator.generate(commandName);

            expect(result).toContain("#!/bin/bash");
            expect(result).toContain(`# Bash completion for ${commandName}`);
            expect(result).toContain(`'${commandName} completion install'`);
        });

        it("should include completion function with correct naming", () => {
            const commandName = "vrooli";
            const result = generator.generate(commandName);

            expect(result).toContain("_vrooli_completions()");
            expect(result).toContain("complete -F _vrooli_completions vrooli");
        });

        it("should handle command names with special characters", () => {
            const commandName = "my-special_cli.exe";
            const result = generator.generate(commandName);

            expect(result).toContain("_my-special_cli.exe_completions()");
            expect(result).toContain("complete -F _my-special_cli.exe_completions my-special_cli.exe");
        });

        it("should include proper bash completion initialization", () => {
            const result = generator.generate("test");

            expect(result).toContain("_init_completion || return");
            expect(result).toContain("local cur prev words cword");
        });

        it("should include completion generation logic", () => {
            const commandName = "test";
            const result = generator.generate(commandName);

            expect(result).toContain(`completions=$(${commandName} --generate-completions`);
            expect(result).toContain('"${COMP_WORDS[@]}" 2>/dev/null)');
        });

        it("should handle completion responses with colons", () => {
            const result = generator.generate("test");

            expect(result).toContain("if [[ \"$completion\" == *:* ]]; then");
            expect(result).toContain('COMPREPLY[i]="${completion%%:*}"');
        });

        it("should include fallback to file completion", () => {
            const result = generator.generate("test");

            expect(result).toContain("_filedir");
            expect(result).toContain("else");
        });

        it("should include proper error handling", () => {
            const result = generator.generate("test");

            expect(result).toContain("2>/dev/null");
            expect(result).toContain('if [[ -n "$completions" ]]; then');
        });

        it("should generate valid bash syntax", () => {
            const result = generator.generate("test");

            // Check for balanced quotes and brackets
            const openBrackets = (result.match(/\[/g) || []).length;
            const closeBrackets = (result.match(/\]/g) || []).length;
            expect(openBrackets).toBe(closeBrackets);

            // Check for proper function structure
            expect(result).toMatch(/_[^_]+_completions\(\) \{[\s\S]*\}/);
            expect(result).toMatch(/complete -F _[^_]+_completions .+/);
        });

        it("should include variable initialization", () => {
            const result = generator.generate("test");

            expect(result).toContain('local IFS=$\'\\n\'');
            expect(result).toContain("local completions");
        });

        it("should handle array operations correctly", () => {
            const result = generator.generate("test");

            expect(result).toContain("COMPREPLY=( $(compgen -W \"$completions\" -- \"$cur\") )");
            expect(result).toContain("for ((i=0; i<${#COMPREPLY[@]}; i++)); do");
        });

        it("should be deterministic", () => {
            const commandName = "consistent";
            const result1 = generator.generate(commandName);
            const result2 = generator.generate(commandName);

            expect(result1).toBe(result2);
        });

        it("should handle empty command name", () => {
            const result = generator.generate("");

            expect(result).toContain("__completions()");
            expect(result).toContain("complete -F __completions ");
        });

        it("should generate multi-line script", () => {
            const result = generator.generate("test");
            const lines = result.split("\n");

            expect(lines.length).toBeGreaterThan(10);
            expect(lines[0]).toBe("#!/bin/bash");
        });

        it("should include proper commenting", () => {
            const commandName = "documented";
            const result = generator.generate(commandName);

            expect(result).toContain("# Bash completion for documented");
            expect(result).toContain("# This file is generated by");
            expect(result).toContain("# Call the CLI with special completion flag");
            expect(result).toContain("# Check if we got any completions");
            expect(result).toContain("# Handle colons in completions (for descriptions)");
            expect(result).toContain("# Extract just the value part before the colon");
            expect(result).toContain("# Fallback to filename completion");
            expect(result).toContain("# Register the completion function");
        });
    });

    describe("edge cases", () => {
        it("should handle very long command names", () => {
            const longName = "a".repeat(100);
            const result = generator.generate(longName);

            expect(result).toContain(`_${longName}_completions()`);
            expect(result).toContain(`complete -F _${longName}_completions ${longName}`);
        });

        it("should handle command names with numbers", () => {
            const result = generator.generate("cli2024");

            expect(result).toContain("_cli2024_completions()");
            expect(result).toContain("complete -F _cli2024_completions cli2024");
        });

        it("should maintain script structure regardless of command name", () => {
            const names = ["a", "test-long-name", "CLI_TOOL", "tool.v2"];

            names.forEach(name => {
                const result = generator.generate(name);

                expect(result).toContain("#!/bin/bash");
                expect(result).toContain("_init_completion || return");
                expect(result).toContain("--generate-completions");
                expect(result).toContain("_filedir");
                expect(result).toMatch(/complete -F _.*_completions/);
            });
        });
    });
});