import { CodeLanguage } from "@vrooli/shared";
import { execFile } from "child_process";
import { chmodSync, promises as fs, mkdtempSync, rmSync, statSync, unlinkSync, writeFileSync } from "fs";
import { tmpdir } from "os";
import { join, resolve } from "path";
import { promisify } from "util";
import { logger } from "../../events/logger.js";
import { generateConfigurationErrorMessage, getExecutionConfig, type ExecutionConfig } from "./executionConfig.js";
import { type RunUserCodeInput, type RunUserCodeOutput } from "./types.js";

const execFileAsync = promisify(execFile);

// Check environment
const IS_DEV_OR_TEST = process.env.NODE_ENV === "development" || process.env.NODE_ENV === "test";

// Constants for file permissions and buffer sizes
const EXECUTABLE_PERMISSION = 0o755;
const MAX_OUTPUT_BUFFER_KB = 1024;
const BYTES_PER_KB = 1024;


/**
 * Safety checks for local execution using configuration
 */
function validateLocalExecution(config: ExecutionConfig): void {
    if (!IS_DEV_OR_TEST) {
        throw new Error("Local execution requires development or test environment (NODE_ENV)");
    }

    if (!config.enabled) {
        throw new Error(generateConfigurationErrorMessage());
    }
}

/**
 * Get the executable command for a given language from configuration
 */
function getLanguageExecutable(language: CodeLanguage, config: ExecutionConfig): string {
    switch (language) {
        case CodeLanguage.Shell:
            return config.runtime.executables.bash;
        case CodeLanguage.Python:
            return config.runtime.executables.python3;
        case CodeLanguage.Javascript:
            return config.runtime.executables.node;
        default:
            throw new Error(`Unsupported language for local execution: ${language}`);
    }
}

/**
 * Get file extension for a language
 */
function getLanguageExtension(language: CodeLanguage): string {
    switch (language) {
        case CodeLanguage.Shell:
            return ".sh";
        case CodeLanguage.Python:
            return ".py";
        case CodeLanguage.Javascript:
            return ".js";
        default:
            throw new Error(`Unknown file extension for language: ${language}`);
    }
}

/**
 * Prepare code content for execution in different languages
 */
function prepareCodeForExecution(code: string, language: CodeLanguage, input: unknown): string {
    switch (language) {
        case CodeLanguage.Shell:
            // For bash, we'll pass input as environment variables
            return code;
        case CodeLanguage.Python: {
            // For Python, we'll inject input as a variable and add execution wrapper
            const inputJson = JSON.stringify(input);
            return `
import json
import sys

# Injected input data
input_data = ${inputJson}

# User code
${code}

# Try to execute main function if it exists
if 'main' in globals():
    result = main(input_data)
    print(json.dumps(result))
elif '__main__' in globals():
    # Code executed directly
    pass
else:
    # No main function, just run the code
    pass
`;
        }
        case CodeLanguage.Javascript: {
            // For Node.js, we'll use require and JSON
            const inputJsonJs = JSON.stringify(input);
            return `
const inputData = ${inputJsonJs};

${code}

// Try to execute main function if it exists
if (typeof main === 'function') {
    const result = main(inputData);
    if (result && typeof result.then === 'function') {
        // Handle async functions
        result.then(res => console.log(JSON.stringify(res)))
              .catch(err => console.error(err));
    } else {
        console.log(JSON.stringify(result));
    }
}
`;
        }
        default:
            return code;
    }
}

/**
 * Prepare environment variables for bash execution
 */
function prepareBashEnvironment(input: unknown): Record<string, string> {
    const env: Record<string, string> = {};

    // Filter out undefined values from process.env
    Object.entries(process.env).forEach(([key, value]) => {
        if (value !== undefined) {
            env[key] = value;
        }
    });

    if (input && typeof input === "object") {
        // Convert input object to environment variables
        Object.entries(input).forEach(([key, value]) => {
            if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
                env[`INPUT_${key.toUpperCase()}`] = String(value);
            } else {
                env[`INPUT_${key.toUpperCase()}`] = JSON.stringify(value);
            }
        });
    } else {
        env.INPUT_DATA = JSON.stringify(input);
    }

    return env;
}

/**
 * Validates and sanitizes the working directory path against configuration
 */
function validateWorkingDirectory(workingDir: string, config: ExecutionConfig): string {
    const projectDir = process.env.PROJECT_DIR || process.cwd();
    const resolvedPath = resolve(projectDir, workingDir);

    // Check that the resolved path is within the project directory
    if (!resolvedPath.startsWith(resolve(projectDir))) {
        throw new Error("Working directory must be within the project directory");
    }

    // Check against allowed paths
    const isAllowed = config.security.allowedPaths.some(allowedPath => {
        const resolvedAllowed = resolve(projectDir, allowedPath);
        return resolvedPath.startsWith(resolvedAllowed);
    });

    if (!isAllowed) {
        throw new Error(`Working directory "${workingDir}" is not in allowed paths: ${config.security.allowedPaths.join(", ")}`);
    }

    // Check against denied paths
    const isDenied = config.security.deniedPaths.some(deniedPath => {
        const resolvedDenied = resolve(projectDir, deniedPath);
        return resolvedPath.startsWith(resolvedDenied);
    });

    if (isDenied) {
        throw new Error(`Working directory "${workingDir}" is in denied paths: ${config.security.deniedPaths.join(", ")}`);
    }

    // Ensure the directory exists
    try {
        statSync(resolvedPath);
    } catch {
        throw new Error(`Working directory does not exist: ${resolvedPath}`);
    }

    return resolvedPath;
}

