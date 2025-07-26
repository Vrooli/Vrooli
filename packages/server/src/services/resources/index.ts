/**
 * Resources System
 * 
 * This system allows Vrooli to discover and interact with both local and cloud services
 * such as AI models (Ollama, OpenRouter), automation platforms (n8n), browser agents (Puppeteer),
 * and storage services (MinIO).
 * 
 * ## Usage:
 * 
 * ### 1. Creating a new resource provider:
 * ```typescript
 * import { ResourceProvider, RegisterResource, ResourceCategory, DeploymentType } from '@services/resources';
 * 
 * @RegisterResource
 * export class OllamaResource extends ResourceProvider<OllamaConfig> {
 *     readonly id = 'ollama';
 *     readonly category = ResourceCategory.AI;
 *     readonly displayName = 'Ollama';
 *     readonly description = 'Local LLM inference engine';
 *     readonly isSupported = true;
 *     readonly deploymentType = DeploymentType.Local;
 *     
 *     protected async performDiscovery(): Promise<boolean> {
 *         // Check if Ollama is running
 *         try {
 *             const result = await this.httpClient!.makeRequest({
 *                 url: `${this.config.baseUrl}/api/tags`,
 *                 method: "GET",
 *             });
 *             return result.success;
 *         } catch {
 *             return false;
 *         }
 *     }
 *     
 *     protected async performHealthCheck(): Promise<HealthCheckResult> {
 *         // Check Ollama health
 *         const result = await this.httpClient!.makeRequest({
 *             url: `${this.config.baseUrl}/api/tags`,
 *             method: "GET",
 *         });
 *         return {
 *             healthy: result.success,
 *             message: result.success ? 'Ollama is running' : 'Ollama is not responding',
 *             timestamp: new Date(),
 *         };
 *     }
 * }
 * ```
 * 
 * ### 2. Using the registry:
 * ```typescript
 * import { ResourceRegistry } from '@services/resources';
 * 
 * // Initialize the registry (usually done at server startup)
 * const registry = ResourceRegistry.getInstance();
 * await registry.initialize();
 * 
 * // Check if a resource is available
 * if (registry.isResourceAvailable('ollama')) {
 *     const ollama = registry.getResource<IAIResource>('ollama');
 *     const models = await ollama?.listModels();
 * }
 * 
 * // Get all available AI resources
 * const aiResources = registry.getResourcesByCategory(ResourceCategory.AI);
 * 
 * // Get resource summary
 * const summary = registry.getResourceSummary();
 * console.log(`${summary.available} of ${summary.total} resources available`);
 * ```
 * 
 * ### 3. Configuration:
 * Create `.vrooli/resources.local.json`:
 * ```json
 * {
 *   "enabled": true,
 *   "services": {
 *     "ai": {
 *       "ollama": {
 *         "enabled": true,
 *         "baseUrl": "http://localhost:11434"
 *       }
 *     }
 *   }
 * }
 * ```
 */

export * from "./constants.js";
export * from "./healthCheck.js";
export * from "./ResourceProvider.js";
export * from "./ResourceRegistry.js";
export * from "./resourcesConfig.js";
export * from "./types.js";

// Import resource providers to ensure @RegisterResource decorators run
import "./providers/OllamaResource.js";
import "./providers/N8nResource.js";
import "./providers/BrowserlessResource.js";

// Export specific provider classes for direct access
export { OllamaResource } from "./providers/OllamaResource.js";
export { N8nResource } from "./providers/N8nResource.js";
export { BrowserlessResource } from "./providers/BrowserlessResource.js";

// Example of how AI services will integrate with the existing system
export interface AIResourceAdapter {
    /**
     * Convert ResourceProvider to AIService interface
     * This allows resources to be used wherever AIService is expected
     */
    toAIService(): any; // Would return AIService instance
}

// Example of how automation services will integrate with routine execution
export interface AutomationResourceAdapter {
    /**
     * Execute a Vrooli routine using external automation service
     */
    executeRoutine(routineId: string, inputs: Record<string, any>): Promise<any>;

    /**
     * Convert Vrooli routine to external format (e.g., n8n workflow)
     */
    exportRoutine(routine: any): Promise<string>;
}

// Example of how agent services will integrate with tool system
export interface AgentResourceAdapter {
    /**
     * Create a tool definition for AI tiers to use this agent
     */
    toToolDefinition(): any; // Would return Tool interface

    /**
     * Execute agent action
     */
    executeAction(action: string, params: Record<string, any>): Promise<any>;
}

// Example of how storage services will integrate with file operations
export interface StorageResourceAdapter {
    /**
     * Upload file to storage service
     */
    uploadFile(bucket: string, key: string, data: Buffer): Promise<string>;

    /**
     * Download file from storage service
     */
    downloadFile(bucket: string, key: string): Promise<Buffer>;

    /**
     * Generate presigned URL for direct access
     */
    getPresignedUrl(bucket: string, key: string, expiresIn: number): Promise<string>;
}

