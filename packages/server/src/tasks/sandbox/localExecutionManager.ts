import { CodeLanguage } from "@vrooli/shared";
import { execFile } from "child_process";
import { promises as fs, chmodSync, mkdtempSync, rmSync, statSync, unlinkSync, writeFileSync } from "fs";
import { tmpdir } from "os";
import { join, resolve } from "path";
import { promisify } from "util";
import { DEFAULT_JOB_TIMEOUT_MS } from "./consts.js";
import { generateEnvironmentRequirementsError, isLanguageSupportedInEnvironment } from "./executionEnvironmentSupport.js";
import { type RunUserCodeInput, type RunUserCodeOutput } from "./types.js";

const execFileAsync = promisify(execFile);

// Environment variable for enabling local execution
const LOCAL_EXECUTION_ENABLED = process.env.VROOLI_ENABLE_LOCAL_EXECUTION === "true";
const IS_DEV_OR_TEST = process.env.NODE_ENV === "development" || process.env.NODE_ENV === "test";

// Constants for file permissions and buffer sizes
const EXECUTABLE_PERMISSION = 0o755;
const MAX_OUTPUT_BUFFER_KB = 1024;
const BYTES_PER_KB = 1024;


/**
 * Safety checks for local execution
 */
function validateLocalExecution(): void {
    if (!IS_DEV_OR_TEST || !LOCAL_EXECUTION_ENABLED) {
        throw new Error(generateEnvironmentRequirementsError("local"));
    }
}

/**
 * Get the executable command for a given language
 */
function getLanguageExecutable(language: CodeLanguage): string {
    switch (language) {
        case CodeLanguage.Shell:
            return "bash";
        case CodeLanguage.Python:
            return "python3";
        case CodeLanguage.Javascript:
            return "node";
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
    const env = { ...process.env };
    
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
 * Validates and sanitizes the working directory path
 */
function validateWorkingDirectory(workingDir: string): string {
    // Ensure it's within the project directory for security
    const projectDir = process.env.PROJECT_DIR || process.cwd();
    const resolvedPath = resolve(projectDir, workingDir);
    
    // Check that the resolved path is within the project directory
    if (!resolvedPath.startsWith(resolve(projectDir))) {
        throw new Error("Working directory must be within the project directory");
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

    constructor() {
        this.tempDir = mkdtempSync(join(tmpdir(), "vrooli-local-"));
    }

    /**
     * Check if a language is supported for local execution
     */
    isLanguageSupported(language: CodeLanguage): boolean {
        return isLanguageSupportedInEnvironment(language, "local");
    }

    /**
     * Execute user code locally
     */
    async runUserCode(input: RunUserCodeInput): Promise<RunUserCodeOutput> {
        try {
            // Safety checks
            validateLocalExecution();
            
            if (!this.isLanguageSupported(input.codeLanguage)) {
                throw new Error(`Language ${input.codeLanguage} is not supported for local execution`);
            }

            const { code, codeLanguage, input: userInput, environmentConfig = {} } = input;
            
            // Apply environment config with defaults
            const timeout = environmentConfig.timeoutMs ?? DEFAULT_JOB_TIMEOUT_MS;
            const workingDirectory = environmentConfig.workingDirectory 
                ? validateWorkingDirectory(environmentConfig.workingDirectory)
                : this.tempDir;
            
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
            const executable = getLanguageExecutable(codeLanguage);
            const args = [tempFile];
            const execOptions = {
                timeout,
                cwd: workingDirectory,
                env: codeLanguage === CodeLanguage.Shell 
                    ? prepareBashEnvironment(userInput)
                    : { ...process.env, ...environmentConfig.environmentVariables },
                maxBuffer: MAX_OUTPUT_BUFFER_KB * BYTES_PER_KB, // 1MB output limit
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

