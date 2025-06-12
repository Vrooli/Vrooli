import { describe, it, expect } from "vitest";
import { 
    CreditConfig, 
    type CreditConfigObject,
    type CreditRolloverSettings,
    type CreditDonationSettings,
    DEFAULT_ROLLOVER_SETTINGS,
    DEFAULT_DONATION_SETTINGS,
} from "./credit.js";

describe("CreditConfig", () => {
    describe("constructor", () => {
        it("should create CreditConfig with complete data", () => {
            const configData: CreditConfigObject = {
                __version: "1.0",
                rollover: {
                    enabled: true,
                    maxMonthsToKeep: 3,
                    lastProcessedMonth: "2024-01",
                },
                donation: {
                    enabled: true,
                    percentage: 15,
                    minThreshold: 500,
                    recipientType: "community",
                    lastProcessedMonth: "2024-01",
                },
            };

            const config = new CreditConfig(configData);

            expect(config.__version).toBe("1.0");
            expect(config.rollover.enabled).toBe(true);
            expect(config.rollover.maxMonthsToKeep).toBe(3);
            expect(config.rollover.lastProcessedMonth).toBe("2024-01");
            expect(config.donation.enabled).toBe(true);
            expect(config.donation.percentage).toBe(15);
            expect(config.donation.minThreshold).toBe(500);
            expect(config.donation.recipientType).toBe("community");
            expect(config.donation.lastProcessedMonth).toBe("2024-01");
        });

        it("should create CreditConfig with default values when no data provided", () => {
            const config = new CreditConfig();

            expect(config.__version).toBe("1.0");
            expect(config.rollover).toEqual(DEFAULT_ROLLOVER_SETTINGS);
            expect(config.donation).toEqual(DEFAULT_DONATION_SETTINGS);
        });

        it("should create CreditConfig with partial data and merge with defaults", () => {
            const partialData: Partial<CreditConfigObject> = {
                rollover: {
                    enabled: true,
                    maxMonthsToKeep: 12,
                },
                donation: {
                    enabled: false,
                    percentage: 20,
                    minThreshold: 2000,
                    recipientType: "platform",
                },
            };

            const config = new CreditConfig(partialData);

            expect(config.__version).toBe("1.0");
            expect(config.rollover.enabled).toBe(true);
            expect(config.rollover.maxMonthsToKeep).toBe(12);
            expect(config.rollover.lastProcessedMonth).toBeUndefined();
            expect(config.donation.enabled).toBe(false);
            expect(config.donation.percentage).toBe(20);
            expect(config.donation.minThreshold).toBe(2000);
            expect(config.donation.recipientType).toBe("platform");
        });

        it("should preserve lastProcessedMonth when merging with defaults", () => {
            const partialData: Partial<CreditConfigObject> = {
                rollover: {
                    enabled: true,
                    maxMonthsToKeep: 6,
                    lastProcessedMonth: "2024-02",
                },
            };

            const config = new CreditConfig(partialData);

            expect(config.rollover.lastProcessedMonth).toBe("2024-02");
        });
    });

    describe("toObject", () => {
        it("should export configuration as object", () => {
            const configData: CreditConfigObject = {
                __version: "1.0",
                rollover: {
                    enabled: true,
                    maxMonthsToKeep: 4,
                    lastProcessedMonth: "2024-01",
                },
                donation: {
                    enabled: true,
                    percentage: 25,
                    minThreshold: 750,
                    recipientType: "community",
                    lastProcessedMonth: "2024-01",
                },
            };

            const config = new CreditConfig(configData);
            const exported = config.toObject();

            expect(exported).toEqual(configData);
        });

        it("should export default configuration", () => {
            const config = new CreditConfig();
            const exported = config.toObject();

            expect(exported).toEqual({
                __version: "1.0",
                rollover: DEFAULT_ROLLOVER_SETTINGS,
                donation: DEFAULT_DONATION_SETTINGS,
            });
        });
    });

    describe("updateRolloverSettings", () => {
        it("should update rollover settings", () => {
            const config = new CreditConfig();

            config.updateRolloverSettings({
                enabled: true,
                maxMonthsToKeep: 9,
            });

            expect(config.rollover.enabled).toBe(true);
            expect(config.rollover.maxMonthsToKeep).toBe(9);
        });

        it("should update partial rollover settings", () => {
            const config = new CreditConfig({
                rollover: {
                    enabled: false,
                    maxMonthsToKeep: 3,
                    lastProcessedMonth: "2024-01",
                },
            });

            config.updateRolloverSettings({
                enabled: true,
            });

            expect(config.rollover.enabled).toBe(true);
            expect(config.rollover.maxMonthsToKeep).toBe(3);
            expect(config.rollover.lastProcessedMonth).toBe("2024-01");
        });

        it("should update lastProcessedMonth", () => {
            const config = new CreditConfig();

            config.updateRolloverSettings({
                lastProcessedMonth: "2024-03",
            });

            expect(config.rollover.lastProcessedMonth).toBe("2024-03");
        });

        it("should throw error for maxMonthsToKeep below 1", () => {
            const config = new CreditConfig();

            expect(() => {
                config.updateRolloverSettings({ maxMonthsToKeep: 0 });
            }).toThrow("maxMonthsToKeep must be between 1 and 12");

            expect(() => {
                config.updateRolloverSettings({ maxMonthsToKeep: -5 });
            }).toThrow("maxMonthsToKeep must be between 1 and 12");
        });

        it("should throw error for maxMonthsToKeep above 12", () => {
            const config = new CreditConfig();

            expect(() => {
                config.updateRolloverSettings({ maxMonthsToKeep: 13 });
            }).toThrow("maxMonthsToKeep must be between 1 and 12");

            expect(() => {
                config.updateRolloverSettings({ maxMonthsToKeep: 100 });
            }).toThrow("maxMonthsToKeep must be between 1 and 12");
        });

        it("should accept valid maxMonthsToKeep values", () => {
            const config = new CreditConfig();

            for (let months = 1; months <= 12; months++) {
                config.updateRolloverSettings({ maxMonthsToKeep: months });
                expect(config.rollover.maxMonthsToKeep).toBe(months);
            }
        });
    });

    describe("updateDonationSettings", () => {
        it("should update donation settings", () => {
            const config = new CreditConfig();

            config.updateDonationSettings({
                enabled: true,
                percentage: 30,
                minThreshold: 2500,
                recipientType: "community",
            });

            expect(config.donation.enabled).toBe(true);
            expect(config.donation.percentage).toBe(30);
            expect(config.donation.minThreshold).toBe(2500);
            expect(config.donation.recipientType).toBe("community");
        });

        it("should update partial donation settings", () => {
            const config = new CreditConfig({
                donation: {
                    enabled: true,
                    percentage: 15,
                    minThreshold: 1000,
                    recipientType: "platform",
                    lastProcessedMonth: "2024-01",
                },
            });

            config.updateDonationSettings({
                percentage: 25,
                minThreshold: 1500,
            });

            expect(config.donation.enabled).toBe(true);
            expect(config.donation.percentage).toBe(25);
            expect(config.donation.minThreshold).toBe(1500);
            expect(config.donation.recipientType).toBe("platform");
            expect(config.donation.lastProcessedMonth).toBe("2024-01");
        });

        it("should throw error for negative percentage", () => {
            const config = new CreditConfig();

            expect(() => {
                config.updateDonationSettings({ percentage: -1 });
            }).toThrow("donation percentage must be between 0 and 100");

            expect(() => {
                config.updateDonationSettings({ percentage: -50 });
            }).toThrow("donation percentage must be between 0 and 100");
        });

        it("should throw error for percentage above 100", () => {
            const config = new CreditConfig();

            expect(() => {
                config.updateDonationSettings({ percentage: 101 });
            }).toThrow("donation percentage must be between 0 and 100");

            expect(() => {
                config.updateDonationSettings({ percentage: 200 });
            }).toThrow("donation percentage must be between 0 and 100");
        });

        it("should accept valid percentage values", () => {
            const config = new CreditConfig();

            const validPercentages = [0, 1, 10, 25, 50, 75, 99, 100];
            for (const percentage of validPercentages) {
                config.updateDonationSettings({ percentage });
                expect(config.donation.percentage).toBe(percentage);
            }
        });

        it("should throw error for negative minThreshold", () => {
            const config = new CreditConfig();

            expect(() => {
                config.updateDonationSettings({ minThreshold: -1 });
            }).toThrow("minThreshold cannot be negative");

            expect(() => {
                config.updateDonationSettings({ minThreshold: -1000 });
            }).toThrow("minThreshold cannot be negative");
        });

        it("should accept zero and positive minThreshold", () => {
            const config = new CreditConfig();

            const validThresholds = [0, 1, 100, 1000, 10000, 1000000];
            for (const threshold of validThresholds) {
                config.updateDonationSettings({ minThreshold: threshold });
                expect(config.donation.minThreshold).toBe(threshold);
            }
        });
    });

    describe("validateDonationPercentage", () => {
        it("should return true for valid percentages", () => {
            const config = new CreditConfig();

            expect(config.validateDonationPercentage(0)).toBe(true);
            expect(config.validateDonationPercentage(10)).toBe(true);
            expect(config.validateDonationPercentage(50)).toBe(true);
            expect(config.validateDonationPercentage(100)).toBe(true);
        });

        it("should return false for invalid percentages", () => {
            const config = new CreditConfig();

            expect(config.validateDonationPercentage(-1)).toBe(false);
            expect(config.validateDonationPercentage(101)).toBe(false);
            expect(config.validateDonationPercentage(150)).toBe(false);
            expect(config.validateDonationPercentage(-50)).toBe(false);
        });
    });

    describe("calculateDonationAmount", () => {
        it("should calculate donation amount correctly", () => {
            const config = new CreditConfig({
                donation: {
                    enabled: true,
                    percentage: 20,
                    minThreshold: 100,
                    recipientType: "platform",
                },
            });

            expect(config.calculateDonationAmount(1000, 500)).toBe(100); // 20% of 500
            expect(config.calculateDonationAmount(5000, 2000)).toBe(400); // 20% of 2000
        });

        it("should return 0 when donation is disabled", () => {
            const config = new CreditConfig({
                donation: {
                    enabled: false,
                    percentage: 20,
                    minThreshold: 100,
                    recipientType: "platform",
                },
            });

            expect(config.calculateDonationAmount(1000, 500)).toBe(0);
        });

        it("should return 0 when below minThreshold", () => {
            const config = new CreditConfig({
                donation: {
                    enabled: true,
                    percentage: 20,
                    minThreshold: 1000,
                    recipientType: "platform",
                },
            });

            expect(config.calculateDonationAmount(999, 500)).toBe(0);
            expect(config.calculateDonationAmount(500, 500)).toBe(0);
        });

        it("should only donate from monthly allocation", () => {
            const config = new CreditConfig({
                donation: {
                    enabled: true,
                    percentage: 10,
                    minThreshold: 100,
                    recipientType: "platform",
                },
            });

            // Has 5000 credits but only 1000 from monthly allocation
            expect(config.calculateDonationAmount(5000, 1000)).toBe(100); // 10% of 1000
        });

        it("should handle case where monthly allocation exceeds available credits", () => {
            const config = new CreditConfig({
                donation: {
                    enabled: true,
                    percentage: 25,
                    minThreshold: 100,
                    recipientType: "platform",
                },
            });

            // Only 500 credits available but monthly allocation is 1000
            expect(config.calculateDonationAmount(500, 1000)).toBe(125); // 25% of 500
        });

        it("should floor the donation amount", () => {
            const config = new CreditConfig({
                donation: {
                    enabled: true,
                    percentage: 15,
                    minThreshold: 100,
                    recipientType: "platform",
                },
            });

            expect(config.calculateDonationAmount(1000, 333)).toBe(49); // 15% of 333 = 49.95, floored to 49
        });

        it("should throw error for negative available credits", () => {
            const config = new CreditConfig();

            expect(() => {
                config.calculateDonationAmount(-100, 1000);
            }).toThrow("Available credits cannot be negative");
        });

        it("should throw error for negative monthly allocation", () => {
            const config = new CreditConfig();

            expect(() => {
                config.calculateDonationAmount(1000, -100);
            }).toThrow("Monthly allocation cannot be negative");
        });

        it("should throw error for invalid percentage in config", () => {
            const config = new CreditConfig();
            // Manually set invalid percentage (bypassing updateDonationSettings validation)
            config.donation.percentage = 150;

            expect(() => {
                config.calculateDonationAmount(1000, 500);
            }).toThrow("Invalid donation percentage: 150. Must be between 0 and 100.");
        });

        it("should handle isValidPercentage method being used internally", () => {
            const config = new CreditConfig();
            // Test the private method logic through calculateDonationAmount
            config.donation.enabled = true;
            config.donation.minThreshold = 100;
            
            // Valid percentage
            config.donation.percentage = 50;
            expect(() => config.calculateDonationAmount(1000, 500)).not.toThrow();
            
            // Invalid percentage
            config.donation.percentage = -10;
            expect(() => config.calculateDonationAmount(1000, 500)).toThrow("Invalid donation percentage: -10. Must be between 0 and 100.");
        });
    });

    describe("shouldProcessRollover", () => {
        it("should return true when enabled and not processed for current month", () => {
            const config = new CreditConfig({
                rollover: {
                    enabled: true,
                    maxMonthsToKeep: 6,
                    lastProcessedMonth: "2024-01",
                },
            });

            expect(config.shouldProcessRollover("2024-02")).toBe(true);
            expect(config.shouldProcessRollover("2024-03")).toBe(true);
        });

        it("should return false when already processed for current month", () => {
            const config = new CreditConfig({
                rollover: {
                    enabled: true,
                    maxMonthsToKeep: 6,
                    lastProcessedMonth: "2024-02",
                },
            });

            expect(config.shouldProcessRollover("2024-02")).toBe(false);
        });

        it("should return false when disabled", () => {
            const config = new CreditConfig({
                rollover: {
                    enabled: false,
                    maxMonthsToKeep: 6,
                    lastProcessedMonth: "2024-01",
                },
            });

            expect(config.shouldProcessRollover("2024-02")).toBe(false);
        });

        it("should return true when never processed before", () => {
            const config = new CreditConfig({
                rollover: {
                    enabled: true,
                    maxMonthsToKeep: 6,
                },
            });

            expect(config.shouldProcessRollover("2024-01")).toBe(true);
        });
    });

    describe("shouldProcessDonation", () => {
        it("should return true when enabled and not processed for current month", () => {
            const config = new CreditConfig({
                donation: {
                    enabled: true,
                    percentage: 10,
                    minThreshold: 1000,
                    recipientType: "platform",
                    lastProcessedMonth: "2024-01",
                },
            });

            expect(config.shouldProcessDonation("2024-02")).toBe(true);
            expect(config.shouldProcessDonation("2024-03")).toBe(true);
        });

        it("should return false when already processed for current month", () => {
            const config = new CreditConfig({
                donation: {
                    enabled: true,
                    percentage: 10,
                    minThreshold: 1000,
                    recipientType: "platform",
                    lastProcessedMonth: "2024-02",
                },
            });

            expect(config.shouldProcessDonation("2024-02")).toBe(false);
        });

        it("should return false when disabled", () => {
            const config = new CreditConfig({
                donation: {
                    enabled: false,
                    percentage: 10,
                    minThreshold: 1000,
                    recipientType: "platform",
                    lastProcessedMonth: "2024-01",
                },
            });

            expect(config.shouldProcessDonation("2024-02")).toBe(false);
        });

        it("should return true when never processed before", () => {
            const config = new CreditConfig({
                donation: {
                    enabled: true,
                    percentage: 10,
                    minThreshold: 1000,
                    recipientType: "platform",
                },
            });

            expect(config.shouldProcessDonation("2024-01")).toBe(true);
        });
    });

    describe("Complex scenarios", () => {
        it("should handle complete monthly processing workflow", () => {
            const config = new CreditConfig({
                rollover: {
                    enabled: true,
                    maxMonthsToKeep: 6,
                },
                donation: {
                    enabled: true,
                    percentage: 15,
                    minThreshold: 500,
                    recipientType: "community",
                },
            });

            const currentMonth = "2024-03";

            // Check if processing is needed
            expect(config.shouldProcessRollover(currentMonth)).toBe(true);
            expect(config.shouldProcessDonation(currentMonth)).toBe(true);

            // Calculate donation
            const availableCredits = 2000;
            const monthlyAllocation = 1500;
            const donationAmount = config.calculateDonationAmount(availableCredits, monthlyAllocation);
            expect(donationAmount).toBe(225); // 15% of 1500

            // Update last processed month
            config.updateRolloverSettings({ lastProcessedMonth: currentMonth });
            config.updateDonationSettings({ lastProcessedMonth: currentMonth });

            // Verify processing won't happen again
            expect(config.shouldProcessRollover(currentMonth)).toBe(false);
            expect(config.shouldProcessDonation(currentMonth)).toBe(false);
        });

        it("should handle edge cases with various percentage values", () => {
            const config = new CreditConfig();

            // 0% donation
            config.updateDonationSettings({
                enabled: true,
                percentage: 0,
                minThreshold: 100,
            });
            expect(config.calculateDonationAmount(1000, 500)).toBe(0);

            // 100% donation
            config.updateDonationSettings({ percentage: 100 });
            expect(config.calculateDonationAmount(1000, 500)).toBe(500);

            // Fractional percentages
            config.updateDonationSettings({ percentage: 33 });
            expect(config.calculateDonationAmount(1000, 100)).toBe(33);
        });

        it("should maintain configuration integrity through multiple updates", () => {
            const config = new CreditConfig();

            // Series of updates
            config.updateRolloverSettings({ enabled: true });
            config.updateDonationSettings({ enabled: true, percentage: 20 });
            config.updateRolloverSettings({ maxMonthsToKeep: 8 });
            config.updateDonationSettings({ minThreshold: 750, recipientType: "community" });
            config.updateRolloverSettings({ lastProcessedMonth: "2024-01" });
            config.updateDonationSettings({ lastProcessedMonth: "2024-01" });

            // Verify final state
            const exported = config.toObject();
            expect(exported.rollover).toEqual({
                enabled: true,
                maxMonthsToKeep: 8,
                lastProcessedMonth: "2024-01",
            });
            expect(exported.donation).toEqual({
                enabled: true,
                percentage: 20,
                minThreshold: 750,
                recipientType: "community",
                lastProcessedMonth: "2024-01",
            });
        });
    });
});