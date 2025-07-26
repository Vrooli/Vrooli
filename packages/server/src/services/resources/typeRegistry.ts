/**
 * Central type registry for the resources system.
 * This file defines all type mappings and relationships for complete type safety.
 */

import type { 
    DeploymentType,
    BaseResourceConfig,
    IResource,
    IAIResource,
    IAutomationResource,
    IAgentResource,
    IStorageResource,
} from "./types.js";
import { ResourceCategory } from "./types.js";

// ============================================================================
// Resource ID Definitions
// ============================================================================

/**
 * All AI resource IDs
 */
export type AIResourceId = 
    | "ollama" 
    | "localai" 
    | "llamacpp" 
    | "vllm" 
    | "tgi"
    | "onnx"
    | "whisper"
    | "comfyui"
    | "stablediffusion"
    | "cloudflare"
    | "openrouter";

/**
 * All automation resource IDs
 */
export type AutomationResourceId =
    | "n8n"
    | "nodered"
    | "windmill"
    | "automatisch"
    | "activepieces"
    | "huginn"
    | "kestra"
    | "beehive"
    | "airflow"
    | "temporal";

/**
 * All agent resource IDs
 */
export type AgentResourceId =
    | "browserless"
    | "claude-code";

/**
 * All storage resource IDs
 */
export type StorageResourceId =
    | "minio"
    | "seaweedfs"
    | "glusterfs"
    | "ipfs"
    | "rclone";

/**
 * Union of all resource IDs
 */
export type ResourceId = 
    | AIResourceId 
    | AutomationResourceId 
    | AgentResourceId 
    | StorageResourceId;

// ============================================================================
// Configuration Types
// ============================================================================

/**
 * Base configuration with common fields
 */
export interface TypedBaseResourceConfig extends BaseResourceConfig {
    enabled: boolean;
    healthCheck?: {
        endpoint?: string;
        intervalMs: number;
        timeoutMs: number;
        requiresAuth?: boolean;
    };
}

/**
 * AI-specific configuration
 */
export interface AIResourceConfig extends TypedBaseResourceConfig {
    baseUrl: string;
    apiKey?: string;
    models?: string[];
    maxConcurrent?: number;
    timeout?: number;
}

/**
 * Automation-specific configuration
 */
export interface AutomationResourceConfig extends TypedBaseResourceConfig {
    baseUrl: string;
    apiKey?: string;
    workspaceId?: string;
    webhookUrl?: string;
    maxWorkflows?: number;
}

/**
 * Agent-specific configuration
 */
export interface AgentResourceConfig extends TypedBaseResourceConfig {
    baseUrl?: string;
    browserEndpoint?: string;
    maxInstances?: number;
    headless?: boolean;
    proxyUrl?: string;
}

/**
 * Storage-specific configuration
 */
export interface StorageResourceConfig extends TypedBaseResourceConfig {
    endpoint: string;
    accessKey?: string;
    secretKey?: string;
    bucket?: string;
    region?: string;
    useSSL?: boolean;
}

// ============================================================================
// Specific Resource Configurations
// ============================================================================

// AI Resource Configurations

export interface OllamaConfig extends AIResourceConfig {
    baseUrl: string;
    models?: string[];
    keepAlive?: string; // e.g., "5m"
    numThread?: number;
    numGpu?: number;
}

export interface LocalAIConfig extends AIResourceConfig {
    baseUrl: string;
    apiKey?: string;
    models?: string[];
    galleries?: string[];
}

export interface LlamaCppConfig extends AIResourceConfig {
    baseUrl: string;
    modelPath?: string;
    contextSize?: number;
    threads?: number;
    gpuLayers?: number;
}

export interface VLLMConfig extends AIResourceConfig {
    baseUrl: string;
    apiKey?: string;
    models?: string[];
    tensorParallelSize?: number;
    pipelineParallelSize?: number;
}

