// Functions for manipulating state objects

// Grabs data from an object using dot notation (ex: 'parent.child.property')
export const valueFromDot = (object, notation) => {
    function index(object, i) { return object[i] }
    if (!object || !notation) return null;
    return notation.split('.').reduce(index, object);
}

export const arrayValueFromDot = (object, notation, index) => {
    const value = valueFromDot(object, notation);
    if (!value || !Array.isArray(value) || index <= 0 || value.length >= index) return null;
    return value[index];
}

// Maps the keys of an object to dot notation
export function convertToDot(obj, parent = [], keyValue = {}) {
    for (let key in obj) {
      let keyPath: any = [...parent, key];
      if (obj[key]!== null && typeof obj[key] === 'object') {
          Object.assign(keyValue, convertToDot(obj[key], keyPath, keyValue));
      } else {
          keyValue[keyPath.join('.')] = obj[key];
      }
  }
  return keyValue;
}

/**
 * Compares an object against its updates. Returns fields that have changed.
 * For non-primitive values, splits field into 4 parts:
 * - connections - existing objects which are being newly connected to the object (e.g. adding an existing standard to a routine input)
 * - disconnections - existing objects which are being disconnected from the object (e.g. removing a standard from a routine input)
 * - updates - existing objects which are being updated (e.g. changing a standard's name)
 */