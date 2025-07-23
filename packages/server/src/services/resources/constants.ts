import { ResourceCategory } from "./types.js";

/**
 * Complete list of all resources that Vrooli supports
 * This is the "ideal" set of resources we aim to implement
 */
export const SUPPORTED_RESOURCES = {
    [ResourceCategory.AI]: [
        { id: "ollama", name: "Ollama", description: "Local LLM inference engine" },
        { id: "localai", name: "LocalAI", description: "OpenAI-compatible local API" },
        { id: "llamacpp", name: "llama.cpp", description: "Direct C++ inference" },
        { id: "onnx", name: "ONNX Runtime", description: "Optimized inference for smaller models" },
        { id: "whisper", name: "Whisper.cpp", description: "Local speech-to-text" },
        { id: "stablediffusion", name: "Stable Diffusion WebUI", description: "Local image generation" },
    ],
    [ResourceCategory.Automation]: [
        { id: "n8n", name: "n8n", description: "Visual workflow automation" },
        { id: "nodered", name: "Node-RED", description: "Flow-based programming" },
        { id: "airflow", name: "Apache Airflow", description: "Data pipeline orchestration" },
        { id: "temporal", name: "Temporal", description: "Durable workflow execution" },
        { id: "make", name: "Make.com", description: "Visual automation platform" },
    ],
    [ResourceCategory.Agents]: [
        { id: "puppeteer", name: "Puppeteer", description: "Headless Chrome automation" },
        { id: "playwright", name: "Playwright", description: "Multi-browser automation" },
        { id: "selenium", name: "Selenium Grid", description: "Distributed browser testing" },
        { id: "browserless", name: "Browserless", description: "Headless browser service" },
        { id: "brightdata", name: "Bright Data", description: "Proxy-based scraping" },
        { id: "apify", name: "Apify", description: "Actor-based web automation" },
    ],
    [ResourceCategory.Storage]: [
        { id: "minio", name: "MinIO", description: "S3-compatible object storage" },
        { id: "seaweedfs", name: "SeaweedFS", description: "Distributed file system" },
        { id: "glusterfs", name: "GlusterFS", description: "Scalable network filesystem" },
        { id: "ipfs", name: "IPFS", description: "Decentralized storage" },
        { id: "rclone", name: "Rclone", description: "Universal cloud storage interface" },
    ],
} as const;

/**
 * Get total count of all supported resources
 */
export function getTotalSupportedResourceCount(): number {
    return Object.values(SUPPORTED_RESOURCES).reduce(
        (sum, resources) => sum + resources.length,
        0,
    );
}

/**
 * Get all supported resource IDs
 */
export function getAllSupportedResourceIds(): string[] {
    const ids: string[] = [];
    for (const resources of Object.values(SUPPORTED_RESOURCES)) {
        for (const resource of resources) {
            ids.push(resource.id);
        }
    }
    return ids;
}

/**
 * Check if a resource ID is in our supported list
 */
export function isSupportedResourceId(id: string): boolean {
    return getAllSupportedResourceIds().includes(id);
}

