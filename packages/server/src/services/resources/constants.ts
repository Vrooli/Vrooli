import { ResourceCategory, DeploymentType } from "./types.js";

/**
 * Complete list of all resources that Vrooli supports
 * This is the "ideal" set of resources we aim to implement
 */
export const SUPPORTED_RESOURCES = {
    [ResourceCategory.AI]: [
        { id: "ollama", name: "Ollama", description: "Local LLM inference engine", deploymentType: DeploymentType.Local },
        { id: "localai", name: "LocalAI", description: "OpenAI-compatible local API", deploymentType: DeploymentType.Local },
        { id: "llamacpp", name: "llama.cpp", description: "Direct C++ inference", deploymentType: DeploymentType.Local },
        { id: "vllm", name: "vLLM", description: "High-performance LLM inference engine", deploymentType: DeploymentType.Local },
        { id: "tgi", name: "Text Generation Inference", description: "Hugging Face production inference server", deploymentType: DeploymentType.Local },
        { id: "onnx", name: "ONNX Runtime", description: "Optimized inference for smaller models", deploymentType: DeploymentType.Local },
        { id: "whisper", name: "Whisper.cpp", description: "Local speech-to-text", deploymentType: DeploymentType.Local },
        { id: "comfyui", name: "ComfyUI", description: "Node-based UI for image generation", deploymentType: DeploymentType.Local },
        { id: "stablediffusion", name: "Stable Diffusion WebUI", description: "Local image generation", deploymentType: DeploymentType.Local },
        { id: "cloudflare", name: "Cloudflare Gateway AI", description: "Cloudflare's global AI inference network", deploymentType: DeploymentType.Cloud },
        { id: "openrouter", name: "OpenRouter", description: "Unified API for multiple AI providers", deploymentType: DeploymentType.Cloud },
    ],
    [ResourceCategory.Automation]: [
        { id: "n8n", name: "n8n", description: "Visual workflow automation", deploymentType: DeploymentType.Local },
        { id: "nodered", name: "Node-RED", description: "Flow-based programming", deploymentType: DeploymentType.Local },
        { id: "windmill", name: "Windmill", description: "Developer-centric workflow automation", deploymentType: DeploymentType.Hybrid },
        { id: "automatisch", name: "Automatisch", description: "Open-source Zapier alternative", deploymentType: DeploymentType.Local },
        { id: "activepieces", name: "ActivePieces", description: "Open-source automation platform", deploymentType: DeploymentType.Local },
        { id: "huginn", name: "Huginn", description: "Agent-based monitoring and automation", deploymentType: DeploymentType.Local },
        { id: "kestra", name: "Kestra", description: "Data pipeline orchestration", deploymentType: DeploymentType.Local },
        { id: "beehive", name: "Beehive", description: "Event-driven automation system", deploymentType: DeploymentType.Local },
        { id: "airflow", name: "Apache Airflow", description: "Data pipeline orchestration", deploymentType: DeploymentType.Local },
        { id: "temporal", name: "Temporal", description: "Durable workflow execution", deploymentType: DeploymentType.Hybrid },
    ],
    [ResourceCategory.Agents]: [
        { id: "puppeteer", name: "Puppeteer", description: "Headless Chrome automation", deploymentType: DeploymentType.Local },
        { id: "playwright", name: "Playwright", description: "Multi-browser automation", deploymentType: DeploymentType.Local },
        { id: "selenium", name: "Selenium Grid", description: "Distributed browser testing", deploymentType: DeploymentType.Local },
        { id: "browserless", name: "Browserless", description: "Headless browser service", deploymentType: DeploymentType.Hybrid },
    ],
    [ResourceCategory.Storage]: [
        { id: "minio", name: "MinIO", description: "S3-compatible object storage", deploymentType: DeploymentType.Local },
        { id: "seaweedfs", name: "SeaweedFS", description: "Distributed file system", deploymentType: DeploymentType.Local },
        { id: "glusterfs", name: "GlusterFS", description: "Scalable network filesystem", deploymentType: DeploymentType.Local },
        { id: "ipfs", name: "IPFS", description: "Decentralized storage", deploymentType: DeploymentType.Local },
        { id: "rclone", name: "Rclone", description: "Universal cloud storage interface", deploymentType: DeploymentType.Local },
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

