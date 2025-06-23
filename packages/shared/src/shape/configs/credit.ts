// Credit config doesn't need resources, so we define our own simple interface

const LATEST_CONFIG_VERSION = "1.0";

export const DEFAULT_ROLLOVER_SETTINGS = {
    enabled: false,
    maxMonthsToKeep: 6,
};

export const DEFAULT_DONATION_SETTINGS = {
    enabled: false,
    percentage: 10,
    minThreshold: 1000, // Minimum credits before donation kicks in
    recipientType: "platform" as const,
};

export interface CreditRolloverSettings {
    enabled: boolean;
    maxMonthsToKeep: number;
    lastProcessedMonth?: string; // Format: "YYYY-MM"
}

export interface CreditDonationSettings {
    enabled: boolean;
    percentage: number; // 0-100
    minThreshold: number; // Minimum credits required to trigger donation
    recipientType: "platform" | "community"; // Future expansion
    lastProcessedMonth?: string; // Format: "YYYY-MM"
}

export interface CreditConfigObject {
    /** Store the version number for future compatibility */
    __version: string;
    rollover: CreditRolloverSettings;
    donation: CreditDonationSettings;
}

export class CreditConfig {
    __version: string;
    rollover: CreditRolloverSettings;
    donation: CreditDonationSettings;

    constructor(existing: Partial<CreditConfigObject> = {}) {
        this.__version = existing.__version ?? LATEST_CONFIG_VERSION;
        this.rollover = existing.rollover ? { ...DEFAULT_ROLLOVER_SETTINGS, ...existing.rollover } : DEFAULT_ROLLOVER_SETTINGS;
        this.donation = existing.donation ? { ...DEFAULT_DONATION_SETTINGS, ...existing.donation } : DEFAULT_DONATION_SETTINGS;
    }

    toObject(): CreditConfigObject {
        return {
            __version: this.__version,
            rollover: this.rollover,
            donation: this.donation,
        };
    }

    /**
     * Updates rollover settings
     */
    updateRolloverSettings(settings: Partial<CreditRolloverSettings>): void {
        const updatedSettings = { ...this.rollover, ...settings };
        
        // Validate maxMonthsToKeep bounds
        if (updatedSettings.maxMonthsToKeep !== undefined) {
            if (updatedSettings.maxMonthsToKeep < 1 || updatedSettings.maxMonthsToKeep > 12) {
                throw new Error("maxMonthsToKeep must be between 1 and 12");
            }
        }
        
        this.rollover = updatedSettings;
    }

    /**
     * Updates donation settings
     */
    updateDonationSettings(settings: Partial<CreditDonationSettings>): void {
        const updatedSettings = { ...this.donation, ...settings };
        
        // Validate donation percentage bounds
        if (updatedSettings.percentage !== undefined) {
            if (updatedSettings.percentage < 0 || updatedSettings.percentage > 100) {
                throw new Error("donation percentage must be between 0 and 100");
            }
        }
        
        // Validate minimum threshold
        if (updatedSettings.minThreshold !== undefined) {
            if (updatedSettings.minThreshold < 0) {
                throw new Error("minThreshold cannot be negative");
            }
        }
        
        this.donation = updatedSettings;
    }

    /**
     * Validates donation percentage is within valid range
     */
    validateDonationPercentage(percentage: number): boolean {
        return percentage >= 0 && percentage <= 100;
    }

    /**
     * Calculates donation amount based on available credits
     */
    calculateDonationAmount(availableCredits: number, monthlyAllocation: number): number {
        // Validate inputs
        if (availableCredits < 0) {
            throw new Error("Available credits cannot be negative");
        }
        if (monthlyAllocation < 0) {
            throw new Error("Monthly allocation cannot be negative");
        }
        if (!this.validateDonationPercentage(this.donation.percentage)) {
            throw new Error(`Invalid donation percentage: ${this.donation.percentage}. Must be between 0 and 100.`);
        }
        
        if (!this.donation.enabled || availableCredits < this.donation.minThreshold) {
            return 0;
        }
        
        // Only donate from monthly allocation, not purchased credits
        const eligibleAmount = Math.min(availableCredits, monthlyAllocation);
        return Math.floor((eligibleAmount * this.donation.percentage) / 100);
    }

    /**
     * Determines if rollover should be processed for a given month
     */
    shouldProcessRollover(currentMonth: string): boolean {
        return this.rollover.enabled === true && 
               this.rollover.lastProcessedMonth !== currentMonth;
    }

    /**
     * Determines if donation should be processed for a given month
     */
    shouldProcessDonation(currentMonth: string): boolean {
        return this.donation.enabled === true && 
               this.donation.lastProcessedMonth !== currentMonth;
    }
}
