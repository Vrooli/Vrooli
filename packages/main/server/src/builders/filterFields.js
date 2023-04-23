export const filterFields = (data, excludes) => {
    const converted = {};
    Object.keys(data).forEach((key) => {
        if (!excludes.some(e => e === key)) {
            converted[key] = data[key];
        }
    });
    return converted;
};
//# sourceMappingURL=filterFields.js.map