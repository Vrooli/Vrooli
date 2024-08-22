/* eslint-disable @typescript-eslint/ban-ts-comment */
import { LlmTask } from "../api/generated/graphqlTypes";
import { PassableLogger } from "../consts/commonTypes";
import { DEFAULT_LANGUAGE } from "../consts/ui";
import { CommandToTask, LanguageModelResponseMode, LlmTaskConfig, LlmTaskStructuredConfig, LlmTaskUnstructuredConfig } from "./types";

export async function getLlmConfigLocation(): Promise<string> {
    // Test environment - Use absolute path
    if (process.env.NODE_ENV === "test") {
        const path = await import("path");
        const url = await import("url");
        const dirname = path.dirname(url.fileURLToPath(import.meta.url));
        return path.join(dirname, "configs");
    }
    // Node.js environment - Use relative path
    else if (process.env.npm_package_name === "@local/server") {
        return "./configs";
    }
    // Browser environment - Fetch from server
    else {
        return "http://localhost:5329/llm/configs";
    }
}

async function loadFile(modulePath: string) {
    const module = await import(/* @vite-ignore */modulePath);
    return module;
}

/**
 * Dynamically imports the configuration for the specified language.
 */
export async function importConfig(
    language: string,
    logger: PassableLogger,
): Promise<LlmTaskConfig> {
    const config_location = await getLlmConfigLocation();
    try {
        return (await loadFile(`${config_location}/${language}.js`)).config;
    } catch (error) {
        logger.error(`Configuration for language ${language} not found. Falling back to ${DEFAULT_LANGUAGE}.`, { trace: "0309", config_location });
        return (await loadFile(`${config_location}/${DEFAULT_LANGUAGE}`)).config;
    }
}

/**
 * Dynamically imports the `commandToTask` function, which converts a command and 
 * action to a task name.
 */
export async function importCommandToTask(
    language: string,
    logger: PassableLogger,
): Promise<CommandToTask> {
    const config_location = await getLlmConfigLocation();
    try {
        return (await loadFile(`${config_location}/${language}.js`)).commandToTask;
    } catch (error) {
        logger.error(`Command to task function for language ${language} not found. Falling back to ${DEFAULT_LANGUAGE}.`, { trace: "0041" });
        return (await loadFile(`${config_location}/${DEFAULT_LANGUAGE}`)).commandToTask;
    }
}

/**
 * @returns The unstructured configuration object for the given task, 
 * in the best language available for the user
 */
export async function getUnstructuredTaskConfig(
    task: LlmTask | `${LlmTask}`,
    language: string = DEFAULT_LANGUAGE,
    logger: PassableLogger,
): Promise<LlmTaskUnstructuredConfig> {
    const unstructuredConfig = await importConfig(language, logger);
    const taskConfig = unstructuredConfig[task];

    // Fallback to a default message if the task is not found in the config
    if (!taskConfig || typeof taskConfig !== "function") {
        logger.error(`Task ${task} was invalid or not found in the configuration for language ${language}`, { trace: "0305" });
        return {} as LlmTaskUnstructuredConfig;
    }

    return taskConfig();
}

type GetStructuredTaskConfigProps = {
    /** Whether to force the LLM to respond with a command */
    force: boolean;
    /** The language to use for the configuration */
    language: string;
    /** Logger to use for logging */
    logger: PassableLogger;
    /** The mode to generate the response in */
    mode: LanguageModelResponseMode;
    /** The task to get the structured configuration for */
    task: LlmTask | `${LlmTask}`;
}

/**
 * Creates a structured configuration object to provide as the system message 
 * or first message to the LLM, based on the given task, requested response mode, 
 * and other parameters.
 */
export async function getStructuredTaskConfig({
    force,
    language,
    logger,
    mode,
    task,
}: GetStructuredTaskConfigProps): Promise<LlmTaskStructuredConfig> {
    const unstructuredConfig = await importConfig(language, logger);
    const taskConfig = unstructuredConfig[task];

    // Fallback to a default message if the task is not found in the config
    if (!taskConfig || typeof taskConfig !== "function") {
        logger.error(`Task ${task} was invalid or not found in the configuration for language ${language}`, { trace: "0046" });
        return {} as LlmTaskStructuredConfig;
    }

    // Build context based on the mode and force parameters
    if (mode === "json") return unstructuredConfig.__construct_context_json(taskConfig());
    if (force === true) return unstructuredConfig.__construct_context_text_force(taskConfig());
    return unstructuredConfig.__construct_context_text(taskConfig());
}