export interface TGIConfig extends AIResourceConfig {
    baseUrl: string;
    apiKey?: string;
    models?: string[];
    maxBatchPrefillTokens?: number;
    maxBatchTotalTokens?: number;
}

export interface ONNXConfig extends AIResourceConfig {
    baseUrl: string;
    modelPath?: string;
    executionProviders?: string[];
}

export interface WhisperConfig extends AIResourceConfig {
    baseUrl: string;
    modelSize?: "tiny" | "base" | "small" | "medium" | "large";
    language?: string;
    device?: "cpu" | "cuda";
}

export interface ComfyUIConfig extends AIResourceConfig {
    baseUrl: string;
    apiKey?: string;
    workflowsPath?: string;
    outputPath?: string;
}

export interface StableDiffusionConfig extends AIResourceConfig {
    baseUrl: string;
    apiKey?: string;
    models?: string[];
    samplers?: string[];
}

export interface CloudflareConfig extends AIResourceConfig {
    accountId: string;
    apiKey: string;
    baseUrl: string;
    models?: string[];
}

export interface OpenRouterConfig extends AIResourceConfig {
    apiKey: string;
    baseUrl: string;
    models?: string[];
    siteUrl?: string;
    siteName?: string;
}

// Automation Resource Configurations

export interface N8nConfig extends AutomationResourceConfig {
    baseUrl: string;
    apiKey?: string;
    username?: string;
    password?: string;
    workflowsPath?: string;
}

export interface NodeREDConfig extends AutomationResourceConfig {
    baseUrl: string;
    adminAuth?: {
        username: string;
        password: string;
    };
    flowsPath?: string;
}

export interface WindmillConfig extends AutomationResourceConfig {
    baseUrl: string;
    token: string;
    workspace: string;
    defaultWorkerGroup?: string;
}

export interface AutomatischConfig extends AutomationResourceConfig {
    baseUrl: string;
    apiKey: string;
    appKey?: string;
    appSecret?: string;
}

export interface ActivePiecesConfig extends AutomationResourceConfig {
    baseUrl: string;
    apiKey: string;
    environment?: string;
    projectId?: string;
}

export interface HuginnConfig extends AutomationResourceConfig {
    baseUrl: string;
    username: string;
    password: string;
    agentIdPrefix?: string;
}

export interface KestraConfig extends AutomationResourceConfig {
    baseUrl: string;
    apiKey?: string;
    namespace?: string;
    tenant?: string;
}

export interface BeehiveConfig extends AutomationResourceConfig {
    baseUrl: string;
    apiKey?: string;
    hivePath?: string;
}

export interface AirflowConfig extends AutomationResourceConfig {
    baseUrl: string;
    username?: string;
    password?: string;
    dagBagPath?: string;
}

export interface TemporalConfig extends AutomationResourceConfig {
    address: string; // Different from baseUrl - gRPC address
    namespace?: string;
    taskQueue?: string;
    identity?: string;
}

// Agent Resource Configurations

export interface BrowserlessConfig extends AgentResourceConfig {
    baseUrl: string;
    token?: string;
    concurrent?: number;
    timeout?: number;
    headless?: boolean;
}

export interface ClaudeCodeConfig extends AgentResourceConfig {
    executablePath?: string;
    workingDirectory?: string;
    sessionTimeout?: number;
    maxConcurrentSessions?: number;
    apiKey?: string;
}

// Storage Resource Configurations

export interface MinIOConfig extends StorageResourceConfig {
    endpoint: string;
    accessKey: string;
    secretKey: string;
    useSSL?: boolean;
    port?: number;
    region?: string;
}

export interface SeaweedFSConfig extends StorageResourceConfig {
    masterUrl: string;
    filerUrl?: string;
    accessKey?: string;
    secretKey?: string;
    replication?: string;
}

export interface GlusterFSConfig extends StorageResourceConfig {
    servers: string[];
    volume: string;
    mountPath?: string;
    transport?: "tcp" | "rdma";
}

