/**
 * Unstructured.io Document Processing Resource Implementation
 * 
 * Provides infrastructure-level management for Unstructured.io service including:
 * - Service discovery and health monitoring
 * - Document processing capabilities detection
 * - Format support enumeration
 * - Processing strategy management
 * - Integration with unified configuration system
 */

import { MINUTES_5_MS } from "@vrooli/shared";
import { logger } from "../../../events/logger.js";
import { ResourceProvider } from "../ResourceProvider.js";
import { RegisterResource } from "../ResourceRegistry.js";
import type { AIMetadata, UnstructuredIoConfig } from "../typeRegistry.js";
import type { HealthCheckResult, IAIResource } from "../types.js";
import { DeploymentType, ResourceCategory, ResourceHealth } from "../types.js";

/**
 * Document processing capabilities
 */
interface ProcessingCapabilities {
    strategies: string[];
    outputFormats: string[];
    supportedLanguages: string[];
    maxFileSize: number;
    ocrEnabled: boolean;
    tableExtraction: boolean;
}

/**
 * Supported document format information
 */
interface DocumentFormat {
    extension: string;
    mimeType?: string;
    category: "document" | "spreadsheet" | "presentation" | "image" | "email" | "other";
    requiresOCR?: boolean;
}

/**
 * Extended metadata for Unstructured.io
 */
export interface UnstructuredIoMetadata extends AIMetadata {
    processingCapabilities: ProcessingCapabilities;
    supportedFormats: DocumentFormat[];
    apiVersion: string;
}

/**
 * Production Unstructured.io resource implementation with document processing capabilities
 */
@RegisterResource
export class UnstructuredIoResource extends ResourceProvider<"unstructured-io", UnstructuredIoConfig> implements IAIResource {
    // Resource identification
    readonly id = "unstructured-io";
    readonly category = ResourceCategory.AI;
    readonly displayName = "Unstructured.io";
    readonly description = "Document processing and extraction service for AI consumption";
    readonly isSupported = true;
    readonly deploymentType = DeploymentType.Local;

    // Processing capabilities caching
    private capabilities: ProcessingCapabilities | null = null;
    private lastCapabilityRefresh = 0;
    private readonly CACHE_TTL_MS = MINUTES_5_MS;

