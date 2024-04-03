import { batch, logger, prismaInstance, sendSubscriptionEnded } from "@local/server";
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
 * Expires premium status for users and organizations
 */
export const paymentsExpirePremium = async () => {
    // Expire for organizations
    try {
        await batch<Prisma.organizationFindManyArgs>({
            objectType: "Organization",
            processBatch: async (batch) => {
                // Remove premium status for organizations
                const premiumIds = batch.map(organization => organization.premium?.id).filter(id => id !== null) as string[];
                await prismaInstance.premium.updateMany({
                    data: { isActive: false }, // Don't remove credits, as they may have paid for them
                    where: { id: { in: premiumIds } },
                });
                // Send notifications
                const emails = batch.map(organization => organization.emails).flat();
                for (const email of emails) {
                    sendSubscriptionEnded(email.emailAddress);
                }
            },
            select: commonSelect,
            where: commonWhere,
        });
    } catch (error) {
        logger.error("paymentsExpirePremium caught error for organizations", { error, trace: "0465" });
    }
    // Expire for users
    try {
        await batch<Prisma.userFindManyArgs>({
            objectType: "User",
            processBatch: async (batch) => {
                // Remove premium status for users
                const premiumIds = batch.map(user => user.premium?.id).filter(id => id !== null) as string[];
                await prismaInstance.premium.updateMany({
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
