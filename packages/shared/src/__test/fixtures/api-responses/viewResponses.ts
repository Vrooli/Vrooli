/* c8 ignore start */
/**
 * View API Response Fixtures
 * 
 * Comprehensive fixtures for view tracking endpoints including
 * analytics, user activity tracking, and content engagement metrics.
 */

import type {
    View,
    ViewCreateInput,
    ViewUpdateInput,
    ViewFor,
    User,
} from "../../../api/types.js";
import { BaseAPIResponseFactory } from "./base.js";
import type { MockDataOptions } from "./types.js";
import { generatePK } from "../../../id/index.js";
import { userResponseFactory } from "./userResponses.js";
import { resourceResponseFactory } from "./resourceResponses.js";

// Constants
const DEFAULT_COUNT = 10;
const DEFAULT_ERROR_RATE = 0.1;
const DEFAULT_DELAY_MS = 500;
const HOURS_24 = 24 * 60 * 60 * 1000;
const DAYS_7 = 7 * HOURS_24;
const DAYS_30 = 30 * HOURS_24;

// Viewable object types
const VIEW_FOR_TYPES: ViewFor[] = ["Api", "Project", "Routine", "Standard", "Team", "User"];

/**
 * View API response factory
 */
export class ViewResponseFactory extends BaseAPIResponseFactory<
    View,
    ViewCreateInput,
    ViewUpdateInput
