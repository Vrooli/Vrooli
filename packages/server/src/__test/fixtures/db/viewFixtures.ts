import { generatePK } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for View model - used for seeding test data
 * Views track user interactions with objects (Issue, Resource, Team, User)
 */

// Consistent IDs for testing
export const viewDbIds = {
    view1: generatePK(),
    view2: generatePK(),
    view3: generatePK(),
    view4: generatePK(),
    view5: generatePK(),
};

/**
 * Minimal view data for database creation
 */
export const minimalViewDb: Prisma.viewCreateInput = {
    id: viewDbIds.view1,
    name: "Test View",
    lastViewedAt: new Date("2024-01-01T00:00:00Z"),
    by: { connect: { id: "user_123" } },
};

/**
 * View for Issue object
 */
export const issueViewDb: Prisma.viewCreateInput = {
    id: viewDbIds.view2,
    name: "Issue View",
    lastViewedAt: new Date("2024-01-01T12:00:00Z"),
    by: { connect: { id: "user_123" } },
    issue: { connect: { id: "issue_123" } },
};

/**
 * View for Resource object
 */
export const resourceViewDb: Prisma.viewCreateInput = {
    id: viewDbIds.view3,
    name: "Resource View", 
    lastViewedAt: new Date("2024-01-02T00:00:00Z"),
    by: { connect: { id: "user_123" } },
    resource: { connect: { id: "resource_123" } },
};

/**
 * View for Team object
 */
export const teamViewDb: Prisma.viewCreateInput = {
    id: viewDbIds.view4,
    name: "Team View",
    lastViewedAt: new Date("2024-01-03T00:00:00Z"),
    by: { connect: { id: "user_123" } },
    team: { connect: { id: "team_123" } },
};

/**
 * View for User object
 */
export const userViewDb: Prisma.viewCreateInput = {
    id: viewDbIds.view5,
    name: "User View",
    lastViewedAt: new Date("2024-01-04T00:00:00Z"),
    by: { connect: { id: "user_123" } },
    user: { connect: { id: "user_456" } },
};

/**
 * Factory for creating view database fixtures with overrides
 */
export class ViewDbFactory {
    static createMinimal(
        byId: string,
        overrides?: Partial<Prisma.viewCreateInput>
    ): Prisma.viewCreateInput {
        return {
            id: generatePK(),
            name: "Test View",
            lastViewedAt: new Date(),
            by: { connect: { id: byId } },
            ...overrides,
        };
    }

    static createForIssue(
        byId: string,
        issueId: string,
        overrides?: Partial<Prisma.viewCreateInput>
    ): Prisma.viewCreateInput {
        return {
            ...this.createMinimal(byId, overrides),
            name: "Issue View",
            issue: { connect: { id: issueId } },
        };
    }

    static createForResource(
        byId: string,
        resourceId: string,
        overrides?: Partial<Prisma.viewCreateInput>
    ): Prisma.viewCreateInput {
        return {
            ...this.createMinimal(byId, overrides),
            name: "Resource View",
            resource: { connect: { id: resourceId } },
        };
    }

    static createForTeam(
        byId: string,
        teamId: string,
        overrides?: Partial<Prisma.viewCreateInput>
    ): Prisma.viewCreateInput {
        return {
            ...this.createMinimal(byId, overrides),
            name: "Team View",
            team: { connect: { id: teamId } },
        };
    }

    static createForUser(
        byId: string,
        userId: string,
        overrides?: Partial<Prisma.viewCreateInput>
    ): Prisma.viewCreateInput {
        return {
            ...this.createMinimal(byId, overrides),
            name: "User Profile View",
            user: { connect: { id: userId } },
        };
    }

    /**
     * Create view for any object type
     */
    static createForObject(
        byId: string,
        objectId: string,
        objectType: "Issue" | "Resource" | "Team" | "User",
        overrides?: Partial<Prisma.viewCreateInput>
    ): Prisma.viewCreateInput {
        const factories = {
            Issue: this.createForIssue,
            Resource: this.createForResource,
            Team: this.createForTeam,
            User: this.createForUser,
        };

        return factories[objectType](byId, objectId, overrides);
    }

    /**
     * Create view with specific timestamp
     */
    static createWithTimestamp(
        byId: string,
        viewedAt: Date,
        overrides?: Partial<Prisma.viewCreateInput>
    ): Prisma.viewCreateInput {
        return {
            ...this.createMinimal(byId, overrides),
            lastViewedAt: viewedAt,
        };
    }

    /**
     * Create multiple views for the same user on different objects
     */
    static createViewHistory(
        byId: string,
        objects: Array<{ id: string; type: "Issue" | "Resource" | "Team" | "User"; viewedAt?: Date }>
    ): Prisma.viewCreateInput[] {
        return objects.map((obj, index) => 
            this.createForObject(byId, obj.id, obj.type, {
                lastViewedAt: obj.viewedAt || new Date(Date.now() - (index * 60000)), // 1 minute intervals
            })
        );
    }
}

/**
 * Helper to seed views for testing
 */
export async function seedViews(
    prisma: any,
    options: {
        byId: string;
        objects: Array<{ id: string; type: "Issue" | "Resource" | "Team" | "User" }>;
        withTimestamps?: boolean;
    }
) {
    const views = [];

    for (let i = 0; i < options.objects.length; i++) {
        const obj = options.objects[i];
        const viewData = ViewDbFactory.createForObject(
            options.byId,
            obj.id,
            obj.type,
            options.withTimestamps ? {
                lastViewedAt: new Date(Date.now() - (i * 60000)), // 1 minute intervals
            } : undefined
        );

        const view = await prisma.view.create({
            data: viewData,
            include: {
                by: true,
                issue: true,
                resource: true,
                team: true,
                user: true,
            },
        });
        views.push(view);
    }

    return views;
}

/**
 * Helper to create recent view activity for a user
 */
export async function seedRecentActivity(
    prisma: any,
    userId: string,
    count: number = 5
) {
    const activities = [];
    const now = new Date();

    for (let i = 0; i < count; i++) {
        const viewedAt = new Date(now.getTime() - (i * 3600000)); // 1 hour intervals
        
        const viewData = ViewDbFactory.createWithTimestamp(
            userId,
            viewedAt,
            {
                name: `Recent Activity ${i + 1}`,
            }
        );

        const activity = await prisma.view.create({
            data: viewData,
            include: { by: true },
        });
        activities.push(activity);
    }

    return activities;
}

/**
 * Helper to create view analytics data for testing
 */
export async function seedViewAnalytics(
    prisma: any,
    objectId: string,
    objectType: "Issue" | "Resource" | "Team" | "User",
    options: {
        viewerIds: string[];
        daysBack?: number;
        viewsPerDay?: number;
    }
) {
    const views = [];
    const { viewerIds, daysBack = 7, viewsPerDay = 3 } = options;
    const now = new Date();

    for (let day = 0; day < daysBack; day++) {
        for (let view = 0; view < viewsPerDay; view++) {
            const viewerId = viewerIds[view % viewerIds.length];
            const viewedAt = new Date(
                now.getTime() - (day * 24 * 60 * 60 * 1000) - (view * 60 * 60 * 1000)
            );

            const viewData = ViewDbFactory.createForObject(
                viewerId,
                objectId,
                objectType,
                {
                    lastViewedAt: viewedAt,
                    name: `Analytics View Day ${day + 1} #${view + 1}`,
                }
            );

            const analyticsView = await prisma.view.create({
                data: viewData,
                include: { by: true },
            });
            views.push(analyticsView);
        }
    }

    return views;
}