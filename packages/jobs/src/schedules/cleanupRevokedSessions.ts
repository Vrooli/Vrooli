import { DbProvider, batch, logger } from "@local/server";
import { DAYS_90_MS } from "@local/shared";
import { Prisma } from "@prisma/client";

const REVOKED_SESSION_TIMEOUT = DAYS_90_MS;

/**
 * Removes sessions from the database that have been revoked for a long time
 */
export async function cleanupRevokedSessions() {
    try {
        let totalDeleted = 0;
        await batch<Prisma.sessionFindManyArgs>({
            objectType: "Session",
            processBatch: async (batch) => {
                const sessionIds = batch.map(session => session.id);
                // Delete sessions
                const { count } = await DbProvider.get().session.deleteMany({
                    where: { id: { in: sessionIds } },
                });
                totalDeleted += count;
            },
            select: {
                id: true,
            },
            where: {
                revoked: true,
                updated_at: { lte: new Date(Date.now() - REVOKED_SESSION_TIMEOUT) },
            },
        });
        logger.info(`Deleted ${totalDeleted} revoked sessions.`);
    } catch (error) {
        logger.error("deleteRevokedSessions caught error", { error, trace: "0223" });
    }
};
