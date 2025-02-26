import { DbProvider, Notify, batch, emitSocketEvent, logger } from "@local/server";
import { API_CREDITS_PREMIUM } from "@local/shared";
import { Prisma } from "@prisma/client";

const MAX_MONTHS_ACCRUED = 6;
/**
 * The max number of free credits to give to premium users.
 */
const MAX_FREE_CREDITS = BigInt(MAX_MONTHS_ACCRUED) * API_CREDITS_PREMIUM;

/**
 * Provides free credits to premium users.
 */
export async function paymentsCreditsFreePremium(): Promise<void> {
    try {
        await batch<Prisma.userFindManyArgs>({
            objectType: "User",
            processBatch: async (batch) => {
                for (const user of batch) {
                    if (!user.premium || user.premium.isActive === false) continue;

                    await DbProvider.get().premium.update({
                        where: { id: user.premium.id },
                        // Only give free credits if the user has less than 6x the monthly free amount
                        data: {
                            // If user has less than the max free amount
                            credits: user.premium.credits < MAX_FREE_CREDITS
                                // If user has less than the max free amount - the monthly free amount
                                ? user.premium.credits < (MAX_FREE_CREDITS - API_CREDITS_PREMIUM)
                                    // Give the monthly free amount
                                    ? user.premium.credits + API_CREDITS_PREMIUM
                                    // Give them the max free amount, since the refill would put them over the max free amount
                                    : MAX_FREE_CREDITS
                                // Otherwise (the user has more than the max free amount) keep their current amount
                                : user.premium.credits,
                        },
                    });
                    // Notify user of free credits
                    if (API_CREDITS_PREMIUM > user.premium.credits) {
                        const language = user.languages.length > 0 ? user.languages[0].language : "en";
                        Notify(language).pushFreeCreditsReceived().toUser(user.id);
                    }
                    // Update credits shown in UI for user, if they happen to be online
                    emitSocketEvent("apiCredits", user.id, { credits: user.premium.credits + "" });
                }
            },
            select: {
                id: true,
                languages: {
                    select: {
                        language: true,
                    },
                },
                premium: {
                    select: {
                        id: true,
                        isActive: true,
                        credits: true,
                    },
                },
            },
            where: {
                isBot: false,
            },
        });
    } catch (error) {
        logger.error("Error giving free credits to pro users", { error, trace: "0470" });
    }
}
