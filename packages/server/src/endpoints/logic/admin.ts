import { AccountStatus, type User } from "@vrooli/shared";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { type ApiEndpoint } from "../../types.js";

// AI_CHECK: admin_endpoint_types=1 | LAST: 2025-06-29 - Fixed userUpdateStatus return type to match User interface

// Constants for date calculations
const MONTHS_IN_HALF_YEAR = 6;
const DAYS_IN_MONTH = 30;
const HOURS_IN_DAY = 24;
const MINUTES_IN_HOUR = 60;
const SECONDS_IN_MINUTE = 60;
const MS_IN_SECOND = 1000;
const DAYS_IN_MONTH_MS = DAYS_IN_MONTH * HOURS_IN_DAY * MINUTES_IN_HOUR * SECONDS_IN_MINUTE * MS_IN_SECOND;
const STRING_SLICE_DATE_MONTH = 7;
const DEFAULT_PAGE_SIZE = 50;

// Admin-specific input types
export interface AdminUserListInput {
    searchTerm?: string;
    status?: AccountStatus;
    skip?: number;
    take?: number;
    sortBy?: "createdAt" | "name" | "email" | "lastActiveAt";
    sortOrder?: "asc" | "desc";
}

export interface AdminUserUpdateStatusInput {
    userId: string;
    status: AccountStatus;
    reason?: string;
}

export interface AdminUserResetPasswordInput {
    userId: string;
    reason?: string;
}

export interface AdminSiteStatsOutput {
    totalUsers: number;
    activeUsers: number;
    newUsersToday: number;
    totalRoutines: number;
    activeRoutines: number;
    totalApiKeys: number;
    totalApiCalls: number;
    apiCallsToday: number;
    totalStorage: number;
    usedStorage: number;
    // Credit System Stats
    creditStats: {
        totalCreditsInCirculation: string; // bigint as string
        totalCreditsDonatedThisMonth: string;
        totalCreditsDonatedAllTime: string;
        activeDonorsThisMonth: number;
        averageDonationPercentage: number;
        lastRolloverJobStatus: "success" | "partial" | "failed" | "never_run";
        lastRolloverJobTime: string | null;
        nextScheduledRollover: string;
        donationsByMonth: Array<{
            month: string;
            amount: string;
            donors: number;
        }>; // Last 6 months
    };
}

export interface AdminUserListOutput {
    users: Array<{
        id: string;
        name?: string;
        email?: string;
        status?: AccountStatus;
        createdAt: string;
        lastActiveAt?: string;
        apiKeyCount: number;
        routineCount: number;
        teamCount: number;
    }>;
    totalCount: number;
}

// Admin endpoint types
export type EndpointsAdmin = {
    siteStats: ApiEndpoint<never, AdminSiteStatsOutput>;
    userList: ApiEndpoint<AdminUserListInput, AdminUserListOutput>;
    userUpdateStatus: ApiEndpoint<AdminUserUpdateStatusInput, User>;
    userResetPassword: ApiEndpoint<AdminUserResetPasswordInput, { success: boolean }>;
}

/**
 * Helper to extract userId from session (handles both API key and user sessions)
 */
function getUserIdFromSession(session: unknown): string | null {
    // API key sessions have userId directly
    const sessionObj = session as { userId?: string | number; users?: Array<{ id: string | number }> };
    if (sessionObj?.userId) {
        return sessionObj.userId.toString();
    }
    // User sessions have users array
    if (sessionObj?.users?.[0]?.id) {
        return sessionObj.users[0].id.toString();
    }
    return null;
}

/**
 * Admin-only endpoints for site management
 */
