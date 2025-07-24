import { EventEmitter } from "events";
import { logger } from "../../events/logger.js";
import type { IResource, PublicResourceInfo, ResourceEventData } from "./types.js";
import { ResourceEvent , ResourceCategory, DiscoveryStatus} from "./types.js";
import type { ResourcesConfig } from "./resourcesConfig.js";
import { loadResourcesConfig } from "./resourcesConfig.js";
import type { ResourceSystemHealthCheck } from "./healthCheck.js";
import { buildHealthCheck } from "./healthCheck.js";

// Constants
const DEFAULT_DISCOVERY_INTERVAL_MS = 3600000; // 1 hour

/**
 * Registry for all resources (local and cloud)
 * Manages registration, discovery, and lifecycle of resources
 */
export class ResourceRegistry extends EventEmitter {
    private static instance: ResourceRegistry;
    
    /** Map of resource ID to resource instance */
    private resources = new Map<string, IResource>();
    
    /** Map of resource ID to resource class constructor */
    private resourceClasses = new Map<string, new () => IResource>();
    
    /** Current configuration */
    private config?: ResourcesConfig;
    
    /** Whether the registry has been initialized */
    private initialized = false;
    
    /** Discovery interval timer */
    private discoveryInterval?: NodeJS.Timeout;
    
    /** Track event cleanup functions for each resource */
    private eventCleanupMap = new Map<string, Array<() => void>>();
    
    private constructor() {
        super();
    }
    
    /**
     * Get singleton instance
     */
    static getInstance(): ResourceRegistry {
        if (!ResourceRegistry.instance) {
            ResourceRegistry.instance = new ResourceRegistry();
        }
        return ResourceRegistry.instance;
    }
    
    /**
     * Register a resource class
     * This is called by resource implementations at module load time
     */
    registerResourceClass(ResourceClass: new () => IResource): void {
        const instance = new ResourceClass();
        const id = instance.id;
        
        if (this.resourceClasses.has(id)) {
            logger.warn(`[ResourceRegistry] Resource ${id} already registered, overwriting`);
        }
        
        this.resourceClasses.set(id, ResourceClass);
        logger.info(`[ResourceRegistry] Registered resource class: ${id}`);
        
        // If already initialized, instantiate this resource
        if (this.initialized && this.config?.enabled) {
            this.instantiateResource(id).catch(error => {
                logger.error(`[ResourceRegistry] Failed to instantiate ${id}`, error);
            });
        }
    }
    
    /**
     * Initialize the registry with configuration
     */
    async initialize(configPath?: string): Promise<void> {
        if (this.initialized) {
            logger.warn("[ResourceRegistry] Already initialized");
            return;
        }
        
        try {
            // Load configuration
            this.config = await loadResourcesConfig(configPath);
            
            if (!this.config.enabled) {
                logger.info("[ResourceRegistry] Resources disabled in configuration");
                this.initialized = true;
                return;
            }
            
            // Instantiate all registered resources
            await this.instantiateAllResources();
            
            // Start discovery if configured
            if (this.config.discovery?.enabled) {
                this.startDiscovery();
            }
            
            this.initialized = true;
            logger.info("[ResourceRegistry] Initialized successfully");
        } catch (error) {
            logger.error("[ResourceRegistry] Failed to initialize", error);
            throw error;
        }
    }
    
    /**
     * Instantiate all registered resource classes
     */
    private async instantiateAllResources(): Promise<void> {
        const promises: Promise<void>[] = [];
        
        for (const [id] of Array.from(this.resourceClasses)) {
            promises.push(this.instantiateResource(id));
        }
        
        await Promise.allSettled(promises);
    }
    
