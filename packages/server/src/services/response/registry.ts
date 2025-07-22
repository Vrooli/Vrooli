import { LlmServiceId, MINUTES_15_MS, aiServicesInfo } from "@vrooli/shared";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { isLocalExecutionEnabled } from "../../utils/executionMode.js";
import { type AIService } from "./services.js";
import { NetworkMonitor } from "./NetworkMonitor.js";

export { LlmServiceId };

/** States a service can be in */
export enum AIServiceState {
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
export enum AIServiceErrorType {
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
const CRITICAL_ERRORS = [AIServiceErrorType.Authentication];
const COOLDOWN_ERRORS = [AIServiceErrorType.RateLimit, AIServiceErrorType.ApiError, AIServiceErrorType.Overloaded];

/** Maps cooldown errors to their cooldown time in milliseconds */
const COOLDOWN_TIMES: Partial<Record<AIServiceErrorType, number>> = {
    [AIServiceErrorType.RateLimit]: MINUTES_15_MS,
    [AIServiceErrorType.ApiError]: MINUTES_15_MS,
    [AIServiceErrorType.Overloaded]: MINUTES_15_MS,
};

const serviceInstances: Partial<Record<LlmServiceId, AIService<any, any>>> = {};

/**
 * Singleton class for managing the states of registered LLM (Large Language Model) services, 
 * and retrieving the best service to use for a given request.
 * 
 * It tracks each service's state, handling cooldown periods and disabling services as necessary based on errors received.
 */
export class AIServiceRegistry {
    private static instance: AIServiceRegistry | undefined;
    private serviceStates: Map<string, { state: AIServiceState; cooldownUntil?: Date }> = new Map();
    private networkMonitor: NetworkMonitor;

    private constructor() {
        this.networkMonitor = NetworkMonitor.getInstance();
    }

    /**
     * Gets the singleton instance of the LLM service registry.
     * @returns The singleton instance of the LLM service registry.
     */
    public static get(): AIServiceRegistry {
        if (!AIServiceRegistry.instance) {
            AIServiceRegistry.instance = new AIServiceRegistry();
        }
        return AIServiceRegistry.instance;
    }