export interface IPFSConfig extends StorageResourceConfig {
    apiUrl: string;
    gatewayUrl?: string;
    swarmAddresses?: string[];
    bootstrap?: string[];
}

export interface RcloneConfig extends StorageResourceConfig {
    configPath: string;
    remote: string;
    flags?: string[];
    transfers?: number;
}

/**
 * Maps resource IDs to their specific configuration types
 */
export interface ResourceConfigMap {
    // AI Resources
    ollama: OllamaConfig;
    localai: LocalAIConfig;
    llamacpp: LlamaCppConfig;
    vllm: VLLMConfig;
    tgi: TGIConfig;
    onnx: ONNXConfig;
    whisper: WhisperConfig;
    comfyui: ComfyUIConfig;
    stablediffusion: StableDiffusionConfig;
    cloudflare: CloudflareConfig;
    openrouter: OpenRouterConfig;
    
    // Automation Resources
    n8n: N8nConfig;
    nodered: NodeREDConfig;
    windmill: WindmillConfig;
    automatisch: AutomatischConfig;
    activepieces: ActivePiecesConfig;
    huginn: HuginnConfig;
    kestra: KestraConfig;
    beehive: BeehiveConfig;
    airflow: AirflowConfig;
    temporal: TemporalConfig;
    
    // Agent Resources
    browserless: BrowserlessConfig;
    "claude-code": ClaudeCodeConfig;
    
    // Storage Resources
    minio: MinIOConfig;
    seaweedfs: SeaweedFSConfig;
    glusterfs: GlusterFSConfig;
    ipfs: IPFSConfig;
    rclone: RcloneConfig;
}

/**
 * Extract config type for a specific resource ID
 */
export type GetConfig<TId extends keyof ResourceConfigMap> = ResourceConfigMap[TId];

// ============================================================================
// Metadata Types
// ============================================================================

/**
 * Base metadata interface
 */
export interface BaseMetadata {
    version?: string;
    capabilities: string[];
    lastUpdated: Date;
    discoveredAt?: Date;
}

/**
 * AI-specific metadata
 */
export interface AIMetadata extends BaseMetadata {
    supportedModels: string[];
    maxTokens?: number;
    costPerToken?: number;
    contextWindow?: number;
}

/**
 * Ollama-specific metadata extending base AI metadata
 */
export interface OllamaMetadata extends AIMetadata {
    totalModels: number;
    baseUrl?: string;
    modelSizes: Array<{
        name: string;
        sizeBytes: number;
        family?: string;
        parameterSize?: string;
    }>;
}

/**
 * Automation-specific metadata
 */
export interface AutomationMetadata extends BaseMetadata {
    workflowCount?: number;
    activeWorkflowCount?: number;
    executionCount?: number;
    supportedTriggers: string[];
    supportedActions?: string[];
    integrationCount?: number;
    nodeVersion?: string;
    platform?: string;
    executionMode?: string;
    communityNodesEnabled?: boolean;
    templatesEnabled?: boolean;
    authRequired?: boolean;
    webhookUrl?: string;
}

/**
 * Agent-specific metadata
 */
export interface AgentMetadata extends BaseMetadata {
    activeInstances: number;
    maxInstances: number;
    supportedBrowsers: string[];
    queuedInstances?: number;
    stats?: Record<string, any>;
    pressure?: Record<string, any>;
    timeout?: number;
    baseUrl?: string;
}

/**
 * Storage-specific metadata
 */
export interface StorageMetadata extends BaseMetadata {
    totalSpace?: number;
    usedSpace?: number;
    bucketCount?: number;
    objectCount?: number;
}

// ============================================================================
// Type Mappings
// ============================================================================

/**
 * Maps resource IDs to their categories
 */
export type ResourceCategoryMap = {
    [K in AIResourceId]: ResourceCategory.AI;
} & {
    [K in AutomationResourceId]: ResourceCategory.Automation;
} & {
    [K in AgentResourceId]: ResourceCategory.Agents;
} & {
    [K in StorageResourceId]: ResourceCategory.Storage;
};

