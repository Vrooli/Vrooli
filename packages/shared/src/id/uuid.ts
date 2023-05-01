import { NIL, v4 as uuidv4 } from "uuid";

const validateRegex = /^(?:[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}|00000000-0000-0000-0000-000000000000)$/i;

/**
 * Generates a v4 UUID
 */
export const uuid = () => uuidv4();

/**
 * Validates a v4 UUID
 */
export const uuidValidate = (uuid: any) => {
    if (!uuid || typeof uuid !== "string") return false;
    return validateRegex.test(uuid);
};

/**
 * Temporary ID to avoid infinite loops. Useful 
 * when ID must be specified for a schema, but formik is 
 * set to enableReinitialize
 */
export const DUMMY_ID = NIL;

