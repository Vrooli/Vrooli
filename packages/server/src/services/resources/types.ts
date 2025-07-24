import type { ResourcesConfig } from "./resourcesConfig.js";

/**
 * Resource categories that can be registered
 */
export enum ResourceCategory {
    AI = "ai",
    Automation = "automation",
    Agents = "agents",
    Storage = "storage",
}

/**
 * Deployment type for resources
 */
export enum DeploymentType {
    /** Resource runs on local infrastructure */
    Local = "local",
    /** Resource runs on cloud infrastructure */
    Cloud = "cloud",
    /** Resource can run on both local and cloud */
    Hybrid = "hybrid",
}

/**
 * Health status for a resource
 */
export enum ResourceHealth {
    Healthy = "healthy",
    Unhealthy = "unhealthy",
    Unknown = "unknown",
    Checking = "checking",
}

/**
 * Discovery status for a resource
 */
export enum DiscoveryStatus {
    /** Resource is implemented and found running */
    Available = "available",
    /** Resource is implemented but not found */
    NotFound = "not_found",
    /** Resource is not implemented in Vrooli */
    NotSupported = "not_supported",
    /** Currently checking if resource is available */
    Discovering = "discovering",
}

/**
 * Base configuration that all resources share
 */
export interface BaseResourceConfig {
    enabled: boolean;
    healthCheck?: {
        endpoint?: string;
        intervalMs: number;
        timeoutMs: number;
        requiresAuth?: boolean;
    };
}

/**
 * Public information about a resource's current state (safe to expose)
 */
export interface PublicResourceInfo {
    id: string;
    category: ResourceCategory;
    displayName: string;
    description: string;
    status: DiscoveryStatus;
    health: ResourceHealth;
    lastHealthCheck?: Date;
    metadata?: Record<string, any>;
}

/**
 * Internal information about a resource's current state (includes sensitive config)
 * @internal WARNING: Contains sensitive configuration data including API keys
 */
export interface InternalResourceInfo extends PublicResourceInfo {
    config?: BaseResourceConfig;
}

/**
 * Result of a health check
 */
export interface HealthCheckResult {
    healthy: boolean;
    message?: string;
    details?: Record<string, any>;
    timestamp: Date;
}

/**
 * Options for initializing a resource
 */
export interface ResourceInitOptions {
    config: any;
    /** Global resources config for cross-resource access */
    globalConfig: ResourcesConfig;
    /** Signal for cancellation */
    signal?: AbortSignal;
}

/**
 * Common interface that all resource implementations must follow
 * Note: Implementations should extend ResourceProvider which provides EventEmitter functionality
 */
export interface IResource {
    /** Unique identifier for this resource type */
    readonly id: string;
    /** Category this resource belongs to */
    readonly category: ResourceCategory;
    /** Human-readable name */
    readonly displayName: string;
    /** Description of what this resource does */
    readonly description: string;
    /** Whether this resource type is currently supported/implemented */
    readonly isSupported: boolean;
    /** Deployment type of this resource */
    readonly deploymentType: DeploymentType;
    
    /** Initialize the resource with configuration */
    initialize(options: ResourceInitOptions): Promise<void>;
    
    /** Check if the resource is healthy and available */
    healthCheck(): Promise<HealthCheckResult>;
    
    /** Discover if this resource is running locally */
    discover(): Promise<boolean>;
    
    /** Get public resource information (safe to expose) */
    getPublicInfo(): PublicResourceInfo;
    
    /** Get internal resource information (includes sensitive config) */
    getInternalInfo(): InternalResourceInfo;
    
    /** Cleanup and shutdown the resource */
    shutdown(): Promise<void>;
}

/**
 * Extended interface for AI resources
 */
export interface IAIResource extends IResource {
    /** List available models */
    listModels(): Promise<string[]>;
    /** Check if a specific model is available */
    hasModel(modelId: string): Promise<boolean>;
}

/**
 * Extended interface for automation resources
 */
export interface IAutomationResource extends IResource {
    /** Get capabilities of this automation service */
    getCapabilities(): string[];
    /** Check if service can execute a specific routine */
    canExecuteRoutine(routineId: string): Promise<boolean>;
}

/**
 * Extended interface for agent resources
 */
export interface IAgentResource extends IResource {
    /** Get current number of active instances */
    getActiveInstances(): number;
    /** Check if can spawn new instance */
    canSpawnInstance(): boolean;
}

/**
 * Extended interface for storage resources
 */
export interface IStorageResource extends IResource {
    /** List available buckets/containers */
    listBuckets(): Promise<string[]>;
    /** Check if a bucket exists */
    bucketExists(name: string): Promise<boolean>;
}

/**
 * Event types emitted by the resource system
 */
export enum ResourceEvent {
    /** A new resource was discovered */
    Discovered = "resource:discovered",
    /** A resource became unavailable */
    Lost = "resource:lost",
    /** A resource's health status changed */
    HealthChanged = "resource:health_changed",
    /** Resource configuration was updated */
    ConfigUpdated = "resource:config_updated",
}

/**
 * Event data for resource events
 */
export interface ResourceEventData {
    resourceId: string;
    category: ResourceCategory;
    event: ResourceEvent;
    timestamp: Date;
    details?: Record<string, any>;
}

