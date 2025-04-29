/**
 * Snowflake ID implementation
 * 
 * A 64-bit time-ordered distributed ID generator.
 * This implementation is based on Twitter's Snowflake ID.
 * 
 * Consists of:
 * - timestamp (41 bits) - milliseconds since the epoch
 * - worker ID (10 bits) - allows for 1024 different worker nodes
 * - sequence (12 bits) - allows for 4096 IDs per millisecond per worker
 * 
 * The implementation works well for distributed systems because worker IDs
 * prevent collisions across different nodes without requiring database coordination.
 */

// Constants
const EPOCH = 1609459200000; // Custom epoch (Jan 1, 2021 UTC)
const WORKER_ID_BITS = 10;
const SEQUENCE_BITS = 12;
const MAX_WORKER_ID = -1 ^ (-1 << WORKER_ID_BITS); // 1023
const MAX_SEQUENCE = -1 ^ (-1 << SEQUENCE_BITS); // 4095
const TIMESTAMP_LEFT_SHIFT = WORKER_ID_BITS + SEQUENCE_BITS; // 22
const WORKER_ID_SHIFT = SEQUENCE_BITS; // 12
const RANDOM_WORKER_ID_MAX = 900; // Maximum random portion of worker ID
const BIGINT_BITS_COUNT = 64; // Number of bits in a BigInt for Snowflake

// Default workerId based on a combination of process ID and a random value
// This should be overridden in production with a stable worker ID
let workerId = (process.pid % 100) + Math.floor(Math.random() * RANDOM_WORKER_ID_MAX);
let sequence = 0;
let lastTimestamp = -1;

/**
 * Set the worker ID for the node
 * @param id Worker ID between 0 and 1023
 */
export function setWorkerId(id: number): void {
    if (id < 0 || id > MAX_WORKER_ID) {
        throw new Error(`Worker ID must be between 0 and ${MAX_WORKER_ID}`);
    }
    workerId = id;
}

/**
 * Waits until the next millisecond
 */
function tilNextMillis(lastTimestamp: number): number {
    let timestamp = Date.now();
    while (timestamp <= lastTimestamp) {
        timestamp = Date.now();
    }
    return timestamp;
}

/**
 * Generates a Snowflake ID as a bigint
 */
export function nextBigInt(): bigint {
    let timestamp = Date.now();

    // If current time is less than last timestamp, there is a clock drift
    if (timestamp < lastTimestamp) {
        throw new Error(`Clock moved backwards. Refusing to generate id for ${lastTimestamp - timestamp} milliseconds`);
    }

    // If current timestamp equals last timestamp, increment sequence
    if (timestamp === lastTimestamp) {
        sequence = (sequence + 1) & MAX_SEQUENCE;
        // If sequence overflows, wait until next millisecond
        if (sequence === 0) {
            timestamp = tilNextMillis(lastTimestamp);
        }
    } else {
        // Reset sequence for this new millisecond
        sequence = 0;
    }

    lastTimestamp = timestamp;

    // Combine components into a 64-bit ID
    const id = BigInt(timestamp - EPOCH) << BigInt(TIMESTAMP_LEFT_SHIFT) |
        BigInt(workerId) << BigInt(WORKER_ID_SHIFT) |
        BigInt(sequence);

    return id;
}

/**
 * Validates if a string is a valid Snowflake ID
 */
export function validatePK(id: unknown): boolean {
    if (typeof id !== "string" && typeof id !== "bigint") return false;
    try {
        // Convert to BigInt if it's a string
        const snowflakeId = typeof id === "string" ? BigInt(id) : id;

        // Define constants for BigInt comparisons
        const ZERO_BIG = BigInt(0);
        const MAX_BITS = BigInt(BIGINT_BITS_COUNT);
        const TWO_BIG = BigInt(2);

        // Valid Snowflake IDs should be positive and fit in 64 bits
        return snowflakeId > ZERO_BIG && snowflakeId < TWO_BIG ** MAX_BITS;
    } catch (error) {
        return false;
    }
}

/**
 * Extract timestamp from Snowflake ID
 */
export function getTimestampFromId(id: string | bigint): Date {
    const snowflakeId = typeof id === "string" ? BigInt(id) : id;
    const timestamp = Number((snowflakeId >> BigInt(TIMESTAMP_LEFT_SHIFT)) + BigInt(EPOCH));
    return new Date(timestamp);
}

/**
 * Generate a primary key for database records
 */
export function generatePK(): bigint {
    return nextBigInt();
}

/**
 * Generate a primary key as a string
 */
export function generatePKString(): string {
    return nextBigInt().toString();
}

/**
 * Dummy ID for testing purposes and satisfying type requirements. 
 * Validation schemas should detect and cast this to a valid ID.
 */
export const DUMMY_ID = "0";