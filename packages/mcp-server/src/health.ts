/**
 * Health check response interface
 */
export interface HealthStatus {
    status: 'ok' | 'degraded' | 'error';
    activeConnections?: number;
    uptime?: number;
    version?: string;
    [key: string]: any;
}

/**
 * Base health check function that only returns status
 * @returns Basic health status
 */
export function getBaseHealthStatus(): HealthStatus {
    return {
        status: 'ok',
        uptime: process.uptime(),
    };
}

/**
 * Enhanced health check for SSE mode
 * @param connectionCount Number of active SSE connections
 * @param version Server version
 * @returns Health status with SSE-specific metrics
 */
export function getSseHealthStatus(connectionCount: number, version: string): HealthStatus {
    return {
        ...getBaseHealthStatus(),
        activeConnections: connectionCount,
        version
    };
}

/**
 * Health check for STDIO mode
 * @param version Server version
 * @returns Health status with STDIO-specific metrics
 */
export function getStdioHealthStatus(version: string): HealthStatus {
    return {
        ...getBaseHealthStatus(),
        mode: 'stdio',
        version
    };
} 