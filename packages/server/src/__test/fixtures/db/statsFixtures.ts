import { generatePK } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for Stats models - used for seeding analytics test data
 * These follow Prisma's shape for database operations
 * 
 * Covers: stats_resource, stats_site, stats_team, stats_user
 */

// Consistent IDs for testing
export const statsDbIds = {
    // Resource stats
    resourceStats1: generatePK(),
    resourceStats2: generatePK(),
    popularResource: generatePK(),
    
    // Site stats
    siteStats1: generatePK(),
    siteStats2: generatePK(),
    monthlyStats: generatePK(),
    
    // Team stats
    teamStats1: generatePK(),
    teamStats2: generatePK(),
    activeTeamStats: generatePK(),
    
    // User stats
    userStats1: generatePK(),
    userStats2: generatePK(),
    activeUserStats: generatePK(),
};

/**
 * Resource Stats Fixtures
 */

/**
 * Basic resource stats
 */
export const basicResourceStatsDb: Prisma.StatsResourceCreateInput = {
    id: statsDbIds.resourceStats1,
    periodStart: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000), // 30 days ago
    periodEnd: new Date(Date.now() - 1 * 24 * 60 * 60 * 1000), // 1 day ago
    periodType: "Monthly",
    references: 15,
    referencedBy: 8,
    runsStarted: 45,
    runsCompleted: 38,
    runCompletionTimeAverage: 120.5,
    resource: {
        connect: { id: generatePK() },
    },
};

/**
 * Popular resource with high engagement
 */
export const popularResourceStatsDb: Prisma.StatsResourceCreateInput = {
    id: statsDbIds.popularResource,
    periodStart: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000), // 7 days ago
    periodEnd: new Date(),
    periodType: "Weekly",
    references: 125,
    referencedBy: 89,
    runsStarted: 340,
    runsCompleted: 245,
    runCompletionTimeAverage: 95.7,
    resource: {
        connect: { id: generatePK() },
    },
};

/**
 * Resource with detailed engagement metrics
 */
export const detailedResourceStatsDb: Prisma.StatsResourceCreateInput = {
    id: statsDbIds.resourceStats2,
    periodStart: new Date(Date.now() - 24 * 60 * 60 * 1000), // 24 hours ago
    periodEnd: new Date(),
    periodType: "Daily",
    references: 12,
    referencedBy: 8,
    runsStarted: 23,
    runsCompleted: 18,
    runCompletionTimeAverage: 142.5,
    resource: {
        connect: { id: generatePK() },
    },
};

/**
 * Site Stats Fixtures
 */

/**
 * Basic site stats
 */
export const basicSiteStatsDb: Prisma.StatsSiteCreateInput = {
    id: statsDbIds.siteStats1,
    periodStart: new Date(Date.now() - 24 * 60 * 60 * 1000), // 24 hours ago
    periodEnd: new Date(),
    periodType: "Daily",
    activeUsers: 1250,
    teamsCreated: 3,
    verifiedEmailsCreated: 18,
    verifiedWalletsCreated: 5,
};

/**
 * Monthly site stats with comprehensive metrics
 */
export const monthlySiteStatsDb: Prisma.StatsSiteCreateInput = {
    id: statsDbIds.monthlyStats,
    periodStart: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000), // 30 days ago
    periodEnd: new Date(),
    periodType: "Monthly",
    activeUsers: 15600,
    teamsCreated: 78,
    verifiedEmailsCreated: 567,
    verifiedWalletsCreated: 189,
};

/**
 * Team Stats Fixtures
 */

/**
 * Basic team stats
 */
export const basicTeamStatsDb: Prisma.StatsTeamCreateInput = {
    id: statsDbIds.teamStats1,
    periodStart: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000), // 7 days ago
    periodEnd: new Date(),
    periodType: "Weekly",
    resources: 12,
    members: 5,
    runsStarted: 45,
    runsCompleted: 38,
    runCompletionTimeAverage: 125.6,
    team: {
        connect: { id: generatePK() },
    },
};

/**
 * Active team with high productivity
 */
export const activeTeamStatsDb: Prisma.StatsTeamCreateInput = {
    id: statsDbIds.activeTeamStats,
    periodStart: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000), // 30 days ago
    periodEnd: new Date(),
    periodType: "Monthly",
    resources: 89,
    members: 23,
    runsStarted: 456,
    runsCompleted: 398,
    runCompletionTimeAverage: 98.7,
    team: {
        connect: { id: generatePK() },
    },
};

