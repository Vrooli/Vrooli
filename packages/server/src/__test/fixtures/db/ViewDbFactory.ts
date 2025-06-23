import { generatePK, generatePublicId, nanoid } from "./idHelpers.js";
import { type view, type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
    TestScenario,
} from "./types.js";

interface ViewRelationConfig extends RelationConfig {
    userId?: string;
    targetType?: "issue" | "resource" | "team" | "user";
    targetId?: string;
    isAuthenticated?: boolean;
}

/**
 * Enhanced database fixture factory for View model
 * Provides comprehensive testing capabilities for view tracking
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for polymorphic relationships (issue, resource, team, user)
 * - Anonymous and authenticated view tracking
 * - View name/identifier management
 * - Predefined test scenarios
 * - Last viewed timestamp handling
 */
export class ViewDbFactory extends EnhancedDatabaseFactory<
    Prisma.viewCreateInput,
    Prisma.viewCreateInput,
    Prisma.viewInclude,
    Prisma.viewUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super("view", prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.view;
    }

    /**
     * Get complete test fixtures for View model
     */
    protected getFixtures(): DbTestFixtures<Prisma.viewCreateInput, Prisma.viewUpdateInput> {
        return {
            minimal: {
                id: generatePK(),
                name: `view_${nanoid(8)}`,
                by: { connect: { id: generatePK() } },
                // Must have at least one target
                resource: { connect: { id: generatePK() } },
            },
            complete: {
                id: generatePK(),
                name: `complete_view_${nanoid(8)}`,
                by: { connect: { id: generatePK() } },
                resource: { connect: { id: generatePK() } },
                lastViewedAt: new Date(),
            },
            invalid: {
                missingRequired: {
                    // Missing id, name, byId, and any target
                },
                invalidTypes: {
                    id: "not-a-snowflake",
                    name: 123, // Should be string
                    by: null, // Should be connect object
                    resource: true, // Should be connect object
                    lastViewedAt: "not-a-date", // Should be Date
                },
                noTarget: {
                    id: generatePK(),
                    name: `view_${nanoid(8)}`,
                    by: { connect: { id: generatePK() } },
                    // No target specified
                },
                multipleTargets: {
                    id: generatePK(),
                    name: `view_${nanoid(8)}`,
                    by: { connect: { id: generatePK() } },
                    resource: { connect: { id: generatePK() } },
                    issue: { connect: { id: generatePK() } }, // Multiple targets not allowed
                },
                nameTooLong: {
                    id: generatePK(),
                    name: "a".repeat(129), // Exceeds 128 character limit
                    by: { connect: { id: generatePK() } },
                    resource: { connect: { id: generatePK() } },
                },
            },
            edgeCases: {
                anonymousView: {
                    id: generatePK(),
                    name: `anon_${nanoid(8)}`, // Anonymous identifier
                    by: { connect: { id: generatePK() } }, // Anonymous user
                    resource: { connect: { id: generatePK() } },
                },
                viewWithSessionId: {
                    id: generatePK(),
                    name: `session_${nanoid(32)}`, // Session-based identifier
                    by: { connect: { id: generatePK() } },
                    resource: { connect: { id: generatePK() } },
                },
                viewOfIssue: {
                    id: generatePK(),
                    name: `issue_view_${nanoid(8)}`,
                    by: { connect: { id: generatePK() } },
                    issue: { connect: { id: generatePK() } },
                },
                viewOfTeam: {
                    id: generatePK(),
                    name: `team_view_${nanoid(8)}`,
                    by: { connect: { id: generatePK() } },
                    team: { connect: { id: generatePK() } },
                },
                viewOfUser: {
                    id: generatePK(),
                    name: `user_view_${nanoid(8)}`,
                    by: { connect: { id: generatePK() } },
                    user: { connect: { id: generatePK() } },
                },
                oldView: {
                    id: generatePK(),
                    name: `old_view_${nanoid(8)}`,
                    by: { connect: { id: generatePK() } },
                    resource: { connect: { id: generatePK() } },
                    lastViewedAt: new Date(Date.now() - 365 * 24 * 60 * 60 * 1000), // 1 year ago
                },
                recentView: {
                    id: generatePK(),
                    name: `recent_view_${nanoid(8)}`,
                    by: { connect: { id: generatePK() } },
                    resource: { connect: { id: generatePK() } },
                    lastViewedAt: new Date(Date.now() - 5 * 60 * 1000), // 5 minutes ago
                },
            },
            updates: {
                minimal: {
                    lastViewedAt: new Date(), // Update view time
                },
                complete: {
                    name: `updated_view_${nanoid(8)}`,
                    lastViewedAt: new Date(),
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.viewCreateInput>): Prisma.viewCreateInput {
        return {
            id: generatePK(),
            name: `view_${nanoid(8)}`,
            by: { connect: { id: generatePK() } },
            resource: { connect: { id: generatePK() } }, // Default to resource
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.viewCreateInput>): Prisma.viewCreateInput {
        return {
            id: generatePK(),
            name: `complete_view_${nanoid(8)}`,
            by: { connect: { id: generatePK() } },
            resource: { connect: { id: generatePK() } },
            lastViewedAt: new Date(),
            ...overrides,
        };
    }

    /**
     * Initialize test scenarios
     */
    protected initializeScenarios(): void {
        this.scenarios = {
            authenticatedView: {
                name: "authenticatedView",
                description: "View from authenticated user",
                config: {
                    isAuthenticated: true,
                    targetType: "resource",
                },
            },
            anonymousView: {
                name: "anonymousView",
                description: "View from anonymous visitor",
                config: {
                    overrides: {
                        by: { connect: { id: generatePK() } }, // Anonymous user ID
                        name: `anon_${nanoid(16)}`,
                    },
                    isAuthenticated: false,
                    targetType: "resource",
                },
            },
            profileView: {
                name: "profileView",
                description: "View of user profile",
                config: {
                    targetType: "user",
                },
            },
            teamPageView: {
                name: "teamPageView",
                description: "View of team page",
                config: {
                    targetType: "team",
                },
            },
            issueView: {
                name: "issueView",
                description: "View of issue details",
                config: {
                    targetType: "issue",
                },
            },
            repeatedView: {
                name: "repeatedView",
                description: "Multiple views from same user",
                config: {
                    overrides: {
                        lastViewedAt: new Date(),
                    },
                },
            },
            oldView: {
                name: "oldView",
                description: "View from long ago",
                config: {
                    overrides: {
                        lastViewedAt: new Date(Date.now() - 180 * 24 * 60 * 60 * 1000), // 6 months ago
                    },
                },
            },
        };
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
            issue: {
                select: {
                    id: true,
                    publicId: true,
                },
            },
            resource: {
                select: {
                    id: true,
                    publicId: true,
                },
            },
            team: {
                select: {
                    id: true,
                    publicId: true,
                    handle: true,
                },
            },
            user: {
                select: {
                    id: true,
                    publicId: true,
                    name: true,
                    handle: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.viewCreateInput,
        config: ViewRelationConfig,
        tx: any,
    ): Promise<Prisma.viewCreateInput> {
        const data = { ...baseData };

        // Handle user association
        if (config.userId) {
            data.by = { connect: { id: BigInt(config.userId) } };
        }

        // Handle anonymous views
        if (config.isAuthenticated === false) {
            data.by = { connect: { id: generatePK() } }; // Anonymous user
            data.name = `anon_${nanoid(16)}`; // Anonymous session identifier
        }

        // Handle target type
        if (config.targetType && config.targetId) {
            // Clear any existing targets
            delete data.issue;
            delete data.resource;
            delete data.team;
            delete data.user;

            // Set the appropriate target
            switch (config.targetType) {
                case "issue":
                    data.issue = { connect: { id: BigInt(config.targetId) } };
                    break;
                case "resource":
                    data.resource = { connect: { id: BigInt(config.targetId) } };
                    break;
                case "team":
                    data.team = { connect: { id: BigInt(config.targetId) } };
                    break;
                case "user":
                    data.user = { connect: { id: BigInt(config.targetId) } };
                    break;
            }
        }

        return data;
    }

    /**
     * Create a view for a specific target
     */
    async createViewFor(
        targetType: "issue" | "resource" | "team" | "user",
        targetId: string,
        viewerId: string,
        viewName?: string,
    ): Promise<view> {
        return await this.createWithRelations({
            overrides: { 
                name: viewName || `view_${nanoid(8)}`,
            },
            userId: viewerId,
            targetType,
            targetId,
        });
    }

    /**
     * Create an anonymous view
     */
    async createAnonymousView(
        targetType: "issue" | "resource" | "team" | "user",
        targetId: string,
        sessionId?: string,
    ): Promise<view> {
        return await this.createWithRelations({
            overrides: {
                name: sessionId || `anon_${nanoid(16)}`,
                by: { connect: { id: generatePK() } },
            },
            isAuthenticated: false,
            targetType,
            targetId,
        });
    }

    /**
     * Create multiple views for analytics testing
     */
    async createViewsForAnalytics(
        targetType: "issue" | "resource" | "team" | "user",
        targetId: string,
        viewCount: number,
        options?: {
            authenticatedRatio?: number; // 0-1, percentage of authenticated views
            timeSpreadDays?: number; // Spread views over this many days
        },
    ): Promise<view[]> {
        const views: view[] = [];
        const authenticatedRatio = options?.authenticatedRatio ?? 0.7;
        const timeSpreadDays = options?.timeSpreadDays ?? 30;

        for (let i = 0; i < viewCount; i++) {
            const isAuthenticated = Math.random() < authenticatedRatio;
            const daysAgo = Math.floor(Math.random() * timeSpreadDays);
            const viewTime = new Date(Date.now() - daysAgo * 24 * 60 * 60 * 1000);

            const view = isAuthenticated
                ? await this.createViewFor(
                    targetType,
                    targetId,
                    generatePK().toString(),
                    `user_view_${i}`,
                )
                : await this.createAnonymousView(
                    targetType,
                    targetId,
                    `anon_session_${i}`,
                );

            // Update the view time
            const updatedView = await this.prisma.view.update({
                where: { id: view.id },
                data: { lastViewedAt: viewTime },
                include: this.getDefaultInclude(),
            });

            views.push(updatedView);
        }

        return views;
    }

    /**
     * Update view timestamp (for repeat views)
     */
    async updateViewTime(viewId: string): Promise<view> {
        return await this.prisma.view.update({
            where: { id: viewId },
            data: { lastViewedAt: new Date() },
            include: this.getDefaultInclude(),
        });
    }

    protected async checkModelConstraints(record: view): Promise<string[]> {
        const violations: string[] = [];
        
        // Check that only one target is specified
        const targetCount = [
            record.issueId,
            record.resourceId,
            record.teamId,
            record.userId,
        ].filter(Boolean).length;

        if (targetCount === 0) {
            violations.push("View must have exactly one target");
        } else if (targetCount > 1) {
            violations.push("View cannot have multiple targets");
        }

        // Check name length
        if (!record.name || record.name.length === 0) {
            violations.push("View name cannot be empty");
        }

        if (record.name && record.name.length > 128) {
            violations.push("View name exceeds maximum length of 128 characters");
        }

        // Check viewer exists
        if (record.byId) {
            const user = await this.prisma.user.findUnique({
                where: { id: record.byId },
            });
            if (!user) {
                violations.push("Viewer user does not exist");
            }
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            // Views don't have dependent records to cascade delete
        };
    }

    protected async deleteRelatedRecords(
        record: view,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[],
    ): Promise<void> {
        // Views don't have dependent records
    }

    /**
     * Get view statistics for a target
     */
    async getViewStats(
        targetType: "issue" | "resource" | "team" | "user",
        targetId: string,
        options?: {
            startDate?: Date;
            endDate?: Date;
        },
    ): Promise<{
        totalViews: number;
        uniqueViewers: number;
        authenticatedViews: number;
        anonymousViews: number;
        viewsByDay: Record<string, number>;
    }> {
        const where: any = {};
        where[`${targetType}Id`] = targetId;

        if (options?.startDate || options?.endDate) {
            where.lastViewedAt = {};
            if (options.startDate) {
                where.lastViewedAt.gte = options.startDate;
            }
            if (options.endDate) {
                where.lastViewedAt.lte = options.endDate;
            }
        }

        const views = await this.prisma.view.findMany({
            where,
            select: {
                byId: true,
                lastViewedAt: true,
            },
        });

        const uniqueViewers = new Set(views.map(v => v.byId)).size;
        const authenticatedViews = views.filter(v => v.byId !== "0").length;
        const anonymousViews = views.filter(v => v.byId === "0").length;

        // Group by day
        const viewsByDay: Record<string, number> = {};
        views.forEach(view => {
            const day = view.lastViewedAt.toISOString().split("T")[0];
            viewsByDay[day] = (viewsByDay[day] || 0) + 1;
        });

        return {
            totalViews: views.length,
            uniqueViewers,
            authenticatedViews,
            anonymousViews,
            viewsByDay,
        };
    }

    /**
     * Clean up old anonymous views
     */
    async cleanupOldAnonymousViews(daysOld: number): Promise<number> {
        const cutoffDate = new Date(Date.now() - daysOld * 24 * 60 * 60 * 1000);
        
        const result = await this.prisma.view.deleteMany({
            where: {
                by: { id: generatePK() }, // Anonymous user ID placeholder
                lastViewedAt: { lt: cutoffDate },
            },
        });

        return result.count;
    }
}

// Export factory creator function
export const createViewDbFactory = (prisma: PrismaClient) => 
    ViewDbFactory.getInstance("view", prisma);

// Export the class for type usage
export { ViewDbFactory as ViewDbFactoryClass };
