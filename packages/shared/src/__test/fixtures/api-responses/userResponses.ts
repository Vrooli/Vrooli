/* c8 ignore start */
/**
 * User API Response Fixtures
 * 
 * Comprehensive fixtures for user management endpoints including
 * profile operations, bot creation, and user queries.
 */

import type {
    Bot,
    BotCreateInput,
    Profile,
    User,
    UserCreateInput,
    UserUpdateInput,
} from "../../../api/types.js";
import { generatePK } from "../../../id/index.js";
import { BaseAPIResponseFactory } from "./base.js";
import type { MockDataOptions } from "./types.js";

/**
 * User API response factory
 * 
 * Handles user profiles, bot creation, and user management operations.
 */
export class UserResponseFactory extends BaseAPIResponseFactory<
    User,
    UserCreateInput,
    UserUpdateInput
> {
    protected readonly entityName = "user";

    /**
     * Create mock user data
     */
    createMockData(options?: MockDataOptions): User {
        const scenario = options?.scenario || "minimal";
        const now = new Date().toISOString();
        const userId = String(options?.overrides?.id || generatePK());

        const baseUser: User = {
            __typename: "User",
            id: userId,
            createdAt: now,
            updatedAt: now,
            bannerImage: null,
            bio: null,
            handle: String(options?.overrides?.handle || `user_${userId.slice(0, 8)}`),
            isBot: false,
            isBotDepictingPerson: false,
            isPrivate: false,
            name: String(options?.overrides?.name || "Test User"),
            profileImage: null,
            isPrivateBookmarks: true,
            isPrivateVotes: true,
            bookmarks: 0,
            reportsReceivedCount: 0,
            you: {
                canDelete: false,
                canReport: false,
                canUpdate: false,
                isBookmarked: false,
                isViewed: false,
            },
        };

        if (scenario === "complete" || scenario === "edge-case") {
            return {
                ...baseUser,
                bannerImage: "https://example.com/banner.jpg",
                bio: "A passionate developer interested in AI and automation.",
                profileImage: "https://example.com/avatar.jpg",
                bookmarks: 42,
                reportsReceivedCount: 0,
                isPrivateBookmarks: false,
                isPrivateVotes: false,
                botSettings: options?.overrides?.isBot ? {
                    creativity: 0.7,
                    verbosity: 0.5,
                    model: "gpt-4",
                    persona: "helpful",
                } : undefined,
                translations: [{
                    __typename: "ProfileTranslation",
                    id: generatePK().toString(),
                    language: "en",
                    bio: "A passionate developer interested in AI and automation.",
                }],
                ...options?.overrides,
            };
        }

        return {
            ...baseUser,
            ...options?.overrides,
        };
    }

    /**
     * Create user from input (not typically used for users)
     */
    createFromInput(input: UserCreateInput): User {
        // Users are typically created through auth signup
        throw new Error("Users are created through auth endpoints, not user endpoints");
    }

    /**
     * Update user from input
     */
    updateFromInput(existing: User, input: UserUpdateInput): User {
        const updates: Partial<User> = {
            updatedAt: new Date().toISOString(),
        };

        if (input.bannerImage !== undefined) updates.bannerImage = input.bannerImage;
        if (input.bio !== undefined) updates.bio = input.bio;
        if (input.handle !== undefined) updates.handle = input.handle;
        if (input.name !== undefined) updates.name = input.name;
        if (input.profileImage !== undefined) updates.profileImage = input.profileImage;
        if (input.isPrivateBookmarks !== undefined) updates.isPrivateBookmarks = input.isPrivateBookmarks;
        if (input.isPrivateVotes !== undefined) updates.isPrivateVotes = input.isPrivateVotes;

        if (input.translationsCreate) {
            updates.translations = [
                ...(existing.translations || []),
                ...input.translationsCreate.map(t => ({
                    __typename: "ProfileTranslation" as const,
                    id: generatePK().toString(),
                    language: t.language,
                    bio: t.bio,
                })),
            ];
        }

        return {
            ...existing,
            ...updates,
        };
    }

    /**
     * Validate create input
     */
    async validateCreateInput(input: UserCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        return {
            valid: false,
            errors: {
                general: "Users must be created through auth signup endpoints",
            },
        };
    }

    /**
     * Validate update input
     */
    async validateUpdateInput(input: UserUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (input.handle !== undefined) {
            if (input.handle.length < 3) {
                errors.handle = "Handle must be at least 3 characters";
            } else if (!/^[a-zA-Z0-9_-]+$/.test(input.handle)) {
                errors.handle = "Handle can only contain letters, numbers, underscores, and hyphens";
            }
        }

        if (input.bio !== undefined && input.bio.length > 500) {
            errors.bio = "Bio must be 500 characters or less";
        }

        if (input.name !== undefined && input.name.length < 1) {
            errors.name = "Name cannot be empty";
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create a bot user
     */
    createBot(input?: Partial<BotCreateInput>): Bot {
        const now = new Date().toISOString();
        const botId = generatePK().toString();

        return {
            __typename: "User",
            id: botId,
            createdAt: now,
            updatedAt: now,
            bannerImage: null,
            bio: input?.bio || "I'm a helpful AI assistant.",
            handle: input?.handle || `bot_${botId.slice(0, 8)}`,
            isBot: true,
            isBotDepictingPerson: input?.isBotDepictingPerson || false,
            isPrivate: input?.isPrivate || false,
            name: input?.name || "AI Assistant",
            profileImage: input?.profileImage || null,
            isPrivateBookmarks: true,
            isPrivateVotes: true,
            bookmarks: 0,
            reportsReceivedCount: 0,
            botSettings: {
                creativity: input?.botSettings?.creativity || 0.7,
                verbosity: input?.botSettings?.verbosity || 0.5,
                model: input?.botSettings?.model || "gpt-4",
                persona: input?.botSettings?.persona || "helpful",
            },
            translations: input?.translationsCreate?.map(t => ({
                __typename: "ProfileTranslation" as const,
                id: generatePK().toString(),
                language: t.language,
                bio: t.bio,
            })) || [],
            you: {
                canDelete: true,
                canReport: false,
                canUpdate: true,
                isBookmarked: false,
                isViewed: false,
            },
        } as Bot;
    }

    /**
     * Create a premium user
     */
    createPremiumUser(): User {
        const user = this.createMockData({ scenario: "complete" });
        return {
            ...user,
            premium: {
                __typename: "Premium",
                id: generatePK().toString(),
                credits: "10000",
                expiresAt: new Date(Date.now().toISOString() + 365 * 24 * 60 * 60 * 1000).toISOString(), // 1 year
            },
        };
    }

    /**
     * Create a user with teams
     */
    createUserWithTeams(teamCount = 2): User {
        const user = this.createMockData({ scenario: "complete" });
        const teams = [];

        for (let i = 0; i < teamCount; i++) {
            teams.push({
                __typename: "Team" as const,
                id: generatePK().toString(),
                createdAt: new Date().toISOString(),
                updatedAt: new Date().toISOString(),
                handle: `team${i + 1}`,
                name: `Team ${i + 1}`,
                isOpenToNewMembers: true,
                profileImage: null,
                bannerImage: null,
                you: {
                    canAddMembers: i === 0, // Can add to first team
                    canDelete: i === 0,
                    canBookmark: true,
                    canReport: false,
                    canUpdate: i === 0,
                    canRead: true,
                    isBookmarked: false,
                    isViewed: true,
                    role: i === 0 ? "Owner" : "Member",
                },
            });
        }

        return {
            ...user,
            memberships: teams.map(team => ({
                __typename: "Member" as const,
                id: generatePK().toString(),
                createdAt: new Date().toISOString(),
                updatedAt: new Date().toISOString(),
                isAdmin: team.you.role === "Owner",
                role: team.you.role,
                team,
                user,
            })),
        };
    }

    /**
     * Create profile update response
     */
    createProfileUpdateResponse(profile: Profile): { profile: Profile } {
        return { profile };
    }
}

/**
 * Pre-configured user response scenarios
 */
export const userResponseScenarios = {
    // Success scenarios
    findSuccess: (user?: User) => {
        const factory = new UserResponseFactory();
        return factory.createSuccessResponse(
            user || factory.createMockData(),
        );
    },

    findCompleteSuccess: () => {
        const factory = new UserResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({ scenario: "complete" }),
        );
    },

    updateSuccess: (updates?: Partial<UserUpdateInput>) => {
        const factory = new UserResponseFactory();
        const existing = factory.createMockData({ scenario: "complete" });
        const input: UserUpdateInput = {
            id: existing.id,
            ...updates,
        };
        return factory.createSuccessResponse(
            factory.updateFromInput(existing, input),
        );
    },

    botCreateSuccess: (bot?: Partial<BotCreateInput>) => {
        const factory = new UserResponseFactory();
        return factory.createSuccessResponse(
            factory.createBot(bot),
        );
    },

    premiumUserSuccess: () => {
        const factory = new UserResponseFactory();
        return factory.createSuccessResponse(
            factory.createPremiumUser(),
        );
    },

    userWithTeamsSuccess: (teamCount?: number) => {
        const factory = new UserResponseFactory();
        return factory.createSuccessResponse(
            factory.createUserWithTeams(teamCount),
        );
    },

    listSuccess: (users?: User[]) => {
        const factory = new UserResponseFactory();
        const DEFAULT_COUNT = 10;
        return factory.createPaginatedResponse(
            users || Array.from({ length: DEFAULT_COUNT }, () => factory.createMockData()),
            { page: 1, totalCount: users?.length || DEFAULT_COUNT },
        );
    },

    // Error scenarios
    updateValidationError: () => {
        const factory = new UserResponseFactory();
        return factory.createValidationErrorResponse({
            handle: "Handle is already taken",
            bio: "Bio must be 500 characters or less",
            profileImage: "Invalid image URL",
        });
    },

    notFoundError: (userId?: string) => {
        const factory = new UserResponseFactory();
        return factory.createNotFoundErrorResponse(
            userId || "non-existent-user",
        );
    },

    permissionError: () => {
        const factory = new UserResponseFactory();
        return factory.createPermissionErrorResponse("update", ["user:update"]);
    },

    handleTakenError: (handle: string) => {
        const factory = new UserResponseFactory();
        return factory.createBusinessErrorResponse("conflict", {
            field: "handle",
            value: handle,
            message: `Handle '${handle}' is already taken`,
        });
    },

    botLimitError: () => {
        const factory = new UserResponseFactory();
        return factory.createBusinessErrorResponse("limit", {
            resource: "bots",
            limit: 5,
            current: 5,
            message: "You have reached the maximum number of bots",
        });
    },

    // MSW handlers
    handlers: {
        success: () => new UserResponseFactory().createMSWHandlers(),
        withErrors: function createWithErrors(errorRate = 0.1) {
            return new UserResponseFactory().createMSWHandlers({ errorRate });
        },
        withDelay: function createWithDelay(delay?: number) {
            const DEFAULT_DELAY_MS = 500;
            return new UserResponseFactory().createMSWHandlers({ delay: delay ?? DEFAULT_DELAY_MS });
        },
    },
};

// Export factory instance for direct use
export const userResponseFactory = new UserResponseFactory();