export const admin: EndpointsAdmin = {
    /**
     * Get site-wide statistics for admin dashboard
     */
    siteStats: async ({ input: _input }, { req }) => {
        // Check admin permissions
        const userData = req.session;
        const currentUserId = getUserIdFromSession(userData);
        if (!currentUserId) {
            throw new CustomError("0191", "Unauthorized", { message: "Authentication required" });
        }

        const isAdmin = false; //TODO there is a function somewhere for this - NOT isOwnerAdminCheck
        if (!isAdmin) {
            throw new CustomError("0192", "Unauthorized", { message: "Admin privileges required" });
        }

        const prisma = DbProvider.get();
        const redis = CacheService.get();

        // Calculate current month for credit queries
        const currentMonth = new Date().toISOString().substring(0, STRING_SLICE_DATE_MONTH);
        const sixMonthsAgo = new Date();
        sixMonthsAgo.setMonth(sixMonthsAgo.getMonth() - MONTHS_IN_HALF_YEAR);

        // Calculate statistics with optimized queries
        const [
            userStats,
            routineStats,
            totalApiKeys,
            totalCreditsInCirculation,
            creditDonationStats,
            avgDonationPercentage,
            donationsByMonth,
            lastJobStatus,
        ] = await Promise.all([
            // Combined user statistics in single query
            prisma.$queryRaw<Array<{
                total_users: string,
                active_users: string,
                new_users_today: string
            }>>`
                SELECT 
                    COUNT(*)::text as total_users,
                    COUNT(CASE WHEN "lastLoginAttempt" >= ${new Date(Date.now() - DAYS_IN_MONTH_MS)} THEN 1 END)::text as active_users,
                    COUNT(CASE WHEN "createdAt" >= ${new Date(new Date().setHours(0, 0, 0, 0))} THEN 1 END)::text as new_users_today
                FROM "user"
            `,

            // Combined routine statistics
            prisma.$queryRaw<Array<{
                total_routines: string,
                active_routines: string
            }>>`
                SELECT 
                    COUNT(*)::text as total_routines,
                    COUNT(*)::text as active_routines
                FROM run
            `,

            // Total API keys
            prisma.api_key.count({
                where: { disabledAt: null },
            }),

            // Total credits in circulation
            prisma.credit_account.aggregate({
                _sum: { currentBalance: true },
            }),

            // Combined credit donation statistics
            prisma.$queryRaw<Array<{
                donated_this_month: string,
                donated_all_time: string,
                donors_this_month: string
            }>>`
                SELECT 
                    ABS(SUM(CASE WHEN "createdAt" >= ${new Date(currentMonth + "-01")} THEN amount ELSE 0 END))::text as donated_this_month,
                    ABS(SUM(amount))::text as donated_all_time,
                    COUNT(DISTINCT CASE WHEN "createdAt" >= ${new Date(currentMonth + "-01")} THEN "accountId" END)::text as donors_this_month
                FROM credit_ledger_entry
                WHERE type = 'DonationGiven'
            `,

            // Average donation percentage from user settings
            prisma.$queryRaw<Array<{ avg: number }>>`
                SELECT AVG((("creditSettings"->'donation')::jsonb->>'percentage')::numeric) as avg
                FROM "user"
                WHERE "creditSettings" IS NOT NULL
                AND "creditSettings"::jsonb->'donation' IS NOT NULL
                AND ("creditSettings"::jsonb->'donation'->>'enabled')::boolean = true
            `,

            // Get donation history by month
            prisma.$queryRaw<Array<{ month: Date, total: string, donors: string }>>`
                SELECT 
                    DATE_TRUNC('month', "createdAt") as month,
                    ABS(SUM(amount))::text as total,
                    COUNT(DISTINCT "accountId")::text as donors
                FROM credit_ledger_entry
                WHERE type = 'DonationGiven'
                    AND "createdAt" >= ${sixMonthsAgo}
                GROUP BY DATE_TRUNC('month', "createdAt")
                ORDER BY month DESC
            `,

            // Job status from Redis
            redis?.get("job:creditRollover:lastRun") ?? null,
        ]);

        // Extract values from combined queries
        const totalUsers = parseInt(userStats[0].total_users, 10);
        const activeUsers = parseInt(userStats[0].active_users, 10);
        const newUsersToday = parseInt(userStats[0].new_users_today, 10);
        const totalRoutines = parseInt(routineStats[0].total_routines, 10);
        const activeRoutines = parseInt(routineStats[0].active_routines, 10);
        const activeDonorsThisMonth = parseInt(creditDonationStats[0].donors_this_month, 10);

        // API calls are placeholders
        const totalApiCalls = 0;
        const apiCallsToday = 0;

        // Parse job status from Redis
        let jobStatus: "success" | "partial" | "failed" | "never_run" = "never_run";
        let jobTime: string | null = null;
        if (lastJobStatus && typeof lastJobStatus === "string") {
            try {
                const jobData = JSON.parse(lastJobStatus);
                if (jobData && typeof jobData === "object") {
                    // Use the status field directly from creditRollover job
                    if (jobData.status) {
                        jobStatus = jobData.status;
                    }
                    // Timestamp is already an ISO string from creditRollover job
                    if (jobData.timestamp) {
                        jobTime = jobData.timestamp;
                    }
                }
            } catch (e) {
                logger.warn("Failed to parse job status from Redis", {
                    error: e instanceof Error ? e.message : String(e),
                    rawData: lastJobStatus,
                    trace: "adminStats_parseJobStatus",
                });
            }
        }

        // Calculate next scheduled rollover (2nd of next month at 2 AM)
        const nextMonth = new Date();
        nextMonth.setMonth(nextMonth.getMonth() + 1);
        nextMonth.setDate(2);
        nextMonth.setHours(2, 0, 0, 0);

        return {
            totalUsers,
            activeUsers,
            newUsersToday,
            totalRoutines,
            activeRoutines,
            totalApiKeys,
            totalApiCalls,
            apiCallsToday,
            totalStorage: 0, // Placeholder - would need file system stats
            usedStorage: 0,  // Placeholder
            creditStats: {
                totalCreditsInCirculation: totalCreditsInCirculation._sum.currentBalance?.toString() || "0",
                totalCreditsDonatedThisMonth: creditDonationStats[0].donated_this_month || "0",
                totalCreditsDonatedAllTime: creditDonationStats[0].donated_all_time || "0",
                activeDonorsThisMonth,
                averageDonationPercentage: Number(avgDonationPercentage[0]?.avg) || 0,
                lastRolloverJobStatus: jobStatus,
                lastRolloverJobTime: jobTime,
                nextScheduledRollover: nextMonth.toISOString(),
                donationsByMonth: donationsByMonth.map(row => ({
                    month: row.month.toISOString().substring(0, STRING_SLICE_DATE_MONTH),
                    amount: row.total,
                    donors: parseInt(row.donors, 10),
                })),
            },
        };
    },

    /**
     * List users with admin search and filtering capabilities
     */
    userList: async ({ input }, { req }) => {
        // Check admin permissions
        const userData = req.session;
        const currentUserId = getUserIdFromSession(userData);
        if (!currentUserId) {
            throw new CustomError("0191", "Unauthorized", { message: "Authentication required" });
        }

        const isAdmin = false; //TODO there is a function somewhere for this - NOT isOwnerAdminCheck
        if (!isAdmin) {
            throw new CustomError("0192", "Unauthorized", { message: "Admin privileges required" });
        }

        const prisma = DbProvider.get();
        const {
            searchTerm,
            status,
            skip = 0,
            take = DEFAULT_PAGE_SIZE,
            sortBy = "createdAt",
            sortOrder = "desc",
        } = input || {};

        // Build where clause
        const where: Record<string, unknown> = {};

        if (searchTerm) {
            where.OR = [
                { name: { contains: searchTerm, mode: "insensitive" } },
                { emails: { some: { emailAddress: { contains: searchTerm, mode: "insensitive" } } } },
            ];
        }

        if (status) {
            where.status = status;
        }

        // Build order by clause
        const orderBy: Record<string, string> = {};
        orderBy[sortBy] = sortOrder;

        // Get users with counts
        const [users, totalCount] = await Promise.all([
            prisma.user.findMany({
                where,
                skip,
                take: Math.min(take, 100), // Limit to 100 for performance
                orderBy,
                include: {
                    emails: {
                        take: 1,
                    },
                    _count: {
                        select: {
                            apiKeysExternal: true,
                            runs: true,
                            memberships: true,
                        },
                    },
                },
            }),
            prisma.user.count({ where }),
        ]);

        return {
            users: users.map(user => ({
                id: user.id.toString(),
                name: user.name || undefined,
                email: user.emails?.[0]?.emailAddress || undefined,
                status: (user.status || AccountStatus.Unlocked) as AccountStatus,
                createdAt: user.createdAt.toISOString(),
                lastActiveAt: user.lastLoginAttempt?.toISOString(),
                apiKeyCount: user._count.apiKeysExternal,
                routineCount: user._count.runs,
                teamCount: user._count.memberships,
            })),
            totalCount,
        };
    },

    /**
     * Update user account status (admin only)
     */
    userUpdateStatus: async ({ input }, { req }) => {
        // Check admin permissions
        const userData = req.session;
        const currentUserId = getUserIdFromSession(userData);
        if (!currentUserId) {
            throw new CustomError("0191", "Unauthorized", { message: "Authentication required" });
        }

        const isAdmin = false; //TODO there is a function somewhere for this - NOT isOwnerAdminCheck
        if (!isAdmin) {
            throw new CustomError("0192", "Unauthorized", { message: "Admin privileges required" });
        }

        const { userId, status, reason } = input;
        const prisma = DbProvider.get();

        // Validate user exists
        const targetUser = await prisma.user.findUnique({
            where: { id: BigInt(userId) },
        });

        if (!targetUser) {
            throw new CustomError("0193", "NotFound", { message: "User not found" });
        }

        // Prevent admin from disabling themselves
        if (userId === currentUserId) {
            throw new CustomError("0194", "Unauthorized", { message: "Cannot modify your own account status" });
        }

        // Update user status
        const updatedUser = await prisma.user.update({
            where: { id: BigInt(userId) },
            data: {
                status,
                updatedAt: new Date(),
            },
        });

        // Log admin action
        logger.info("Admin user status change", {
            adminId: currentUserId,
            action: "USER_STATUS_CHANGE",
            targetUserId: userId,
            oldStatus: targetUser.status,
            newStatus: status,
            reason,
            timestamp: new Date().toISOString(),
        });

        // Fetch the complete user data to return a valid User type
        const completeUser = await prisma.user.findUnique({
            where: { id: BigInt(userId) },
            include: {
                emails: true,
                translations: true,
                plan: true,
            },
        });

        if (!completeUser) {
            throw new CustomError("0196", "NotFound", { message: "User not found after update" });
        }

        // Return user matching the User type from shared package
        return {
            __typename: "User" as const,
            id: completeUser.id.toString(),
            name: completeUser.name || "",
            handle: completeUser.handle,
            publicId: completeUser.publicId || "",
            isPrivate: completeUser.isPrivate || false,
            isPrivateBookmarks: completeUser.isPrivateBookmarks || true,
            isPrivateMemberships: completeUser.isPrivateMemberships || true,
            isPrivatePullRequests: completeUser.isPrivatePullRequests || true,
            isPrivateResources: completeUser.isPrivateResources || true,
            isPrivateResourcesCreated: completeUser.isPrivateResourcesCreated || true,
            isPrivateTeamsCreated: completeUser.isPrivateTeamsCreated || true,
            isPrivateVotes: completeUser.isPrivateVotes || true,
            isBot: completeUser.isBot || false,
            isBotDepictingPerson: completeUser.isBotDepictingPerson || false,
            botSettings: completeUser.botSettings as import("@vrooli/shared").BotConfigObject | null,
            creditSettings: completeUser.creditSettings as import("@vrooli/shared").CreditConfigObject | null,
            bannerImage: completeUser.bannerImage,
            profileImage: completeUser.profileImage,
            theme: completeUser.theme,
            notificationSettings: completeUser.notificationSettings,
            status: (completeUser.status || AccountStatus.Unlocked) as AccountStatus,
            createdAt: completeUser.createdAt.toISOString(),
            updatedAt: completeUser.updatedAt.toISOString(),
            bookmarks: 0,
            membershipsCount: 0,
            reportsReceivedCount: 0,
            resourcesCount: 0,
            views: 0,
            bookmarkedBy: [],
            reportsReceived: [],
            translations: completeUser.translations.map(t => ({
                __typename: "UserTranslation" as const,
                id: t.id.toString(),
                language: t.language,
                bio: t.bio,
            })),
            you: {
                __typename: "UserYou" as const,
                canBookmark: false,
                canDelete: false,
                canReport: false,
                canUpdate: false,
                canUpdateApiKeys: false,
                canUpdateEmailSettings: false,
                canUpdatePrivacySettings: false,
                isBookmarked: false,
                isViewed: false,
            },
        };
    },

    /**
     * Force password reset for a user (admin only)
     */
    userResetPassword: async ({ input }, { req }) => {
        // Check admin permissions
        const userData = req.session;
        const currentUserId = getUserIdFromSession(userData);
        if (!currentUserId) {
            throw new CustomError("0191", "Unauthorized", { message: "Authentication required" });
        }

        const isAdmin = false; //TODO there is a function somewhere for this - NOT isOwnerAdminCheck
        if (!isAdmin) {
            throw new CustomError("0192", "Unauthorized", { message: "Admin privileges required" });
        }

        const { userId, reason } = input;
        const prisma = DbProvider.get();

        // Validate user exists
        const targetUser = await prisma.user.findUnique({
            where: { id: BigInt(userId) },
            include: { emails: true },
        });

        if (!targetUser) {
            throw new CustomError("0193", "NotFound", { message: "User not found" });
        }

        if (!targetUser.emails || targetUser.emails.length === 0) {
            throw new CustomError("0195", "InvalidArgs", { message: "User has no email address for password reset" });
        }

        // Invalidate all existing sessions for the user
        await prisma.session.deleteMany({
            where: { user_id: BigInt(userId) },
        });

        // Invalidate all API keys for the user
        await prisma.api_key.updateMany({
            where: { userId: BigInt(userId) },
            data: { disabledAt: new Date() },
        });

        // TODO: Send password reset email
        // This would integrate with the existing email service

        // Log admin action
        logger.info("Admin password reset", {
            adminId: currentUserId,
            action: "USER_PASSWORD_RESET",
            targetUserId: userId,
            reason,
            email: targetUser.emails[0].emailAddress,
            timestamp: new Date().toISOString(),
        });

        return { success: true };
    },
};
