import { CodeLanguage, type ExecutionEnvironment } from "@vrooli/shared";
import { getExecutionConfig } from "./executionConfig.js";

/**
 * Mapping of execution environments to their supported languages
 */
export const EXECUTION_ENVIRONMENT_SUPPORT: Record<ExecutionEnvironment, {
    languages: CodeLanguage[];
    description: string;
    requirements?: string[];
}> = {
    sandbox: {
        languages: [CodeLanguage.Javascript],
        description: "Secure, isolated JavaScript execution using V8 isolates",
        requirements: ["No additional requirements - always available"],
    },
    local: {
        languages: [CodeLanguage.Javascript, CodeLanguage.Shell, CodeLanguage.Python],
        description: "Local execution for development and testing",
        requirements: [
            "NODE_ENV must be 'development' or 'test'",
            "VROOLI_ENABLE_LOCAL_EXECUTION must be set to 'true'",
            "Runtime executables must be available (node, bash, python3)",
        ],
    },
};

/**
 * Check if a language is supported in a given execution environment
 * For local environment, also checks the configuration
 */
export function isLanguageSupportedInEnvironment(
    language: CodeLanguage,
    environment: ExecutionEnvironment,
): boolean {
    if (environment === "local") {
        const config = getExecutionConfig();
        return config.enabled && config.allowedLanguages.includes(language);
    }
    return EXECUTION_ENVIRONMENT_SUPPORT[environment]?.languages.includes(language) ?? false;
}

/**
 * Get supported languages for an execution environment
 */
export function getSupportedLanguagesForEnvironment(environment: ExecutionEnvironment): CodeLanguage[] {
    return EXECUTION_ENVIRONMENT_SUPPORT[environment]?.languages ?? [];
}

/**
 * Get all supported execution environments
 */
export function getSupportedExecutionEnvironments(): ExecutionEnvironment[] {
    return Object.keys(EXECUTION_ENVIRONMENT_SUPPORT) as ExecutionEnvironment[];
}

/**
 * Generate a helpful error message for unsupported language/environment combinations
 */
export function generateUnsupportedLanguageError(
    language: CodeLanguage,
    environment: ExecutionEnvironment,
): string {
    const envSupport = EXECUTION_ENVIRONMENT_SUPPORT[environment];

    if (!envSupport) {
        const supportedEnvs = getSupportedExecutionEnvironments().join(", ");
        return `Unsupported execution environment '${environment}'. Supported environments: ${supportedEnvs}`;
    }

    const supportedLanguages = envSupport.languages.join(", ");
    let errorMessage = `Language '${language}' is not supported in execution environment '${environment}'. `;
    errorMessage += `Supported languages for ${environment}: ${supportedLanguages}.`;

    // Add requirements if they exist
    if (envSupport.requirements && envSupport.requirements.length > 0) {
        errorMessage += ` Requirements for ${environment}: ${envSupport.requirements.join(", ")}.`;
    }

    // Suggest alternative environments if the language is supported elsewhere
    const alternativeEnvs = getSupportedExecutionEnvironments().filter(env =>
        env !== environment && isLanguageSupportedInEnvironment(language, env),
    );

    if (alternativeEnvs.length > 0) {
        errorMessage += ` Language '${language}' is supported in: ${alternativeEnvs.join(", ")}.`;
    }

    return errorMessage;
}

/**
 * Generate a helpful error message for environment requirements not being met
 */
export function generateEnvironmentRequirementsError(environment: ExecutionEnvironment): string {
    const envSupport = EXECUTION_ENVIRONMENT_SUPPORT[environment];

    if (!envSupport || !envSupport.requirements) {
        return `Requirements not met for execution environment '${environment}'`;
    }

    let message = `Requirements not met for execution environment '${environment}'. Required: ${envSupport.requirements.join(", ")}`;
    
    if (environment === "local") {
        message += "\n\nFor detailed configuration, see the execution section in .vrooli/service.json or examples in .vrooli/examples/components/";
    }
    
    return message;
}

/**
 * Get a summary of all execution environment capabilities
 */
export function getExecutionEnvironmentSummary(): string {
    const environments = getSupportedExecutionEnvironments();
    const summaries = environments.map(env => {
        const support = EXECUTION_ENVIRONMENT_SUPPORT[env];
        const languages = support.languages.join(", ");
        const requirements = support.requirements ? ` (Requirements: ${support.requirements.join(", ")})` : "";
        return `- ${env}: ${support.description}. Languages: ${languages}${requirements}`;
    });

    return `Supported execution environments:\n${summaries.join("\n")}`;
}