/**
 * User Stats Fixtures
 */

/**
 * Basic user stats
 */
export const basicUserStatsDb: Prisma.StatsUserCreateInput = {
    id: statsDbIds.userStats1,
    periodStart: new Date(Date.now() - 24 * 60 * 60 * 1000), // 24 hours ago
    periodEnd: new Date(),
    periodType: "Daily",
    resourcesCreatedByType: { "ROUTINE": 5, "PROJECT": 2, "CODE": 1 },
    resourcesCompletedByType: { "ROUTINE": 8, "PROJECT": 1 },
    resourceCompletionTimeAverageByType: { "ROUTINE": 95.2, "PROJECT": 450.0 },
    runsStarted: 12,
    runsCompleted: 9,
    user: {
        connect: { id: generatePK() },
    },
};

/**
 * Active user with comprehensive metrics
 */
export const activeUserStatsDb: Prisma.StatsUserCreateInput = {
    id: statsDbIds.activeUserStats,
    periodStart: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000), // 30 days ago
    periodEnd: new Date(),
    periodType: "Monthly",
    resourcesCreatedByType: { 
        "ROUTINE": 42, 
        "PROJECT": 15, 
        "CODE": 23, 
        "API": 8, 
        "NOTE": 89, 
        "STANDARD": 3, 
        "SMART_CONTRACT": 2 
    },
    resourcesCompletedByType: { 
        "ROUTINE": 198, 
        "PROJECT": 45, 
        "CODE": 67 
    },
    resourceCompletionTimeAverageByType: { 
        "ROUTINE": 125.6, 
        "PROJECT": 890.3, 
        "CODE": 45.8 
    },
    runsStarted: 234,
    runsCompleted: 198,
    user: {
        connect: { id: generatePK() },
    },
};

/**
 * Factory functions for creating stats dynamically
 */
export class StatsDbFactory {
    /**
     * Create resource stats
     */
    static createResourceStats(
        resourceId: bigint,
        periodType: "Daily" | "Weekly" | "Monthly" | "Yearly" = "Daily",
        daysAgo: number = 1,
        engagement: "low" | "medium" | "high" = "medium",
        overrides?: Partial<Prisma.StatsResourceCreateInput>,
    ): Prisma.StatsResourceCreateInput {
        const periodDays = { Daily: 1, Weekly: 7, Monthly: 30, Yearly: 365 }[periodType];
        const multiplier = { low: 0.3, medium: 1, high: 3 }[engagement];

        return {
            id: generatePK(),
            periodStart: new Date(Date.now() - (daysAgo + periodDays) * 24 * 60 * 60 * 1000),
            periodEnd: new Date(Date.now() - daysAgo * 24 * 60 * 60 * 1000),
            periodType,
            views: Math.floor(100 * periodDays * multiplier),
            downloads: Math.floor(20 * periodDays * multiplier),
            forks: Math.floor(3 * periodDays * multiplier),
            comments: Math.floor(8 * periodDays * multiplier),
            bookmarks: Math.floor(15 * periodDays * multiplier),
            votes: Math.floor(12 * periodDays * multiplier),
            score: 5 + (engagement === "high" ? 3 : engagement === "low" ? -2 : 0),
            completions: Math.floor(5 * periodDays * multiplier),
            resource: { connect: { id: resourceId } },
            ...overrides,
        };
    }

