// eslint-disable-next-line no-magic-numbers
export const MB = 1024 * 1024;
export const DEFAULT_MEMORY_LIMIT_MB = 16; // Minimum is 8
/** How often to check for a heartbeat from the worker thread, in milliseconds */
export const DEFAULT_HEARTBEAT_CHECK_INTERVAL_MS = 1_000;
/** How long to wait for a heartbeat from the worker thread, in milliseconds */
export const DEFAULT_HEARTBEAT_TIMEOUT_MS = 3_000;
/** How often the worker thread should send a heartbeat to the parent thread, in milliseconds */
export const DEFAULT_HEARTBEAT_SEND_INTERVAL_MS = 500;
/** How long to wait for the worker thread to finish executing a job, in milliseconds */
export const DEFAULT_IDLE_TIMEOUT_MS = 30_000;
/** How long to wait for the worker thread to finish executing a job, in milliseconds */
export const DEFAULT_JOB_TIMEOUT_MS = 500;
/** How long to wait for the worker thread to be ready to receive a job, in milliseconds */
export const WORKER_READY_TIMEOUT_MS = process.env.NODE_ENV === "test" ? 5_000 : 1_000;
