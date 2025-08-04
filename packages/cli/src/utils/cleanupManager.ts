import chalk from "chalk";

// Cleanup priorities
const CLEANUP_PRIORITY = {
    HIGH: 10,
    MEDIUM: 20,
    LOW: 30,
    DEFAULT: 50,
} as const;

// Timeouts
const ASYNC_CLEANUP_TIMEOUT_MS = 5000;

type CleanupTask = {
    name: string;
    handler: () => void | Promise<void>;
    priority: number; // Lower number = higher priority (runs first)
};

/**
 * Manages cleanup tasks for graceful shutdown
 * Ensures resources are properly cleaned up before process exit
 */
export class CleanupManager {
    private static instance: CleanupManager | null = null;
    private cleanupTasks: CleanupTask[] = [];
    private isCleaningUp = false;
    private hasCleanedUp = false;
    
    private constructor() {
        // Private constructor for singleton
    }
    
    static getInstance(): CleanupManager {
        if (!CleanupManager.instance) {
            CleanupManager.instance = new CleanupManager();
        }
        return CleanupManager.instance;
    }
    
    /**
     * Register a cleanup task
     * @param name - Name of the task for debugging
     * @param handler - Cleanup function (can be async)
     * @param priority - Priority for execution order (default: 50)
     */
    register(name: string, handler: () => void | Promise<void>, priority = CLEANUP_PRIORITY.DEFAULT): void {
        // Check if task already registered
        const existingIndex = this.cleanupTasks.findIndex(t => t.name === name);
        if (existingIndex >= 0) {
            // Replace existing task
            this.cleanupTasks[existingIndex] = { name, handler, priority };
        } else {
            this.cleanupTasks.push({ name, handler, priority });
        }
        
        // Keep tasks sorted by priority
        this.cleanupTasks.sort((a, b) => a.priority - b.priority);
    }
    
    /**
     * Unregister a cleanup task
     * @param name - Name of the task to remove
     */
    unregister(name: string): void {
        this.cleanupTasks = this.cleanupTasks.filter(t => t.name !== name);
    }
    
    /**
     * Execute all cleanup tasks
     * @param exitCode - Exit code to return after cleanup
     * @param error - Optional error that triggered cleanup
     * @returns Promise that resolves when all cleanup is done
     */
    async executeAll(exitCode = 0, error?: Error): Promise<void> {
        // Prevent double cleanup
        if (this.isCleaningUp || this.hasCleanedUp) {
            return;
        }
        
        this.isCleaningUp = true;
        
        if (process.env.DEBUG) {
            console.error(chalk.yellow(`\n[DEBUG] Starting cleanup with exit code ${exitCode}`));
            if (error) {
                console.error(chalk.yellow(`[DEBUG] Error: ${error.message}`));
            }
        }
        
        // Execute cleanup tasks in priority order
        for (const task of this.cleanupTasks) {
            try {
                if (process.env.DEBUG) {
                    console.error(chalk.gray(`[DEBUG] Running cleanup: ${task.name}`));
                }
                
                const result = task.handler();
                
                // Handle async cleanup tasks
                if (result instanceof Promise) {
                    // Set a timeout for async cleanup to prevent hanging
                    await Promise.race([
                        result,
                        new Promise((resolve) => setTimeout(resolve, ASYNC_CLEANUP_TIMEOUT_MS)),
                    ]);
                }
            } catch (cleanupError) {
                // Log but don't throw - we want to continue cleanup
                if (process.env.DEBUG) {
                    console.error(chalk.red(`[DEBUG] Cleanup task '${task.name}' failed:`), cleanupError);
                }
            }
        }
        
        this.hasCleanedUp = true;
        this.isCleaningUp = false;
        
        if (process.env.DEBUG) {
            console.error(chalk.green("[DEBUG] Cleanup completed"));
        }
    }
    
    /**
     * Synchronous cleanup for exit handlers
     * Uses sync versions where possible, skips async tasks
     */
    executeAllSync(_exitCode = 0): void {
        // Prevent double cleanup
        if (this.isCleaningUp || this.hasCleanedUp) {
            return;
        }
        
        this.isCleaningUp = true;
        
        // Execute only synchronous cleanup tasks
        for (const task of this.cleanupTasks) {
            try {
                const result = task.handler();
                // Skip if it returns a promise (async task)
                if (result instanceof Promise) {
                    if (process.env.DEBUG) {
                        console.error(chalk.yellow(`[DEBUG] Skipping async cleanup task: ${task.name}`));
                    }
                }
            } catch (cleanupError) {
                // Silently continue
            }
        }
        
        this.hasCleanedUp = true;
        this.isCleaningUp = false;
    }
    
    /**
     * Check if cleanup has been performed
     */
    get cleanedUp(): boolean {
        return this.hasCleanedUp;
    }
    
    /**
     * Reset cleanup state (mainly for testing)
     */
    reset(): void {
        this.cleanupTasks = [];
        this.isCleaningUp = false;
        this.hasCleanedUp = false;
    }
}

// Export singleton instance
export const cleanup = CleanupManager.getInstance();

// Export priority constants for external use
export { CLEANUP_PRIORITY };
