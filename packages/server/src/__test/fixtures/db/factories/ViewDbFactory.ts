import { generatePK, generatePublicId, nanoid } from "../idHelpers.js";
import { type ViewFor } from "@vrooli/shared";
import { type view, type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";

interface ViewRelationConfig extends RelationConfig {
    isAnonymous?: boolean;
    sessionId?: string;
    objectType?: keyof typeof ViewFor | string;
    objectId?: string;
    viewedAt?: Date;
}

/**
 * Database fixture factory for View model
 * Handles both authenticated and anonymous view tracking for analytics
 */
export class ViewDbFactory extends DatabaseFixtureFactory<
    view,
    Prisma.viewCreateInput,
    Prisma.viewInclude,
    Prisma.viewUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super("view", prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.view;
    }

    protected getMinimalData(overrides?: Partial<Prisma.viewCreateInput>): Prisma.viewCreateInput {
        return {
            id: generatePK(),
            name: "Page View",
            lastViewedAt: new Date(),
            by: { connect: { id: generatePK() } }, // Will be overridden by relationship config
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.viewCreateInput>): Prisma.viewCreateInput {
        return {
            id: generatePK(),
            name: "Detailed View",
            lastViewedAt: new Date(),
            by: { connect: { id: generatePK() } }, // Will be overridden by relationship config
            ...overrides,
        };
    }

    /**
     * Create a view for a specific object
     */
    async createForObject(
        byId: string,
        objectType: keyof typeof ViewFor | string,
        objectId: string,
        overrides?: Partial<Prisma.viewCreateInput>,
    ): Promise<view> {
        const typeMapping: Record<string, string> = {
            Resource: "resource",
            Team: "team",
            User: "user",
            Issue: "issue",
        };

        const fieldName = typeMapping[objectType];
        if (!fieldName) {
            throw new Error(`Invalid view object type: ${objectType}`);
        }

        const data: Prisma.viewCreateInput = {
            ...this.getMinimalData(),
            name: `${objectType} View`,
            by: { connect: { id: BigInt(byId) } },
            [fieldName]: { connect: { id: BigInt(objectId) } },
            ...overrides,
        };

        const result = await this.prisma.view.create({ data });
        this.trackCreatedId(result.id);
        return result;
    }

    /**
     * Create an anonymous view (using a user ID to represent anonymous viewer)
     */
    async createAnonymous(
        anonymousUserId: string,
        objectType: keyof typeof ViewFor | string,
        objectId: string,
        overrides?: Partial<Prisma.viewCreateInput>,
    ): Promise<view> {
        const typeMapping: Record<string, string> = {
            Resource: "resource",
            Team: "team",
            User: "user",
            Issue: "issue",
        };

        const fieldName = typeMapping[objectType];
        if (!fieldName) {
            throw new Error(`Invalid view object type: ${objectType}`);
        }

        const data: Prisma.viewCreateInput = {
            id: generatePK(),
            name: `Anonymous ${objectType} View`,
            lastViewedAt: new Date(),
            by: { connect: { id: BigInt(anonymousUserId) } },
            [fieldName]: { connect: { id: BigInt(objectId) } },
            ...overrides,
        };

        const result = await this.prisma.view.create({ data });
        this.trackCreatedId(result.id);
        return result;
    }

    /**
     * Create multiple views for analytics
     */
    async createViewHistory(
        byId: string,
        objects: Array<{ type: keyof typeof ViewFor | string; id: string; viewedAt?: Date }>,
    ): Promise<view[]> {
        const views: view[] = [];
        
        for (let i = 0; i < objects.length; i++) {
            const obj = objects[i];
            const view = await this.createForObject(byId, obj.type, obj.id, {
                lastViewedAt: obj.viewedAt || new Date(Date.now() - (i * 60000)), // 1 minute intervals
            });
            views.push(view);
        }
        
        return views;
    }

    protected getDefaultInclude(): Prisma.viewInclude {
        return {
            by: {
                select: {
                    id: true,
                    publicId: true,
                    name: true,
                    handle: true,
                },
            },
            // Include polymorphic relationships
            issue: { select: { id: true, publicId: true } },
            resource: { select: { id: true, publicId: true } },
            team: { select: { id: true, publicId: true, handle: true } },
            user: { select: { id: true, publicId: true, name: true, handle: true } },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.viewCreateInput,
        config: ViewRelationConfig,
        tx: any,
    ): Promise<Prisma.viewCreateInput> {
        const data = { ...baseData };

        // Handle viewer (user)
        if (config.byId) {
            data.by = { connect: { id: BigInt(config.byId) } };
        }

        // Handle viewed object
        if (config.objectType && config.objectId) {
            const typeMapping: Record<string, string> = {
                Resource: "resource",
                Team: "team",
                User: "user",
                Issue: "issue",
            };

            const fieldName = typeMapping[config.objectType];
            if (fieldName) {
                data[fieldName] = { connect: { id: BigInt(config.objectId) } };
            }
        }

        // Handle timestamp
        if (config.viewedAt) {
            data.lastViewedAt = config.viewedAt;
        }

        return data;
    }

    /**
     * Create test scenarios
     */
    async createRecentActivity(userId: string, count = 5): Promise<view[]> {
        const views: view[] = [];
        const now = new Date();
        
        for (let i = 0; i < count; i++) {
            const viewedAt = new Date(now.getTime() - (i * 3600000)); // 1 hour intervals
            const objectTypes = ["Resource", "Team", "User"];
            const objectType = objectTypes[i % objectTypes.length];
            
            const view = await this.createForObject(
                userId,
                objectType,
                generatePK(),
                {
                    name: `Recent ${objectType} Activity ${i + 1}`,
                    lastViewedAt: viewedAt,
                },
            );
            views.push(view);
        }
        
        return views;
    }

    async createPopularityMetrics(
        objectId: string,
        objectType: keyof typeof ViewFor,
        options: {
            userIds: string[];
            sessionIds: string[];
            daysBack: number;
            viewsPerDay: number;
        },
    ): Promise<view[]> {
        const views: view[] = [];
        const now = new Date();
        const { userIds, sessionIds, daysBack, viewsPerDay } = options;
        
        for (let day = 0; day < daysBack; day++) {
            for (let viewNum = 0; viewNum < viewsPerDay; viewNum++) {
                const viewedAt = new Date(
                    now.getTime() - (day * 24 * 60 * 60 * 1000) - (viewNum * 60 * 60 * 1000),
                );
                
                // Mix authenticated and anonymous views
                const useAnonymous = sessionIds.length > 0 && Math.random() > 0.7;
                
                if (useAnonymous) {
                    const anonymousUserId = sessionIds[viewNum % sessionIds.length];
                    const view = await this.createAnonymous(
                        anonymousUserId,
                        objectType,
                        objectId,
                        {
                            lastViewedAt: viewedAt,
                            name: `Analytics View Day ${day + 1} #${viewNum + 1} (Anonymous)`,
                        },
                    );
                    views.push(view);
                } else if (userIds.length > 0) {
                    const userId = userIds[viewNum % userIds.length];
                    const view = await this.createForObject(
                        userId,
                        objectType,
                        objectId,
                        {
                            lastViewedAt: viewedAt,
                            name: `Analytics View Day ${day + 1} #${viewNum + 1}`,
                        },
                    );
                    views.push(view);
                }
            }
        }
        
        return views;
    }

    async createRetentionData(
        userId: string,
        objectId: string,
        objectType: keyof typeof ViewFor,
        pattern: "daily" | "weekly" | "sporadic",
    ): Promise<view[]> {
        const views: view[] = [];
        const now = new Date();
        
        const patterns = {
            daily: { count: 30, interval: 24 * 60 * 60 * 1000 }, // 30 days, daily
            weekly: { count: 12, interval: 7 * 24 * 60 * 60 * 1000 }, // 12 weeks, weekly
            sporadic: { count: 10, interval: 0 }, // Random intervals
        };
        
        const { count, interval } = patterns[pattern];
        
        for (let i = 0; i < count; i++) {
            let viewedAt: Date;
            
            if (pattern === "sporadic") {
                // Random interval between 1 and 30 days
                const randomDays = Math.floor(Math.random() * 30) + 1;
                viewedAt = new Date(now.getTime() - (randomDays * 24 * 60 * 60 * 1000));
            } else {
                viewedAt = new Date(now.getTime() - (i * interval));
            }
            
            const view = await this.createForObject(
                userId,
                objectType,
                objectId,
                {
                    lastViewedAt: viewedAt,
                    name: `${pattern} retention view ${i + 1}`,
                },
            );
            views.push(view);
        }
        
        return views;
    }

    protected async checkModelConstraints(record: view): Promise<string[]> {
        const violations: string[] = [];
        
        // Must have byId (viewer)
        if (!record.byId) {
            violations.push("View must have byId (user)");
        }

        // Check viewer exists
        if (record.byId) {
            const user = await this.prisma.user.findUnique({
                where: { id: record.byId },
            });
            if (!user) {
                violations.push("View user must exist");
            }
        }

        // Check exactly one viewed object
        const viewableFields = ["issueId", "resourceId", "teamId", "userId"];
        const connectedObjects = viewableFields.filter(field => 
            record[field as keyof view],
        );
        
        if (connectedObjects.length === 0) {
            violations.push("View must reference exactly one viewable object");
        } else if (connectedObjects.length > 1) {
            violations.push("View cannot reference multiple objects");
        }

        // Check timestamp validity
        if (record.lastViewedAt && record.lastViewedAt > new Date(Date.now() + 60000)) {
            violations.push("View timestamp should not be in the future");
        }

        return violations;
    }

    /**
     * Get invalid data scenarios
     */
    getInvalidScenarios(): Record<string, any> {
        return {
            missingRequired: {
                // Missing byId and any target
                name: "Invalid View",
                lastViewedAt: new Date(),
            },
            invalidTypes: {
                id: "not-a-snowflake",
                name: 123, // Should be string
                lastViewedAt: "not-a-date", // Should be Date
                by: "invalid-user-reference", // Should be connect object
            },
            noViewedObject: {
                id: generatePK(),
                name: "View with no object",
                lastViewedAt: new Date(),
                by: { connect: { id: generatePK() } },
                // No object connected
            },
            multipleObjects: {
                id: generatePK(),
                name: "Multiple Objects View",
                lastViewedAt: new Date(),
                by: { connect: { id: generatePK() } },
                resource: { connect: { id: generatePK() } },
                team: { connect: { id: generatePK() } },
                // Multiple objects (invalid)
            },
            futureTimestamp: {
                id: generatePK(),
                name: "Future View",
                lastViewedAt: new Date(Date.now() + 86400000), // Tomorrow
                by: { connect: { id: generatePK() } },
                resource: { connect: { id: generatePK() } },
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.viewCreateInput> {
        const userId = generatePK();
        const sessionId = generatePK();
        const resourceId = generatePK();
        
        return {
            anonymousView: {
                id: generatePK(),
                name: "Anonymous Page View",
                lastViewedAt: new Date(),
                by: { connect: { id: userId } },
                resource: { connect: { id: resourceId } },
            },
            veryOldView: {
                ...this.getMinimalData(),
                by: { connect: { id: userId } },
                team: { connect: { id: generatePK() } },
                lastViewedAt: new Date("2020-01-01"),
            },
            maxLengthName: {
                ...this.getMinimalData(),
                by: { connect: { id: userId } },
                user: { connect: { id: generatePK() } },
                name: "A".repeat(255), // Maximum name length
            },
            selfView: {
                ...this.getMinimalData(),
                by: { connect: { id: userId } },
                user: { connect: { id: userId } }, // User viewing themselves
            },
            resourceView: {
                ...this.getMinimalData(),
                by: { connect: { id: userId } },
                resource: { connect: { id: generatePK() } },
                name: "Specific Resource View",
            },
            issueView: {
                ...this.getMinimalData(),
                by: { connect: { id: userId } },
                issue: { connect: { id: generatePK() } },
                name: "Issue Tracking View",
            },
        };
    }

    protected getCascadeInclude(): any {
        return {
            // Views don't have child records
        };
    }

    protected async deleteRelatedRecords(
        record: view,
        remainingDepth: number,
        tx: any,
    ): Promise<void> {
        // Views don't have child records to delete
    }
}

// Export factory creator function
export const createViewDbFactory = (prisma: PrismaClient) => new ViewDbFactory(prisma);