/**
 * Maps resource categories to their configuration types
 */
export interface CategoryConfigMap {
    [ResourceCategory.AI]: AIResourceConfig;
    [ResourceCategory.Automation]: AutomationResourceConfig;
    [ResourceCategory.Agents]: AgentResourceConfig;
    [ResourceCategory.Storage]: StorageResourceConfig;
}

/**
 * Maps resource categories to their metadata types
 */
export interface CategoryMetadataMap {
    [ResourceCategory.AI]: AIMetadata;
    [ResourceCategory.Automation]: AutomationMetadata;
    [ResourceCategory.Agents]: AgentMetadata;
    [ResourceCategory.Storage]: StorageMetadata;
}

/**
 * Maps specific resource IDs to their custom metadata types
 * This allows providers to extend beyond their category's base metadata
 */
export interface ProviderMetadataMap {
    "ollama": OllamaMetadata;
    // Other providers use their category's default metadata type
}

/**
 * Maps resource categories to their interface types
 */
export interface CategoryInterfaceMap {
    [ResourceCategory.AI]: IAIResource;
    [ResourceCategory.Automation]: IAutomationResource;
    [ResourceCategory.Agents]: IAgentResource;
    [ResourceCategory.Storage]: IStorageResource;
}

// ============================================================================
// Utility Types
// ============================================================================

/**
 * Extract category from resource ID
 */
export type GetResourceCategory<TId extends ResourceId> = 
    TId extends keyof ResourceCategoryMap ? ResourceCategoryMap[TId] : never;

/**
 * Extract config type from resource ID
 */
export type GetResourceConfig<TId extends ResourceId> = 
    CategoryConfigMap[GetResourceCategory<TId>];

/**
 * Extract metadata type from resource ID
 * Checks for provider-specific metadata first, then falls back to category metadata
 */
export type GetResourceMetadata<TId extends ResourceId> = 
    TId extends keyof ProviderMetadataMap 
        ? ProviderMetadataMap[TId]
        : CategoryMetadataMap[GetResourceCategory<TId>];

/**
 * Extract interface type from resource ID
 */
export type GetResourceInterface<TId extends ResourceId> = 
    CategoryInterfaceMap[GetResourceCategory<TId>];

/**
 * Get all resource IDs for a specific category
 */
export type GetResourceIdsByCategory<TCategory extends ResourceCategory> = 
    TCategory extends ResourceCategory.AI ? AIResourceId :
    TCategory extends ResourceCategory.Automation ? AutomationResourceId :
    TCategory extends ResourceCategory.Agents ? AgentResourceId :
    TCategory extends ResourceCategory.Storage ? StorageResourceId :
    never;

// ============================================================================
// Resource Implementation Registry
// ============================================================================

/**
 * This will be populated by actual resource implementations
 * For now, it's a placeholder for the type system
 */
export interface ResourceImplementationRegistry {
    // Will be extended by actual implementations
    // e.g., ollama?: OllamaResource;
}

/**
 * Helper type to check if a resource is implemented
 */
export type IsResourceImplemented<TId extends ResourceId> = 
    TId extends keyof ResourceImplementationRegistry ? true : false;

/**
 * Get implementation type for a resource ID
 */
export type GetResourceImplementation<TId extends ResourceId> = 
    TId extends keyof ResourceImplementationRegistry 
        ? ResourceImplementationRegistry[TId] 
        : never;

// ============================================================================
// Type Guards
// ============================================================================

/**
 * Check if a string is a valid resource ID
 */
export function isResourceId(id: string): id is ResourceId {
    const allIds: ResourceId[] = [
        // AI
        "ollama", "localai", "llamacpp", "vllm", "tgi", "onnx", 
        "whisper", "comfyui", "stablediffusion", "cloudflare", "openrouter",
        // Automation
        "n8n", "nodered", "windmill", "automatisch", "activepieces",
        "huginn", "kestra", "beehive", "airflow", "temporal",
        // Agents
        "browserless", "claude-code",
        // Storage
        "minio", "seaweedfs", "glusterfs", "ipfs", "rclone",
    ];
    return allIds.includes(id as ResourceId);
}

