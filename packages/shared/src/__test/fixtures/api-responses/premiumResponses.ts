/* c8 ignore start */
/**
 * Premium API Response Fixtures
 * 
 * Comprehensive fixtures for premium subscription and billing management including
 * subscription plans, payment processing, feature access, and billing lifecycle.
 */

import type {
    Premium,
    PremiumCreateInput,
    PremiumUpdateInput,
} from "../../../api/types.js";
import { generatePK } from "../../../id/index.js";
import { BaseAPIResponseFactory } from "./base.js";
import type { MockDataOptions } from "./types.js";

// Constants
const DEFAULT_COUNT = 10;
const DEFAULT_ERROR_RATE = 0.1;
const DEFAULT_DELAY_MS = 500;
const MILLISECONDS_PER_HOUR = 60 * 60 * 1000;
const MILLISECONDS_PER_DAY = 24 * MILLISECONDS_PER_HOUR;
const HOURS_IN_2 = 2;
const HOURS_IN_6 = 6;
const DAYS_IN_1 = 1;
const DAYS_IN_7 = 7;
const DAYS_IN_30 = 30;
const DAYS_IN_365 = 365;

// Premium tier configurations
const PREMIUM_TIERS = {
    free: {
        name: "Free",
        credits: 1000,
        maxProjects: 5,
        maxTeamMembers: 2,
        maxApiCalls: 100,
        advancedFeatures: false,
        prioritySupport: false,
        customBranding: false,
    },
    monthly: {
        name: "Premium Monthly",
        credits: 10000,
        maxProjects: 100,
        maxTeamMembers: 25,
        maxApiCalls: 10000,
        advancedFeatures: true,
        prioritySupport: true,
        customBranding: true,
    },
    yearly: {
        name: "Premium Yearly",
        credits: 120000,
        maxProjects: 500,
        maxTeamMembers: 100,
        maxApiCalls: 100000,
        advancedFeatures: true,
        prioritySupport: true,
        customBranding: true,
    },
    enterprise: {
        name: "Enterprise",
        credits: 1000000,
        maxProjects: -1, // Unlimited
        maxTeamMembers: -1, // Unlimited
        maxApiCalls: -1, // Unlimited
        advancedFeatures: true,
        prioritySupport: true,
        customBranding: true,
    },
} as const;

// Premium statuses
const PREMIUM_STATUSES = [
    "active",
    "expired",
    "cancelled",
    "pending",
    "trialing",
    "past_due",
    "unpaid",
] as const;

/**
 * Premium API response factory
 */
export class PremiumResponseFactory extends BaseAPIResponseFactory<
    Premium,
    PremiumCreateInput,
    PremiumUpdateInput
