// AI_CHECK: TEST_COVERAGE=1 | LAST: 2025-06-24
import { CreditEntryType, CreditSourceSystem, type Prisma } from "@prisma/client";
import { batch, BusService, logger, type BillingEvent, CacheService } from "@vrooli/server";
import { API_CREDITS_PREMIUM, CreditConfig, type CreditConfigObject, generatePK } from "@vrooli/shared";
import { calculateFreeCreditsBalance } from "@vrooli/server";

// Time constants
const SECONDS_PER_MINUTE = 60;
const MINUTES_PER_HOUR = 60;
const HOURS_PER_DAY = 24;
const DAYS_FOR_ROLLOVER_KEY_EXPIRY = 45;
const ROLLOVER_KEY_EXPIRY_SECONDS = SECONDS_PER_MINUTE * MINUTES_PER_HOUR * HOURS_PER_DAY * DAYS_FOR_ROLLOVER_KEY_EXPIRY;

/**
 * Processes credit rollovers and donations for premium users
 * Runs monthly after free credits are awarded
 */
export async function creditRollover(): Promise<void> {
    // Use UTC to ensure consistent monthly processing across timezones
    const now = new Date();
    const currentMonth = `${now.getUTCFullYear()}-${String(now.getUTCMonth() + 1).padStart(2, "0")}`; // "YYYY-MM"
    
    // Check if this month has already been processed (idempotency protection)
    const redis = await CacheService.get().raw();
    
    let processedUsers = 0;
    let errorCount = 0;
    
    // Use Redis to track monthly processing instead of looking for specific ledger entries
    const rolloverKey = `creditRollover:processed:${currentMonth}`;
    const alreadyProcessed = await redis?.get(rolloverKey);
    
    if (alreadyProcessed) {
        logger.info(`Credit rollover for month ${currentMonth} already processed`, {
            trace: "creditRollover_alreadyProcessed",
            month: currentMonth,
        });
        return;
    }
    

    try {
        await batch<Prisma.userFindManyArgs, CreditSettingsPayload>({
            objectType: "User",
            batchSize: 100, // Process users in chunks to avoid memory issues
            processBatch: async (batch) => {
                for (const user of batch) {
                    try {
                        await processUserCreditRollover(user, currentMonth);
                        processedUsers++;
                    } catch (userError) {
                        errorCount++;
                        logger.error(`Failed to process credit rollover for user ${user.id}`, { 
                            error: userError, 
                            userId: user.id, 
                            trace: "creditRollover_userError", 
                        });
                    }
                }
            },
            select: creditSettingsSelect,
            where: {
                isBot: false,
                plan: {
                    enabledAt: { not: null },
                },
                creditAccount: {
                    isNot: null,
                },
                // Remove credit settings filter - we'll check it in processBatch
            },
        });
        
        // Mark this month as processed (idempotency protection)
        if (redis) {
            await redis.set(rolloverKey, "1", "EX", ROLLOVER_KEY_EXPIRY_SECONDS);
        }
        
        // Update Redis with job completion status
        if (redis) {
            try {
                const jobStatus = {
                    status: errorCount === 0 ? "success" : (processedUsers > 0 ? "partial" : "failed"),
                    timestamp: new Date().toISOString(),
                    processedUsers,
                    errorCount,
                    month: currentMonth,
                };
                await redis.set("job:creditRollover:lastRun", JSON.stringify(jobStatus));
            } catch (redisError) {
                logger.error("Failed to update Redis job status", { 
                    error: redisError, 
                    trace: "creditRollover_redisError", 
                });
            }
        }
        
        logger.info(`Successfully completed credit rollover for month ${currentMonth}`, {
            trace: "creditRollover_monthlyCompleted",
            month: currentMonth,
            processedUsers,
            errorCount,
        });
        
    } catch (error) {
        errorCount++;
        
        // Update Redis with failure status
        const redis = await CacheService.get().raw();
        if (redis) {
            try {
                const jobStatus = {
                    status: "failed",
                    timestamp: new Date().toISOString(),
                    processedUsers,
                    errorCount,
                    month: currentMonth,
                    error: error instanceof Error ? error.message : "Unknown error",
                };
                await redis.set("job:creditRollover:lastRun", JSON.stringify(jobStatus));
            } catch (redisError) {
                logger.error("Failed to update Redis job status on error", { 
                    error: redisError, 
                    trace: "creditRollover_redisErrorOnFailure", 
                });
            }
        }
        
        logger.error("Error in creditRollover job", { 
            error, 
            trace: "creditRollover_jobError",
            processedUsers,
            errorCount,
        });
    }
}

