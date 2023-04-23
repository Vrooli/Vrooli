import { PrismaClient } from "@prisma/client";
import { logger } from "../../events";
const checkOrganizations = async (prisma, organizations) => {
    const expiredOrganizations = organizations.filter(organization => {
        const premium = organization.premium;
        return premium && premium.isActive && (premium.expiresAt ?? 0) < Date.now();
    });
    await Promise.all(expiredOrganizations.map(organization => {
        return prisma.organization.update({
            where: { id: organization.id },
            data: {
                premium: {
                    update: {
                        isActive: false,
                    },
                },
            },
        });
    }));
};
const checkUsers = async (prisma, users) => {
    const expiredUsers = users.filter(user => {
        const premium = user.premium;
        return premium && premium.isActive && (premium.expiresAt ?? 0) < Date.now();
    });
    await Promise.all(expiredUsers.map(user => {
        return prisma.user.update({
            where: { id: user.id },
            data: {
                premium: {
                    update: {
                        isActive: false,
                    },
                },
            },
        });
    }));
};
export const expirePremium = async () => {
    const prisma = new PrismaClient();
    try {
        const batchSize = 100;
        let skip = 0;
        let currentBatchSize = 0;
        do {
            const batch = await prisma.organization.findMany({
                select: {
                    id: true,
                    premium: {
                        select: {
                            id: true,
                            expiresAt: true,
                            isActive: true,
                        },
                    },
                },
                skip,
                take: batchSize,
            });
            skip += batchSize;
            currentBatchSize = batch.length;
            await checkOrganizations(prisma, batch);
        } while (currentBatchSize === batchSize);
        skip = 0;
        currentBatchSize = 0;
        do {
            const batch = await prisma.user.findMany({
                select: {
                    id: true,
                    premium: {
                        select: {
                            id: true,
                            expiresAt: true,
                            isActive: true,
                        },
                    },
                },
                skip,
                take: batchSize,
            });
            skip += batchSize;
            currentBatchSize = batch.length;
            await checkUsers(prisma, batch);
        } while (currentBatchSize === batchSize);
    }
    catch (error) {
        logger.error("Caught error removing premium", { trace: "0442" });
    }
    finally {
        await prisma.$disconnect();
    }
};
//# sourceMappingURL=expirePremium.js.map