import { Prisma } from "@prisma/client";
import { batch } from "../../utils/batch";

/**
 * Expires premium status for users and organizations
 */
export const expirePremium = async () => {
    // Expire for organizations
    await batch<Prisma.organizationFindManyArgs>({
        objectType: "Organization",
        processBatch: async (batch, prisma) => {
            // Remove premium status for organizations
            const premiumIds = batch.map(organization => organization.premium?.id).filter(id => id !== null) as string[];
            await prisma.premium.updateMany({
                data: {
                    isActive: false,
                },
                where: {
                    id: { in: premiumIds },
                },
            });
            // Send notification
            //TODO
        },
        select: {
            id: true,
            premium: {
                select: {
                    id: true,
                },
            },
        },
        trace: "0465",
        where: {
            premium: {
                isActive: true,
                OR: [
                    { expiresAt: null },
                    { expiresAt: { lte: new Date() } },
                ],
            },
        },
    });
    // Expire for users
    await batch<Prisma.userFindManyArgs>({
        objectType: "User",
        processBatch: async (batch, prisma) => {
            // Remove premium status for users
            const premiumIds = batch.map(user => user.premium?.id).filter(id => id !== null) as string[];
            await prisma.premium.updateMany({
                data: {
                    isActive: false,
                },
                where: {
                    id: { in: premiumIds },
                },
            });
            // Send notification
            //TODO
        },
        select: {
            id: true,
            premium: {
                select: {
                    id: true,
                },
            },
        },
        trace: "0466",
        where: {
            premium: {
                isActive: true,
                OR: [
                    { expiresAt: null },
                    { expiresAt: { lte: new Date() } },
                ],
            },
        },
    });
};
