import path from "path";
import { fileURLToPath } from "url";
import { logger } from "../../events";
import { BotSettings } from "./service";

export const DEFAULT_LANGUAGE = "en";

export const llmActions = [
    "ApiCreate",
    "ApiDelete",
    "ApiFind",
    "ApiUpdate",
    "BotCreate",
    "BotDelete",
    "BotFind",
    "BotUpdate",
    "MembersCreate",
    "MembersDelete",
    "MembersFind",
    "MembersUpdate",
    "NoteCreate",
    "NoteDelete",
    "NoteFind",
    "NoteUpdate",
    "ProjectCreate",
    "ProjectDelete",
    "ProjectFind",
    "ProjectUpdate",
    "ReminderCreate",
    "ReminderDelete",
    "ReminderFind",
    "ReminderUpdate",
    "RoleCreate",
    "RoleDelete",
    "RoleFind",
    "RoleUpdate",
    "RoutineCreate",
    "RoutineDelete",
    "RoutineFind",
    "RoutineUpdate",
    "ScheduleCreate",
    "ScheduleDelete",
    "ScheduleFind",
    "ScheduleUpdate",
    "SmartContractCreate",
    "SmartContractDelete",
    "SmartContractFind",
    "SmartContractUpdate",
    "StandardCreate",
    "StandardDelete",
    "StandardFind",
    "StandardUpdate",
    "Start",
    "TeamCreate",
    "TeamDelete",
    "TeamFind",
    "TeamUpdate",
] as const;
export type LlmAction = typeof llmActions[number];
export type LlmActionConfig = Record<string, any>;
export type LlmConfig = Record<LlmAction, (botSettings: BotSettings) => LlmActionConfig> & { [x: string]: any };

const dirname = path.dirname(fileURLToPath(import.meta.url));
export const LLM_CONFIG_LOCATION = `${dirname}/configs`;

/**
 * Dynamically imports the configuration for the specified language.
 */
export const importConfig = async (language: string): Promise<LlmConfig> => {
    try {
        const config = await import(`${LLM_CONFIG_LOCATION}/${language}`);
        return config.default;
    } catch (error) {
        logger.error(`Configuration for language ${language} not found. Falling back to ${DEFAULT_LANGUAGE}.`, { trace: "0309" });
        const defaultConfig = await import(`${LLM_CONFIG_LOCATION}/${DEFAULT_LANGUAGE}`);
        return defaultConfig?.default;
    }
}


/**
 * @returns The configuration object for the given action, 
 * in the best language available for the user
 */
export const getActionConfig = async (
    action: LlmAction,
    botSettings: BotSettings,
    language: string = DEFAULT_LANGUAGE,
): Promise<LlmActionConfig> => {
    const config = await importConfig(language);
    const actionConfig = config[action];

    // Fallback to a default message if the action is not found in the config
    if (!actionConfig || typeof actionConfig !== "function") {
        logger.error(`Action ${action} was invalid or not found in the configuration for language ${language}`, { trace: "0305" });
        return {} as LlmActionConfig;
    }

    return actionConfig(botSettings);
};
