/**
 * Test record tracking utility for coordinated cleanup
 * 
 * This utility helps track all records created during tests (both factory-created
 * and manually created) to enable comprehensive cleanup.
 */

import { type PrismaClient } from "@prisma/client";

/**
 * Information about tracked records for a specific table
 */
interface TrackedTable {
    ids: Set<bigint>;
    dependencyLevel: number; // For cleanup ordering
}

/**
 * Singleton record tracker for coordinating test cleanup
 * Tracks records across factories and manual creation
 */
class TestRecordTrackerClass {
    private records = new Map<string, TrackedTable>();
    private isActive = false;
    
    /**
     * Dependency levels for cleanup order (children deleted before parents)
     * Higher numbers = deleted first, lower numbers = deleted last
     */
    private readonly dependencyLevels: Record<string, number> = {
        // Level 20: Deepest children
        "award": 20,
        "user_translation": 20,
        "team_translation": 20,
        "chat_translation": 20,
        "comment_translation": 20,
        "issue_translation": 20,
        "meeting_translation": 20,
        "pull_request_translation": 20,
        "resource_translation": 20,
        
        // Level 19: Execution children
        "run_step": 19,
        "run_io": 19,
        
        // Level 18: Chat children
        "chat_message": 18,
        "chat_participants": 18,
        "chat_invite": 18,
        
        // Level 17: Meeting/team children
        "meeting_attendees": 17,
        "meeting_invite": 17,
        "member": 17,
        "member_invite": 17,
        
        // Level 16: Auth children
        "session": 16,
        "user_auth": 16,
        
        // Level 15: Contact/notification children
        "email": 15,
        "phone": 15,
        "push_device": 15,
        "notification": 15,
        "notification_subscription": 15,
        
        // Level 14: Financial children
        "credit_ledger_entry": 14,
        "payment": 14,
        
        // Level 13: Content children
        "comment": 13,
        "bookmark": 13,
        "bookmark_list": 13,
        "reaction": 13,
        "reaction_summary": 13,
        "view": 13,
        "report_response": 13,
        "report": 13,
        
        // Level 12: Resource relationships
        "resource_version_relation": 12,
        "resource_version": 12,
        "pull_request": 12,
        
        // Level 11: API and external
        "api_key": 11,
        "api_key_external": 11,
        
        // Level 10: Schedule/reminder children
        "schedule_exception": 10,
        "schedule_recurrence": 10,
        "reminder_item": 10,
        
        // Level 9: Issues and runs
        "issue": 9,
        "run": 9,
        
        // Level 8: Tags and schedules
        "resource_tag": 8,
        "team_tag": 8,
        "schedule": 8,
        "reminder_list": 8,
        "reminder": 8,
        
        // Level 7: Financial parents
        "credit_account": 7,
        "plan": 7,
        "wallet": 7,
        "transfer": 7,
        
        // Level 6: Stats
        "stats_site": 6,
        "stats_team": 6,
        "stats_user": 6,
        "stats_resource": 6,
        
        // Level 5: Meetings
        "meeting": 5,
        
        // Level 4: Tags
        "tag": 4,
        
        // Level 3: Chats
        "chat": 3,
        
        // Level 2: Resources
        "resource": 2,
        
        // Level 1: Teams (must be deleted before users due to some relationships)
        "team": 1,
        
        // Level 0: Users (root entities)
        "user": 0,
    };
    
    /**
     * Start tracking records for the current test
     */
    start(): void {
        this.isActive = true;
        this.records.clear();
    }
    
    /**
     * Stop tracking records
     */
    stop(): void {
        this.isActive = false;
    }
    
    /**
     * Track a record ID for cleanup
     */
    track(table: string, id: bigint): void {
        if (!this.isActive) return;
        
        if (!this.records.has(table)) {
            this.records.set(table, {
                ids: new Set(),
                dependencyLevel: this.dependencyLevels[table] ?? 10, // Default level
            });
        }
        
        this.records.get(table)!.ids.add(id);
    }
    
    /**
     * Track multiple record IDs for cleanup
     */
    trackMany(table: string, ids: bigint[]): void {
        if (!this.isActive) return;
        
        for (const id of ids) {
            this.track(table, id);
        }
    }
    
