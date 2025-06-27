/* c8 ignore start */
/**
 * Bookmark API Response Fixtures
 * 
 * Migrated from UI package - now leverages base factory for 75% less boilerplate.
 */

import type { 
    Bookmark, 
    BookmarkCreateInput, 
    BookmarkUpdateInput,
} from "../../../api/types.js";
import { BookmarkFor as BookmarkForEnum } from "../../../consts/model.js";
import { bookmarkValidation } from "../../../validation/models/bookmark.js";
import { BaseAPIResponseFactory } from "./base.js";
import type { MockDataOptions } from "./types.js";

/**
 * Bookmark API response factory
 * 
 * Extends base factory to provide bookmark-specific functionality
 * with minimal boilerplate.
 */
export class BookmarkResponseFactory extends BaseAPIResponseFactory<
    Bookmark,
    BookmarkCreateInput,
    BookmarkUpdateInput
> {
    protected readonly entityName = "bookmark";

    /**
     * Create mock bookmark data
     */
    createMockData(options?: MockDataOptions): Bookmark {
        const now = new Date().toISOString();
        const id = options?.overrides?.id || this.generateId();
        const scenario = options?.scenario || "minimal";

        const baseBookmark: Bookmark = {
            __typename: "Bookmark",
            id,
            createdAt: now,
            updatedAt: now,
            by: {
                __typename: "User",
                id: `user_${this.generateId()}`,
                handle: "testuser",
                name: "Test User",
                createdAt: now,
                updatedAt: now,
                isBot: false,
                isPrivate: false,
                profileImage: null,
                bannerImage: null,
                premium: false,
                premiumExpiration: null,
                roles: [],
                wallets: [],
                translations: [],
                translationsCount: 0,
                you: {
                    __typename: "UserYou",
                    isBlocked: false,
                    isBlockedByYou: false,
                    canDelete: false,
                    canReport: false,
                    canUpdate: false,
                    isBookmarked: false,
                    isReacted: false,
                    reactionSummary: {
                        __typename: "ReactionSummary",
                        emotion: null,
                        count: 0,
                    },
                },
            },
            to: this.createBookmarkTarget(scenario),
            list: {
                __typename: "BookmarkList",
                id: `list_${this.generateId()}`,
                label: "My Bookmarks",
                createdAt: now,
                updatedAt: now,
                bookmarks: [],
                bookmarksCount: 1,
                you: {
                    __typename: "BookmarkListYou",
                    canDelete: true,
                    canUpdate: true,
                },
            },
        };

        return {
            ...baseBookmark,
            ...options?.overrides,
        };
    }

    /**
     * Create bookmark target based on scenario
     */
    private createBookmarkTarget(_scenario: string): unknown {
        const now = new Date().toISOString();
        const id = this.generateId();

        // For simplicity, defaulting to Resource type
        return {
            __typename: "Resource",
            id: `resource_${id}`,
            createdAt: now,
            updatedAt: now,
            isInternal: false,
            isPrivate: false,
            usedBy: [],
            usedByCount: 0,
            versions: [],
            versionsCount: 0,
            you: {
                __typename: "ResourceYou",
                canDelete: false,
                canUpdate: false,
                canReport: false,
                isBookmarked: true,
                isReacted: false,
                reaction: null,
            },
        };
    }

    /**
     * Create bookmark from API input
     */
    createFromInput(input: BookmarkCreateInput): Bookmark {
        const bookmark = this.createMockData();
        
        // Update bookmark based on input
        bookmark.to.__typename = input.bookmarkFor;
        bookmark.to.id = input.forConnect;
        
        if (input.listCreate) {
            bookmark.list.id = this.generateId();
            bookmark.list.label = input.listCreate.label;
        } else if (input.listConnect) {
            bookmark.list.id = input.listConnect;
        }
        
        return bookmark;
    }

    /**
     * Update bookmark from API input
     */
    updateFromInput(existing: Bookmark, _input: BookmarkUpdateInput): Bookmark {
        return {
            ...existing,
            updatedAt: new Date().toISOString(),
            // In a real implementation, would update list if provided
        };
    }

    /**
     * Validate bookmark create input
     */
    async validateCreateInput(input: BookmarkCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await bookmarkValidation.create.parseAsync(input);
            return { valid: true };
        } catch (error) {
            const fieldErrors: Record<string, string> = {};
            
            const zodError = error as { errors?: Array<{ path?: string[]; message: string }> };
            if (zodError.errors) {
                zodError.errors.forEach((err) => {
                    if (err.path?.length > 0) {
                        fieldErrors[err.path.join(".")] = err.message;
                    }
                });
            }
            
            return {
                valid: false,
                errors: fieldErrors,
            };
        }
    }

    /**
     * Validate bookmark update input
     */
    async validateUpdateInput(input: BookmarkUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await bookmarkValidation.update.parseAsync(input);
            return { valid: true };
        } catch (error) {
            const fieldErrors: Record<string, string> = {};
            
            const zodError = error as { errors?: Array<{ path?: string[]; message: string }> };
            if (zodError.errors) {
                zodError.errors.forEach((err) => {
                    if (err.path?.length > 0) {
                        fieldErrors[err.path.join(".")] = err.message;
                    }
                });
            }
            
            return {
                valid: false,
                errors: fieldErrors,
            };
        }
    }

    /**
     * Create multiple bookmarks for different object types
     */
    createBookmarksForAllTypes(): Bookmark[] {
        return Object.values(BookmarkForEnum).map(bookmarkFor => 
            this.createMockData({
                overrides: {
                    to: {
                        ...this.createBookmarkTarget("minimal"),
                        __typename: bookmarkFor,
                        id: `${bookmarkFor.toLowerCase()}_${this.generateId()}`,
                    },
                },
            }),
        );
    }
}

