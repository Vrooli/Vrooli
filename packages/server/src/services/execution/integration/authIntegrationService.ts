import { type PrismaClient } from "@prisma/client";
import { type Logger } from "winston";
import { getUserLanguages } from "../../../auth/request.js";
import { type IEventBus, EventUtils } from "../../events/index.js";

/**
 * User data for execution context
 */
export interface ExecutionUserData {
    id: string;
    email?: string;
    name?: string;
    languages: string[];
    preferences: Record<string, unknown>;
    permissions: {
        canExecuteRoutines: boolean;
        canCreateSwarms: boolean;
        canUseCredits: boolean;
        maxConcurrentRuns: number;
        maxCreditUsage: number;
    };
    teamMemberships: Array<{
        teamId: string;
        role: string;
        permissions: Record<string, unknown>;
    }>;
}

/**
 * Authentication Integration Service
 * 
 * This service provides user authentication and emits access attempt events
 * for security agents to evaluate. It integrates with Vrooli's existing
 * user management systems while enabling fully emergent security decisions.
 * 
 * IMPORTANT: This service does NOT make security decisions. It only:
 * 1. Fetches user data and context
 * 2. Emits access attempt events
 * 3. Waits for security agent responses via barrier-sync
 * 
 * All security intelligence emerges from agent behaviors.
 */
export class AuthIntegrationService {
    private readonly prisma: PrismaClient;
    private readonly logger: Logger;
    private readonly eventBus: IEventBus;

    constructor(prisma: PrismaClient, logger: Logger, eventBus: IEventBus) {
        this.prisma = prisma;
        this.logger = logger;
        this.eventBus = eventBus;
    }

