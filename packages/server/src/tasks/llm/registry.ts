import { LlmServiceId, MINUTES_15_MS, aiServicesInfo } from "@local/shared";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { LanguageModelService } from "./types.js";

/** States a service can be in */
export enum LlmServiceState {
    /** The service is active and can be used */
    Active = "Active",
    /** The service is in a cooldown period and cannot be used until the cooldown period has elapsed */
    Cooldown = "Cooldown",
    /** The service is disabled and cannot be used until the server is restarted or the service is manually re-enabled */
    Disabled = "Disabled",
}

/** 
 * General errors a service may send, which are 
 * mapped from the service's actual error types.
 */
export enum LlmServiceErrorType {
    /** An internal error occurred in the API */
    ApiError = "ApiError",
    /** The API key is invalid or has incorrect permissions */
    Authentication = "Authentication",
    /** Something is wrong with the request we sent */
    InvalidRequest = "InvalidRequest",
    /** The service is at capacity and cannot take requests at this time */
    Overloaded = "Overloaded",
    /** We've sent to many requests too quickly */
    RateLimit = "RateLimit",
}
const CRITICAL_ERRORS = [LlmServiceErrorType.Authentication];
const COOLDOWN_ERRORS = [LlmServiceErrorType.RateLimit, LlmServiceErrorType.ApiError, LlmServiceErrorType.Overloaded];

/** Maps cooldown errors to their cooldown time in milliseconds */
const COOLDOWN_TIMES: Partial<Record<LlmServiceErrorType, number>> = {
    [LlmServiceErrorType.RateLimit]: MINUTES_15_MS,
    [LlmServiceErrorType.ApiError]: MINUTES_15_MS,
    [LlmServiceErrorType.Overloaded]: MINUTES_15_MS,
};

const serviceInstances: Partial<Record<LlmServiceId, LanguageModelService<any>>> = {};

/**
 * Singleton class for managing the states of registered LLM (Large Language Model) services, 
 * and retrieving the best service to use for a given request.
 * 
 * It tracks each service's state, handling cooldown periods and disabling services as necessary based on errors received.
 */
