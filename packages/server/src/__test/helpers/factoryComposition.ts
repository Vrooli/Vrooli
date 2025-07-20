/**
 * Factory composition utilities for coordinating multiple factories
 * 
 * These utilities make it easier to create complex test scenarios that involve
 * multiple related models while ensuring proper cleanup coordination.
 */

import { type PrismaClient } from "@prisma/client";
import { type EnhancedDatabaseFactory } from "../fixtures/db/EnhancedDatabaseFactory.js";
import { type DatabaseFactoryRegistry, getFactoryRegistry } from "../fixtures/db/DatabaseFactoryRegistry.js";
import { TestRecordTracker } from "./testRecordTracker.js";
import { cleanupGroups, type CleanupGroup } from "./testCleanupHelpers.js";

/**
 * Configuration for a factory composition scenario
 */
export interface FactoryScenarioConfig {
    /** Which cleanup scope to use when the scenario is cleaned up */
    cleanupScope?: CleanupGroup;
    /** Whether to use global record tracking */
    useGlobalTracking?: boolean;
    /** Custom relationships to set up between created records */
    relationships?: Record<string, any>;
    /** Transaction settings */
    useTransaction?: boolean;
}

/**
 * A managed group of factories with coordinated lifecycle
 */
export class FactoryComposition {
    private registry: DatabaseFactoryRegistry;
    private config: FactoryScenarioConfig;
    private isActive = false;
    private createdData: Record<string, any> = {};

    constructor(
        private prisma: PrismaClient,
        config: FactoryScenarioConfig = {},
    ) {
        this.registry = getFactoryRegistry(prisma);
        this.config = {
            cleanupScope: "full",
            useGlobalTracking: true,
            useTransaction: false,
            ...config,
        };
    }

    /**
     * Start the composition session
     */
    start(): void {
        if (this.isActive) {
            throw new Error("FactoryComposition is already active");
        }
        
        this.isActive = true;
        this.createdData = {};
        
        if (this.config.useGlobalTracking) {
            TestRecordTracker.start();
        }
    }

    /**
     * Register a factory for use in this composition
     */
    useFactory<T extends EnhancedDatabaseFactory<any, any>>(
        modelName: string,
        factoryClass: new (modelName: string, prisma: PrismaClient) => T,
    ): T {
        if (!this.isActive) {
            throw new Error("FactoryComposition must be started before using factories");
        }
        
        return this.registry.register(modelName, factoryClass);
    }

    /**
     * Create a record using a registered factory
     */
    async create<T>(
        modelName: string,
        type: "minimal" | "complete" | "withRelations" = "minimal",
        data?: any,
    ): Promise<T> {
        if (!this.isActive) {
            throw new Error("FactoryComposition must be started before creating records");
        }
        
        const factory = this.registry.get(modelName);
        let result: T;
        
        switch (type) {
            case "minimal":
                result = await factory.createMinimal(data);
                break;
            case "complete":
                result = await factory.createComplete(data);
                break;
            case "withRelations":
                result = await factory.createWithRelations(data);
                break;
        }
        
        // Store the created data for relationship setup
        if (!this.createdData[modelName]) {
            this.createdData[modelName] = [];
        }
        this.createdData[modelName].push(result);
        
        return result;
    }

    /**
     * Create multiple records of the same type
     */
    async createMany<T>(
        modelName: string,
        count: number,
        type: "minimal" | "complete" = "minimal",
        template?: any,
    ): Promise<T[]> {
        const results: T[] = [];
        
        for (let i = 0; i < count; i++) {
            const result = await this.create<T>(modelName, type, template);
            results.push(result);
        }
        
        return results;
    }

    /**
     * Create a coordinated set of related records
     */
    async createRelatedSet(config: {
        user?: { type?: "minimal" | "complete"; data?: any };
        team?: { type?: "minimal" | "complete"; data?: any; users?: string[] };
        chat?: { type?: "minimal" | "complete"; data?: any; participants?: string[] };
        resources?: { count?: number; type?: "minimal" | "complete"; data?: any };
    }): Promise<{
        user?: any;
        team?: any;
        chat?: any;
        resources?: any[];
    }> {
        const result: any = {};
        
        // Create user first (most other entities depend on users)
        if (config.user) {
            result.user = await this.create("User", config.user.type || "minimal", config.user.data);
        }
        
        // Create team and add user as member
        if (config.team) {
            result.team = await this.create("Team", config.team.type || "minimal", {
                ...config.team.data,
                createdById: result.user?.id,
            });
            
            // Add users as team members if specified
            if (config.team.users && result.team?.id) {
                // This would require a Member factory to be registered
                for (const userId of config.team.users) {
                    await this.create("Member", "minimal", {
                        teamId: result.team.id,
                        userId: userId === "self" ? result.user?.id : userId,
                    });
                }
            }
        }
        
        // Create chat and add participants
        if (config.chat) {
            result.chat = await this.create("Chat", config.chat.type || "minimal", {
                ...config.chat.data,
                creatorId: result.user?.id,
                teamId: result.team?.id,
            });
            
            // Add participants if specified
            if (config.chat.participants && result.chat?.id) {
                for (const participantId of config.chat.participants) {
                    await this.create("ChatParticipant", "minimal", {
                        chatId: result.chat.id,
                        userId: participantId === "self" ? result.user?.id : participantId,
                    });
                }
            }
        }
        
        // Create resources
        if (config.resources) {
            result.resources = await this.createMany(
                "Resource",
                config.resources.count || 1,
                config.resources.type || "minimal",
                {
                    ...config.resources.data,
                    createdById: result.user?.id,
                    ownedByUserId: result.user?.id,
                    ownedByTeamId: result.team?.id,
                },
            );
        }
        
        return result;
    }

