export type CreditValue = number | string | bigint;

/**
 * Calculates the maximum credits allowed for a task. 
 * Ensures that we don't exceed the user's remaining credits or the task's maximum credits 
 * (factoring in credits already spent on this task).
 * 
 * @param userRemainingCredits The user's remaining credits.
 * @param taskMaxCredits The maximum credits defined for the task.
 * @param creditsSpent Credits already spent on this task.
 * @returns The maximum credits allowed for the task, as a BigInt.
 * 
 * @example
 * calculateMaxCredits(500000, 800000, 100000) // Returns 700000n
 * calculateMaxCredits("750000", 800000, "50000") // Returns 750000n
 * calculateMaxCredits(1500000n, "1000000", 200000) // Returns 800000n
 * calculateMaxCredits("2000000", undefined, 500000n) // Returns 1000000n (using DEFAULT_MAX_CREDITS)
 */
export function calculateMaxCredits(
    userRemainingCredits: CreditValue,
    taskMaxCredits: CreditValue,
    creditsSpent: CreditValue | undefined,
): bigint {
    // Convert all inputs to BigInt
    const userRemainingCreditsBigInt = BigInt(userRemainingCredits || 0);
    const creditsSpentBigInt = BigInt(creditsSpent || 0);
    const taskMaxCreditsBigInt = BigInt(taskMaxCredits || 0);

    // eslint-disable-next-line no-magic-numbers
    if (userRemainingCreditsBigInt <= 0n || taskMaxCreditsBigInt <= 0n || creditsSpentBigInt < 0n) {
        // eslint-disable-next-line no-magic-numbers
        return 0n;
    }

    // Calculate the effective task max credits (task max minus credits already spent on this task
    const effectiveTaskMax = taskMaxCreditsBigInt > creditsSpentBigInt
        ? BigInt(taskMaxCreditsBigInt) - BigInt(creditsSpentBigInt)
        : BigInt(0);

    // Return the smaller of the userRemaining credits and the effective task max
    return userRemainingCreditsBigInt < effectiveTaskMax
        ? userRemainingCreditsBigInt
        : effectiveTaskMax;
}
