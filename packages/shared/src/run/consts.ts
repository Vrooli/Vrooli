/* c8 ignore start */
import { API_CREDITS_MULTIPLIER } from "../consts/api.js";
import { DOLLARS_1_CENTS, MB_1_BYTES, MINUTES_1_MS, MINUTES_5_MS, SECONDS_1_MS, SECONDS_5_MS } from "../consts/numbers.js";

export const LATEST_RUN_CONFIG_VERSION = "1.0";

/** Limit routines to $1 for now */
export const DEFAULT_MAX_RUN_CREDITS = BigInt(DOLLARS_1_CENTS) * API_CREDITS_MULTIPLIER;
/** Limit routines to 5 minutes for now */
export const DEFAULT_MAX_RUN_TIME = MINUTES_5_MS;

/** 
 * How many branches can be active at once.
 * If there are more branches than this, they will be run in batches.
 */
export const MAX_PARALLEL_BRANCHES = 10;
/**
 * How many times the main loop can run before pausing.
 */
export const MAX_MAIN_LOOP_ITERATIONS = 100;
/** 
 * How quickly to increase the main loop delay when all branches are waiting.
 * Must be greater than 1.
 */
export const DEFAULT_LOOP_DELAY_MULTIPLIER = 1.1;
/**
 * The maximum delay between main loop iterations.
 */
export const DEFAULT_MAX_LOOP_DELAY_MS = MINUTES_1_MS;
/**
 * Default behavior when a branch fails
 */
export const DEFAULT_ON_BRANCH_FAILURE = "Stop";

// Cache limits
export const BPMN_DEFINITIONS_CACHE_LIMIT = 1000;
export const BPMM_INSTANCES_CACHE_MAX_SIZE_BYTES = MB_1_BYTES;

export const BPMN_ELEMENT_CACHE_LIMIT = 1000;
export const BPMN_ELEMENT_CACHE_MAX_SIZE_BYTES = MB_1_BYTES;

export const ROUTINE_CACHE_LIMIT = 1000;
export const ROUTINE_CACHE_MAX_SIZE_BYTES = MB_1_BYTES;

export const PROJECT_CACHE_LIMIT = 1000;
export const PROJECT_CACHE_MAX_SIZE_BYTES = MB_1_BYTES;

/**
 * Debounce time for storing run progress
 */
export const STORE_RUN_PROGRESS_DEBOUNCE_MS = SECONDS_1_MS;
/**
 * How long to wait before finalizing a run (i.e. waiting for all saved progress to be stored)
 * is considered a timeout.
 */
export const FINALIZE_RUN_TIMEOUT_MS = SECONDS_5_MS;
export const FINALIZE_RUN_POLL_INTERVAL_MS = 50;

/**
 * Throttle time for sending progress updates to the client
 */
export const SEND_PROGRESS_UPDATE_THROTTLE_MS = SECONDS_1_MS;