    /**
     * Create site stats
     */
    static createSiteStats(
        periodType: "Daily" | "Weekly" | "Monthly" | "Yearly" = "Daily",
        daysAgo: number = 1,
        activity: "low" | "medium" | "high" = "medium",
        overrides?: Partial<Prisma.StatsSiteCreateInput>,
    ): Prisma.StatsSiteCreateInput {
        const periodDays = { Daily: 1, Weekly: 7, Monthly: 30, Yearly: 365 }[periodType];
        const multiplier = { low: 0.5, medium: 1, high: 2 }[activity];

        return {
            id: generatePK(),
            periodStart: new Date(Date.now() - (daysAgo + periodDays) * 24 * 60 * 60 * 1000),
            periodEnd: new Date(Date.now() - daysAgo * 24 * 60 * 60 * 1000),
            periodType,
            usersCreated: Math.floor(20 * periodDays * multiplier),
            teamsCreated: Math.floor(2 * periodDays * multiplier),
            projectsCreated: Math.floor(15 * periodDays * multiplier),
            routinesCreated: Math.floor(40 * periodDays * multiplier),
            runsStarted: Math.floor(200 * periodDays * multiplier),
            runsCompleted: Math.floor(170 * periodDays * multiplier),
            apiCalls: Math.floor(10000 * periodDays * multiplier),
            apisCreated: Math.floor(5 * periodDays * multiplier),
            notesCreated: Math.floor(50 * periodDays * multiplier),
            smartContractsCreated: Math.floor(1 * periodDays * multiplier),
            standardsCreated: Math.floor(2 * periodDays * multiplier),
            codesCreated: Math.floor(25 * periodDays * multiplier),
            questionsCreated: Math.floor(30 * periodDays * multiplier),
            chatMessagesCreated: Math.floor(500 * periodDays * multiplier),
            botsCreated: Math.floor(1 * periodDays * multiplier),
            paymentsProcessed: Math.floor(8 * periodDays * multiplier),
            creditsEarned: BigInt(Math.floor(50000 * periodDays * multiplier)),
            creditsSpent: BigInt(Math.floor(30000 * periodDays * multiplier)),
            ...overrides,
        };
    }

    /**
     * Create team stats
     */
    static createTeamStats(
        teamId: bigint,
        periodType: "Daily" | "Weekly" | "Monthly" | "Yearly" = "Daily",
        daysAgo: number = 1,
        productivity: "low" | "medium" | "high" = "medium",
        overrides?: Partial<Prisma.StatsTeamCreateInput>,
    ): Prisma.StatsTeamCreateInput {
        const periodDays = { Daily: 1, Weekly: 7, Monthly: 30, Yearly: 365 }[periodType];
        const multiplier = { low: 0.4, medium: 1, high: 2.5 }[productivity];

        return {
            id: generatePK(),
            periodStart: new Date(Date.now() - (daysAgo + periodDays) * 24 * 60 * 60 * 1000),
            periodEnd: new Date(Date.now() - daysAgo * 24 * 60 * 60 * 1000),
            periodType,
            membersAdded: Math.floor(2 * periodDays * multiplier),
            projectsCreated: Math.floor(5 * periodDays * multiplier),
            routinesCreated: Math.floor(15 * periodDays * multiplier),
            runsStarted: Math.floor(50 * periodDays * multiplier),
            runsCompleted: Math.floor(40 * periodDays * multiplier),
            apisCreated: Math.floor(2 * periodDays * multiplier),
            notesCreated: Math.floor(20 * periodDays * multiplier),
            smartContractsCreated: Math.floor(1 * periodDays * multiplier),
            standardsCreated: Math.floor(1 * periodDays * multiplier),
            codesCreated: Math.floor(8 * periodDays * multiplier),
            questionsCreated: Math.floor(10 * periodDays * multiplier),
            chatMessagesCreated: Math.floor(100 * periodDays * multiplier),
            paymentsProcessed: Math.floor(3 * periodDays * multiplier),
            creditsEarned: BigInt(Math.floor(25000 * periodDays * multiplier)),
            creditsSpent: BigInt(Math.floor(15000 * periodDays * multiplier)),
            team: { connect: { id: teamId } },
            ...overrides,
        };
    }

    /**
     * Create user stats
     */
    static createUserStats(
        userId: bigint,
        periodType: "Daily" | "Weekly" | "Monthly" | "Yearly" = "Daily",
        daysAgo: number = 1,
        activity: "low" | "medium" | "high" = "medium",
        overrides?: Partial<Prisma.StatsUserCreateInput>,
    ): Prisma.StatsUserCreateInput {
        const periodDays = { Daily: 1, Weekly: 7, Monthly: 30, Yearly: 365 }[periodType];
        const multiplier = { low: 0.3, medium: 1, high: 2 }[activity];

        return {
            id: generatePK(),
            periodStart: new Date(Date.now() - (daysAgo + periodDays) * 24 * 60 * 60 * 1000),
            periodEnd: new Date(Date.now() - daysAgo * 24 * 60 * 60 * 1000),
            periodType,
            projectsCreated: Math.floor(3 * periodDays * multiplier),
            routinesCreated: Math.floor(8 * periodDays * multiplier),
            runsStarted: Math.floor(25 * periodDays * multiplier),
            runsCompleted: Math.floor(20 * periodDays * multiplier),
            apisCreated: Math.floor(1 * periodDays * multiplier),
            notesCreated: Math.floor(12 * periodDays * multiplier),
            smartContractsCreated: Math.floor(0.5 * periodDays * multiplier),
            standardsCreated: Math.floor(0.5 * periodDays * multiplier),
            codesCreated: Math.floor(5 * periodDays * multiplier),
            questionsCreated: Math.floor(6 * periodDays * multiplier),
            chatMessagesCreated: Math.floor(50 * periodDays * multiplier),
            paymentsProcessed: Math.floor(1 * periodDays * multiplier),
            creditsEarned: BigInt(Math.floor(10000 * periodDays * multiplier)),
            creditsSpent: BigInt(Math.floor(7000 * periodDays * multiplier)),
            user: { connect: { id: userId } },
            ...overrides,
        };
    }

