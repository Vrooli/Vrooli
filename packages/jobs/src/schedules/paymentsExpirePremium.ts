import { AUTH_EMAIL_TEMPLATES, DbProvider, QueueService, batch, logger } from "@local/server";
import { nanoid } from "@local/shared";
import { Prisma } from "@prisma/client";

type TeamWithPlanAndEmails = Pick<Prisma.teamGetPayload<{
    select: {
        id: true;
    }
}>, "id"> & {
    emails: Pick<Prisma.emailGetPayload<{
        select: { emailAddress: true }
    }>, "emailAddress">[];
    plan: Pick<Prisma.planGetPayload<{
        select: { id: true }
    }>, "id"> | null;
};

type UserWithPlanAndEmails = Pick<Prisma.userGetPayload<{
    select: {
        id: true;
    }
}>, "id"> & {
    emails: Pick<Prisma.emailGetPayload<{
        select: { emailAddress: true }
    }>, "emailAddress">[];
    plan: Pick<Prisma.planGetPayload<{
        select: { id: true }
    }>, "id"> | null;
};

const commonSelect = {
    id: true,
    emails: { select: { emailAddress: true } },
    plan: {
        select: {
            id: true,
        },
    },
};

const commonWhere = {
    plan: {
        enabledAt: { lte: new Date() },
        OR: [
            { expiresAt: null },
            { expiresAt: { lte: new Date() } },
        ],
    },
};

/**
 * Expires plan status for users and teams
 * @returns Promise that resolves when the operation is complete
 */
export async function paymentsExpirePlan(): Promise<void> {
    // Expire for teams
    try {
        await batch<Prisma.teamFindManyArgs>({
            objectType: "Team",
            processBatch: async (batch: TeamWithPlanAndEmails[]) => {
                // Remove plan status for teams
                const planIds = batch
                    .map(team => team.plan?.id)
                    .filter((id): id is bigint => id !== undefined && id !== null);

                if (planIds.length > 0) {
                    await DbProvider.get().plan.updateMany({
                        // Don't remove credits, as they may have paid for them
                        data: {
                            enabledAt: null,
                            expiresAt: new Date(),
                        },
                        where: { id: { in: planIds } },
                    });
                }

                // Send notifications
                const emailAddresses = batch
                    .flatMap(team => team.emails)
                    .map(email => email.emailAddress);

                if (emailAddresses.length > 0) {
                    await QueueService.get().email.addTask({
                        to: emailAddresses,
                        ...AUTH_EMAIL_TEMPLATES.SubscriptionCanceled(nanoid()), // Using a random ID is fine here
                    });
                }
            },
            select: commonSelect,
            where: commonWhere,
        });
    } catch (error) {
        logger.error("paymentsExpirePlan caught error for teams", { error, trace: "0465" });
    }

    // Expire for users
    try {
        await batch<Prisma.userFindManyArgs>({
            objectType: "User",
            processBatch: async (batch: UserWithPlanAndEmails[]) => {
                // Remove plan status for users
                const planIds = batch
                    .map(user => user.plan?.id)
                    .filter((id): id is bigint => id !== undefined && id !== null);

                if (planIds.length > 0) {
                    await DbProvider.get().plan.updateMany({
                        // Don't remove credits, as they may have paid for them
                        data: {
                            enabledAt: null,
                            expiresAt: new Date(),
                        },
                        where: { id: { in: planIds } },
                    });
                }

                // Send notifications
                const emailAddresses = batch
                    .flatMap(user => user.emails)
                    .map(email => email.emailAddress);

                if (emailAddresses.length > 0) {
                    await QueueService.get().email.addTask({
                        to: emailAddresses,
                        ...AUTH_EMAIL_TEMPLATES.SubscriptionCanceled(nanoid()), // Using a random ID is fine here
                    });
                }
            },
            select: commonSelect,
            where: commonWhere,
        });
    } catch (error) {
        logger.error("paymentsExpirePlan caught error for users", { error, trace: "0466" });
    }
}
