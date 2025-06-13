import { AccountStatus, CreditEntryType, type User, uuid } from "@vrooli/shared";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { ModelMap } from "../../models/base/index.js";
import { type ApiEndpoint } from "../../types.js";
import { isOwnerAdminCheck } from "../../utils/defaultPermissions.js";
import { CacheService } from "../../services/cache.js";

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
        lastRolloverJobStatus: 'success' | 'partial' | 'failed' | 'never_run';
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
 * Admin-only endpoints for site management
 */
export const admin: EndpointsAdmin = {
    /**
     * Get site-wide statistics for admin dashboard
     */
    siteStats: async ({ input }, { req }) => {
        // Check admin permissions
        const userData = req.session;
        if (!userData?.userId) {
            throw new CustomError("0191", "Unauthorized", { message: "Authentication required" });
        }

        const isAdmin = await isOwnerAdminCheck(userData.userId.toString());
        if (!isAdmin) {
            throw new CustomError("0192", "Forbidden", { message: "Admin privileges required" });
        }

        const prisma = DbProvider.get();
        const redis = CacheService.get().redis;
        
        // Calculate current month for credit queries
        const currentMonth = new Date().toISOString().substring(0, 7);
        const sixMonthsAgo = new Date();
        sixMonthsAgo.setMonth(sixMonthsAgo.getMonth() - 6);
        
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
                    COUNT(CASE WHEN last_active >= ${new Date(Date.now() - 30 * 24 * 60 * 60 * 1000)} THEN 1 END)::text as active_users,
                    COUNT(CASE WHEN created_at >= ${new Date(new Date().setHours(0, 0, 0, 0))} THEN 1 END)::text as new_users_today
                FROM "user"
            `,
            
            // Combined routine statistics
            prisma.$queryRaw<Array<{ 
                total_routines: string, 
                active_routines: string 
            }>>`
                SELECT 
                    COUNT(*)::text as total_routines,
                    COUNT(CASE WHEN is_deleted = false THEN 1 END)::text as active_routines
                FROM routine
            `,
            
            // Total API keys
            prisma.apiKey.count({
                where: { disabledAt: null },
            }),
            
            // Total credits in circulation
            prisma.credit_account.aggregate({
                _sum: { currentBalance: true }
            }),
            
            // Combined credit donation statistics
            prisma.$queryRaw<Array<{ 
                donated_this_month: string, 
                donated_all_time: string,
                donors_this_month: string
            }>>`
                SELECT 
                    ABS(SUM(CASE WHEN created_at >= ${new Date(currentMonth + '-01')} THEN amount ELSE 0 END))::text as donated_this_month,
                    ABS(SUM(amount))::text as donated_all_time,
                    COUNT(DISTINCT CASE WHEN created_at >= ${new Date(currentMonth + '-01')} THEN credit_account_id END)::text as donors_this_month
                FROM credit_ledger_entry
                WHERE type = ${CreditEntryType.DonationGiven}
            `,
            
            // Average donation percentage from user settings
            prisma.$queryRaw<Array<{ avg: number }>>`
                SELECT AVG(((creditSettings->'donation')::jsonb->>'percentage')::numeric) as avg
                FROM "user"
                WHERE creditSettings IS NOT NULL
                AND creditSettings::jsonb->'donation' IS NOT NULL
                AND (creditSettings::jsonb->'donation'->>'enabled')::boolean = true
            `,
            
            // Get donation history by month
            prisma.$queryRaw<Array<{ month: Date, total: string, donors: string }>>`
                SELECT 
                    DATE_TRUNC('month', created_at) as month,
                    ABS(SUM(amount))::text as total,
                    COUNT(DISTINCT credit_account_id)::text as donors
                FROM credit_ledger_entry
                WHERE type = ${CreditEntryType.DonationGiven}
                    AND created_at >= ${sixMonthsAgo}
                GROUP BY DATE_TRUNC('month', created_at)
                ORDER BY month DESC
            `,
            
            // Job status from Redis
            redis?.get('job:creditRollover:lastRun') ?? null,
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
        let jobStatus: 'success' | 'partial' | 'failed' | 'never_run' = 'never_run';
        let jobTime: string | null = null;
        if (lastJobStatus) {
            try {
                const jobData = JSON.parse(lastJobStatus);
                if (jobData && typeof jobData === 'object') {
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
                activeDonorsThisMonth: activeDonorsThisMonth,
                averageDonationPercentage: Number(avgDonationPercentage[0]?.avg) || 0,
                lastRolloverJobStatus: jobStatus,
                lastRolloverJobTime: jobTime,
                nextScheduledRollover: nextMonth.toISOString(),
                donationsByMonth: donationsByMonth.map(row => ({
                    month: row.month.toISOString().substring(0, 7),
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
        if (!userData?.userId) {
            throw new CustomError("0191", "Unauthorized", { message: "Authentication required" });
        }

        const isAdmin = await isOwnerAdminCheck(userData.userId.toString());
        if (!isAdmin) {
            throw new CustomError("0192", "Forbidden", { message: "Admin privileges required" });
        }

        const prisma = DbProvider.get();
        const {
            searchTerm,
            status,
            skip = 0,
            take = 50,
            sortBy = "createdAt",
            sortOrder = "desc",
        } = input || {};

        // Build where clause
        const where: any = {};
        
        if (searchTerm) {
            where.OR = [
                { name: { contains: searchTerm, mode: "insensitive" } },
                { email: { contains: searchTerm, mode: "insensitive" } },
            ];
        }
        
        if (status) {
            where.status = status;
        }

        // Build order by clause
        const orderBy: any = {};
        orderBy[sortBy] = sortOrder;

        // Get users with counts
        const [users, totalCount] = await Promise.all([
            prisma.user.findMany({
                where,
                skip,
                take: Math.min(take, 100), // Limit to 100 for performance
                orderBy,
                include: {
                    _count: {
                        select: {
                            apiKeys: true,
                            routines: true,
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
                email: user.email || undefined,
                status: user.status || AccountStatus.Unlocked,
                createdAt: user.createdAt.toISOString(),
                lastActiveAt: user.lastActive?.toISOString(),
                apiKeyCount: user._count.apiKeys,
                routineCount: user._count.routines,
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
        if (!userData?.userId) {
            throw new CustomError("0191", "Unauthorized", { message: "Authentication required" });
        }

        const isAdmin = await isOwnerAdminCheck(userData.userId.toString());
        if (!isAdmin) {
            throw new CustomError("0192", "Forbidden", { message: "Admin privileges required" });
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
        if (BigInt(userId) === userData.userId) {
            throw new CustomError("0194", "Forbidden", { message: "Cannot modify your own account status" });
        }

        // Update user status
        const updatedUser = await prisma.user.update({
            where: { id: BigInt(userId) },
            data: {
                status,
                statusReason: reason,
                updatedAt: new Date(),
            },
        });

        // Log admin action
        logger.info("Admin user status change", {
            adminId: userData.userId.toString(),
            action: "USER_STATUS_CHANGE",
            targetUserId: userId,
            oldStatus: targetUser.status,
            newStatus: status,
            reason,
            timestamp: new Date().toISOString(),
        });

        // Return updated user (using existing User format)
        const userFormat = ModelMap.get("User").format;
        return userFormat.dbToApi(updatedUser);
    },

    /**
     * Force password reset for a user (admin only)
     */
    userResetPassword: async ({ input }, { req }) => {
        // Check admin permissions
        const userData = req.session;
        if (!userData?.userId) {
            throw new CustomError("0191", "Unauthorized", { message: "Authentication required" });
        }

        const isAdmin = await isOwnerAdminCheck(userData.userId.toString());
        if (!isAdmin) {
            throw new CustomError("0192", "Forbidden", { message: "Admin privileges required" });
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
            throw new CustomError("0195", "BadRequest", { message: "User has no email address for password reset" });
        }

        // Invalidate all existing sessions for the user
        await prisma.session.deleteMany({
            where: { userId: BigInt(userId) },
        });

        // Invalidate all API keys for the user
        await prisma.apiKey.updateMany({
            where: { userId: BigInt(userId) },
            data: { disabledAt: new Date() },
        });

        // TODO: Send password reset email
        // This would integrate with the existing email service
        
        // Log admin action
        logger.info("Admin password reset", {
            adminId: userData.userId.toString(),
            action: "USER_PASSWORD_RESET",
            targetUserId: userId,
            reason,
            email: targetUser.emails[0].emailAddress,
            timestamp: new Date().toISOString(),
        });

        return { success: true };
    },
};