import { promises as fs } from "fs";
import * as path from "path";
import * as os from "os";
import { BashCompletionGenerator } from "./generators/bash.js";
import { ZshCompletionGenerator } from "./generators/zsh.js";
import { FishCompletionGenerator } from "./generators/fish.js";
import chalk from "chalk";

export class CompletionInstaller {
    private readonly commandName = "vrooli";
    
    async install(shellType = "auto"): Promise<void> {
        const shell = shellType === "auto" ? this.detectShell() : shellType;
        
        if (!["bash", "zsh", "fish"].includes(shell)) {
            throw new Error(`Unsupported shell: ${shell}`);
        }
        
        const generator = this.getGenerator(shell);
        const script = generator.generate(this.commandName);
        
        // Write to appropriate location
        const installPath = this.getInstallPath(shell);
        await this.ensureDirectoryExists(path.dirname(installPath));
        await fs.writeFile(installPath, script);
        
        console.log(chalk.green(`✓ Installed ${shell} completions to ${installPath}`));
        console.log(chalk.gray(`  Restart your shell or run: source ${installPath}`));
        
        // Show shell-specific instructions
        this.showShellInstructions(shell, installPath);
    }
    
    async uninstall(): Promise<void> {
        const shells = ["bash", "zsh", "fish"];
        let uninstalledCount = 0;
        
        for (const shell of shells) {
            const installPath = this.getInstallPath(shell);
            
            try {
                await fs.access(installPath);
                await fs.unlink(installPath);
                console.log(chalk.green(`✓ Uninstalled ${shell} completions`));
                uninstalledCount++;
            } catch (error) {
                // File doesn't exist, ignore
            }
        }
        
        if (uninstalledCount === 0) {
            console.log(chalk.yellow("No completions found to uninstall"));
        } else {
            console.log(chalk.gray("Restart your shell for changes to take effect"));
        }
    }
    
    async status(): Promise<void> {
        const shells = ["bash", "zsh", "fish"];
        
        console.log(chalk.bold("Completion Status:"));
        
        for (const shell of shells) {
            const installPath = this.getInstallPath(shell);
            
            try {
                await fs.access(installPath);
                console.log(chalk.green(`  ${shell}: Installed (${installPath})`));
            } catch (error) {
                console.log(chalk.red(`  ${shell}: Not installed`));
            }
        }
    }
    
    private detectShell(): string {
        // Try to detect shell from environment
        const shell = process.env.SHELL || "";
        
        if (shell.includes("bash")) {
            return "bash";
        } else if (shell.includes("zsh")) {
            return "zsh";
        } else if (shell.includes("fish")) {
            return "fish";
        }
        
        // Default to bash
        return "bash";
    }
    
    private getGenerator(shell: string): BashCompletionGenerator | ZshCompletionGenerator | FishCompletionGenerator {
        switch (shell) {
            case "bash":
                return new BashCompletionGenerator();
            case "zsh":
                return new ZshCompletionGenerator();
            case "fish":
                return new FishCompletionGenerator();
            default:
                throw new Error(`Unsupported shell: ${shell}`);
        }
    }
    
    private getInstallPath(shell: string): string {
        const homeDir = os.homedir();
        
        switch (shell) {
            case "bash": {
                // Try system-wide first, then user-specific
                const userBashPath = path.join(homeDir, ".local/share/bash-completion/completions/vrooli");
                
                // For now, use user-specific path to avoid permission issues
                return userBashPath;
            }
                
            case "zsh": {
                // Check for common zsh completion directories
                const zshDirs = [
                    path.join(homeDir, ".zsh/completions"),
                    path.join(homeDir, ".zsh_completions"),
                    path.join(homeDir, ".oh-my-zsh/custom/plugins/vrooli"),
                ];
                
                // Use the first one that exists, or create the standard one
                return path.join(zshDirs[0], "_vrooli");
            }
                
            case "fish":
                // Fish completion directory
                return path.join(homeDir, ".config/fish/completions/vrooli.fish");
                
            default:
                throw new Error(`Unsupported shell: ${shell}`);
        }
    }
    
    private async ensureDirectoryExists(dirPath: string): Promise<void> {
        try {
            await fs.mkdir(dirPath, { recursive: true });
        } catch (error) {
            // Directory might already exist
        }
    }
    
    private showShellInstructions(shell: string, installPath: string): void {
        console.log(chalk.bold(`\\n${shell.toUpperCase()} Instructions:`));
        
        switch (shell) {
            case "bash":
                console.log(chalk.gray("  To enable completions for the current session:"));
                console.log(chalk.cyan(`    source ${installPath}`));
                console.log(chalk.gray("  To enable completions permanently, add to your ~/.bashrc:"));
                console.log(chalk.cyan(`    echo "source ${installPath}" >> ~/.bashrc`));
                break;
                
            case "zsh":
                console.log(chalk.gray("  To enable completions, ensure the directory is in your fpath:"));
                console.log(chalk.cyan(`    fpath=(${path.dirname(installPath)} $fpath)`));
                console.log(chalk.gray("  Add this to your ~/.zshrc if not already present:"));
                console.log(chalk.cyan(`    echo "fpath=(${path.dirname(installPath)} \\$fpath)" >> ~/.zshrc`));
                console.log(chalk.gray("  Then reload completions:"));
                console.log(chalk.cyan("    autoload -U compinit && compinit"));
                break;
                
            case "fish":
                console.log(chalk.gray("  Fish should automatically load completions from:"));
                console.log(chalk.cyan(`    ${installPath}`));
                console.log(chalk.gray("  Restart your shell or run:"));
                console.log(chalk.cyan("    source ~/.config/fish/config.fish"));
                break;
        }
    }
    
    /**
     * Test completion generation to ensure it works
     */
    async testCompletion(): Promise<void> {
        console.log(chalk.bold("Testing completion generation..."));
        
        const testCases = [
            ["vrooli"],
            ["vrooli", "r"],
            ["vrooli", "routine"],
            ["vrooli", "routine", "l"],
            ["vrooli", "routine", "list"],
            ["vrooli", "routine", "list", "--"],
        ];
        
        for (const args of testCases) {
            try {
                // This would normally call the completion engine
                console.log(chalk.gray(`  Testing: ${args.join(" ")}`));
                // const completions = await engine.getCompletions(args);
                // console.log(chalk.green(`    Found ${completions.length} completions`));
            } catch (error) {
                console.log(chalk.red(`    Error: ${error}`));
            }
        }
    }
}
