import crypto from 'crypto';

const validateRegex = /^(?:[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}|00000000-0000-0000-0000-000000000000)$/i;

const rnds8Pool = new Uint8Array(256); // # of random values to pre-allocate
let poolPtr = rnds8Pool.length;

const rng = () => {
    if (poolPtr > rnds8Pool.length - 16) {
        crypto.randomFillSync(rnds8Pool);
        poolPtr = 0;
    }
    return rnds8Pool.slice(poolPtr, (poolPtr += 16));
}

/**
 * Convert array of 16 byte values to UUID string format of the form:
 * XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX
 */
const byteToHex = [];
for (let i = 0; i < 256; ++i) {
    byteToHex.push((i + 0x100).toString(16).slice(1));
}

const unsafeStringify = (arr, offset = 0) => {
    // Note: Be careful editing this code!  It's been tuned for performance
    // and works in ways you may not expect. See https://github.com/uuidjs/uuid/pull/434
    return (
        byteToHex[arr[offset + 0]] +
        byteToHex[arr[offset + 1]] +
        byteToHex[arr[offset + 2]] +
        byteToHex[arr[offset + 3]] +
        '-' +
        byteToHex[arr[offset + 4]] +
        byteToHex[arr[offset + 5]] +
        '-' +
        byteToHex[arr[offset + 6]] +
        byteToHex[arr[offset + 7]] +
        '-' +
        byteToHex[arr[offset + 8]] +
        byteToHex[arr[offset + 9]] +
        '-' +
        byteToHex[arr[offset + 10]] +
        byteToHex[arr[offset + 11]] +
        byteToHex[arr[offset + 12]] +
        byteToHex[arr[offset + 13]] +
        byteToHex[arr[offset + 14]] +
        byteToHex[arr[offset + 15]]
    ).toLowerCase();
}

/**
 * Generates a v4 UUID. Attempts to use crypto.randomUUID() if available, otherwise falls back to
 * custom buffer-based implementation.
 */
export const uuid = () => {
    // Use crypto.randomUUID() if available
    if (typeof crypto?.randomUUID === 'function') return crypto.randomUUID();
    // Create a buffer to hold the random values
    const rnds = rng();
    // Per 4.4, set bits for version and `clock_seq_hi_and_reserved`
    rnds[6] = (rnds[6] & 0x0f) | 0x40;
    rnds[8] = (rnds[8] & 0x3f) | 0x80;
    return unsafeStringify(rnds);
}

/**
 * Validates a v4 UUID
 */
export const uuidValidate = (uuid) => {
    if (!uuid || typeof uuid !== 'string') return false;
    return validateRegex.test(uuid);
}

/**
 * Temporary ID to avoid infinite loops. Useful 
 * when ID must be specified for a schema, but formik is 
 * set to enableReinitialize
 */
export const DUMMY_ID = '11111111-1111-1111-1111-111111111111';