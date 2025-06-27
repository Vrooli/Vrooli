/* c8 ignore start */
/**
 * Award API Response Fixtures
 * 
 * Comprehensive fixtures for user achievement and recognition system including
 * badges, trophies, milestone tracking, and progress visualization.
 * 
 * Note: Awards are system-managed entities that are automatically granted
 * based on user achievements and cannot be directly created/updated/deleted.
 */

import type {
    Award,
    AwardSearchInput,
    AwardSearchResult,
    AwardCategory,
    AwardSortBy,
} from "../../../api/types.js";
import { BaseAPIResponseFactory } from "./base.js";
import type { MockDataOptions } from "./types.js";
import { generatePK } from "../../../id/index.js";

// Constants
const DEFAULT_COUNT = 10;
const DEFAULT_ERROR_RATE = 0.1;
const DEFAULT_DELAY_MS = 500;
const MAX_PROGRESS_VALUE = 10000;
const MINUTES_PER_HOUR = 60;
const SECONDS_PER_MINUTE = 60;
const MILLISECONDS_PER_SECOND = 1000;
const MILLISECONDS_PER_HOUR = MINUTES_PER_HOUR * SECONDS_PER_MINUTE * MILLISECONDS_PER_SECOND;
const HOURS_PER_DAY = 24;
const MILLISECONDS_PER_DAY = HOURS_PER_DAY * MILLISECONDS_PER_HOUR;
const HOURS_IN_2 = 2;
const DAYS_IN_1 = 1;
const DAYS_IN_7 = 7;
const DAYS_IN_30 = 30;

// Progression tiers for different award types
const TIER_1 = 1;
const TIER_3 = 3;
const TIER_5 = 5;
const TIER_7 = 7;
const TIER_10 = 10;
const TIER_14 = 14;
const TIER_25 = 25;
const TIER_30 = 30;
const TIER_50 = 50;
const TIER_60 = 60;
const TIER_100 = 100;
const TIER_180 = 180;
const TIER_250 = 250;
const TIER_365 = 365;
const TIER_500 = 500;
const TIER_1000 = 1000;
const TIER_2500 = 2500;
const TIER_5000 = 5000;

const PROGRESSION_TIERS = {
    CommentCreate: [TIER_1, TIER_5, TIER_10, TIER_25, TIER_50, TIER_100, TIER_250, TIER_500],
    ObjectBookmark: [TIER_1, TIER_10, TIER_25, TIER_50, TIER_100, TIER_250, TIER_500, TIER_1000],
    ObjectReact: [TIER_1, TIER_10, TIER_25, TIER_50, TIER_100, TIER_250, TIER_500, TIER_1000],
    RunRoutine: [TIER_1, TIER_5, TIER_10, TIER_25, TIER_50, TIER_100, TIER_250, TIER_500],
    RunProject: [TIER_1, TIER_3, TIER_5, TIER_10, TIER_25, TIER_50, TIER_100, TIER_250],
    UserInvite: [TIER_1, TIER_5, TIER_10, TIER_25, TIER_50, TIER_100],
    Reputation: [TIER_10, TIER_50, TIER_100, TIER_250, TIER_500, TIER_1000, TIER_2500, TIER_5000],
    Streak: [TIER_3, TIER_7, TIER_14, TIER_30, TIER_60, TIER_100, TIER_180, TIER_365],
    AccountNew: [TIER_1], // Special category - one-time award
} as const;

/**
 * Award API response factory
 */
export class AwardResponseFactory extends BaseAPIResponseFactory<
    Award,
    never, // Awards cannot be created via API
    never  // Awards cannot be updated via API