    /**
     * Get all data created in this composition
     */
    getCreatedData(): Record<string, any> {
        return { ...this.createdData };
    }

    /**
     * Get tracking summary for this composition
     */
    getTrackingSummary() {
        return this.registry.getTrackingSummary();
    }

    /**
     * Clean up all records created in this composition
     */
    async cleanup(): Promise<void> {
        if (!this.isActive) {
            return;
        }
        
        try {
            if (this.config.cleanupScope && this.config.cleanupScope !== "full") {
                // Use focused cleanup for better performance
                await this.registry.smartCleanup(this.config.cleanupScope);
            } else {
                // Use comprehensive cleanup
                await this.registry.endTrackingSession();
            }
        } finally {
            this.isActive = false;
            this.createdData = {};
        }
    }

    /**
     * Execute a function within a managed composition context
     */
    static async withComposition<T>(
        prisma: PrismaClient,
        config: FactoryScenarioConfig,
        fn: (composition: FactoryComposition) => Promise<T>,
    ): Promise<T> {
        const composition = new FactoryComposition(prisma, config);
        
        try {
            composition.start();
            return await fn(composition);
        } finally {
            await composition.cleanup();
        }
    }

    /**
     * Execute a function within a transaction-based composition
     */
    static async withTransactionComposition<T>(
        prisma: PrismaClient,
        config: Omit<FactoryScenarioConfig, "useTransaction">,
        fn: (composition: FactoryComposition, tx: PrismaClient) => Promise<T>,
    ): Promise<T> {
        return await prisma.$transaction(async (tx) => {
            const composition = new FactoryComposition(tx, {
                ...config,
                useTransaction: true,
            });
            
            try {
                composition.start();
                return await fn(composition, tx);
            } finally {
                // Transaction will auto-rollback, so just clean tracking
                composition.config.useGlobalTracking && TestRecordTracker.stop();
            }
        });
    }
}

/**
 * Quick utility functions for common scenarios
 */
export const factoryScenarios = {
    /**
     * Create a basic user with authentication
     */
    async userWithAuth(prisma: PrismaClient): Promise<{ user: any; cleanup: () => Promise<void> }> {
        const composition = new FactoryComposition(prisma, {
            cleanupScope: "userAuth",
            useGlobalTracking: true,
        });
        
        composition.start();
        const user = await composition.create("User", "withRelations", {
            withAuth: true,
        });
        
        return {
            user,
            cleanup: () => composition.cleanup(),
        };
    },

    /**
     * Create a team with members
     */
    async teamWithMembers(
        prisma: PrismaClient,
        memberCount = 2,
    ): Promise<{ team: any; members: any[]; users: any[]; cleanup: () => Promise<void> }> {
        const composition = new FactoryComposition(prisma, {
            cleanupScope: "team",
            useGlobalTracking: true,
        });
        
        composition.start();
        
        // Create users first
        const users = await composition.createMany("User", memberCount + 1, "minimal");
        
        // Create team with first user as creator
        const team = await composition.create("Team", "minimal", {
            createdById: users[0].id,
        });
        
        // Create memberships
        const members = [];
        for (const user of users) {
            const member = await composition.create("Member", "minimal", {
                teamId: team.id,
                userId: user.id,
                isAdmin: user.id === users[0].id, // First user is admin
            });
            members.push(member);
        }
        
        return {
            team,
            members,
            users,
            cleanup: () => composition.cleanup(),
        };
    },

    /**
     * Create a chat with participants
     */
    async chatWithParticipants(
        prisma: PrismaClient,
        participantCount = 3,
    ): Promise<{ chat: any; participants: any[]; users: any[]; cleanup: () => Promise<void> }> {
        const composition = new FactoryComposition(prisma, {
            cleanupScope: "chat",
            useGlobalTracking: true,
        });
        
        composition.start();
        
        // Create users
        const users = await composition.createMany("User", participantCount, "minimal");
        
        // Create chat
        const chat = await composition.create("Chat", "minimal", {
            creatorId: users[0].id,
        });
        
        // Add participants
        const participants = [];
        for (const user of users) {
            const participant = await composition.create("ChatParticipant", "minimal", {
                chatId: chat.id,
                userId: user.id,
            });
            participants.push(participant);
        }
        
        return {
            chat,
            participants,
            users,
            cleanup: () => composition.cleanup(),
        };
    },
} as const;
