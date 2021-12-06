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