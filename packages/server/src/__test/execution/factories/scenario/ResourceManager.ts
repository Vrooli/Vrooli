/**
 * Resource Manager
 * 
 * Manages resources for scenario execution with proper cleanup and monitoring
 */

import { ExecutionTestError } from "../../types.js";

export interface ResourceLimits {
    maxMemoryMB: number;
    maxDurationMs: number;
    maxEventSubscriptions: number;
    maxConcurrentOperations: number;
}

export interface ResourceUsage {
    memoryUsedMB: number;
    durationMs: number;
    eventSubscriptions: number;
    concurrentOperations: number;
    startTime: Date;
    endTime?: Date;
}

export interface ManagedResource {
    id: string;
    type: "subscription" | "operation" | "timer" | "memory";
    cleanup: () => Promise<void>;
    metadata?: Record<string, unknown>;
}

export class ResourceManager {
    private resources = new Map<string, ManagedResource>();
    private limits: ResourceLimits;
    private usage: ResourceUsage;
    private isDestroyed = false;

    constructor(limits: ResourceLimits) {
        this.limits = limits;
        this.usage = {
            memoryUsedMB: 0,
            durationMs: 0,
            eventSubscriptions: 0,
            concurrentOperations: 0,
            startTime: new Date(),
        };
    }

    async addResource(resource: ManagedResource): Promise<void> {
        if (this.isDestroyed) {
            throw new ExecutionTestError("ResourceManager is destroyed", "RESOURCE_MANAGER_DESTROYED");
        }

        // Check resource limits
        if (resource.type === "subscription" && this.usage.eventSubscriptions >= this.limits.maxEventSubscriptions) {
            throw new ExecutionTestError(
                `Event subscription limit exceeded: ${this.limits.maxEventSubscriptions}`,
                "RESOURCE_LIMIT_EXCEEDED",
            );
        }

        if (resource.type === "operation" && this.usage.concurrentOperations >= this.limits.maxConcurrentOperations) {
            throw new ExecutionTestError(
                `Concurrent operation limit exceeded: ${this.limits.maxConcurrentOperations}`,
                "RESOURCE_LIMIT_EXCEEDED",
            );
        }

        // Add resource and update usage
        this.resources.set(resource.id, resource);
        this.updateUsage(resource.type, 1);
    }

    async removeResource(resourceId: string): Promise<void> {
        const resource = this.resources.get(resourceId);
        if (!resource) return;

        try {
            await resource.cleanup();
        } catch (error) {
            console.error(`Error cleaning up resource ${resourceId}:`, error);
        }

        this.resources.delete(resourceId);
        this.updateUsage(resource.type, -1);
    }

    async checkLimits(): Promise<void> {
        const currentTime = new Date();
        this.usage.durationMs = currentTime.getTime() - this.usage.startTime.getTime();

        if (this.usage.durationMs > this.limits.maxDurationMs) {
            throw new ExecutionTestError(
                `Duration limit exceeded: ${this.limits.maxDurationMs}ms`,
                "RESOURCE_LIMIT_EXCEEDED",
            );
        }

        // Check memory usage (simplified)
        const memoryUsage = process.memoryUsage();
        const memoryUsedMB = memoryUsage.heapUsed / 1024 / 1024;
        this.usage.memoryUsedMB = memoryUsedMB;

        if (memoryUsedMB > this.limits.maxMemoryMB) {
            throw new ExecutionTestError(
                `Memory limit exceeded: ${this.limits.maxMemoryMB}MB`,
                "RESOURCE_LIMIT_EXCEEDED",
            );
        }
    }

    getCurrentUsage(): ResourceUsage {
        const currentTime = new Date();
        return {
            ...this.usage,
            durationMs: currentTime.getTime() - this.usage.startTime.getTime(),
        };
    }

    async destroy(): Promise<void> {
        if (this.isDestroyed) return;

        this.isDestroyed = true;
        this.usage.endTime = new Date();

        // Clean up all resources
        const cleanupPromises = Array.from(this.resources.values()).map(async (resource) => {
            try {
                await resource.cleanup();
            } catch (error) {
                console.error(`Error cleaning up resource ${resource.id}:`, error);
            }
        });

        await Promise.allSettled(cleanupPromises);
        this.resources.clear();
    }

    private updateUsage(type: string, delta: number): void {
        switch (type) {
            case "subscription":
                this.usage.eventSubscriptions += delta;
                break;
            case "operation":
                this.usage.concurrentOperations += delta;
                break;
        }
    }
}