export class LlmServiceRegistry {
    private static instance: LlmServiceRegistry | undefined;
    private serviceStates: Map<string, { state: LlmServiceState; cooldownUntil?: Date }> = new Map();

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }

    /**
     * Gets the singleton instance of the LLM service registry.
     * @returns The singleton instance of the LLM service registry.
     */
    public static get(): LlmServiceRegistry {
        if (!LlmServiceRegistry.instance) {
            LlmServiceRegistry.instance = new LlmServiceRegistry();
        }
        return LlmServiceRegistry.instance;
    }

    /**
     * Initializes all service instances. This should be called at server startup.
     */
    static async init() {
        const registry = LlmServiceRegistry.get();

        for (const [serviceIdKey, serviceInfo] of Object.entries(aiServicesInfo.services)) {
            const serviceId = serviceIdKey as LlmServiceId;

            if (!serviceInfo.enabled) {
                // Set service state to Disabled
                registry.registerService(serviceId, LlmServiceState.Disabled);
                logger.info(`Service ${serviceId} is disabled in configuration.`);
                continue;
            }

            // Initialize the service
            switch (serviceId) {
                case LlmServiceId.Anthropic:
                    {
                        const { AnthropicService } = await import("./services/anthropic.js");
                        serviceInstances[LlmServiceId.Anthropic] = new AnthropicService();
                    }
                    break;
                case LlmServiceId.Mistral:
                    {
                        const { MistralService } = await import("./services/mistral.js");
                        serviceInstances[LlmServiceId.Mistral] = new MistralService();
                    }
                    break;
                case LlmServiceId.OpenAI:
                    {
                        const { OpenAIService } = await import("./services/openai.js");
                        serviceInstances[LlmServiceId.OpenAI] = new OpenAIService();
                    }
                    break;
                default:
                    logger.warning(`Unknown service ID: ${serviceId}`, { trace: "0656" });
                    break;
            }

            // Set service state to Active
            registry.registerService(serviceId, LlmServiceState.Active);
        }
    }

    /**
     * Finds the targeted language model service, based on the model name
     * 
     * @param model The model name to find the service for, or undefined 
     * to use the default service.
     * @returns The service ID for the model, or the default service ID if the model is not found.
     */
    getServiceId(model: string | undefined): LlmServiceId {
        if (!model) return aiServicesInfo.defaultService;
        if (model.includes("gpt")) return LlmServiceId.OpenAI;
        if (model.includes("claude")) return LlmServiceId.Anthropic;
        if (model.includes("stral")) return LlmServiceId.Mistral;
        return aiServicesInfo.defaultService;
    }

    /**
     * Gets the current state of a registered LLM service.
     * 
     * @param serviceId The unique identifier for the LLM service.
     * @returns The current state of the service, or Disabled if the service is not registered.
     */
    getServiceState(serviceId: string): LlmServiceState {
        return this.serviceStates.get(serviceId)?.state || LlmServiceState.Disabled;
    }

    /**
     * Gets the specified service instance, if it is active.
     * 
     * @param serviceId The unique identifier for the LLM service.
     * @returns The service instance, or throws an error if the service is not active 
     * or not registered.
     */
    getService(serviceId: LlmServiceId): LanguageModelService<any> {
        // Check if the service is active
        const state = this.getServiceState(serviceId);
        if (state !== LlmServiceState.Active) {
            throw new CustomError("0652", "ServiceDisabled", { serviceId });
        }

        // Get the service instance
        const instance = serviceInstances[serviceId];
        if (!instance) {
            throw new CustomError("0251", "InternalError", { serviceId });
        }
        return instance;
    }

    /** 
     * Finds the best active service to use for a given model name, 
     * or null if no active services are available.
     * 
     * @param model The model name to find the best service for, or undefined 
     * to use the default service.
     * @returns The service ID for the model, or the default service ID if the model is not found.
     */
    getBestService(model: string | undefined): LlmServiceId | null {
        // Try requested service first
        const serviceId = this.getServiceId(model);
        if (this.getServiceState(serviceId) === LlmServiceState.Active) {
            return serviceId;
        }

        // The model name is the model string itself
        const modelName = serviceInstances[serviceId]?.getModel(model);

        // Get fallbacks
        const fallbacksForModel = aiServicesInfo.fallbacks[modelName] || [];
        // Try fallbacks
        for (const fallback of fallbacksForModel) {
            const fallbackServiceId = this.getServiceId(fallback);
            if (this.getServiceState(fallbackServiceId) === LlmServiceState.Active) {
                return fallbackServiceId;
            }
        }

        // If no active services are available, return null
        return null;
    }

    registerService(serviceId: string, state: LlmServiceState = LlmServiceState.Active) {
        this.serviceStates.set(serviceId, { state });
    }

    /**
     * Updates the state of a registered LLM service based on an error type.
     * This could place the service in a cooldown period or disable it.
     * @param {string} serviceId - The unique identifier for the LLM service.
     * @param {LlmServiceErrorType} errorType - The type of error received from the LLM service.
     */
    updateServiceState(serviceId: string, errorType: LlmServiceErrorType) {
        if (!this.serviceStates.has(serviceId)) {
            this.registerService(serviceId);
        }
        const service = this.serviceStates.get(serviceId)!;

        if (CRITICAL_ERRORS.includes(errorType)) {
            logger.error(`Critical error received from service ${serviceId}: ${errorType}. Disabling service.`, { trace: "0240" });
            service.state = LlmServiceState.Disabled;
            this.serviceStates.set(serviceId, service);
        } else if (COOLDOWN_ERRORS.includes(errorType)) {
            logger.warning(`Cooldown error received from service ${serviceId}: ${errorType}. Placing service in cooldown.`, { trace: "0241" });
            service.state = LlmServiceState.Cooldown;
            service.cooldownUntil = new Date(Date.now() + (COOLDOWN_TIMES[errorType] || MINUTES_15_MS));
            this.serviceStates.set(serviceId, service);
            this.resetServiceStateAfterCooldown(serviceId);
        }
    }

    /**
     * Resets the state of a service from Cooldown to Active after the cooldown period has elapsed.
     * This method is called internally after updating a service's state to Cooldown.
     * @param {string} serviceId - The unique identifier for the LLM service.
     */
    private resetServiceStateAfterCooldown(serviceId: string) {
        const service = this.serviceStates.get(serviceId);
        if (service && service.state === LlmServiceState.Cooldown && service.cooldownUntil) {
            const cooldownPeriod = service.cooldownUntil.getTime() - Date.now();
            setTimeout(() => {
                service.state = LlmServiceState.Active;
                this.serviceStates.set(serviceId, service);
            }, cooldownPeriod);
        }
    }
}
