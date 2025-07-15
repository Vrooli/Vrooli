import { generatePK } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for Premium data - used for embedding in User/Team models
 * Premium is not a standalone Prisma model but a JSON field embedded in other models
 */

/**
 * Minimal premium data (active subscription)
 */
export const minimalPremiumDb = {
    credits: 100,
    enabledAt: new Date().toISOString(),
    expiresAt: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString(), // 30 days from now
};

/**
 * Complete premium data with custom plan
 */
export const completePremiumDb = {
    credits: 5000,
    customPlan: "enterprise",
    enabledAt: new Date().toISOString(),
    expiresAt: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000).toISOString(), // 1 year from now
};

/**
 * Expired premium subscription
 */
export const expiredPremiumDb = {
    credits: 0,
    enabledAt: new Date(Date.now() - 60 * 24 * 60 * 60 * 1000).toISOString(), // 60 days ago
    expiresAt: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(), // 30 days ago
};

/**
 * Trial premium subscription
 */
export const trialPremiumDb = {
    credits: 50,
    customPlan: "trial",
    enabledAt: new Date().toISOString(),
    expiresAt: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(), // 7 days from now
};

/**
 * Premium with low credits
 */
export const lowCreditsPremiumDb = {
    credits: 5,
    enabledAt: new Date().toISOString(),
    expiresAt: new Date(Date.now() + 15 * 24 * 60 * 60 * 1000).toISOString(), // 15 days from now
};

/**
 * Premium with no expiration (lifetime)
 */
export const lifetimePremiumDb = {
    credits: 10000,
    customPlan: "lifetime",
    enabledAt: new Date().toISOString(),
    expiresAt: null,
};

/**
 * Factory for creating premium data with overrides
 */
export class PremiumDbFactory {
    /**
     * Create minimal premium data
     */
    static createMinimal(overrides?: Partial<typeof minimalPremiumDb>) {
        return {
            ...minimalPremiumDb,
            ...overrides,
        };
    }

    /**
     * Create complete premium data
     */
    static createComplete(overrides?: Partial<typeof completePremiumDb>) {
        return {
            ...completePremiumDb,
            ...overrides,
        };
    }

    /**
     * Create premium data with specific expiration
     */
    static createWithExpiration(daysFromNow: number, overrides?: Partial<typeof minimalPremiumDb>) {
        const expiresAt = daysFromNow === 0 
            ? null // No expiration
            : new Date(Date.now() + daysFromNow * 24 * 60 * 60 * 1000).toISOString();
        
        return {
            ...minimalPremiumDb,
            expiresAt,
            ...overrides,
        };
    }

    /**
     * Create premium data with specific credits
     */
    static createWithCredits(credits: number, overrides?: Partial<typeof minimalPremiumDb>) {
        return {
            ...minimalPremiumDb,
            credits,
            ...overrides,
        };
    }

    /**
     * Create expired premium
     */
    static createExpired(overrides?: Partial<typeof expiredPremiumDb>) {
        return {
            ...expiredPremiumDb,
            ...overrides,
        };
    }

    /**
     * Create trial premium
     */
    static createTrial(trialDays = 7, overrides?: Partial<typeof trialPremiumDb>) {
        return {
            ...trialPremiumDb,
            expiresAt: new Date(Date.now() + trialDays * 24 * 60 * 60 * 1000).toISOString(),
            ...overrides,
        };
    }

    /**
     * Create custom plan premium
     */
    static createCustomPlan(
        planName: string,
        credits: number,
        expirationDays?: number,
        overrides?: Partial<typeof completePremiumDb>,
    ) {
        const baseData = {
            credits,
            customPlan: planName,
            enabledAt: new Date().toISOString(),
            expiresAt: expirationDays 
                ? new Date(Date.now() + expirationDays * 24 * 60 * 60 * 1000).toISOString()
                : null,
        };

        return {
            ...baseData,
            ...overrides,
        };
    }
}