    /**
     * Gets user data for execution context
     */
    async getUserData(userId: string): Promise<ExecutionUserData | null> {
        this.logger.debug("[AuthIntegrationService] Getting user data", { userId });

        try {
            const user = await this.prisma.user.findUnique({
                where: {
                    id: BigInt(userId),
                },
                include: {
                    translations: {
                        include: {
                            language: true,
                        },
                    },
                    memberships: {
                        include: {
                            team: true,
                        },
                        where: {
                            // Only active memberships
                            NOT: {
                                team: {
                                    isDeleted: true,
                                },
                            },
                        },
                    },
                    wallets: true,
                },
            });

            if (!user) {
                this.logger.warn("[AuthIntegrationService] User not found", { userId });
                return null;
            }

            // Get user languages
            const languages = getUserLanguages(user as any);

            // Extract user name
            const userName = this.extractUserName(user);

            // Build permissions
            const permissions = {
                canExecuteRoutines: !user.isDeleted && !user.banReason,
                canCreateSwarms: !user.isDeleted && !user.banReason,
                canUseCredits: !user.isDeleted && !user.banReason,
                maxConcurrentRuns: this.calculateMaxConcurrentRuns(user),
                maxCreditUsage: this.calculateMaxCreditUsage(user),
            };

            // Build team memberships
            const teamMemberships = user.memberships.map(membership => ({
                teamId: membership.team.id.toString(),
                role: membership.role,
                permissions: this.extractTeamPermissions(membership),
            }));

            // Build preferences
            const preferences = this.extractUserPreferences(user);

            const userData: ExecutionUserData = {
                id: user.id.toString(),
                email: user.email || undefined,
                name: userName,
                languages,
                preferences,
                permissions,
                teamMemberships,
            };

            this.logger.debug("[AuthIntegrationService] User data retrieved", {
                userId,
                userName,
                languageCount: languages.length,
                teamCount: teamMemberships.length,
                canExecute: permissions.canExecuteRoutines,
            });

            return userData;

        } catch (error) {
            this.logger.error("[AuthIntegrationService] Failed to get user data", {
                userId,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Validates API key and returns user data
     */
    async validateApiKey(apiKey: string): Promise<ExecutionUserData | null> {
        this.logger.debug("[AuthIntegrationService] Validating API key");

        try {
            const keyRecord = await this.prisma.api_key.findUnique({
                where: {
                    key: apiKey,
                },
                include: {
                    user: {
                        include: {
                            translations: {
                                include: {
                                    language: true,
                                },
                            },
                            memberships: {
                                include: {
                                    team: true,
                                },
                            },
                        },
                    },
                    team: {
                        include: {
                            members: {
                                include: {
                                    user: true,
                                },
                            },
                        },
                    },
                },
            });

            if (!keyRecord) {
                this.logger.warn("[AuthIntegrationService] Invalid API key");
                return null;
            }

            // Check if key is disabled
            if (keyRecord.disabledAt) {
                this.logger.warn("[AuthIntegrationService] API key is disabled", {
                    keyId: keyRecord.id,
                    disabledAt: keyRecord.disabledAt,
                });
                return null;
            }

            // Check credit limits
            if (keyRecord.stopAtLimit && keyRecord.creditsUsed >= keyRecord.limitHard) {
                this.logger.warn("[AuthIntegrationService] API key credit limit exceeded", {
                    keyId: keyRecord.id,
                    creditsUsed: keyRecord.creditsUsed,
                    limitHard: keyRecord.limitHard,
                });
                return null;
            }

            // Get user data from either user or team context
            let targetUser = keyRecord.user;
            if (!targetUser && keyRecord.team) {
                // For team keys, use the team owner or first admin
                const teamAdmin = keyRecord.team.members.find(m => m.role === "Admin");
                targetUser = teamAdmin?.user;
            }

            if (!targetUser) {
                this.logger.warn("[AuthIntegrationService] No user associated with API key", {
                    keyId: keyRecord.id,
                });
                return null;
            }

            // Get user data with API key permissions
            const userData = await this.getUserData(targetUser.id.toString());
            if (!userData) {
                return null;
            }

            // Override permissions with API key permissions
            const apiPermissions = this.parseApiKeyPermissions(keyRecord.permissions);
            userData.permissions = {
                ...userData.permissions,
                ...apiPermissions,
                maxCreditUsage: Math.min(
                    userData.permissions.maxCreditUsage,
                    Number(keyRecord.limitHard - keyRecord.creditsUsed),
                ),
            };

            this.logger.info("[AuthIntegrationService] API key validated", {
                keyId: keyRecord.id,
                userId: userData.id,
                remainingCredits: keyRecord.limitHard - keyRecord.creditsUsed,
            });

            return userData;

        } catch (error) {
            this.logger.error("[AuthIntegrationService] API key validation failed", {
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Records credit usage for API key
     */
    async recordCreditUsage(apiKey: string, credits: number): Promise<void> {
        this.logger.debug("[AuthIntegrationService] Recording credit usage", {
            credits,
        });

        try {
            await this.prisma.api_key.update({
                where: {
                    key: apiKey,
                },
                data: {
                    creditsUsed: {
                        increment: BigInt(Math.ceil(credits)),
                    },
                    updatedAt: new Date(),
                },
            });

            this.logger.debug("[AuthIntegrationService] Credit usage recorded", {
                credits,
            });

        } catch (error) {
            this.logger.error("[AuthIntegrationService] Failed to record credit usage", {
                credits,
                error: error instanceof Error ? error.message : String(error),
            });
            throw error;
        }
    }

    /**
     * Emits an access attempt event and waits for security agent decisions
     */
    async canUserExecuteRoutine(
        userId: string,
        routineId: string,
    ): Promise<{ allowed: boolean; reason?: string }> {
        this.logger.debug("[AuthIntegrationService] Emitting routine access attempt event", {
            userId,
            routineId,
        });

        try {
            // Get user data and routine data for context
            const userData = await this.getUserData(userId);
            if (!userData) {
                // Even for missing users, emit the event so agents can track attempts
                const noUserEvent = EventUtils.createBaseEvent(
                    "access/attempt/routine",
                    {
                        userId,
                        routineId,
                        userExists: false,
                        resource: "routine",
                        action: "execute",
                        context: {
                            timestamp: new Date(),
                            source: "execution",
                        },
                    },
                    EventUtils.createEventSource("cross-cutting", "AuthIntegrationService"),
                    EventUtils.createEventMetadata("barrier-sync", "high", {
                        barrierConfig: {
                            quorum: 1,
                            timeoutMs: 5000,
                            timeoutAction: "auto-reject",
                            requiredResponders: ["security_agent"],
                        },
                        userId,
                    }),
                );

                const result = await this.eventBus.publishBarrierSync(noUserEvent as any);
                return {
                    allowed: result.success,
                    reason: result.success ? undefined : "User not found",
                };
            }

            // Get routine information
            const resourceVersion = await this.prisma.resource_version.findFirst({
                where: {
                    OR: [
                        { publicId: routineId },
                        { root: { publicId: routineId } },
                    ],
                },
                include: {
                    root: true,
                },
            });

            // Create access attempt event with full context
            const accessEvent = EventUtils.createBaseEvent(
                "access/attempt/routine",
                {
                    userId: userData.id,
                    routineId,
                    resource: "routine",
                    action: "execute",
                    context: {
                        userPermissions: userData.permissions,
                        userTeams: userData.teamMemberships.map(tm => tm.teamId),
                        userLanguages: userData.languages,
                        routineExists: !!resourceVersion,
                        routineOwner: resourceVersion?.root.ownedByUserId?.toString(),
                        routineTeam: resourceVersion?.root.ownedByTeamId?.toString(),
                        routineIsPrivate: resourceVersion?.root.isPrivate,
                        timestamp: new Date(),
                        source: "execution",
                    },
                },
                EventUtils.createEventSource("cross-cutting", "AuthIntegrationService"),
                EventUtils.createEventMetadata("barrier-sync", "high", {
                    barrierConfig: {
                        quorum: 1,
                        timeoutMs: 5000,
                        timeoutAction: "auto-reject",
                        requiredResponders: ["security_agent"],
                    },
                    userId: userData.id,
                }),
            );

            // Publish barrier-sync event and wait for security agent responses
            const result = await this.eventBus.publishBarrierSync(accessEvent as any);

            // Aggregate agent responses
            if (result.success) {
                this.logger.info("[AuthIntegrationService] Access granted by security agents", {
                    userId,
                    routineId,
                    responses: result.responses.length,
                    duration: result.duration,
                });
                return { allowed: true };
            } else {
                // Extract denial reasons from agent responses
                const denialReasons = result.responses
                    .filter(r => r.response === "ALARM")
                    .map(r => r.reason)
                    .filter(Boolean);

                const reason = denialReasons.length > 0
                    ? denialReasons.join("; ")
                    : result.timedOut
                        ? "Security check timed out"
                        : "Access denied by security policy";

                this.logger.warn("[AuthIntegrationService] Access denied by security agents", {
                    userId,
                    routineId,
                    reason,
                    responses: result.responses,
                });

                return { allowed: false, reason };
            }

        } catch (error) {
            this.logger.error("[AuthIntegrationService] Failed to check access permission", {
                userId,
                routineId,
                error: error instanceof Error ? error.message : String(error),
            });

            // In case of errors, emit an error event and default to deny
            await this.eventBus.publish(EventUtils.createBaseEvent(
                "access/error",
                {
                    userId,
                    routineId,
                    resource: "routine",
                    action: "execute",
                    error: error instanceof Error ? error.message : String(error),
                },
                EventUtils.createEventSource("cross-cutting", "AuthIntegrationService"),
            ));

            return { allowed: false, reason: "Access check failed due to system error" };
        }
    }

    /**
     * Private helper methods
     */
    private extractUserName(user: any): string {
        // Try to get name from translations
        const englishTranslation = user.translations?.find(
            (t: any) => t.language.code === "en",
        );

        if (englishTranslation?.name) {
            return englishTranslation.name;
        }

        // Fallback to any translation
        const firstTranslation = user.translations?.[0];
        if (firstTranslation?.name) {
            return firstTranslation.name;
        }

        // Fallback to username or email
        return user.name || user.email || `User ${user.id}`;
    }

    private calculateMaxConcurrentRuns(user: any): number {
        // Base limit
        let limit = 5;

        // Increase for premium users (would need to check subscriptions)
        if (user.isPrivate) {
            limit += 5;
        }

        // Increase based on reputation/score
        if (user.reputationScore > 100) {
            limit += 2;
        }

        return limit;
    }

    private calculateMaxCreditUsage(user: any): number {
        // Base daily credit limit
        let limit = 10000;

        // Increase for verified users
        if (user.isVerified) {
            limit += 5000;
        }

        // Increase based on reputation
        if (user.reputationScore > 100) {
            limit += Math.min(user.reputationScore * 50, 20000);
        }

        return limit;
    }

    private extractTeamPermissions(membership: any): Record<string, unknown> {
        // Extract permissions based on role
        const basePermissions = {
            canExecuteTeamRoutines: true,
            canViewTeamRuns: true,
        };

        if (membership.role === "Admin" || membership.role === "Owner") {
            return {
                ...basePermissions,
                canManageTeamExecution: true,
                canCreateSwarms: true,
                canManageTeamCredits: true,
            };
        }

        return basePermissions;
    }

    private extractUserPreferences(user: any): Record<string, unknown> {
        // Extract user preferences from stored data
        return {
            theme: "system",
            language: "en",
            notifications: {
                execution: true,
                errors: true,
                completion: true,
            },
            execution: {
                defaultModel: "gpt-4o-mini",
                maxCostPerRun: 1.0,
                timeoutMinutes: 30,
            },
        };
    }

    private parseApiKeyPermissions(permissions: any): Partial<ExecutionUserData["permissions"]> {
        try {
            const perms = typeof permissions === "string" ? JSON.parse(permissions) : permissions;

            return {
                canExecuteRoutines: perms.execution?.allowed !== false,
                canCreateSwarms: perms.swarms?.allowed !== false,
                canUseCredits: perms.credits?.allowed !== false,
                maxConcurrentRuns: perms.execution?.maxConcurrent || 5,
                maxCreditUsage: perms.credits?.maxUsage || 10000,
            };
        } catch (error) {
            this.logger.warn("[AuthIntegrationService] Failed to parse API key permissions", {
                permissions,
                error: error instanceof Error ? error.message : String(error),
            });
            return {};
        }
    }
}
