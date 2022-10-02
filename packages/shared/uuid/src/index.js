const validateRegex = /^(?:[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}|00000000-0000-0000-0000-000000000000)$/i;

/**
 * Generates a v4 UUID
 */
export const uuid = () => crypto.randomUUID();

/**
 * Validates a v4 UUID
 */
export const uuidValidate = (uuid) => {
    if (!uuid || typeof uuid !== 'string') return false;
    return validateRegex.test(uuid);
}