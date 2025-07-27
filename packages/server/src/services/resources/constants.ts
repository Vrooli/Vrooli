import { ResourceCategory, DeploymentType } from "./types.js";

/**
 * Complete list of all resources that Vrooli supports
 * This is the "ideal" set of resources we aim to implement
 */
export const SUPPORTED_RESOURCES = {
    [ResourceCategory.AI]: [
        { id: "ollama", name: "Ollama", description: "Local LLM inference engine", deploymentType: DeploymentType.Local },
        { id: "whisper", name: "Whisper.cpp", description: "Local speech-to-text", deploymentType: DeploymentType.Local },
        { id: "comfyui", name: "ComfyUI", description: "Node-based UI for image generation", deploymentType: DeploymentType.Local },
        { id: "cloudflare", name: "Cloudflare Gateway AI", description: "Cloudflare's global AI inference network", deploymentType: DeploymentType.Cloud },
        { id: "openrouter", name: "OpenRouter", description: "Unified API for multiple AI providers", deploymentType: DeploymentType.Cloud },
    ],
    [ResourceCategory.Automation]: [
        { id: "n8n", name: "n8n", description: "Visual workflow automation", deploymentType: DeploymentType.Local },
        { id: "node-red", name: "Node-RED", description: "Flow-based programming", deploymentType: DeploymentType.Local },
        { id: "windmill", name: "Windmill", description: "Developer-centric workflow automation", deploymentType: DeploymentType.Hybrid },
        { id: "activepieces", name: "ActivePieces", description: "Open-source automation platform", deploymentType: DeploymentType.Local },
        { id: "huginn", name: "Huginn", description: "Agent-based monitoring and automation", deploymentType: DeploymentType.Local },
        { id: "airflow", name: "Apache Airflow", description: "Data pipeline orchestration", deploymentType: DeploymentType.Local },
        { id: "temporal", name: "Temporal", description: "Durable workflow execution", deploymentType: DeploymentType.Hybrid },
    ],
    [ResourceCategory.Agents]: [
        { id: "browserless", name: "Browserless", description: "Headless browser service", deploymentType: DeploymentType.Hybrid },
        { id: "claude-code", name: "Claude Code", description: "AI-powered coding assistant", deploymentType: DeploymentType.Local },
        { id: "agent-s2", name: "Agent S2", description: "Autonomous computer interaction with GUI automation", deploymentType: DeploymentType.Local },
    ],
    [ResourceCategory.Storage]: [
        { id: "minio", name: "MinIO", description: "S3-compatible object storage", deploymentType: DeploymentType.Local },
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