    /**
     * Create time series stats for analytics testing
     */
    static createTimeSeries(
        type: "resource" | "site" | "team" | "user",
        entityId: bigint | null,
        days: number = 30,
        periodType: "Daily" | "Weekly" | "Monthly" = "Daily",
    ): Prisma.StatsResourceCreateInput[] | Prisma.StatsSiteCreateInput[] | Prisma.StatsTeamCreateInput[] | Prisma.StatsUserCreateInput[] {
        const stats: any[] = [];
        const periodDays = { Daily: 1, Weekly: 7, Monthly: 30 }[periodType];

        for (let i = 0; i < Math.ceil(days / periodDays); i++) {
            const daysAgo = i * periodDays;
            const activity = Math.random() > 0.7 ? "high" : Math.random() > 0.3 ? "medium" : "low";

            switch (type) {
                case "resource":
                    stats.push(this.createResourceStats(entityId!, periodType, daysAgo, activity));
                    break;
                case "site":
                    stats.push(this.createSiteStats(periodType, daysAgo, activity));
                    break;
                case "team":
                    stats.push(this.createTeamStats(entityId!, periodType, daysAgo, activity));
                    break;
                case "user":
                    stats.push(this.createUserStats(entityId!, periodType, daysAgo, activity));
                    break;
            }
        }

        return stats;
    }
}

/**
 * Helper functions for seeding test data
 */

/**
 * Seed comprehensive stats for testing analytics
 */
export async function seedTestStats(db: any) {
    // Create entities first
    const user1 = await db.user.create({
        data: {
            id: generatePK(),
            publicId: "stats_user_1",
            name: "Stats Test User 1",
            handle: "statsuser1",
            status: "Unlocked",
            isBot: false,
            isBotDepictingPerson: false,
            isPrivate: false,
        },
    });

    const team1 = await db.team.create({
        data: {
            id: generatePK(),
            publicId: "stats_team_1",
            handle: "statsteam1",
            createdById: user1.id,
        },
    });

    const resource1 = await db.resource.create({
        data: {
            id: generatePK(),
            index: 1,
            resourceType: "Project",
            createdById: user1.id,
        },
    });

    // Create stats
    const stats = await Promise.all([
        // Site stats
        db.statsSite.create({
            data: StatsDbFactory.createSiteStats("Daily", 0, "high"),
        }),
        db.statsSite.create({
            data: StatsDbFactory.createSiteStats("Weekly", 7, "medium"),
        }),
        
        // User stats
        db.statsUser.create({
            data: StatsDbFactory.createUserStats(user1.id, "Daily", 0, "high"),
        }),
        db.statsUser.create({
            data: StatsDbFactory.createUserStats(user1.id, "Monthly", 30, "medium"),
        }),
        
        // Team stats
        db.statsTeam.create({
            data: StatsDbFactory.createTeamStats(team1.id, "Weekly", 0, "high"),
        }),
        
        // Resource stats
        db.statsResource.create({
            data: StatsDbFactory.createResourceStats(resource1.id, "Daily", 0, "high"),
        }),
    ]);

    return { user1, team1, resource1, stats };
}

/**
 * Create analytics time series for dashboard testing
 */
export async function seedAnalyticsTimeSeries(db: any, entityId: bigint, entityType: "user" | "team" | "resource") {
    const timeSeries = StatsDbFactory.createTimeSeries(entityType, entityId, 30, "Daily");
    
    const modelName = `stats${entityType.charAt(0).toUpperCase() + entityType.slice(1)}`;
    const stats = await Promise.all(
        timeSeries.map(stat => db[modelName].create({ data: stat }))
    );

    return stats;
}