    /**
     * Initializes all service instances. This should be called at server startup.
     */
    static async init() {
        console.log("=== AIServiceRegistry.init() starting ===");
        const registry = AIServiceRegistry.get();
        
        // Start network monitoring
        console.log("=== AIServiceRegistry: Starting network monitoring ===");
        registry.networkMonitor.start();
        console.log("=== AIServiceRegistry: Network monitoring started ===");

        console.log("=== AIServiceRegistry: Beginning service initialization loop ===");
        for (const [serviceIdKey, serviceInfo] of Object.entries(aiServicesInfo.services)) {
            const serviceId = serviceIdKey as LlmServiceId;
            console.log(`=== AIServiceRegistry: Processing service ${serviceId} ===`);

            // Special handling for ClaudeCode - check execution mode
            if (serviceId === LlmServiceId.ClaudeCode) {
                console.log("=== AIServiceRegistry: Checking ClaudeCode service ===");
                console.log(`=== NODE_ENV: ${process.env.NODE_ENV} ===`);
                console.log(`=== PROJECT_DIR: ${process.env.PROJECT_DIR} ===`);
                
                const isLocalMode = isLocalExecutionEnabled();
                console.log(`=== isLocalExecutionEnabled returned: ${isLocalMode} ===`);
                
                if (!isLocalMode) {
                    registry.registerService(serviceId, AIServiceState.Disabled);
                    logger.info(`Service ${serviceId} is disabled because local execution mode is not enabled.`);
                    console.log(`=== AIServiceRegistry: Service ${serviceId} disabled (local execution mode required) ===`);
                    continue;
                }
                // If local mode is enabled, proceed with normal initialization
                logger.info(`Service ${serviceId} is enabled in local execution mode.`);
                console.log(`=== AIServiceRegistry: Service ${serviceId} enabled in local execution mode ===`);
            } else if (!serviceInfo.enabled) {
                // For other services, check the configuration as usual
                registry.registerService(serviceId, AIServiceState.Disabled);
                logger.info(`Service ${serviceId} is disabled in configuration.`);
                console.log(`=== AIServiceRegistry: Service ${serviceId} disabled ===`);
                continue;
            }

            console.log(`=== AIServiceRegistry: Initializing service ${serviceId} ===`);
            // Initialize the service
            switch (serviceId) {
                case LlmServiceId.LocalOllama:
                    {
                        const { LocalOllamaService } = await import("./providers/LocalOllamaService.js");
                        serviceInstances[LlmServiceId.LocalOllama] = new LocalOllamaService();
                    }
                    break;
                case LlmServiceId.CloudflareGateway:
                    {
                        const { CloudflareGatewayService } = await import("./providers/CloudflareGatewayService.js");
                        serviceInstances[LlmServiceId.CloudflareGateway] = new CloudflareGatewayService();
                    }
                    break;
                case LlmServiceId.OpenRouter:
                    {
                        const { OpenRouterService } = await import("./providers/OpenRouterService.js");
                        serviceInstances[LlmServiceId.OpenRouter] = new OpenRouterService();
                    }
                    break;
                case LlmServiceId.ClaudeCode:
                    {
                        const { ClaudeCodeService } = await import("./providers/ClaudeCodeService.js");
                        serviceInstances[LlmServiceId.ClaudeCode] = new ClaudeCodeService();
                    }
                    break;
                default:
                    logger.warning(`Unknown service ID: ${serviceId}`, { trace: "0656" });
                    break;
            }

            // Set service state to Active
            registry.registerService(serviceId, AIServiceState.Active);
            console.log(`=== AIServiceRegistry: Service ${serviceId} initialized and registered ===`);
        }
        console.log("=== AIServiceRegistry: All services processed ===");
        console.log("=== AIServiceRegistry.init() completed ===");
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
        
        // Check for Cloudflare AI Gateway models (identified by @cf/ prefix)
        if (model.startsWith("@cf/")) return LlmServiceId.CloudflareGateway;
        
        // Check for OpenRouter models (identified by provider/model format)
        if (model.includes("/") && (
            model.startsWith("openai/") || 
            model.startsWith("anthropic/") || 
            model.startsWith("meta-llama/") || 
            model.startsWith("google/") || 
            model.startsWith("mistralai/")
        )) {
            return LlmServiceId.OpenRouter;
        }
        
        // Check for local Ollama models (typically versioned with colons like "llama3.1:8b")
        if (model.includes(":") || 
            (model.includes("llama") && !model.includes("/")) || 
            (model.includes("codellama") && !model.includes("/")) || 
            (model.includes("mistral") && !model.includes("/"))
        ) {
            return LlmServiceId.LocalOllama;
        }
        
        return aiServicesInfo.defaultService;
    }

    /**
     * Gets the current state of a registered LLM service.
     * 
     * @param serviceId The unique identifier for the LLM service.
     * @returns The current state of the service, or Disabled if the service is not registered.
     */
    getServiceState(serviceId: string): AIServiceState {
        return this.serviceStates.get(serviceId)?.state || AIServiceState.Disabled;
    }

