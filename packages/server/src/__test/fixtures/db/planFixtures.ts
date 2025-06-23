import { generatePK } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for Plan model - used for seeding subscription test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const planDbIds = {
    userBasic: generatePK(),
    userPremium: generatePK(),
    teamBasic: generatePK(),
    teamPremium: generatePK(),
    expired: generatePK(),
    trial: generatePK(),
    custom: generatePK(),
};

/**
 * Basic user plan
 */
export const basicUserPlanDb: Prisma.PlanCreateInput = {
    id: planDbIds.userBasic,
    enabledAt: new Date(),
    expiresAt: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000), // 30 days from now
    user: {
        connect: { id: generatePK() },
    },
};

/**
 * Premium user plan with trial history
 */
export const premiumUserPlanDb: Prisma.PlanCreateInput = {
    id: planDbIds.userPremium,
    enabledAt: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000), // Started 7 days ago
    expiresAt: new Date(Date.now() + 23 * 24 * 60 * 60 * 1000), // 23 days remaining
    receivedFreeTrialAt: new Date(Date.now() - 14 * 24 * 60 * 60 * 1000), // Trial 14 days ago
    user: {
        connect: { id: generatePK() },
    },
};

/**
 * Team basic plan
 */
export const basicTeamPlanDb: Prisma.PlanCreateInput = {
    id: planDbIds.teamBasic,
    enabledAt: new Date(),
    expiresAt: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000), // 30 days from now
    team: {
        connect: { id: generatePK() },
    },
};

/**
 * Team premium plan with custom features
 */
export const premiumTeamPlanDb: Prisma.PlanCreateInput = {
    id: planDbIds.teamPremium,
    enabledAt: new Date(Date.now() - 15 * 24 * 60 * 60 * 1000), // Started 15 days ago
    expiresAt: new Date(Date.now() + 345 * 24 * 60 * 60 * 1000), // ~11 months remaining
    customPlan: JSON.stringify({
        maxMembers: 50,
        features: ["advanced_analytics", "priority_support", "custom_branding"],
        limits: {
            storage: "unlimited",
            apiCalls: 1000000,
            teamProjects: 100,
        },
    }),
    team: {
        connect: { id: generatePK() },
    },
};

/**
 * Expired plan
 */
export const expiredPlanDb: Prisma.PlanCreateInput = {
    id: planDbIds.expired,
    enabledAt: new Date(Date.now() - 60 * 24 * 60 * 60 * 1000), // Started 60 days ago
    expiresAt: new Date(Date.now() - 5 * 24 * 60 * 60 * 1000), // Expired 5 days ago
    receivedFreeTrialAt: new Date(Date.now() - 75 * 24 * 60 * 60 * 1000), // Trial 75 days ago
    user: {
        connect: { id: generatePK() },
    },
};

/**
 * Trial plan (not yet converted)
 */
export const trialPlanDb: Prisma.PlanCreateInput = {
    id: planDbIds.trial,
    receivedFreeTrialAt: new Date(Date.now() - 3 * 24 * 60 * 60 * 1000), // Trial started 3 days ago
    user: {
        connect: { id: generatePK() },
    },
};

/**
 * Custom enterprise plan
 */
export const customEnterprisePlanDb: Prisma.PlanCreateInput = {
    id: planDbIds.custom,
    enabledAt: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000), // Started 30 days ago
    expiresAt: new Date(Date.now() + 335 * 24 * 60 * 60 * 1000), // ~11 months remaining
    customPlan: JSON.stringify({
        tier: "enterprise",
        maxMembers: 500,
        features: [
            "unlimited_storage",
            "advanced_analytics",
            "priority_support",
            "custom_branding",
            "white_label",
            "dedicated_support",
            "custom_integrations",
        ],
        limits: {
            storage: "unlimited",
            apiCalls: "unlimited",
            teamProjects: "unlimited",
            bandwidth: "unlimited",
        },
        support: {
            level: "platinum",
            responseTime: "1h",
            dedicatedManager: true,
        },
    }),
    team: {
        connect: { id: generatePK() },
    },
};

/**
 * Factory functions for creating plans dynamically
 */
