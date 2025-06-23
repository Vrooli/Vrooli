import { type PrismaClient } from "@prisma/client";
import { type DatabaseFixtureFactory } from "./DatabaseFixtureFactory.js";

/**
 * Registry for managing all database fixture factories
 * Provides centralized access and coordination between factories
 */
export class FixtureFactoryRegistry {
    private static instance: FixtureFactoryRegistry;
    private factories: Map<string, DatabaseFixtureFactory<any, any, any>> = new Map();
    private prisma: PrismaClient;

    private constructor(prisma: PrismaClient) {
        this.prisma = prisma;
    }

    /**
     * Get singleton instance
     */
    static getInstance(prisma: PrismaClient): FixtureFactoryRegistry {
        if (!this.instance) {
            this.instance = new FixtureFactoryRegistry(prisma);
        }
        return this.instance;
    }

    /**
     * Register a factory
     */
    register<T extends DatabaseFixtureFactory<any, any, any>>(
        modelName: string,
        factoryClass: new (prisma: PrismaClient) => T,
    ): T {
        if (!this.factories.has(modelName)) {
            const factory = new factoryClass(this.prisma);
            this.factories.set(modelName, factory);
        }
        return this.factories.get(modelName) as T;
    }

    /**
     * Get a registered factory
     */
    get<T extends DatabaseFixtureFactory<any, any, any>>(modelName: string): T | undefined {
        return this.factories.get(modelName) as T;
    }

    /**
     * Get all registered factories
     */
    getAll(): Map<string, DatabaseFixtureFactory<any, any, any>> {
        return new Map(this.factories);
    }

    /**
     * Clean up all factories
     */
    async cleanupAll(): Promise<void> {
        const cleanupPromises = Array.from(this.factories.values()).map(factory => 
            factory.cleanupAll(),
        );
        await Promise.all(cleanupPromises);
    }

    /**
     * Clear all registrations
     */
    clear(): void {
        this.factories.clear();
    }

    /**
     * Create a cross-model test scenario
     */
    async createScenario(scenarioName: string, setup: (registry: this) => Promise<void>): Promise<void> {
        await this.prisma.$transaction(async (tx) => {
            // Create a temporary registry with transaction-bound factories
            const txRegistry = new FixtureFactoryRegistry(tx as any);
            
            // Copy factory registrations
            for (const [name, factory] of this.factories) {
                const FactoryClass = factory.constructor as any;
                txRegistry.register(name, FactoryClass);
            }
            
            // Execute scenario setup
            await setup(txRegistry as any);
        });
    }

    /**
     * Bulk seed multiple models
     */
    async bulkSeed(config: Record<string, { count: number; template?: any }>): Promise<Record<string, any[]>> {
        const results: Record<string, any[]> = {};
        
        for (const [modelName, seedConfig] of Object.entries(config)) {
            const factory = this.get(modelName);
            if (factory) {
                results[modelName] = await factory.seedMultiple(seedConfig.count, seedConfig.template);
            }
        }
        
        return results;
    }
}

/**
 * Utility functions for working with fixture factories
 */
export const fixtureUtils = {
    /**
     * Create interconnected test data
     */
    async createInterconnectedData(
        registry: FixtureFactoryRegistry,
        config: {
            users?: number;
            teams?: number;
            projects?: number;
            interconnect?: boolean;
        },
    ): Promise<{
        users: any[];
        teams: any[];
        projects: any[];
    }> {
        const result = {
            users: [] as any[],
            teams: [] as any[],
            projects: [] as any[],
        };

        // Create users
        if (config.users) {
            const userFactory = registry.get("User");
            if (userFactory) {
                result.users = await userFactory.seedMultiple(config.users);
            }
        }

        // Create teams
        if (config.teams) {
            const teamFactory = registry.get("Team");
            if (teamFactory) {
                result.teams = await teamFactory.seedMultiple(config.teams);
            }
        }

        // Create projects
        if (config.projects) {
            const projectFactory = registry.get("Project");
            if (projectFactory) {
                result.projects = await projectFactory.seedMultiple(config.projects);
            }
        }

        // Interconnect if requested
        if (config.interconnect) {
            await interconnectData(registry, result);
        }

        return result;
    },

    /**
     * Verify data integrity across models
     */
    async verifyDataIntegrity(
        registry: FixtureFactoryRegistry,
        checks: Array<{
            model: string;
            id: string;
            expectations: Record<string, any>;
        }>,
    ): Promise<{
        valid: boolean;
        errors: string[];
    }> {
        const errors: string[] = [];

        for (const check of checks) {
            const factory = registry.get(check.model);
            if (!factory) {
                errors.push(`Factory for model ${check.model} not found`);
                continue;
            }

            try {
                await factory.verifyState(check.id, check.expectations);
            } catch (error) {
                errors.push(error instanceof Error ? error.message : "Unknown error");
            }
        }

        return {
            valid: errors.length === 0,
            errors,
        };
    },

    /**
     * Clean up test data in correct order
     */
    async cleanupInOrder(
        registry: FixtureFactoryRegistry,
        order: string[],
    ): Promise<void> {
        for (const modelName of order) {
            const factory = registry.get(modelName);
            if (factory) {
                await factory.cleanupAll();
            }
        }
    },
};

/**
 * Helper to interconnect test data
 */
async function interconnectData(
    registry: FixtureFactoryRegistry,
    data: {
        users: any[];
        teams: any[];
        projects: any[];
    },
): Promise<void> {
    // Add users to teams
    if (data.users.length > 0 && data.teams.length > 0) {
        const memberFactory = registry.get("Member");
        if (memberFactory) {
            for (let i = 0; i < data.users.length; i++) {
                const teamIndex = i % data.teams.length;
                await memberFactory.createMinimal({
                    userId: data.users[i].id,
                    teamId: data.teams[teamIndex].id,
                    role: i === 0 ? "Owner" : "Member",
                });
            }
        }
    }

    // Assign projects to teams
    if (data.projects.length > 0 && data.teams.length > 0) {
        const projectFactory = registry.get("Project");
        if (projectFactory) {
            for (let i = 0; i < data.projects.length; i++) {
                const teamIndex = i % data.teams.length;
                await projectFactory.connectExisting(
                    data.projects[i].id,
                    { team: { id: data.teams[teamIndex].id } },
                );
            }
        }
    }
}