/**
 * Manager for executing code locally (not in sandbox)
 * This is for development use only with strict safety checks
 */
export class LocalExecutionManager {
    private tempDir: string;
    private config: ExecutionConfig;

    constructor() {
        this.tempDir = mkdtempSync(join(tmpdir(), "vrooli-local-"));
        this.config = getExecutionConfig();
    }

    /**
     * Check if a language is supported for local execution
     */
    isLanguageSupported(language: CodeLanguage): boolean {
        return this.config.allowedLanguages.includes(language);
    }

    /**
     * Execute user code locally
     */
    async runUserCode(input: RunUserCodeInput): Promise<RunUserCodeOutput> {
        try {
            // Safety checks
            validateLocalExecution(this.config);

            if (!this.isLanguageSupported(input.codeLanguage)) {
                const supportedLangs = this.config.allowedLanguages.join(", ");
                throw new Error(`Language ${input.codeLanguage} is not supported for local execution. Allowed languages: ${supportedLangs}`);
            }

            const { code, codeLanguage, input: userInput, environmentConfig = {} } = input;

            // Apply environment config with defaults and security limits
            const requestedTimeout = environmentConfig.timeoutMs ?? this.config.defaults.timeoutMs;
            const timeout = Math.min(requestedTimeout, this.config.security.maxTimeoutMs);

            const requestedWorkingDir = environmentConfig.workingDirectory ?? this.config.defaults.workingDirectory;
            const workingDirectory = requestedWorkingDir === this.config.defaults.workingDirectory && this.config.defaults.workingDirectory === "./"
                ? this.tempDir  // Use temp dir for default "./" to avoid conflicts
                : validateWorkingDirectory(requestedWorkingDir, this.config);

            // Prepare code for execution
            const preparedCode = prepareCodeForExecution(code, codeLanguage, userInput);

            // Create temporary file
            const fileExtension = getLanguageExtension(codeLanguage);
            const tempFile = join(this.tempDir, `code_${Date.now()}${fileExtension}`);
            writeFileSync(tempFile, preparedCode);

            // Make bash scripts executable
            if (codeLanguage === CodeLanguage.Shell) {
                chmodSync(tempFile, EXECUTABLE_PERMISSION);
            }

            // Prepare execution environment
            const executable = getLanguageExecutable(codeLanguage, this.config);
            const args = [tempFile];

            // Merge environment variables with configuration
            const baseEnv = { ...this.config.runtime.environment };
            const userEnv = environmentConfig.environmentVariables ?? {};
            const execEnv = codeLanguage === CodeLanguage.Shell
                ? { ...baseEnv, ...prepareBashEnvironment(userInput), ...userEnv }
                : { ...process.env, ...baseEnv, ...userEnv };

            const maxBuffer = Math.min(
                this.config.security.maxOutputBufferKb * BYTES_PER_KB,
                MAX_OUTPUT_BUFFER_KB * BYTES_PER_KB,
            );

            const execOptions = {
                timeout,
                cwd: workingDirectory,
                env: execEnv,
                maxBuffer,
            };

            // Execute the code
            const { stdout, stderr } = await execFileAsync(executable, args, execOptions);

            // Clean up temp file
            unlinkSync(tempFile);

            // Parse output based on language
            let output: unknown;
            if (codeLanguage === CodeLanguage.Shell) {
                // For bash, return stdout as string (could be enhanced to parse JSON if needed)
                output = stdout.trim();
            } else {
                // For Python and JavaScript, try to parse JSON output
                try {
                    output = JSON.parse(stdout);
                } catch {
                    output = stdout.trim();
                }
            }

            // Include stderr in output if present (as warning)
            if (stderr.trim()) {
                if (typeof output === "object" && output !== null) {
                    (output as Record<string, unknown>)._stderr = stderr.trim();
                } else {
                    output = { result: output, _stderr: stderr.trim() };
                }
            }

            return { __type: "output", output };

        } catch (error) {
            // Clean up any temp files on error
            try {
                const files = await fs.readdir(this.tempDir);
                for (const file of files) {
                    if (file.startsWith("code_")) {
                        unlinkSync(join(this.tempDir, file));
                    }
                }
            } catch {
                // Ignore cleanup errors
            }

            const errorMessage = error instanceof Error ? error.message : String(error);
            return { __type: "error", error: errorMessage };
        }
    }

    /**
     * Clean up temporary directory
     */
    dispose(): void {
        try {
            rmSync(this.tempDir, { recursive: true, force: true });
        } catch {
            // Ignore cleanup errors
        }
    }
}

// Create a singleton instance
let localManagerInstance: LocalExecutionManager | null = null;

export function getLocalExecutionManager(): LocalExecutionManager {
    if (!localManagerInstance) {
        localManagerInstance = new LocalExecutionManager();
    }
    return localManagerInstance;
}

// Clean up on process exit
process.on("exit", () => {
    if (localManagerInstance) {
        localManagerInstance.dispose();
    }
});