    // Supported formats definition
    private readonly SUPPORTED_FORMATS: DocumentFormat[] = [
        // Documents
        { extension: "pdf", mimeType: "application/pdf", category: "document" },
        { extension: "docx", mimeType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document", category: "document" },
        { extension: "doc", mimeType: "application/msword", category: "document" },
        { extension: "txt", mimeType: "text/plain", category: "document" },
        { extension: "rtf", mimeType: "application/rtf", category: "document" },
        { extension: "odt", mimeType: "application/vnd.oasis.opendocument.text", category: "document" },
        { extension: "md", mimeType: "text/markdown", category: "document" },
        { extension: "rst", mimeType: "text/x-rst", category: "document" },
        { extension: "html", mimeType: "text/html", category: "document" },
        { extension: "xml", mimeType: "application/xml", category: "document" },
        { extension: "epub", mimeType: "application/epub+zip", category: "document" },

        // Spreadsheets
        { extension: "xlsx", mimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", category: "spreadsheet" },
        { extension: "xls", mimeType: "application/vnd.ms-excel", category: "spreadsheet" },

        // Presentations
        { extension: "pptx", mimeType: "application/vnd.openxmlformats-officedocument.presentationml.presentation", category: "presentation" },
        { extension: "ppt", mimeType: "application/vnd.ms-powerpoint", category: "presentation" },

        // Images (with OCR)
        { extension: "png", mimeType: "image/png", category: "image", requiresOCR: true },
        { extension: "jpg", mimeType: "image/jpeg", category: "image", requiresOCR: true },
        { extension: "jpeg", mimeType: "image/jpeg", category: "image", requiresOCR: true },
        { extension: "tiff", mimeType: "image/tiff", category: "image", requiresOCR: true },
        { extension: "bmp", mimeType: "image/bmp", category: "image", requiresOCR: true },
        { extension: "heic", mimeType: "image/heic", category: "image", requiresOCR: true },

        // Email
        { extension: "eml", mimeType: "message/rfc822", category: "email" },
        { extension: "msg", mimeType: "application/vnd.ms-outlook", category: "email" },
    ];

    /**
     * Perform service discovery by checking Unstructured.io API availability
     */
    protected async performDiscovery(): Promise<boolean> {
        if (!this.config?.baseUrl) {
            logger.debug(`[${this.id}] No baseUrl configured`);
            return false;
        }

        try {
            // Check health endpoint
            const result = await this.httpClient!.makeRequest({
                url: `${this.config.baseUrl}/healthcheck`,
                method: "GET",
                timeout: this.config.healthCheck?.timeoutMs || 5000,
            });

            return result.success && result.status === 200;
        } catch (error) {
            logger.debug(`[${this.id}] Discovery failed:`, error);
            return false;
        }
    }

    /**
     * Perform comprehensive health check
     */
    protected async performHealthCheck(): Promise<HealthCheckResult> {
        if (!this.config?.baseUrl) {
            return {
                healthy: false,
                message: "No baseUrl configured",
                timestamp: new Date(),
            };
        }

        try {
            // Check health endpoint
            const healthResult = await this.httpClient!.makeRequest({
                url: `${this.config.baseUrl}/healthcheck`,
                method: "GET",
                timeout: this.config.healthCheck?.timeoutMs || 5000,
            });

            if (!healthResult.success || healthResult.status !== 200) {
                return {
                    healthy: false,
                    message: "Service not responding",
                    timestamp: new Date(),
                    details: { error: healthResult.error },
                };
            }

            // Try to get processing capabilities
            await this.refreshCapabilities();

            const capabilityInfo = this.capabilities
                ? `Supporting ${this.SUPPORTED_FORMATS.length} formats with ${this.capabilities.strategies.length} strategies`
                : "Capabilities not yet determined";

            return {
                healthy: true,
                message: `Unstructured.io service is healthy - ${capabilityInfo}`,
                timestamp: new Date(),
                details: {
                    baseUrl: this.config.baseUrl,
                    processingStrategy: this.config.processing?.defaultStrategy || "hi_res",
                    supportedFormats: this.SUPPORTED_FORMATS.length,
                },
            };

        } catch (error) {
            logger.error(`[${this.id}] Health check failed:`, error);
            return {
                healthy: false,
                message: error instanceof Error ? error.message : "Health check failed",
                timestamp: new Date(),
            };
        }
    }

    /**
     * Get resource-specific metadata
     */
    protected getMetadata(): UnstructuredIoMetadata {
        return {
            version: "latest",
            apiVersion: "v0",
            capabilities: ["document-processing", "ocr", "table-extraction", "layout-analysis"],
            lastUpdated: new Date(),
            discoveredAt: this._lastHealthCheck,
            supportedModels: [], // Not applicable for document processing
            contextWindow: undefined,
            maxTokens: undefined,
            costPerToken: undefined,
            processingCapabilities: this.capabilities || {
                strategies: ["fast", "hi_res", "auto"],
                outputFormats: ["json", "markdown", "text", "elements"],
                supportedLanguages: ["eng"],
                maxFileSize: 52428800, // 50MB default
                ocrEnabled: true,
                tableExtraction: true,
            },
            supportedFormats: this.SUPPORTED_FORMATS,
        };
    }

    /**
     * List available models (not applicable for Unstructured.io)
     */
    async listModels(): Promise<string[]> {
        // Unstructured.io doesn't have selectable models
        return [];
    }

    /**
     * Check if a specific model is available (not applicable)
     */
    async hasModel(_modelId: string): Promise<boolean> {
        // Unstructured.io doesn't have selectable models
        return false;
    }

    /**
     * Process a document
     */
    async processDocument(file: Buffer, options?: {
        strategy?: "fast" | "hi_res" | "auto";
        languages?: string[];
        includePageBreaks?: boolean;
        outputFormat?: "json" | "markdown" | "text";
    }): Promise<any> {
        if (this._health !== ResourceHealth.Healthy) {
            throw new Error("Unstructured.io service is not healthy");
        }

        const formData = new FormData();
        formData.append("files", new Blob([file]));
        formData.append("strategy", options?.strategy || this.config.processing?.defaultStrategy || "hi_res");

        if (options?.languages) {
            formData.append("languages", options.languages.join(","));
        }

        formData.append("include_page_breaks", String(options?.includePageBreaks ?? true));

        const result = await this.httpClient!.makeRequest({
            url: `${this.config.baseUrl}/general/v0/general`,
            method: "POST",
            body: formData,
            timeout: 300000, // 5 minutes for large documents
        });

        if (!result.success) {
            throw new Error(`Document processing failed: ${result.error}`);
        }

        // Convert to requested format if needed
        if (options?.outputFormat && options.outputFormat !== "json") {
            return this.convertOutput(result.data, options.outputFormat);
        }

        return result.data;
    }

    /**
     * Get supported file formats
     */
    getSupportedFormats(): DocumentFormat[] {
        return this.SUPPORTED_FORMATS;
    }

    /**
     * Check if a file type is supported
     */
    isFormatSupported(extension: string): boolean {
        const ext = extension.toLowerCase().replace(".", "");
        return this.SUPPORTED_FORMATS.some(f => f.extension === ext);
    }

    /**
     * Refresh processing capabilities
     */
    private async refreshCapabilities(): Promise<void> {
        const now = Date.now();
        if (this.capabilities && now - this.lastCapabilityRefresh < this.CACHE_TTL_MS) {
            return; // Use cached data
        }

        // Unstructured.io doesn't expose a capabilities endpoint,
        // so we use hardcoded defaults based on documentation
        this.capabilities = {
            strategies: ["fast", "hi_res", "auto"],
            outputFormats: ["json", "markdown", "text", "elements"],
            supportedLanguages: this.config.processing?.ocrLanguages || ["eng"],
            maxFileSize: 52428800, // 50MB
            ocrEnabled: true,
            tableExtraction: true,
        };

        this.lastCapabilityRefresh = now;
        logger.debug(`[${this.id}] Updated processing capabilities`);
    }

    /**
     * Convert JSON output to other formats
     */
    private convertOutput(data: any, format: "markdown" | "text"): string {
        if (!Array.isArray(data)) {
            return "";
        }

        if (format === "text") {
            return data.map(element => element.text || "").join("\n");
        }

        // Convert to markdown
        return data.map(element => {
            switch (element.type) {
                case "Title":
                    return `# ${element.text}\n`;
                case "Header":
                    return `## ${element.text}\n`;
                case "NarrativeText":
                    return `${element.text}\n`;
                case "ListItem":
                    return `- ${element.text}`;
                case "Table":
                    return `\`\`\`\n${element.text}\n\`\`\``;
                case "PageBreak":
                    return "\n---\n";
                default:
                    return element.text || "";
            }
        }).join("\n");
    }
}
