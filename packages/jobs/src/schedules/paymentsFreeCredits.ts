import { batch } from "@local/server";
import { API_CREDITS_FREE, API_CREDITS_PREMIUM } from "@local/shared";
import { Prisma } from "@prisma/client";

/**
 * Provides free credits to users based on their premium status.
 * NOTE: Skips users who don't have a verified email address or wallet.
 */
export const paymentsFreeCredits = async (): Promise<void> => {
    await batch<Prisma.userFindManyArgs>({
        objectType: "User",
        processBatch: async (batch, prisma) => {
            for (const user of batch) {
                // If the user doesn't have a verified email or wallet, skip them
                const hasVerifiedEmail = user.emails.some(email => email.verified);
                const hasVerifiedWallet = user.wallets.some(wallet => wallet.verified);
                if (!(hasVerifiedEmail || hasVerifiedWallet)) continue;

                const currentCredits = user.premium?.credits ?? 0;
                const newCredits = user.premium?.isActive
                    ? Math.max(API_CREDITS_PREMIUM, currentCredits)
                    : Math.max(API_CREDITS_FREE, currentCredits);

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
            emails: {
                select: {
                    verified: true,
                },
            },
            premium: {
                select: {
                    id: true,
                    isActive: true,
                    credits: true,
                },
            },
            wallets: {
                select: {
                    verified: true,
                },
            },
        },
        where: {
            isBot: false,
        },
        trace: "0470",
    });
};
