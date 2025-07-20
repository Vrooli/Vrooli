import { existsSync, mkdirSync, readFileSync, writeFileSync } from "fs";
import { join } from "path";
import { logger } from "../../../../events/logger.js";

interface IntegrationTestState {
    lastRuns: {
        [testSuite: string]: {
            timestamp: string;
            count: number;
        }
    }
}

export class IntegrationTestRunner {
    private stateFile: string;
    private readonly rateLimit = 1; // Once per day
    
    constructor() {
        // Use data directory at project root
        // Go up from server/src/services/response/providers/__test to root
        const projectRoot = join(__dirname, "..", "..", "..", "..", "..", "..", "..");
        const dataDir = join(projectRoot, "data", "test-state");
        
        // Create directory if it doesn't exist
        if (!existsSync(dataDir)) {
            mkdirSync(dataDir, { recursive: true });
        }
        
        this.stateFile = join(dataDir, "integration-tests.json");
    }
    
    /**
     * Check if a test suite can run based on environment variables and rate limits
     */
    canRunTest(suiteName: string): boolean {
        // Check if integration tests are enabled
        if (!process.env.VROOLI_INTEGRATION_TESTS_ENABLED) {
            logger.debug(`Integration tests disabled for ${suiteName} (set VROOLI_INTEGRATION_TESTS_ENABLED=true to enable)`);
            return false;
        }
        
        // Force run if specified
        if (process.env.VROOLI_INTEGRATION_TESTS_FORCE === "true") {
            logger.info(`Force running integration tests for ${suiteName}`);
            return true;
        }
        
        // Check rate limit
        const state = this.loadState();
        const lastRun = state.lastRuns[suiteName];
        
        if (!lastRun) {
            logger.info(`First time running integration tests for ${suiteName}`);
            return true;
        }
        
        const lastRunDate = new Date(lastRun.timestamp);
        const now = new Date();
        
        // Reset count if it's a new day
        if (this.isDifferentDay(lastRunDate, now)) {
            logger.info(`New day - resetting integration test count for ${suiteName}`);
            return true;
        }
        
        const canRun = lastRun.count < this.rateLimit;
        if (!canRun) {
            logger.info(`Integration test limit reached for ${suiteName} today (${lastRun.count}/${this.rateLimit})`);
        }
        
        return canRun;
    }
    
    /**
     * Record that a test suite has run
     */
    recordTestRun(suiteName: string): void {
        const state = this.loadState();
        const now = new Date();
        const today = now.toISOString().split("T")[0];
        
        const lastRun = state.lastRuns[suiteName];
        const lastRunDate = lastRun ? new Date(lastRun.timestamp).toISOString().split("T")[0] : null;
        
        if (lastRunDate === today) {
            // Increment count for today
            state.lastRuns[suiteName].count++;
        } else {
            // New day, reset count
            state.lastRuns[suiteName] = {
                timestamp: now.toISOString(),
                count: 1,
            };
        }
        
        this.saveState(state);
        logger.info(`Recorded integration test run for ${suiteName} (count: ${state.lastRuns[suiteName].count})`);
    }
    
    /**
     * Get status of all integration test suites
     */
    getStatus(): Record<string, { lastRun: string; runsToday: number; canRunToday: boolean }> {
        const state = this.loadState();
        const status: Record<string, { lastRun: string; runsToday: number; canRunToday: boolean }> = {};
        
        for (const [suite, data] of Object.entries(state.lastRuns)) {
            const runsToday = this.isToday(new Date(data.timestamp)) ? data.count : 0;
            status[suite] = {
                lastRun: new Date(data.timestamp).toLocaleString(),
                runsToday,
                canRunToday: runsToday < this.rateLimit,
            };
        }
        
        return status;
    }
    
    private loadState(): IntegrationTestState {
        if (!existsSync(this.stateFile)) {
            return { lastRuns: {} };
        }
        
        try {
            const content = readFileSync(this.stateFile, "utf-8");
            return JSON.parse(content);
        } catch (error) {
            logger.warn("Failed to load integration test state, starting fresh", error);
            return { lastRuns: {} };
        }
    }
    
    private saveState(state: IntegrationTestState): void {
        try {
            writeFileSync(this.stateFile, JSON.stringify(state, null, 2));
        } catch (error) {
            logger.error("Failed to save integration test state", error);
        }
    }
    
    private isDifferentDay(date1: Date, date2: Date): boolean {
        return date1.toDateString() !== date2.toDateString();
    }
    
    private isToday(date: Date): boolean {
        return date.toDateString() === new Date().toDateString();
    }
}

// Allow running this file directly to check status
if (process.argv[1] === __filename) {
    const runner = new IntegrationTestRunner();
    console.log("Integration Test Status:");
    console.log(JSON.stringify(runner.getStatus(), null, 2));
}
