import { DAYS_1_MS, LlmTaskConfigJSON, importConfig } from "@local/shared";
import { apiUrlBase } from "utils/consts";

const LLM_CONFIG_CACHE_KEY = "configCache";
const LLM_CONFIG_CACHE_DURATION = DAYS_1_MS;

type LlmCacheData = {
    config: LlmTaskConfigJSON,
    timestamp: number,
    cachedLanguage: string
};

export function getExistingLlmConfig(): LlmCacheData | null {
    try {
        const cachedData = localStorage.getItem(LLM_CONFIG_CACHE_KEY);
        if (cachedData) {
            return JSON.parse(cachedData);
        }
    } catch (error) {
        console.error(`Error fetching existing configuration: ${Object.prototype.hasOwnProperty.call(error, "message") ? (error as { message: string }).message : error}`);
    }
    return null;
}

/**
 * Fetches the LLM configuration for the given language. This is 
 * used to determine task labels and other task-specific information.
 * 
 * NOTE: You should call this function whenever the user's language changes.
 * @param language The language to fetch the configuration for
 */
export async function fetchLlmConfig(language: string): Promise<object | null> {
    const configData = getExistingLlmConfig();

    if (configData) {
        const { config, timestamp, cachedLanguage } = configData;
        const isExpired = Date.now() - timestamp > LLM_CONFIG_CACHE_DURATION;
        if (!isExpired && cachedLanguage === language && typeof config === "object" && Object.keys(config).length > 0) {
            return config;
        }
    }

    try {
        const config = await importConfig(language, console, apiUrlBase.replace("/api", ""));
        const cacheData = JSON.stringify({
            config,
            timestamp: Date.now(),
            cachedLanguage: language,
        });
        localStorage.setItem(LLM_CONFIG_CACHE_KEY, cacheData);
        return config;
    } catch (error) {
        console.error(`Error fetching configuration: ${Object.prototype.hasOwnProperty.call(error, "message") ? (error as { message: string }).message : error}`);
        return null;
    }
}

/**
 * Used to initialize the LLM configuration. Use this when the server version changes.
 */
export function invalidateLlmConfigCache() {
    const configData = getExistingLlmConfig();
    localStorage.removeItem(LLM_CONFIG_CACHE_KEY);
    if (configData?.cachedLanguage) {
        fetchLlmConfig(configData.cachedLanguage);
    }
}