// Define the select type for user queries
const creditSettingsSelect = {
    id: true,
    creditSettings: true,
    languages: true,
    plan: {
        select: {
            id: true,
            enabledAt: true,
            expiresAt: true,
        },
    },
    creditAccount: {
        select: {
            id: true,
            currentBalance: true,
        },
    },
} as const;

type CreditSettingsPayload = Prisma.userGetPayload<{ select: typeof creditSettingsSelect }>;

async function processUserCreditRollover(
    user: CreditSettingsPayload,
    currentMonth: string,
): Promise<void> {
    if (!user.plan || !user.creditAccount || !user.creditSettings) {
        return;
    }

    // Check if premium is active
    const isPremiumActive = user.plan.enabledAt &&
        (!user.plan.expiresAt || user.plan.expiresAt > new Date());

    if (!isPremiumActive) {
        return;
    }

    // Parse credit settings
    let creditConfig: CreditConfig;
    try {
        if (typeof user.creditSettings !== "object" || user.creditSettings === null) {
            logger.warn(`Invalid credit settings type for user ${user.id}`, { 
                settings: user.creditSettings, 
                trace: "creditRollover_invalidSettingsType", 
            });
            return;
        }
        creditConfig = new CreditConfig(user.creditSettings as Partial<CreditConfigObject>);
    } catch (error) {
        logger.warn(`Invalid credit settings for user ${user.id}`, { 
            settings: user.creditSettings, 
            error,
            trace: "creditRollover_invalidSettings", 
        });
        return;
    }

    const currentBalance = user.creditAccount.currentBalance ?? BigInt(0);
    
    // Process donation if enabled and not already processed this month
    if (creditConfig.shouldProcessDonation(currentMonth)) {
        await processCreditDonation(user, creditConfig, currentBalance, currentMonth);
    }

    // Process rollover expiration if enabled and not already processed this month
    if (creditConfig.shouldProcessRollover(currentMonth)) {
        await processCreditRolloverExpiration(user, creditConfig, currentBalance, currentMonth);
    }
}

async function processCreditDonation(
    user: CreditSettingsPayload,
    creditConfig: CreditConfig,
    currentBalance: bigint,
    currentMonth: string,
): Promise<void> {
    if (!creditConfig.donation.enabled) {
        return;
    }

    // Calculate actual free credits balance
    const freeCreditsBalance = await calculateFreeCreditsBalance(user.creditAccount!.id);
    
    // Skip if no free credits to donate
    if (freeCreditsBalance <= BigInt(0)) {
        logger.info(`User ${user.id} has no free credits to donate`, {
            userId: user.id.toString(),
            currentBalance: currentBalance.toString(),
            freeCreditsBalance: freeCreditsBalance.toString(),
            trace: "creditRollover_noFreeCredits",
        });
        return;
    }
    
    // Calculate donation amount using BigInt math to avoid precision loss
    const donationPercentage = BigInt(creditConfig.donation.percentage);
    const donationAmountBigInt = (freeCreditsBalance * donationPercentage) / BigInt(100);
    
    // Log warning if amount is very large but continue processing
    if (donationAmountBigInt > BigInt(Number.MAX_SAFE_INTEGER)) {
        logger.warn(`Large donation amount ${donationAmountBigInt.toString()} for user ${user.id}`, {
            userId: user.id,
            freeCreditsBalance: freeCreditsBalance.toString(),
            donationAmount: donationAmountBigInt.toString(),
            trace: "creditRollover_largeDonation",
        });
    }
    
    if (donationAmountBigInt > BigInt(0)) {
        const billingEventId = generatePK().toString(); // Use proper UUID
        const billingEvent: BillingEvent = {
            type: "billing:event",
            id: billingEventId,
            accountId: user.creditAccount!.id.toString(),
            delta: (-donationAmountBigInt).toString(), // Negative for outgoing
            entryType: CreditEntryType.DonationGiven,
            source: CreditSourceSystem.Scheduler,
            meta: {
                reason: "Monthly credit donation to platform",
                jobName: "creditRollover",
                originalUserId: user.id.toString(),
                donationPercentage: creditConfig.donation.percentage,
                currentMonth,
            },
        };

        try {
            await BusService.get().getBus().publish(billingEvent);
            logger.info(`Successfully published credit donation event ${billingEventId} for user ${user.id}`, {
                userId: user.id.toString(),
                donatedCredits: donationAmountBigInt.toString(),
                freeCreditsBalance: freeCreditsBalance.toString(),
                totalBalance: currentBalance.toString(),
                donationPercentage: creditConfig.donation.percentage,
                trace: "creditRollover_donationSuccess",
            });
            
            // Update user's credit settings to mark donation as processed
            await updateCreditSettingsProcessedMonth(user.id, "donation", currentMonth);
            
        } catch (publishError) {
            logger.error(`Failed to publish donation BillingEvent for user ${user.id}`, { 
                error: publishError, 
                eventId: billingEventId, 
                userId: user.id, 
                donationAmount: donationAmountBigInt.toString(),
                trace: "creditRollover_donationPublishError", 
            });
        }
    }
}

