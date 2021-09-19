// Functions for manipulating state objects

export const addToObject = (object, key, value) => {
    return {...object, [key]: value};
}

export const updateObject = (object, key, value) => {
    if (JSON.stringify(object.key) === JSON.stringify(value)) return object;
    return {...object, [key]: value};
}

export const deleteObjectKey = (object, key) => {
    let copy = {...object};
    delete copy[key];
    return copy;
}

// Grabs data from an object using dot notation (ex: 'parent.child.property')
export const valueFromDot = (object, notation) => {
    function index(object, i) { return object[i] }
    if (!notation) return null;
    return notation.split('.').reduce(index, object);
}

// Maps the keys of an object to dot notation
export function convertToDot(obj, parent = [], keyValue = {}) {
    for (let key in obj) {
      let keyPath = [...parent, key];
      if (obj[key]!== null && typeof obj[key] === 'object') {
          Object.assign(keyValue, convertToDot(obj[key], keyPath, keyValue));
      } else {
          keyValue[keyPath.join('.')] = obj[key];
      }
  }
  return keyValue;
}