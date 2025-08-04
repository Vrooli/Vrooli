import { describe, it, expect, beforeEach } from "vitest";
import { ZshCompletionGenerator } from "./zsh.js";

describe("ZshCompletionGenerator", () => {
    let generator: ZshCompletionGenerator;

    beforeEach(() => {
        generator = new ZshCompletionGenerator();
    });

    describe("generate", () => {
        it("should generate zsh completion script for given command name", () => {
            const commandName = "test-cli";
            const result = generator.generate(commandName);

            expect(result).toContain(`#compdef ${commandName}`);
            expect(result).toContain(`# Zsh completion for ${commandName}`);
            expect(result).toContain(`'${commandName} completion install'`);
        });

        it("should include completion function with correct naming", () => {
            const commandName = "vrooli";
            const result = generator.generate(commandName);

            expect(result).toContain("_vrooli() {");
            expect(result).toContain("_vrooli");
        });

        it("should handle command names with special characters", () => {
            const commandName = "my-special_cli.exe";
            const result = generator.generate(commandName);

            expect(result).toContain("_my-special_cli.exe() {");
            expect(result).toContain("_my-special_cli.exe");
        });

        it("should include proper zsh variable declarations", () => {
            const result = generator.generate("test");

            expect(result).toContain("local -a completions");
            expect(result).toContain("local -a completions_with_descriptions");
            expect(result).toContain("local current_word");
            expect(result).toContain("current_word=\"${words[CURRENT]}\"");
        });

        it("should include completion generation logic", () => {
            const commandName = "test";
            const result = generator.generate(commandName);

            expect(result).toContain("local completion_output");
            expect(result).toContain(`completion_output=$(${commandName} --generate-completions "\${words[@]}" 2>/dev/null)`);
        });

        it("should handle completion responses with descriptions", () => {
            const result = generator.generate("test");

            expect(result).toContain("if [[ \"$line\" == *:* ]]; then");
            expect(result).toContain("completions_with_descriptions+=(\"$line\")");
        });

        it("should handle completions without descriptions", () => {
            const result = generator.generate("test");

            expect(result).toContain("completions_with_descriptions+=(\"$line:$line\")");
        });

        it("should include proper loop structure", () => {
            const result = generator.generate("test");

            expect(result).toContain("while IFS= read -r line; do");
            expect(result).toContain("done <<< \"$completion_output\"");
        });

        it("should include fallback to file completion", () => {
            const result = generator.generate("test");

            expect(result).toContain("_files");
        });

        it("should include proper error handling", () => {
            const result = generator.generate("test");

            expect(result).toContain("2>/dev/null");
            expect(result).toContain("if [[ -n \"$completion_output\" ]]; then");
        });

        it("should use zsh-specific completion features", () => {
            const result = generator.generate("test");

            expect(result).toContain("_describe '' completions_with_descriptions");
            expect(result).toContain("return 0");
        });

        it("should generate valid zsh syntax", () => {
            const result = generator.generate("test");

            // Check for proper function structure
            expect(result).toMatch(/_[^_]+\(\) \{[\s\S]*\}/);
            expect(result).toMatch(/_[^_]+$/);
        });

        it("should include array length checking", () => {
            const result = generator.generate("test");

            expect(result).toContain("if [[ ${#completions_with_descriptions[@]} -gt 0 ]]; then");
        });

        it("should be deterministic", () => {
            const commandName = "consistent";
            const result1 = generator.generate(commandName);
            const result2 = generator.generate(commandName);

            expect(result1).toBe(result2);
        });

        it("should handle empty command name", () => {
            const result = generator.generate("");

            expect(result).toContain("#compdef ");
            expect(result).toContain("_() {");
            expect(result).toContain("_");
        });

        it("should generate multi-line script", () => {
            const result = generator.generate("test");
            const lines = result.split("\n");

            expect(lines.length).toBeGreaterThan(10);
            expect(lines[0]).toContain("#compdef");
        });

        it("should include proper commenting", () => {
            const commandName = "documented";
            const result = generator.generate(commandName);

            expect(result).toContain("# Zsh completion for documented");
            expect(result).toContain("# This file is generated by");
            expect(result).toContain("# Call the CLI with special completion flag");
            expect(result).toContain("# Parse completions");
            expect(result).toContain("# Completion with description");
            expect(result).toContain("# Completion without description");
            expect(result).toContain("# Fallback to file completion");
            expect(result).toContain("# Register the completion function");
        });

        it("should handle zsh-specific variable access", () => {
            const result = generator.generate("test");

            expect(result).toContain("${words[CURRENT]}");
            expect(result).toContain("${words[@]}");
            expect(result).toContain("${#completions_with_descriptions[@]}");
        });
    });

    describe("edge cases", () => {
        it("should handle very long command names", () => {
            const longName = "a".repeat(100);
            const result = generator.generate(longName);

            expect(result).toContain(`#compdef ${longName}`);
            expect(result).toContain(`_${longName}() {`);
            expect(result).toContain(`_${longName}`);
        });

        it("should handle command names with numbers", () => {
            const result = generator.generate("cli2024");

            expect(result).toContain("#compdef cli2024");
            expect(result).toContain("_cli2024() {");
            expect(result).toContain("_cli2024");
        });

        it("should maintain script structure regardless of command name", () => {
            const names = ["a", "test-long-name", "CLI_TOOL", "tool.v2"];
            
            names.forEach(name => {
                const result = generator.generate(name);
                
                expect(result).toContain("#compdef");
                expect(result).toContain("# Zsh completion for");
                expect(result).toContain("local -a completions");
                expect(result).toContain("--generate-completions");
                expect(result).toContain("_files");
                expect(result).toContain("_describe");
            });
        });

        it("should handle conditional logic correctly", () => {
            const result = generator.generate("test");

            expect(result).toContain("if [[ -n \"$completion_output\" ]]; then");
            expect(result).toContain("if [[ \"$line\" == *:* ]]; then");
            expect(result).toContain("if [[ ${#completions_with_descriptions[@]} -gt 0 ]]; then");
        });

        it("should properly handle string interpolation", () => {
            const result = generator.generate("test");

            expect(result).toContain("\"${words[@]}\"");
            expect(result).toContain("\"$completion_output\"");
            expect(result).toContain("\"$line\"");
        });

        it("should include proper array operations", () => {
            const result = generator.generate("test");

            expect(result).toContain("completions_with_descriptions+=(\"$line\")");
            expect(result).toContain("completions_with_descriptions+=(\"$line:$line\")");
        });
    });
});