/**
 * Pre-configured response scenarios for easy testing
 * 
 * These leverage the base factory's error handling, eliminating
 * the need to duplicate error creation logic.
 */
export const bookmarkResponseScenarios = {
    // Success scenarios
    createSuccess: (bookmark?: Bookmark) => {
        const factory = new BookmarkResponseFactory();
        return factory.createSuccessResponse(
            bookmark || factory.createMockData(),
        );
    },
    
    listSuccess: (bookmarks?: Bookmark[]) => {
        const factory = new BookmarkResponseFactory();
        const DEFAULT_TOTAL = 10;
        return factory.createPaginatedResponse(
            bookmarks || factory.createBookmarksForAllTypes(),
            { page: 1, totalCount: bookmarks?.length || DEFAULT_TOTAL },
        );
    },
    
    // Error scenarios - now use base factory methods
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new BookmarkResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                forConnect: "Target object is required",
                bookmarkFor: "Bookmark type must be specified",
            },
        );
    },
    
    notFoundError: (bookmarkId?: string) => {
        const factory = new BookmarkResponseFactory();
        return factory.createNotFoundErrorResponse(
            bookmarkId || "non-existent-id",
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new BookmarkResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
        );
    },
    
    serverError: () => {
        const factory = new BookmarkResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    rateLimitError: () => {
        const factory = new BookmarkResponseFactory();
        const RATE_LIMIT = 100;
        const REMAINING = 0;
        const ONE_HOUR_MS = 3600000;
        return factory.createRateLimitErrorResponse(
            RATE_LIMIT,
            REMAINING,
            new Date(Date.now() + ONE_HOUR_MS), // Reset in 1 hour
        );
    },
    
    // MSW handlers
    handlers: {
        success: () => new BookmarkResponseFactory().createMSWHandlers(),
        withErrors: function createWithErrors(errorRate = 0.5) { 
            return new BookmarkResponseFactory().createMSWHandlers({ errorRate });
        },
        withDelay: function createWithDelay(delay?: number) {
            const DEFAULT_DELAY_MS = 2000;
            return new BookmarkResponseFactory().createMSWHandlers({ delay: delay ?? DEFAULT_DELAY_MS });
        },
        withNetworkFailure: function createWithNetworkFailure(failureRate?: number) {
            const DEFAULT_FAILURE_RATE = 0.3;
            return new BookmarkResponseFactory().createMSWHandlers({ networkFailureRate: failureRate ?? DEFAULT_FAILURE_RATE });
        },
    },
};

// Export factory instance for direct use
export const bookmarkResponseFactory = new BookmarkResponseFactory();

