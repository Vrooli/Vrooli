import { CreditEntryType, CreditSourceSystem, type Prisma } from "@prisma/client";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { CustomError } from "../../events/error.js";

/**
 * Calculate how many free credits (from monthly bonus) a user currently has available.
 * Uses FIFO (First In, First Out) logic to track which credits have been spent.
 * 
 * @param creditAccountId The credit account ID to calculate balance for
 * @returns The number of free credits remaining
 */
export async function calculateFreeCreditsBalance(creditAccountId: bigint): Promise<bigint> {
    try {
        // Get all credit ledger entries for this account, ordered by creation time (FIFO)
        const ledgerEntries = await DbProvider.get().credit_ledger_entry.findMany({
            where: { accountId: creditAccountId },
            orderBy: { createdAt: "asc" },
            select: {
                id: true,
                amount: true,
                type: true,
                source: true,
                createdAt: true,
                meta: true,
            },
        });

        // Track credits by source in FIFO order
        let freeCreditsAdded = BigInt(0);
        let purchasedCreditsAdded = BigInt(0);
        let totalSpent = BigInt(0);

        // First pass: calculate total credits added and spent
        for (const entry of ledgerEntries) {
            const amount = BigInt(entry.amount);

            // Credits added
            if (amount > 0) {
                // Free monthly credits
                if (entry.type === CreditEntryType.Bonus && entry.source === CreditSourceSystem.Scheduler) {
                    freeCreditsAdded += amount;
                }
                // Rollovers are always free credits (they came from previous month's free credits)
                else if (entry.type === CreditEntryType.RollOver) {
                    freeCreditsAdded += amount;
                }
                // Purchased credits (from Stripe)
                else if (entry.type === CreditEntryType.Purchase && entry.source === CreditSourceSystem.Stripe) {
                    purchasedCreditsAdded += amount;
                }
                // Transfers - check meta for original source tracking
                else if (entry.type === CreditEntryType.TransferIn) {
                    // Check if meta contains source information
                    if (entry.meta && typeof entry.meta === "object" && "originalSource" in entry.meta) {
                        const meta = entry.meta as { originalSource?: string };
                        if (meta.originalSource === "free") {
                            freeCreditsAdded += amount;
                        } else {
                            purchasedCreditsAdded += amount;
                        }
                    } else {
                        // For legacy transfers without source tracking, treat conservatively as purchased
                        purchasedCreditsAdded += amount;
                    }
                }
            }
            // Credits spent/removed
            else if (amount < 0) {
                totalSpent += -amount; // Convert to positive for easier calculation
            }
        }

        // Apply FIFO: Free credits are spent first
        let freeCreditsSpent = BigInt(0);
        let purchasedCreditsSpent = BigInt(0);

        if (totalSpent > 0) {
            // Spend free credits first
            if (totalSpent <= freeCreditsAdded) {
                freeCreditsSpent = totalSpent;
            } else {
                // All free credits spent, remainder comes from purchased
                freeCreditsSpent = freeCreditsAdded;
                purchasedCreditsSpent = totalSpent - freeCreditsAdded;
            }
        }

        // Calculate remaining free credits
        const freeCreditsRemaining = freeCreditsAdded - freeCreditsSpent;
        const purchasedCreditsRemaining = purchasedCreditsAdded - purchasedCreditsSpent;

        // Log calculation details for debugging
        logger.info("Calculated free credits balance", {
            creditAccountId: creditAccountId.toString(),
            freeCreditsAdded: freeCreditsAdded.toString(),
            purchasedCreditsAdded: purchasedCreditsAdded.toString(),
            totalSpent: totalSpent.toString(),
            freeCreditsSpent: freeCreditsSpent.toString(),
            purchasedCreditsSpent: purchasedCreditsSpent.toString(),
            freeCreditsRemaining: freeCreditsRemaining.toString(),
            purchasedCreditsRemaining: purchasedCreditsRemaining.toString(),
            trace: "calculateFreeCreditsBalance",
        });

        return freeCreditsRemaining;
    } catch (error) {
        logger.error("Failed to calculate free credits balance", {
            error,
            creditAccountId: creditAccountId.toString(),
            trace: "calculateFreeCreditsBalance_error",
        });
        // Return 0 on error to be safe (don't donate if we can't calculate)
        return BigInt(0);
    }
}

/**
 * Get detailed credit balance breakdown by source
 */
export async function getCreditBalancesBySource(creditAccountId: bigint): Promise<{
    free: bigint;
    purchased: bigint;
    total: bigint;
}> {
    const freeBalance = await calculateFreeCreditsBalance(creditAccountId);
    
    // Get total balance from credit_account
    const account = await DbProvider.get().credit_account.findUnique({
        where: { id: creditAccountId },
        select: { currentBalance: true },
    });

    if (!account) {
        logger.error(`Credit account not found: ${creditAccountId}`, {
            creditAccountId: creditAccountId.toString(),
            trace: "getCreditBalancesBySource_accountNotFound",
        });
        throw new CustomError("0100", "NotFound", { message: `Credit account not found: ${creditAccountId}` });
    }

    const totalBalance = account.currentBalance ?? BigInt(0);
    const purchasedBalance = totalBalance - freeBalance;

    return {
        free: freeBalance,
        purchased: purchasedBalance > BigInt(0) ? purchasedBalance : BigInt(0),
        total: totalBalance,
    };
}

/**
 * Record which source of credits was consumed when spending
 * This helps maintain accurate FIFO tracking
 */
export function getConsumedCreditSource(freeBalance: bigint, spendAmount: bigint): "free" | "purchased" | "mixed" {
    if (spendAmount <= freeBalance) {
        return "free";
    } else if (freeBalance === BigInt(0)) {
        return "purchased";
    } else {
        return "mixed";
    }
}
