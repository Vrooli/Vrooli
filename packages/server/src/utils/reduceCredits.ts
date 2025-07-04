// AI_CHECK: TYPE_SAFETY=server-phase3-1 | LAST: 2025-07-03 - Added explicit return type annotation Promise<bigint | undefined>
import { CreditEntryType, CreditSourceSystem } from "@prisma/client";
import { EventTypes, generatePK } from "@vrooli/shared";
import { DbProvider } from "../db/provider.js";
import { SocketService } from "../sockets/io.js";

/**
 * Reduces the credits of a user by the specified amount, and 
 * sends a socket event to update the user's credits.
 * 
 * NOTE: This function assumes that the user exists
 * @param userId The ID of the user whose credits are to be reduced
 * @param decrement The amount to reduce the credits by
 * @param idempotencyKey A unique key to prevent duplicate credit operations
 * @param meta Optional metadata about the credit operation
 * @returns The new credit amount
 */
export async function reduceUserCredits(
    userId: string,
    decrement: number | bigint,
    idempotencyKey: string,
    meta?: Record<string, unknown>,
): Promise<bigint | undefined> {
    if (decrement <= 0) {
        return;
    }

    const userIdBigInt = BigInt(userId);
    const decrementBigInt = typeof decrement === "bigint" ? decrement : BigInt(decrement);

    // Use a transaction to ensure atomicity and return the final balance
    const finalBalance = await DbProvider.get().$transaction(async (tx) => {
        // First ensure user has a credit account
        const user = await tx.user.findUniqueOrThrow({
            where: { id: userIdBigInt },
            select: {
                id: true,
                creditAccount: { select: { id: true, currentBalance: true } },
            },
        });

        let creditAccountId: bigint;
        let currentBalance: bigint;

        if (!user.creditAccount) {
            // Create credit account if it doesn't exist
            const newAccount = await tx.credit_account.create({
                data: {
                    id: generatePK(),
                    userId: userIdBigInt,
                    currentBalance: 0,
                },
            });
            creditAccountId = newAccount.id;
            currentBalance = newAccount.currentBalance;
        } else {
            creditAccountId = user.creditAccount.id;
            currentBalance = user.creditAccount.currentBalance;
        }

        // Create ledger entry for the credit reduction (spend)
        await tx.credit_ledger_entry.create({
            data: {
                id: generatePK(),
                idempotencyKey,
                accountId: creditAccountId,
                amount: -decrementBigInt, // Negative for spend
                type: CreditEntryType.Spend,
                source: CreditSourceSystem.InternalAgent,
                meta: meta ? JSON.parse(JSON.stringify(meta)) : {},
            },
        });

        // Update the account balance
        const newBalance = currentBalance - decrementBigInt;
        await tx.credit_account.update({
            where: { id: creditAccountId },
            data: { currentBalance: newBalance },
        });

        return newBalance; // Return the final balance from the transaction
    }, { isolationLevel: "Serializable" });

    // Send a socket event to update the user's credits in open sessions
    SocketService.get().emitSocketEvent(EventTypes.USER.CREDITS_UPDATED, userId, {
        userId,
        credits: finalBalance.toString(),
    });

    return finalBalance;
}
