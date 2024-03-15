import { Notify, batch } from "@local/server";
import { API_CREDITS_PREMIUM } from "@local/shared";
import { Prisma } from "@prisma/client";

/**
 * Provides free credits to premium users.
 */
export const paymentsCreditsFreePremium = async (): Promise<void> => {
    await batch<Prisma.userFindManyArgs>({
        objectType: "User",
        processBatch: async (batch, prisma) => {
            for (const user of batch) {
                if (!user.premium || user.premium.isActive === false) continue;

                await prisma.premium.update({
                    where: { id: user.premium.id },
                    // Make sure not to get rid of any existing credits over the free amount
                    data: { credits: API_CREDITS_PREMIUM > user.premium.credits ? API_CREDITS_PREMIUM : user.premium.credits },
                });
                // Notify user of free credits
                if (API_CREDITS_PREMIUM > user.premium.credits) {
                    const language = user.languages.length > 0 ? user.languages[0].language : "en";
                    Notify(prisma, language).pushFreeCreditsReceived().toUser(user.id);
                }
            }
        },
        select: {
            id: true,
            languages: {
                select: {
                    language: true
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
        trace: "0470",
    });
};
