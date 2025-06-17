import { HOURS_2_MS, MINUTES_1_MS, MINUTES_5_MS, SECONDS_10_MS } from "@vrooli/shared";
import type { ActiveTaskRegistryLimits } from "../activeTaskRegistry.js";

export const RUN_QUEUE_LIMITS: ActiveTaskRegistryLimits = {
    maxActive: parseInt(process.env.WORKER_RUN_MAX_ACTIVE || "1000"),
    highLoadCheckIntervalMs: parseInt(process.env.WORKER_RUN_HIGH_LOAD_CHECK_INTERVAL_MS || MINUTES_1_MS.toString()),
    highLoadThresholdPercentage: parseFloat(process.env.WORKER_RUN_HIGH_LOAD_THRESHOLD_PERCENTAGE || "0.8"),
    longRunningThresholdFreeMs: parseInt(process.env.WORKER_RUN_LONG_RUNNING_THRESHOLD_FREE_MS || MINUTES_1_MS.toString()),
    longRunningThresholdPremiumMs: parseInt(process.env.WORKER_RUN_LONG_RUNNING_THRESHOLD_PREMIUM_MS || MINUTES_5_MS.toString()),
    taskTimeoutMs: parseInt(process.env.WORKER_RUN_TASK_TIMEOUT_MS || HOURS_2_MS.toString()),
    shutdownGracePeriodMs: parseInt(process.env.WORKER_RUN_SHUTDOWN_GRACE_PERIOD_MS || SECONDS_10_MS.toString()),
    onLongRunningFirstThreshold: (process.env.WORKER_RUN_ON_LONG_RUNNING_FIRST_THRESHOLD || "pause") as "pause" | "stop",
    longRunningPauseRetries: parseInt(process.env.WORKER_RUN_LONG_RUNNING_PAUSE_RETRIES || "1"),
    longRunningStopRetries: parseInt(process.env.WORKER_RUN_LONG_RUNNING_STOP_RETRIES || "0"),
};