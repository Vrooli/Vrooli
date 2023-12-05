import { batch, sendSubscriptionEnded } from "@local/server";
import { Prisma } from "@prisma/client";

const CREDITS_FREE = 100;

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
    await batch<Prisma.organizationFindManyArgs>({
        objectType: "Organization",
        processBatch: async (batch, prisma) => {
            // Remove premium status for organizations
            const premiumIds = batch.map(organization => organization.premium?.id).filter(id => id !== null) as string[];
            await prisma.premium.updateMany({
                data: { isActive: false, credits: CREDITS_FREE },
                where: { id: { in: premiumIds } },
            });
            // Send notifications
            const emails = batch.map(organization => organization.emails).flat();
            for (const email of emails) {
                sendSubscriptionEnded(email.emailAddress);
            }
        },
        select: commonSelect,
        trace: "0465",
        where: commonWhere,
    });
    // Expire for users
    await batch<Prisma.userFindManyArgs>({
        objectType: "User",
        processBatch: async (batch, prisma) => {
            // Remove premium status for users
            const premiumIds = batch.map(user => user.premium?.id).filter(id => id !== null) as string[];
            await prisma.premium.updateMany({
                data: { isActive: false, credits: CREDITS_FREE },
                where: { id: { in: premiumIds } },
            });
            // Send notifications
            const emails = batch.map(user => user.emails).flat();
            for (const email of emails) {
                sendSubscriptionEnded(email.emailAddress);
            }
        },
        select: commonSelect,
        trace: "0466",
        where: commonWhere,
    });
};
