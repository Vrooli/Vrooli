import { DbProvider } from "./provider.js";
import { logger } from "../events/logger.js";

export interface MigrationStatus {
    hasPending: boolean;
    pendingCount: number;
    pendingMigrations: string[];
    appliedCount: number;
    appliedMigrations: string[];
    error?: string;
}

export class MigrationService {
    /**
     * Check the status of database migrations
     * Returns information about applied and pending migrations
     */
    static async checkStatus(): Promise<MigrationStatus> {
        try {
            const db = DbProvider.get();
            
            // Query the Prisma migrations table
            const migrations = await db.$queryRaw<Array<{ migration_name: string; finished_at: Date | null }>>`
                SELECT migration_name, finished_at 
                FROM _prisma_migrations 
                ORDER BY migration_name ASC
            `;
            
            // Separate applied and pending migrations
            const appliedMigrations = migrations
                .filter(m => m.finished_at !== null)
                .map(m => m.migration_name);
            
            const pendingMigrations = migrations
                .filter(m => m.finished_at === null)
                .map(m => m.migration_name);
            
            return {
                hasPending: pendingMigrations.length > 0,
                pendingCount: pendingMigrations.length,
                pendingMigrations,
                appliedCount: appliedMigrations.length,
                appliedMigrations,
            };
        } catch (error) {
            logger.error("Failed to check migration status", { error });
            throw error;
        }
    }
}