// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-04
import { DbProvider, logger } from "@vrooli/server";
import { DAYS_90_MS } from "@vrooli/shared";

const REVOKED_SESSION_TIMEOUT = DAYS_90_MS;
const BATCH_SIZE = 100;

/**
 * Removes sessions from the database that have been revoked for a long time
 */
export async function cleanupRevokedSessions(): Promise<void> {
    try {
        const cutoffDate = new Date(Date.now() - REVOKED_SESSION_TIMEOUT);
        let totalDeleted = 0;
        let hasMore = true;

        // Process in batches to avoid overloading the database
        while (hasMore) {
            // Find sessions to delete
            const sessionsToDelete = await DbProvider.get().session.findMany({
                where: { revokedAt: { lte: cutoffDate } },
                select: { id: true },
                take: BATCH_SIZE,
            });

            if (sessionsToDelete.length === 0) {
                hasMore = false;
                break;
            }

            // Delete the batch
            const sessionIds = sessionsToDelete.map(session => session.id);
            const { count } = await DbProvider.get().session.deleteMany({
                where: { id: { in: sessionIds } },
            });

            totalDeleted += count;

            // If we got less than BATCH_SIZE, we're done
            if (sessionsToDelete.length < BATCH_SIZE) {
                hasMore = false;
            }
        }

        logger.info(`Deleted ${totalDeleted} revoked sessions.`);
    } catch (error) {
        logger.error("deleteRevokedSessions caught error", { error, trace: "0223" });
    }
}
