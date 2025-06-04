import { DbProvider, batch, logger } from "@local/server";
import { DAYS_90_MS, ModelType } from "@local/shared";
import { type Prisma } from "@prisma/client";

const REVOKED_SESSION_TIMEOUT = DAYS_90_MS;

// 1) Define the select shape once and derive payload type
const sessionSelect = { id: true } as const;
type SessionIdPayload = Prisma.sessionGetPayload<{ select: typeof sessionSelect }>;

/**
 * Removes sessions from the database that have been revoked for a long time
 */
export async function cleanupRevokedSessions(): Promise<void> {
    try {
        // Array to store deletion counts from each batch
        const counts: number[] = [];
        await batch<Prisma.sessionFindManyArgs, SessionIdPayload>({
            objectType: ModelType.Session,
            processBatch: async (sessions) => {
                const sessionIds = sessions.map(session => session.id);
                // Delete sessions
                const { count } = await DbProvider.get().session.deleteMany({
                    where: { id: { in: sessionIds } },
                });
                // Add the count from this batch to the array
                counts.push(count);
            },
            select: sessionSelect,
            where: { revokedAt: { lte: new Date(Date.now() - REVOKED_SESSION_TIMEOUT) } },
        });
        // Sum all collected counts after all batch processing is complete
        const totalDeleted = counts.reduce((sum, current) => sum + current, 0);
        logger.info(`Deleted ${totalDeleted} revoked sessions.`);
    } catch (error) {
        logger.error("deleteRevokedSessions caught error", { error, trace: "0223" });
    }
}