> {
    protected readonly entityName = "premium";

    /**
     * Create mock premium data
     */
    createMockData(options?: MockDataOptions): Premium {
        const scenario = options?.scenario || "minimal";
        const now = new Date().toISOString();
        const premiumId = options?.overrides?.id || generatePK().toString();

        // Calculate expiration dates based on scenario
        const enabledAt = new Date(Date.now().toISOString() - (DAYS_IN_7 * MILLISECONDS_PER_DAY)).toISOString();
        const expiresAt = new Date(Date.now().toISOString() + (DAYS_IN_30 * MILLISECONDS_PER_DAY)).toISOString();

        const basePremium: Premium = {
            __typename: "Premium",
            id: premiumId,
            credits: PREMIUM_TIERS.monthly.credits,
            customPlan: null,
            enabledAt,
            expiresAt,
        };

        if (scenario === "complete" || scenario === "edge-case") {
            const isEdgeCase = scenario === "edge-case";
            const tierConfig = isEdgeCase ? PREMIUM_TIERS.enterprise : PREMIUM_TIERS.yearly;

            return {
                ...basePremium,
                credits: tierConfig.credits,
                customPlan: isEdgeCase ? "Custom Enterprise Plan with unlimited everything" : null,
                enabledAt: isEdgeCase
                    ? new Date(Date.now().toISOString() - (DAYS_IN_365 * MILLISECONDS_PER_DAY)).toISOString() // 1 year ago
                    : new Date(Date.now().toISOString() - (DAYS_IN_30 * MILLISECONDS_PER_DAY)).toISOString(), // 30 days ago
                expiresAt: isEdgeCase
                    ? new Date(Date.now().toISOString() - (DAYS_IN_1 * MILLISECONDS_PER_DAY)).toISOString() // Expired 1 day ago
                    : new Date(Date.now().toISOString() + (DAYS_IN_365 * MILLISECONDS_PER_DAY)).toISOString(), // 1 year from now
                ...options?.overrides,
            };
        }

        return {
            ...basePremium,
            ...options?.overrides,
        };
    }

    /**
     * Create premium from input
     */
    createFromInput(input: PremiumCreateInput): Premium {
        const now = new Date().toISOString();
        const premiumId = generatePK().toString();

        return {
            __typename: "Premium",
            id: premiumId,
            credits: input.credits || PREMIUM_TIERS.monthly.credits,
            customPlan: input.customPlan || null,
            enabledAt: now,
            expiresAt: input.expiresAt || new Date(Date.now().toISOString() + (DAYS_IN_30 * MILLISECONDS_PER_DAY)).toISOString(),
        };
    }

    /**
     * Update premium from input
     */
    updateFromInput(existing: Premium, input: PremiumUpdateInput): Premium {
        const updates: Partial<Premium> = {};

        if (input.credits !== undefined) updates.credits = input.credits;
        if (input.customPlan !== undefined) updates.customPlan = input.customPlan;
        if (input.expiresAt !== undefined) updates.expiresAt = input.expiresAt;

        return {
            ...existing,
            ...updates,
        };
    }

    /**
     * Validate create input
     */
    async validateCreateInput(input: PremiumCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (input.credits !== undefined && input.credits < 0) {
            errors.credits = "Credits must be a positive number";
        }

        if (input.customPlan && input.customPlan.length > 500) {
            errors.customPlan = "Custom plan description must be 500 characters or less";
        }

        if (input.expiresAt) {
            const expiryDate = new Date(input.expiresAt).toISOString();
            const now = new Date().toISOString();
            if (expiryDate <= now) {
                errors.expiresAt = "Expiration date must be in the future";
            }
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Validate update input
     */
    async validateUpdateInput(input: PremiumUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (input.credits !== undefined && input.credits < 0) {
            errors.credits = "Credits must be a positive number";
        }

        if (input.customPlan !== undefined && input.customPlan && input.customPlan.length > 500) {
            errors.customPlan = "Custom plan description must be 500 characters or less";
        }

        if (input.expiresAt) {
            const expiryDate = new Date(input.expiresAt).toISOString();
            const now = new Date().toISOString();
            if (expiryDate <= now) {
                errors.expiresAt = "Expiration date must be in the future";
            }
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create premium subscriptions for different tiers
     */
    createPremiumForAllTiers(): Premium[] {
        return Object.entries(PREMIUM_TIERS).map(([tierKey, tierConfig], index) => {
            const daysOffset = index * DAYS_IN_7;
            const isActive = tierKey !== "free";

            return this.createMockData({
                overrides: {
                    id: `premium_${tierKey}_${index}`,
                    credits: tierConfig.credits,
                    customPlan: tierKey === "enterprise" ? `${tierConfig.name} - Unlimited Access` : null,
                    enabledAt: isActive
                        ? new Date(Date.now().toISOString() - (daysOffset * MILLISECONDS_PER_DAY)).toISOString()
                        : null,
                    expiresAt: isActive
                        ? new Date(Date.now().toISOString() + ((DAYS_IN_30 + daysOffset) * MILLISECONDS_PER_DAY)).toISOString()
                        : null,
                },
            });
        });
    }

    /**
     * Create premium subscriptions with different statuses
     */
    createPremiumForAllStatuses(): Premium[] {
        return PREMIUM_STATUSES.map((status, index) => {
            const baseTime = Date.now();
            const daysOffset = index * DAYS_IN_7;

            let enabledAt: string | null = null;
            let expiresAt: string | null = null;
            let credits = PREMIUM_TIERS.monthly.credits;

            switch (status) {
                case "active":
                    enabledAt = new Date(baseTime - (DAYS_IN_7 * MILLISECONDS_PER_DAY).toISOString()).toISOString();
                    expiresAt = new Date(baseTime + (DAYS_IN_30 * MILLISECONDS_PER_DAY).toISOString()).toISOString();
                    credits = PREMIUM_TIERS.yearly.credits;
                    break;
                case "expired":
                    enabledAt = new Date(baseTime - (DAYS_IN_60 * MILLISECONDS_PER_DAY).toISOString()).toISOString();
                    expiresAt = new Date(baseTime - (DAYS_IN_1 * MILLISECONDS_PER_DAY).toISOString()).toISOString();
                    credits = 0;
                    break;
                case "cancelled":
                    enabledAt = new Date(baseTime - (DAYS_IN_30 * MILLISECONDS_PER_DAY).toISOString()).toISOString();
                    expiresAt = new Date(baseTime + (DAYS_IN_7 * MILLISECONDS_PER_DAY).toISOString()).toISOString(); // Grace period
                    break;
                case "trialing":
                    enabledAt = new Date(baseTime - (DAYS_IN_1 * MILLISECONDS_PER_DAY).toISOString()).toISOString();
                    expiresAt = new Date(baseTime + (DAYS_IN_7 * MILLISECONDS_PER_DAY).toISOString()).toISOString(); // 7-day trial
                    credits = PREMIUM_TIERS.monthly.credits;
                    break;
                case "pending":
                    credits = PREMIUM_TIERS.free.credits;
                    break;
                case "past_due":
                    enabledAt = new Date(baseTime - (DAYS_IN_30 * MILLISECONDS_PER_DAY).toISOString()).toISOString();
                    expiresAt = new Date(baseTime - (DAYS_IN_3 * MILLISECONDS_PER_DAY).toISOString()).toISOString();
                    credits = PREMIUM_TIERS.monthly.credits;
                    break;
                case "unpaid":
                    enabledAt = new Date(baseTime - (DAYS_IN_30 * MILLISECONDS_PER_DAY).toISOString()).toISOString();
                    expiresAt = new Date(baseTime - (DAYS_IN_7 * MILLISECONDS_PER_DAY).toISOString()).toISOString();
                    credits = 0;
                    break;
            }

            return this.createMockData({
                overrides: {
                    id: `premium_${status}_${index}`,
                    credits,
                    enabledAt,
                    expiresAt,
                    customPlan: status === "pending" ? "Pending approval for custom enterprise plan" : null,
                },
            });
        });
    }

    /**
     * Create premium subscriptions with different expiration scenarios
     */
    createPremiumExpirationScenarios(): Premium[] {
        const baseTime = Date.now();

        return [
            // Expires in 30 days
            this.createMockData({
                overrides: {
                    id: "premium_expires_30_days",
                    credits: PREMIUM_TIERS.monthly.credits,
                    enabledAt: new Date(baseTime - (DAYS_IN_30 * MILLISECONDS_PER_DAY).toISOString()).toISOString(),
                    expiresAt: new Date(baseTime + (DAYS_IN_30 * MILLISECONDS_PER_DAY).toISOString()).toISOString(),
                },
            }),

            // Expires in 7 days (needs renewal)
            this.createMockData({
                overrides: {
                    id: "premium_expires_7_days",
                    credits: PREMIUM_TIERS.yearly.credits,
                    enabledAt: new Date(baseTime - (DAYS_IN_365 * MILLISECONDS_PER_DAY).toISOString()).toISOString(),
                    expiresAt: new Date(baseTime + (DAYS_IN_7 * MILLISECONDS_PER_DAY).toISOString()).toISOString(),
                },
            }),

            // Expires tomorrow (urgent renewal)
            this.createMockData({
                overrides: {
                    id: "premium_expires_tomorrow",
                    credits: PREMIUM_TIERS.monthly.credits,
                    enabledAt: new Date(baseTime - (DAYS_IN_30 * MILLISECONDS_PER_DAY).toISOString()).toISOString(),
                    expiresAt: new Date(baseTime + (DAYS_IN_1 * MILLISECONDS_PER_DAY).toISOString()).toISOString(),
                },
            }),

            // Expired yesterday
            this.createMockData({
                overrides: {
                    id: "premium_expired_yesterday",
                    credits: 0,
                    enabledAt: new Date(baseTime - (DAYS_IN_30 * MILLISECONDS_PER_DAY).toISOString()).toISOString(),
                    expiresAt: new Date(baseTime - (DAYS_IN_1 * MILLISECONDS_PER_DAY).toISOString()).toISOString(),
                },
            }),

            // Long-term active (enterprise)
            this.createMockData({
                overrides: {
                    id: "premium_longterm_enterprise",
                    credits: PREMIUM_TIERS.enterprise.credits,
                    customPlan: "Enterprise Annual Contract with unlimited access",
                    enabledAt: new Date(baseTime - (DAYS_IN_365 * MILLISECONDS_PER_DAY).toISOString()).toISOString(),
                    expiresAt: new Date(baseTime + (DAYS_IN_365 * MILLISECONDS_PER_DAY).toISOString()).toISOString(),
                },
            }),
        ];
    }

    /**
     * Create usage tracking premiums (showing credit consumption)
     */
    createPremiumUsageScenarios(): Premium[] {
        const baseTime = Date.now();

        return [
            // High usage - low credits remaining
            this.createMockData({
                overrides: {
                    id: "premium_high_usage",
                    credits: 500, // Low remaining from 10000
                    enabledAt: new Date(baseTime - (DAYS_IN_15 * MILLISECONDS_PER_DAY).toISOString()).toISOString(),
                    expiresAt: new Date(baseTime + (DAYS_IN_15 * MILLISECONDS_PER_DAY).toISOString()).toISOString(),
                },
            }),

            // Medium usage
            this.createMockData({
                overrides: {
                    id: "premium_medium_usage",
                    credits: 5000, // Half remaining from 10000
                    enabledAt: new Date(baseTime - (DAYS_IN_10 * MILLISECONDS_PER_DAY).toISOString()).toISOString(),
                    expiresAt: new Date(baseTime + (DAYS_IN_20 * MILLISECONDS_PER_DAY).toISOString()).toISOString(),
                },
            }),

            // Low usage - most credits remaining
            this.createMockData({
                overrides: {
                    id: "premium_low_usage",
                    credits: 9500, // Most remaining from 10000
                    enabledAt: new Date(baseTime - (DAYS_IN_3 * MILLISECONDS_PER_DAY).toISOString()).toISOString(),
                    expiresAt: new Date(baseTime + (DAYS_IN_27 * MILLISECONDS_PER_DAY).toISOString()).toISOString(),
                },
            }),

            // No credits remaining
            this.createMockData({
                overrides: {
                    id: "premium_no_credits",
                    credits: 0,
                    enabledAt: new Date(baseTime - (DAYS_IN_20 * MILLISECONDS_PER_DAY).toISOString()).toISOString(),
                    expiresAt: new Date(baseTime + (DAYS_IN_10 * MILLISECONDS_PER_DAY).toISOString()).toISOString(),
                },
            }),
        ];
    }

    /**
     * Create payment processing error response
     */
    createPaymentProcessingErrorResponse(reason: string) {
        return this.createBusinessErrorResponse("payment_failed", {
            resource: "premium",
            reason,
            retryable: true,
            supportedMethods: ["card", "bank_transfer", "paypal"],
            message: `Payment processing failed: ${reason}`,
        });
    }

    /**
     * Create subscription limit error response
     */
    createSubscriptionLimitErrorResponse(feature: string, limit: number, current: number) {
        return this.createBusinessErrorResponse("subscription_limit", {
            resource: "premium",
            feature,
            limit,
            current,
            upgradeRequired: true,
            message: `You have reached the ${feature} limit for your current plan (${current}/${limit})`,
        });
    }

    /**
     * Create billing cycle error response
     */
    createBillingCycleErrorResponse(reason: string) {
        return this.createBusinessErrorResponse("billing_error", {
            resource: "premium",
            reason,
            contactSupport: true,
            message: `Billing cycle error: ${reason}`,
        });
    }

    /**
     * Create plan downgrade restriction error
     */
    createDowngradeRestrictionErrorResponse(restriction: string) {
        return this.createBusinessErrorResponse("downgrade_restricted", {
            resource: "premium",
            restriction,
            contactSupport: true,
            message: `Cannot downgrade plan: ${restriction}`,
        });
    }
}

// Add missing constant
const DAYS_IN_15 = 15;
const DAYS_IN_60 = 60;

/**
 * Pre-configured premium response scenarios
 */
export const premiumResponseScenarios = {
    // Success scenarios
    createSuccess: (input?: Partial<PremiumCreateInput>) => {
        const factory = new PremiumResponseFactory();
        const defaultInput: PremiumCreateInput = {
            credits: PREMIUM_TIERS.monthly.credits,
            customPlan: null,
            expiresAt: new Date(Date.now().toISOString() + (DAYS_IN_30 * MILLISECONDS_PER_DAY)).toISOString(),
            ...input,
        };
        return factory.createSuccessResponse(
            factory.createFromInput(defaultInput),
        );
    },

    findSuccess: (premium?: Premium) => {
        const factory = new PremiumResponseFactory();
        return factory.createSuccessResponse(
            premium || factory.createMockData(),
        );
    },

    findCompleteSuccess: () => {
        const factory = new PremiumResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({ scenario: "complete" }),
        );
    },

    updateSuccess: (existing?: Premium, updates?: Partial<PremiumUpdateInput>) => {
        const factory = new PremiumResponseFactory();
        const premium = existing || factory.createMockData({ scenario: "complete" });
        const input: PremiumUpdateInput = {
            id: premium.id,
            ...updates,
        };
        return factory.createSuccessResponse(
            factory.updateFromInput(premium, input),
        );
    },

    listSuccess: (premiums?: Premium[]) => {
        const factory = new PremiumResponseFactory();
        return factory.createPaginatedResponse(
            premiums || Array.from({ length: DEFAULT_COUNT }, () => factory.createMockData()),
            { page: 1, totalCount: premiums?.length || DEFAULT_COUNT },
        );
    },

    allTiersSuccess: () => {
        const factory = new PremiumResponseFactory();
        const premiums = factory.createPremiumForAllTiers();
        return factory.createPaginatedResponse(
            premiums,
            { page: 1, totalCount: premiums.length },
        );
    },

    allStatusesSuccess: () => {
        const factory = new PremiumResponseFactory();
        const premiums = factory.createPremiumForAllStatuses();
        return factory.createPaginatedResponse(
            premiums,
            { page: 1, totalCount: premiums.length },
        );
    },

    expirationScenariosSuccess: () => {
        const factory = new PremiumResponseFactory();
        const premiums = factory.createPremiumExpirationScenarios();
        return factory.createPaginatedResponse(
            premiums,
            { page: 1, totalCount: premiums.length },
        );
    },

    usageScenariosSuccess: () => {
        const factory = new PremiumResponseFactory();
        const premiums = factory.createPremiumUsageScenarios();
        return factory.createPaginatedResponse(
            premiums,
            { page: 1, totalCount: premiums.length },
        );
    },

    activePremiumSuccess: () => {
        const factory = new PremiumResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: {
                    credits: PREMIUM_TIERS.yearly.credits,
                    enabledAt: new Date(Date.now().toISOString() - (DAYS_IN_30 * MILLISECONDS_PER_DAY)).toISOString(),
                    expiresAt: new Date(Date.now().toISOString() + (DAYS_IN_365 * MILLISECONDS_PER_DAY)).toISOString(),
                },
            }),
        );
    },

    expiredPremiumSuccess: () => {
        const factory = new PremiumResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: {
                    credits: 0,
                    enabledAt: new Date(Date.now().toISOString() - (DAYS_IN_60 * MILLISECONDS_PER_DAY)).toISOString(),
                    expiresAt: new Date(Date.now().toISOString() - (DAYS_IN_1 * MILLISECONDS_PER_DAY)).toISOString(),
                },
            }),
        );
    },

    // Error scenarios
    createValidationError: () => {
        const factory = new PremiumResponseFactory();
        return factory.createValidationErrorResponse({
            credits: "Credits must be a positive number",
            expiresAt: "Expiration date must be in the future",
        });
    },

    notFoundError: (premiumId?: string) => {
        const factory = new PremiumResponseFactory();
        return factory.createNotFoundErrorResponse(
            premiumId || "non-existent-premium",
        );
    },

    permissionError: (operation?: string) => {
        const factory = new PremiumResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "manage",
            ["premium:admin"],
        );
    },

    paymentProcessingError: (reason = "Credit card declined") => {
        const factory = new PremiumResponseFactory();
        return factory.createPaymentProcessingErrorResponse(reason);
    },

    subscriptionLimitError: (feature = "projects", limit = 5, current = 5) => {
        const factory = new PremiumResponseFactory();
        return factory.createSubscriptionLimitErrorResponse(feature, limit, current);
    },

    billingCycleError: (reason = "Failed to process monthly renewal") => {
        const factory = new PremiumResponseFactory();
        return factory.createBillingCycleErrorResponse(reason);
    },

    downgradeRestrictionError: (restriction = "Active team subscriptions must be cancelled first") => {
        const factory = new PremiumResponseFactory();
        return factory.createDowngradeRestrictionErrorResponse(restriction);
    },

    creditsExhaustedError: () => {
        const factory = new PremiumResponseFactory();
        return factory.createSubscriptionLimitErrorResponse("API calls", 10000, 10000);
    },

    // MSW handlers
    handlers: {
        success: () => new PremiumResponseFactory().createMSWHandlers(),
        withErrors: function createWithErrors(errorRate?: number) {
            return new PremiumResponseFactory().createMSWHandlers({ errorRate: errorRate ?? DEFAULT_ERROR_RATE });
        },
        withDelay: function createWithDelay(delay?: number) {
            return new PremiumResponseFactory().createMSWHandlers({ delay: delay ?? DEFAULT_DELAY_MS });
        },
    },
};

// Export factory instance for direct use
export const premiumResponseFactory = new PremiumResponseFactory();

