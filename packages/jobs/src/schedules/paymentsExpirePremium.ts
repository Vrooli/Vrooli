import { type Prisma } from "@prisma/client";
import { AUTH_EMAIL_TEMPLATES, DbProvider, QueueService, batch, logger } from "@vrooli/server";
import { nanoid } from "@vrooli/shared";

const commonSelect = {
    id: true,
    emails: { select: { emailAddress: true } },
    plan: {
        select: {
            id: true,
            expiresAt: true,
        },
    },
} as const;

// Declare payload types for teams and users using commonSelect
type TeamPayload = Prisma.teamGetPayload<{ select: typeof commonSelect }>;
type UserPayload = Prisma.userGetPayload<{ select: typeof commonSelect }>;

/**
 * Sends a subscription canceled email to the provided email addresses.
 * @param emailAddresses - Array of email addresses to send the notification to.
 */
async function sendSubscriptionCanceledEmail(emailAddresses: string[]): Promise<void> {
    if (emailAddresses.length <= 0) {
        return;
    }
    await QueueService.get().email.addTask({
        to: emailAddresses,
        ...AUTH_EMAIL_TEMPLATES.SubscriptionCanceled(nanoid()), // Using a random ID is fine here
    });
}

/**
 * Helper function to process a batch of plan expirations for users or teams.
 * @param batchItems - Array of user or team payloads.
 * @param now - The current date/time for expiration.
 */
async function _processPlanExpirationBatch(
    batchItems: TeamPayload[] | UserPayload[],
    now: Date,
): Promise<void> {
    const planIdsToSetExpiry: bigint[] = [];
    const planIdsToKeepExpiry: bigint[] = [];

    batchItems.forEach(item => {
        if (item.plan?.id != null) {
            if (item.plan.expiresAt == null) {
                planIdsToSetExpiry.push(item.plan.id);
            } else {
                planIdsToKeepExpiry.push(item.plan.id);
            }
        }
    });

    if (planIdsToSetExpiry.length > 0) {
        await DbProvider.get().plan.updateMany({
            data: {
                enabledAt: null,
                expiresAt: now,
            },
            where: { id: { in: planIdsToSetExpiry } },
        });
    }

    if (planIdsToKeepExpiry.length > 0) {
        await DbProvider.get().plan.updateMany({
            data: {
                enabledAt: null,
                // expiresAt is intentionally not set here
            },
            where: { id: { in: planIdsToKeepExpiry } },
        });
    }

    // Send notifications
    const emailAddresses = batchItems
        .flatMap(item => item.emails ?? [])
        .map(email => email.emailAddress);
    await sendSubscriptionCanceledEmail(emailAddresses);
}

/**
 * Expires plan status for users and teams
 * @returns Promise that resolves when the operation is complete
 */
export async function paymentsExpirePlan(): Promise<void> {
    const now = new Date();

    const currentCommonWhere = {
        plan: {
            enabledAt: { lte: now },
            OR: [
                { expiresAt: null },
                { expiresAt: { lte: now } },
            ],
        },
    };

    // Expire for teams
    try {
        await batch<Prisma.teamFindManyArgs, TeamPayload>({
            objectType: "Team",
            processBatch: async (batch: TeamPayload[]) => {
                await _processPlanExpirationBatch(batch, now);
            },
            select: commonSelect,
            where: currentCommonWhere,
        });
    } catch (error) {
        logger.error("paymentsExpirePlan caught error for teams", { error, trace: "0465" });
    }

    // Expire for users
    try {
        await batch<Prisma.userFindManyArgs, UserPayload>({
            objectType: "User",
            processBatch: async (batch: UserPayload[]) => {
                await _processPlanExpirationBatch(batch, now);
            },
            select: commonSelect,
            where: currentCommonWhere,
        });
    } catch (error) {
        logger.error("paymentsExpirePlan caught error for users", { error, trace: "0466" });
    }
}
