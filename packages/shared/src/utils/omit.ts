/**
 * Omit an array of keys from an object. Supports dot notation.
 * @param obj The object to omit keys from
 * @param keysToOmit The keys to omit
 * @returns The object with the omitted keys
 */
export function omit(obj: any, keysToOmit: string[]): any {
    // Make a shallow copy of the original object
    let result = { ...obj };
    // Loop through each key in the keysToOmit array
    for (const key of keysToOmit) {
      // Split the key using dot notation to extract nested keys
      const path = key.split(".");
      let current = result;
      // Loop through each nested key
      for (let i = 0; i < path.length - 1; i++) {
        // Get the current object based on the nested key
        current = current[path[i]];
      }
      // Delete the last nested key from the current object
      delete current[path[path.length - 1]];
      let currentKey = path[0];
      // Check if parent objects should be removed as well
      for (let i = 1; i < path.length; i++) {
        // If the current object is empty after removing its keys, remove it
        if (Object.keys(current).length === 0) {
          delete result[currentKey];
          break;
        }
        current = current[path[i]];
        currentKey = currentKey + "." + path[i];
      }
    }
    // Return the modified object
    return result;
  }