/**
 * Helper to create a user with premium
 */
export function createUserWithPremium(
    prisma: any,
    userOverrides?: Partial<Prisma.userCreateInput>,
    premiumData = minimalPremiumDb,
): Prisma.userCreateInput {
    return {
        id: generatePK(),
        publicId: generatePK(),
        name: "Premium User",
        handle: `premium_user_${generatePK().slice(-6)}`,
        status: "Unlocked",
        isBot: false,
        isBotDepictingPerson: false,
        isPrivate: false,
        premium: premiumData,
        ...userOverrides,
    };
}

/**
 * Helper to create a team with premium
 */
export function createTeamWithPremium(
    prisma: any,
    teamOverrides?: Partial<Prisma.TeamCreateInput>,
    premiumData = minimalPremiumDb,
): Prisma.TeamCreateInput {
    return {
        id: generatePK(),
        publicId: generatePK(),
        createdBy: { connect: { id: generatePK() } }, // Will be overridden when used
        premium: premiumData,
        ...teamOverrides,
    };
}

/**
 * Helper to add premium to existing user/team
 */
export async function addPremiumToEntity(
    prisma: any,
    entityType: "user" | "team",
    entityId: string,
    premiumData = minimalPremiumDb,
) {
    const model = entityType === "user" ? prisma.user : prisma.team;
    
    return model.update({
        where: { id: entityId },
        data: { premium: premiumData },
    });
}

/**
 * Helper to check if premium is active
 */
export function isPremiumActive(premium: typeof minimalPremiumDb | null): boolean {
    if (!premium) return false;
    if (!premium.expiresAt) return true; // Lifetime premium
    
    const expirationDate = new Date(premium.expiresAt);
    return expirationDate > new Date();
}

/**
 * Helper to calculate remaining premium days
 */
export function getRemainingPremiumDays(premium: typeof minimalPremiumDb | null): number {
    if (!premium || !premium.expiresAt) return -1; // -1 means lifetime or no premium
    
    const expirationDate = new Date(premium.expiresAt);
    const now = new Date();
    
    if (expirationDate <= now) return 0;
    
    const diffTime = expirationDate.getTime() - now.getTime();
    const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
    
    return diffDays;
}

/**
 * Helper to deduct credits from premium
 */
export function deductCredits(
    premium: typeof minimalPremiumDb,
    amount: number,
): typeof minimalPremiumDb {
    return {
        ...premium,
        credits: Math.max(0, premium.credits - amount),
    };
}

/**
 * Premium scenarios for testing
 */
export const premiumScenarios = {
    // Active subscriptions
    activeMonthly: PremiumDbFactory.createWithExpiration(30),
    activeYearly: PremiumDbFactory.createWithExpiration(365),
    activeLifetime: PremiumDbFactory.createWithExpiration(0),
    
    // Trial states
    trialActive: PremiumDbFactory.createTrial(7),
    trialExpiringSoon: PremiumDbFactory.createTrial(1),
    
    // Credit states
    highCredits: PremiumDbFactory.createWithCredits(10000),
    lowCredits: PremiumDbFactory.createWithCredits(10),
    noCredits: PremiumDbFactory.createWithCredits(0),
    
    // Custom plans
    enterprise: PremiumDbFactory.createCustomPlan("enterprise", 50000, 365),
    startup: PremiumDbFactory.createCustomPlan("startup", 5000, 180),
    educational: PremiumDbFactory.createCustomPlan("educational", 2000, 365),
    
    // Expired states
    recentlyExpired: {
        ...expiredPremiumDb,
        expiresAt: new Date(Date.now() - 1 * 24 * 60 * 60 * 1000).toISOString(), // 1 day ago
    },
    longExpired: {
        ...expiredPremiumDb,
        expiresAt: new Date(Date.now() - 90 * 24 * 60 * 60 * 1000).toISOString(), // 90 days ago
    },
};