async function processCreditRolloverExpiration(
    user: CreditSettingsPayload,
    creditConfig: CreditConfig,
    currentBalance: bigint,
    currentMonth: string,
): Promise<void> {
    if (!creditConfig.rollover.enabled) {
        return;
    }

    const maxAllowedCredits = BigInt(creditConfig.rollover.maxMonthsToKeep) * API_CREDITS_PREMIUM;
    
    if (currentBalance > maxAllowedCredits) {
        const excessCredits = currentBalance - maxAllowedCredits;
        
        const billingEventId = generatePK().toString(); // Use proper UUID
        const billingEvent: BillingEvent = {
            type: "billing:event",
            id: billingEventId,
            accountId: user.creditAccount!.id.toString(),
            delta: (-excessCredits).toString(), // Negative for outgoing
            entryType: CreditEntryType.Expire,
            source: CreditSourceSystem.Scheduler,
            meta: {
                reason: "Credit rollover expiration",
                jobName: "creditRollover",
                originalUserId: user.id.toString(),
                maxMonthsToKeep: creditConfig.rollover.maxMonthsToKeep,
                currentMonth,
            },
        };

        try {
            await BusService.get().getBus().publish(billingEvent);
            logger.info(`Successfully published credit expiration event ${billingEventId} for user ${user.id} - expired ${excessCredits} credits`);
            
            // Update user's credit settings to mark rollover as processed
            await updateCreditSettingsProcessedMonth(user.id, "rollover", currentMonth);
            
        } catch (publishError) {
            logger.error(`Failed to publish expiration BillingEvent for user ${user.id}`, { 
                error: publishError, 
                eventId: billingEventId, 
                userId: user.id, 
                excessCredits,
                trace: "creditRollover_expirationPublishError", 
            });
        }
    } else {
        // No expiration needed, but still mark as processed
        await updateCreditSettingsProcessedMonth(user.id, "rollover", currentMonth);
    }
}

async function updateCreditSettingsProcessedMonth(
    userId: bigint,
    settingsType: "donation" | "rollover",
    currentMonth: string,
): Promise<void> {
    try {
        const { DbProvider } = await import("@vrooli/server");
        
        // Use atomic update with retry mechanism for race condition safety
        const maxRetries = 3;
        let retryCount = 0;
        
        while (retryCount < maxRetries) {
            try {
                // Use a transaction to ensure atomic read-modify-write
                await DbProvider.get().$transaction(async (tx) => {
                    const user = await tx.user.findUnique({
                        where: { id: userId },
                        select: { creditSettings: true },
                    });
                    
                    if (!user?.creditSettings) {
                        throw new Error("User or credit settings not found");
                    }
                    
                    const creditConfig = new CreditConfig(user.creditSettings as Partial<CreditConfigObject>);
                    
                    if (settingsType === "donation") {
                        creditConfig.updateDonationSettings({ lastProcessedMonth: currentMonth });
                    } else if (settingsType === "rollover") {
                        creditConfig.updateRolloverSettings({ lastProcessedMonth: currentMonth });
                    }
                    
                    return await tx.user.update({
                        where: { id: userId },
                        data: {
                            creditSettings: creditConfig.toObject() as unknown as Prisma.InputJsonValue,
                        },
                    });
                });
                
                // Success - break out of retry loop
                break;
                
            } catch (transactionError) {
                retryCount++;
                if (retryCount >= maxRetries) {
                    throw transactionError;
                }
                
                // Wait before retrying (exponential backoff)
                await new Promise(resolve => setTimeout(resolve, 100 * Math.pow(2, retryCount)));
            }
        }
    } catch (error) {
        logger.error(`Failed to update credit settings processed month for user ${userId}`, {
            error,
            userId,
            settingsType,
            currentMonth,
            trace: "creditRollover_updateSettingsError",
        });
    }
}