> {
    protected readonly entityName = "award";

    /**
     * Create mock award data
     */
    createMockData(options?: MockDataOptions): Award {
        const scenario = options?.scenario || "minimal";
        const now = new Date().toISOString();
        const awardId = options?.overrides?.id || generatePK().toString();

        // Select category and determine progress
        const categories = Object.keys(PROGRESSION_TIERS) as (keyof typeof PROGRESSION_TIERS)[];
        const category = categories[Math.floor(Math.random() * categories.length)]!;
        const tiers = PROGRESSION_TIERS[category];
        const tierIndex = Math.floor(Math.random() * tiers.length);
        const progress = tiers[tierIndex]!;

        const baseAward: Award = {
            __typename: "Award",
            id: awardId,
            created_at: now,
            updated_at: now,
            category: category as AwardCategory,
            progress,
            title: this.getAwardTitle(category as AwardCategory, progress),
            description: this.getAwardDescription(category as AwardCategory, progress),
            tierCompletedAt: scenario !== "minimal" ? now : null,
        };

        if (scenario === "complete" || scenario === "edge-case") {
            const recentDate = new Date(Date.now() - (DAYS_IN_7 * MILLISECONDS_PER_DAY)).toISOString();
            const oldDate = new Date(Date.now() - (DAYS_IN_30 * MILLISECONDS_PER_DAY)).toISOString();

            return {
                ...baseAward,
                category: scenario === "edge-case" ? "AccountNew" as AwardCategory : "Reputation" as AwardCategory,
                progress: scenario === "edge-case" ? 1 : MAX_PROGRESS_VALUE,
                title: scenario === "edge-case" 
                    ? "Welcome to the Community!" 
                    : "Reputation Master - Ultimate Achievement",
                description: scenario === "complete"
                    ? "Outstanding achievement! You've reached the highest tier of community reputation through consistent high-quality contributions."
                    : scenario === "edge-case"
                    ? "Welcome! You've successfully created your account and joined our community."
                    : null,
                tierCompletedAt: scenario === "edge-case" ? oldDate : recentDate,
                created_at: scenario === "edge-case" ? oldDate : 
                    new Date(Date.now() - (DAYS_IN_1 * MILLISECONDS_PER_DAY)).toISOString(),
                ...options?.overrides,
            };
        }

        return {
            ...baseAward,
            ...options?.overrides,
        };
    }

    /**
     * Awards cannot be created via API - throw error
     */
    createFromInput(): never {
        throw new Error("Awards are system-managed and cannot be created via API");
    }

    /**
     * Awards cannot be updated via API - throw error
     */
    updateFromInput(): never {
        throw new Error("Awards are system-managed and cannot be updated via API");
    }

    /**
     * Awards are read-only - no validation needed for create/update
     */
    async validateCreateInput(): Promise<{ valid: false; errors: Record<string, string> }> {
        return {
            valid: false,
            errors: { general: "Awards cannot be created via API" },
        };
    }

    async validateUpdateInput(): Promise<{ valid: false; errors: Record<string, string> }> {
        return {
            valid: false,
            errors: { general: "Awards cannot be updated via API" },
        };
    }

    /**
     * Create awards for all categories with realistic progression
     */
    createAwardsForAllCategories(): Award[] {
        return Object.entries(PROGRESSION_TIERS).flatMap(([category, tiers]) => {
            // Create awards for first 3 tiers to show progression
            const tierCount = Math.min(3, tiers.length);
            return Array.from({ length: tierCount }, (_, index) => {
                const progress = tiers[index]!;
                const daysAgo = (tierCount - index) * DAYS_IN_7;
                
                return this.createMockData({
                    overrides: {
                        id: `award_${category}_${progress}`,
                        category: category as AwardCategory,
                        progress,
                        tierCompletedAt: new Date(Date.now() - (daysAgo * MILLISECONDS_PER_DAY)).toISOString(),
                        created_at: new Date(Date.now() - ((daysAgo + 1) * MILLISECONDS_PER_DAY)).toISOString(),
                    },
                });
            });
        });
    }

    /**
     * Create achievement journey for a specific category
     */
    createAchievementJourney(category: AwardCategory, currentProgress: number): Award[] {
        const categoryKey = category as keyof typeof PROGRESSION_TIERS;
        const tiers = PROGRESSION_TIERS[categoryKey] || [1];
        const completedTiers = tiers.filter(tier => currentProgress >= tier);

        return completedTiers.map((progress, index) => {
            const daysAgo = (completedTiers.length - index) * DAYS_IN_7;
            
            return this.createMockData({
                overrides: {
                    id: `journey_${category}_${progress}`,
                    category,
                    progress,
                    tierCompletedAt: new Date(Date.now() - (daysAgo * MILLISECONDS_PER_DAY)).toISOString(),
                    created_at: new Date(Date.now() - ((daysAgo + 1) * MILLISECONDS_PER_DAY)).toISOString(),
                },
            });
        });
    }

    /**
     * Create recent achievements (last 30 days)
     */
    createRecentAchievements(count = 5): Award[] {
        const now = Date.now();
        const thirtyDaysAgo = now - (DAYS_IN_30 * MILLISECONDS_PER_DAY);

        return Array.from({ length: count }, (_, index) => {
            const achievedAt = new Date(thirtyDaysAgo + Math.random() * (now - thirtyDaysAgo));
            
            return this.createMockData({
                overrides: {
                    id: `recent_award_${index}`,
                    tierCompletedAt: achievedAt.toISOString(),
                    created_at: new Date(achievedAt.getTime() - (HOURS_IN_2 * MILLISECONDS_PER_HOUR)).toISOString(),
                    updated_at: achievedAt.toISOString(),
                },
            });
        }).sort((a, b) => 
            new Date(b.tierCompletedAt || b.created_at).getTime() - 
            new Date(a.tierCompletedAt || a.created_at).getTime(),
        );
    }

    /**
     * Create milestone achievements (high-tier awards)
     */
    createMilestoneAchievements(): Award[] {
        const milestones = [
            { category: "Reputation" as AwardCategory, progress: 5000 },
            { category: "RunRoutine" as AwardCategory, progress: 500 },
            { category: "RunProject" as AwardCategory, progress: 250 },
            { category: "Streak" as AwardCategory, progress: 365 },
            { category: "ObjectReact" as AwardCategory, progress: 1000 },
            { category: "CommentCreate" as AwardCategory, progress: 500 },
        ];

        return milestones.map(({ category, progress }, index) => {
            const daysAgo = Math.random() * DAYS_IN_30 * 3; // Random date in last 90 days
            
            return this.createMockData({
                overrides: {
                    id: `milestone_${category}_${progress}`,
                    category,
                    progress,
                    tierCompletedAt: new Date(Date.now() - (daysAgo * MILLISECONDS_PER_DAY)).toISOString(),
                    created_at: new Date(Date.now() - ((daysAgo + 1) * MILLISECONDS_PER_DAY)).toISOString(),
                },
            });
        });
    }

    /**
     * Create awards by category
     */
    createAwardsByCategory(category: AwardCategory, count = 3): Award[] {
        const categoryKey = category as keyof typeof PROGRESSION_TIERS;
        const tiers = PROGRESSION_TIERS[categoryKey] || [1];
        const tierCount = Math.min(count, tiers.length);

        return Array.from({ length: tierCount }, (_, index) => {
            const progress = tiers[index]!;
            const daysAgo = (tierCount - index) * DAYS_IN_7;
            
            return this.createMockData({
                overrides: {
                    id: `category_${category}_${progress}`,
                    category,
                    progress,
                    tierCompletedAt: new Date(Date.now() - (daysAgo * MILLISECONDS_PER_DAY)).toISOString(),
                },
            });
        });
    }

    /**
     * Create user progress awards showing advancement
     */
    createUserProgressAwards(): Award[] {
        const progressCategories = [
            { category: "CommentCreate" as AwardCategory, current: 47, tier: 50 },
            { category: "ObjectReact" as AwardCategory, current: 156, tier: 250 },
            { category: "RunRoutine" as AwardCategory, current: 23, tier: 25 },
            { category: "Streak" as AwardCategory, current: 12, tier: 14 },
        ];

        return progressCategories.map(({ category, current, tier }, index) => {
            const daysAgo = (index + 1) * DAYS_IN_7;
            const isCompleted = current >= tier;
            
            return this.createMockData({
                overrides: {
                    id: `progress_${category}_${current}`,
                    category,
                    progress: isCompleted ? tier : current,
                    tierCompletedAt: isCompleted ? 
                        new Date(Date.now() - (daysAgo * MILLISECONDS_PER_DAY)).toISOString() : 
                        null,
                    created_at: new Date(Date.now() - ((daysAgo + 1) * MILLISECONDS_PER_DAY)).toISOString(),
                },
            });
        });
    }

    /**
     * Create permission denied error for write operations
     */
    createWriteOperationErrorResponse(operation: string) {
        return this.createPermissionErrorResponse(
            operation,
            [],
            `Awards are system-managed and cannot be ${operation}d via API`,
        );
    }

    /**
     * Create award search result
     */
    createAwardSearchResult(awards: Award[], searchInput?: Partial<AwardSearchInput>): AwardSearchResult {
        // Apply search filters
        let filteredAwards = [...awards];

        if (searchInput?.ids && searchInput.ids.length > 0) {
            filteredAwards = filteredAwards.filter(award => searchInput.ids!.includes(award.id));
        }

        // Apply sorting
        if (searchInput?.sortBy) {
            filteredAwards.sort((a, b) => {
                switch (searchInput.sortBy) {
                    case "DateUpdatedAsc" as AwardSortBy:
                        return new Date(a.updated_at).getTime() - new Date(b.updated_at).getTime();
                    case "DateUpdatedDesc" as AwardSortBy:
                        return new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime();
                    case "ProgressAsc" as AwardSortBy:
                        return a.progress - b.progress;
                    case "ProgressDesc" as AwardSortBy:
                        return b.progress - a.progress;
                    default:
                        return 0;
                }
            });
        }

        // Apply take limit
        if (searchInput?.take && searchInput.take > 0) {
            filteredAwards = filteredAwards.slice(0, searchInput.take);
        }

        return {
            __typename: "AwardSearchResult",
            edges: filteredAwards.map(award => ({
                __typename: "AwardEdge",
                cursor: btoa(`award:${award.id}`),
                node: award,
            })),
            pageInfo: {
                __typename: "PageInfo",
                hasNextPage: false, // Simplified for fixtures
                hasPreviousPage: false,
                startCursor: filteredAwards.length > 0 ? btoa(`award:${filteredAwards[0]!.id}`) : null,
                endCursor: filteredAwards.length > 0 ? 
                    btoa(`award:${filteredAwards[filteredAwards.length - 1]!.id}`) : null,
            },
        };
    }

    /**
     * Get award title based on category and progress
     */
    private getAwardTitle(category: AwardCategory, progress: number): string {
        const titles = {
            CommentCreate: (p: number) => `Commentator ${this.getTierName(p, PROGRESSION_TIERS.CommentCreate)}`,
            ObjectBookmark: (p: number) => `Collector ${this.getTierName(p, PROGRESSION_TIERS.ObjectBookmark)}`,
            ObjectReact: (p: number) => `Reactor ${this.getTierName(p, PROGRESSION_TIERS.ObjectReact)}`,
            RunRoutine: (p: number) => `Routine Runner ${this.getTierName(p, PROGRESSION_TIERS.RunRoutine)}`,
            RunProject: (p: number) => `Project Master ${this.getTierName(p, PROGRESSION_TIERS.RunProject)}`,
            UserInvite: (p: number) => `Community Builder ${this.getTierName(p, PROGRESSION_TIERS.UserInvite)}`,
            Reputation: (p: number) => `Reputation ${this.getTierName(p, PROGRESSION_TIERS.Reputation)}`,
            Streak: (p: number) => `${p} Day Streak`,
            AccountNew: () => "Welcome to the Community!",
        };

        const titleFunction = titles[category as keyof typeof titles];
        return titleFunction ? titleFunction(progress) : `${category} Achievement`;
    }

    /**
     * Get award description based on category and progress
     */
    private getAwardDescription(category: AwardCategory, progress: number): string {
        const descriptions = {
            CommentCreate: (p: number) => `Created ${p} helpful comments that contribute to the community discussion.`,
            ObjectBookmark: (p: number) => `Bookmarked ${p} valuable resources for future reference.`,
            ObjectReact: (p: number) => `Reacted to ${p} posts, showing engagement with the community.`,
            RunRoutine: (p: number) => `Successfully completed ${p} routine executions.`,
            RunProject: (p: number) => `Completed ${p} project runs, demonstrating execution excellence.`,
            UserInvite: (p: number) => `Invited ${p} new members to join the community.`,
            Reputation: (p: number) => `Earned ${p} reputation points through quality contributions.`,
            Streak: (p: number) => `Maintained a ${p}-day activity streak.`,
            AccountNew: () => "Successfully created an account and joined the community.",
        };

        const descFunction = descriptions[category as keyof typeof descriptions];
        return descFunction ? descFunction(progress) : `Achieved ${progress} in ${category}.`;
    }

    /**
     * Get tier name based on progress
     */
    private getTierName(progress: number, tiers: readonly number[]): string {
        const tierIndex = tiers.findIndex(tier => progress <= tier);
        const tierNames = ["Bronze", "Silver", "Gold", "Platinum", "Diamond", "Master", "Grandmaster", "Legend"];
        return tierNames[tierIndex] || "Ultimate";
    }
}

