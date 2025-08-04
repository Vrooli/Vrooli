/* eslint-disable no-magic-numbers */
import { type Prisma } from "@prisma/client";
import { generatePK, ViewFor } from "@vrooli/shared";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type { BulkSeedResult, DbErrorScenarios, DbTestFixtures } from "./types.js";

/**
 * Database fixtures for View model - used for seeding test data
 * Views track user interactions with objects and support anonymous viewing
 * 
 * Supported object types from ViewFor enum:
 * Resource, ResourceVersion, Team, User
 */

// Consistent IDs for testing - using lazy initialization to avoid module-level generatePK() calls
let _viewDbIds: Record<string, bigint> | null = null;
export function getViewDbIds() {
    if (!_viewDbIds) {
        _viewDbIds = {
            view1: generatePK(),
            view2: generatePK(),
            view3: generatePK(),
            view4: generatePK(),
            view5: generatePK(),
            user1: generatePK(),
            user2: generatePK(),
            session1: generatePK(),
            resource1: generatePK(),
            resourceVersion1: generatePK(),
            team1: generatePK(),
        };
    }
    return _viewDbIds;
}

/**
 * Enhanced test fixtures for View model following standard structure
 */
export function getViewDbFixtures(): DbTestFixtures<Prisma.viewUncheckedCreateInput> {
    return {
        minimal: {
            id: generatePK(),
            name: "Test View",
            lastViewedAt: new Date(),
            // Note: Views must have either byId (user) or bySessionId
            byId: getViewDbIds().user1,
        },
        complete: {
            id: generatePK(),
            name: "Complete View",
            lastViewedAt: new Date(),
            byId: getViewDbIds().user1,
            resourceId: getViewDbIds().resource1,
        },
        invalid: {
            missingRequired: {
                // Missing both byId and bySessionId
                name: "Invalid View",
                lastViewedAt: new Date(),
            },
            invalidTypes: {
                id: "not-a-valid-snowflake",
                name: 123, // Should be string
                lastViewedAt: "not-a-date", // Should be Date
                byId: "invalid-user-id", // Should be valid BigInt
            },
            bothUserAndSession: {
                id: generatePK(),
                name: "Invalid View",
                lastViewedAt: new Date(),
                byId: getViewDbIds().user1,
                bySessionId: getViewDbIds().session1, // Cannot have both
            },
            noViewedObject: {
                id: generatePK(),
                name: "View with no object",
                lastViewedAt: new Date(),
                byId: getViewDbIds().user1,
                // No object connected
            },
            multipleObjects: {
                id: generatePK(),
                name: "Multiple Objects View",
                lastViewedAt: new Date(),
                byId: getViewDbIds().user1,
                resourceId: getViewDbIds().resource1,
                teamId: getViewDbIds().team1, // Multiple objects (invalid)
            },
        },
        edgeCases: {
            anonymousView: {
                id: generatePK(),
                name: "Anonymous View",
                lastViewedAt: new Date(),
                bySessionId: getViewDbIds().session1, // Session instead of user
                resourceId: getViewDbIds().resource1,
            },
            recentView: {
                id: generatePK(),
                name: "Recent View",
                lastViewedAt: new Date(), // Current time
                byId: getViewDbIds().user1,
                teamId: getViewDbIds().team1,
            },
            oldView: {
                id: generatePK(),
                name: "Old View",
                lastViewedAt: new Date("2020-01-01"), // Old timestamp
                byId: getViewDbIds().user1,
                userId: getViewDbIds().user2,
            },
            resourceVersionView: {
                id: generatePK(),
                name: "Resource Version View",
                lastViewedAt: new Date(),
                byId: getViewDbIds().user1,
                resourceVersionId: getViewDbIds().resourceVersion1,
            },
            maxLengthName: {
                id: generatePK(),
                name: "a".repeat(255), // Maximum name length
                lastViewedAt: new Date(),
                byId: getViewDbIds().user1,
                resourceId: getViewDbIds().resource1,
            },
        },
    };
}

/**
 * Enhanced factory for creating view database fixtures
 */
export class ViewDbFactory extends EnhancedDbFactory<Prisma.viewUncheckedCreateInput> {

    /**
     * Get the test fixtures for View model
     */
    protected getFixtures(): DbTestFixtures<Prisma.viewUncheckedCreateInput> {
        return getViewDbFixtures();
    }

