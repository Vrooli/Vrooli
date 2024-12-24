/* eslint-disable @typescript-eslint/ban-ts-comment */
import { LlmTask } from "../api/types";
import { PassableLogger } from "../consts/commonTypes";
import { DEFAULT_LANGUAGE } from "../consts/ui";
import { pascalCase } from "../utils/casing";
import { type AIServicesInfo } from "./services";
import { AITaskConfig, AITaskConfigBuilder, AITaskUnstructuredConfig, CommandToTask, LanguageModelResponseMode, LlmTaskStructuredConfig, TaskNameMap } from "./types";

const CONFIG_RELATIVE_PATH = "../../dist/ai/configs";

/**
 * Determines the correct location to fetch the AI configuration from, depending on the environment.
 */
export async function getAIConfigLocation(urlBase?: string): Promise<{ location: string, shouldFetch: boolean }> {
    // Test environment - Use absolute path
    if (process.env.NODE_ENV === "test") {
        const path = await import("path");
        const url = await import("url");
        const dirname = path.dirname(url.fileURLToPath(import.meta.url));
        return { location: path.join(dirname, CONFIG_RELATIVE_PATH), shouldFetch: false };
    }
    // Node.js environment - Use relative path
    else if (process.env.npm_package_name === "@local/server") {
        return { location: CONFIG_RELATIVE_PATH, shouldFetch: false };
    }
    // Browser environment - Fetch from server
    else {
        return { location: `${urlBase || "http://localhost:5329"}/ai/configs`, shouldFetch: true };
    }
}

/**
 * Helper function to import or fetch the configuration file.
 */
async function loadFile(path: string, shouldFetch: boolean) {
    if (shouldFetch) {
        const response = await fetch(path);
        if (!response.ok) {
            throw new Error(`Configuration at ${path} not found`);
        }
        return await response.json();
    } else {
        const module = await import(/* @vite-ignore */path);
        return module;
    }
}

export async function importHelper(
    filename: string,
    subdirectory: "services" | "tasks",
    logger: PassableLogger,
    urlBase?: string,
): Promise<{
    file: Record<string, unknown>,
    extension: string,
}> {
    const { location, shouldFetch } = await getAIConfigLocation(urlBase);
    const extension = shouldFetch ? "json" : "js";
    try {
        const path = `${location}/${subdirectory}/${filename}.${extension}`;
        const file = await loadFile(path, shouldFetch);
        return { file, extension };
    } catch (error) {
        logger.error(`Configuration ${filename} not found. Falling back to ${DEFAULT_LANGUAGE}.`, { trace: "0309", location, shouldFetch });
        const path = `${location}/${subdirectory}/${DEFAULT_LANGUAGE}.${extension}`;
        const file = await loadFile(path, shouldFetch);
        return { file, extension };
    }
}

/**
 * Dynamically imports the AI services configuration
 */
export async function importAIServiceConfig(
    logger: PassableLogger,
    urlBase?: string,
): Promise<AIServicesInfo> {
    const { file, extension } = await importHelper("services", "services", logger, urlBase);
    return extension === "json" ? file as AIServicesInfo : (file as { aiServicesInfo: AIServicesInfo }).aiServicesInfo;
}

/**
 * Dynamically imports the configuration for the specified language.
 */
export async function importAITaskConfig(
    language: string,
    logger: PassableLogger,
    urlBase?: string,
): Promise<AITaskConfig> {
    const { file, extension } = await importHelper(language, "tasks", logger, urlBase);
    return extension === "json" ? file as AITaskConfig : (file as { config: AITaskConfig }).config;
}

/**
 * Dynamically imports the configuration builder for the specified language.
 */
export async function importAITaskBuilder(
    language: string,
    logger: PassableLogger,
    urlBase?: string,
): Promise<AITaskConfigBuilder> {
    const { file, extension } = await importHelper(language, "tasks", logger, urlBase);
    return extension === "json" ? {} as AITaskConfigBuilder : (file as { builder: AITaskConfigBuilder }).builder;
}

/**
 * Dynamically imports the `commandToTask` function, which converts a command and 
 * action to a task name.
 */
export async function importCommandToTask(
    language: string,
    logger: PassableLogger,
): Promise<CommandToTask> {
    // Fetch the task name map
    let map: TaskNameMap = { command: {}, action: {} };
    try {
        const config = await importAITaskConfig(language, logger);
        map = config.__task_name_map;
    } catch (error) {
        logger.error(`Unable to fetch task name map for language ${language}`, { trace: "0041" });
    }
    // Wrap in a function that converts commands to tasks
    return function commandToTask(command: Parameters<CommandToTask>[0], action: Parameters<CommandToTask>[1]): ReturnType<CommandToTask> {
        let result: string;
        // if (action) result = `${pascalCase(command)}${pascalCase(action)}`;
        if (action) result = `${map.command[command] || pascalCase(command)}${map.action[action] || pascalCase(action)}`;
        else result = map.command[command] || pascalCase(command);
        if (Object.keys(LlmTask).includes(result)) return result as LlmTask;
        return null;
    };
}

/**
 * @returns The unstructured configuration object for the given task, 
 * in the best language available for the user
 */
export async function getUnstructuredTaskConfig(
    task: LlmTask | `${LlmTask}`,
    language: string = DEFAULT_LANGUAGE,
    logger: PassableLogger,
): Promise<AITaskUnstructuredConfig> {
    const taskConfig = (await importAITaskConfig(language, logger))[task];

    // Fallback to a default message if the task is not found in the config
    if (!taskConfig || typeof taskConfig !== "function") {
        logger.error(`Task ${task} was invalid or not found in the configuration for language ${language}`, { trace: "0305" });
        return {} as AITaskUnstructuredConfig;
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
    const taskConfig = (await importAITaskConfig(language, logger))[task];
    const builder = await importAITaskBuilder(language, logger);

    // Fallback to a default message if the task is not found in the config
    if (!taskConfig || typeof taskConfig !== "function") {
        logger.error(`Task ${task} was invalid or not found in the configuration for language ${language}`, { trace: "0046" });
        return {} as LlmTaskStructuredConfig;
    }

    // Build context based on the mode and force parameters
    if (mode === "json") return builder.__construct_context_json(taskConfig());
    if (force === true) return builder.__construct_context_text_force(taskConfig());
    return builder.__construct_context_text(taskConfig());
}