> {
    protected readonly entityName = "view";

    /**
     * Create mock view data
     */
    createMockData(options?: MockDataOptions): View {
        const scenario = options?.scenario || "minimal";
        const now = new Date().toISOString();
        const viewId = options?.overrides?.id || generatePK().toString();

        const baseView: View = {
            __typename: "View",
            id: viewId,
            created_at: now,
            updated_at: now,
            lastViewedAt: now,
            by: userResponseFactory.createMockData(),
            to: resourceResponseFactory.createMockData(),
        };

        if (scenario === "complete" || scenario === "edge-case") {
            const user = userResponseFactory.createMockData({ scenario: "complete" });
            const resource = resourceResponseFactory.createMockData({ scenario: "complete" });
            
            return {
                ...baseView,
                lastViewedAt: new Date(Date.now() - (2 * 60 * 60 * 1000)).toISOString(), // 2 hours ago
                by: user,
                to: scenario === "edge-case" 
                    ? { ...resource, __typename: "Team" } // Switch to different viewable type
                    : resource,
                ...options?.overrides,
            };
        }

        return {
            ...baseView,
            ...options?.overrides,
        };
    }

    /**
     * Create view from input
     */
    createFromInput(input: ViewCreateInput): View {
        const now = new Date().toISOString();
        const viewId = generatePK().toString();

        // Create appropriate target object based on viewFor
        let target: any;
        if (input.viewFor === "User") {
            target = userResponseFactory.createMockData({ overrides: { id: input.forConnect } });
        } else if (input.viewFor === "Team") {
            target = { __typename: "Team", id: input.forConnect, name: "Viewed Team" };
        } else {
            target = resourceResponseFactory.createMockData({ overrides: { id: input.forConnect } });
            target.__typename = input.viewFor;
        }

        return {
            __typename: "View",
            id: viewId,
            created_at: now,
            updated_at: now,
            lastViewedAt: now,
            by: userResponseFactory.createMockData(), // Current user viewing
            to: target,
        };
    }

    /**
     * Update view from input
     */
    updateFromInput(existing: View, input: ViewUpdateInput): View {
        const updates: Partial<View> = {
            updated_at: new Date().toISOString(),
            lastViewedAt: new Date().toISOString(), // Update last viewed time
        };

        return {
            ...existing,
            ...updates,
        };
    }

    /**
     * Validate create input
     */
    async validateCreateInput(input: ViewCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (!input.forConnect) {
            errors.forConnect = "Target object ID is required";
        }

        if (!input.viewFor) {
            errors.viewFor = "View type must be specified";
        } else if (!VIEW_FOR_TYPES.includes(input.viewFor)) {
            errors.viewFor = "Invalid view type";
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Validate update input
     */
    async validateUpdateInput(input: ViewUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        // Views typically don't have much to update, but validate ID exists
        if (!input.id) {
            errors.id = "View ID is required";
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create views for all object types
     */
    createViewsForAllTypes(): View[] {
        return VIEW_FOR_TYPES.map((viewFor, index) => {
            let target: any;
            
            if (viewFor === "User") {
                target = userResponseFactory.createMockData();
            } else if (viewFor === "Team") {
                target = { __typename: "Team", id: generatePK().toString(), name: `Team ${index}` };
            } else {
                target = resourceResponseFactory.createMockData();
                target.__typename = viewFor;
            }

            return this.createMockData({
                overrides: {
                    id: `view_${viewFor.toLowerCase()}_${index}`,
                    to: target,
                    lastViewedAt: new Date(Date.now() - (index * 60 * 60 * 1000)).toISOString(),
                },
            });
        });
    }

    /**
     * Create trending views (high engagement)
     */
    createTrendingViews(): View[] {
        const trendingObjects = [
            { type: "Routine", views: 1500, score: 4.8 },
            { type: "Project", views: 1200, score: 4.6 },
            { type: "Api", views: 1000, score: 4.7 },
            { type: "Standard", views: 800, score: 4.5 },
        ];

        return trendingObjects.map((obj, index) => {
            const target = resourceResponseFactory.createMockData({
                overrides: {
                    __typename: obj.type,
                    views: obj.views,
                    score: obj.score,
                },
            });

            return this.createMockData({
                overrides: {
                    id: `trending_view_${index}`,
                    to: target,
                    lastViewedAt: new Date(Date.now() - (index * 30 * 60 * 1000)).toISOString(), // Recent views
                },
            });
        });
    }

    /**
     * Create recent views for a specific user
     */
    createRecentViewsForUser(userId: string, count = 10): View[] {
        const baseTime = Date.now();
        
        return Array.from({ length: count }, (_, index) => {
            const viewFor = VIEW_FOR_TYPES[index % VIEW_FOR_TYPES.length];
            let target: any;
            
            if (viewFor === "User") {
                target = userResponseFactory.createMockData();
            } else if (viewFor === "Team") {
                target = { __typename: "Team", id: generatePK().toString(), name: `Team ${index}` };
            } else {
                target = resourceResponseFactory.createMockData();
                target.__typename = viewFor;
            }

            return this.createMockData({
                overrides: {
                    id: `user_view_${userId}_${index}`,
                    by: userResponseFactory.createMockData({ overrides: { id: userId } }),
                    to: target,
                    lastViewedAt: new Date(baseTime - (index * 60 * 60 * 1000)).toISOString(), // 1 hour intervals
                },
            });
        });
    }

    /**
     * Create views by time period
     */
    createViewsByTimePeriod(period: "today" | "week" | "month"): View[] {
        const now = Date.now();
        const periods = {
            today: HOURS_24,
            week: DAYS_7,
            month: DAYS_30,
        };
        
        const timespan = periods[period];
        const count = period === "today" ? 20 : period === "week" ? 100 : 300;

        return Array.from({ length: count }, (_, index) => {
            const viewFor = VIEW_FOR_TYPES[index % VIEW_FOR_TYPES.length];
            let target: any;
            
            if (viewFor === "User") {
                target = userResponseFactory.createMockData();
            } else if (viewFor === "Team") {
                target = { __typename: "Team", id: generatePK().toString(), name: `Team ${index}` };
            } else {
                target = resourceResponseFactory.createMockData();
                target.__typename = viewFor;
            }

            return this.createMockData({
                overrides: {
                    id: `period_view_${period}_${index}`,
                    to: target,
                    lastViewedAt: new Date(now - (Math.random() * timespan)).toISOString(),
                    by: userResponseFactory.createMockData(),
                },
            });
        });
    }

    /**
     * Create views grouped by object type
     */
    createViewsByObjectType(objectType: ViewFor, count = 15): View[] {
        return Array.from({ length: count }, (_, index) => {
            let target: any;
            
            if (objectType === "User") {
                target = userResponseFactory.createMockData();
            } else if (objectType === "Team") {
                target = { __typename: "Team", id: generatePK().toString(), name: `Team ${index}` };
            } else {
                target = resourceResponseFactory.createMockData();
                target.__typename = objectType;
            }

            return this.createMockData({
                overrides: {
                    id: `type_view_${objectType}_${index}`,
                    to: target,
                    lastViewedAt: new Date(Date.now() - (index * 30 * 60 * 1000)).toISOString(),
                    by: userResponseFactory.createMockData(),
                },
            });
        });
    }

    /**
     * Create analytics views with patterns
     */
    createAnalyticsViews(): {
        hourlyViews: View[];
        dailyViews: View[];
        topViewedContent: View[];
    } {
        const now = Date.now();
        
        // Hourly views for the last 24 hours
        const hourlyViews = Array.from({ length: 24 }, (_, hour) => {
            const viewCount = Math.floor(Math.random() * 50) + 10; // 10-60 views per hour
            return Array.from({ length: viewCount }, (_, viewIndex) => 
                this.createMockData({
                    overrides: {
                        id: `hourly_${hour}_${viewIndex}`,
                        lastViewedAt: new Date(now - (hour * 60 * 60 * 1000) - (viewIndex * 60 * 1000)).toISOString(),
                        by: userResponseFactory.createMockData(),
                        to: resourceResponseFactory.createMockData(),
                    },
                }),
            );
        }).flat();

        // Daily views for the last 30 days
        const dailyViews = Array.from({ length: 30 }, (_, day) => {
            const viewCount = Math.floor(Math.random() * 200) + 50; // 50-250 views per day
            return Array.from({ length: viewCount }, (_, viewIndex) => 
                this.createMockData({
                    overrides: {
                        id: `daily_${day}_${viewIndex}`,
                        lastViewedAt: new Date(now - (day * HOURS_24) - (viewIndex * 60 * 1000)).toISOString(),
                        by: userResponseFactory.createMockData(),
                        to: resourceResponseFactory.createMockData(),
                    },
                }),
            );
        }).flat();

        // Top viewed content
        const topViewedContent = this.createTrendingViews();

        return {
            hourlyViews,
            dailyViews,
            topViewedContent,
        };
    }

    /**
     * Create duplicate view error response
     */
    createDuplicateViewErrorResponse(userId: string, objectId: string) {
        return this.createBusinessErrorResponse("duplicate", {
            resource: "view",
            userId,
            objectId,
            message: "View already recorded for this user and object",
        });
    }

    /**
     * Create rate limit error response
     */
    createRateLimitErrorResponse(limit: number, windowMs: number) {
        return this.createBusinessErrorResponse("rate_limit", {
            resource: "view",
            limit,
            windowMs,
            retryAfter: windowMs,
            message: `Too many views. Limit: ${limit} views per ${windowMs}ms`,
        });
    }

    /**
     * Create view analytics summary response
     */
    createViewAnalyticsSummary(objectId: string, period: "day" | "week" | "month" = "week") {
        const periods = {
            day: { views: 45, unique: 38 },
            week: { views: 287, unique: 152 },
            month: { views: 1234, unique: 456 },
        };

        const stats = periods[period];

        return {
            data: {
                objectId,
                period,
                totalViews: stats.views,
                uniqueViews: stats.unique,
                avgViewsPerDay: Math.round(stats.views / (period === "day" ? 1 : period === "week" ? 7 : 30)),
                topReferrers: [
                    { source: "direct", count: Math.round(stats.views * 0.4) },
                    { source: "search", count: Math.round(stats.views * 0.3) },
                    { source: "social", count: Math.round(stats.views * 0.2) },
                    { source: "other", count: Math.round(stats.views * 0.1) },
                ],
                timeRange: {
                    start: new Date(Date.now() - (period === "day" ? HOURS_24 : period === "week" ? DAYS_7 : DAYS_30)).toISOString(),
                    end: new Date().toISOString(),
                },
            },
            meta: {
                timestamp: new Date().toISOString(),
                requestId: generatePK().toString(),
                version: "1.0",
                links: {
                    self: `/api/view/analytics/${objectId}?period=${period}`,
                    detailed: `/api/view/analytics/${objectId}/detailed?period=${period}`,
                },
            },
        };
    }
}

/**
 * Pre-configured view response scenarios
 */
export const viewResponseScenarios = {
    // Success scenarios
    createSuccess: (input?: Partial<ViewCreateInput>) => {
        const factory = new ViewResponseFactory();
        const defaultInput: ViewCreateInput = {
            forConnect: generatePK().toString(),
            viewFor: "Routine",
            ...input,
        };
        return factory.createSuccessResponse(
            factory.createFromInput(defaultInput),
        );
    },

    findSuccess: (view?: View) => {
        const factory = new ViewResponseFactory();
        return factory.createSuccessResponse(
            view || factory.createMockData(),
        );
    },

    findCompleteSuccess: () => {
        const factory = new ViewResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({ scenario: "complete" }),
        );
    },

    updateSuccess: (existing?: View, updates?: Partial<ViewUpdateInput>) => {
        const factory = new ViewResponseFactory();
        const view = existing || factory.createMockData({ scenario: "complete" });
        const input: ViewUpdateInput = {
            id: view.id,
            ...updates,
        };
        return factory.createSuccessResponse(
            factory.updateFromInput(view, input),
        );
    },

    listSuccess: (views?: View[]) => {
        const factory = new ViewResponseFactory();
        return factory.createPaginatedResponse(
            views || Array.from({ length: DEFAULT_COUNT }, () => factory.createMockData()),
            { page: 1, totalCount: views?.length || DEFAULT_COUNT },
        );
    },

    allTypesSuccess: () => {
        const factory = new ViewResponseFactory();
        const views = factory.createViewsForAllTypes();
        return factory.createPaginatedResponse(
            views,
            { page: 1, totalCount: views.length },
        );
    },

    trendingSuccess: () => {
        const factory = new ViewResponseFactory();
        const views = factory.createTrendingViews();
        return factory.createPaginatedResponse(
            views,
            { page: 1, totalCount: views.length },
        );
    },

    recentViewsSuccess: (userId?: string) => {
        const factory = new ViewResponseFactory();
        const views = factory.createRecentViewsForUser(userId || generatePK().toString());
        return factory.createPaginatedResponse(
            views,
            { page: 1, totalCount: views.length },
        );
    },

    todayViewsSuccess: () => {
        const factory = new ViewResponseFactory();
        const views = factory.createViewsByTimePeriod("today");
        return factory.createPaginatedResponse(
            views,
            { page: 1, totalCount: views.length },
        );
    },

    weekViewsSuccess: () => {
        const factory = new ViewResponseFactory();
        const views = factory.createViewsByTimePeriod("week");
        return factory.createPaginatedResponse(
            views,
            { page: 1, totalCount: views.length },
        );
    },

    monthViewsSuccess: () => {
        const factory = new ViewResponseFactory();
        const views = factory.createViewsByTimePeriod("month");
        return factory.createPaginatedResponse(
            views,
            { page: 1, totalCount: views.length },
        );
    },

    byTypeSuccess: (objectType: ViewFor) => {
        const factory = new ViewResponseFactory();
        const views = factory.createViewsByObjectType(objectType);
        return factory.createPaginatedResponse(
            views,
            { page: 1, totalCount: views.length },
        );
    },

    analyticsSuccess: (objectId?: string, period: "day" | "week" | "month" = "week") => {
        const factory = new ViewResponseFactory();
        return factory.createViewAnalyticsSummary(objectId || generatePK().toString(), period);
    },

    // Error scenarios
    createValidationError: () => {
        const factory = new ViewResponseFactory();
        return factory.createValidationErrorResponse({
            forConnect: "Target object ID is required",
            viewFor: "View type must be specified",
        });
    },

    notFoundError: (viewId?: string) => {
        const factory = new ViewResponseFactory();
        return factory.createNotFoundErrorResponse(
            viewId || "non-existent-view",
        );
    },

    permissionError: (operation?: string) => {
        const factory = new ViewResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
            ["view:write"],
        );
    },

    duplicateViewError: (userId?: string, objectId?: string) => {
        const factory = new ViewResponseFactory();
        return factory.createDuplicateViewErrorResponse(
            userId || generatePK().toString(),
            objectId || generatePK().toString(),
        );
    },

    rateLimitError: (limit = 100, windowMs = 60000) => {
        const factory = new ViewResponseFactory();
        return factory.createRateLimitErrorResponse(limit, windowMs);
    },

    invalidObjectError: () => {
        const factory = new ViewResponseFactory();
        return factory.createValidationErrorResponse({
            forConnect: "Target object does not exist or is not viewable",
        });
    },

    invalidViewTypeError: () => {
        const factory = new ViewResponseFactory();
        return factory.createValidationErrorResponse({
            viewFor: "Invalid view type. Must be one of: " + VIEW_FOR_TYPES.join(", "),
        });
    },

    // MSW handlers
    handlers: {
        success: () => new ViewResponseFactory().createMSWHandlers(),
        withErrors: function createWithErrors(errorRate?: number) {
            return new ViewResponseFactory().createMSWHandlers({ errorRate: errorRate ?? DEFAULT_ERROR_RATE });
        },
        withDelay: function createWithDelay(delay?: number) {
            return new ViewResponseFactory().createMSWHandlers({ delay: delay ?? DEFAULT_DELAY_MS });
        },
    },
};

// Export factory instance for direct use
export const viewResponseFactory = new ViewResponseFactory();
