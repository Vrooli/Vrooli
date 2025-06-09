import { AccountStatus, type User, uuid } from "@vrooli/shared";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { ModelMap } from "../../models/base/index.js";
import { type ApiEndpoint } from "../../types.js";
import { isOwnerAdminCheck } from "../../utils/defaultPermissions.js";

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
        
        // Calculate statistics
        const [
            totalUsers,
            activeUsers,
            newUsersToday,
            totalRoutines,
            activeRoutines,
            totalApiKeys,
            totalApiCalls,
            apiCallsToday,
        ] = await Promise.all([
            // Total users
            prisma.user.count(),
            
            // Active users (logged in within last 30 days)
            prisma.user.count({
                where: {
                    lastActive: {
                        gte: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000),
                    },
                },
            }),
            
            // New users today
            prisma.user.count({
                where: {
                    createdAt: {
                        gte: new Date(new Date().setHours(0, 0, 0, 0)),
                    },
                },
            }),
            
            // Total routines
            prisma.routine.count(),
            
            // Active routines (not deleted)
            prisma.routine.count({
                where: { isDeleted: false },
            }),
            
            // Total API keys
            prisma.apiKey.count({
                where: { disabledAt: null },
            }),
            
            // Total API calls (placeholder - would need API usage tracking)
            0,
            
            // API calls today (placeholder)
            0,
        ]);

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
        await prisma.adminLog.create({
            data: {
                id: uuid(),
                adminId: userData.userId,
                action: "USER_STATUS_CHANGE",
                targetId: BigInt(userId),
                details: {
                    oldStatus: targetUser.status,
                    newStatus: status,
                    reason,
                },
                createdAt: new Date(),
            },
        }).catch((error) => {
            // Log error but don't fail the main operation
            console.error("Failed to log admin action:", error);
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
        await prisma.adminLog.create({
            data: {
                id: uuid(),
                adminId: userData.userId,
                action: "USER_PASSWORD_RESET",
                targetId: BigInt(userId),
                details: {
                    reason,
                    email: targetUser.emails[0].emailAddress,
                },
                createdAt: new Date(),
            },
        }).catch((error) => {
            // Log error but don't fail the main operation
            console.error("Failed to log admin action:", error);
        });

        return { success: true };
    },
};