/**
 * Example integration of Resources health check with a server health endpoint
 * This demonstrates how to use the ResourceRegistry in your application
 */

import type { Request, Response } from "express";
import { ResourceRegistry, ResourceSystemHealth } from "./index.js";

// HTTP Status codes
const HTTP_OK = 200;
const HTTP_MULTI_STATUS = 207;
const HTTP_INTERNAL_SERVER_ERROR = 500;
const HTTP_SERVICE_UNAVAILABLE = 503;

/**
 * Example health check endpoint handler
 * Add this to your Express routes
 */
export async function resourcesHealthCheck(req: Request, res: Response): Promise<void> {
    try {
        const registry = ResourceRegistry.getInstance();
        const healthCheck = registry.getHealthCheck();
        
        // Determine HTTP status code based on health status
        let statusCode: number;
        switch (healthCheck.status) {
            case ResourceSystemHealth.Down:
                statusCode = HTTP_SERVICE_UNAVAILABLE;
                break;
            case ResourceSystemHealth.Degraded:
                statusCode = HTTP_MULTI_STATUS;
                break;
            case ResourceSystemHealth.Operational:
            default:
                statusCode = HTTP_OK;
                break;
        }
        
        // Add cache headers to prevent hammering
        res.setHeader("Cache-Control", "public, max-age=30"); // Cache for 30 seconds
        
        res.status(statusCode).json(healthCheck);
    } catch (error) {
        console.error("Failed to get resource health check:", error);
        res.status(HTTP_INTERNAL_SERVER_ERROR).json({
            status: ResourceSystemHealth.Down,
            message: "Failed to retrieve resource health status",
            timestamp: new Date(),
            error: error instanceof Error ? error.message : "Unknown error",
        });
    }
}

/**
 * Example: Initialize the registry at server startup
 */
export async function initializeResources(): Promise<void> {
    try {
        const registry = ResourceRegistry.getInstance();
        await registry.initialize();
        
        // Log initial status
        const healthCheck = registry.getHealthCheck();
        console.log(`Resources initialized: ${healthCheck.message}`);
        console.log(`- Total supported: ${healthCheck.stats.totalSupported}`);
        console.log(`- Implemented: ${healthCheck.stats.totalRegistered}`);
        console.log(`- Enabled: ${healthCheck.stats.totalEnabled}`);
        console.log(`- Active: ${healthCheck.stats.totalActive}`);
        
        // Subscribe to resource events
        registry.on("resource:discovered", (data) => {
            console.log(`Resource discovered: ${data.resourceId}`);
        });
        
        registry.on("resource:lost", (data) => {
            console.log(`Resource lost: ${data.resourceId}`);
        });
        
        registry.on("resource:health_changed", (data) => {
            console.log(`Resource health changed: ${data.resourceId}`);
        });
    } catch (error) {
        console.error("Failed to initialize resources:", error);
        // Don't throw - allow server to start even if resources fail
    }
}

/**
 * Example: Graceful shutdown
 */
export async function shutdownResources(): Promise<void> {
    try {
        const registry = ResourceRegistry.getInstance();
        await registry.shutdown();
        console.log("Resources shut down successfully");
    } catch (error) {
        console.error("Error shutting down resources:", error);
    }
}

/**
 * Example route setup (Express)
 * 
 * // In your routes file:
 * import { resourcesHealthCheck } from './services/resources/example-integration.js';
 * 
 * app.get('/health/resources', resourcesHealthCheck);
 * 
 * // In your server startup:
 * import { initializeResources } from './services/resources/example-integration.js';
 * 
 * await initializeResources();
 * 
 * // In your graceful shutdown:
 * import { shutdownResources } from './services/resources/example-integration.js';
 * 
 * process.on('SIGTERM', async () => {
 *     await shutdownResources();
 *     process.exit(0);
 * });
 */

