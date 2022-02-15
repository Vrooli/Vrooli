export const removeTypename = (value) => {
    if (value === null || value === undefined) return value;
    if (Array.isArray(value)) {
        return value.map(v => removeTypename(v));
    }
    if (typeof value === 'object') {
        const newObj = {};
        Object.keys(value).forEach(key => {
            if (key !== '__typename') {
                newObj[key] = removeTypename(value[key])
            }
        })
        return newObj;
    }
    return value;
  };