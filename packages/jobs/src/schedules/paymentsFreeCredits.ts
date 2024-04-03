import { Notify, batch, logger, prismaInstance } from "@local/server";
import { API_CREDITS_PREMIUM } from "@local/shared";
import { Prisma } from "@prisma/client";

/**
 * The max number of free credits to give to premium users.
 * This is 6x the monthly free amount.
 */
const MAX_FREE_CREDITS = BigInt(6) * API_CREDITS_PREMIUM;

/**
 * Provides free credits to premium users.
 */
export const paymentsCreditsFreePremium = async (): Promise<void> => {
    try {
        await batch<Prisma.userFindManyArgs>({
            objectType: "User",
            processBatch: async (batch) => {
                for (const user of batch) {
                    if (!user.premium || user.premium.isActive === false) continue;

                    await prismaInstance.premium.update({
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
};
