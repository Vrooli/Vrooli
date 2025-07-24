/**
 * Resource Dependency Management System
 * Ensures resources are initialized in the correct order based on their dependencies
 */

import type { ResourceId } from "./typeRegistry.js";
import { logger } from "../../events/logger.js";

export interface ResourceDependency {
    /** The resource that has dependencies */
    resourceId: ResourceId;
    /** Resources that must be initialized before this resource */
    dependsOn: ResourceId[];
    /** Priority for initialization order (higher numbers initialize first) */
    priority: number;
    /** Whether this dependency is optional (resource can start without it) */
    optional?: boolean;
}

export interface ResourceInitializationPlan {
    /** Resources grouped by initialization phase */
    phases: ResourceId[][];
    /** Resources that have circular dependencies (need manual resolution) */
    circularDependencies: ResourceId[];
    /** Resources with missing dependencies */
    missingDependencies: Array<{
        resourceId: ResourceId;
        missingDeps: ResourceId[];
    }>;
}

/**
 * Manages resource dependencies and creates initialization plans
 */
export class ResourceDependencyManager {
    private dependencies = new Map<ResourceId, ResourceDependency>();
    private registeredResources = new Set<ResourceId>();

    /**
     * Register a resource with its dependencies
     */
    registerResourceDependency(dependency: ResourceDependency): void {
        // Validate that the resource doesn't depend on itself
        if (dependency.dependsOn.includes(dependency.resourceId)) {
            throw new Error(`Resource ${dependency.resourceId} cannot depend on itself`);
        }

        this.dependencies.set(dependency.resourceId, dependency);
        this.registeredResources.add(dependency.resourceId);
        
        logger.debug(`[DependencyManager] Registered dependency for ${dependency.resourceId}`, {
            dependsOn: dependency.dependsOn,
            priority: dependency.priority,
            optional: dependency.optional,
        });
    }

    /**
     * Register a resource without dependencies
     */
    registerResource(resourceId: ResourceId, priority = 0): void {
        if (!this.dependencies.has(resourceId)) {
            this.registerResourceDependency({
                resourceId,
                dependsOn: [],
                priority,
            });
        }
        this.registeredResources.add(resourceId);
    }

    /**
     * Create an initialization plan for all registered resources
     */
    createInitializationPlan(enabledResources: Set<ResourceId>): ResourceInitializationPlan {
        // Filter to only include enabled resources
        const enabledDependencies = new Map<ResourceId, ResourceDependency>();
        for (const [resourceId, dependency] of this.dependencies) {
            if (enabledResources.has(resourceId)) {
                enabledDependencies.set(resourceId, dependency);
            }
        }

        // Find circular dependencies
        const circularDependencies = this.findCircularDependencies(enabledDependencies);

        // Find missing dependencies
        const missingDependencies: Array<{ resourceId: ResourceId; missingDeps: ResourceId[] }> = [];
        for (const [resourceId, dependency] of enabledDependencies) {
            const missingDeps = dependency.dependsOn.filter(dep => 
                !enabledResources.has(dep) && !dependency.optional,
            );
            if (missingDeps.length > 0) {
                missingDependencies.push({ resourceId, missingDeps });
            }
        }

        // Create initialization phases using topological sort
        const phases = this.createInitializationPhases(enabledDependencies, circularDependencies);

        return {
            phases,
            circularDependencies,
            missingDependencies,
        };
    }

    /**
     * Find circular dependencies using depth-first search
     */
    private findCircularDependencies(dependencies: Map<ResourceId, ResourceDependency>): ResourceId[] {
        const visited = new Set<ResourceId>();
        const visiting = new Set<ResourceId>();
        const circularDeps = new Set<ResourceId>();

        const visit = (resourceId: ResourceId): void => {
            if (visiting.has(resourceId)) {
                // Found a cycle
                circularDeps.add(resourceId);
                return;
            }

            if (visited.has(resourceId)) {
                return;
            }

            visiting.add(resourceId);
            const dependency = dependencies.get(resourceId);
            
            if (dependency) {
                for (const dep of dependency.dependsOn) {
                    if (dependencies.has(dep)) {
                        visit(dep);
                    }
                }
            }

            visiting.delete(resourceId);
            visited.add(resourceId);
        };

        for (const resourceId of dependencies.keys()) {
            if (!visited.has(resourceId)) {
                visit(resourceId);
            }
        }

        return Array.from(circularDeps);
    }

