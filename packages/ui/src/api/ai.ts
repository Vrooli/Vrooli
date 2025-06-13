import { DAYS_1_MS, type AIServicesInfo, type AITaskConfigJSON, type LlmServiceModel, type ModelInfo, type TranslationKeyService } from "@vrooli/shared";
import { publishModelNotification } from "../hooks/useModelNotifications.js";
import { useModelPreferencesStore } from "../stores/modelPreferencesStore.js";
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
        // Service configuration fetch failed - will use cached or defaults
    }
    try {
        const cachedTaskData = localStorage.getItem(AI_TASK_CONFIG_CACHE_KEY);
        if (cachedTaskData) {
            taskConfig = JSON.parse(cachedTaskData);
        }
    } catch (error) {
        // Task configuration fetch failed - will use cached or defaults
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
                    // Service configuration fetch failed - will try again next time
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
                    // Task configuration fetch failed - will try again next time
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

/**
 * Converts an AvailableModel to the LlmServiceModel for storage
 */
export function availableModelToStorageModel(model: AvailableModel): LlmServiceModel {
    return model.value;
}

/**
 * Extracts model values from an array of AvailableModels for validation
 */
export function getAvailableModelValues(availableModels: AvailableModel[]): LlmServiceModel[] {
    return availableModels.map(model => model.value);
}

/**
 * Converts a stored LlmServiceModel back to an AvailableModel, if it exists in available models
 */
export function storageModelToAvailableModel(
    modelValue: LlmServiceModel, 
    availableModels: AvailableModel[]
): AvailableModel | null {
    return availableModels.find(model => model.value === modelValue) || null;
}

/**
 * Validates that a stored model preference is still available
 */
export function validateStoredModelPreference(
    storedModel: LlmServiceModel | null,
    availableModels: AvailableModel[]
): AvailableModel | null {
    if (!storedModel) return null;
    return storageModelToAvailableModel(storedModel, availableModels);
}

/**
 * Gets the user's preferred model as an AvailableModel, with validation and fallbacks.
 * 
 * @param availableModels - Current list of available models for validation
 * @returns The user's preferred model or a fallback
 */
export function getPreferredAvailableModel(availableModels: AvailableModel[]): AvailableModel | null {
    // Get the user's stored preference
    const storedPreference = useModelPreferencesStore.getState().preferredModel;
    
    // Validate the stored preference against available models
    if (storedPreference) {
        const validatedModel = validateStoredModelPreference(storedPreference, availableModels);
        if (validatedModel) {
            return validatedModel;
        }
        // Clear invalid preference and notify user
        useModelPreferencesStore.getState().clearPreferredModel();
        
        // Find the fallback model that will be used
        const fallbackModel = availableModels.length > 0 ? availableModels[0] : null;
        if (fallbackModel) {
            publishModelNotification({
                type: "model-unavailable",
                previousModel: storedPreference,
                newModel: fallbackModel.name,
            });
        }
    }

    // Fall back to the service default model
    const servicesInfo = getExistingAIConfig()?.service?.config;
    const fallbackModelValue = getFallbackModel(servicesInfo);
    if (fallbackModelValue) {
        const fallbackModel = storageModelToAvailableModel(
            fallbackModelValue as LlmServiceModel, 
            availableModels
        );
        if (fallbackModel) return fallbackModel;
    }

    // Last resort: first available model
    return availableModels.length > 0 ? availableModels[0] : null;
}

/**
 * Gets a validated preferred model by checking against currently available models.
 * This is the recommended function for hooks that need a guaranteed valid model.
 * 
 * @returns A validated model or fallback, never returns null
 */
export function getValidatedPreferredModel(): LlmServiceModel {
    try {
        // Get current AI config and available models
        const servicesInfo = getExistingAIConfig()?.service?.config;
        if (!servicesInfo) {
            // If no config available, return safe fallback
            return "gpt-4o-mini" as LlmServiceModel;
        }

        const availableModels = getAvailableModels(servicesInfo);
        
        // Get stored preference and validate it
        const storedPreference = useModelPreferencesStore.getState().preferredModel;
        if (storedPreference && availableModels.some(m => m.value === storedPreference)) {
            return storedPreference;
        }

        // Try service default
        const defaultModel = getFallbackModel(servicesInfo);
        if (defaultModel && availableModels.some(m => m.value === defaultModel)) {
            return defaultModel as LlmServiceModel;
        }

        // Return first available model or safe fallback
        return availableModels.length > 0 ? availableModels[0].value : "gpt-4o-mini" as LlmServiceModel;
    } catch (error) {
        // On any error, return safe fallback
        return "gpt-4o-mini" as LlmServiceModel;
    }
}

/**
 * Gets the user's preferred model, falling back to the default model if none is set.
 * 
 * @returns The user's preferred model or a fallback
 * @deprecated Use getValidatedPreferredModel for validation or getPreferredAvailableModel for type safety
 */
export function getPreferredModel(): LlmServiceModel {
    // For backward compatibility, now delegates to validated version
    return getValidatedPreferredModel();
}