    /**
     * Get View-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {
                    id: getViewDbIds().view1, // Duplicate ID
                    name: "Test View",
                    lastViewedAt: new Date(),
                    byId: getViewDbIds().user1,
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    name: "Test View",
                    lastViewedAt: new Date(),
                    byId: BigInt("9999999999999999"), // Non-existent user
                },
                checkConstraintViolation: {
                    id: generatePK(),
                    name: "", // Empty name might violate constraint
                    lastViewedAt: new Date(),
                    byId: getViewDbIds().user1,
                },
            },
            validation: {
                requiredFieldMissing: getViewDbFixtures().invalid.missingRequired,
                invalidDataType: getViewDbFixtures().invalid.invalidTypes,
                outOfRange: {
                    id: generatePK(),
                    name: "a".repeat(256), // Name too long
                    lastViewedAt: new Date(),
                    byId: getViewDbIds().user1,
                },
            },
            businessLogic: {
                bothUserAndSession: getViewDbFixtures().invalid.bothUserAndSession,
                noViewedObject: getViewDbFixtures().invalid.noViewedObject,
                multipleObjects: getViewDbFixtures().invalid.multipleObjects,
                futureTimestamp: {
                    id: generatePK(),
                    name: "Future View",
                    lastViewedAt: new Date(Date.now() + 86400000), // Tomorrow
                    byId: getViewDbIds().user1,
                    resourceId: getViewDbIds().resource1,
                },
            },
        };
    }

    /**
     * Generate fresh identifiers for views
     */
    protected generateFreshIdentifiers(): Record<string, any> {
        return {
            id: generatePK(),
            // Views don't have publicId or handle
        };
    }

    /**
     * View-specific validation
     */
    protected validateSpecific(data: Prisma.viewUncheckedCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields specific to View
        if (!data.name) {
            errors.push("View name is required");
        }

        if (!data.lastViewedAt) {
            errors.push("View lastViewedAt is required");
        }

        // Must have either byId or bySessionId
        if (!data.byId && !data.bySessionId) {
            errors.push("View must have either byId (user) or bySessionId");
        }

        if (data.byId && data.bySessionId) {
            errors.push("View cannot have both byId and bySessionId");
        }

        // Check business logic - must have exactly one viewed object
        const viewableFields = ["resourceId", "resourceVersionId", "teamId", "userId"];
        const connectedObjects = viewableFields.filter(field => data[field as keyof Prisma.viewUncheckedCreateInput]);

        if (connectedObjects.length === 0) {
            warnings.push("View should reference exactly one viewable object");
        } else if (connectedObjects.length > 1) {
            errors.push("View cannot reference multiple objects");
        }

        // Check timestamp validity
        if (data.lastViewedAt && data.lastViewedAt > new Date(Date.now() + 60000)) { // Allow 1 minute future tolerance
            warnings.push("View timestamp is in the future");
        }

        return { errors, warnings };
    }

    // Static methods for backward compatibility
    static createMinimal(
        byId: bigint,
        overrides?: Partial<Prisma.viewUncheckedCreateInput>,
    ): Prisma.viewUncheckedCreateInput {
        const factory = new ViewDbFactory();
        return factory.createMinimal({
            byId,
            ...overrides,
        });
    }

    static createForObject(
        byId: bigint,
        objectId: bigint,
        objectType: ViewFor | "Issue" | "Resource" | "Team" | "User",
        overrides?: Partial<Prisma.viewUncheckedCreateInput>,
    ): Prisma.viewUncheckedCreateInput {
        const factory = new ViewDbFactory();

        // Map ViewFor enum and legacy types to field names
        const typeMapping: Record<string, string> = {
            [ViewFor.Resource]: "resourceId",
            [ViewFor.ResourceVersion]: "resourceVersionId",
            [ViewFor.Team]: "teamId",
            [ViewFor.User]: "userId",
            // Legacy mappings
            "Issue": "issueId",
            "Resource": "resourceId",
            "Team": "teamId",
            "User": "userId",
        };

        const fieldName = typeMapping[objectType];
        if (!fieldName) {
            throw new Error(`Invalid view object type: ${objectType}`);
        }

        return factory.createMinimal({
            byId,
            [fieldName]: objectId,
            name: `${objectType} View`,
            ...overrides,
        });
    }

