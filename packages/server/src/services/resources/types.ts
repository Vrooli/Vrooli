import type { ServiceConfig } from "./resourcesConfig.js";
import type {
    AgentResourceId,
    AIResourceId,
    AutomationResourceId,
    ExecutionResourceId,
    GetConfig,
    GetResourceMetadata,
    ResourceId,
    StorageResourceId,
} from "./typeRegistry.js";

/**
 * Resource categories that can be registered
 */
export enum ResourceCategory {
    AI = "ai",
    Automation = "automation",
    Agents = "agents",
    Storage = "storage",
    Execution = "execution",
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
    // Authentication options (used by ResourceProvider.getAuthConfig)
    apiKey?: string;
    apiKeyHeader?: string;
    bearerToken?: string;
    username?: string;
    password?: string;
    // Health check configuration
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
 * Fully typed public resource information with category-specific metadata
 */
export interface TypedPublicResourceInfo<TId extends ResourceId = ResourceId>
    extends Omit<PublicResourceInfo, "metadata"> {
    metadata?: GetResourceMetadata<TId>;
}

/**
 * Fully typed internal resource information with category-specific metadata and config
 * @internal WARNING: Contains sensitive configuration data including API keys
 */
export interface TypedInternalResourceInfo<TId extends ResourceId = ResourceId>
    extends Omit<InternalResourceInfo, "metadata" | "config"> {
    config?: GetConfig<TId>;
    metadata?: GetResourceMetadata<TId>;
}

/**
 * Union type for all possible typed resource info (for runtime collections)
 */
export type AnyTypedResourceInfo =
    | TypedPublicResourceInfo<AIResourceId>
    | TypedPublicResourceInfo<AutomationResourceId>
    | TypedPublicResourceInfo<AgentResourceId>
    | TypedPublicResourceInfo<StorageResourceId>
    | TypedPublicResourceInfo<ExecutionResourceId>;

// ============================================================================
// Health Check Detail Types
// ============================================================================

/**
 * Health details specific to AI resources
 */
export interface AIHealthDetails {
    modelCount?: number;
    availableModels?: string[];
    version?: string;
    memoryUsage?: number;
    activeRequests?: number;
    totalRequests?: number;
    averageResponseTime?: number;
}

/**
 * Health details specific to automation resources  
 */
export interface AutomationHealthDetails {
    workflowCount?: number;
    activeExecutions?: number;
    queueSize?: number;
    version?: string;
    uptime?: number;
    failedExecutions?: number;
    totalExecutions?: number;
}

/**
 * Health details specific to agent resources
 */
export interface AgentHealthDetails {
    activeInstances: number;
    maxInstances: number;
    browserVersion?: string;
    availableBrowsers?: string[];
    totalSessions?: number;
    averageSessionDuration?: number;
}

/**
 * Health details specific to storage resources
 */
export interface StorageHealthDetails {
    totalSpace?: number;
    usedSpace?: number;
    bucketCount?: number;
    objectCount?: number;
    connectivity?: "healthy" | "degraded" | "offline";
    averageResponseTime?: number;
}

/**
 * Health details specific to execution resources
 */
export interface ExecutionHealthDetails {
    supportedLanguages?: number;
    activeWorkers?: number;
    configuredWorkers?: number;
    queuedSubmissions?: number;
    securityLimits?: {
        cpuTime: number;
        memory: number;
        processes: number;
    };
    version?: string;
}

/**
 * Result of a health check with untyped details (legacy)
 */
export interface HealthCheckResult {
    healthy: boolean;
    message?: string;
    details?: Record<string, any>;
    timestamp: Date;
}

/**
 * Fully typed health check result with category-specific details
 */
export interface TypedHealthCheckResult<TCategory extends ResourceCategory = ResourceCategory>
    extends Omit<HealthCheckResult, "details"> {
    details?: TCategory extends ResourceCategory.AI ? AIHealthDetails :
    TCategory extends ResourceCategory.Automation ? AutomationHealthDetails :
    TCategory extends ResourceCategory.Agents ? AgentHealthDetails :
    TCategory extends ResourceCategory.Storage ? StorageHealthDetails :
    TCategory extends ResourceCategory.Execution ? ExecutionHealthDetails :
    Record<string, unknown>;
}

/**
 * Options for initializing a resource with full type safety
 */
export interface ResourceInitOptions<TId extends ResourceId = ResourceId> {
    config: GetConfig<TId>;
    /** Global service config for cross-resource access */
    globalConfig: ServiceConfig;
    /** Signal for cancellation */
    signal?: AbortSignal;
}

/**
 * Runtime-safe version for when resource ID is not known at compile time
 */
export interface UntypedResourceInitOptions {
    config: unknown;
    /** Global service config for cross-resource access */
    globalConfig: ServiceConfig;
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
    /** List available workflows/routines */
    listWorkflows(): Promise<string[]>;
    /** Get count of active workflows */
    getActiveWorkflowCount(): Promise<number>;
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