    /**
     * Get all tracked records for a table
     */
    getTrackedIds(table: string): bigint[] {
        const tracked = this.records.get(table);
        return tracked ? Array.from(tracked.ids) : [];
    }
    
    /**
     * Get tracking summary for debugging
     */
    getSummary(): Record<string, number> {
        const summary: Record<string, number> = {};
        this.records.forEach(({ ids }, table) => {
            if (ids.size > 0) {
                summary[table] = ids.size;
            }
        });
        return summary;
    }
    
    /**
     * Clean up all tracked records in proper dependency order
     */
    async cleanup(prisma: PrismaClient): Promise<void> {
        if (this.records.size === 0) {
            return;
        }
        
        // Sort tables by dependency level (highest first)
        const entries: Array<[string, TrackedTable]> = [];
        this.records.forEach((value, key) => {
            entries.push([key, value]);
        });
        const sortedTables = entries
            .filter(([, { ids }]) => ids.size > 0)
            .sort(([, a], [, b]) => b.dependencyLevel - a.dependencyLevel)
            .map(([table]) => table);
        
        // Clean up tables in dependency order
        for (const table of sortedTables) {
            const tracked = this.records.get(table);
            if (!tracked || tracked.ids.size === 0) continue;
            
            try {
                const idsArray = Array.from(tracked.ids);
                await (prisma as any)[table].deleteMany({
                    where: {
                        id: {
                            in: idsArray,
                        },
                    },
                });
                
                // Clear the tracked IDs after successful deletion
                tracked.ids.clear();
            } catch (error) {
                console.error(`Failed to cleanup tracked records in table '${table}':`, error);
                // Continue with other tables even if one fails
            }
        }
        
        // Clear all tracking data
        this.records.clear();
    }
    
    /**
     * Clear tracking data without cleaning up records
     */
    clear(): void {
        this.records.clear();
    }
    
    /**
     * Check if tracker is currently active
     */
    get active(): boolean {
        return this.isActive;
    }
    
    /**
     * Get count of tracked tables
     */
    get trackedTableCount(): number {
        let count = 0;
        this.records.forEach(({ ids }) => {
            if (ids.size > 0) count++;
        });
        return count;
    }
    
    /**
     * Get total count of tracked records
     */
    get totalTrackedRecords(): number {
        let total = 0;
        this.records.forEach(({ ids }) => {
            total += ids.size;
        });
        return total;
    }
}

/**
 * Singleton instance of the record tracker
 */
export const TestRecordTracker = new TestRecordTrackerClass();

/**
 * Decorator for Prisma create operations to automatically track created records
 * This can be used to wrap Prisma operations and automatically track their results
 */
export function withTracking<T extends { id: bigint }>(
    table: string,
    operation: () => Promise<T>
): Promise<T>;
export function withTracking<T extends { id: bigint }[]>(
    table: string,
    operation: () => Promise<T>
): Promise<T>;
export async function withTracking<T extends { id: bigint } | { id: bigint }[]>(
    table: string,
    operation: () => Promise<T>,
): Promise<T> {
    const result = await operation();
    
    if (Array.isArray(result)) {
        TestRecordTracker.trackMany(table, result.map(r => r.id));
    } else {
        TestRecordTracker.track(table, result.id);
    }
    
    return result;
}

/**
 * Helper to manually track a record creation
 * Use this when creating records outside of factories
 */
export function trackRecord(table: string, id: bigint): void {
    TestRecordTracker.track(table, id);
}

/**
 * Helper to manually track multiple record creations
 */
export function trackRecords(table: string, ids: bigint[]): void {
    TestRecordTracker.trackMany(table, ids);
}

/**
 * Test lifecycle helpers for easy integration
 */
export const testLifecycle = {
    /**
     * Start tracking at the beginning of a test
     */
    beforeEach(): void {
        TestRecordTracker.start();
    },
    
    /**
     * Clean up and stop tracking at the end of a test
     */
    async afterEach(prisma: PrismaClient): Promise<void> {
        try {
            await TestRecordTracker.cleanup(prisma);
        } finally {
            TestRecordTracker.stop();
        }
    },
    
    /**
     * Clean up without stopping tracking (useful for mid-test cleanup)
     */
    async cleanup(prisma: PrismaClient): Promise<void> {
        await TestRecordTracker.cleanup(prisma);
    },
} as const;
