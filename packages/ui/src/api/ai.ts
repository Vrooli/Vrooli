import { DAYS_1_MS, type AIServicesInfo, type AITaskConfigJSON, type LlmServiceModel, type ModelInfo, type TranslationKeyService } from "@vrooli/shared";
import { apiUrlBase } from "../utils/consts.js";

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
    const configData = getExistingAIConfig();

    let shouldFetchTaskInfo = true;
    if (configData?.task) {
        const { config, timestamp, cachedLanguage } = configData.task;
        const isLanguageValid =
            !isExpired(timestamp, AI_TASK_CONFIG_CACHE_DURATION) &&
            cachedLanguage === language &&
            typeof config === "object" &&
            Object.keys(config).length > 0;
        if (isLanguageValid) {
            shouldFetchTaskInfo = false;
        }
    }

    let shouldFetchServiceInfo = true;
    if (configData?.service) {
        const { config, timestamp } = configData.service;
        const isServiceValid =
            !isExpired(timestamp, AI_SERVICE_CACHE_DURATION) &&
            typeof config === "object" &&
            Object.keys(config).length > 0;
        if (isServiceValid) {
            shouldFetchServiceInfo = false;
        }
    }

    if (!shouldFetchServiceInfo && !shouldFetchTaskInfo) {
        return configData;
    }

    const fetchPromises: Promise<void>[] = [];
    const urlBase = apiUrlBase.replace("/api", "");

    let fetchedServiceInfo: AIServiceCacheData | null = configData?.service || null;
    let fetchedTaskConfig: AITaskCacheData | null = configData?.task || null;

    if (shouldFetchServiceInfo) {
        const serviceConfigUrl = `${urlBase}/ai/configs/services.json`;
        fetchPromises.push(
            fetch(serviceConfigUrl)
                .then(response => {
                    if (!response.ok) {
                        throw new Error(`Failed to fetch service configuration: ${response.statusText}`);
                    }
                    return response.json();
                })
                .then((serviceConfig: AIServicesInfo) => {
                    fetchedServiceInfo = {
                        config: serviceConfig,
                        timestamp: Date.now(),
                    };
                    localStorage.setItem(AI_SERVICE_CACHE_KEY, JSON.stringify(fetchedServiceInfo));
                })
                .catch((error) => {
                    console.error(`Error fetching service configuration from ${serviceConfigUrl}:`, error);
                }),
        );
    }

    if (shouldFetchTaskInfo) {
        const taskConfigUrl = `${urlBase}/ai/configs/${language}.json`;
        fetchPromises.push(
            fetch(taskConfigUrl)
                .then(response => {
                    if (!response.ok) {
                        throw new Error(`Failed to fetch task configuration: ${response.statusText}`);
                    }
                    return response.json();
                })
                .then((taskConfig: AITaskConfigJSON) => {
                    fetchedTaskConfig = {
                        config: taskConfig,
                        timestamp: Date.now(),
                        cachedLanguage: language,
                    };
                    localStorage.setItem(AI_TASK_CONFIG_CACHE_KEY, JSON.stringify(fetchedTaskConfig));
                })
                .catch((error) => {
                    console.error(`Error fetching task configuration from ${taskConfigUrl}:`, error);
                }),
        );
    }

    if (fetchPromises.length > 0) {
        await Promise.all(fetchPromises);
    }

    const result: AICacheData = {
        service: fetchedServiceInfo,
        task: fetchedTaskConfig,
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

export function getFallbackModel(servicesInfo: AIServicesInfo | null | undefined): string | null {
    if (!servicesInfo) {
        return null;
    }
    const defaultServiceId = servicesInfo.defaultService;
    if (!defaultServiceId) {
        return null;
    }
    const defaultService = servicesInfo.services[defaultServiceId];
    if (!defaultService) {
        return null;
    }
    return defaultService.defaultModel;
}

export type AvailableModel = {
    name: TranslationKeyService;
    description: TranslationKeyService;
    value: LlmServiceModel;
};

export function getAvailableModels(servicesInfo: AIServicesInfo | null | undefined): AvailableModel[] {
    const availableModels: AvailableModel[] = [];
    if (!servicesInfo) {
        return availableModels;
    }

    for (const serviceName of Object.keys(servicesInfo.services) as Array<keyof AIServicesInfo["services"]>) {
        const service = servicesInfo.services[serviceName];
        if (service.enabled) {
            const models = service.models as Record<string, ModelInfo>;
            for (const modelValue in models) {
                const model = models[modelValue as string];
                if (model.enabled) {
                    availableModels.push({
                        name: model.name,
                        description: model.descriptionShort,
                        value: modelValue as LlmServiceModel,
                    });
                }
            }
        }
    }
    return availableModels;
}
