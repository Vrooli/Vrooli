/**
 * Award API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for award endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 * 
 * Note: Awards are read-only entities in the system - they are granted automatically
 * based on user achievements and cannot be created/updated/deleted via API.
 */
// AI_CHECK: TYPE_SAFETY=fixed-switch-statement-comparison-error | LAST: 2025-07-02 - Fixed TypeScript comparison error in switch statement by replacing with if-else

import { http, type HttpHandler } from "msw";
import type { 
    Award,
    AwardSearchInput,
    AwardSearchResult,
    User,
} from "@vrooli/shared";
import { awardNames, awardVariants , 
    AwardCategory,
    AwardSortBy} from "@vrooli/shared";

/**
 * Standard API response wrapper
 */
export interface APIResponse<T> {
    data: T;
    meta: {
        timestamp: string;
        requestId: string;
        version: string;
        links?: {
            self?: string;
            related?: Record<string, string>;
        };
    };
}

/**
 * API error response structure
 */
export interface APIErrorResponse {
    error: {
        code: string;
        message: string;
        details?: Record<string, any>;
        timestamp: string;
        requestId: string;
        path: string;
    };
}

/**
 * Paginated response structure
 */
export interface PaginatedAPIResponse<T> extends APIResponse<T[]> {
    pagination: {
        page: number;
        pageSize: number;
        totalCount: number;
        totalPages: number;
        hasNextPage: boolean;
        hasPreviousPage: boolean;
    };
}

/**
 * Award API response factory
 */
export class AwardResponseFactory {
    private readonly baseUrl: string;
    
    constructor(baseUrl: string = process.env.VITE_SERVER_URL || "http://localhost:5329") {
        this.baseUrl = baseUrl;
    }
    
    /**
     * Generate unique request ID
     */
    private generateRequestId(): string {
        return `req_${Date.now()}_${Math.random().toString(36).substring(2, 11)}`;
    }
    
    /**
     * Generate unique resource ID
     */
    private generateId(): string {
        return `award_${Date.now()}_${Math.random().toString(36).substring(2, 11)}`;
    }
    