    /**
     * Create initialization phases using topological sort
     */
    private createInitializationPhases(
        dependencies: Map<ResourceId, ResourceDependency>,
        circularDeps: ResourceId[],
    ): ResourceId[][] {
        const phases: ResourceId[][] = [];
        const remaining = new Set(dependencies.keys());
        const initialized = new Set<ResourceId>();

        // Remove circular dependencies from consideration
        for (const circularDep of circularDeps) {
            remaining.delete(circularDep);
        }

        let phaseNumber = 0;
        const maxPhases = remaining.size + 1; // Prevent infinite loops

        while (remaining.size > 0 && phaseNumber < maxPhases) {
            const currentPhase: ResourceId[] = [];

            // Find resources that can be initialized in this phase
            for (const resourceId of remaining) {
                const dependency = dependencies.get(resourceId)!;
                const canInitialize = dependency.dependsOn.every(dep => 
                    initialized.has(dep) || 
                    dependency.optional ||
                    circularDeps.includes(dep),
                );

                if (canInitialize) {
                    currentPhase.push(resourceId);
                }
            }

            if (currentPhase.length === 0) {
                // No progress made - there might be unresolved dependencies
                logger.warn("[DependencyManager] Unable to resolve all dependencies", {
                    remaining: Array.from(remaining),
                    phase: phaseNumber,
                });
                // Add remaining resources to final phase
                currentPhase.push(...remaining);
            }

            // Sort current phase by priority (higher priority first)
            currentPhase.sort((a, b) => {
                const priorityA = dependencies.get(a)?.priority || 0;
                const priorityB = dependencies.get(b)?.priority || 0;
                return priorityB - priorityA;
            });

            phases.push(currentPhase);

            // Mark resources as initialized and remove from remaining
            for (const resourceId of currentPhase) {
                initialized.add(resourceId);
                remaining.delete(resourceId);
            }

            phaseNumber++;
        }

        logger.info(`[DependencyManager] Created initialization plan with ${phases.length} phases`);
        return phases;
    }

    /**
     * Get dependency information for a specific resource
     */
    getDependencyInfo(resourceId: ResourceId): ResourceDependency | undefined {
        return this.dependencies.get(resourceId);
    }

    /**
     * Check if a resource can be safely shut down (no other resources depend on it)
     */
    canSafelyShutdown(resourceId: ResourceId, currentlyRunning: Set<ResourceId>): boolean {
        for (const [otherId, dependency] of this.dependencies) {
            if (otherId !== resourceId && 
                currentlyRunning.has(otherId) && 
                dependency.dependsOn.includes(resourceId) &&
                !dependency.optional) {
                return false;
            }
        }
        return true;
    }

    /**
     * Get resources that depend on a given resource
     */
    getDependentResources(resourceId: ResourceId): ResourceId[] {
        const dependents: ResourceId[] = [];
        for (const [otherId, dependency] of this.dependencies) {
            if (dependency.dependsOn.includes(resourceId)) {
                dependents.push(otherId);
            }
        }
        return dependents;
    }

    /**
     * Clear all dependencies (for testing)
     */
    clear(): void {
        this.dependencies.clear();
        this.registeredResources.clear();
    }

    /**
     * Get summary of dependency management state
     */
    getSummary(): {
        totalResources: number;
        resourcesWithDependencies: number;
        averageDependencies: number;
        maxDependencies: number;
    } {
        const dependencyCounts = Array.from(this.dependencies.values())
            .map(dep => dep.dependsOn.length);

        return {
            totalResources: this.dependencies.size,
            resourcesWithDependencies: dependencyCounts.filter(count => count > 0).length,
            averageDependencies: dependencyCounts.length > 0 
                ? dependencyCounts.reduce((sum, count) => sum + count, 0) / dependencyCounts.length 
                : 0,
            maxDependencies: Math.max(0, ...dependencyCounts),
        };
    }
}

/**
 * Pre-defined dependency configurations for common resource types
 */
export const COMMON_DEPENDENCIES: Record<string, Partial<ResourceDependency>> = {
    // AI resources typically don't depend on others
    ai: {
        priority: 100,
        dependsOn: [],
    },
    
    // Automation resources might depend on AI
    automation: {
        priority: 50,
        dependsOn: [], // Can be configured per resource
    },
    
    // Agent resources often need storage
    agents: {
        priority: 30,
        dependsOn: [], // Can be configured per resource
    },
    
    // Storage resources should start early
    storage: {
        priority: 200,
        dependsOn: [],
    },
};