    /**
     * Instantiate a specific resource
     */
    private async instantiateResource(id: string): Promise<void> {
        const ResourceClass = this.resourceClasses.get(id);
        if (!ResourceClass) {
            logger.warn(`[ResourceRegistry] No class registered for resource ${id}`);
            return;
        }
        
        try {
            const resource = new ResourceClass();
            this.resources.set(id, resource);
            
            // Subscribe to resource events
            // All resources should extend ResourceProvider which is an EventEmitter
            if (resource instanceof EventEmitter) {
                const cleanupFunctions: Array<() => void> = [];
                
                // Create event handlers
                const discoveredHandler = (data: ResourceEventData) => {
                    this.emit(ResourceEvent.Discovered, data);
                };
                const lostHandler = (data: ResourceEventData) => {
                    this.emit(ResourceEvent.Lost, data);
                };
                const healthChangedHandler = (data: ResourceEventData) => {
                    this.emit(ResourceEvent.HealthChanged, data);
                };
                
                // Add listeners
                resource.on(ResourceEvent.Discovered, discoveredHandler);
                resource.on(ResourceEvent.Lost, lostHandler);
                resource.on(ResourceEvent.HealthChanged, healthChangedHandler);
                
                // Store cleanup functions
                cleanupFunctions.push(
                    () => resource.removeListener(ResourceEvent.Discovered, discoveredHandler),
                    () => resource.removeListener(ResourceEvent.Lost, lostHandler),
                    () => resource.removeListener(ResourceEvent.HealthChanged, healthChangedHandler),
                );
                
                this.eventCleanupMap.set(id, cleanupFunctions);
            } else {
                logger.warn(`[ResourceRegistry] Resource ${id} does not extend EventEmitter`);
            }
            
            // Get configuration for this resource
            const category = resource.category;
            const serviceConfig = this.config?.services?.[category]?.[id];
            
            if (serviceConfig?.enabled) {
                await resource.initialize({
                    config: serviceConfig,
                    globalConfig: this.config as ResourcesConfig,
                });
            } else {
                logger.debug(`[ResourceRegistry] Resource ${id} not enabled in configuration`);
            }
        } catch (error) {
            logger.error(`[ResourceRegistry] Failed to instantiate resource ${id}`, error);
        }
    }
    
    /**
     * Start periodic discovery
     */
    private startDiscovery(): void {
        if (!this.config?.discovery?.methods?.portScanning?.enabled) {
            return;
        }
        
        const intervalMs = this.config.discovery.methods.portScanning.scanIntervalMs || DEFAULT_DISCOVERY_INTERVAL_MS;
        
        // Perform initial discovery
        this.performDiscovery().catch(error => {
            logger.error("[ResourceRegistry] Initial discovery failed", error);
        });
        
        // Start periodic discovery
        this.discoveryInterval = setInterval(() => {
            this.performDiscovery().catch(error => {
                logger.error("[ResourceRegistry] Periodic discovery failed", error);
            });
        }, intervalMs);
        
        logger.info(`[ResourceRegistry] Started discovery with interval ${intervalMs}ms`);
    }
    
    /**
     * Perform discovery for all resources
     */
    private async performDiscovery(): Promise<void> {
        logger.debug("[ResourceRegistry] Starting discovery scan");
        
        const promises: Promise<void>[] = [];
        
        for (const [id, resource] of Array.from(this.resources)) {
            promises.push(
                resource.discover()
                    .then(found => {
                        if (found) {
                            logger.info(`[ResourceRegistry] Discovered resource: ${id}`);
                        }
                    })
                    .catch(error => {
                        logger.error(`[ResourceRegistry] Discovery failed for ${id}`, error);
                    }),
            );
        }
        
        await Promise.allSettled(promises);
        logger.debug("[ResourceRegistry] Discovery scan complete");
    }
    
    /**
     * Get a specific resource by ID
     */
    getResource<T extends IResource = IResource>(id: string): T | undefined {
        return this.resources.get(id) as T | undefined;
    }
    
    /**
     * Get all resources
     */
    getAllResources(): IResource[] {
        return Array.from(this.resources.values());
    }
    
    /**
     * Get resources by category
     */
    getResourcesByCategory(category: ResourceCategory): IResource[] {
        return this.getAllResources().filter(r => r.category === category);
    }
    
    /**
     * Get resources by status
     */
    getResourcesByStatus(status: DiscoveryStatus): IResource[] {
        return this.getAllResources().filter(r => r.getPublicInfo().status === status);
    }
    
