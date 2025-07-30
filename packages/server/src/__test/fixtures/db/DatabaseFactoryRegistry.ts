// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-03 - Fixed type safety issues: improved generic type constraints
import { type PrismaClient } from "@prisma/client";
import { cleanupGroups } from "../../helpers/testCleanupHelpers.js";
import { TestRecordTracker } from "../../helpers/testRecordTracker.js";
import { type EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";

/**
 * Registry for managing database fixture factories
 * Provides centralized access to all model factories and utilities
 * for cross-model operations and cleanup
 */
export class DatabaseFactoryRegistry {
    private factories = new Map<string, EnhancedDatabaseFactory<any, any>>();
    private prisma: PrismaClient;

    constructor(prisma: PrismaClient) {
        this.prisma = prisma;
    }

    /**
     * Register a factory for a model
     */
    register<T extends EnhancedDatabaseFactory<any, any>>(
        modelName: string,
        factoryClass: new (modelName: string, prisma: PrismaClient) => T,
    ): T {
        const factory = new factoryClass(modelName, this.prisma);
        this.factories.set(modelName, factory);
        return factory;
    }

    /**
     * Get a factory by model name
     */
    get<T extends EnhancedDatabaseFactory<any, any>>(modelName: string): T {
        const factory = this.factories.get(modelName);
        if (!factory) {
            throw new Error(`No factory registered for model: ${modelName}`);
        }
        return factory as T;
    }

    /**
     * Check if a factory is registered
     */
    has(modelName: string): boolean {
        return this.factories.has(modelName);
    }

    /**
     * Get all registered model names
     */
    getModelNames(): string[] {
        return Array.from(this.factories.keys());
    }

    /**
     * Clean up all records created by all factories
     * Now coordinates with global record tracker for comprehensive cleanup
     */
    async cleanupAll(): Promise<void> {
        // First clean up factory-tracked records
        const cleanupPromises = Array.from(this.factories.values()).map(
            factory => factory.cleanupAll(),
        );
        await Promise.all(cleanupPromises);

        // Then clean up any records tracked by the global tracker
        await TestRecordTracker.cleanup(this.prisma);
    }

    /**
     * Clear tracking for all factories without cleanup
     * Now also clears global record tracker
     */
    clearAllTracking(): void {
        this.factories.forEach(factory => factory.clearTracking());
        TestRecordTracker.clear();
    }

    /**
     * Get all created IDs across all factories
     */
    getAllCreatedIds(): Record<string, string[]> {
        const result: Record<string, string[]> = {};
        this.factories.forEach((factory, modelName) => {
            const ids = factory.getCreatedIds();
            if (ids.length > 0) {
                result[modelName] = ids;
            }
        });
        return result;
    }

    /**
     * Create a test environment with related models
     */
    async createTestEnvironment(config: {
        models: string[];
        relationships?: Record<string, any>;
    }): Promise<Record<string, any>> {
        const result: Record<string, any> = {};

        // Create base models
        for (const modelName of config.models) {
            const factory = this.get(modelName);
            result[modelName] = await factory.createMinimal();
        }

        // Setup relationships if specified
        if (config.relationships) {
            for (const [modelName, relationConfig] of Object.entries(config.relationships)) {
                const factory = this.get(modelName);
                const record = result[modelName];
                if (record?.id) {
                    await factory.setupRelationships(record.id, relationConfig);
                }
            }
        }

        return result;
    }

    /**
     * Verify database integrity across models
     */
    async verifyIntegrity(): Promise<{ valid: boolean; issues: string[] }> {
        const issues: string[] = [];

        for (const [modelName, factory] of this.factories) {
            const ids = factory.getCreatedIds();
            for (const id of ids) {
                try {
                    const validation = await factory.verifyConstraints(id);
                    if (!validation.valid) {
                        issues.push(`${modelName}#${id}: ${validation.violations.join(", ")}`);
                    }
                } catch (error) {
                    issues.push(`${modelName}#${id}: ${error instanceof Error ? error.message : "Unknown error"}`);
                }
            }
        }

        return {
            valid: issues.length === 0,
            issues,
        };
    }

    /**
     * Execute a transaction with all factories having transaction-aware delegates
     */
    async transaction<T>(
        callback: (tx: PrismaClient, factories: Map<string, EnhancedDatabaseFactory<any, any>>) => Promise<T>,
    ): Promise<T> {
        return await this.prisma.$transaction(async (tx) => {
            // Create transaction-aware factory map
            const txFactories = new Map<string, EnhancedDatabaseFactory<any, any>>();

            this.factories.forEach((factory, modelName) => {
                // Create a proxy that uses the transaction
                const txFactory = new Proxy(factory, {
                    get(target, prop) {
                        if (prop === "prisma") {
                            return tx;
                        }
                        return target[prop as keyof typeof target];
                    },
                });
                txFactories.set(modelName, txFactory as any);
            });

            return await callback(tx, txFactories);
        });
    }

    /**
     * Create complex multi-model scenarios
     */
    async createScenario(scenarioName: string, config: Record<string, any>): Promise<Record<string, any>> {
        return await this.transaction(async (tx, factories) => {
            const result: Record<string, any> = {};

            // Execute scenario steps in order
            for (const [step, stepConfig] of Object.entries(config)) {
                const [modelName, action] = step.split(".");
                const factory = factories.get(modelName);

                if (!factory) {
                    throw new Error(`Unknown model in scenario step: ${modelName}`);
                }

                switch (action) {
                    case "create":
                        result[step] = await factory.createMinimal(stepConfig);
                        break;
                    case "createComplete":
                        result[step] = await factory.createComplete(stepConfig);
                        break;
                    case "createWithRelations":
                        result[step] = await factory.createWithRelations(stepConfig);
                        break;
                    case "seedMultiple":
                        const count = stepConfig.count || 1;
                        const items = [];
                        for (let i = 0; i < count; i++) {
                            items.push(await factory.create(stepConfig.template));
                        }
                        result[step] = items;
                        break;
                    default:
                        throw new Error(`Unknown action in scenario step: ${action}`);
                }
            }

            return result;
        });
    }

    /**
     * Smart cleanup using dependency-ordered cleanup helpers
     * This is more efficient than factory-by-factory cleanup
     */
    async smartCleanup(scope: "full" | "userAuth" | "chat" | "team" | "execution" | "financial" | "content" | "minimal" = "full"): Promise<void> {
        // Use the dependency-ordered cleanup helpers for more efficient cleanup
        await cleanupGroups[scope](this.prisma);

        // Clear all factory tracking since records are now cleaned up
        this.clearAllTracking();
    }

    /**
     * Get comprehensive tracking summary across all systems
     */
    getTrackingSummary(): {
        factoryTracked: Record<string, string[]>;
        globalTracked: Record<string, number>;
        totalFactoryRecords: number;
        totalGlobalRecords: number;
    } {
        const factoryTracked = this.getAllCreatedIds();
        const globalTracked = TestRecordTracker.getSummary();

        const totalFactoryRecords = Object.values(factoryTracked).reduce(
            (total, ids) => total + ids.length,
            0,
        );
        const totalGlobalRecords = TestRecordTracker.totalTrackedRecords;

        return {
            factoryTracked,
            globalTracked,
            totalFactoryRecords,
            totalGlobalRecords,
        };
    }

    /**
     * Coordinate factory operations with global record tracking
     * Starts tracking session and ensures all created records are tracked
     */
    startTrackingSession(): void {
        TestRecordTracker.start();
    }

    /**
     * End tracking session with comprehensive cleanup
     */
    async endTrackingSession(): Promise<void> {
        await this.cleanupAll();
        TestRecordTracker.stop();
    }

    /**
     * Create a test environment with smart cleanup integration
     */
    async createManagedTestEnvironment(config: {
        models: string[];
        relationships?: Record<string, any>;
        trackingScope?: "factory" | "global" | "both";
    }): Promise<{
        data: Record<string, any>;
        cleanup: () => Promise<void>;
    }> {
        const { trackingScope = "both" } = config;

        // Start tracking if requested
        if (trackingScope === "global" || trackingScope === "both") {
            this.startTrackingSession();
        }

        // Create the environment
        const data = await this.createTestEnvironment(config);

        // Return with custom cleanup function
        return {
            data,
            cleanup: async () => {
                if (trackingScope === "factory") {
                    // Only clean up factory-tracked records
                    const cleanupPromises = Array.from(this.factories.values()).map(
                        factory => factory.cleanupAll(),
                    );
                    await Promise.all(cleanupPromises);
                } else if (trackingScope === "global") {
                    // Only clean up globally-tracked records
                    await TestRecordTracker.cleanup(this.prisma);
                    TestRecordTracker.stop();
                } else {
                    // Clean up both (default)
                    await this.endTrackingSession();
                }
            },
        };
    }
}

// Singleton instance
let registryInstance: DatabaseFactoryRegistry | null = null;

/**
 * Get or create the database factory registry
 */
export function getFactoryRegistry(prisma: PrismaClient): DatabaseFactoryRegistry {
    if (!registryInstance) {
        registryInstance = new DatabaseFactoryRegistry(prisma);
    }
    return registryInstance;
}

/**
 * Reset the registry (useful for tests)
 */
export function resetFactoryRegistry(): void {
    if (registryInstance) {
        registryInstance.clearAllTracking();
        registryInstance = null;
    }
}