export class PlanDbFactory {
    /**
     * Create minimal plan
     */
    static createMinimal(overrides?: Partial<Prisma.PlanCreateInput>): Prisma.PlanCreateInput {
        return {
            id: generatePK(),
            enabledAt: new Date(),
            expiresAt: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000),
            ...overrides,
        };
    }

    /**
     * Create user plan with specific duration
     */
    static createUserPlan(
        userId: bigint,
        durationDays = 30,
        overrides?: Partial<Prisma.PlanCreateInput>,
    ): Prisma.PlanCreateInput {
        return {
            id: generatePK(),
            enabledAt: new Date(),
            expiresAt: new Date(Date.now() + durationDays * 24 * 60 * 60 * 1000),
            user: { connect: { id: userId } },
            ...overrides,
        };
    }

    /**
     * Create team plan with specific duration
     */
    static createTeamPlan(
        teamId: bigint,
        durationDays = 30,
        overrides?: Partial<Prisma.PlanCreateInput>,
    ): Prisma.PlanCreateInput {
        return {
            id: generatePK(),
            enabledAt: new Date(),
            expiresAt: new Date(Date.now() + durationDays * 24 * 60 * 60 * 1000),
            team: { connect: { id: teamId } },
            ...overrides,
        };
    }

    /**
     * Create trial plan
     */
    static createTrial(
        userId: bigint,
        trialStartDaysAgo = 0,
        overrides?: Partial<Prisma.PlanCreateInput>,
    ): Prisma.PlanCreateInput {
        return {
            id: generatePK(),
            receivedFreeTrialAt: new Date(Date.now() - trialStartDaysAgo * 24 * 60 * 60 * 1000),
            user: { connect: { id: userId } },
            ...overrides,
        };
    }

    /**
     * Create expired plan
     */
    static createExpired(
        userId: bigint,
        expiredDaysAgo = 5,
        overrides?: Partial<Prisma.PlanCreateInput>,
    ): Prisma.PlanCreateInput {
        return {
            id: generatePK(),
            enabledAt: new Date(Date.now() - (expiredDaysAgo + 30) * 24 * 60 * 60 * 1000),
            expiresAt: new Date(Date.now() - expiredDaysAgo * 24 * 60 * 60 * 1000),
            user: { connect: { id: userId } },
            ...overrides,
        };
    }

    /**
     * Create custom plan with features
     */
    static createCustom(
        entityId: bigint,
        entityType: "user" | "team",
        features: string[],
        limits: Record<string, any>,
        overrides?: Partial<Prisma.PlanCreateInput>,
    ): Prisma.PlanCreateInput {
        const customPlan = JSON.stringify({
            features,
            limits,
            tier: "custom",
        });

        return {
            id: generatePK(),
            enabledAt: new Date(),
            expiresAt: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000), // 1 year
            customPlan,
            ...(entityType === "user" 
                ? { user: { connect: { id: entityId } } }
                : { team: { connect: { id: entityId } } }
            ),
            ...overrides,
        };
    }
}

/**
 * Helper functions for seeding test data
 */

/**
 * Seed basic plan scenarios for testing
 */
export async function seedTestPlans(db: any) {
    // Create users and teams first
    const user1 = await db.user.create({
        data: {
            id: generatePK(),
            publicId: "test_user_1",
            name: "Plan Test User 1",
            handle: "planuser1",
            status: "Unlocked",
            isBot: false,
            isBotDepictingPerson: false,
            isPrivate: false,
        },
    });

    const user2 = await db.user.create({
        data: {
            id: generatePK(),
            publicId: "test_user_2", 
            name: "Plan Test User 2",
            handle: "planuser2",
            status: "Unlocked",
            isBot: false,
            isBotDepictingPerson: false,
            isPrivate: false,
        },
    });

    const team1 = await db.team.create({
        data: {
            id: generatePK(),
            publicId: "test_team_1",
            handle: "planteam1",
            createdById: user1.id,
        },
    });

    // Create plans
    const plans = await Promise.all([
        db.plan.create({
            data: PlanDbFactory.createUserPlan(user1.id, 30),
        }),
        db.plan.create({
            data: PlanDbFactory.createExpired(user2.id, 5),
        }),
        db.plan.create({
            data: PlanDbFactory.createTeamPlan(team1.id, 365),
        }),
        db.plan.create({
            data: PlanDbFactory.createTrial(user1.id, 3),
        }),
    ]);

    return { user1, user2, team1, plans };
}