    /**
     * Get summary of all resources
     */
    getResourceSummary(): {
        total: number;
        supported: number;
        available: number;
        byCategory: Record<ResourceCategory, PublicResourceInfo[]>;
    } {
        const resources = this.getAllResources();
        const infos = resources.map(r => r.getPublicInfo());
        
        const byCategory: Record<string, PublicResourceInfo[]> = {};
        for (const category of Object.values(ResourceCategory)) {
            byCategory[category] = infos.filter(i => i.category === category);
        }
        
        return {
            total: this.resourceClasses.size,
            supported: resources.filter(r => r.isSupported).length,
            available: infos.filter(i => i.status === DiscoveryStatus.Available).length,
            byCategory: byCategory as Record<ResourceCategory, PublicResourceInfo[]>,
        };
    }
    
    /**
     * Check if a specific resource is available
     */
    isResourceAvailable(id: string): boolean {
        const resource = this.resources.get(id);
        return resource?.getPublicInfo().status === DiscoveryStatus.Available || false;
    }
    
    /**
     * Get a comprehensive health check of the resources system
     * This is designed to be used by health check endpoints
     */
    getHealthCheck(): ResourceSystemHealthCheck {
        // Collect all resource information (use public info for health checks)
        const resourceInfoMap = new Map<string, PublicResourceInfo>();
        for (const [id, resource] of Array.from(this.resources)) {
            resourceInfoMap.set(id, resource.getPublicInfo());
        }
        
        // Determine which resources are enabled in config
        const enabledResourceIds = new Set<string>();
        if (this.config?.services) {
            for (const category of Object.values(ResourceCategory)) {
                const categoryServices = this.config.services[category];
                if (categoryServices) {
                    for (const [id, serviceConfig] of Object.entries(categoryServices)) {
                        if ((serviceConfig as any)?.enabled) {
                            enabledResourceIds.add(id);
                        }
                    }
                }
            }
        }
        
        // Build and return the health check
        return buildHealthCheck(resourceInfoMap, enabledResourceIds);
    }
    
    /**
     * Unregister a specific resource
     * This properly cleans up event listeners and shuts down the resource
     */
    async unregisterResource(id: string): Promise<void> {
        const resource = this.resources.get(id);
        if (!resource) {
            logger.warn(`[ResourceRegistry] Attempt to unregister non-existent resource: ${id}`);
            return;
        }
        
        logger.info(`[ResourceRegistry] Unregistering resource: ${id}`);
        
        // Clean up event listeners
        this.cleanupResourceEvents(id);
        
        // Shutdown the resource
        try {
            await resource.shutdown();
        } catch (error) {
            logger.error(`[ResourceRegistry] Error shutting down resource ${id}:`, error);
        }
        
        // Remove from maps
        this.resources.delete(id);
        this.resourceClasses.delete(id);
    }
    
    /**
     * Clean up a specific resource's event listeners
     */
    private cleanupResourceEvents(resourceId: string): void {
        const cleanupFunctions = this.eventCleanupMap.get(resourceId);
        if (cleanupFunctions) {
            for (const cleanup of cleanupFunctions) {
                try {
                    cleanup();
                } catch (error) {
                    logger.error(`[ResourceRegistry] Error cleaning up events for ${resourceId}:`, error);
                }
            }
            this.eventCleanupMap.delete(resourceId);
        }
    }
    
    /**
     * Shutdown the registry and all resources
     */
    async shutdown(): Promise<void> {
        logger.info("[ResourceRegistry] Shutting down");
        
        // Stop discovery
        if (this.discoveryInterval) {
            clearInterval(this.discoveryInterval);
            this.discoveryInterval = undefined;
        }
        
        // Clean up all event listeners before shutting down resources
        for (const resourceId of Array.from(this.eventCleanupMap.keys())) {
            this.cleanupResourceEvents(resourceId);
        }
        
        // Shutdown all resources
        const promises: Promise<void>[] = [];
        for (const resource of Array.from(this.resources.values())) {
            promises.push(resource.shutdown());
        }
        
        await Promise.allSettled(promises);
        
        // Clear resources
        this.resources.clear();
        this.eventCleanupMap.clear();
        this.initialized = false;
        
        // Remove all event listeners
        this.removeAllListeners();
        
        logger.info("[ResourceRegistry] Shutdown complete");
    }
}

/**
 * Decorator to auto-register resource classes
 * Usage: @RegisterResource
 */
export function RegisterResource(target: new () => IResource): void {
    ResourceRegistry.getInstance().registerResourceClass(target);
}

