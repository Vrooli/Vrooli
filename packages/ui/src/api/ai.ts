import { AIServicesInfo, AITaskConfigJSON, DAYS_1_MS, importAIServiceConfig, importAITaskConfig } from "@local/shared";
import { apiUrlBase } from "utils/consts";

const AI_TASK_CONFIG_CACHE_KEY = "AI_CONFIG_CACHE";
const AI_TASK_CONFIG_CACHE_DURATION = DAYS_1_MS;

const AI_SERVICE_CACHE_KEY = "AI_SERVICE_CACHE";
const AI_SERVICE_CACHE_DURATION = DAYS_1_MS;

type AIServiceCacheData = {
    config: AIServicesInfo,
    timestamp: number,
}

type AITaskCacheData = {
    config: AITaskConfigJSON,
    timestamp: number,
    cachedLanguage: string
};

type AICacheData = {
    service: AIServiceCacheData | null,
    task: AITaskCacheData | null,
}

export function getExistingAIConfig(): AICacheData | null {
    let serviceConfig: AIServiceCacheData | null = null;
    let taskConfig: AITaskCacheData | null = null;

    try {
        const cachedServiceData = localStorage.getItem(AI_SERVICE_CACHE_KEY);
        if (cachedServiceData) {
            serviceConfig = JSON.parse(cachedServiceData);
        }
    } catch (error) {
        console.error("Error fetching existing service configuration", error);
    }
    try {
        const cachedTaskData = localStorage.getItem(AI_TASK_CONFIG_CACHE_KEY);
        if (cachedTaskData) {
            taskConfig = JSON.parse(cachedTaskData);
        }
    } catch (error) {
        console.error("Error fetching existing task configuration", error);
    }

    return {
        service: serviceConfig,
        task: taskConfig,
    };
}

/**
 * Checks if the cached data has expired based on the timestamp and duration.
 * @param timestamp The time when the data was cached
 * @param duration The duration (in milliseconds) after which the cache expires
 * @returns True if expired, false otherwise
 */
function isExpired(timestamp: number, duration: number): boolean {
    return (Date.now() - timestamp) > duration;
}


/**
 * Fetches the LLM configuration for the given language. This is 
 * used to determine task labels and other task-specific information.
 * 
 * NOTE: You should call this function whenever the user's language changes.
 * @param language The language to fetch the configuration for
 */
export async function fetchAIConfig(language: string): Promise<AICacheData | null> {
    // const configData = getExistingAIConfig();

    // if (configData) {
    //     const { config, timestamp, cachedLanguage } = configData;
    //     const isExpired = Date.now() - timestamp > AI_TASK_CONFIG_CACHE_DURATION;
    //     if (!isExpired && cachedLanguage === language && typeof config === "object" && Object.keys(config).length > 0) {
    //         return config;
    //     }
    // }

    // try {
    //     const config = await importConfig(language, console, apiUrlBase.replace("/api", ""));
    //     const cacheData = JSON.stringify({
    //         config,
    //         timestamp: Date.now(),
    //         cachedLanguage: language,
    //     });
    //     localStorage.setItem(AI_TASK_CONFIG_CACHE_KEY, cacheData);
    //     return config;
    // } catch (error) {
    //     console.error(`Error fetching configuration: ${Object.prototype.hasOwnProperty.call(error, "message") ? (error as { message: string }).message : error}`);
    //     return null;
    // }
    const configData = getExistingAIConfig();

    let shouldFetchTaskInfo = true;
    if (configData?.task) {
        const { config, timestamp, cachedLanguage } = configData.task;
        const isLanguageValid =
            !isExpired(timestamp, AI_TASK_CONFIG_CACHE_DURATION) &&  // Check if the cache has expired
            cachedLanguage === language &&  // Check if the cached language matches the requested language
            typeof config === "object" &&  // Check if the config is an object
            Object.keys(config).length > 0;  // Check if the config is not empty
        if (isLanguageValid) {
            shouldFetchTaskInfo = false;
        }
    }

    let shouldFetchServiceInfo = true;
    if (configData?.service) {
        const { config, timestamp } = configData.service;
        const isServiceValid =
            !isExpired(timestamp, AI_SERVICE_CACHE_DURATION) &&  // Check if the cache has expired
            typeof config === "object" &&  // Check if the config is an object
            Object.keys(config).length > 0;  // Check if the config is not empty
        if (isServiceValid) {
            shouldFetchServiceInfo = false;
        }
    }

    // If both configs are valid, return them
    if (!shouldFetchServiceInfo && !shouldFetchTaskInfo) {
        return configData;
    }

    // Prepare fetch promises
    const fetchPromises: Promise<void>[] = [];
    const urlBase = apiUrlBase.replace("/api", "");

    let fetchedServiceInfo: AIServiceCacheData | null = null;
    let fetchedTaskConfig: AITaskCacheData | null = null;

    if (shouldFetchServiceInfo) {
        fetchPromises.push(
            importAIServiceConfig(console, urlBase)
                .then((serviceConfig) => {
                    fetchedServiceInfo = {
                        config: serviceConfig,
                        timestamp: Date.now(),
                    };
                    localStorage.setItem(AI_SERVICE_CACHE_KEY, JSON.stringify(fetchedServiceInfo));
                })
                .catch((error) => {
                    console.error("Error fetching service configuration:", error);
                }),
        );
    }

    if (shouldFetchTaskInfo) {
        fetchPromises.push(
            importAITaskConfig(language, console, urlBase)
                .then((taskConfig) => {
                    fetchedTaskConfig = {
                        config: taskConfig as unknown as AITaskConfigJSON,
                        timestamp: Date.now(),
                        cachedLanguage: language,
                    };
                    localStorage.setItem(AI_TASK_CONFIG_CACHE_KEY, JSON.stringify(fetchedTaskConfig));
                })
                .catch((error) => {
                    console.error("Error fetching task configuration:", error);
                }),
        );
    }

    // Await all fetches
    await Promise.all(fetchPromises);

    // After fetching, construct the updated config data
    const result: AICacheData = {
        service: fetchedServiceInfo || configData?.service || null,
        task: fetchedTaskConfig || configData?.task || null,
    };

    return result;
}

/**
 * Used to initialize the LLM configuration. Use this when the server version changes.
 */
export function invalidateAIConfigCache() {
    const configData = getExistingAIConfig();
    localStorage.removeItem(AI_TASK_CONFIG_CACHE_KEY);
    if (configData?.task?.cachedLanguage) {
        fetchAIConfig(configData.task.cachedLanguage);
    }
}
