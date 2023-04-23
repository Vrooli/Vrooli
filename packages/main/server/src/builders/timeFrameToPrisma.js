export const timeFrameToPrisma = (fieldName, time) => {
    if (!time || (!time.before && !time.after))
        return undefined;
    const where = ({ [fieldName]: {} });
    if (time.before)
        where[fieldName].lte = time.before;
    if (time.after)
        where[fieldName].gte = time.after;
    return where;
};
//# sourceMappingURL=timeFrameToPrisma.js.map