import { PrismaClient } from "@prisma/client";
import { logger } from "../../events";
import { PrismaType } from "../../types";

const checkOrganizations = async (
    prisma: PrismaType,
    organizations: { id: string, premium?: { id: string, expiresAt?: Date | null, isActive: boolean } | null }[],
): Promise<void> => {
    // Find all organizations that have premium status, and expiresAt is in the past
    const expiredOrganizations = organizations.filter(organization => {
        const premium = organization.premium;
        return premium && premium.isActive && (premium.expiresAt ?? 0) < Date.now();
    });
    // Remove premium status for organizations
    await Promise.all(expiredOrganizations.map(organization => {
        return prisma.organization.update({
            where: { id: organization.id },
            data: {
                premium: {
                    update: {
                        isActive: false,
                    }
                }
            }
        });
    }));
}

const checkUsers = async (
    prisma: PrismaType,
    users: { id: string, premium?: { id: string, expiresAt?: Date | null, isActive: boolean } | null }[],
): Promise<void> => {
    // Find all users that have premium status, and expiresAt is in the past
    const expiredUsers = users.filter(user => {
        const premium = user.premium;
        return premium && premium.isActive && (premium.expiresAt ?? 0) < Date.now();
    });
    // Remove premium status for users
    await Promise.all(expiredUsers.map(user => {
        return prisma.user.update({
            where: { id: user.id },
            data: {
                premium: {
                    update: {
                        isActive: false,
                    }
                }
            }
        });
    }));
}

/**
 * Expires premium status for users and organizations
 */
export const expirePremium = async () => {
    // Initialize the Prisma client
    const prisma = new PrismaClient();
    try {
        // We may be dealing with a lot of data, so we need to do this in batches
        const batchSize = 100;
        let skip = 0;
        let currentBatchSize = 0;
        // Start with organizations
        do {
            const batch = await prisma.organization.findMany({
                select: {
                    id: true,
                    premium: {
                        select: {
                            id: true,
                            expiresAt: true,
                            isActive: true,
                        }
                    }
                },
                skip,
                take: batchSize,
            });
            // Increment skip
            skip += batchSize;
            // Update current batch size
            currentBatchSize = batch.length;
            // Remove premium status for organizations
            await checkOrganizations(prisma, batch);
        } while (currentBatchSize === batchSize);
        // Reset skip, current batch size
        skip = 0;
        currentBatchSize = 0;
        // Now check users
        do {
            const batch = await prisma.user.findMany({
                select: {
                    id: true,
                    premium: {
                        select: {
                            id: true,
                            expiresAt: true,
                            isActive: true,
                        }
                    }
                },
                skip,
                take: batchSize,
            });
            // Increment skip
            skip += batchSize;
            // Update current batch size
            currentBatchSize = batch.length;
            // Remove premium status for users
            await checkUsers(prisma, batch);
        } while (currentBatchSize === batchSize);
    } catch (error) {
        logger.error('Caught error removing premium', { trace: '0442' });
    } finally {
        // Close the Prisma client
        await prisma.$disconnect();
    }
}