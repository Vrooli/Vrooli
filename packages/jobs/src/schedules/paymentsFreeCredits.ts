import { batch } from "@local/server";
import { Prisma } from "@prisma/client";

const CREDITS_FREE = 100;
const CREDITS_PREMIUM = 10_000;

/**
 * Provides free credits to users based on their premium status.
 */
export const paymentsFreeCredits = async (): Promise<void> => {
    await batch<Prisma.userFindManyArgs>({
        objectType: "User",
        processBatch: async (batch, prisma) => {
            for (const user of batch) {
                const currentCredits = user.premium?.credits ?? 0;
                const newCredits = user.premium?.isActive
                    ? Math.max(CREDITS_PREMIUM, currentCredits)
                    : Math.max(CREDITS_FREE, currentCredits);

                if (user.premium) {
                    // Update existing premium record
                    await prisma.premium.update({
                        where: { id: user.premium.id },
                        data: { credits: newCredits },
                    });
                } else {
                    // Create new premium record
                    await prisma.premium.create({
                        data: {
                            user: { connect: { id: user.id } },
                            isActive: false,
                            credits: newCredits,
                        },
                    });
                }
            }
        },
        select: {
            id: true,
            premium: {
                select: {
                    id: true,
                    isActive: true,
                    credits: true,
                },
            },
        },
        where: { isBot: false },
        trace: "0470",
    });
};