/**
 * Pre-configured award response scenarios
 */
export const awardResponseScenarios = {
    // Success scenarios
    findSuccess: (award?: Award) => {
        const factory = new AwardResponseFactory();
        return factory.createSuccessResponse(
            award || factory.createMockData(),
        );
    },

    findCompleteSuccess: () => {
        const factory = new AwardResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({ scenario: "complete" }),
        );
    },

    listSuccess: (awards?: Award[]) => {
        const factory = new AwardResponseFactory();
        return factory.createPaginatedResponse(
            awards || Array.from({ length: DEFAULT_COUNT }, () => factory.createMockData()),
            { page: 1, totalCount: awards?.length || DEFAULT_COUNT },
        );
    },

    allCategoriesSuccess: () => {
        const factory = new AwardResponseFactory();
        const awards = factory.createAwardsForAllCategories();
        return factory.createPaginatedResponse(
            awards,
            { page: 1, totalCount: awards.length },
        );
    },

    recentAchievementsSuccess: (count?: number) => {
        const factory = new AwardResponseFactory();
        const awards = factory.createRecentAchievements(count);
        return factory.createPaginatedResponse(
            awards,
            { page: 1, totalCount: awards.length },
        );
    },

    milestoneAchievementsSuccess: () => {
        const factory = new AwardResponseFactory();
        const awards = factory.createMilestoneAchievements();
        return factory.createPaginatedResponse(
            awards,
            { page: 1, totalCount: awards.length },
        );
    },

    categoryAwardsSuccess: (category: AwardCategory, count?: number) => {
        const factory = new AwardResponseFactory();
        const awards = factory.createAwardsByCategory(category, count);
        return factory.createPaginatedResponse(
            awards,
            { page: 1, totalCount: awards.length },
        );
    },

    achievementJourneySuccess: (category: AwardCategory, progress: number) => {
        const factory = new AwardResponseFactory();
        const awards = factory.createAchievementJourney(category, progress);
        return factory.createPaginatedResponse(
            awards,
            { page: 1, totalCount: awards.length },
        );
    },

    userProgressSuccess: () => {
        const factory = new AwardResponseFactory();
        const awards = factory.createUserProgressAwards();
        return factory.createPaginatedResponse(
            awards,
            { page: 1, totalCount: awards.length },
        );
    },

    searchSuccess: (searchInput?: Partial<AwardSearchInput>) => {
        const factory = new AwardResponseFactory();
        const awards = factory.createAwardsForAllCategories();
        return factory.createAwardSearchResult(awards, searchInput);
    },

    // Error scenarios
    notFoundError: (awardId?: string) => {
        const factory = new AwardResponseFactory();
        return factory.createNotFoundErrorResponse(
            awardId || "non-existent-award",
        );
    },

    createPermissionError: () => {
        const factory = new AwardResponseFactory();
        return factory.createWriteOperationErrorResponse("create");
    },

    updatePermissionError: () => {
        const factory = new AwardResponseFactory();
        return factory.createWriteOperationErrorResponse("update");
    },

    deletePermissionError: () => {
        const factory = new AwardResponseFactory();
        return factory.createWriteOperationErrorResponse("delete");
    },

    // MSW handlers
    handlers: {
        success: () => new AwardResponseFactory().createMSWHandlers(),
        withErrors: function createWithErrors(errorRate?: number) {
            return new AwardResponseFactory().createMSWHandlers({ errorRate: errorRate ?? DEFAULT_ERROR_RATE });
        },
        withDelay: function createWithDelay(delay?: number) {
            return new AwardResponseFactory().createMSWHandlers({ delay: delay ?? DEFAULT_DELAY_MS });
        },
    },
};

// Export factory instance for direct use
export const awardResponseFactory = new AwardResponseFactory();