    /**
     * Gets the specified service instance, if it is active.
     * 
     * @param serviceId The unique identifier for the LLM service.
     * @returns The service instance, or throws an error if the service is not active 
     * or not registered.
     */
    getService(serviceId: LlmServiceId): AIService<any> {
        // Check if the service is active
        const state = this.getServiceState(serviceId.toString());
        if (state !== AIServiceState.Active) {
            const registry = AIServiceRegistry.get();
            const availableServices = Object.entries(serviceInstances)
                .filter(([serviceIdKey, _]) => registry.getServiceState(serviceIdKey) === AIServiceState.Active)
                .map(([serviceIdKey]) => serviceIdKey);
            throw new CustomError("0652", "ServiceDisabled", { serviceId, availableServices });
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
     * with enhanced fallback logic that checks model support and network connectivity.
     * 
     * @param model The model name to find the best service for, or undefined 
     * to use the default service.
     * @param offlineOnly Force local models only (default: false)
     * @returns The service ID for the model, or the default service ID if the model is not found.
     */
    async getBestService(model: string | undefined, offlineOnly = false): Promise<LlmServiceId | null> {
        // Get network state
        const networkState = await this.networkMonitor.getState();
        
        // If offline or offline-only mode, prioritize local services
        if (offlineOnly || !networkState.cloudServicesReachable) {
            // Try local services first
            const localServices = [LlmServiceId.LocalOllama];
            for (const serviceId of localServices) {
                if (this.getServiceState(serviceId.toString()) === AIServiceState.Active) {
                    const serviceInstance = serviceInstances[serviceId];
                    if (serviceInstance && await this.doesServiceSupportModel(serviceInstance, model)) {
                        return serviceId;
                    }
                }
            }
            
            // If offline-only mode and no local services available, return null
            if (offlineOnly) {
                return null;
            }
        }

        // Try requested service first (if network allows)
        const primaryServiceId = this.getServiceId(model);
        if (this.isServiceAvailable(primaryServiceId, networkState)) {
            const serviceInstance = serviceInstances[primaryServiceId];
            if (serviceInstance && await this.doesServiceSupportModel(serviceInstance, model)) {
                return primaryServiceId;
            }
        }

        // Try fallback chain for this model
        const modelName = model || "";
        const fallbacksForModel = aiServicesInfo.fallbacks[modelName] || [];
        
        for (const fallbackModel of fallbacksForModel) {
            const fallbackServiceId = this.getServiceId(fallbackModel);
            if (this.isServiceAvailable(fallbackServiceId, networkState)) {
                const serviceInstance = serviceInstances[fallbackServiceId];
                if (serviceInstance && await this.doesServiceSupportModel(serviceInstance, fallbackModel)) {
                    return fallbackServiceId;
                }
            }
        }

        // Try the universal fallback: LocalOllama -> CloudflareGateway -> OpenRouter
        const universalFallbacks = [LlmServiceId.LocalOllama, LlmServiceId.CloudflareGateway, LlmServiceId.OpenRouter];
        
        for (const serviceId of universalFallbacks) {
            if (this.isServiceAvailable(serviceId, networkState)) {
                const serviceInstance = serviceInstances[serviceId];
                if (serviceInstance && await this.doesServiceSupportModel(serviceInstance, model)) {
                    return serviceId;
                }
            }
        }

        // If no active services are available, return null
        return null;
    }

    /**
     * Check if a service is available considering both state and network connectivity
     */
    private isServiceAvailable(serviceId: LlmServiceId, networkState: any): boolean {
        // Check if service is active
        if (this.getServiceState(serviceId.toString()) !== AIServiceState.Active) {
            return false;
        }

        // Check network connectivity for cloud services
        if (serviceId === LlmServiceId.LocalOllama) {
            return networkState.localServicesReachable;
        } else {
            return networkState.cloudServicesReachable;
        }
    }

    /**
     * Check if a service supports a specific model
     */
    private async doesServiceSupportModel(serviceInstance: any, model: string | undefined): Promise<boolean> {
        if (!model) return true; // If no model specified, assume supported
        
        try {
            // Check if the service has a supportsModel method
            if (typeof serviceInstance.supportsModel === "function") {
                const result = await serviceInstance.supportsModel(model);
                return Boolean(result);
            }
            
            // Fallback: assume the service supports the model
            return true;
        } catch (error) {
            logger.warn(`Error checking model support for ${serviceInstance.__id}`, { model, error });
            return false;
        }
    }

    registerService(serviceId: string, state: AIServiceState = AIServiceState.Active) {
        this.serviceStates.set(serviceId, { state });
    }

    /**
     * Updates the state of a registered LLM service based on an error type.
     * This could place the service in a cooldown period or disable it.
     * @param {string} serviceId - The unique identifier for the LLM service.
     * @param {AIServiceErrorType} errorType - The type of error received from the LLM service.
     */
    updateServiceState(serviceId: string, errorType: AIServiceErrorType) {
        if (!this.serviceStates.has(serviceId)) {
            this.registerService(serviceId);
        }
        const service = this.serviceStates.get(serviceId)!;

        if (CRITICAL_ERRORS.includes(errorType)) {
            logger.error(`Critical error received from service ${serviceId}: ${errorType}. Disabling service.`, { trace: "0240" });
            service.state = AIServiceState.Disabled;
            this.serviceStates.set(serviceId, service);
        } else if (COOLDOWN_ERRORS.includes(errorType)) {
            logger.warning(`Cooldown error received from service ${serviceId}: ${errorType}. Placing service in cooldown.`, { trace: "0241" });
            service.state = AIServiceState.Cooldown;
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
        if (service && service.state === AIServiceState.Cooldown && service.cooldownUntil) {
            const cooldownPeriod = service.cooldownUntil.getTime() - Date.now();
            setTimeout(() => {
                service.state = AIServiceState.Active;
                this.serviceStates.set(serviceId, service);
            }, cooldownPeriod);
        }
    }
}
