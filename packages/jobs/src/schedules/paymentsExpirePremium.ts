import { DbProvider, batch, logger, sendSubscriptionEnded } from "@local/server";
import { Prisma } from "@prisma/client";

const commonSelect = {
    id: true,
    emails: { select: { emailAddress: true } },
    premium: {
        select: {
            id: true,
        },
    },
};

const commonWhere = {
    premium: {
        isActive: true,
        OR: [
            { expiresAt: null },
            { expiresAt: { lte: new Date() } },
        ],
    },
};

/**
 * Expires premium status for users and teams
 */
export async function paymentsExpirePremium() {
    // Expire for teams
    try {
        await batch<Prisma.teamFindManyArgs>({
            objectType: "Team",
            processBatch: async (batch) => {
                // Remove premium status for teams
                const premiumIds = batch.map(team => team.premium?.id).filter(id => id !== null) as string[];
                await DbProvider.get().premium.updateMany({
                    data: { isActive: false }, // Don't remove credits, as they may have paid for them
                    where: { id: { in: premiumIds } },
                });
                // Send notifications
                const emails = batch.map(team => team.emails).flat();
                for (const email of emails) {
                    sendSubscriptionEnded(email.emailAddress);
                }
            },
            select: commonSelect,
            where: commonWhere,
        });
    } catch (error) {
        logger.error("paymentsExpirePremium caught error for teams", { error, trace: "0465" });
    }
    // Expire for users
    try {
        await batch<Prisma.userFindManyArgs>({
            objectType: "User",
            processBatch: async (batch) => {
                // Remove premium status for users
                const premiumIds = batch.map(user => user.premium?.id).filter(id => id !== null) as string[];
                await DbProvider.get().premium.updateMany({
                    data: { isActive: false }, // Don't remove credits, as they may have paid for them
                    where: { id: { in: premiumIds } },
                });
                // Send notifications
                const emails = batch.map(user => user.emails).flat();
                for (const email of emails) {
                    sendSubscriptionEnded(email.emailAddress);
                }
            },
            select: commonSelect,
            where: commonWhere,
        });
    } catch (error) {
        logger.error("paymentsExpirePremium caught error for users", { error, trace: "0466" });
    }
};
