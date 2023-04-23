import { isRelationshipObject } from "./isRelationshipObject";
export const selPad = (fields) => {
    if (!isRelationshipObject(fields))
        return fields;
    if ("select" in fields)
        return fields;
    const converted = {};
    Object.keys(fields).forEach((key) => {
        if (Object.keys(fields[key]).length > 0)
            converted[key] = selPad(fields[key]);
        else
            converted[key] = true;
    });
    return { select: converted };
};
//# sourceMappingURL=selPad.js.map