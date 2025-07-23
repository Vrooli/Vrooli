import { ResourceCategory, DiscoveryStatus, ResourceHealth } from "./types.js";
import type { ResourceInfo } from "./types.js";
import { 
    SUPPORTED_RESOURCES, 
    getTotalSupportedResourceCount,
} from "./constants.js";

/**
 * Health status for the resources system
 */
export enum ResourceSystemHealth {
    /** All enabled services are available */
    Operational = "Operational",
    /** Some enabled services are not available */
    Degraded = "Degraded",
    /** All enabled services are unavailable */
    Down = "Down",
}

/**
 * Detailed information about a resource for health check
 */
export interface ResourceHealthInfo {
    id: string;
    name: string;
    category: ResourceCategory;
    enabled: boolean;
    registered: boolean;
    available: boolean;
    health?: string;
    lastCheck?: Date;
    error?: string;
}

/**
 * Complete health check result for the resources system
 */
export interface ResourceSystemHealthCheck {
    /** Overall system health status */
    status: ResourceSystemHealth;
    /** Human-readable status message */
    message: string;
    /** Timestamp of this health check */
    timestamp: Date;
    /** Detailed statistics */
    stats: {
        /** Total number of resources we aim to support */
        totalSupported: number;
        /** Number of resources actually implemented */
        totalRegistered: number;
        /** Number of resources enabled in config */
        totalEnabled: number;
        /** Number of resources confirmed available and healthy */
        totalActive: number;
    };
    /** Resources that we support but haven't implemented yet */
    missingImplementations: Array<{
        id: string;
        name: string;
        category: ResourceCategory;
    }>;
    /** Resources that are enabled but not available */
    unavailableServices: Array<{
        id: string;
        name: string;
        category: ResourceCategory;
        reason: string;
    }>;
    /** Breakdown by category */
    categories: Record<ResourceCategory, {
        supported: number;
        registered: number;
        enabled: number;
        active: number;
    }>;
    /** All resources with their current status */
    resources: ResourceHealthInfo[];
}

/**
 * Calculate the overall system health based on resource states
 */
export function calculateSystemHealth(
    enabledCount: number,
    activeCount: number,
    unavailableServices: unknown[],
): { status: ResourceSystemHealth; message: string } {
    // No enabled services
    if (enabledCount === 0) {
        return {
            status: ResourceSystemHealth.Operational,
            message: "No services enabled",
        };
    }
    
    // All enabled services are down
    if (activeCount === 0) {
        return {
            status: ResourceSystemHealth.Down,
            message: `All ${enabledCount} enabled services are unavailable`,
        };
    }
    
    // Some enabled services are down
    if (unavailableServices.length > 0) {
        return {
            status: ResourceSystemHealth.Degraded,
            message: `${activeCount} of ${enabledCount} enabled services are available`,
        };
    }
    
    // All enabled services are up
    return {
        status: ResourceSystemHealth.Operational,
        message: `All ${enabledCount} enabled services are available`,
    };
}

/**
 * Build a comprehensive health check from resource information
 */
export function buildHealthCheck(
    registeredResources: Map<string, ResourceInfo>,
    enabledResourceIds: Set<string>,
): ResourceSystemHealthCheck {
    const timestamp = new Date();
    const stats = {
        totalSupported: getTotalSupportedResourceCount(),
        totalRegistered: registeredResources.size,
        totalEnabled: 0,
        totalActive: 0,
    };
    
    // Category breakdown
    const categoryStats: Record<string, { supported: number; registered: number; enabled: number; active: number }> = {};
    for (const category of Object.values(ResourceCategory)) {
        categoryStats[category] = {
            supported: SUPPORTED_RESOURCES[category]?.length || 0,
            registered: 0,
            enabled: 0,
            active: 0,
        };
    }
    
    // Missing implementations
    const missingImplementations: ResourceSystemHealthCheck["missingImplementations"] = [];
    const registeredIds = new Set(registeredResources.keys());
    
    for (const [category, resources] of Object.entries(SUPPORTED_RESOURCES)) {
        for (const resource of resources) {
            if (!registeredIds.has(resource.id)) {
                missingImplementations.push({
                    id: resource.id,
                    name: resource.name,
                    category: category as ResourceCategory,
                });
            }
        }
    }
    
    // Process registered resources
    const unavailableServices: ResourceSystemHealthCheck["unavailableServices"] = [];
    const resourceHealthInfos: ResourceHealthInfo[] = [];
    
    for (const [id, info] of Array.from(registeredResources)) {
        const isEnabled = enabledResourceIds.has(id);
        const isActive = info.status === DiscoveryStatus.Available && 
                        info.health === ResourceHealth.Healthy;
        
        // Update stats
        categoryStats[info.category].registered++;
        if (isEnabled) {
            stats.totalEnabled++;
            categoryStats[info.category].enabled++;
        }
        if (isActive) {
            stats.totalActive++;
            categoryStats[info.category].active++;
        }
        
        // Check for unavailable but enabled services
        if (isEnabled && !isActive) {
            const supportedResource = Object.values(SUPPORTED_RESOURCES)
                .flat()
                .find(r => r.id === id);
                
            unavailableServices.push({
                id,
                name: supportedResource?.name || info.displayName,
                category: info.category,
                reason: info.health === "unhealthy" 
                    ? "Health check failed" 
                    : "Service not found",
            });
        }
        
        // Build health info
        resourceHealthInfos.push({
            id,
            name: info.displayName,
            category: info.category,
            enabled: isEnabled,
            registered: true,
            available: isActive,
            health: info.health,
            lastCheck: info.lastHealthCheck,
        });
    }
    
    // Add unregistered but supported resources to the health info
    for (const missing of missingImplementations) {
        resourceHealthInfos.push({
            id: missing.id,
            name: missing.name,
            category: missing.category,
            enabled: enabledResourceIds.has(missing.id),
            registered: false,
            available: false,
            error: "Not implemented",
        });
    }
    
    // Calculate overall health
    const { status, message } = calculateSystemHealth(
        stats.totalEnabled,
        stats.totalActive,
        unavailableServices,
    );
    
    return {
        status,
        message,
        timestamp,
        stats,
        missingImplementations,
        unavailableServices,
        categories: categoryStats as Record<ResourceCategory, {
            supported: number;
            registered: number;
            enabled: number;
            active: number;
        }>,
        resources: resourceHealthInfos.sort((a, b) => {
            // Sort by category, then by ID
            if (a.category !== b.category) {
                return a.category.localeCompare(b.category);
            }
            return a.id.localeCompare(b.id);
        }),
    };
}

