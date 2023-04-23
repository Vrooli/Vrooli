export const findFirstRel = (obj, fieldsToCheck) => {
    for (const fieldName of fieldsToCheck) {
        const value = obj[fieldName];
        if (value !== null && value !== undefined) {
            return [fieldName, value];
        }
    }
    return [undefined, undefined];
};
//# sourceMappingURL=findFirstRel.js.map