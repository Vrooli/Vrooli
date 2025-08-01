/**
 * Port Registry Integration for TypeScript
 * 
 * This module provides TypeScript access to the port registry defined in shell scripts,
 * enabling smart discovery and auto-correction of misconfigured resources.
 * Dynamically loads ports from scripts/resources/port-registry.sh at startup.
 */

import { execSync } from "child_process";
import path from "path";
import { logger } from "../../events/logger.js";

// Standard HTTP/HTTPS ports for URL validation
const STANDARD_HTTP_PORT = 80;
const STANDARD_HTTPS_PORT = 443;
const MAX_PORT_NUMBER = 65535;

/**
 * Interface for the JSON output from the shell script
 */
interface PortsJson {
    resource_ports: Record<string, number>;
}

/**
 * Possible locations for the port registry shell script
 */
const POSSIBLE_SCRIPT_PATHS = [
    path.join(process.cwd(), "scripts/resources/port-registry.sh"),
    path.join(__dirname, "../../../../../scripts/resources/port-registry.sh"),
    "/app/scripts/resources/port-registry.sh",  // Docker path
];

/**
 * Load resource ports from the shell script
 * This is called once at module initialization
 */
function loadPortsFromShellScript(): Record<string, number> {
    let lastError: Error | null = null;

    // Try each possible path
    for (const scriptPath of POSSIBLE_SCRIPT_PATHS) {
        try {
            logger.debug(`[PortRegistry] Attempting to load ports from: ${scriptPath}`);

            const output = execSync(`bash "${scriptPath}" --export-json`, {
                encoding: "utf8",
                timeout: 5000,
                env: process.env,
                stdio: ["ignore", "pipe", "pipe"],
            });

            const parsed: PortsJson = JSON.parse(output.trim());

            // Validate the structure
            if (!parsed.resource_ports || typeof parsed.resource_ports !== "object") {
                throw new Error("Invalid JSON structure: missing resource_ports");
            }

            // Validate port numbers
            for (const [resource, port] of Object.entries(parsed.resource_ports)) {
                if (!Number.isInteger(port) || port < 1 || port > MAX_PORT_NUMBER) {
                    throw new Error(`Invalid port number ${port} for resource ${resource}`);
                }
            }

            logger.info(`[PortRegistry] Successfully loaded ${Object.keys(parsed.resource_ports).length} resource ports from ${scriptPath}`);
            return parsed.resource_ports;

        } catch (error) {
            lastError = error as Error;
            logger.debug(`[PortRegistry] Failed to load from ${scriptPath}: ${error}`);
        }
    }

    // If we get here, all paths failed
    const errorMsg = `Failed to load port registry from shell script. Last error: ${lastError?.message}`;
    logger.error(`[PortRegistry] ${errorMsg}`);
    throw new Error(errorMsg);
}

/**
 * Resource ports loaded from the shell script
 * This is populated once at module initialization
 */
const RESOURCE_PORTS: Record<string, number> = loadPortsFromShellScript();

/**
 * Get the registered port for a resource
 * @param resourceId - The resource identifier
 * @returns The port number if found, undefined otherwise
 */
export function getResourcePort(resourceId: string): number | undefined {
    return RESOURCE_PORTS[resourceId];
}

/**
 * Get the default base URL for a resource
 * @param resourceId - The resource identifier
 * @returns The base URL if port is known, undefined otherwise
 */
export function getResourceBaseUrl(resourceId: string): string | undefined {
    const port = getResourcePort(resourceId);
    if (!port) {
        return undefined;
    }
    return `http://localhost:${port}`;
}

/**
 * Check if a URL matches the expected port for a resource
 * @param resourceId - The resource identifier
 * @param url - The URL to check
 * @returns true if the URL uses the expected port
 */
export function isCorrectResourceUrl(resourceId: string, url: string): boolean {
    const expectedPort = getResourcePort(resourceId);
    if (!expectedPort) {
        return true; // Can't validate unknown resources
    }

    try {
        const parsedUrl = new URL(url);
        const urlPort = parsedUrl.port || (parsedUrl.protocol === "https:" ? STANDARD_HTTPS_PORT : STANDARD_HTTP_PORT);
        return urlPort === String(expectedPort);
    } catch (error) {
        logger.warn(`[PortRegistry] Invalid URL: ${url}`);
        return false;
    }
}

/**
 * Get all known resource IDs
 * @returns Array of resource IDs
 */
export function getAllResourceIds(): string[] {
    return Object.keys(RESOURCE_PORTS);
}

/**
 * Log port registry information (for debugging)
 */
export function logPortRegistry(): void {
    logger.info("[PortRegistry] Known resource ports:");
    for (const [resource, port] of Object.entries(RESOURCE_PORTS)) {
        logger.info(`  ${resource}: ${port}`);
    }
}