/**
 * Check if a resource ID belongs to a specific category
 */
export function isResourceInCategory<TCategory extends ResourceCategory>(
    id: ResourceId,
    category: TCategory,
): id is GetResourceIdsByCategory<TCategory> {
    const categoryMap: Record<ResourceCategory, ResourceId[]> = {
        [ResourceCategory.AI]: [
            "ollama", "localai", "llamacpp", "vllm", "tgi", "onnx",
            "whisper", "comfyui", "stablediffusion", "cloudflare", "openrouter",
        ],
        [ResourceCategory.Automation]: [
            "n8n", "nodered", "windmill", "automatisch", "activepieces",
            "huginn", "kestra", "beehive", "airflow", "temporal",
        ],
        [ResourceCategory.Agents]: [
            "browserless", "claude-code",
        ],
        [ResourceCategory.Storage]: [
            "minio", "seaweedfs", "glusterfs", "ipfs", "rclone",
        ],
    };
    
    return categoryMap[category]?.includes(id) ?? false;
}

/**
 * Type guard for AI resources
 */
export function isAIResource(resource: IResource): resource is IAIResource {
    return resource.category === ResourceCategory.AI;
}

/**
 * Type guard for automation resources
 */
export function isAutomationResource(resource: IResource): resource is IAutomationResource {
    return resource.category === ResourceCategory.Automation;
}

/**
 * Type guard for agent resources
 */
export function isAgentResource(resource: IResource): resource is IAgentResource {
    return resource.category === ResourceCategory.Agents;
}

/**
 * Type guard for storage resources
 */
export function isStorageResource(resource: IResource): resource is IStorageResource {
    return resource.category === ResourceCategory.Storage;
}

/**
 * Type guard to check if a config object matches a specific resource config
 */
export function isResourceConfig<TId extends keyof ResourceConfigMap>(
    config: unknown,
    resourceId: TId,
): config is ResourceConfigMap[TId] {
    if (!config || typeof config !== "object") {
        return false;
    }
    
    // Basic validation - check for required fields
    const c = config as Record<string, unknown>;
    
    // All configs must have enabled field
    if (typeof c.enabled !== "boolean") {
        return false;
    }
    
    // Category-specific validation
    switch (resourceId) {
        // AI resources typically need baseUrl
        case "ollama":
        case "localai":
        case "llamacpp":
        case "vllm":
        case "tgi":
        case "onnx":
        case "whisper":
        case "comfyui":
        case "stablediffusion":
            return typeof c.baseUrl === "string";
            
        // Cloud AI resources need API keys
        case "cloudflare":
            return typeof c.accountId === "string" && typeof c.apiKey === "string";
        case "openrouter":
            return typeof c.apiKey === "string";
            
        // Automation resources
        case "n8n":
        case "nodered":
        case "automatisch":
        case "activepieces":
        case "huginn":
        case "kestra":
        case "beehive":
        case "airflow":
            return typeof c.baseUrl === "string";
        case "windmill":
            return typeof c.baseUrl === "string" && typeof c.token === "string" && typeof c.workspace === "string";
        case "temporal":
            return typeof c.address === "string";
            
        // Agent resources
        case "browserless":
            return typeof c.baseUrl === "string";
        case "claude-code":
            return true; // Can work with defaults
            
        // Storage resources
        case "minio":
            return typeof c.endpoint === "string" && 
                   typeof c.accessKey === "string" && 
                   typeof c.secretKey === "string";
        case "seaweedfs":
            return typeof c.masterUrl === "string";
        case "glusterfs":
            return Array.isArray(c.servers) && typeof c.volume === "string";
        case "ipfs":
            return typeof c.apiUrl === "string";
        case "rclone":
            return typeof c.configPath === "string" && typeof c.remote === "string";
            
        default:
            return false;
    }
}