    static createAnonymousView(
        sessionId: bigint,
        objectId: bigint,
        objectType: ViewFor | "Issue" | "Resource" | "Team" | "User",
        overrides?: Partial<Prisma.viewUncheckedCreateInput>,
    ): Prisma.viewUncheckedCreateInput {
        const factory = new ViewDbFactory();
        const typeMapping: Record<string, string> = {
            [ViewFor.Resource]: "resourceId",
            [ViewFor.ResourceVersion]: "resourceVersionId",
            [ViewFor.Team]: "teamId",
            [ViewFor.User]: "userId",
            "Issue": "issueId",
            "Resource": "resourceId",
            "Team": "teamId",
            "User": "userId",
        };

        const fieldName = typeMapping[objectType];
        const data = factory.createMinimal({
            [fieldName]: objectId,
            name: `Anonymous ${objectType} View`,
            ...overrides,
        });

        // Remove byId and set bySessionId for anonymous views
        delete data.byId;
        data.bySessionId = sessionId;

        return data;
    }

    static createWithTimestamp(
        byId: bigint,
        viewedAt: Date,
        overrides?: Partial<Prisma.viewUncheckedCreateInput>,
    ): Prisma.viewUncheckedCreateInput {
        return this.createMinimal(byId, {
            lastViewedAt: viewedAt,
            ...overrides,
        });
    }

    static createViewHistory(
        byId: bigint,
        objects: Array<{ id: bigint; type: ViewFor | "Issue" | "Resource" | "Team" | "User"; viewedAt?: Date }>,
    ): Prisma.viewUncheckedCreateInput[] {
        return objects.map((obj, index) =>
            this.createForObject(byId, obj.id, obj.type, {
                lastViewedAt: obj.viewedAt || new Date(Date.now() - (index * 60000)), // 1 minute intervals
            }),
        );
    }

    static createForIssue(
        byId: bigint,
        issueId: bigint,
        overrides?: Partial<Prisma.viewUncheckedCreateInput>,
    ): Prisma.viewUncheckedCreateInput {
        return this.createForObject(byId, issueId, "Issue", overrides);
    }

    static createForResource(
        byId: bigint,
        resourceId: bigint,
        overrides?: Partial<Prisma.viewUncheckedCreateInput>,
    ): Prisma.viewUncheckedCreateInput {
        return this.createForObject(byId, resourceId, ViewFor.Resource, overrides);
    }

    static createForTeam(
        byId: bigint,
        teamId: bigint,
        overrides?: Partial<Prisma.viewUncheckedCreateInput>,
    ): Prisma.viewUncheckedCreateInput {
        return this.createForObject(byId, teamId, ViewFor.Team, overrides);
    }

    static createForUser(
        byId: bigint,
        userId: bigint,
        overrides?: Partial<Prisma.viewUncheckedCreateInput>,
    ): Prisma.viewUncheckedCreateInput {
        return this.createForObject(byId, userId, ViewFor.User, overrides);
    }
}

/**
 * Helper to seed views for testing
 */
export async function seedViews(
    prisma: any,
    options: {
        byId?: bigint;
        bySessionId?: bigint;
        objects: Array<{ id: bigint; type: ViewFor | "Issue" | "Resource" | "Team" | "User" }>;
        withTimestamps?: boolean;
    },
): Promise<BulkSeedResult<any>> {
    const views = [];

    if (!options.byId && !options.bySessionId) {
        throw new Error("Must provide either byId or bySessionId for views");
    }

    for (let i = 0; i < options.objects.length; i++) {
        const obj = options.objects[i];
        const viewData = options.byId
            ? ViewDbFactory.createForObject(
                options.byId,
                obj.id,
                obj.type,
                options.withTimestamps ? {
                    lastViewedAt: new Date(Date.now() - (i * 60000)), // 1 minute intervals
                } : undefined,
            )
            : ViewDbFactory.createAnonymousView(
                options.bySessionId!,
                obj.id,
                obj.type,
                options.withTimestamps ? {
                    lastViewedAt: new Date(Date.now() - (i * 60000)),
                } : undefined,
            );

        const view = await prisma.view.create({
            data: viewData,
            include: {
                by: true,
                issue: true,
                resource: true,
                resourceVersion: true,
                team: true,
                user: true,
            },
        });
        views.push(view);
    }

    return {
        records: views,
        summary: {
            total: views.length,
            withAuth: options.byId ? views.length : 0,
            bots: 0,
            teams: 0,
        },
    };
}

