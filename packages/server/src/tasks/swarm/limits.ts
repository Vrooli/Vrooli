import type { ActiveTaskRegistryLimits } from "../activeTaskRegistry.js";

export const SWARM_QUEUE_LIMITS: ActiveTaskRegistryLimits = {
    maxActive: parseInt(process.env.WORKER_SWARM_MAX_ACTIVE || "10"),
    highLoadCheckIntervalMs: parseInt(process.env.WORKER_SWARM_HIGH_LOAD_CHECK_INTERVAL_MS || "5000"),
    highLoadThresholdPercentage: parseFloat(process.env.WORKER_SWARM_HIGH_LOAD_THRESHOLD_PERCENTAGE || "0.8"),
    longRunningThresholdFreeMs: parseInt(process.env.WORKER_SWARM_LONG_RUNNING_THRESHOLD_FREE_MS || "60000"),
    longRunningThresholdPremiumMs: parseInt(process.env.WORKER_SWARM_LONG_RUNNING_THRESHOLD_PREMIUM_MS || "300000"),
    taskTimeoutMs: parseInt(process.env.WORKER_SWARM_TASK_TIMEOUT_MS || "600000"),
    shutdownGracePeriodMs: parseInt(process.env.WORKER_SWARM_SHUTDOWN_GRACE_PERIOD_MS || "30000"),
    onLongRunningFirstThreshold: (process.env.WORKER_SWARM_ON_LONG_RUNNING_FIRST_THRESHOLD || "pause") as "pause" | "stop",
    longRunningPauseRetries: parseInt(process.env.WORKER_SWARM_LONG_RUNNING_PAUSE_RETRIES || "1"),
    longRunningStopRetries: parseInt(process.env.WORKER_SWARM_LONG_RUNNING_STOP_RETRIES || "0"),
};