    /**
     * Create successful award response
     */
    createSuccessResponse(award: Award): APIResponse<Award> {
        return {
            data: award,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/award/${award.id}`,
                    related: {
                        user: `${this.baseUrl}/api/user/${award.id.split("_")[0]}`,
                    },
                },
            },
        };
    }
    
    /**
     * Create award search result response
     */
    createAwardSearchResponse(awards: Award[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): AwardSearchResult {
        const paginationData = pagination || {
            page: 1,
            pageSize: awards.length,
            totalCount: awards.length,
        };
        
        return {
            __typename: "AwardSearchResult",
            edges: awards.map(award => ({
                __typename: "AwardEdge",
                cursor: btoa(`award:${award.id}`),
                node: award,
            })),
            pageInfo: {
                __typename: "PageInfo",
                hasNextPage: paginationData.page * paginationData.pageSize < paginationData.totalCount,
                endCursor: awards.length > 0 ? btoa(`award:${awards[awards.length - 1]!.id}`) : null,
            },
        };
    }
    
    /**
     * Create paginated award list response
     */
    createAwardListResponse(awards: Award[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<Award> {
        const paginationData = pagination || {
            page: 1,
            pageSize: awards.length,
            totalCount: awards.length,
        };
        
        return {
            data: awards,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/awards?page=${paginationData.page}&limit=${paginationData.pageSize}`,
                },
            },
            pagination: {
                ...paginationData,
                totalPages: Math.ceil(paginationData.totalCount / paginationData.pageSize),
                hasNextPage: paginationData.page * paginationData.pageSize < paginationData.totalCount,
                hasPreviousPage: paginationData.page > 1,
            },
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(awardId: string): APIErrorResponse {
        return {
            error: {
                code: "AWARD_NOT_FOUND",
                message: `Award with ID '${awardId}' was not found`,
                details: {
                    awardId,
                    searchCriteria: { id: awardId },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/award/${awardId}`,
            },
        };
    }
    
    /**
     * Create permission error response
     */
    createPermissionErrorResponse(operation: string): APIErrorResponse {
        return {
            error: {
                code: "PERMISSION_DENIED",
                message: `You do not have permission to ${operation} awards`,
                details: {
                    operation,
                    reason: "Awards are system-managed and cannot be directly modified",
                    allowedOperations: ["view", "list"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/awards",
            },
        };
    }
    
    /**
     * Create network error response
     */
    createNetworkErrorResponse(): APIErrorResponse {
        return {
            error: {
                code: "NETWORK_ERROR",
                message: "Network request failed",
                details: {
                    reason: "Connection timeout",
                    retryable: true,
                    retryAfter: 5000,
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/awards",
            },
        };
    }
    
    /**
     * Create server error response
     */
    createServerErrorResponse(): APIErrorResponse {
        return {
            error: {
                code: "INTERNAL_SERVER_ERROR",
                message: "An unexpected server error occurred",
                details: {
                    errorId: `ERR_${Date.now()}`,
                    reportable: true,
                    retryable: true,
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/awards",
            },
        };
    }
    
    /**
     * Create rate limit error response
     */
    createRateLimitErrorResponse(): APIErrorResponse {
        return {
            error: {
                code: "RATE_LIMIT_EXCEEDED",
                message: "Too many requests. Please slow down.",
                details: {
                    limit: 100,
                    remaining: 0,
                    resetTime: new Date(Date.now() + 60000).toISOString(),
                    retryAfter: 60,
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/awards",
            },
        };
    }
    
    /**
     * Create mock award data with realistic achievement information
     */
    createMockAward(overrides?: Partial<Award>): Award {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        // Random category selection with weighted distribution toward common achievements
        const categories = Object.values(AwardCategory);
        const commonCategories = [
            AwardCategory.CommentCreate,
            AwardCategory.ObjectBookmark,
            AwardCategory.ObjectReact,
            AwardCategory.RunRoutine,
            AwardCategory.UserInvite,
        ];
        const isCommon = Math.random() < 0.7;
        const selectedCategories = isCommon ? commonCategories : categories;
        const category = selectedCategories[Math.floor(Math.random() * selectedCategories.length)]!;
        
        // Generate realistic progress based on category
        let progress: number;
        if (category in awardVariants) {
            const variants = awardVariants[category as keyof typeof awardVariants];
            // Select a random tier from the variants
            const tierIndex = Math.floor(Math.random() * variants.length);
            progress = variants[tierIndex] || 1;
        } else {
            // For special categories without variants
            progress = category === AwardCategory.AccountNew ? 1 : Math.floor(Math.random() * 365) + 1;
        }
        
        // Get award details from the naming system
        const awardDetails = awardNames[category](progress);
        
        const defaultAward: Award = {
            __typename: "Award",
            id,
            createdAt: new Date(Date.now() - Math.random() * 365 * 24 * 60 * 60 * 1000).toISOString(), // Random date in past year
            updatedAt: now,
            category,
            progress,
            title: awardDetails.name || `${category} Achievement`,
            description: awardDetails.body ? `Achievement unlocked: ${awardDetails.body}` : `You've achieved level ${awardDetails.level} in ${category}`,
            tierCompletedAt: progress >= (awardVariants[category as keyof typeof awardVariants]?.[0] || 1) ? 
                new Date(Date.now() - Math.random() * 30 * 24 * 60 * 60 * 1000).toISOString() : // Random date in past month
                null,
        };
        
        return {
            ...defaultAward,
            ...overrides,
        };
    }
    
    /**
     * Create awards for all categories with realistic progression
     */
    createAwardsForAllCategories(): Award[] {
        return Object.values(AwardCategory).flatMap((category: AwardCategory) => {
            // Create multiple tiers for categories with variants
            if (category in awardVariants) {
                const variants = awardVariants[category as keyof typeof awardVariants];
                // Create awards for first 3 tiers (or all if less than 3)
                const tierCount = Math.min(3, variants.length);
                return Array.from({ length: tierCount }, (_, index) => {
                    const progress = variants[index] || 1;
                    return this.createMockAward({ 
                        category, 
                        progress,
                        tierCompletedAt: new Date(Date.now() - (tierCount - index) * 7 * 24 * 60 * 60 * 1000).toISOString(),
                    });
                });
            } else {
                // Special categories
                const progress = category === AwardCategory.AccountNew ? 1 : Math.floor(Math.random() * 5) + 1;
                return [this.createMockAward({ category, progress })];
            }
        });
    }
    
    /**
     * Create achievement journey - progressive awards for a single category
     */
    createAchievementJourney(category: AwardCategory, currentProgress: number): Award[] {
        if (!(category in awardVariants)) {
            return [this.createMockAward({ category, progress: currentProgress })];
        }
        
        const variants = awardVariants[category as keyof typeof awardVariants];
        const completedTiers = variants.filter(tier => currentProgress >= tier);
        
        return completedTiers.map((progress, index) => 
            this.createMockAward({ 
                category, 
                progress,
                tierCompletedAt: new Date(Date.now() - (completedTiers.length - index) * 14 * 24 * 60 * 60 * 1000).toISOString(),
                createdAt: new Date(Date.now() - (completedTiers.length - index + 1) * 14 * 24 * 60 * 60 * 1000).toISOString(),
            }),
        );
    }
    
    /**
     * Create recent achievements (last 30 days)
     */
    createRecentAchievements(count = 5): Award[] {
        const now = Date.now();
        const thirtyDaysAgo = now - (30 * 24 * 60 * 60 * 1000);
        
        return Array.from({ length: count }, (_, index) => {
            const achievedAt = new Date(thirtyDaysAgo + Math.random() * (now - thirtyDaysAgo));
            return this.createMockAward({
                tierCompletedAt: achievedAt.toISOString(),
                createdAt: achievedAt.toISOString(),
                updatedAt: achievedAt.toISOString(),
            });
        }).sort((a, b) => new Date(b.tierCompletedAt || b.createdAt).getTime() - new Date(a.tierCompletedAt || a.createdAt).getTime());
    }
    
    /**
     * Create milestone achievements - high-tier awards
     */
    createMilestoneAchievements(): Award[] {
        const milestoneCategories = [
            { category: AwardCategory.Reputation, progress: 1000 },
            { category: AwardCategory.RunRoutine, progress: 1000 },
            { category: AwardCategory.RunProject, progress: 500 },
            { category: AwardCategory.Streak, progress: 365 },
            { category: AwardCategory.CommentCreate, progress: 500 },
        ];
        
        return milestoneCategories.map(({ category, progress }) => 
            this.createMockAward({ 
                category, 
                progress,
                tierCompletedAt: new Date(Date.now() - Math.random() * 90 * 24 * 60 * 60 * 1000).toISOString(),
            }),
        );
    }
    
    /**
     * Filter awards by search criteria
     */
    filterAwards(awards: Award[], searchInput: Partial<AwardSearchInput>): Award[] {
        let filtered = [...awards];
        
        // Filter by IDs if provided
        if (searchInput.ids && searchInput.ids.length > 0) {
            filtered = filtered.filter(award => searchInput.ids!.includes(award.id));
        }
        
        // Sort awards
        if (searchInput.sortBy) {
            const sortBy = searchInput.sortBy;
            filtered.sort((awardA, awardB) => {
                switch (String(sortBy)) {
                    case AwardSortBy.DateUpdatedAsc:
                        return new Date(awardA.updatedAt).getTime() - new Date(awardB.updatedAt).getTime();
                    case AwardSortBy.DateUpdatedDesc:
                        return new Date(awardB.updatedAt).getTime() - new Date(awardA.updatedAt).getTime();
                    case AwardSortBy.ProgressAsc:
                        return awardA.progress - awardB.progress;
                    case AwardSortBy.ProgressDesc:
                        return awardB.progress - awardA.progress;
                    default:
                        return 0;
                }
            });
        }
        
        // Apply take limit
        if (searchInput.take && searchInput.take > 0) {
            filtered = filtered.slice(0, searchInput.take);
        }
        
        return filtered;
    }
}

/**
 * MSW handlers factory for award endpoints
 */
export class AwardMSWHandlers {
    private responseFactory: AwardResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new AwardResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all award endpoints
     */
    createSuccessHandlers(): HttpHandler[] {
        return [
            // Get award by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/award/:id`, ({ params }) => {
                const { id } = params;
                
                const award = this.responseFactory.createMockAward({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(award);
                
                return Response.json(response, { status: 200 });
            }),
            
            // Search awards
            http.post(`${this.responseFactory["baseUrl"]}/api/awards`, async ({ request }) => {
                const searchInput = await request.json() as AwardSearchInput;
                
                // Generate base awards set
                let awards = this.responseFactory.createAwardsForAllCategories();
                
                // Apply search filters
                awards = this.responseFactory.filterAwards(awards, searchInput);
                
                const response = this.responseFactory.createAwardSearchResponse(awards);
                
                return Response.json(response, { status: 200 });
            }),
            
            // List user awards (simplified endpoint)
            http.get(`${this.responseFactory["baseUrl"]}/api/awards`, ({ request }) => {
                const url = new URL(request.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const category = url.searchParams.get("category") as AwardCategory | null;
                const recent = url.searchParams.get("recent") === "true";
                const milestones = url.searchParams.get("milestones") === "true";
                
                let awards: Award[];
                
                if (recent) {
                    awards = this.responseFactory.createRecentAchievements(limit);
                } else if (milestones) {
                    awards = this.responseFactory.createMilestoneAchievements();
                } else {
                    awards = this.responseFactory.createAwardsForAllCategories();
                }
                
                // Filter by category if specified
                if (category) {
                    awards = awards.filter(award => award.category === category);
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedAwards = awards.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createAwardListResponse(
                    paginatedAwards,
                    {
                        page,
                        pageSize: limit,
                        totalCount: awards.length,
                    },
                );
                
                return Response.json(response, { status: 200 });
            }),
            
            // Get achievement journey for a specific category
            http.get(`${this.responseFactory["baseUrl"]}/api/awards/journey/:category`, ({ params, request }) => {
                const { category } = params;
                const url = new URL(request.url);
                const progress = parseInt(url.searchParams.get("progress") || "100");
                
                const awards = this.responseFactory.createAchievementJourney(
                    category as AwardCategory, 
                    progress,
                );
                
                const response = this.responseFactory.createAwardListResponse(awards);
                
                return Response.json(response, { status: 200 });
            }),
        ];
    }
    
    /**
     * Create error handlers for testing error scenarios
     */
    createErrorHandlers(): HttpHandler[] {
        return [
            // Not found error
            http.get(`${this.responseFactory["baseUrl"]}/api/award/:id`, ({ params }) => {
                const { id } = params;
                return Response.json(
                    this.responseFactory.createNotFoundErrorResponse(id as string),
                    { status: 404 }
                );
            }),
            
            // Permission error for write operations
            http.post(`${this.responseFactory["baseUrl"]}/api/award`, () => {
                return Response.json(
                    this.responseFactory.createPermissionErrorResponse("create"),
                    { status: 403 }
                );
            }),
            
            http.put(`${this.responseFactory["baseUrl"]}/api/award/:id`, () => {
                return Response.json(
                    this.responseFactory.createPermissionErrorResponse("update"),
                    { status: 403 }
                );
            }),
            
            http.delete(`${this.responseFactory["baseUrl"]}/api/award/:id`, () => {
                return Response.json(
                    this.responseFactory.createPermissionErrorResponse("delete"),
                    { status: 403 }
                );
            }),
            
            // Server error
            http.get(`${this.responseFactory["baseUrl"]}/api/awards`, () => {
                return Response.json(
                    this.responseFactory.createServerErrorResponse(),
                    { status: 500 }
                );
            }),
            
            // Rate limit error
            http.post(`${this.responseFactory["baseUrl"]}/api/awards`, () => {
                return Response.json(
                    this.responseFactory.createRateLimitErrorResponse(),
                    { status: 429 }
                );
            }),
        ];
    }
    
    /**
     * Create loading simulation handlers
     */
    createLoadingHandlers(delay = 2000): HttpHandler[] {
        return [
            http.get(`${this.responseFactory["baseUrl"]}/api/awards`, async () => {
                await new Promise(resolve => setTimeout(resolve, delay));
                const awards = this.responseFactory.createAwardsForAllCategories();
                const response = this.responseFactory.createAwardListResponse(awards);
                
                return Response.json(response, { status: 200 });
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/award/:id`, async ({ params }) => {
                await new Promise(resolve => setTimeout(resolve, delay));
                const { id } = params;
                const award = this.responseFactory.createMockAward({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(award);
                
                return Response.json(response, { status: 200 });
            }),
        ];
    }
    
    /**
     * Create network error handlers
     */
    createNetworkErrorHandlers(): HttpHandler[] {
        return [
            http.get(`${this.responseFactory["baseUrl"]}/api/awards`, () => {
                return Response.error();
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/award/:id`, () => {
                return Response.error();
            }),
            
            http.post(`${this.responseFactory["baseUrl"]}/api/awards`, () => {
                return Response.error();
            }),
        ];
    }
    
    /**
     * Create custom handler with specific configuration
     */
    createCustomHandler(config: {
        endpoint: string;
        method: "GET" | "POST" | "PUT" | "DELETE";
        status: number;
        response: unknown;
        delay?: number;
    }): HttpHandler {
        const { endpoint, method, status, response, delay } = config;
        const fullEndpoint = `${this.responseFactory["baseUrl"]}${endpoint}`;
        
        const httpMethod = method.toLowerCase() as keyof typeof http;
        return http[httpMethod](fullEndpoint, async () => {
            if (delay) {
                await new Promise(resolve => setTimeout(resolve, delay));
            }
            
            return Response.json(response, { status });
        });
    }
}

/**
 * Pre-configured response scenarios
 */
export const awardResponseScenarios = {
    // Success scenarios
    singleAward: (award?: Award) => {
        const factory = new AwardResponseFactory();
        return factory.createSuccessResponse(
            award || factory.createMockAward(),
        );
    },
    
    allCategories: () => {
        const factory = new AwardResponseFactory();
        return factory.createAwardListResponse(
            factory.createAwardsForAllCategories(),
        );
    },
    
    recentAchievements: (count?: number) => {
        const factory = new AwardResponseFactory();
        return factory.createAwardListResponse(
            factory.createRecentAchievements(count),
        );
    },
    
    milestoneAchievements: () => {
        const factory = new AwardResponseFactory();
        return factory.createAwardListResponse(
            factory.createMilestoneAchievements(),
        );
    },
    
    achievementJourney: (category: AwardCategory, progress: number) => {
        const factory = new AwardResponseFactory();
        return factory.createAwardListResponse(
            factory.createAchievementJourney(category, progress),
        );
    },
    
    // Error scenarios
    notFoundError: (awardId?: string) => {
        const factory = new AwardResponseFactory();
        return factory.createNotFoundErrorResponse(
            awardId || "non-existent-award-id",
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new AwardResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "modify",
        );
    },
    
    serverError: () => {
        const factory = new AwardResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    rateLimitError: () => {
        const factory = new AwardResponseFactory();
        return factory.createRateLimitErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new AwardMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new AwardMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new AwardMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new AwardMSWHandlers().createNetworkErrorHandlers(),
};

// Export factory instances for easy use
export const awardResponseFactory = new AwardResponseFactory();
export const awardMSWHandlers = new AwardMSWHandlers();
