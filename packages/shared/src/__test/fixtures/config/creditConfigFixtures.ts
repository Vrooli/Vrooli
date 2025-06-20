import {
    type CreditConfigObject,
    DEFAULT_DONATION_SETTINGS,
    DEFAULT_ROLLOVER_SETTINGS,
} from "../../../shape/configs/credit.js";
import { type ConfigTestFixtures } from "./baseConfigFixtures.js";

const LATEST_CONFIG_VERSION = "1.0";

/**
 * Credit configuration fixtures for testing credit rollover and donation settings
 */
export const creditConfigFixtures: ConfigTestFixtures<CreditConfigObject> = {
    minimal: {
        __version: LATEST_CONFIG_VERSION,
        rollover: DEFAULT_ROLLOVER_SETTINGS,
        donation: DEFAULT_DONATION_SETTINGS,
    },

    complete: {
        __version: LATEST_CONFIG_VERSION,
        rollover: {
            enabled: true,
            maxMonthsToKeep: 6,
            lastProcessedMonth: "2024-01",
        },
        donation: {
            enabled: true,
            percentage: 15,
            minThreshold: 5000,
            recipientType: "platform",
            lastProcessedMonth: "2024-01",
        },
    },

    withDefaults: {
        __version: LATEST_CONFIG_VERSION,
        rollover: DEFAULT_ROLLOVER_SETTINGS,
        donation: DEFAULT_DONATION_SETTINGS,
    },

    invalid: {
        missingVersion: {
            // Missing __version
            rollover: DEFAULT_ROLLOVER_SETTINGS,
            donation: DEFAULT_DONATION_SETTINGS,
        },
        invalidVersion: {
            __version: "0.5", // Invalid version
            rollover: DEFAULT_ROLLOVER_SETTINGS,
            donation: DEFAULT_DONATION_SETTINGS,
        },
        malformedStructure: {
            __version: LATEST_CONFIG_VERSION,
            rollover: "invalid", // Wrong type
            donation: DEFAULT_DONATION_SETTINGS,
        },
        invalidTypes: {
            __version: LATEST_CONFIG_VERSION,
            rollover: {
                enabled: false,
                maxMonthsToKeep: 0,
            },
            donation: {
                enabled: true,
                percentage: 0,
                minThreshold: -100,
                recipientType: "platform",
            },
        },
    },

    variants: {
        rolloverOnlyEnabled: {
            __version: LATEST_CONFIG_VERSION,
            rollover: {
                enabled: true,
                maxMonthsToKeep: 3,
            },
            donation: DEFAULT_DONATION_SETTINGS,
        },

        donationOnlyEnabled: {
            __version: LATEST_CONFIG_VERSION,
            rollover: DEFAULT_ROLLOVER_SETTINGS,
            donation: {
                enabled: true,
                percentage: 10,
                minThreshold: 1000,
                recipientType: "platform",
            },
        },

        bothEnabledHighDonation: {
            __version: LATEST_CONFIG_VERSION,
            rollover: {
                enabled: true,
                maxMonthsToKeep: 12,
                lastProcessedMonth: "2023-12",
            },
            donation: {
                enabled: true,
                percentage: 50,
                minThreshold: 10000,
                recipientType: "community",
                lastProcessedMonth: "2023-12",
            },
        },

        bothEnabledLowThreshold: {
            __version: LATEST_CONFIG_VERSION,
            rollover: {
                enabled: true,
                maxMonthsToKeep: 1,
            },
            donation: {
                enabled: true,
                percentage: 5,
                minThreshold: 100,
                recipientType: "platform",
            },
        },

        maxRolloverMonths: {
            __version: LATEST_CONFIG_VERSION,
            rollover: {
                enabled: true,
                maxMonthsToKeep: 12,
            },
            donation: DEFAULT_DONATION_SETTINGS,
        },

        minRolloverMonths: {
            __version: LATEST_CONFIG_VERSION,
            rollover: {
                enabled: true,
                maxMonthsToKeep: 1,
            },
            donation: DEFAULT_DONATION_SETTINGS,
        },

        zeroPercentDonation: {
            __version: LATEST_CONFIG_VERSION,
            rollover: DEFAULT_ROLLOVER_SETTINGS,
            donation: {
                enabled: true,
                percentage: 0,
                minThreshold: 1000,
                recipientType: "platform",
            },
        },

        maxPercentDonation: {
            __version: LATEST_CONFIG_VERSION,
            rollover: DEFAULT_ROLLOVER_SETTINGS,
            donation: {
                enabled: true,
                percentage: 100,
                minThreshold: 50000,
                recipientType: "community",
            },
        },

        processedLastMonth: {
            __version: LATEST_CONFIG_VERSION,
            rollover: {
                enabled: true,
                maxMonthsToKeep: 6,
                lastProcessedMonth: "2024-06",
            },
            donation: {
                enabled: true,
                percentage: 20,
                minThreshold: 2000,
                recipientType: "platform",
                lastProcessedMonth: "2024-06",
            },
        },
    },
};

/**
 * Create a credit config with specific rollover settings
 */
export function createCreditConfigWithRollover(
    maxMonthsToKeep: number,
    lastProcessedMonth?: string,
): CreditConfigObject {
    return {
        __version: LATEST_CONFIG_VERSION,
        rollover: {
            enabled: true,
            maxMonthsToKeep,
            lastProcessedMonth,
        },
        donation: DEFAULT_DONATION_SETTINGS,
    };
}

/**
 * Create a credit config with specific donation settings
 */
export function createCreditConfigWithDonation(
    percentage: number,
    minThreshold: number,
    recipientType: "platform" | "community" = "platform",
): CreditConfigObject {
    return {
        __version: LATEST_CONFIG_VERSION,
        rollover: DEFAULT_ROLLOVER_SETTINGS,
        donation: {
            enabled: true,
            percentage,
            minThreshold,
            recipientType,
        },
    };
}

/**
 * Create a credit config for testing edge cases
 */
export function createCreditConfigForBoundaryTesting(
    rolloverMonths?: number,
    donationPercentage?: number,
): CreditConfigObject {
    return {
        __version: LATEST_CONFIG_VERSION,
        rollover: rolloverMonths !== undefined ? {
            enabled: true,
            maxMonthsToKeep: rolloverMonths,
        } : DEFAULT_ROLLOVER_SETTINGS,
        donation: donationPercentage !== undefined ? {
            enabled: true,
            percentage: donationPercentage,
            minThreshold: 1000,
            recipientType: "platform",
        } : DEFAULT_DONATION_SETTINGS,
    };
}
