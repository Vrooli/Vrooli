const validateRegex = /^(?:[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}|00000000-0000-0000-0000-000000000000)$/i;

/**
 * Generates a v4 UUID
 */
export const uuid = () => {
    let
      d = new Date().getTime(),
      d2 = ((typeof performance !== 'undefined') && performance.now && (performance.now() * 1000)) || 0;
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, c => {
      let r = Math.random() * 16;
      if (d > 0) {
        r = (d + r) % 16 | 0;
        d = Math.floor(d / 16);
      } else {
        r = (d2 + r) % 16 | 0;
        d2 = Math.floor(d2 / 16);
      }
      return (c == 'x' ? r : (r & 0x7 | 0x8)).toString(16);
    });
  };

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