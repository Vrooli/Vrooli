import { DbProvider } from "../db/provider.js";
import { emitSocketEvent } from "../sockets/events.js";

/**
 * Reduces the credits of a user by the specified amount, and 
 * sends a socket event to update the user's credits.
 * 
 * NOTE: This function assumes that the user exists
 * @param userId The ID of the user whose credits are to be reduced
 * @param decrement The amount to reduce the credits by
 * @returns The new credit amount
 */
export async function reduceUserCredits(userId: string, decrement: number | bigint) {
    if (decrement <= 0) {
        return;
    }
    // Update the user's credits
    const updatedUser = await DbProvider.get().user.update({
        where: { id: userId },
        data: {
            premium: {
                upsert: {
                    // Shouldn't call this when the user doesn't have a premium relation 
                    // (i.e. never paid for or received free credits), but just in case
                    create: { credits: 0, hasReceivedFreeTrial: false, isActive: false },
                    // The actual update for most users
                    update: { credits: { decrement } },
                },
            },
        },
        select: { premium: { select: { credits: true } } },
    });
    // Send a socket event to update the user's credits in open sessions
    if (updatedUser.premium) {
        emitSocketEvent("apiCredits", userId, {
            credits: updatedUser.premium.credits + "", // Sent as a string because BigInt
        });
    }
}