/**
 * Helper to create recent view activity for a user
 */
export async function seedRecentActivity(
    prisma: any,
    userId: bigint,
    count = 5,
): Promise<BulkSeedResult<any>> {
    const activities = [];
    const now = new Date();

    for (let i = 0; i < count; i++) {
        const viewedAt = new Date(now.getTime() - (i * 3600000)); // 1 hour intervals

        const viewData = ViewDbFactory.createWithTimestamp(
            userId,
            viewedAt,
            {
                name: `Recent Activity ${i + 1}`,
            },
        );

        const activity = await prisma.view.create({
            data: viewData,
            include: { by: true },
        });
        activities.push(activity);
    }

    return {
        records: activities,
        summary: {
            total: activities.length,
            withAuth: activities.length,
            bots: 0,
            teams: 0,
        },
    };
}

/**
 * Helper to create view analytics data for testing
 */
export async function seedViewAnalytics(
    prisma: any,
    objectId: bigint,
    objectType: ViewFor | "Issue" | "Resource" | "Team" | "User",
    options: {
        viewerIds?: bigint[];
        sessionIds?: bigint[];
        daysBack?: number;
        viewsPerDay?: number;
    },
): Promise<BulkSeedResult<any>> {
    const views = [];
    const { viewerIds = [], sessionIds = [], daysBack = 7, viewsPerDay = 3 } = options;
    const now = new Date();
    let anonymousCount = 0;

    if (viewerIds.length === 0 && sessionIds.length === 0) {
        throw new Error("Must provide either viewerIds or sessionIds for analytics");
    }

    for (let day = 0; day < daysBack; day++) {
        for (let view = 0; view < viewsPerDay; view++) {
            const useSession = sessionIds.length > 0 && (viewerIds.length === 0 || Math.random() > 0.5);
            const viewedAt = new Date(
                now.getTime() - (day * 24 * 60 * 60 * 1000) - (view * 60 * 60 * 1000),
            );

            const viewData = useSession
                ? ViewDbFactory.createAnonymousView(
                    sessionIds[view % sessionIds.length],
                    objectId,
                    objectType,
                    {
                        lastViewedAt: viewedAt,
                        name: `Analytics View Day ${day + 1} #${view + 1} (Anonymous)`,
                    },
                )
                : ViewDbFactory.createForObject(
                    viewerIds[view % viewerIds.length],
                    objectId,
                    objectType,
                    {
                        lastViewedAt: viewedAt,
                        name: `Analytics View Day ${day + 1} #${view + 1}`,
                    },
                );

            if (useSession) anonymousCount++;

            const analyticsView = await prisma.view.create({
                data: viewData,
                include: { by: true },
            });
            views.push(analyticsView);
        }
    }

    return {
        records: views,
        summary: {
            total: views.length,
            withAuth: views.length - anonymousCount,
            bots: 0,
            teams: 0,
            anonymous: anonymousCount,
        },
    };
}

/**
 * Helper to seed views by date range
 */
export async function seedViewsByDateRange(
    prisma: any,
    options: {
        userId: bigint;
        objectId: bigint;
        objectType: ViewFor | "Issue" | "Resource" | "Team" | "User";
        startDate: Date;
        endDate: Date;
        viewsPerDay?: number;
    },
): Promise<BulkSeedResult<any>> {
    const views = [];
    const { userId, objectId, objectType, startDate, endDate, viewsPerDay = 1 } = options;

    const days = Math.ceil((endDate.getTime() - startDate.getTime()) / (1000 * 60 * 60 * 24));

    for (let day = 0; day < days; day++) {
        for (let viewNum = 0; viewNum < viewsPerDay; viewNum++) {
            const viewedAt = new Date(
                startDate.getTime() + (day * 24 * 60 * 60 * 1000) + (viewNum * 60 * 60 * 1000),
            );

            if (viewedAt > endDate) break;

            const viewData = ViewDbFactory.createForObject(
                userId,
                objectId,
                objectType,
                {
                    lastViewedAt: viewedAt,
                    name: `Date Range View Day ${day + 1} #${viewNum + 1}`,
                },
            );

            const view = await prisma.view.create({ data: viewData });
            views.push(view);
        }
    }

    return {
        records: views,
        summary: {
            total: views.length,
            withAuth: views.length,
            bots: 0,
            teams: 0,
        },
    };
}
