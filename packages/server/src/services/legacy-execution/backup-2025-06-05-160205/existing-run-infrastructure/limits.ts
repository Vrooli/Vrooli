import { RunStatus } from "../api/types.js";
import { type PassableLogger } from "../consts/commonTypes.js";
import { type RunLimitBehavior, type RunProgress, type RunRequestLimits, RunStatusChangeReason } from "./types.js";

/**
 * Result of limit checking function when a limit has been reached.
 */
type LimitReachedResult = {
    /** The reason for the status change */
    reason: RunStatusChangeReason;
    /** The limit behavior, if any */
    behavior: RunLimitBehavior | undefined;
};

/**
 * Performs limit checking and handling for runs.
 */
export class RunLimitsManager {
    private logger: PassableLogger;

    constructor(logger: PassableLogger) {
        this.logger = logger;
    }

    /**
     * Checks if we have reached the time limit.
     * 
     * @param run The current run
     * @param runLimits The limits for this run
     * @param startTime The time when the run started
     * @returns Undefined if the limit has not been reached, or LimitReachedResult if it has.
     */
    private checkTimeLimit(run: RunProgress, runLimits: RunRequestLimits, startTime: number): LimitReachedResult | undefined {
        const elapsed = Date.now() - startTime;
        if (!runLimits.maxTime || elapsed < runLimits.maxTime) {
            return undefined;
        }

        this.logger.error(`checkLimits: Run ${run.runId} reached time limit.`);
        return {
            reason: RunStatusChangeReason.MaxTime,
            behavior: runLimits.onMaxTime,
        };
    }

    /**
     * Checks if we have reached the credits limit.
     * 
     * @param run The current run
     * @param runLimits The limits for this run
     * @returns Undefined if the limit has not been reached, or LimitReachedResult if it has.
     */
    private checkCreditsLimit(run: RunProgress, runLimits: RunRequestLimits): LimitReachedResult | undefined {
        if (!runLimits.maxCredits) {
            return undefined;
        }

        const spent = BigInt(run.metrics.creditsSpent || "0");
        const limit = BigInt(runLimits.maxCredits);
        if (spent < limit) {
            return undefined;
        }

        this.logger.error(`checkLimits: Run ${run.runId} reached credits limit.`);
        return {
            reason: RunStatusChangeReason.MaxCredits,
            behavior: runLimits.onMaxCredits,
        };
    }

    /**
     * Checks if we have reached the steps limit.
     * 
     * @param run The current run
     * @param runLimits The limits for this run
     * @returns Undefined if the limit has not been reached, or LimitReachedResult if it has.
     */
    private checkStepsLimit(run: RunProgress, runLimits: RunRequestLimits): LimitReachedResult | undefined {
        if (runLimits.maxSteps === undefined || run.metrics.stepsRun < runLimits.maxSteps) {
            return undefined;
        }

        this.logger.error(`checkLimits: Run ${run.runId} reached steps limit.`);
        return {
            reason: RunStatusChangeReason.MaxSteps,
            behavior: runLimits.onMaxSteps,
        };
    }

    /**
     * Helper function to determine new status if we've hit a limit.
     * 
     * @param behavior The behavior to use if we've hit a limit
     * @returns The new status
     */
    private getNextStatus(behavior: RunLimitBehavior | undefined) {
        return behavior === "Pause" ? RunStatus.Paused : RunStatus.Failed;
    }

    /**
     * Checks if we have reached any time/credits/steps limit.
     * 
     * Also updates the run status if a limit has been reached.
     * 
     * @param runProgress The current run
     * @param runLimits The limits for this run
     * @param startTime The time when the run started
     * @returns A reason for the status change, if limits have been reached. 
     * Otherwise, undefined.
     */
    public checkLimits(
        run: RunProgress,
        runLimits: RunRequestLimits,
        startTime: number,
    ): RunStatusChangeReason | undefined {
        const limitReached =
            this.checkTimeLimit(run, runLimits, startTime) ||
            this.checkCreditsLimit(run, runLimits) ||
            this.checkStepsLimit(run, runLimits);
        if (!limitReached) {
            return undefined;
        }

        const { reason, behavior } = limitReached;
        run.status = this.getNextStatus(behavior);
        return reason;
    }
}
